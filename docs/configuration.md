# Configuration

Guide for configuring the Harbor provider.

## ProviderConfig

Create a ProviderConfig to configure connection settings:

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
```

## Authentication

Create a secret with your Harbor credentials:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: harbor-credentials
  namespace: crossplane-system
type: Opaque
stringData:
  url: https://harbor.example.com
  username: admin
  password: your-password
```
