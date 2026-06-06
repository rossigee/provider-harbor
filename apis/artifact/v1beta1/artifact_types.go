/*
Copyright 2024 Crossplane Harbor Provider.
*/

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

// ArtifactObservation defines the observed state of an Artifact
type ArtifactObservation struct {
	// ID is the unique identifier of the artifact in Harbor
	ID *string `json:"id,omitempty"`

	// Digest is the content digest of the artifact
	Digest *string `json:"digest,omitempty"`

	// Size is the size of the artifact in bytes
	Size *int64 `json:"size,omitempty"`

	// PullCount is the number of times this artifact has been pulled
	PullCount *int64 `json:"pullCount,omitempty"`

	// CreationTime is when the artifact was created
	CreationTime *metav1.Time `json:"creationTime,omitempty"`

	// UpdateTime is when the artifact was last updated
	UpdateTime *metav1.Time `json:"updateTime,omitempty"`

	// VulnerabilityCount is the number of vulnerabilities found
	VulnerabilityCount *int64 `json:"vulnerabilityCount,omitempty"`
}

// A ArtifactSpec defines the desired state of an Artifact.
type ArtifactSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              ArtifactParameters `json:"forProvider"`
}

// A ArtifactStatus represents the observed state of an Artifact.
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
// +kubebuilder:printcolumn:name="SIZE",type="integer",JSONPath=".status.atProvider.size"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,harbor}

// An Artifact is a managed resource that represents a Harbor artifact.
type Artifact struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ArtifactSpec   `json:"spec"`
	Status ArtifactStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ArtifactList contains a list of Artifact.
type ArtifactList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Artifact `json:"items"`
}

// GetCondition of this Artifact.
func (mg *Artifact) GetCondition(ct xpv1.ConditionType) xpv1.Condition {
	return mg.Status.GetCondition(ct)
}

// GetManagementPolicies of this Artifact.
func (mg *Artifact) GetManagementPolicies() xpv1.ManagementPolicies {
	return mg.Spec.ManagementPolicies
}

// GetProviderConfigReference of this Artifact.
func (mg *Artifact) GetProviderConfigReference() *xpv1.ProviderConfigReference {
	return mg.Spec.ProviderConfigReference
}

// GetWriteConnectionSecretToReference of this Artifact.
func (mg *Artifact) GetWriteConnectionSecretToReference() *xpv1.LocalSecretReference {
	return mg.Spec.WriteConnectionSecretToReference
}

// SetConditions of this Artifact.
func (mg *Artifact) SetConditions(c ...xpv1.Condition) {
	mg.Status.SetConditions(c...)
}

// SetManagementPolicies of this Artifact.
func (mg *Artifact) SetManagementPolicies(r xpv1.ManagementPolicies) {
	mg.Spec.ManagementPolicies = r
}

// SetProviderConfigReference of this Artifact.
func (mg *Artifact) SetProviderConfigReference(r *xpv1.ProviderConfigReference) {
	mg.Spec.ProviderConfigReference = r
}

// SetWriteConnectionSecretToReference of this Artifact.
func (mg *Artifact) SetWriteConnectionSecretToReference(r *xpv1.LocalSecretReference) {
	mg.Spec.WriteConnectionSecretToReference = r
}
