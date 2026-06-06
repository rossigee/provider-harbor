# Phase 2 Completion Guide - Ready to Implement

## Current Status

✅ **Completed:**
- Repository resource scaffold (types, client methods, controller)
- Implementation guide with templates
- Code generation infrastructure
- Phase 2 implementation script
- All supporting documentation

⏳ **Needs Completion:**
- Code generation for Repository (GetCondition implementation)
- Artifact resource (4 files)
- Member resource (4 files)
- Scan resource (4 files)
- Wire-up in main.go
- Testing

---

## Quick Start: Copy-Paste Implementation

### 1. Create Artifact Resource

Copy these files exactly as shown:

#### File: `apis/artifact/v1beta1/artifact_types.go`

```go
/*
Copyright 2024 Crossplane Harbor Provider.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

// ArtifactParameters defines the desired state
type ArtifactParameters struct {
	ProjectID      string `json:"projectId"`
	RepositoryName string `json:"repositoryName"`
	Reference      string `json:"reference"`
	Type           *string `json:"type,omitempty"`
}

// ArtifactObservation defines the observed state
type ArtifactObservation struct {
	ID                 *string      `json:"id,omitempty"`
	Digest             *string      `json:"digest,omitempty"`
	Size               *int64       `json:"size,omitempty"`
	PullCount          *int64       `json:"pullCount,omitempty"`
	CreationTime       *metav1.Time `json:"creationTime,omitempty"`
	UpdateTime         *metav1.Time `json:"updateTime,omitempty"`
	VulnerabilityCount *int64       `json:"vulnerabilityCount,omitempty"`
}

// A ArtifactSpec defines the desired state of an Artifact.
type ArtifactSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              ArtifactParameters `json:"forProvider"`
}

// A ArtifactStatus represents the observed state
type ArtifactStatus struct {
	xpv1.ConditionedStatus `json:",inline"`
	AtProvider             ArtifactObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="DIGEST",type="string",JSONPath=".status.atProvider.digest"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,harbor}

// An Artifact is a managed resource that represents a Harbor artifact.
type Artifact struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ArtifactSpec   `json:"spec"`
	Status            ArtifactStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ArtifactList contains a list of Artifacts.
type ArtifactList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Artifact `json:"items"`
}
```

#### File: `apis/artifact/v1beta1/register.go`

```go
/*
Copyright 2024 Crossplane Harbor Provider.
*/

package v1beta1

import (
	"reflect"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Artifact type metadata.
var (
	ArtifactKind             = reflect.TypeOf(Artifact{}).Name()
	ArtifactGroupKind        = schema.GroupKind{Group: Group, Kind: ArtifactKind}
	ArtifactKindAPIVersion   = ArtifactKind + "." + SchemeGroupVersion.String()
	ArtifactGroupVersionKind = SchemeGroupVersion.WithKind(ArtifactKind)
)

func init() {
	SchemeBuilder.Register(&Artifact{}, &ArtifactList{})
}
```

#### File: `apis/artifact/v1beta1/zz_generated.deepcopy.go`

Generate using: `go generate ./apis/artifact/v1beta1/...`

---

### 2. Create Member Resource  

Follow identical pattern:

#### File: `apis/member/v1beta1/member_types.go`

```go
/*
Copyright 2024 Crossplane Harbor Provider.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

// MemberParameters defines the desired state
type MemberParameters struct {
	ProjectID string `json:"projectId"`
	Username  string `json:"username"`
	Role      string `json:"role"` // developer, maintainer, master, guest
}

// MemberObservation defines the observed state
type MemberObservation struct {
	ID           *string      `json:"id,omitempty"`
	MemberName   *string      `json:"memberName,omitempty"`
	MemberType   *string      `json:"memberType,omitempty"`
	Role         *string      `json:"role,omitempty"`
	CreationTime *metav1.Time `json:"creationTime,omitempty"`
}

// A MemberSpec defines the desired state of a Member.
type MemberSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              MemberParameters `json:"forProvider"`
}

// A MemberStatus represents the observed state
type MemberStatus struct {
	xpv1.ConditionedStatus `json:",inline"`
	AtProvider             MemberObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="USERNAME",type="string",JSONPath=".spec.forProvider.username"
// +kubebuilder:printcolumn:name="ROLE",type="string",JSONPath=".spec.forProvider.role"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,harbor}

// A Member is a managed resource that represents a Harbor project member.
type Member struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              MemberSpec   `json:"spec"`
	Status            MemberStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MemberList contains a list of Members.
type MemberList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Member `json:"items"`
}
```

(Continue with register.go, groupversion_info.go following Artifact pattern)

---

### 3. Create Scan Resource

Follow identical pattern with:

```go
type ScanParameters struct {
	ProjectID      string `json:"projectId"`
	RepositoryName string `json:"repositoryName"`
	Reference      string `json:"reference"`
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

---

## 4. Wire Up in main.go

Add to `cmd/provider/main.go`:

```go
import (
	// ... existing imports
	artifactcontroller "github.com/rossigee/provider-harbor/internal/controller/artifact"
	membercontroller "github.com/rossigee/provider-harbor/internal/controller/member"
	scancontroller "github.com/rossigee/provider-harbor/internal/controller/scan"
	
	artifactv1beta1 "github.com/rossigee/provider-harbor/apis/artifact/v1beta1"
	memberv1beta1 "github.com/rossigee/provider-harbor/apis/member/v1beta1"
	scanv1beta1 "github.com/rossigee/provider-harbor/apis/scan/v1beta1"
	repositoryv1beta1 "github.com/rossigee/provider-harbor/apis/repository/v1beta1"
)

func main() {
	// ... existing code ...
	
	// Add APIs to scheme
	kingpin.FatalIfError(repositoryv1beta1.AddToScheme(mgr.GetScheme()), "Cannot add Repository APIs to scheme")
	kingpin.FatalIfError(artifactv1beta1.AddToScheme(mgr.GetScheme()), "Cannot add Artifact APIs to scheme")
	kingpin.FatalIfError(memberv1beta1.AddToScheme(mgr.GetScheme()), "Cannot add Member APIs to scheme")
	kingpin.FatalIfError(scanv1beta1.AddToScheme(mgr.GetScheme()), "Cannot add Scan APIs to scheme")
	
	// Setup controllers
	kingpin.FatalIfError(repositorycontroller.Setup(mgr, o), "Cannot setup Repository controller")
	kingpin.FatalIfError(artifactcontroller.Setup(mgr, o), "Cannot setup Artifact controller")
	kingpin.FatalIfError(membercontroller.Setup(mgr, o), "Cannot setup Member controller")
	kingpin.FatalIfError(scancontroller.Setup(mgr, o), "Cannot setup Scan controller")
	
	// ... rest of main ...
}
```

---

## 5. Create Controllers

For each resource, create `internal/controller/{resource}/{resource}_controller.go` using the Repository controller pattern (already created).

Copy `internal/controller/repository/repository_controller.go` and replace:
- All instances of "repository" → "{resource}"
- All instances of "Repository" → "{Resource}"  
- Field names in parameters (ProjectID, Name, etc.)

---

## 6. Add Client Methods

In `internal/clients/harbor.go`, add methods for each resource. Example for Artifact:

```go
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

func (c *HarborClient) ListArtifacts(ctx context.Context, projectID, repoName string) ([]*ArtifactStatus, error) {
	c.logger.Info("Listing Harbor artifacts", "projectId", projectID, "repo", repoName)
	// Implementation here
	return []*ArtifactStatus{}, nil
}

// ... GetArtifact, DeleteArtifact, GetArtifactVulnerabilities
```

---

## 7. Code Generation

Once all files are in place:

```bash
# Generate deepcopy methods
go generate ./apis/...

# Generate managed methodsets  
go run -tags generate github.com/crossplane/crossplane-tools/cmd/angryjet generate-methodsets \
    --header-file=hack/boilerplate.go.txt \
    ./apis/.../{artifact,member,scan}/v1beta1/...

# Verify compilation
go build ./...

# Run linting
golangci-lint run ./...
```

---

## 8. Testing

Create example manifests in `examples/`:

```yaml
# examples/artifact.yaml
apiVersion: artifact.harbor.m.crossplane.io/v1beta1
kind: Artifact
metadata:
  name: my-artifact
spec:
  forProvider:
    projectId: "1"
    repositoryName: my-repo
    reference: "v1.0.0"
  providerConfigRef:
    name: default
  deletionPolicy: Delete
```

Test:
```bash
kubectl apply -f examples/artifact.yaml
kubectl get artifacts
kubectl describe artifact my-artifact
```

---

## Time Estimates (with templates provided)

- Artifact resource: 30 minutes (copy-paste + tweaks)
- Member resource: 30 minutes (copy-paste + tweaks)
- Scan resource: 30 minutes (copy-paste + tweaks)
- Wire-up in main.go: 15 minutes
- Code generation & testing: 30 minutes
- **Total: ~2.5 hours** (down from 19 hours with templates)

---

## Troubleshooting

### "does not implement resource.Managed"
→ Run: `go generate ./apis/...` and ensure ConditionedStatus is used (not ManagedResourceStatus)

### "undefined: GetCondition"
→ Code generation incomplete. Verify zz_generated.deepcopy.go exists and imports runtime

### Controllers don't register
→ Add to main.go APIs before controllers and verify import paths

### Tests fail
→ All tests are marked `[no test files]` - create simple integration tests if needed

---

## File Checklist

- [x] Repository resource (complete scaffold)
- [ ] Artifact types file
- [ ] Artifact register + groupversion
- [ ] Artifact deepcopy (auto-generated)
- [ ] Artifact controller
- [ ] Member types file
- [ ] Member register + groupversion  
- [ ] Member deepcopy (auto-generated)
- [ ] Member controller
- [ ] Scan types file
- [ ] Scan register + groupversion
- [ ] Scan deepcopy (auto-generated)
- [ ] Scan controller
- [ ] Update main.go (wire-up)
- [ ] Add client methods for all resources
- [ ] Code generation
- [ ] Test manifests in examples/
- [ ] Documentation

---

## After Phase 2 Completion

🎉 **Provider will be production-ready with:**
- ✅ Full project CRUD + all parameters
- ✅ Repository browsing and management
- ✅ Artifact (image/chart) management
- ✅ Member access control
- ✅ Vulnerability scanning
- ✅ User/Registry/Scanner management
- ✅ Comprehensive logging
- ✅ Proper error handling
- ✅ Resource lifecycle management

Ready for: Multi-tenancy, security enforcement, GitOps, CI/CD integration

---

## Support Resources

- See `IMPLEMENTATION_GUIDE.md` for detailed instructions
- See `API_GAPS_ANALYSIS.md` for API coverage details
- Reference `internal/controller/project/` for controller pattern
- Reference `internal/clients/harbor.go` for client method pattern
- Check `apis/project/v1beta1/` for CRD structure
