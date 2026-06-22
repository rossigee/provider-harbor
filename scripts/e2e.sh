#!/usr/bin/env bash
# Self-contained e2e for provider-harbor on KIND with a REAL Harbor.
#
# Stands up (idempotently): a kind cluster -> Crossplane -> Harbor (goharbor
# Helm chart, in-cluster, no TLS/persistence) -> the provider package -> then
# runs uptest (apply -> Ready -> delete) over examples/e2e/*.
#
# Runs uptest v2 (namespaced-aware): apply -> Ready -> import -> delete. The
# import step (delete local state, re-observe the real external resource) works
# on namespaced v2 MRs with uptest v2 (it didn't on v1). The update step is
# skipped — it needs per-example uptest.upbound.io/update-parameter annotations;
# drift/observe-rematch logic is otherwise covered by the unit tests.
#
# Env knobs:
#   KIND_CLUSTER (harbor-e2e)  REGISTRY (ghcr.io/mosabastion)  PROVIDER (provider-harbor)
#   VERSION (required: a published package tag, e.g. v0.17.0-spike.13)
#   HARBOR_PASSWORD (Harbor12345)  GHCR_USER / GHCR_TOKEN (for private pull)
#   KEEP (set to keep the cluster after the run)
set -euo pipefail

KIND_CLUSTER="${KIND_CLUSTER:-harbor-e2e}"
REGISTRY="${REGISTRY:-ghcr.io/mosabastion}"
PROVIDER="${PROVIDER:-provider-harbor}"
VERSION="${VERSION:?set VERSION to a published package tag, e.g. v0.17.0-spike.13}"
HARBOR_PASSWORD="${HARBOR_PASSWORD:-Harbor12345}"
IMAGE="${IMAGE:-${REGISTRY}/${PROVIDER}}"
KCTX="kind-${KIND_CLUSTER}"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

log() { printf '\033[36m==>\033[0m %s\n' "$*"; }
k() { kubectl --context "$KCTX" "$@"; }

require() { for c in "$@"; do command -v "$c" >/dev/null || { echo "missing: $c"; exit 1; }; done; }
require kind kubectl helm
CHAINSAW="${CHAINSAW:-$(command -v chainsaw || true)}"
[ -n "$CHAINSAW" ] || { echo "missing: chainsaw"; exit 1; }
# uptest v2 (namespaced-aware) is NOT `go install`-able — its go.mod has an
# exclude directive, which `go install pkg@version` rejects — so fetch the
# released binary. Cached by version so a pin bump just downloads a new file.
UPTEST_VERSION="${UPTEST_VERSION:-v2.2.0}"
UPTEST="${UPTEST:-$(go env GOPATH)/bin/uptest-${UPTEST_VERSION}}"
if [ ! -x "$UPTEST" ]; then
  log "downloading uptest ${UPTEST_VERSION}"
  curl -fsSL "https://github.com/crossplane/uptest/releases/download/${UPTEST_VERSION}/uptest_$(go env GOOS)-$(go env GOARCH)" -o "$UPTEST"
  chmod +x "$UPTEST"
fi

log "ensure kind cluster ${KIND_CLUSTER}"
kind get clusters 2>/dev/null | grep -qx "$KIND_CLUSTER" || kind create cluster --name "$KIND_CLUSTER"

log "install Crossplane"
helm repo add crossplane-stable https://charts.crossplane.io/stable >/dev/null 2>&1 || true
helm repo update crossplane-stable >/dev/null 2>&1
helm --kube-context "$KCTX" upgrade --install crossplane crossplane-stable/crossplane \
  -n crossplane-system --create-namespace --wait --timeout 5m >/dev/null

log "install Harbor (in-cluster, no TLS/persistence)"
helm repo add harbor https://helm.goharbor.io >/dev/null 2>&1 || true
helm repo update harbor >/dev/null 2>&1
k create namespace harbor --dry-run=client -o yaml | k apply -f - >/dev/null
helm --kube-context "$KCTX" upgrade --install my-harbor harbor/harbor -n harbor \
  --set expose.type=clusterIP --set expose.tls.enabled=false \
  --set externalURL=http://harbor.harbor.svc --set persistence.enabled=false \
  --set harborAdminPassword="$HARBOR_PASSWORD" --set trivy.enabled=true \
  --set jobservice.replicas=1 --wait --timeout 10m >/dev/null

log "install provider ${IMAGE}:${VERSION}"
PULL_REF=""
if [ "${PRIVATE:-true}" = "true" ]; then
  TOKEN="${GHCR_TOKEN:-$(gh auth token 2>/dev/null || true)}"
  USER="${GHCR_USER:-$(gh api user -q .login 2>/dev/null || echo x)}"
  [ -n "$TOKEN" ] || { echo "no GHCR token (export GHCR_TOKEN or 'gh auth refresh -s read:packages')"; exit 1; }
  k create secret docker-registry ghcr-pull -n crossplane-system \
    --docker-server=ghcr.io --docker-username="$USER" --docker-password="$TOKEN" \
    --dry-run=client -o yaml | k apply -f - >/dev/null
  PULL_REF=$'\n  packagePullSecrets:\n    - name: ghcr-pull'
fi
cat <<EOF | k apply -f -
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: ${PROVIDER}
spec:
  package: ${IMAGE}:${VERSION}${PULL_REF}
EOF
log "wait provider Healthy"
for i in $(seq 1 60); do
  [ "$(k get provider.pkg "$PROVIDER" -o jsonpath='{.status.conditions[?(@.type=="Healthy")].status}' 2>/dev/null)" = "True" ] && break
  sleep 5
done
n="$(k get crd -o name 2>/dev/null | grep -c 'harbor.m.crossplane.io' || true)"
[ "$n" -gt 0 ] || { echo "provider Healthy but $n CRDs registered — packaging broken"; exit 1; }
log "provider Healthy, ${n} CRDs"

log "run uptest e2e (apply -> Ready -> delete)"
cd "$ROOT"   # uptest resolves manifest paths relative to cwd
LIST="$(cd "$ROOT" && ls examples/e2e/*.yaml | paste -sd, -)"
# uptest + chainsaw use the *current* kubectl context; point it at the kind
# cluster and restore the caller's context on exit.
ORIG_CTX="$(kubectl config current-context 2>/dev/null || true)"
restore_ctx() { [ -n "$ORIG_CTX" ] && kubectl config use-context "$ORIG_CTX" >/dev/null 2>&1 || true; }
trap restore_ctx EXIT
kubectl config use-context "$KCTX" >/dev/null

rc=0
KUBECTL=$(command -v kubectl) CHAINSAW="$CHAINSAW" \
  "$UPTEST" e2e "$LIST" \
  --setup-script="$ROOT/test/e2e/uptest-setup.sh" \
  --default-conditions=Ready --skip-update --default-timeout=600s || rc=$?

if [ -z "${KEEP:-}" ]; then
  log "delete kind cluster ${KIND_CLUSTER}"
  kind delete cluster --name "$KIND_CLUSTER" >/dev/null 2>&1 || true
fi
exit $rc
