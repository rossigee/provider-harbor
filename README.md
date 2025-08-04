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
- **Robot Accounts**: Automated service accounts for CI/CD

## Container Registry
- **Primary**: `ghcr.io/rossigee/provider-harbor:v0.3.0`
- **Harbor**: Available via environment configuration
- **Upbound**: Available via environment configuration

## Getting Started

Install the provider by using the following command:
```
up ctp provider install ghcr.io/rossigee/provider-harbor:v0.3.0
```

Alternatively, you can use declarative installation:
```
cat <<EOF | kubectl apply -f -
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-harbor
spec:
  package: ghcr.io/rossigee/provider-harbor:v0.3.0
EOF
```

Notice that in this example Provider resource is referencing ControllerConfig with debug enabled.

You can see the API reference [here](https://doc.crds.dev/github.com/globallogicuki/provider-harbor).

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
open an [issue](https://github.com/globallogicuki/provider-harbor/issues).
