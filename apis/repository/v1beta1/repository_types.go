/*
Copyright 2024 Crossplane Harbor Provider.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

// RepositoryParameters defines the desired state of a Repository
type RepositoryParameters struct {
	// ProjectID is the numeric Harbor project id this repository belongs to
	// (a project name is also accepted for backward compat).
	// +kubebuilder:validation:Required
	ProjectID string `json:"projectId"`

	// Name is the name of the repository (without the project prefix)
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Description of the repository
	// +kubebuilder:validation:Optional
	Description *string `json:"description,omitempty"`
}

// RepositoryObservation defines the observed state of a Repository
type RepositoryObservation struct {
	// ID is the unique identifier of the repository in Harbor
	ID *string `json:"id,omitempty"`

	// FullName is the fully qualified repository name (project/name)
	FullName *string `json:"fullName,omitempty"`

	// ProjectID is the ID of the parent project
	ProjectID *string `json:"projectId,omitempty"`

	// ArtifactCount is the number of artifacts in this repository
	ArtifactCount *int64 `json:"artifactCount,omitempty"`

	// CreationTime is when the repository was created
	CreationTime *metav1.Time `json:"creationTime,omitempty"`

	// UpdateTime is when the repository was last updated
	UpdateTime *metav1.Time `json:"updateTime,omitempty"`

	// Description of the repository
	Description *string `json:"description,omitempty"`
}

// A RepositorySpec defines the desired state of a Repository.
type RepositorySpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              RepositoryParameters `json:"forProvider"`
}

// A RepositoryStatus represents the observed state of a Repository.
type RepositoryStatus struct {
	xpv1.ConditionedStatus `json:",inline"`
	AtProvider             RepositoryObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="REPOSITORY",type="string",JSONPath=".spec.forProvider.name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,harbor}

// A Repository is a managed resource that represents a Harbor repository.
type Repository struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RepositorySpec   `json:"spec"`
	Status RepositoryStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RepositoryList contains a list of Repository.
type RepositoryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Repository `json:"items"`
}

// GetCondition of this Repository.
func (mg *Repository) GetCondition(ct xpv1.ConditionType) xpv1.Condition {
	return mg.Status.GetCondition(ct)
}

// GetManagementPolicies of this Repository.
func (mg *Repository) GetManagementPolicies() xpv1.ManagementPolicies {
	return mg.Spec.ManagementPolicies
}

// GetProviderConfigReference of this Repository.
func (mg *Repository) GetProviderConfigReference() *xpv1.ProviderConfigReference {
	return mg.Spec.ProviderConfigReference
}

// GetWriteConnectionSecretToReference of this Repository.
func (mg *Repository) GetWriteConnectionSecretToReference() *xpv1.LocalSecretReference {
	return mg.Spec.WriteConnectionSecretToReference
}

// SetConditions of this Repository.
func (mg *Repository) SetConditions(c ...xpv1.Condition) {
	mg.Status.SetConditions(c...)
}

// SetManagementPolicies of this Repository.
func (mg *Repository) SetManagementPolicies(r xpv1.ManagementPolicies) {
	mg.Spec.ManagementPolicies = r
}

// SetProviderConfigReference of this Repository.
func (mg *Repository) SetProviderConfigReference(r *xpv1.ProviderConfigReference) {
	mg.Spec.ProviderConfigReference = r
}

// SetWriteConnectionSecretToReference of this Repository.
func (mg *Repository) SetWriteConnectionSecretToReference(r *xpv1.LocalSecretReference) {
	mg.Spec.WriteConnectionSecretToReference = r
}
