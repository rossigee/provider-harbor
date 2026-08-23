# provider-harbor

[![CI](https://img.shields.io/github/actions/workflow/status/rossigee/provider-harbor/ci.yml?branch=master)][build]
[![Version](https://img.shields.io/github/v/release/rossigee/provider-harbor)][releases]
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

[build]: https://github.com/rossigee/provider-harbor/actions/workflows/ci.yml
[releases]: https://github.com/rossigee/provider-harbor/releases

## Overview

A native Crossplane provider for Harbor container registry management. Manage Harbor resources (projects, registries, users, repositories, webhooks, and more) using Kubernetes-native declarative configuration.

## Container Registry

- **Primary**: `ghcr.io/rossigee/provider-harbor:v0.17.2`

## Features

- **Projects** — create and manage Harbor projects with security policies
- **Registries** — register and manage remote registries
- **Users & User Groups** — user accounts with password secrets; LDAP/HTTP/OIDC group management
- **Repositories & Artifacts** — repository lifecycle, metadata, and image artifact management with vulnerability scanning
- **Scanners** — scanner registration (Trivy, Clair, Aqua, etc.)
- **Robot Accounts** — CI/CD service accounts with scoped permissions
- **Webhooks** — event automation for scan completion, image push
- **Replication Policies** — cross-registry image replication with filtering
- **Retention Policies** — automated artifact cleanup with custom rules
- **Members** — project member management and role-based access control
- **Scans** — vulnerability scan management and reporting

## Getting Started

### Prerequisites

- Kubernetes with Crossplane installed
- A Harbor instance and admin (or sufficiently scoped) credentials

### Installation

```bash
kubectl crossplane install provider ghcr.io/rossigee/provider-harbor:v0.17.2
```

### Configuration

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: harbor-creds
type: Opaque
stringData:
  url: https://harbor.example.com
  username: admin
  password: password
---
apiVersion: harbor.m.crossplane.io/v1beta1
kind: ProviderConfig
metadata:
  name: default
spec:
  credentials:
    source: Secret
    secretRef:
      name: harbor-creds
```

## Usage

```yaml
apiVersion: project.harbor.m.crossplane.io/v1beta1
kind: Project
metadata:
  name: my-project
spec:
  forProvider:
    name: my-project
    public: false
  providerConfigRef:
    name: default
```

## Resource Types

| Resource | API Group | Description |
|----------|-----------|-------------|
| Project | `project.harbor.m.crossplane.io` | Harbor projects and security policies |
| Registry | `registry.harbor.m.crossplane.io` | Remote registry registration |
| User | `user.harbor.m.crossplane.io` | User accounts |
| UserGroup | `usergroup.harbor.m.crossplane.io` | LDAP/HTTP/OIDC group management |
| Repository | `repository.harbor.m.crossplane.io` | Repository lifecycle and metadata |
| Artifact | `artifact.harbor.m.crossplane.io` | Image artifact and vulnerability management |
| Scanner | `scanner.harbor.m.crossplane.io` | Scanner registration |
| Robot | `robot.harbor.m.crossplane.io` | CI/CD service accounts |
| Webhook | `webhook.harbor.m.crossplane.io` | Event automation |
| Replication | `replication.harbor.m.crossplane.io` | Cross-registry replication policies |
| Retention | `retention.harbor.m.crossplane.io` | Artifact retention policies |
| Member | `member.harbor.m.crossplane.io` | Project membership and RBAC |
| Scan | `scan.harbor.m.crossplane.io` | Vulnerability scan management |

## Development

```bash
# Build
make build

# Test
make test

# Lint
make lint

# Generate
make generate
```

Further documentation:

- **[DEPLOYMENT.md](docs/DEPLOYMENT.md)** — production deployment, security, monitoring, RBAC, troubleshooting
- **[RELEASE_PROCESS.md](docs/RELEASE_PROCESS.md)** — release versioning, timeline, checklist
- **[IMPLEMENTATION.md](docs/IMPLEMENTATION.md)** — implementation guide for features and resources
- **[API_ANALYSIS.md](docs/API_ANALYSIS.md)** — Harbor API gaps and coverage analysis
- **[MIGRATION_UPJET.md](docs/MIGRATION_UPJET.md)** — migration guide from the Upjet-based provider
- **[MIGRATION_TERRAFORM.md](docs/MIGRATION_TERRAFORM.md)** — migration guide from the Terraform provider
- **[ROBOTACCOUNT-DOCKER-CONFIG.md](docs/ROBOTACCOUNT-DOCKER-CONFIG.md)** — Docker config JSON support for RobotAccount
- **[CHANGELOG.md](CHANGELOG.md)** — version history and release notes

## Contributing

Issues and pull requests are welcome at [github.com/rossigee/provider-harbor](https://github.com/rossigee/provider-harbor).

## License

provider-harbor is under the Apache 2.0 license.

## Implementation

This provider is a native Crossplane controller that directly implements the provider APIs without using Terraform or upjet scaffolding. This approach yields smaller binaries, simpler code, and reduced dependencies.
