/*
Copyright 2024 Crossplane Harbor Provider.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

// RetentionRule defines a retention rule
type RetentionRule struct {
	// RuleType: always, latestPushedK, latestPulledN
	// +kubebuilder:validation:Enum=always;latestPushedK;latestPulledN;daysSinceLastPull;daysSinceLastPush
	RuleType string `json:"ruleType"`

	// TagSelectors define which tags to apply this rule to
	// +kubebuilder:validation:Optional
	TagSelectors []string `json:"tagSelectors,omitempty"`

	// Parameters are rule-specific parameters (e.g., {"k": "10"})
	// +kubebuilder:validation:Optional
	Parameters map[string]string `json:"parameters,omitempty"`
}

// RetentionParameters defines the desired state of a Retention policy
type RetentionParameters struct {
	// ProjectID is the ID of the project
	// +kubebuilder:validation:Required
	ProjectID string `json:"projectId"`

	// Description of the retention policy
	// +kubebuilder:validation:Optional
	Description *string `json:"description,omitempty"`

	// Rules define the cleanup rules
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Rules []RetentionRule `json:"rules"`

	// Trigger: manual, scheduled
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=manual;scheduled
	Trigger string `json:"trigger"`

	// Enabled controls if the policy is active
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	Enabled *bool `json:"enabled,omitempty"`
}

// RetentionObservation defines the observed state of a Retention policy
type RetentionObservation struct {
	// ID is the unique identifier of the retention policy
	ID *string `json:"id,omitempty"`

	// Enabled indicates if the policy is active
	Enabled *bool `json:"enabled,omitempty"`

	// CreationTime is when the policy was created
	CreationTime *metav1.Time `json:"creationTime,omitempty"`

	// UpdateTime is when the policy was last updated
	UpdateTime *metav1.Time `json:"updateTime,omitempty"`

	// LastExecutionTime of the retention cleanup
	LastExecutionTime *metav1.Time `json:"lastExecutionTime,omitempty"`
}

// A RetentionSpec defines the desired state of a Retention policy.
type RetentionSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              RetentionParameters `json:"forProvider"`
}

// A RetentionStatus represents the observed state of a Retention policy.
type RetentionStatus struct {
	xpv1.ConditionedStatus `json:",inline"`
	AtProvider             RetentionObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="PROJECT",type="string",JSONPath=".spec.forProvider.projectId"
// +kubebuilder:printcolumn:name="RULES",type="integer",JSONPath=".spec.forProvider.rules | length"
// +kubebuilder:printcolumn:name="TRIGGER",type="string",JSONPath=".spec.forProvider.trigger"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,harbor}

// A Retention is a managed resource that represents a Harbor retention policy for automatic image cleanup.
type Retention struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RetentionSpec   `json:"spec"`
	Status RetentionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RetentionList contains a list of Retention.
type RetentionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Retention `json:"items"`
}

// GetCondition of this Retention.
func (mg *Retention) GetCondition(ct xpv1.ConditionType) xpv1.Condition {
	return mg.Status.GetCondition(ct)
}

// GetManagementPolicies of this Retention.
func (mg *Retention) GetManagementPolicies() xpv1.ManagementPolicies {
	return mg.Spec.ManagementPolicies
}

// GetProviderConfigReference of this Retention.
func (mg *Retention) GetProviderConfigReference() *xpv1.ProviderConfigReference {
	return mg.Spec.ProviderConfigReference
}

// GetWriteConnectionSecretToReference of this Retention.
func (mg *Retention) GetWriteConnectionSecretToReference() *xpv1.LocalSecretReference {
	return mg.Spec.WriteConnectionSecretToReference
}

// SetConditions of this Retention.
func (mg *Retention) SetConditions(c ...xpv1.Condition) {
	mg.Status.SetConditions(c...)
}

// SetManagementPolicies of this Retention.
func (mg *Retention) SetManagementPolicies(r xpv1.ManagementPolicies) {
	mg.Spec.ManagementPolicies = r
}

// SetProviderConfigReference of this Retention.
func (mg *Retention) SetProviderConfigReference(r *xpv1.ProviderConfigReference) {
	mg.Spec.ProviderConfigReference = r
}

// SetWriteConnectionSecretToReference of this Retention.
func (mg *Retention) SetWriteConnectionSecretToReference(r *xpv1.LocalSecretReference) {
	mg.Spec.WriteConnectionSecretToReference = r
}
