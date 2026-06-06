# Production Safety Best Practices

This guide covers implementing production-safe configurations with Crossplane and Harbor.

## Status Conditions

All resources implement Crossplane managed resource conditions:

```yaml
# Status example from kubectl describe project my-project
Status:
  Conditions:
  - lastTransitionTime: "2025-06-06T10:30:00Z"
    message: "Resource is ready"
    reason: Ready
    status: "True"
    type: Ready
  - lastTransitionTime: "2025-06-06T10:30:00Z"
    message: "Resource is synced with external system"
    reason: Synced
    status: "True"
    type: Synced
  AtProvider:
    id: "1"
    name: production
    ...
```

### Condition Types

- **Ready**: Resource exists in Harbor and Crossplane has successfully managed it
- **Synced**: Resource configuration is in sync with Harbor (no drift detected)
- **Failed**: Error state with detailed message (check `.status.conditions[].message`)

### Check Resource Status

```bash
# See all conditions
kubectl describe project my-project

# Watch resource status
kubectl get project -w

# Get status as JSON
kubectl get project my-project -o jsonpath='{.status.conditions}' | jq
```

## Validation Best Practices

### Project Security Configuration

```yaml
# ❌ INSECURE: Public project without scanning
apiVersion: project.harbor.crossplane.io/v1beta1
kind: Project
metadata:
  name: public-app
spec:
  forProvider:
    name: public-app
    public: true  # WARNING: Publicly accessible

# ✅ SECURE: Private project with security controls
apiVersion: project.harbor.crossplane.io/v1beta1
kind: Project
metadata:
  name: production
spec:
  forProvider:
    name: production
    public: false  # Private project
    enableContentTrust: true  # Require signatures
    enableContentTrustCosign: true  # Support Cosign
    autoScanImages: true  # Automatic vulnerability scanning
    preventVulnerableImages: true  # Block deployment of vulnerable images
    severity: critical  # Block critical severity vulnerabilities
    cveAllowlist:  # Whitelist specific CVEs if needed
      - "CVE-2024-12345"  # Justification: Fixed in next patch
```

### Robot Account Security

```yaml
# ❌ WEAK: Robot without expiration
apiVersion: robot.harbor.crossplane.io/v1beta1
kind: Robot
metadata:
  name: ci-robot-weak
spec:
  forProvider:
    name: ci-robot
    projectID: "1"
    # expiresIn: 0  # Never expires - security risk!
    permissions:
      - namespace: project
        access: push

# ✅ STRONG: Robot with time-limited credentials
apiVersion: robot.harbor.crossplane.io/v1beta1
kind: Robot
metadata:
  name: ci-robot-strong
spec:
  forProvider:
    name: ci-robot
    projectID: "1"
    expiresIn: 7776000  # 90 days - requires renewal
    permissions:
      - namespace: project
        access: push
      - namespace: repository
        access: pull
  deletionPolicy: Delete  # Clean up on removal
```

### Webhook Configuration

```yaml
# ✅ SECURE: Webhook with authentication and verification
apiVersion: webhook.harbor.crossplane.io/v1beta1
kind: Webhook
metadata:
  name: secure-webhook
spec:
  forProvider:
    projectID: "1"
    name: scan-notifications
    url: https://my-system.example.com/webhooks/harbor  # HTTPS only
    eventTypes:
      - SCAN_IMAGE_COMPLETED  # Specific events
      - IMAGE_PUSH
    authHeader: Bearer YOUR_AUTH_TOKEN  # Authenticate to webhook
    skipCertVerify: false  # Verify TLS certificates
    enabled: true
  deletionPolicy: Delete
```

## Drift Detection

Crossplane monitors resource drift continuously:

```bash
# Check poll interval (default: 10 minutes)
kubectl get providerconfig default -o yaml | grep pollInterval

# Force immediate reconciliation
kubectl patch project my-project -p '{"metadata":{"annotations":{"crossplane.io/paused":"false"}}}' --type merge

# Monitor for drift in real-time
kubectl logs -l app=provider-harbor -f | grep -i "drift\|synced"
```

## Deletion Safety

### Deletion Policies

```yaml
# ✅ RECOMMENDED: Delete resource in both systems
deletionPolicy: Delete

# ⚠️  CAUTION: Keep Harbor resource, remove Crossplane resource
deletionPolicy: Orphan

# Example: Keep Harbor project but remove Crossplane
apiVersion: project.harbor.crossplane.io/v1beta1
kind: Project
metadata:
  name: keep-in-harbor
spec:
  ...
  deletionPolicy: Orphan  # Harbor project survives kubectl delete
```

### Safe Deletion Flow

```bash
# 1. Verify what will be deleted
kubectl get project my-project -o yaml

# 2. Backup Harbor state if needed
# (export project configuration manually or via Harbor API)

# 3. Delete Crossplane resource
kubectl delete project my-project

# 4. Verify Harbor state after deletion
# (project deleted from Harbor if deletionPolicy: Delete)
```

## Monitoring and Alerts

### Recommended Monitoring

```yaml
# Prometheus rules for monitoring (example)
groups:
- name: harbor-provider
  rules:
  - alert: HarborResourceNotReady
    expr: crossplane_managed_resource_status_condition_ready{provider="harbor"} == 0
    for: 5m
    annotations:
      summary: Harbor resource {{ $labels.resource }} not ready for 5 minutes
      
  - alert: HarborResourceNotSynced
    expr: crossplane_managed_resource_status_condition_synced{provider="harbor"} == 0
    for: 10m
    annotations:
      summary: Harbor resource {{ $labels.resource }} drift detected
```

### Health Checks

```bash
# Check provider health
kubectl get provider provider-harbor -o wide

# Check ProviderConfig connection
kubectl get providerconfig default -o yaml | grep -A5 "status:"

# Verify Harbor connectivity
kubectl logs -l app=provider-harbor --tail=50 | grep -i "connection\|error"
```

## Troubleshooting Guide

### Resource Stuck in Failed State

```bash
# Check error message
kubectl describe project my-project

# View recent logs
kubectl logs -l app=provider-harbor --tail=100

# Common causes:
# 1. Invalid credentials in ProviderConfig
# 2. Harbor API unreachable
# 3. Resource already exists in Harbor with different configuration
# 4. Missing required parameters (e.g., projectID for robot)
```

### Reconciliation Loop Issues

```bash
# Check if resource is paused
kubectl get project -o json | jq '.items[].metadata.annotations'

# Pause resource for investigation
kubectl annotate project my-project crossplane.io/paused=true

# Resume after fixing
kubectl annotate project my-project crossplane.io/paused=false --overwrite
```

### Connection Pool Issues

```bash
# Provider reuses Harbor client connections
# If experiencing timeout issues:

# 1. Check ProviderConfig timeout settings
kubectl get providerconfig default -o yaml

# 2. Restart provider to reset connections
kubectl rollout restart deployment provider-harbor \
  -n crossplane-system

# 3. Monitor connection health
kubectl logs -l app=provider-harbor -f | grep -i "connection"
```

## RBAC and Multi-Tenancy

### Namespace Isolation

```yaml
# Harbor-1 in namespace harbor-prod
apiVersion: harbor.crossplane.io/v1beta1
kind: ProviderConfig
metadata:
  name: production
  namespace: harbor-prod
spec:
  credentials:
    source: Secret
    secretRef:
      name: harbor-prod-creds
      namespace: harbor-prod

---
apiVersion: project.harbor.crossplane.io/v1beta1
kind: Project
metadata:
  name: prod-project
  namespace: harbor-prod  # Isolated to namespace
spec:
  forProvider:
    name: production
  providerConfigRef:
    name: production

# Harbor-2 in namespace harbor-staging
---
apiVersion: harbor.crossplane.io/v1beta1
kind: ProviderConfig
metadata:
  name: staging
  namespace: harbor-staging
spec:
  credentials:
    source: Secret
    secretRef:
      name: harbor-staging-creds
      namespace: harbor-staging

---
apiVersion: project.harbor.crossplane.io/v1beta1
kind: Project
metadata:
  name: staging-project
  namespace: harbor-staging  # Isolated to namespace
spec:
  forProvider:
    name: staging
  providerConfigRef:
    name: staging
```

### RBAC Permissions

```yaml
# Restrict project management to DevOps team
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: harbor-devops
  namespace: harbor-prod
rules:
- apiGroups:
  - project.harbor.crossplane.io
  resources:
  - projects
  verbs:
  - create
  - delete
  - patch
  - update
- apiGroups:
  - robot.harbor.crossplane.io
  resources:
  - robots
  verbs:
  - create
  - get
  - list
  - patch
  - update
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: harbor-devops
  namespace: harbor-prod
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: harbor-devops
subjects:
- kind: Group
  name: devops@mycompany.com
  apiGroup: rbac.authorization.k8s.io
```

## Audit and Compliance

### Event Auditing

```bash
# Audit all Harbor resource changes via Kubernetes API
kubectl get events -n harbor-prod -w

# Audit in YAML format
kubectl get events -n harbor-prod -o yaml | grep -E "involved|message|reason"
```

### Change Validation

```yaml
# Use admission webhooks to validate Harbor config changes
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: harbor-validation
webhooks:
- name: harbor.validation.example.com
  clientConfig:
    service:
      name: harbor-validator
      namespace: crossplane-system
      path: "/validate"
    caBundle: ...
  rules:
  - operations:
    - CREATE
    - UPDATE
    apiGroups:
    - "*.harbor.crossplane.io"
    resources:
    - projects
    - robots
    - webhooks
  admissionReviewVersions:
  - v1
  sideEffects: None
```

## Summary

✅ **Production Ready Checklist**

- [ ] All projects have `autoScanImages: true`
- [ ] All projects have `preventVulnerableImages: true`
- [ ] All robot accounts have `expiresIn` set (max 90 days)
- [ ] Webhooks use HTTPS with TLS verification enabled
- [ ] ProviderConfig credentials stored in encrypted Secret
- [ ] Namespace isolation for multi-Harbor deployments
- [ ] RBAC rules restrict resource management
- [ ] Monitoring and alerting configured
- [ ] Deletion policies explicitly set
- [ ] Audit logging enabled for compliance
