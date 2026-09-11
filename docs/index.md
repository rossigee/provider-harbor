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

### Other Resources

See `apis/` directory for all available resources.
