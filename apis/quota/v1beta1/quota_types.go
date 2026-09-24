/*
Copyright 2025 Crossplane Harbor Provider.
*/

package v1beta1

import (
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// QuotaObservation defines the observed state of Harbor resource quotas
type QuotaObservation struct {
	// ID is the unique identifier of the quota
	// +kubebuilder:validation:Optional
	ID *int64 `json:"id,omitempty"`

	// ProjectID is the ID of the project this quota applies to
	// +kubebuilder:validation:Optional
	ProjectID *int64 `json:"projectID,omitempty"`

	// Type indicates the resource type (storage, artifact count, etc.)
	// +kubebuilder:validation:Optional
	Type *string `json:"type,omitempty"`

	// Hard is the hard limit for the quota
	// +kubebuilder:validation:Optional
	Hard *map[string]string `json:"hard,omitempty"`

	// Used represents current usage
	// +kubebuilder:validation:Optional
	Used *map[string]string `json:"used,omitempty"`

	// CreationTime is when the quota was created
	// +kubebuilder:validation:Optional
	CreationTime *metav1.Time `json:"creationTime,omitempty"`

	// UpdateTime is when the quota was last updated
	// +kubebuilder:validation:Optional
	UpdateTime *metav1.Time `json:"updateTime,omitempty"`
}

// QuotaParameters defines the desired state of Quota
type QuotaParameters struct {
	// ProjectID or ProjectRef references the project to fetch quotas for
	// +kubebuilder:validation:Required
	ProjectRef xpv1.Reference `json:"projectRef"`
}

// A QuotaSpec defines the desired state of a Quota resource.
type QuotaSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              QuotaParameters `json:"forProvider"`
}

// A QuotaStatus represents the observed state of a Quota.
type QuotaStatus struct {
	xpv1.ConditionedStatus `json:",inline"`
	AtProvider             QuotaObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="PROJECT-ID",type="string",JSONPath=".status.atProvider.projectID"
// +kubebuilder:printcolumn:name="TYPE",type="string",JSONPath=".status.atProvider.type"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,harbor}
type Quota struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   QuotaSpec   `json:"spec"`
	Status QuotaStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type QuotaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Quota `json:"items"`
}

// GetCondition of this Quota.
func (mg *Quota) GetCondition(ct xpv1.ConditionType) xpv1.Condition {
	return mg.Status.GetCondition(ct)
}

// GetManagementPolicies of this Quota.
func (mg *Quota) GetManagementPolicies() xpv1.ManagementPolicies {
	return mg.Spec.ManagementPolicies
}

// SetConditions of this Quota.
func (mg *Quota) SetConditions(c ...xpv1.Condition) {
	mg.Status.SetConditions(c...)
}

// SetManagementPolicies of this Quota.
func (mg *Quota) SetManagementPolicies(c xpv1.ManagementPolicies) {
	mg.Spec.ManagementPolicies = c
}
