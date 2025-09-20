/*
Copyright 2024 Crossplane Harbor Provider.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

// RegistryCredential represents registry authentication credentials
type RegistryCredential struct {
	// Type is the type of credential (basic, oauth, etc.)
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=basic;oauth
	Type *string `json:"type,omitempty"`

	// AccessKey is the access key for the registry
	// +kubebuilder:validation:Optional
	AccessKey *string `json:"accessKey,omitempty"`

	// AccessSecret contains the secret reference for registry access
	// +kubebuilder:validation:Optional
	AccessSecretRef *xpv1.SecretKeySelector `json:"accessSecretRef,omitempty"`
}

// RegistryParameters defines the desired state of a Registry
type RegistryParameters struct {
	// Name is the name of the registry
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Description is an optional description of the registry
	// +kubebuilder:validation:Optional
	Description *string `json:"description,omitempty"`

	// Type is the type of registry (harbor, docker-hub, docker-registry, etc.)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=harbor;docker-hub;docker-registry;helm-hub;aws-ecr;azure-acr;google-gcr;gitlab;quay
	Type string `json:"type"`

	// URL is the URL of the registry
	// +kubebuilder:validation:Required
	URL string `json:"url"`

	// Insecure indicates whether to skip TLS verification
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	Insecure *bool `json:"insecure,omitempty"`

	// Credential contains the authentication credentials for the registry
	// +kubebuilder:validation:Optional
	Credential *RegistryCredential `json:"credential,omitempty"`
}

// RegistryObservation defines the observed state of a Registry
type RegistryObservation struct {
	// ID is the unique identifier of the registry in Harbor
	ID *int64 `json:"id,omitempty"`

	// CreationTime is when the registry was created
	CreationTime *metav1.Time `json:"creationTime,omitempty"`

	// UpdateTime is when the registry was last updated
	UpdateTime *metav1.Time `json:"updateTime,omitempty"`

	// Status is the status of the registry
	Status *string `json:"status,omitempty"`
}

// A RegistrySpec defines the desired state of a Registry.
type RegistrySpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       RegistryParameters `json:"forProvider"`
}

// A RegistryStatus represents the observed state of a Registry.
type RegistryStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          RegistryObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="REGISTRY-ID",type="string",JSONPath=".status.atProvider.id"
// +kubebuilder:printcolumn:name="TYPE",type="string",JSONPath=".spec.forProvider.type"
// +kubebuilder:printcolumn:name="URL",type="string",JSONPath=".spec.forProvider.url"
// +kubebuilder:printcolumn:name="STATUS",type="string",JSONPath=".status.atProvider.status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,harbor}
type Registry struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RegistrySpec   `json:"spec"`
	Status RegistryStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type RegistryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Registry `json:"items"`
}

// GetCondition of this Registry.
func (mg *Registry) GetCondition(ct xpv1.ConditionType) xpv1.Condition {
	return mg.Status.GetCondition(ct)
}

// GetDeletionPolicy of this Registry.
func (mg *Registry) GetDeletionPolicy() xpv1.DeletionPolicy {
	return mg.Spec.DeletionPolicy
}

// GetManagementPolicies of this Registry.
func (mg *Registry) GetManagementPolicies() xpv1.ManagementPolicies {
	return mg.Spec.ManagementPolicies
}

// GetProviderConfigReference of this Registry.
func (mg *Registry) GetProviderConfigReference() *xpv1.Reference {
	return mg.Spec.ProviderConfigReference
}

// GetPublishConnectionDetailsTo of this Registry.
func (mg *Registry) GetPublishConnectionDetailsTo() *xpv1.PublishConnectionDetailsTo {
	return mg.Spec.PublishConnectionDetailsTo
}

// GetWriteConnectionSecretToReference of this Registry.
func (mg *Registry) GetWriteConnectionSecretToReference() *xpv1.SecretReference {
	return mg.Spec.WriteConnectionSecretToReference
}

// SetConditions of this Registry.
func (mg *Registry) SetConditions(c ...xpv1.Condition) {
	mg.Status.SetConditions(c...)
}

// SetDeletionPolicy of this Registry.
func (mg *Registry) SetDeletionPolicy(r xpv1.DeletionPolicy) {
	mg.Spec.DeletionPolicy = r
}

// SetManagementPolicies of this Registry.
func (mg *Registry) SetManagementPolicies(r xpv1.ManagementPolicies) {
	mg.Spec.ManagementPolicies = r
}

// SetProviderConfigReference of this Registry.
func (mg *Registry) SetProviderConfigReference(r *xpv1.Reference) {
	mg.Spec.ProviderConfigReference = r
}

// SetPublishConnectionDetailsTo of this Registry.
func (mg *Registry) SetPublishConnectionDetailsTo(r *xpv1.PublishConnectionDetailsTo) {
	mg.Spec.PublishConnectionDetailsTo = r
}

// SetWriteConnectionSecretToReference of this Registry.
func (mg *Registry) SetWriteConnectionSecretToReference(r *xpv1.SecretReference) {
	mg.Spec.WriteConnectionSecretToReference = r
}