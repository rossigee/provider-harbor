# Harbor Provider Migration Guide: Terraform → Native

## Overview

This guide shows how to migrate from the Terraform-based Harbor provider to the native Go implementation (v0.8.0).

## Key Benefits

| Aspect | Terraform Provider | Native Provider v0.8.0 | Improvement |
|--------|-------------------|------------------------|-------------|
| **Memory Usage** | ~1GB | ~150MB | **85% reduction** |
| **Startup Time** | 30-60s | 5-10s | **80% faster** |
| **CPU Requests** | 500m | 100m | **80% less** |
| **Architecture** | Multi-layer stack | Direct API | **Simplified** |

## Migration Steps

### 1. Update Provider Package

**Before (Terraform):**
```yaml
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-harbor
spec:
  package: ghcr.io/rossigee/provider-harbor:v0.7.0  # Terraform-based
```

**After (Native):**
```yaml
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-harbor
spec:
  package: ghcr.io/rossigee/provider-harbor:v0.8.0  # Native implementation
  runtimeConfigRef:
    name: provider-harbor-native-config
```

### 2. Update Resource Limits

**Before (Terraform):**
```yaml
resources:
  requests:
    memory: "1Gi"     # High memory for Terraform stack
    cpu: "500m"
  limits:
    memory: "2Gi"
    cpu: "1000m"
```

**After (Native):**
```yaml
resources:
  requests:
    memory: "200Mi"   # 80% reduction
    cpu: "100m"       # 80% reduction
  limits:
    memory: "500Mi"   # 75% reduction
    cpu: "500m"       # 50% reduction
```

### 3. No Changes to Resources

**Important**: Existing Harbor Project resources work unchanged!

```yaml
# This works exactly the same with native provider
apiVersion: project.harbor.crossplane.io/v1alpha1
kind: Project
metadata:
  name: my-project
spec:
  forProvider:
    name: "my-harbor-project"
    public: false
  providerConfigRef:
    name: harbor-config  # Same ProviderConfig format
```

### 4. ProviderConfig Compatibility

**No changes needed** - existing ProviderConfigs work with native provider:

```yaml
# Same format for both Terraform and Native providers
apiVersion: harbor.crossplane.io/v1beta1
kind: ProviderConfig
metadata:
  name: harbor-config
spec:
  credentials:
    source: Secret
    secretRef:
      namespace: crossplane-system
      name: harbor-credentials
      key: credentials
```

## Testing Migration

1. **Deploy native provider alongside existing:**
   ```bash
   kubectl apply -f examples/native-provider-test.yaml
   ```

2. **Create test project with native provider:**
   ```bash
   kubectl apply -f - <<EOF
   apiVersion: project.harbor.crossplane.io/v1alpha1
   kind: Project
   metadata:
     name: native-test
   spec:
     forProvider:
       name: "native-test-project"
       public: false
     providerConfigRef:
       name: harbor-native-config
   EOF
   ```

3. **Monitor resource usage:**
   ```bash
   kubectl top pods -n crossplane-system | grep provider-harbor
   ```

4. **Check provider status:**
   ```bash
   kubectl get providers
   kubectl describe provider provider-harbor-native
   ```

## Rollback Plan

If issues occur, rollback is simple:

```bash
# Switch back to Terraform provider
kubectl patch provider provider-harbor --type='merge' -p='{"spec":{"package":"ghcr.io/rossigee/provider-harbor:v0.7.0"}}'
```

## Verification

**Memory Usage Comparison:**
```bash
# Before (Terraform): ~1GB
# After (Native): ~150MB
kubectl top pods -n crossplane-system | grep provider-harbor
```

**Startup Time:**
```bash
# Native provider starts 5-10x faster
kubectl logs -n crossplane-system deployment/provider-harbor-native | grep "starting"
```

## Troubleshooting

### Common Issues

1. **OOMKilled errors**: Should not occur with native provider (150MB vs 1GB)
2. **Slow startup**: Native provider starts in 5-10s vs 30-60s for Terraform
3. **Resource conflicts**: Deploy native provider with different name for testing

### Debug Commands

```bash
# Check provider status
kubectl get providers -o wide

# Check pod resources
kubectl describe pod -n crossplane-system -l pkg.crossplane.io/provider=provider-harbor

# View provider logs
kubectl logs -n crossplane-system deployment/provider-harbor-native -f

# Check memory usage
kubectl top pods -n crossplane-system | grep provider-harbor
```

## Success Metrics

After migration, you should see:

- ✅ **Memory usage**: ~150MB (down from ~1GB)
- ✅ **Startup time**: 5-10 seconds (down from 30-60s)
- ✅ **CPU usage**: Significantly reduced
- ✅ **Resource conflicts**: Eliminated due to lower resource requirements
- ✅ **Identical functionality**: All Harbor Project operations work unchanged

The native implementation provides the same functionality with dramatically improved resource efficiency!