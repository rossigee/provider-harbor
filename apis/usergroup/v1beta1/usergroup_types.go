/*
Copyright 2024 Crossplane Harbor Provider.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

// UserGroupParameters defines the desired state of a UserGroup
type UserGroupParameters struct {
	// GroupName is the name of the user group
	// +kubebuilder:validation:Required
	GroupName string `json:"groupName"`

	// GroupType is the group type: 1 for LDAP, 2 for HTTP, 3 for OIDC
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=1;2;3
	GroupType int64 `json:"groupType"`

	// LdapGroupDn is the DN of the LDAP group if group type is 1 (LDAP group)
	// +kubebuilder:validation:Optional
	LdapGroupDn *string `json:"ldapGroupDn,omitempty"`
}

// UserGroupObservation defines the observed state of a UserGroup
type UserGroupObservation struct {
	// ID is the unique identifier of the user group in Harbor
	ID *int64 `json:"id,omitempty"`
}

// A UserGroupSpec defines the desired state of a UserGroup.
type UserGroupSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              UserGroupParameters `json:"forProvider"`
}

// A UserGroupStatus represents the observed state of a UserGroup.
type UserGroupStatus struct {
	xpv1.ConditionedStatus `json:",inline"`
	AtProvider             UserGroupObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="GROUP-ID",type="string",JSONPath=".status.atProvider.id"
// +kubebuilder:printcolumn:name="GROUP-NAME",type="string",JSONPath=".spec.forProvider.groupName"
// +kubebuilder:printcolumn:name="GROUP-TYPE",type="integer",JSONPath=".spec.forProvider.groupType"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,harbor}
type UserGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UserGroupSpec   `json:"spec"`
	Status UserGroupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type UserGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []UserGroup `json:"items"`
}

// GetCondition of this UserGroup.
func (mg *UserGroup) GetCondition(ct xpv1.ConditionType) xpv1.Condition {
	return mg.Status.GetCondition(ct)
}

// GetManagementPolicies of this UserGroup.
func (mg *UserGroup) GetManagementPolicies() xpv1.ManagementPolicies {
	return mg.Spec.ManagementPolicies
}

// GetProviderConfigReference of this UserGroup.
func (mg *UserGroup) GetProviderConfigReference() *xpv1.ProviderConfigReference {
	return mg.Spec.ProviderConfigReference
}

// GetWriteConnectionSecretToReference of this UserGroup.
func (mg *UserGroup) GetWriteConnectionSecretToReference() *xpv1.LocalSecretReference {
	return mg.Spec.WriteConnectionSecretToReference
}

// SetConditions of this UserGroup.
func (mg *UserGroup) SetConditions(c ...xpv1.Condition) {
	mg.Status.SetConditions(c...)
}

// SetManagementPolicies of this UserGroup.
func (mg *UserGroup) SetManagementPolicies(r xpv1.ManagementPolicies) {
	mg.Spec.ManagementPolicies = r
}

// SetProviderConfigReference of this UserGroup.
func (mg *UserGroup) SetProviderConfigReference(r *xpv1.ProviderConfigReference) {
	mg.Spec.ProviderConfigReference = r
}

// SetWriteConnectionSecretToReference of this UserGroup.
func (mg *UserGroup) SetWriteConnectionSecretToReference(r *xpv1.LocalSecretReference) {
	mg.Spec.WriteConnectionSecretToReference = r
}
