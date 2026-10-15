# Getting Started

Guide to getting started with the Harbor provider.

## Installation

Install the Harbor provider:

```bash
kubectl crossplane install provider ghcr.io/rossigee/provider-harbor:latest
```

## Prerequisites

- Kubernetes cluster with Crossplane installed
- Harbor instance

## Quick Start

1. Create a ProviderConfig:

```yaml
apiVersion: harbor.m.crossplane.io/v1beta1
kind: ProviderConfig
metadata:
  name: default
spec:
  credentials:
    source: Secret
    secretRef:
      name: harbor-credentials
      namespace: crossplane-system
      key: credentials
```

2. Create the credentials Secret as a single JSON blob at `key: credentials`:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: harbor-credentials
  namespace: crossplane-system
type: Opaque
stringData:
  credentials: |
    {
      "url": "https://harbor.example.com",
      "username": "admin",
      "password": "your-password"
    }
```
