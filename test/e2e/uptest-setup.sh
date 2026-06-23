#!/usr/bin/env bash
# uptest setup: prepares the test namespace for the harbor e2e examples.
#   - namespaced creds Secret + v2 ProviderConfig pointing at in-cluster Harbor
#   - a user-password Secret (for the User example)
#   - seeds a real image: creates an "uptest-images" project (Harbor API, not a
#     CR — so it doesn't trip uptest's "wait managed --all --for=delete") and
#     pushes busybox into it, so the repository/artifact examples have something
#     real to observe.
set -aeuo pipefail
: "${KUBECTL:=kubectl}"
NS="${UPTEST_NAMESPACE:-uptest}"
HARBOR_URL="${HARBOR_URL:-http://harbor.harbor.svc}"
HARBOR_PASSWORD="${HARBOR_PASSWORD:-Harbor12345}"
IMAGES_PROJECT="${IMAGES_PROJECT:-uptest-images}"

echo "uptest-setup: namespace + secrets + ProviderConfig in $NS"
${KUBECTL} create namespace "$NS" --dry-run=client -o yaml | ${KUBECTL} apply -f -
${KUBECTL} -n "$NS" create secret generic harbor-creds \
  --from-literal=url="$HARBOR_URL" --from-literal=username=admin --from-literal=password="$HARBOR_PASSWORD" \
  --dry-run=client -o yaml | ${KUBECTL} apply -f -
${KUBECTL} -n "$NS" create secret generic user-password \
  --from-literal=password='Uptest-User-123' \
  --dry-run=client -o yaml | ${KUBECTL} apply -f -
cat <<YAML | ${KUBECTL} apply -f -
apiVersion: harbor.m.crossplane.io/v1beta1
kind: ProviderConfig
metadata:
  name: harbor-e2e
  namespace: ${NS}
spec:
  credentials:
    source: Secret
    secretRef:
      namespace: ${NS}
      name: harbor-creds
      key: password
YAML

echo "uptest-setup: seeding image into project '${IMAGES_PROJECT}' (in-cluster Job)"
${KUBECTL} -n "$NS" delete job seed-image --ignore-not-found >/dev/null
cat <<YAML | ${KUBECTL} apply -f -
apiVersion: batch/v1
kind: Job
metadata:
  name: seed-image
  namespace: ${NS}
spec:
  backoffLimit: 2
  template:
    spec:
      restartPolicy: Never
      containers:
        - name: seed
          # gcr mirror avoids Docker Hub anonymous pull-rate limits in CI.
          image: mirror.gcr.io/library/alpine:3.20
          env:
            - {name: HARBOR, value: "harbor.harbor.svc"}
            - {name: PW, value: "${HARBOR_PASSWORD}"}
            - {name: PROJECT, value: "${IMAGES_PROJECT}"}
          command: ["/bin/sh", "-c"]
          args:
            - |
              set -e
              apk add --no-cache curl tar >/dev/null
              wget -qO /tmp/c.tgz https://github.com/google/go-containerregistry/releases/download/v0.20.2/go-containerregistry_Linux_x86_64.tar.gz
              tar -xzf /tmp/c.tgz -C /usr/local/bin crane
              echo "creating project \$PROJECT (ignore if exists)"
              curl -sf -u "admin:\$PW" -X POST "http://\$HARBOR/api/v2.0/projects" \
                -H 'Content-Type: application/json' \
                -d "{\"project_name\":\"\$PROJECT\",\"public\":true}" || true
              echo "pushing busybox -> \$HARBOR/\$PROJECT/busybox:latest"
              crane auth login "\$HARBOR" -u admin -p "\$PW" --insecure
              crane copy mirror.gcr.io/library/busybox:latest "\$HARBOR/\$PROJECT/busybox:latest" --insecure
              echo "seed done"
YAML
${KUBECTL} -n "$NS" wait --for=condition=complete job/seed-image --timeout=300s

echo "uptest-setup: waiting until provider is healthy"
${KUBECTL} wait provider.pkg --all --for condition=Healthy --timeout 5m
echo "uptest-setup: done"
