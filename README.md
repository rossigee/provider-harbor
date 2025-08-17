# Provider Harbor

**✅ BUILD STATUS: STABLE** - Successfully migrated to modern stable dependencies (v0.3.0)

`provider-harbor` is a [Crossplane](https://crossplane.io/) provider that
is built using [Upjet](https://github.com/crossplane/upjet) v1.9.0 and exposes XRM-conformant managed resources 
for the [Harbor](https://goharbor.io/) container registry API.

## Migration Success ✅
**2025-01-27**: Successfully migrated from RC dependencies to stable versions
- **Upjet**: v0.11.0-rc → v1.9.0 (stable)
- **Crossplane Runtime**: v1.16.0-rc → v1.20.0 (stable)
- **Generated Resources**: 20 Harbor resources with modern Crossplane patterns
- **Build Status**: ✅ Successful compilation and generation

## Features
- **Registry Management**: External container registry integration (Docker Hub, AWS, Azure, etc.)
- **Project Management**: Projects, repositories, and member access control
- **Security Scanning**: Vulnerability scanning, CVE allowlists, and compliance policies
- **Webhooks**: HTTP/Slack notifications for project events
- **Replication**: Cross-registry replication with filtering
- **Robot Accounts**: Automated service accounts for CI/CD with Docker config JSON support

## Container Registry
- **Primary**: `ghcr.io/rossigee/provider-harbor:v0.5.3`
- **Harbor**: Available via environment configuration
- **Upbound**: Available via environment configuration

## Getting Started

Install the provider by using the following command:
```
up ctp provider install ghcr.io/rossigee/provider-harbor:v0.5.3
```

Alternatively, you can use declarative installation:
```
cat <<EOF | kubectl apply -f -
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-harbor
spec:
  package: ghcr.io/rossigee/provider-harbor:v0.5.3
EOF
```

Notice that in this example Provider resource is referencing ControllerConfig with debug enabled.

You can see the API reference [here](https://doc.crds.dev/github.com/rossigee/provider-harbor).

## Docker Config JSON Support

**New in v0.5.3**: RobotAccount resources now support creating Docker config JSON style secrets while maintaining 100% backward compatibility.

### Enhanced Connection Details

All RobotAccount resources now provide:
- **Legacy fields**: `username`, `password`, `robot_id` (unchanged)
- **Docker config helpers**: `docker-config-template`, `docker-auth`, etc. (new)

### Quick Example

```yaml
apiVersion: robotaccount.harbor.crossplane.io/v1alpha1
kind: RobotAccount
metadata:
  name: docker-config-robot
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

The resulting secret contains both legacy fields and Docker config JSON helpers that can be used to create `kubernetes.io/dockerconfigjson` secrets for image pull authentication.

📖 **Full Documentation**: [RobotAccount Docker Config JSON Guide](docs/ROBOTACCOUNT-DOCKER-CONFIG.md)

## Provider Config
Note, that the ProviderConfig uses basic auth and requires a local user. A robot user can be used, 
but has restrictions around what can be created (e.g. a Robot cannot create other Robots).

## Developing

Run code-generation pipeline:
```console
go run cmd/generator/main.go "$PWD"
```

Run against a Kubernetes cluster:

```console
make run
```

Build, push, and install:

```console
make all
```

Build binary:

```console
make build
```

## Report a Bug

For filing bugs, suggesting improvements, or requesting new features, please
open an [issue](https://github.com/rossigee/provider-harbor/issues).
