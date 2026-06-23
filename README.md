# Provider Harbor

[![Build](https://github.com/rossigee/provider-harbor/actions/workflows/ci.yml/badge.svg)](https://github.com/rossigee/provider-harbor/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

A v2-only Crossplane provider for declarative [Harbor](https://goharbor.io/)
container-registry management: **12 namespaced resource kinds**, each with a
working create/observe/update/delete reconciler proven against a real Harbor
server in CI.

## Overview

This provider manages Harbor resources (projects, registries, users, groups,
robot accounts, webhooks, replication, retention, and more) as
Kubernetes custom resources. Every kind has a reconciler and a table-driven unit
test against an in-memory Harbor fake; the mutable kinds additionally run
apply→Ready→import→delete against a **real Harbor** (the official goharbor Helm
chart) on a kind cluster, wired into CI.

The controllers bake in the v2-runtime correctness lessons: `Available()` set in
`Observe` (crossplane-runtime/v2 no longer does it for you), not-found
classified through a shared `isHarborNotFound` `(nil, nil)` contract, a non-nil
rate limiter on every controller (a nil one panics), external-name as the
authoritative identity for Observe/Update/Delete, and idempotent deletes.

## Resource catalog

All kinds use the namespaced v2 group `<group>.harbor.m.crossplane.io/v1beta1`
and must carry `metadata.namespace`.

| Resource | Group | Purpose | Example |
|----------|-------|---------|---------|
| `Project` | `project.harbor.*` | Project lifecycle (public flag, quotas) | [project.yaml](examples/e2e/project.yaml) |
| `Registry` | `registry.harbor.*` | Remote/proxy registry endpoint + credentials 🔑 | [registry.yaml](examples/e2e/registry.yaml) |
| `ScannerRegistration` | `scanner.harbor.*` | Register an external scanner adapter | [scanner.yaml](examples/e2e/scanner.yaml) |
| `User` | `user.harbor.*` | Local Harbor user account 🔑 | [user.yaml](examples/e2e/user.yaml) |
| `UserGroup` | `usergroup.harbor.*` | LDAP(1)/HTTP(2)/OIDC(3) group | [usergroup.yaml](examples/e2e/usergroup.yaml) |
| `UserMember` | `member.harbor.*` | Project membership for a user | [usermember.yaml](examples/e2e/usermember.yaml) |
| `GroupMember` | `member.harbor.*` | Project membership for a group | [groupmember.yaml](examples/e2e/groupmember.yaml) |
| `Member` | `member.harbor.*` | **Deprecated** — user-only catch-all; use `UserMember`/`GroupMember` | [member.yaml](examples/e2e/member.yaml) |
| `Robot` 🔑 | `robot.harbor.*` | Robot account (project- or system-level CI/CD credential) | [robot.yaml](examples/e2e/robot.yaml) |
| `Webhook` | `webhook.harbor.*` | Project webhook policy | [webhook.yaml](examples/e2e/webhook.yaml) |
| `Replication` | `replication.harbor.*` | Replication policy between registries | [replication.yaml](examples/e2e/replication.yaml) |
| `Retention` | `retention.harbor.*` | Tag-retention policy (one per project) | [retention.yaml](examples/e2e/retention.yaml.disabled) |

🔑 = involves a secret value — see [Working with secret-bearing
resources](#working-with-secret-bearing-resources).

### What is NOT modeled

Some Harbor concepts have no CRD. They are runtime actions, content produced
out-of-band, or global singletons — a managed resource around them would never
hold meaningful desired state:

| Concept | Why there is no CRD |
|---------|---------------------|
| Repository | Auto-created on first `docker push` and cannot be explicitly created; only its metadata would be manageable. Not worth a CRD. |
| Artifact | Image content arrives via `docker push`, not the API — read-only observe/delete. Not desired state. |
| Scan | A scan is a *trigger/action*, not a stored object (no in-place update). Run it via Harbor's API/UI or CI, not a CRD. |
| Quota | Project quota is a sub-field of `Project`, not an independent object. |
| Garbage collection / purge | A scheduled maintenance action, not desired state. |
| Audit log | Read-only event stream, not config. |
| Immutable-tag rule | Not yet modeled (could be added as a per-project policy CR). |
| Label | An `examples/label/` stub exists but there is **no** `apis/label` group or controller — not served. |
| System / project configuration | Global singletons set out-of-band. |

## Non-default resource behaviors

Several Harbor resources do **not** follow the plain create-from-spec /
update-in-place managed-resource shape. Read this before relying on them — it is
the Harbor equivalent of the gitea provider's `AccessToken` caveat.

### `Robot` — once-only secret, not adoptable, numeric-id identity

Robot accounts are the credential-bearing kind, and the most non-standard:

- **The secret is returned exactly once.** On `Create`, Harbor mints the robot
  and returns its secret one time only. The provider publishes it as the
  resource's **connection details** (`name` + `secret`) and mirrors it to
  `status.atProvider.secret`. It is never re-fetched.
- **External-name is the numeric Harbor robot id**, set via
  `meta.SetExternalName` after Create. `Observe` is external-name-only: it
  get-by-id when the id is known, otherwise reports "does not exist" and Creates.
- **It cannot be imported / adopted.** There is no list-and-name-match fallback;
  a non-numeric external name (Crossplane's default `metadata.name`) is treated
  as "not created yet". You cannot bring an existing Harbor robot under
  management by guessing its name.
- **A create conflict (HTTP 409) is actionable, never auto-resolved.** If a robot
  of that name already exists, the controller returns an error telling you to
  delete it — it will **not** delete/recreate and silently rotate the secret.
- **Level-aware.** `spec.forProvider.level` is `project` (default) or `system`;
  system robots carry `permissions[].kind` + `permissions[].scope` (`/` for
  system-wide).
- **Update touches `description` only** — the project binding is immutable, and
  Harbor returns the observed `projectId` as the project *name* rather than the
  numeric spec value, so it is deliberately excluded from drift comparison.

### `Member` is split and deprecated

Harbor distinguishes user members (`member_user`) from group members
(`member_group`). The original catch-all `Member` (user-only) is **deprecated**
(its served version carries a deprecation warning) in favour of the
single-responsibility `UserMember` and `GroupMember`. `GroupMember.groupType`
defaults to `3` (OIDC; `1`=LDAP, `2`=HTTP).

### `projectId` is a numeric id, not a name

`Robot` takes `spec.forProvider.projectId` as Harbor's **numeric** project id
(not the project name). The string-keyed kinds (`Project`→`name`,
`Registry`→`name`, `User`→`username`, `ScannerRegistration`→`name`) use their
natural name as identity; the numeric-id-keyed kinds (`Robot`, `UserGroup`,
`UserMember`, `GroupMember`) store the Harbor-assigned id as the external-name.

## Working with secret-bearing resources

**Input secret values are never set inline** — they are Kubernetes Secret
references (`*SecretRef`), following the platform's provider convention:

| Resource | Field | Holds |
|----------|-------|-------|
| `User` | `spec.forProvider.passwordSecretRef` | the user's password (create) |
| `Registry` | `spec.forProvider.credential.accessSecretRef` | the remote registry access secret (paired with `accessKey`) |

`Robot` is the inverse — it does not take a secret in, it **emits** one as
connection details (see [above](#robot--once-only-secret-not-adoptable-numeric-id-identity)).

The `ProviderConfig` itself authenticates from a Secret carrying `url`,
`username`, and `password` (secret-source only).

## Quick start

```yaml
apiVersion: v1
kind: Secret
metadata: {name: harbor-creds, namespace: default}
stringData:
  url: https://harbor.example.com
  username: admin
  password: password
---
apiVersion: harbor.m.crossplane.io/v1beta1
kind: ProviderConfig
metadata: {name: default}
spec:
  credentials:
    source: Secret
    secretRef: {namespace: default, name: harbor-creds, key: password}
---
apiVersion: project.harbor.m.crossplane.io/v1beta1
kind: Project
metadata: {name: my-project, namespace: default}
spec:
  forProvider:
    name: my-project
    public: false
  providerConfigRef: {name: default, kind: ProviderConfig}
```

Confirm the CRDs registered (`kubectl get crds | grep harbor.m.crossplane.io`)
and install the provider with
`kubectl crossplane install provider ghcr.io/rossigee/provider-harbor:<tag>`.

## Testing

```bash
make test          # unit tests (offline, table-driven per controller against in-memory Harbor fakes)
make lint          # golangci-lint
make e2e           # self-contained kind + REAL Harbor e2e (apply->Ready->import->delete)
```

- Every controller has a unit test asserting the correctness invariants
  (Available on the exists path, typed not-found, drift, external-name identity,
  idempotent delete).
- `scripts/e2e.sh` (and CI `e2e.yaml`) install a real Harbor (official goharbor
  Helm chart) on a throwaway kind cluster and drive every `examples/e2e/`
  resource through its lifecycle via uptest v2 — no mock backend. The live
  **update** step is skipped (drift is covered by the unit tests); apply, Ready,
  import, and delete run live.

## Registry

`ghcr.io/rossigee/provider-harbor`
