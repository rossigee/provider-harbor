/*
Copyright 2024 Crossplane Harbor Provider.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
)

// ProjectParameters defines the desired state of a Project
type ProjectParameters struct {
	// Name is the name of the project in Harbor
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Public indicates if the project is publicly accessible
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	Public *bool `json:"public,omitempty"`

	// EnableContentTrust enables Docker Content Trust for this project
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	EnableContentTrust *bool `json:"enableContentTrust,omitempty"`

	// EnableContentTrustCosign enables Cosign-based content trust
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	EnableContentTrustCosign *bool `json:"enableContentTrustCosign,omitempty"`

	// AutoScanImages automatically scans images for vulnerabilities
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	AutoScanImages *bool `json:"autoScanImages,omitempty"`

	// PreventVulnerableImages prevents vulnerable images from being pulled
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	PreventVulnerableImages *bool `json:"preventVulnerableImages,omitempty"`

	// Severity represents the severity level for vulnerability prevention
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=negligible;low;medium;high;critical
	Severity *string `json:"severity,omitempty"`

	// CVEAllowlist is a list of CVE IDs that are allowed even if they match the severity level
	// +kubebuilder:validation:Optional
	CVEAllowlist []string `json:"cveAllowlist,omitempty"`

	// RegistryID is the ID of the registry for proxy cache projects
	// +kubebuilder:validation:Optional
	RegistryID *int64 `json:"registryId,omitempty"`

	// StorageLimit is the storage quota for the project (in bytes)
	// +kubebuilder:validation:Optional
	StorageLimit *int64 `json:"storageLimit,omitempty"`

	// Metadata contains additional metadata for the project
	// +kubebuilder:validation:Optional
	Metadata map[string]string `json:"metadata,omitempty"`
}

// ProjectObservation defines the observed state of a Project
type ProjectObservation struct {
	// ID is the unique identifier of the project in Harbor
	ID *int64 `json:"id,omitempty"`

	// CreationTime is when the project was created
	CreationTime *metav1.Time `json:"creationTime,omitempty"`

	// UpdateTime is when the project was last updated
	UpdateTime *metav1.Time `json:"updateTime,omitempty"`

	// OwnerID is the ID of the project owner
	OwnerID *int64 `json:"ownerId,omitempty"`

	// OwnerName is the name of the project owner
	OwnerName *string `json:"ownerName,omitempty"`

	// RepoCount is the number of repositories in the project
	RepoCount *int64 `json:"repoCount,omitempty"`

	// ChartCount is the number of charts in the project
	ChartCount *int64 `json:"chartCount,omitempty"`

	// CurrentStorageUsage is the current storage usage in bytes
	CurrentStorageUsage *int64 `json:"currentStorageUsage,omitempty"`
}

// A ProjectSpec defines the desired state of a Project.
type ProjectSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       ProjectParameters `json:"forProvider"`
}

// A ProjectStatus represents the observed state of a Project.
type ProjectStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ProjectObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="PROJECT-ID",type="string",JSONPath=".status.atProvider.id"
// +kubebuilder:printcolumn:name="PUBLIC",type="boolean",JSONPath=".spec.forProvider.public"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,harbor}
type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectSpec   `json:"spec"`
	Status ProjectStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Project `json:"items"`
}

// GetCondition of this Project.
func (mg *Project) GetCondition(ct xpv1.ConditionType) xpv1.Condition {
	return mg.Status.GetCondition(ct)
}

// GetDeletionPolicy of this Project.
func (mg *Project) GetDeletionPolicy() xpv1.DeletionPolicy {
	return mg.Spec.DeletionPolicy
}

// GetManagementPolicies of this Project.
func (mg *Project) GetManagementPolicies() xpv1.ManagementPolicies {
	return mg.Spec.ManagementPolicies
}

// GetProviderConfigReference of this Project.
func (mg *Project) GetProviderConfigReference() *xpv1.Reference {
	return mg.Spec.ProviderConfigReference
}

// GetWriteConnectionSecretToReference of this Project.
func (mg *Project) GetWriteConnectionSecretToReference() *xpv1.SecretReference {
	return mg.Spec.WriteConnectionSecretToReference
}

// SetConditions of this Project.
func (mg *Project) SetConditions(c ...xpv1.Condition) {
	mg.Status.SetConditions(c...)
}

// SetDeletionPolicy of this Project.
func (mg *Project) SetDeletionPolicy(r xpv1.DeletionPolicy) {
	mg.Spec.DeletionPolicy = r
}

// SetManagementPolicies of this Project.
func (mg *Project) SetManagementPolicies(r xpv1.ManagementPolicies) {
	mg.Spec.ManagementPolicies = r
}

// SetProviderConfigReference of this Project.
func (mg *Project) SetProviderConfigReference(r *xpv1.Reference) {
	mg.Spec.ProviderConfigReference = r
}

}

// SetWriteConnectionSecretToReference of this Project.
func (mg *Project) SetWriteConnectionSecretToReference(r *xpv1.SecretReference) {
	mg.Spec.WriteConnectionSecretToReference = r
}