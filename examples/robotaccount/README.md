# RobotAccount Docker Config JSON Support

The Harbor provider's RobotAccount resource now supports creating Docker config JSON style secrets, while maintaining full backward compatibility with the legacy individual field format.

## Backward Compatibility

**All existing RobotAccount resources continue to work unchanged.** The provider maintains 100% backward compatibility.

### Legacy Format (continues to work)

```yaml
apiVersion: robotaccount.harbor.crossplane.io/v1alpha1
kind: RobotAccount
metadata:
  name: legacy-robot
spec:
  forProvider:
    # ... robot account configuration
  writeConnectionSecretToRef:
    name: robot-credentials
    namespace: default
```

**Secret Contents (Legacy):**
- `username`: Robot account full name (e.g., `robot$my-robot`)
- `password`: Robot account secret
- `robot_id`: Robot account ID

## Enhanced Format (Docker Config JSON Support)

The enhanced format provides the same legacy fields PLUS additional Docker config helpers:

```yaml
apiVersion: robotaccount.harbor.crossplane.io/v1alpha1  
kind: RobotAccount
metadata:
  name: enhanced-robot
spec:
  forProvider:
    # ... robot account configuration
  writeConnectionSecretToRef:
    name: enhanced-credentials
    namespace: default
```

**Secret Contents (Enhanced):**
- `username`: Robot account full name *(legacy)*
- `password`: Robot account secret *(legacy)*
- `robot_id`: Robot account ID *(legacy)*
- `docker-username`: Same as username *(new)*
- `docker-password`: Same as password *(new)*
- `docker-auth`: Base64 encoded `username:password` *(new)*
- `docker-config-template`: Docker config JSON template *(new)*

## Creating Docker Config JSON Secrets

The provider provides the building blocks for Docker config JSON secrets. You can create proper `kubernetes.io/dockerconfigjson` secrets using:

### Option 1: Composition/Function Pattern

Create a Composition or Function that:
1. Reads the robot account connection details
2. Substitutes the registry URL in the template
3. Creates a `kubernetes.io/dockerconfigjson` secret

### Option 2: Manual Template Substitution

Use the `docker-config-template` field and replace `REGISTRY_URL_PLACEHOLDER` with your Harbor registry URL.

**Template Content:**
```json
{
  "auths": {
    "REGISTRY_URL_PLACEHOLDER": {
      "username": "robot$my-robot",
      "password": "generated-secret", 
      "auth": "base64-encoded-credentials"
    }
  }
}
```

**Final Docker Config JSON:**
```json
{
  "auths": {
    "harbor.example.com": {
      "username": "robot$my-robot",
      "password": "generated-secret",
      "auth": "base64-encoded-credentials"
    }
  }
}
```

## Usage Examples

### Using with Image Pull Secrets

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-pod
spec:
  containers:
  - name: app
    image: harbor.example.com/myproject/myapp:latest
  imagePullSecrets:
  - name: harbor-docker-config  # The kubernetes.io/dockerconfigjson secret
```

### Integration with Crossplane Functions

```yaml
apiVersion: pkg.crossplane.io/v1beta1
kind: Function
metadata:
  name: create-docker-config
spec:
  # Function that reads robot credentials and creates Docker config JSON
  input:
    credentialsSecret: enhanced-credentials
    registryURL: harbor.example.com
  output:
    secretName: harbor-docker-config
    secretType: kubernetes.io/dockerconfigjson
```

## Migration Guide

**No migration required!** Existing deployments continue to work without changes. 

To adopt the new Docker config JSON capabilities:
1. Existing robots: No changes needed, they automatically get the new fields
2. New robots: Use the enhanced connection details as needed
3. Create Compositions/Functions to leverage Docker config JSON templates

## Benefits

1. **Backward Compatible**: Zero breaking changes
2. **Flexible**: Choose legacy individual fields or Docker config JSON
3. **Efficient**: Pre-computed Docker auth credentials
4. **Standard**: Follows Kubernetes `kubernetes.io/dockerconfigjson` secret format
5. **Composable**: Works with Crossplane Compositions and Functions