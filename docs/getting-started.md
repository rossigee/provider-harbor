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
apiVersion: harbor.crossplane.io/v1
kind: ProviderConfig
metadata:
  name: default
spec:
  credentials:
    source: Secret
    secretRef:
      name: harbor-credentials
      namespace: crossplane-system
```
