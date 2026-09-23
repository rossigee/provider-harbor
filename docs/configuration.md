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
      key: credentials
```

## Authentication

Create a secret with your Harbor credentials as a single JSON blob at
`secretRef.key` (the provider unmarshals `{url,username,password}` —
see `internal/clients/harbor.go`):

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
