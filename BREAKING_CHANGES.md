# Breaking Changes

## Next release — API group consolidation + Member redesign

> All existing CRs must be deleted and re-applied. CRD names change; in-place
> upgrade is not possible.

### 1. Unified API group for all managed resources

Every managed resource moves from a per-resource group to a single shared group.

| Before | After |
|--------|-------|
| `project.harbor.m.crossplane.io/v1beta1` | `harbor.m.crossplane.io/v1beta1` |
| `registry.harbor.m.crossplane.io/v1beta1` | `harbor.m.crossplane.io/v1beta1` |
| `scanner.harbor.m.crossplane.io/v1beta1` | `harbor.m.crossplane.io/v1beta1` |
| `user.harbor.m.crossplane.io/v1beta1` | `harbor.m.crossplane.io/v1beta1` |
| `usergroup.harbor.m.crossplane.io/v1beta1` | `harbor.m.crossplane.io/v1beta1` |
| `member.harbor.m.crossplane.io/v1beta1` | `harbor.m.crossplane.io/v1beta1` |
| `robot.harbor.m.crossplane.io/v1beta1` | `harbor.m.crossplane.io/v1beta1` |
| `webhook.harbor.m.crossplane.io/v1beta1` | `harbor.m.crossplane.io/v1beta1` |
| `replication.harbor.m.crossplane.io/v1beta1` | `harbor.m.crossplane.io/v1beta1` |
| `retention.harbor.m.crossplane.io/v1beta1` | `harbor.m.crossplane.io/v1beta1` |

**Why:** the per-resource group (inherited from the upjet pattern) is redundant —
the kind already carries the resource name. A unified group is simpler to RBAC,
easier to `kubectl get`, and eliminates the `project.project.harbor.m...` noise.

**Migration:** update every manifest's `apiVersion` from
`<resource>.harbor.m.crossplane.io/v1beta1` to `harbor.m.crossplane.io/v1beta1`.

### 2. ProviderConfig group change

| Before | After |
|--------|-------|
| `harbor.m.crossplane.io/v1beta1 ProviderConfig` | `harbor.crossplane.io/v1beta1 ProviderConfig` |
| `harbor.m.crossplane.io/v1beta1 ProviderConfigUsage` | `harbor.crossplane.io/v1beta1 ProviderConfigUsage` |

**Why:** `.m.` marks managed resources (resources with `forProvider`/`atProvider`
and an external-name identity). `ProviderConfig` is a provider identity resource —
it does not reconcile an external object. Using `.m.` on it is semantically
incorrect. `harbor.crossplane.io` is consistent with how official Crossplane
providers (provider-kubernetes, provider-helm) name their identity resources.

**Migration:** update ProviderConfig/ProviderConfigUsage manifests to
`apiVersion: harbor.crossplane.io/v1beta1`.

### 3. `UserMember` and `GroupMember` merged into `Member`

`UserMember` and `GroupMember` are removed. `Member` becomes the single kind with
a required `type` discriminator, following the same pattern as `Robot.level`.

**New `Member` spec:**

```yaml
apiVersion: harbor.m.crossplane.io/v1beta1
kind: Member
spec:
  forProvider:
    projectId: "42"
    type: user          # required — "user" or "group"
    username: alice     # required when type=user
    # groupName: ops    # required when type=group
    # groupType: 3      # optional when type=group: 1=LDAP, 2=HTTP, 3=OIDC (default 2)
    role: developer
```

| Field | When present |
|-------|-------------|
| `type` | always (required) |
| `username` | `type: user` only |
| `groupName` | `type: group` only |
| `groupType` | `type: group` only (optional, default 2) |
| `projectId` | always (required) |
| `role` | always (required) |

**Why:** `UserMember` and `GroupMember` have no structural difference that
justifies separate kinds — the discriminator is behavioral (which Harbor endpoint
sub-path to call), the same situation `Robot.level` already handles. Separate
kinds introduced redundancy (`member.harbor.m.crossplane.io/v1beta1 GroupMember`
contains "member" three times). The deprecated catch-all `Member` kind is now
the canonical, unified kind.

**Migration:**
- `UserMember` → `Member` with `type: user`; rename `username` stays as-is.
- `GroupMember` → `Member` with `type: group`; rename `groupName`/`groupType`
  stay as-is.
