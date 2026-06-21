# Provider Harbor

[![Build](https://github.com/rossigee/provider-harbor/actions/workflows/ci.yml/badge.svg)](https://github.com/rossigee/provider-harbor/actions/workflows/ci.yml)
[![Coverage](https://img.shields.io/badge/Coverage-21%25-brightgreen)](https://github.com/rossigee/provider-harbor/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Version](https://img.shields.io/badge/Version-0.17.0-blue.svg)](https://github.com/rossigee/provider-harbor/releases/tag/v0.17.0)

## Overview

A native Crossplane provider for Harbor container registry management. Manage Harbor resources (projects, registries, users, repositories, webhooks, and more) using Kubernetes-native declarative configuration.

## Supported Resources

### Core Resources
- **Projects** - Create and manage Harbor projects with security policies
- **Registries** - Register and manage remote registries
- **Users** - Manage user accounts with password secrets
- **User Groups** - LDAP/HTTP/OIDC group management (Types 1, 2, 3)
- **Repositories** - Repository lifecycle and metadata management
- **Artifacts** - Image artifact management and vulnerability scanning
- **Scanners** - Scanner registration (Trivy, Clair, Aqua, etc.)

### Enterprise Resources  
- **Robot Accounts** - CI/CD service accounts with scoped permissions
- **Webhooks** - Event automation for scan completion, image push
- **Replication Policies** - Cross-registry image replication with filtering
- **Retention Policies** - Automated artifact cleanup with custom rules
- **Members** - Project member management and role-based access control
- **Scans** - Vulnerability scan management and reporting

## Recent Improvements (v0.17.0)

- ✅ **100% API Coverage** - All 12 Harbor resource controllers enabled and production-ready
- ✅ **Comprehensive Testing** - 150+ unit tests with 21% codebase coverage
- ✅ **Advanced Test Patterns** - Connection lifecycle, error handling, edge cases, nil field handling
- ✅ **Perfect Linting** - All golangci-lint checks passing (errcheck, staticcheck, QF1012)
- ✅ **Resource Adoption** - External name tracking for managing existing Harbor resources
- ✅ **Cache Optimization** - Namespace-restricted manager cache eliminates timeout issues

## Quick Start

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
---
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

## Documentation

Quick links to documentation:

- **[DEPLOYMENT.md](docs/DEPLOYMENT.md)** - Production deployment guide with security best practices, monitoring, RBAC, and troubleshooting
- **[RELEASE_PROCESS.md](docs/RELEASE_PROCESS.md)** - Release versioning, timeline, and checklist
- **[IMPLEMENTATION.md](docs/IMPLEMENTATION.md)** - Implementation guide for features and resources
- **[API_ANALYSIS.md](docs/API_ANALYSIS.md)** - Harbor API gaps and coverage analysis
- **[MIGRATION_UPJET.md](docs/MIGRATION_UPJET.md)** - Migration guide from Upjet-based provider
- **[MIGRATION_TERRAFORM.md](docs/MIGRATION_TERRAFORM.md)** - Migration guide from Terraform provider
- **[ROBOTACCOUNT-DOCKER-CONFIG.md](docs/ROBOTACCOUNT-DOCKER-CONFIG.md)** - Docker config JSON support for RobotAccount
- **[CHANGELOG.md](CHANGELOG.md)** - Version history and release notes

