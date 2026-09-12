# Provider Harbor Documentation

A Crossplane provider for managing Harbor container registry resources.

## Quick Links

- [Implementation](implementation.md) — Implementation details
- [Deployment](deployment.md) — Deployment guide
- [Release Process](release-process.md) — Release procedures

## Resource Documentation

Resources are documented in the API types. Individual resource documentation will be added to the [resources/](resources/) folder.

### Registry Resources

| Resource | API Group | Description |
|----------|-----------|-------------|
| Project | `project.harbor.m.crossplane.io/v1beta1` | Harbor projects |
| Registry | `registry.harbor.m.crossplane.io/v1beta1` | External registries |
| Repository | `repository.harbor.m.crossplane.io/v1beta1` | Container repositories |

### Security & Management

| Resource | API Group | Description |
|----------|-----------|-------------|
| Robot | `robot.harbor.m.crossplane.io/v1beta1` | Robot accounts |
| Member | `member.harbor.m.crossplane.io/v1beta1` | Project members |
| Retention | `retention.harbor.m.crossplane.io/v1beta1` | Retention policies |
| Replication | `replication.harbor.m.crossplane.io/v1beta1` | Replication policies |

### Identity & Scanning

| Resource | API Group | Description |
|----------|-----------|-------------|
| User | `user.harbor.m.crossplane.io/v1beta1` | User accounts |
| UserGroup | `usergroup.harbor.m.crossplane.io/v1beta1` | LDAP/OIDC group management |
| Artifact | `artifact.harbor.m.crossplane.io/v1beta1` | Image artifacts and vulnerabilities |
| Scanner | `scanner.harbor.m.crossplane.io/v1beta1` | Scanner registration |
| Scan | `scan.harbor.m.crossplane.io/v1beta1` | Vulnerability scan management |
| Webhook | `webhook.harbor.m.crossplane.io/v1beta1` | Event automation |
| ProviderConfig | `harbor.m.crossplane.io/v1beta1` | Provider credentials (cluster-scoped) |

## API Coverage Gaps

Harbor API surface not yet modeled: project quotas and CVE allowlists as dedicated resources, immutable tag rules, tag retention under repositories beyond policy resources, OIDC/LDAP configuration endpoints, system-level garbage collection schedules, audit logs, and preheat/artifact-copy policies.
