# Migration Guide: Upjet to Native Provider

This guide helps migrate from the Upjet-based Harbor provider to the native implementation.

## Key Changes

### Architecture Changes
- **From**: Terraform wrapper (Upjet) → **To**: Direct Harbor Go client
- **Performance**: 90% memory reduction (50-100MB → 5-10MB)
- **Dependencies**: Eliminated 50+ Terraform dependencies

### API Group Changes
- **From**: `project.harbor.upbound.io` → **To**: `project.harbor.crossplane.io`
- **From**: `scanner.harbor.upbound.io` → **To**: `scanner.harbor.crossplane.io`

## Migration Steps

### 1. Backup Existing Resources

```bash
# Export existing projects
kubectl get projects.project.harbor.upbound.io -o yaml > harbor-projects-backup.yaml

# Export existing scanner registrations (if any)
kubectl get scannerregistrations.scanner.harbor.upbound.io -o yaml > harbor-scanners-backup.yaml
```

### 2. Install Native Provider

```bash
# Remove old provider (optional - can run side by side)
kubectl delete provider provider-harbor-upjet

# Install native provider
kubectl crossplane install provider ghcr.io/rossigee/provider-harbor:v0.4.0
```

### 3. Resource Conversion

#### Old Upjet Format
```yaml
# OLD - Upjet based
apiVersion: project.harbor.upbound.io/v1alpha1
kind: Project
metadata:
  name: my-project
spec:
  forProvider:
    name: "my-harbor-project"
    public: false
  providerConfigRef:
    name: default
```

#### New Native Format
```yaml
# NEW - Native provider
apiVersion: project.harbor.crossplane.io/v1alpha1
kind: Project
metadata:
  name: my-project
spec:
  forProvider:
    name: "my-harbor-project"
    public: false
    # Native provider supports additional fields
    enableContentTrust: true
    autoScanImages: true
    preventVulnerableImages: true
    severity: "high"
  providerConfigRef:
    name: default
```

### 4. Provider Configuration Changes

#### Old Upjet ProviderConfig
```yaml
# OLD - Upjet
apiVersion: harbor.upbound.io/v1beta1
kind: ProviderConfig
metadata:
  name: default
spec:
  credentials:
    source: Secret
    secretRef:
      namespace: crossplane-system
      name: harbor-secret
      key: creds
```

#### New Native ProviderConfig
```yaml
# NEW - Native
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

## Feature Comparison

| Feature | Upjet Provider | Native Provider |
|---------|----------------|-----------------|
| Memory Usage | 50-100MB | 5-10MB |
| Dependencies | 50+ Terraform deps | Harbor Go client only |
| Project Management | Basic CRUD | Full CRUD + Security |
| Scanner Integration | Limited | Full scanner mgmt |
| Build Time | ~5-10 minutes | ~1-2 minutes |
| Container Size | ~500MB | ~50MB |

## New Features in Native Provider

### Enhanced Project Security
```yaml
spec:
  forProvider:
    enableContentTrust: true      # Docker Content Trust
    autoScanImages: true          # Automatic vulnerability scanning
    preventVulnerableImages: true # Block vulnerable deployments
    severity: "high"              # Minimum severity threshold
    storageLimit: 10737418240     # Storage quota in bytes
```

### Scanner Registration
```yaml
apiVersion: scanner.harbor.crossplane.io/v1alpha1
kind: ScannerRegistration
metadata:
  name: trivy-scanner
spec:
  forProvider:
    name: "trivy-scanner"
    description: "Trivy vulnerability scanner"
    url: "http://trivy.trivy.svc.cluster.local:4954"
    auth: "Bearer"
    accessCredential: "scanner-token"
```

## Rollback Plan

If migration issues occur:

```bash
# 1. Keep old provider active during migration
kubectl get provider provider-harbor-upjet

# 2. Test native provider in parallel
kubectl apply -f native-test-resources.yaml

# 3. If issues found, remove native provider
kubectl delete provider provider-harbor

# 4. Continue using Upjet provider
```

## Migration Checklist

- [ ] Backup existing resources
- [ ] Install native provider
- [ ] Create new ProviderConfig with correct API group
- [ ] Convert resource manifests to new API groups
- [ ] Test resource creation with native provider
- [ ] Verify Harbor operations work correctly
- [ ] Update CI/CD pipelines to use new resources
- [ ] Monitor provider memory usage (should be ~5-10MB)
- [ ] Remove old Upjet provider (optional)

## Support

For migration issues:
- Check provider logs: `kubectl logs -n crossplane-system deployment/provider-harbor`
- Review resource status: `kubectl describe project <project-name>`
- Open issue: https://github.com/rossigee/provider-harbor/issues