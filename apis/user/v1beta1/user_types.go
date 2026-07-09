/*
Copyright 2024 Crossplane Harbor Provider.
*/

package v1beta1

import (
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UserParameters defines the desired state of a User
type UserParameters struct {
	// Username is the username for the Harbor user
	// +kubebuilder:validation:Required
	Username string `json:"username"`

	// Email is the email address of the user
	// +kubebuilder:validation:Required
	Email string `json:"email"`

	// Password is the password for the user
	// +kubebuilder:validation:Optional
	PasswordSecretRef *xpv1.SecretKeySelector `json:"passwordSecretRef,omitempty"`

	// Realname is the real name of the user
	// +kubebuilder:validation:Optional
	Realname *string `json:"realname,omitempty"`

	// Comment is an optional comment about the user
	// +kubebuilder:validation:Optional
	Comment *string `json:"comment,omitempty"`

	// SysAdminFlag indicates if the user is a system administrator
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	SysAdminFlag *bool `json:"sysAdminFlag,omitempty"`
}

// UserObservation defines the observed state of a User
type UserObservation struct {
	// ID is the unique identifier of the user in Harbor
	ID *int64 `json:"id,omitempty"`

	// CreationTime is when the user was created
	CreationTime *metav1.Time `json:"creationTime,omitempty"`

	// UpdateTime is when the user was last updated
	UpdateTime *metav1.Time `json:"updateTime,omitempty"`

	// AdminRoleInAuth indicates if the user has admin role in authentication
	AdminRoleInAuth *bool `json:"adminRoleInAuth,omitempty"`
}

// A UserSpec defines the desired state of a User.
type UserSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              UserParameters `json:"forProvider"`
}

// A UserStatus represents the observed state of a User.
type UserStatus struct {
	xpv1.ConditionedStatus `json:",inline"`
	AtProvider             UserObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="USER-ID",type="string",JSONPath=".status.atProvider.id"
// +kubebuilder:printcolumn:name="USERNAME",type="string",JSONPath=".spec.forProvider.username"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,harbor}
type User struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UserSpec   `json:"spec"`
	Status UserStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type UserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []User `json:"items"`
}

// GetCondition of this User.
func (mg *User) GetCondition(ct xpv1.ConditionType) xpv1.Condition {
	return mg.Status.GetCondition(ct)
}

// GetManagementPolicies of this User.
func (mg *User) GetManagementPolicies() xpv1.ManagementPolicies {
	return mg.Spec.ManagementPolicies
}

// GetProviderConfigReference of this User.
func (mg *User) GetProviderConfigReference() *xpv1.ProviderConfigReference {
	return mg.Spec.ProviderConfigReference
}

// GetWriteConnectionSecretToReference of this User.
func (mg *User) GetWriteConnectionSecretToReference() *xpv1.LocalSecretReference {
	return mg.Spec.WriteConnectionSecretToReference
}

// SetConditions of this User.
func (mg *User) SetConditions(c ...xpv1.Condition) {
	mg.Status.SetConditions(c...)
}

// SetManagementPolicies of this User.
func (mg *User) SetManagementPolicies(r xpv1.ManagementPolicies) {
	mg.Spec.ManagementPolicies = r
}

// SetProviderConfigReference of this User.
func (mg *User) SetProviderConfigReference(r *xpv1.ProviderConfigReference) {
	mg.Spec.ProviderConfigReference = r
}

// SetWriteConnectionSecretToReference of this User.
func (mg *User) SetWriteConnectionSecretToReference(r *xpv1.LocalSecretReference) {
	mg.Spec.WriteConnectionSecretToReference = r
}
