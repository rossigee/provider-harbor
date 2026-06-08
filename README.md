# Provider Harbor

[![Build](https://github.com/rossigee/provider-harbor/actions/workflows/ci.yml/badge.svg)](https://github.com/rossigee/provider-harbor/actions/workflows/ci.yml)
[![Coverage](https://img.shields.io/badge/Coverage-10%25-green)](https://github.com/rossigee/provider-harbor/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

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

## Recent Improvements (v0.16.0)

- ✅ Fixed critical CRD API group mismatch in v1beta1 resources
- ✅ Enabled proper logging in all controllers for debugging
- ✅ Implemented password secret reading for Users
- ✅ Added complete UserGroup (LDAP/HTTP/OIDC) support
- ✅ All tests passing (65+ unit tests)
- ✅ Code formatting and linting verified

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

- **CHANGELOG.md** - Version history and release notes
- **PRODUCTION_SAFETY.md** - Secure configuration patterns
- **PRODUCTION_READINESS.md** - Pre-deployment checklist

