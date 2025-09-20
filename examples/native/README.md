# Native Harbor Provider Examples

This directory contains examples for the native Harbor Crossplane provider.

## Overview

These examples demonstrate how to use the native Harbor provider, which directly uses the Harbor Go client instead of wrapping the Terraform provider.

## Benefits of Native Provider

- **Performance**: Direct API calls instead of Terraform execution
- **Reduced Dependencies**: No Terraform binaries or state management
- **Better Error Handling**: Native Go error handling and debugging
- **Kubernetes Native**: Full integration with Kubernetes patterns
- **Memory Efficiency**: Lower memory footprint (~10MB vs 50-100MB)

## Prerequisites

1. A Harbor instance (version 2.0+)
2. Admin credentials for Harbor
3. Crossplane installed in your cluster

## Quick Start

1. **Configure credentials:**
   ```bash
   kubectl apply -f providerconfig.yaml
   ```

2. **Create a Harbor project:**
   ```bash
   kubectl apply -f project.yaml
   ```

3. **Create a Harbor user:**
   ```bash
   kubectl apply -f user.yaml
   ```

4. **Create a registry endpoint:**
   ```bash
   kubectl apply -f registry.yaml
   ```

## Credentials Format

The Harbor credentials secret should contain:

```yaml
stringData:
  url: "https://your-harbor-instance.com"
  username: "admin"
  password: "your-password"
  insecure: "false"  # Optional: skip TLS verification
```

## Resource Status

You can check the status of created resources:

```bash
# Check project status
kubectl describe project example-project

# Check user status
kubectl describe user example-user

# Check registry status
kubectl describe registry example-registry
```

## Migration from Terraform Provider

If migrating from the Terraform-based provider:

1. The API groups have changed:
   - Old: `harbor.crossplane.io/v1alpha1`
   - New: `project.harbor.crossplane.io/v1alpha1`, `user.harbor.crossplane.io/v1alpha1`, etc.

2. Some field names may have changed for better Go/Kubernetes conventions

3. Performance and memory usage should be significantly improved

## Troubleshooting

- Check provider logs: `kubectl logs -n crossplane-system deployment/provider-harbor`
- Verify credentials: Ensure the Harbor instance is accessible and credentials are correct
- Check resource events: `kubectl describe <resource-type> <resource-name>`