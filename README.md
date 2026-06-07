# Provider Harbor

[![Build](https://github.com/rossigee/provider-harbor/actions/workflows/ci.yml/badge.svg)](https://github.com/rossigee/provider-harbor/actions/workflows/ci.yml)

**✅ V2 NATIVE PROVIDER: PRODUCTION READY** - Fully v2-native Crossplane provider with comprehensive Harbor API coverage

`provider-harbor` is a [Crossplane](https://crossplane.io/) v2-native provider that provides comprehensive integration with the [Harbor](https://goharbor.io/) container registry API.

## Key Capabilities ✅

- **60% API Coverage**: 12 resource types covering 90+ Harbor API endpoints
- **Production Ready**: 65+ unit & integration tests with error handling and retries
- **Enterprise Features**: Replication policies, retention rules, webhook automation
- **Security First**: Content trust, vulnerability scanning, image signature verification
- **Pure Go**: Direct Harbor API integration without Terraform overhead
- **Cloud Native**: Namespaced resources for multi-tenant Kubernetes deployments

## Features

**Resource Management**
- **Projects**: Complete project lifecycle with security policies and member access control
- **Repositories**: Repository management with artifact tracking and cleanup policies
- **Artifacts**: Container image/artifact management with vulnerability data
- **Members**: Project membership, role-based access control, team management

**Automation & Workflow**
- **Robots**: CI/CD robot accounts with scoped permissions and expiration policies
- **Webhooks**: Event-driven automation for scan completion, push events, etc.
- **Replication**: Cross-registry image synchronization with filtering and scheduling
- **Retention**: Automated artifact cleanup based on age, count, or custom rules

**Security & Compliance**
- **Scanning**: Vulnerability scans with severity levels and CVE allowlists
- **Registry Management**: External registry configuration for replication
- **Scanner Registration**: Trivy, Clair, or custom scanner integration
- **Users & Roles**: User management with fine-grained access controls

## Container Registry
- **Primary**: `ghcr.io/rossigee/provider-harbor:v0.13.0`
- **Harbor**: Available via environment configuration
- **Upbound**: Available via environment configuration

## Getting Started

Install the provider by using the following command:
```bash
kubectl crossplane install provider ghcr.io/rossigee/provider-harbor:v0.13.0
```

Alternatively, you can use declarative installation:
```yaml
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-harbor
spec:
  package: ghcr.io/rossigee/provider-harbor:v0.13.0
```

## Provider Configuration

Create a ProviderConfig with Harbor credentials:

### V2 Namespaced ProviderConfig (Recommended)
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
---
apiVersion: harbor.m.crossplane.io/v1beta1
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

### Legacy ProviderConfig (Backward Compatibility)
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

### Harbor Project with Security Policies
```yaml
apiVersion: project.harbor.crossplane.io/v1beta1
kind: Project
metadata:
  name: production
spec:
  forProvider:
    name: production
    public: false
    enableContentTrust: true
    enableContentTrustCosign: true
    autoScanImages: true
    preventVulnerableImages: true
    severity: high  # Only allow critical/high
    cveAllowlist:
      - "CVE-2024-1234"  # Whitelist specific CVEs
    metadata:
      description: "Production container images"
      environment: "prod"
  providerConfigRef:
    name: default
  deletionPolicy: Delete
```

### CI/CD Robot Account
```yaml
apiVersion: robot.harbor.crossplane.io/v1beta1
kind: Robot
metadata:
  name: gitlab-ci
spec:
  forProvider:
    name: gitlab-ci-robot
    projectID: "1"
    expiresIn: 7776000  # 90 days
    permissions:
      - namespace: project
        access: push
      - namespace: repository
        access: pull
  providerConfigRef:
    name: default
  deletionPolicy: Delete
```

### Event-Driven Webhook
```yaml
apiVersion: webhook.harbor.crossplane.io/v1beta1
kind: Webhook
metadata:
  name: scan-notifications
spec:
  forProvider:
    projectID: "1"
    name: scan-notifications
    description: Send notifications on scan completion
    url: https://my-system.example.com/webhooks/scan
    eventTypes:
      - SCAN_IMAGE_COMPLETED
    authHeader: Bearer my-auth-token
    skipCertVerify: false
    enabled: true
  providerConfigRef:
    name: default
  deletionPolicy: Delete
```

### Cross-Registry Image Replication
```yaml
apiVersion: replication.harbor.crossplane.io/v1beta1
kind: Replication
metadata:
  name: backup-registry
spec:
  forProvider:
    name: backup-replica
    description: Replicate to backup registry
    destinationRegistry:
      name: backup
      url: https://backup.example.com
    filters:
      - type: name
        value: "production/*"
    trigger: scheduled
    deleteSourceTag: false
    override: true
    enabled: true
  providerConfigRef:
    name: default
  deletionPolicy: Delete
```

### Artifact Retention Policy
```yaml
apiVersion: retention.harbor.crossplane.io/v1beta1
kind: Retention
metadata:
  name: cleanup-old
spec:
  forProvider:
    projectID: "1"
    description: Remove artifacts older than 30 days
    rules:
      - ruleType: latestPushedK
        parameters:
          latestPushedK: "10"
    trigger: scheduled
    enabled: true
  providerConfigRef:
    name: default
  deletionPolicy: Delete
```

### Scanner Registration (Trivy)
```yaml
apiVersion: scanner.harbor.crossplane.io/v1beta1
kind: ScannerRegistration
metadata:
  name: trivy
spec:
  forProvider:
    name: trivy
    description: Trivy vulnerability scanner
    url: http://trivy.harbor.svc.cluster.local:4954
    auth: Bearer
    accessCredential: my-scanner-token
  providerConfigRef:
    name: default
  deletionPolicy: Delete
```

## API Coverage Matrix

| Resource | Create | Read | Update | Delete | Special | Status |
|----------|--------|------|--------|--------|---------|--------|
| Project | ✅ | ✅ | ✅ | ✅ | - | Complete |
| Repository | ❌ | ✅ | ✅ | ✅ | List | Complete |
| Artifact | ❌ | ✅ | ❌ | ✅ | GetVulnerabilities, Scan | Complete |
| Member | ✅ | ✅ | ✅ | ✅ | - | Complete |
| Scan | ❌ | ✅ | ❌ | ✅ | Trigger, Stop | Complete |
| Robot | ✅ | ✅ | ✅ | ✅ | - | Complete |
| Webhook | ✅ | ✅ | ✅ | ✅ | - | Complete |
| Registry | ✅ | ✅ | ✅ | ✅ | - | Complete |
| Scanner | ✅ | ✅ | ✅ | ✅ | - | Complete |
| User | ✅ | ✅ | ✅ | ✅ | - | Complete |
| Replication | ✅ | ✅ | ✅ | ✅ | Trigger, ListExecutions | Complete |
| Retention | ✅ | ✅ | ✅ | ✅ | - | Complete |

## Architecture

### Controller Pattern

Each resource follows the Crossplane managed resource pattern:

```
ProviderConfig (Harbor credentials)
    ↓
Connector (establishes Harbor client)
    ↓
External Controller (Observe/Create/Update/Delete)
    ↓
Harbor Go Client (REST with exponential backoff)
```

### Error Handling

- **Exponential backoff retry**: Transient errors retried up to 3x with 100ms-5s delays
- **Transient error detection**: Network timeouts, service unavailable, rate limiting
- **Connection pooling**: Reusable clients per ProviderConfig
- **Proper cleanup**: Graceful disconnection and resource finalization

### Project Structure

```
provider-harbor/
├── apis/
│   ├── artifact/v1beta1/        # Artifact CRD
│   ├── member/v1beta1/          # Member CRD
│   ├── project/v1beta1/         # Project CRD
│   ├── registry/v1beta1/        # Registry CRD
│   ├── replication/v1beta1/     # Replication CRD
│   ├── retention/v1beta1/       # Retention CRD
│   ├── robot/v1beta1/           # Robot CRD
│   ├── scan/v1beta1/            # Scan CRD
│   ├── scanner/v1beta1/         # Scanner CRD
│   ├── user/v1beta1/            # User CRD
│   ├── webhook/v1beta1/         # Webhook CRD
│   └── v1beta1/                 # ProviderConfig
├── cmd/provider/                # Provider binary
├── internal/
│   ├── clients/                 # Harbor API client
│   │   ├── harbor.go           # Client implementation
│   │   └── mock.go             # Mock for testing
│   └── controller/              # 12 resource controllers
├── examples/                    # Resource examples
└── test/                        # Test suite (65+ tests)
```

## Testing

### Unit Tests (55+ tests)
- Type safety and interface validation
- Error handling paths for all controller methods
- Resource CRUD operation validation

```bash
go test ./internal/controller/... -v
```

### Integration Tests (5+ workflows)
- Project creation/update/deletion
- Robot account lifecycle
- Repository management
- Artifact operations
- Member access control

```bash
go test ./internal/controller -v -run Integration
```

### Run All Tests
```bash
go test ./... -v
```

## Development

### Prerequisites
- Go 1.26.3+
- golangci-lint 2.12.2+
- Controller-gen (code generation)
- Docker (for building images)

### Build and Test

```bash
# Run all tests
go test ./... -v

# Build provider binary
go build ./cmd/provider

# Run provider locally
./cmd/provider/provider

# Build container image
docker build -f cluster/images/xpkg.Dockerfile . \
  -t ghcr.io/rossigee/provider-harbor:latest
```

### Code Generation

Generated code files (deepcopy, register) are auto-generated:

```bash
# Generate deepcopy and register methods
controller-gen object:headerFile="hack/boilerplate.go.txt" \
  paths="./apis/..."
```

## Compatibility

- **Crossplane**: v1.20.0+
- **Kubernetes**: v1.28+
- **Harbor**: v2.0+
- **Go**: 1.26.3+

## Known Limitations

- Artifacts cannot be created directly (created by Harbor during push)
- Repositories created by Harbor during artifact push (not directly creatable)
- Scans cannot be created directly (triggered on existing artifacts)

## Report a Bug

For filing bugs, suggesting improvements, or requesting new features, please
open an [issue](https://github.com/rossigee/provider-harbor/issues).
