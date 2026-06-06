# Phase 2 Critical Features Implementation Guide

## Overview

This guide provides step-by-step instructions for implementing the 4 critical Phase 2 features:
1. **Repository Management** (4 methods)
2. **Artifact Management** (7 methods)  
3. **Member Management** (5 methods)
4. **Scan Management** (4 methods)

## Current Status

✅ **Completed:**
- Repository API types (`apis/repository/v1beta1/*`)
- Repository client methods in `internal/clients/harbor.go`
- Repository controller skeleton
- Comprehensive API gaps analysis
- Logging improvements across codebase

❌ **Needs Completion:**
- Code generation for Repository types
- Controller implementation fixes
- Integration in main.go
- Artifact types and client implementation
- Member types and client implementation
- Scan types and client implementation

## Step 1: Generate Repository Code

The Repository types need code generation to implement Crossplane interfaces.

### A. Run Code Generation

```bash
cd apis
go generate ./...
cd ..
```

This will:
- Generate `zz_generated.deepcopy.go` files
- Create CRD manifests in `package/crds/`
- Add necessary method implementations

### B. Verify Compilation

```bash
go build ./...
go test ./...
```

## Step 2: Complete Repository Controller

Once code generation is complete, the controller needs minor fixes:

1. **Time conversion** (line 99-101): Convert `time.Time` to `metav1.Time`
   ```go
   // Current (wrong):
   cr.Status.AtProvider.CreationTime = &t  // t is *time.Time
   
   // Should be:
   mt := metav1.NewTime(status.CreationTime)
   cr.Status.AtProvider.CreationTime = &mt
   ```

2. **Add Disconnect method**:
   ```go
   func (c *external) Disconnect(ctx context.Context) error {
       return c.service.Close()
   }
   ```

## Step 3: Register Repository in Main

Update `cmd/provider/main.go` to wire up Repository controller:

```go
// Add import
repositorycontroller "github.com/rossigee/provider-harbor/internal/controller/repository"

// In main() after other controller setup:
kingpin.FatalIfError(repositorycontroller.Setup(mgr, o), "Cannot setup Repository controller")

// Add to API scheme
kingpin.FatalIfError(repositoryv1beta1.AddToScheme(mgr.GetScheme()), "Cannot add Repository APIs to scheme")
```

## Step 4: Implement Artifact Resource

Create `apis/artifact/v1beta1/` following the Repository pattern:

### A. Create artifact_types.go

```go
package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

// ArtifactParameters defines the desired state of an Artifact
type ArtifactParameters struct {
	// ProjectID is the ID or name of the project
	// +kubebuilder:validation:Required
	ProjectID string `json:"projectId"`

	// RepositoryName is the name of the repository
	// +kubebuilder:validation:Required
	RepositoryName string `json:"repositoryName"`

	// Reference is the image reference (tag or digest)
	// +kubebuilder:validation:Required
	Reference string `json:"reference"`

	// Type is the artifact type (image, chart, etc.)
	// +kubebuilder:validation:Optional
	Type *string `json:"type,omitempty"`
}

// ArtifactObservation defines the observed state
type ArtifactObservation struct {
	ID              *string       `json:"id,omitempty"`
	Digest          *string       `json:"digest,omitempty"`
	Size            *int64        `json:"size,omitempty"`
	PullCount       *int64        `json:"pullCount,omitempty"`
	CreationTime    *metav1.Time  `json:"creationTime,omitempty"`
	UpdateTime      *metav1.Time  `json:"updateTime,omitempty"`
	VulnerabilityCount *int64     `json:"vulnerabilityCount,omitempty"`
}

// A ArtifactSpec defines the desired state of an Artifact
type ArtifactSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              ArtifactParameters `json:"forProvider"`
}

// A ArtifactStatus represents the observed state
type ArtifactStatus struct {
	xpv1.ManagedResourceStatus `json:",inline"`
	AtProvider                 ArtifactObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,harbor}

// An Artifact is a managed resource that represents a Harbor artifact.
type Artifact struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ArtifactSpec   `json:"spec"`
	Status ArtifactStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ArtifactList contains a list of Artifacts.
type ArtifactList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Artifact `json:"items"`
}
```

### B. Create groupversion_info.go and register.go

Copy from Repository pattern, changing:
- Group to `artifact.harbor.m.crossplane.io`
- Kind names to `Artifact`

### C. Add Client Methods

In `internal/clients/harbor.go`, add:

```go
// ArtifactSpec and ArtifactStatus types
type ArtifactSpec struct {
	ProjectID      string
	RepositoryName string
	Reference      string
	Type          *string
}

type ArtifactStatus struct {
	ID                 string
	Digest             string
	Size               int64
	PullCount          int64
	CreationTime       time.Time
	UpdateTime         time.Time
	VulnerabilityCount int64
}

// Implementation methods
func (c *HarborClient) ListArtifacts(ctx context.Context, projectID, repoName string) ([]*ArtifactStatus, error)
func (c *HarborClient) GetArtifact(ctx context.Context, projectID, repoName, reference string) (*ArtifactStatus, error)
func (c *HarborClient) DeleteArtifact(ctx context.Context, projectID, repoName, reference string) error
func (c *HarborClient) GetArtifactVulnerabilities(ctx context.Context, projectID, repoName, reference string) (*ArtifactStatus, error)
```

## Step 5: Implement Member Resource

Follow the same pattern as Artifact:

### A. Member Types
```go
type MemberParameters struct {
	ProjectID string  // Required
	Username  string  // Required
	Role      string  // Required: developer, master, maintainer, guest
}

type MemberObservation struct {
	ID           *string      `json:"id,omitempty"`
	MemberName   *string      `json:"memberName,omitempty"`
	MemberType   *string      `json:"memberType,omitempty"`
	Role         *string      `json:"role,omitempty"`
	CreationTime *metav1.Time `json:"creationTime,omitempty"`
}
```

### B. Member Client Methods
```go
func (c *HarborClient) AddProjectMember(ctx context.Context, projectID, username, role string) error
func (c *HarborClient) ListProjectMembers(ctx context.Context, projectID string) ([]*MemberStatus, error)
func (c *HarborClient) GetProjectMember(ctx context.Context, projectID, username string) (*MemberStatus, error)
func (c *HarborClient) UpdateProjectMember(ctx context.Context, projectID, username, role string) error
func (c *HarborClient) DeleteProjectMember(ctx context.Context, projectID, username string) error
```

## Step 6: Implement Scan Resource

Follow the same pattern:

### A. Scan Types
```go
type ScanParameters struct {
	ProjectID      string // Required
	RepositoryName string // Required
	Reference      string // Required
}

type ScanObservation struct {
	ID             *string      `json:"id,omitempty"`
	Status         *string      `json:"status,omitempty"` // pending, scanning, completed, failed
	CriticalCount  *int64       `json:"criticalCount,omitempty"`
	HighCount      *int64       `json:"highCount,omitempty"`
	MediumCount    *int64       `json:"mediumCount,omitempty"`
	LowCount       *int64       `json:"lowCount,omitempty"`
	StartTime      *metav1.Time `json:"startTime,omitempty"`
	EndTime        *metav1.Time `json:"endTime,omitempty"`
}
```

### B. Scan Client Methods
```go
func (c *HarborClient) TriggerScan(ctx context.Context, projectID, repoName, reference string) error
func (c *HarborClient) ListScans(ctx context.Context, projectID, repoName string) ([]*ScanStatus, error)
func (c *HarborClient) GetScan(ctx context.Context, projectID, repoName, reference string) (*ScanStatus, error)
func (c *HarborClient) StopScan(ctx context.Context, projectID, repoName, reference string) error
```

## Step 7: Update Project Resource

Enhance existing Project resource to use all CRD parameters:

### A. Update Client Methods

```go
// Current Project client methods only use Name and Public
// Need to extend to:
// - EnableContentTrust
// - EnableContentTrustCosign
// - AutoScanImages
// - PreventVulnerableImages
// - Severity
// - CVEAllowlist
// - RegistryID
// - StorageLimit
// - Metadata
```

### B. Example Enhancement

```go
func (c *HarborClient) CreateProject(ctx context.Context, spec *ProjectSpec) (*ProjectStatus, error) {
	// Current implementation only uses Name and Public
	// Add code to handle remaining parameters:
	
	if spec.EnableContentTrust {
		// Enable content trust
	}
	
	if spec.AutoScanImages {
		// Enable auto scanning
	}
	
	if spec.StorageLimit > 0 {
		// Set storage quota
	}
	
	// ... etc for all parameters
}
```

## Step 8: Integration Checklist

- [ ] Generate code for all new resources
- [ ] Add all controllers to main.go
- [ ] Register all APIs in API scheme
- [ ] Test each resource with example manifests
- [ ] Verify controllers reconcile correctly
- [ ] Run full test suite
- [ ] Update documentation
- [ ] Create example manifests in `examples/`

## File Structure Template

Each new resource should have:

```
apis/{resource}/v1beta1/
├── groupversion_info.go          # Group, Version, Scheme
├── register.go                   # Type metadata
├── {resource}_types.go           # CRD definitions  
├── zz_generated.deepcopy.go      # Auto-generated
└── zz_generated.managed*.go      # Auto-generated (after code gen)

internal/controller/{resource}/
├── {resource}_controller.go      # Controller implementation
└── (optional) {resource}_test.go # Tests
```

## Testing Each Resource

After implementation, test with:

```bash
# Verify it compiles
go build ./...

# Run linting
golangci-lint run ./...

# Create a test manifest
kubectl apply -f examples/{resource}_example.yaml

# Watch reconciliation
kubectl logs -f deployment/provider-harbor
```

## Common Patterns

### Time Conversion
```go
// From time.Time to metav1.Time
mt := metav1.NewTime(someTime)
cr.Status.AtProvider.CreationTime = &mt
```

### String Pointers
```go
// Safe handling of optional string fields
if spec.Description != nil {
	cr.Status.AtProvider.Description = spec.Description
}
```

### Error Handling
```go
// Always wrap errors with context
return errors.Wrap(err, "failed to create artifact")
```

## References

- **Crossplane Provider Development**: https://docs.crossplane.io/latest/concepts/providers/
- **Controller-Tools**: https://github.com/kubernetes-sigs/controller-tools
- **Harbor API**: https://github.com/goharbor/go-client
- **Existing Project Resource**: `apis/project/v1beta1/`
- **Existing Project Controller**: `internal/controller/project/`

## Estimated Effort

| Resource   | CRD | Controller | Client | Testing | Total |
|-----------|-----|-----------|--------|---------|-------|
| Repository | ✅  | 🔧        | ✅     | ⏳      | 2h    |
| Artifact   | 1h  | 1.5h      | 1h     | 1h      | 4.5h  |
| Member     | 1h  | 1.5h      | 1h     | 1h      | 4.5h  |
| Scan       | 1h  | 1.5h      | 1h     | 1h      | 4.5h  |
| Project    | 0.5h| 0h        | 2h     | 1h      | 3.5h  |
| **Total**  | -   | -         | -      | -       | **19h** |

## Next Steps

1. ✅ Start with Repository (foundation is ready)
2. Complete Repository tests
3. Implement Artifact (most critical for users)
4. Implement Member (essential for security)
5. Implement Scan (important for vulnerability management)
6. Enhance Project parameters
7. Full integration testing
8. Documentation update

## Support

For issues during implementation:
1. Check the existing Project resource pattern
2. Review Harbor Go client documentation
3. Refer to Crossplane provider development guide
4. Check controller-runtime examples
