# Harbor Crossplane Provider

## Overview
The Harbor Crossplane provider enables declarative management of Harbor container registry resources through Kubernetes Custom Resource Definitions (CRDs). This provider is based on the terraform-provider-harbor and provides GitOps-style management for projects, users, robot accounts, and security policies.

## Current Status (2025-07-14)

### ✅ Working Features
- **Project Management**: Create/update/delete Harbor projects with resource quotas
- **User Management**: Basic user lifecycle (create, assign to projects)
- **Project Memberships**: Role-based access control for project members
- **Robot Account Management**: Full CRD support for system and project-level robot accounts
- **Labels**: Project and global label management
- **Registry Management**: Basic registry configuration

### ⚠️ Known Issues & Limitations

#### 1. **API Version Migration Required** ⚠️ **BREAKING**
**Issue**: Deprecated API versions in production deployments

**Details**:
- **Deprecated**: `harbor.crossplane.io/v1alpha1` (used in ~25 production files)
- **Current**: `project.harbor.crossplane.io/v1alpha1` 
- **Affected Resources**: Users, Projects, Memberships across production
- **Migration Required**: Manual update of all YAML manifests

**Files Requiring Updates**:
- `/clusters/golder-secops/harbor-management/projects/*.yaml` (13 files)
- `/clusters/golder-secops/harbor-management/users/*.yaml` (6 files)
- `/clusters/golder-secops/harbor-management/memberships/*.yaml` (8 files)

#### 2. **Provider Version Conflicts** 🔧 **RESOLVED**
**Issue**: Custom provider image causing revision conflicts (FIXED 2025-07-14)

**Resolution**:
- **Previous**: `harbor.golder.lan/library/provider-harbor:v0.2.3-fixed` falling back to v0.2.2
- **Fixed**: Updated to use upstream `xpkg.upbound.io/globallogicuki/provider-harbor:v0.2.2` directly
- **Error Resolved**: "cannot establish control of object" provider revision conflicts

#### 3. **Schema Field Changes** ⚠️ **SCHEMA**
**Issue**: Field name inconsistencies between API versions

**Known Changes**:
- `storageLimit` → `storageQuota` (in some contexts)
- Resource structure variations between v1alpha1 versions
- **Validation Needed**: Full field mapping comparison required

#### 4. **Limited Webhook Support** ⚠️ **FEATURE**
**Issue**: Basic webhook CRD support but limited configuration options

**Limitations**:
- Limited webhook event type coverage
- Basic authentication methods only
- No webhook secret rotation capabilities

### Current Deployment Architecture

#### Production Configuration
```yaml
# Current working configuration
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-harbor
spec:
  package: xpkg.upbound.io/globallogicuki/provider-harbor:v0.2.2
```

#### ProviderConfig Example
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
      name: harbor-creds
      key: credentials
```

## Required Actions

### **Immediate Priority**

1. **API Version Migration**
   - Update all 25+ production YAML files to use correct API versions
   - Create migration validation script
   - Test in staging before production deployment

2. **Provider Health Verification**
   - Confirm provider becomes healthy after image fix
   - Validate resource synchronization

### **Short-term Enhancements**

1. **Schema Documentation**
   - Document field mapping between API versions
   - Create migration guide with examples
   - Add validation for common misconfigurations

2. **Monitoring & Observability**
   - Add provider-specific Prometheus metrics
   - Create Grafana dashboards for Harbor resource health
   - Implement alerting for resource sync failures

### **Long-term Roadmap**

1. **Provider Enhancements**
   - Add advanced webhook configuration support
   - Implement vulnerability scanning policy management

2. **Security Improvements**
   - Add certificate-based authentication support
   - Implement secret rotation for robot accounts
   - Add audit logging for Harbor resource changes

3. **Integration Features**
   - OCI artifact management
   - Notary signing integration
   - Replication policy automation

## Development Notes

### Build Requirements
- Based on terraform-provider-harbor
- Requires Harbor API v2.0+
- Uses Upjet for code generation

### Testing Strategy
- Unit tests for controller logic
- Integration tests with Harbor test instance
- E2E tests for complete resource lifecycle

### Contribution Guidelines
- Follow Crossplane provider conventions
- Add comprehensive examples for new CRDs
- Include migration documentation for breaking changes

## References
- **Upstream**: [globallogicuki/provider-harbor](https://github.com/globallogicuki/provider-harbor)
- **Documentation**: [doc.crds.dev](https://doc.crds.dev/github.com/globallogicuki/provider-harbor)
- **Harbor API**: [Harbor REST API v2.0](https://goharbor.io/docs/2.0.0/build-customize-contribute/configure-swagger/)

## Issue Tracking
- API Migration: Required for production stability
- Provider Health: Critical for resource synchronization

*Last Updated: 2025-07-17*