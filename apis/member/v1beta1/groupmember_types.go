/*
Copyright 2024 Crossplane Harbor Provider.
*/

package v1beta1

import (
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GroupMemberParameters are the configurable fields of a GroupMember.
type GroupMemberParameters struct {
	// ProjectID is the project name or numeric id the member belongs to.
	ProjectID string `json:"projectId"`
	// GroupName is the Harbor user group added as a project member.
	GroupName string `json:"groupName"`
	// Role is the project role: projectAdmin, developer, guest or maintainer.
	Role string `json:"role"`
	// GroupType selects the Harbor group source:
	//   1 = LDAP — the group is an LDAP group, matched by its LDAP group DN.
	//   2 = HTTP — the group is supplied by an HTTP auth proxy via request headers.
	//   3 = OIDC — the group comes from the OIDC provider's groups claim.
	// Defaults to 2 (HTTP).
	// +optional
	// +kubebuilder:default=2
	GroupType *int64 `json:"groupType,omitempty"`
}

// GroupMemberObservation are the observable fields of a GroupMember.
type GroupMemberObservation struct {
	ID           *string      `json:"id,omitempty"`
	MemberName   *string      `json:"memberName,omitempty"`
	MemberType   *string      `json:"memberType,omitempty"`
	Role         *string      `json:"role,omitempty"`
	CreationTime *metav1.Time `json:"creationTime,omitempty"`
}

// A GroupMemberSpec defines the desired state of a GroupMember.
type GroupMemberSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              GroupMemberParameters `json:"forProvider"`
}

// A GroupMemberStatus represents the observed state of a GroupMember.
type GroupMemberStatus struct {
	xpv1.ConditionedStatus `json:",inline"`
	AtProvider             GroupMemberObservation `json:"atProvider,omitempty"`
}

// A GroupMember is a managed resource that represents a Harbor project group
// member (member_group). It is one half of the split that replaces the
// deprecated catch-all Member kind; UserMember covers user members.
//
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="GROUP",type="string",JSONPath=".spec.forProvider.groupName"
// +kubebuilder:printcolumn:name="ROLE",type="string",JSONPath=".spec.forProvider.role"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,harbor}
type GroupMember struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              GroupMemberSpec   `json:"spec"`
	Status            GroupMemberStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GroupMemberList contains a list of GroupMember
type GroupMemberList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GroupMember `json:"items"`
}

// GetCondition of this GroupMember.
func (mg *GroupMember) GetCondition(ct xpv1.ConditionType) xpv1.Condition {
	return mg.Status.GetCondition(ct)
}

// GetManagementPolicies of this GroupMember.
func (mg *GroupMember) GetManagementPolicies() xpv1.ManagementPolicies {
	return mg.Spec.ManagementPolicies
}

// GetProviderConfigReference of this GroupMember.
func (mg *GroupMember) GetProviderConfigReference() *xpv1.ProviderConfigReference {
	return mg.Spec.ProviderConfigReference
}

// GetWriteConnectionSecretToReference of this GroupMember.
func (mg *GroupMember) GetWriteConnectionSecretToReference() *xpv1.LocalSecretReference {
	return mg.Spec.WriteConnectionSecretToReference
}

// SetConditions of this GroupMember.
func (mg *GroupMember) SetConditions(c ...xpv1.Condition) {
	mg.Status.SetConditions(c...)
}

// SetManagementPolicies of this GroupMember.
func (mg *GroupMember) SetManagementPolicies(r xpv1.ManagementPolicies) {
	mg.Spec.ManagementPolicies = r
}

// SetProviderConfigReference of this GroupMember.
func (mg *GroupMember) SetProviderConfigReference(r *xpv1.ProviderConfigReference) {
	mg.Spec.ProviderConfigReference = r
}

// SetWriteConnectionSecretToReference of this GroupMember.
func (mg *GroupMember) SetWriteConnectionSecretToReference(r *xpv1.LocalSecretReference) {
	mg.Spec.WriteConnectionSecretToReference = r
}
