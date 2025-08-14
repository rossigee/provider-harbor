# Harbor User Examples

This directory contains examples for creating Harbor users with the provider-harbor Crossplane provider.

## Generated Password (Recommended)

The `UserWithGeneratedPassword` resource automatically generates secure random passwords and stores them in Kubernetes secrets:

```yaml
apiVersion: user.harbor.crossplane.io/v1alpha1
kind: UserWithGeneratedPassword
metadata:
  name: harbor-user-with-generated-password
spec:
  forProvider:
    username: testuser
    email: testuser@example.com
    fullName: Test User
    admin: false
    comment: "Harbor user with automatically generated password"
    
    # Generate a secure random password and store it in a secret
    generatePasswordInSecret:
      name: harbor-user-password
      namespace: default  # optional, defaults to resource namespace
      key: password       # optional, defaults to "password"
      length: 20          # optional, defaults to 16, min 8, max 128
      
  providerConfigRef:
    name: default
```

### How It Works

The controller uses a **two-phase approach** to ensure proper ordering:

1. **Phase 1 - Secret Creation**: Generates a cryptographically secure random password and creates the Kubernetes Secret
2. **Phase 2 - User Creation**: Creates the underlying Harbor User with reference to the existing password secret
3. **Lifecycle Management**: Both secret and user are owned by the `UserWithGeneratedPassword` resource for automatic cleanup

This ensures the Harbor User is only created **after** the password secret exists, preventing any orphaned resources if user creation fails.

### Created Resources

The controller creates:

1. **Password Secret**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: harbor-user-password
  namespace: default
  labels:
    app.kubernetes.io/managed-by: provider-harbor
    harbor.crossplane.io/user: harbor-user-with-generated-password
    harbor.crossplane.io/generated: "true"
    harbor.crossplane.io/secret-type: password
type: Opaque
data:
  password: <base64-encoded-random-password>
```

2. **Underlying User Resource**: A regular `User` resource that references the generated password secret

### Consuming the Password

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: app-using-harbor-user
spec:
  containers:
  - name: app
    image: myapp:latest
    env:
    - name: HARBOR_PASSWORD
      valueFrom:
        secretKeyRef:
          name: harbor-user-password
          key: password
    - name: HARBOR_USERNAME
      value: testuser
```

### Configuration Options

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `name` | Yes | - | Name of the secret to create |
| `namespace` | No | Resource namespace or "default" | Namespace for the secret |
| `key` | No | "password" | Key within the secret |
| `length` | No | 16 | Password length (8-128) |

### Benefits

- ✅ **Truly Random**: Cryptographically secure random password generation
- ✅ **No Manual Secrets**: No need to pre-create password secrets  
- ✅ **Configurable**: Control password length and secret location
- ✅ **Clean Lifecycle**: Automatic cleanup when resource is deleted
- ✅ **Secure Storage**: Passwords stored in Kubernetes secrets
- ✅ **Owner References**: Proper resource ownership and cleanup
- ✅ **Safe Ordering**: Secret created first, then user - no orphaned resources
- ✅ **Failure Resilient**: If user creation fails, no orphaned secrets are left behind

## Traditional Password Example

You can still create users with pre-existing passwords using the regular `User` resource:

```yaml
apiVersion: user.harbor.crossplane.io/v1alpha1
kind: User
metadata:
  name: harbor-user-with-existing-password
spec:
  forProvider:
    username: existinguser
    email: existinguser@example.com
    fullName: Existing User
    passwordSecretRef:
      name: my-existing-secret
      key: password
  providerConfigRef:
    name: default
```

This approach is useful when:
- You need to use existing passwords
- External systems manage password rotation  
- You have specific password requirements

## Architecture Comparison

| Approach | Password Type | Lifecycle | Complexity | 
|----------|---------------|-----------|------------|
| **UserWithGeneratedPassword** | Secure random | Automatic | Low |
| Manual User + Secret | User-defined | Manual | Medium |