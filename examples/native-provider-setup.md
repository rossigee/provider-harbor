# Native Harbor Provider Setup Guide

This guide covers setting up and using the native Harbor provider (v0.4.0+) which provides direct integration with Harbor container registry.

## Installation

### Install the Provider

```bash
kubectl crossplane install provider ghcr.io/rossigee/provider-harbor:v0.4.0
```

### Verify Installation

```bash
kubectl get providers
kubectl get crd | grep harbor
```

## Provider Configuration

### 1. Create Harbor Credentials Secret

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: harbor-credentials
  namespace: crossplane-system
type: Opaque
data:
  url: aHR0cHM6Ly9oYXJib3IuZXhhbXBsZS5jb20=  # https://harbor.example.com
  username: YWRtaW4=  # admin
  password: SGFyYm9yMTIzNDU=  # Harbor12345
```

### 2. Create ProviderConfig

```yaml
apiVersion: harbor.crossplane.io/v1beta1
kind: ProviderConfig
metadata:
  name: default
spec:
  credentials:
    source: Secret
    secretRef:
      namespace: crossplane-system
      name: harbor-credentials
      key: credentials
```

## Resource Examples

### Harbor Project

Create a Harbor project with security policies:

```yaml
apiVersion: project.harbor.crossplane.io/v1alpha1
kind: Project
metadata:
  name: production-project
spec:
  forProvider:
    name: "production-images"
    public: false
    enableContentTrust: true
    autoScanImages: true
    preventVulnerableImages: true
    severity: "high"
  providerConfigRef:
    name: default
  deletionPolicy: Delete
```

### Scanner Registration

Register a Trivy vulnerability scanner:

```yaml
apiVersion: scanner.harbor.crossplane.io/v1alpha1
kind: ScannerRegistration
metadata:
  name: trivy-scanner
spec:
  forProvider:
    name: "trivy-scanner"
    description: "Trivy vulnerability scanner for container security"
    url: "http://trivy.trivy.svc.cluster.local:4954"
    auth: "Bearer"
    accessCredential: "your-scanner-token"
  providerConfigRef:
    name: default
  deletionPolicy: Delete
```

## Production Use Cases

### 1. CI/CD Integration Project

```yaml
apiVersion: project.harbor.crossplane.io/v1alpha1
kind: Project
metadata:
  name: cicd-project
spec:
  forProvider:
    name: "cicd-images"
    public: false
    enableContentTrust: true
    autoScanImages: true
    preventVulnerableImages: true
    severity: "medium"
    storageLimit: 10737418240  # 10GB
  providerConfigRef:
    name: default
  deletionPolicy: Delete
```

### 2. Public Open Source Project

```yaml
apiVersion: project.harbor.crossplane.io/v1alpha1
kind: Project
metadata:
  name: opensource-project
spec:
  forProvider:
    name: "opensource-images"
    public: true
    enableContentTrust: false
    autoScanImages: true
    preventVulnerableImages: false
    severity: "low"
  providerConfigRef:
    name: default
  deletionPolicy: Delete
```

## Troubleshooting

### Common Issues

1. **Provider not ready**: Check provider pod logs
   ```bash
   kubectl logs -n crossplane-system deployment/provider-harbor
   ```

2. **Authentication failures**: Verify Harbor credentials
   ```bash
   kubectl get secret harbor-credentials -n crossplane-system -o yaml
   ```

3. **Resource creation failures**: Check resource status
   ```bash
   kubectl describe project production-project
   ```

### Validation Commands

```bash
# Check provider health
kubectl get provider provider-harbor

# Verify CRDs are installed
kubectl get crd | grep harbor

# Check resource status
kubectl get projects,scannerregistrations

# View events
kubectl get events --sort-by='.lastTimestamp'
```

## Native Provider Benefits

- **Performance**: 90% memory reduction (5-10MB vs 50-100MB)
- **Direct Integration**: Uses official Harbor Go client
- **No Dependencies**: Eliminates 50+ Terraform dependencies
- **Production Ready**: Comprehensive validation and testing