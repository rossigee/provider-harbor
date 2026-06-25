/*
Copyright 2024 Crossplane Harbor Provider.
*/

package v1beta1

import (
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MemberParameters are the configurable fields of a Member.
type MemberParameters struct {
	// ProjectID is the project name or numeric id the member belongs to.
	// +kubebuilder:validation:Required
	ProjectID string `json:"projectId"`

	// Type selects the member entity kind.
	// "user" adds a local Harbor user by username.
	// "group" adds an LDAP/HTTP/OIDC group by group name.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=user;group
	Type string `json:"type"`

	// Username is the Harbor local user to add. Required when type is "user".
	// +kubebuilder:validation:Optional
	Username *string `json:"username,omitempty"`

	// GroupName is the Harbor group to add. Required when type is "group".
	// +kubebuilder:validation:Optional
	GroupName *string `json:"groupName,omitempty"`

	// GroupType selects the group source. Only used when type is "group".
	//   1 = LDAP — group matched by LDAP group DN.
	//   2 = HTTP — group supplied by an HTTP auth proxy.
	//   3 = OIDC — group from the OIDC provider's groups claim (default).
	// +optional
	// +kubebuilder:default=3
	GroupType *int64 `json:"groupType,omitempty"`

	// Role is the project role: projectAdmin, developer, guest or maintainer.
	// +kubebuilder:validation:Required
	Role string `json:"role"`
}

// MemberObservation are the observable fields of a Member.
type MemberObservation struct {
	ID           *string      `json:"id,omitempty"`
	MemberName   *string      `json:"memberName,omitempty"`
	MemberType   *string      `json:"memberType,omitempty"`
	Role         *string      `json:"role,omitempty"`
	CreationTime *metav1.Time `json:"creationTime,omitempty"`
}

// A MemberSpec defines the desired state of a Member.
type MemberSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              MemberParameters `json:"forProvider"`
}

// A MemberStatus represents the observed state of a Member.
type MemberStatus struct {
	xpv1.ConditionedStatus `json:",inline"`
	AtProvider             MemberObservation `json:"atProvider,omitempty"`
}

// A Member is a managed resource that represents a Harbor project member —
// either a local user (type: user) or a group (type: group). The type
// discriminator follows the same pattern as Robot.level: the spec shape is
// uniform; only the Harbor endpoint sub-path differs.
//
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="TYPE",type="string",JSONPath=".spec.forProvider.type"
// +kubebuilder:printcolumn:name="ROLE",type="string",JSONPath=".spec.forProvider.role"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,harbor}
type Member struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              MemberSpec   `json:"spec"`
	Status            MemberStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MemberList contains a list of Member.
type MemberList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Member `json:"items"`
}

func (mg *Member) GetCondition(ct xpv1.ConditionType) xpv1.Condition {
	return mg.Status.GetCondition(ct)
}

func (mg *Member) GetManagementPolicies() xpv1.ManagementPolicies {
	return mg.Spec.ManagementPolicies
}

func (mg *Member) GetProviderConfigReference() *xpv1.ProviderConfigReference {
	return mg.Spec.ProviderConfigReference
}

func (mg *Member) GetWriteConnectionSecretToReference() *xpv1.LocalSecretReference {
	return mg.Spec.WriteConnectionSecretToReference
}

func (mg *Member) SetConditions(c ...xpv1.Condition) {
	mg.Status.SetConditions(c...)
}

func (mg *Member) SetManagementPolicies(r xpv1.ManagementPolicies) {
	mg.Spec.ManagementPolicies = r
}

func (mg *Member) SetProviderConfigReference(r *xpv1.ProviderConfigReference) {
	mg.Spec.ProviderConfigReference = r
}

func (mg *Member) SetWriteConnectionSecretToReference(r *xpv1.LocalSecretReference) {
	mg.Spec.WriteConnectionSecretToReference = r
}
