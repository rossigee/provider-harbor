# RobotAccount Docker Config JSON Support

This document describes the enhanced Docker config JSON support for Harbor RobotAccount resources in provider-harbor v0.3.0+.

## Overview

The Harbor provider's RobotAccount resource now supports creating Docker config JSON style secrets while maintaining 100% backward compatibility with existing deployments.

## Key Features

- **100% Backward Compatible**: All existing RobotAccount resources continue working unchanged
- **Enhanced Connection Details**: New robots automatically get additional Docker config helpers
- **Docker Config JSON Ready**: Pre-computed credentials for `kubernetes.io/dockerconfigjson` secrets
- **Template-Based**: Registry URL substitution for flexible deployment patterns
- **Composition Friendly**: Designed for use with Crossplane Compositions and Functions

## Connection Details

### Legacy Format (Continues to Work)

All existing RobotAccount resources provide these connection details:

```yaml
# Secret contents - legacy format
username: robot$my-robot       # Robot account full name
password: generated-secret     # Robot account password
robot_id: "123"               # Robot account ID
```

### Enhanced Format (New)

New RobotAccount resources provide legacy fields PLUS Docker config helpers:

```yaml
# Secret contents - enhanced format
# Legacy fields (unchanged)
username: robot$my-robot       # Robot account full name
password: generated-secret     # Robot account password  
robot_id: "123"               # Robot account ID

# New Docker config helpers
docker-username: robot$my-robot                    # Same as username
docker-password: generated-secret                  # Same as password
docker-auth: cm9ib3QkbXktcm9ib3Q6Z2VuZXJhdGVkLXNlY3JldA==  # Base64(username:password)
docker-config-template: |                         # Docker config JSON template
  {
    "auths": {
      "REGISTRY_URL_PLACEHOLDER": {
        "username": "robot$my-robot",
        "password": "generated-secret",
        "auth": "cm9ib3QkbXktcm9ib3Q6Z2VuZXJhdGVkLXNlY3JldA=="
      }
    }
  }
```

## Usage Patterns

### Pattern 1: Legacy Usage (Unchanged)

```yaml
apiVersion: robotaccount.harbor.crossplane.io/v1alpha1
kind: RobotAccount
metadata:
  name: legacy-robot
spec:
  forProvider:
    level: project
    name: my-robot
    permissions:
      - access:
          - action: pull
            resource: repository
        kind: project
        namespace: myproject
  writeConnectionSecretToRef:
    name: robot-credentials
    namespace: default
```

**Result**: Secret with individual fields (`username`, `password`, `robot_id`)

### Pattern 2: Enhanced Usage

```yaml
apiVersion: robotaccount.harbor.crossplane.io/v1alpha1
kind: RobotAccount
metadata:
  name: enhanced-robot
spec:
  forProvider:
    level: project
    name: my-robot
    permissions:
      - access:
          - action: pull
            resource: repository
          - action: push
            resource: repository
        kind: project
        namespace: myproject
  writeConnectionSecretToRef:
    name: enhanced-credentials
    namespace: default
```

**Result**: Secret with legacy fields + Docker config helpers

### Pattern 3: Docker Config JSON with Composition

```yaml
apiVersion: apiextensions.crossplane.io/v1
kind: Composition
metadata:
  name: harbor-docker-config
spec:
  compositeTypeRef:
    apiVersion: example.com/v1alpha1
    kind: XHarborAuth
  resources:
  - name: robot-account
    base:
      apiVersion: robotaccount.harbor.crossplane.io/v1alpha1
      kind: RobotAccount
      spec:
        writeConnectionSecretToRef:
          name: robot-credentials
          namespace: default
    patches:
    - type: FromCompositeFieldPath
      fromFieldPath: spec.project
      toFieldPath: spec.forProvider.permissions[0].namespace
    connectionDetails:
    - fromConnectionSecretKey: docker-config-template
      name: docker-config-template
  - name: docker-secret
    base:
      apiVersion: v1
      kind: Secret
      metadata:
        namespace: default
      type: kubernetes.io/dockerconfigjson
    patches:
    - type: FromConnectionSecretKey
      fromConnectionSecretKey: docker-config-template
      toFieldPath: data[".dockerconfigjson"]
      transforms:
      - type: string
        string:
          type: Replace
          search: "REGISTRY_URL_PLACEHOLDER"
          replace: "harbor.example.com"
      - type: string
        string:
          type: Convert
          convert: "base64"
```

### Pattern 4: Function-Based Docker Config

```yaml
apiVersion: pkg.crossplane.io/v1beta1
kind: Function
metadata:
  name: harbor-docker-config-fn
spec:
  step: create-docker-config
  input:
    apiVersion: function.example.com/v1beta1
    kind: CreateDockerConfig
    registryURL: harbor.example.com
    credentialsFrom:
      secretRef:
        name: enhanced-credentials
        key: docker-config-template
    outputSecret:
      name: harbor-docker-config
      type: kubernetes.io/dockerconfigjson
```

## Migration Guide

### For Existing Users

**No action required!** Your existing RobotAccount resources:
- Continue working exactly as before
- Get the new Docker config helpers automatically
- Can be gradually enhanced with new patterns

### For New Deployments

1. **Simple Deployments**: Use Pattern 1 or 2 as needed
2. **Docker Config JSON**: Use Pattern 3 (Composition) or Pattern 4 (Function)
3. **Enterprise**: Combine with external secret stores via `publishConnectionDetailsTo`

## Docker Config JSON Creation

### Manual Approach

1. Read the `docker-config-template` from the robot credentials secret
2. Replace `REGISTRY_URL_PLACEHOLDER` with your Harbor registry URL
3. Base64 encode the result
4. Create a secret with `type: kubernetes.io/dockerconfigjson`

```bash
# Example bash script
TEMPLATE=$(kubectl get secret enhanced-credentials -o jsonpath='{.data.docker-config-template}' | base64 -d)
DOCKER_CONFIG=$(echo "$TEMPLATE" | sed 's/REGISTRY_URL_PLACEHOLDER/harbor.example.com/g')
kubectl create secret generic harbor-docker-config \
  --type=kubernetes.io/dockerconfigjson \
  --from-literal=.dockerconfigjson="$DOCKER_CONFIG"
```

### Composition Approach

Use the example Composition above to automate the process.

### Function Approach

Create or use a Crossplane Function to handle registry URL substitution and secret creation.

## Security Considerations

- **Credential Rotation**: Robot account passwords can be rotated via Harbor UI/API
- **Secret Scope**: Limit secret access using Kubernetes RBAC  
- **Registry URL**: Ensure registry URLs in Docker config match your Harbor deployment
- **Image Pull Secrets**: Bind Docker config secrets to appropriate ServiceAccounts

## Testing

### Unit Tests

The provider includes comprehensive unit tests for the Docker config functionality:

```bash
cd config/robotaccount
go test -v .
```

### Integration Tests

Test examples are provided in:
- `examples/robotaccount/dockerconfig.yaml`
- `examples-generated/robotaccount/v1alpha1/robotaccount-with-docker-config.yaml`

### Validation

Verify your Docker config JSON secret:

```bash
# Check secret type
kubectl get secret harbor-docker-config -o jsonpath='{.type}'
# Should output: kubernetes.io/dockerconfigjson

# Decode and validate JSON
kubectl get secret harbor-docker-config -o jsonpath='{.data\.dockerconfigjson}' | base64 -d | jq .
# Should show valid Docker config JSON structure
```

## Troubleshooting

### Common Issues

1. **Missing Docker Config Fields**
   - Ensure both `username` and `password` are available in the source credentials
   - Check that the robot account was created successfully

2. **Invalid JSON in Template**
   - Verify the `docker-config-template` field contains valid JSON
   - Check for proper base64 encoding/decoding

3. **Registry URL Mismatch**
   - Ensure the registry URL in your Docker config matches your Harbor deployment
   - Verify harbor.example.com is replaced with your actual registry

4. **Image Pull Failures**
   - Check that the secret is properly bound to the pod's ServiceAccount
   - Verify the robot account has appropriate permissions for the target project/repository

### Debug Commands

```bash
# Check robot account status
kubectl describe robotaccount enhanced-robot

# Examine connection details
kubectl get secret enhanced-credentials -o yaml

# Validate Docker config JSON
kubectl get secret harbor-docker-config -o jsonpath='{.data\.dockerconfigjson}' | base64 -d | python -m json.tool

# Test image pull with secret
kubectl create pod test-pod --image=harbor.example.com/myproject/myimage:latest --dry-run=client -o yaml | kubectl apply -f -
kubectl patch serviceaccount default -p '{"imagePullSecrets": [{"name": "harbor-docker-config"}]}'
```

## Changelog

### v0.3.0
- Added Docker config JSON support to RobotAccount resources
- Enhanced connection details with Docker config helpers  
- Added `docker-config-template` for registry URL substitution
- Maintained 100% backward compatibility
- Added comprehensive unit tests and examples

## See Also

- [Harbor Robot Account Documentation](https://goharbor.io/docs/latest/administration/robot-accounts/)
- [Kubernetes Docker Config Secrets](https://kubernetes.io/docs/concepts/configuration/secret/#docker-config-secrets)
- [Crossplane Composition Guide](https://docs.crossplane.io/latest/concepts/compositions/)
- [Crossplane Functions](https://docs.crossplane.io/latest/concepts/functions/)