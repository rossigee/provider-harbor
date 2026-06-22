/*
Copyright 2024 Crossplane Harbor Provider.
*/

package v1beta1

import (
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UserMemberParameters are the configurable fields of a UserMember.
type UserMemberParameters struct {
	// ProjectID is the project name or numeric id the member belongs to.
	ProjectID string `json:"projectId"`
	// Username is the Harbor user added as a project member.
	Username string `json:"username"`
	// Role is the project role: projectAdmin, developer, guest or maintainer.
	Role string `json:"role"`
}

// UserMemberObservation are the observable fields of a UserMember.
type UserMemberObservation struct {
	ID           *string      `json:"id,omitempty"`
	MemberName   *string      `json:"memberName,omitempty"`
	MemberType   *string      `json:"memberType,omitempty"`
	Role         *string      `json:"role,omitempty"`
	CreationTime *metav1.Time `json:"creationTime,omitempty"`
}

// A UserMemberSpec defines the desired state of a UserMember.
type UserMemberSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              UserMemberParameters `json:"forProvider"`
}

// A UserMemberStatus represents the observed state of a UserMember.
type UserMemberStatus struct {
	xpv1.ConditionedStatus `json:",inline"`
	AtProvider             UserMemberObservation `json:"atProvider,omitempty"`
}

// A UserMember is a managed resource that represents a Harbor project user
// member (member_user). It is one half of the split that replaces the deprecated
// catch-all Member kind; GroupMember covers group members.
//
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="USERNAME",type="string",JSONPath=".spec.forProvider.username"
// +kubebuilder:printcolumn:name="ROLE",type="string",JSONPath=".spec.forProvider.role"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,harbor}
type UserMember struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              UserMemberSpec   `json:"spec"`
	Status            UserMemberStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// UserMemberList contains a list of UserMember
type UserMemberList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []UserMember `json:"items"`
}

// GetCondition of this UserMember.
func (mg *UserMember) GetCondition(ct xpv1.ConditionType) xpv1.Condition {
	return mg.Status.GetCondition(ct)
}

// GetManagementPolicies of this UserMember.
func (mg *UserMember) GetManagementPolicies() xpv1.ManagementPolicies {
	return mg.Spec.ManagementPolicies
}

// GetProviderConfigReference of this UserMember.
func (mg *UserMember) GetProviderConfigReference() *xpv1.ProviderConfigReference {
	return mg.Spec.ProviderConfigReference
}

// GetWriteConnectionSecretToReference of this UserMember.
func (mg *UserMember) GetWriteConnectionSecretToReference() *xpv1.LocalSecretReference {
	return mg.Spec.WriteConnectionSecretToReference
}

// SetConditions of this UserMember.
func (mg *UserMember) SetConditions(c ...xpv1.Condition) {
	mg.Status.SetConditions(c...)
}

// SetManagementPolicies of this UserMember.
func (mg *UserMember) SetManagementPolicies(r xpv1.ManagementPolicies) {
	mg.Spec.ManagementPolicies = r
}

// SetProviderConfigReference of this UserMember.
func (mg *UserMember) SetProviderConfigReference(r *xpv1.ProviderConfigReference) {
	mg.Spec.ProviderConfigReference = r
}

// SetWriteConnectionSecretToReference of this UserMember.
func (mg *UserMember) SetWriteConnectionSecretToReference(r *xpv1.LocalSecretReference) {
	mg.Spec.WriteConnectionSecretToReference = r
}
