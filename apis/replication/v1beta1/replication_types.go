/*
Copyright 2024 Crossplane Harbor Provider.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

// ReplicationFilter defines filter rules for replication
type ReplicationFilter struct {
	// Type is the filter type: repository, tag, label, resource
	// +kubebuilder:validation:Enum=repository;tag;label;resource
	Type string `json:"type"`

	// Value is the filter value
	// +kubebuilder:validation:Required
	Value string `json:"value"`
}

// ReplicationDestination defines the destination registry
type ReplicationDestination struct {
	// Name is the destination registry name
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Namespace is the namespace in destination registry
	// +kubebuilder:validation:Optional
	Namespace string `json:"namespace,omitempty"`

	// URL is the destination registry URL
	// +kubebuilder:validation:Optional
	URL string `json:"url,omitempty"`
}

// ReplicationParameters defines the desired state of a Replication policy
type ReplicationParameters struct {
	// Name is the name of the replication policy
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Description of the replication policy
	// +kubebuilder:validation:Optional
	Description *string `json:"description,omitempty"`

	// SourceRegistry is the source registry name (optional for local registry)
	// +kubebuilder:validation:Optional
	SourceRegistry *string `json:"sourceRegistry,omitempty"`

	// DestinationReg is the destination registry configuration
	// +kubebuilder:validation:Required
	DestinationReg ReplicationDestination `json:"destinationReg"`

	// Filters define which repositories/tags to replicate
	// +kubebuilder:validation:Required
	Filters []ReplicationFilter `json:"filters"`

	// Trigger is the replication trigger: manual, scheduled, event_based
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=manual;scheduled;event_based
	Trigger string `json:"trigger"`

	// DeleteSourceTag removes source image tags after replication
	// +kubebuilder:validation:Optional
	DeleteSourceTag *bool `json:"deleteSourceTag,omitempty"`

	// Override overwrites images in destination
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	Override *bool `json:"override,omitempty"`

	// Enabled controls if the policy is active
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	Enabled *bool `json:"enabled,omitempty"`
}

// ReplicationObservation defines the observed state of a Replication policy
type ReplicationObservation struct {
	// ID is the unique identifier of the replication policy
	ID *string `json:"id,omitempty"`

	// Enabled indicates if the policy is currently active
	Enabled *bool `json:"enabled,omitempty"`

	// CreationTime is when the policy was created
	CreationTime *metav1.Time `json:"creationTime,omitempty"`

	// UpdateTime is when the policy was last updated
	UpdateTime *metav1.Time `json:"updateTime,omitempty"`

	// LastExecutionStatus is the status of the last execution
	LastExecutionStatus *string `json:"lastExecutionStatus,omitempty"`
}

// A ReplicationSpec defines the desired state of a Replication policy.
type ReplicationSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              ReplicationParameters `json:"forProvider"`
}

// A ReplicationStatus represents the observed state of a Replication policy.
type ReplicationStatus struct {
	xpv1.ConditionedStatus `json:",inline"`
	AtProvider             ReplicationObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="POLICY",type="string",JSONPath=".spec.forProvider.name"
// +kubebuilder:printcolumn:name="DESTINATION",type="string",JSONPath=".spec.forProvider.destinationReg.name"
// +kubebuilder:printcolumn:name="TRIGGER",type="string",JSONPath=".spec.forProvider.trigger"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,harbor}

// A Replication is a managed resource that represents a Harbor replication policy for cross-registry synchronization.
type Replication struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ReplicationSpec   `json:"spec"`
	Status ReplicationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ReplicationList contains a list of Replication.
type ReplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Replication `json:"items"`
}

// GetCondition of this Replication.
func (mg *Replication) GetCondition(ct xpv1.ConditionType) xpv1.Condition {
	return mg.Status.GetCondition(ct)
}

// GetManagementPolicies of this Replication.
func (mg *Replication) GetManagementPolicies() xpv1.ManagementPolicies {
	return mg.Spec.ManagementPolicies
}

// GetProviderConfigReference of this Replication.
func (mg *Replication) GetProviderConfigReference() *xpv1.ProviderConfigReference {
	return mg.Spec.ProviderConfigReference
}

// GetWriteConnectionSecretToReference of this Replication.
func (mg *Replication) GetWriteConnectionSecretToReference() *xpv1.LocalSecretReference {
	return mg.Spec.WriteConnectionSecretToReference
}

// SetConditions of this Replication.
func (mg *Replication) SetConditions(c ...xpv1.Condition) {
	mg.Status.SetConditions(c...)
}

// SetManagementPolicies of this Replication.
func (mg *Replication) SetManagementPolicies(r xpv1.ManagementPolicies) {
	mg.Spec.ManagementPolicies = r
}

// SetProviderConfigReference of this Replication.
func (mg *Replication) SetProviderConfigReference(r *xpv1.ProviderConfigReference) {
	mg.Spec.ProviderConfigReference = r
}

// SetWriteConnectionSecretToReference of this Replication.
func (mg *Replication) SetWriteConnectionSecretToReference(r *xpv1.LocalSecretReference) {
	mg.Spec.WriteConnectionSecretToReference = r
}
