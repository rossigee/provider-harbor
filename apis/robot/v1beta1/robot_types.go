/*
Copyright 2024 Crossplane Harbor Provider.
*/

package v1beta1

import (
	"github.com/crossplane/crossplane/apis/v2/core/v2"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RobotPermission defines permissions for a robot account
type RobotPermission struct {
	// Namespace is the resource namespace (e.g., "project", "repository")
	// +kubebuilder:validation:Required
	Namespace string `json:"namespace"`

	// Access is a list of access types (e.g., "pull", "push", "delete")
	// +kubebuilder:validation:Required
	Access []string `json:"access"`
}

// RobotParameters defines the desired state of a Robot account
type RobotParameters struct {
	// Name is the name of the robot account
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Description of the robot account
	// +kubebuilder:validation:Optional
	Description *string `json:"description,omitempty"`

	// ProjectID is the ID of the project (optional for system-level robots)
	// +kubebuilder:validation:Optional
	ProjectID *string `json:"projectId,omitempty"`

	// ExpiresIn is the number of days until the robot account expires
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=1
	ExpiresIn *int64 `json:"expiresIn,omitempty"`

	// Permissions define what the robot can do
	// +kubebuilder:validation:Required
	Permissions []RobotPermission `json:"permissions"`
}

// RobotObservation defines the observed state of a Robot account
type RobotObservation struct {
	// ID is the unique identifier of the robot account
	ID *string `json:"id,omitempty"`

	// Secret is the authentication secret (token) for the robot
	Secret *string `json:"secret,omitempty"`

	// ExpiresAt is when the robot account expires
	ExpiresAt *metav1.Time `json:"expiresAt,omitempty"`

	// CreationTime is when the robot was created
	CreationTime *metav1.Time `json:"creationTime,omitempty"`

	// UpdateTime is when the robot was last updated
	UpdateTime *metav1.Time `json:"updateTime,omitempty"`
}

// A RobotSpec defines the desired state of a Robot account.
type RobotSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              RobotParameters `json:"forProvider"`
}

// A RobotStatus represents the observed state of a Robot account.
type RobotStatus struct {
	xpv1.ConditionedStatus `json:",inline"`
	AtProvider             RobotObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="NAME",type="string",JSONPath=".spec.forProvider.name"
// +kubebuilder:printcolumn:name="EXPIRES",type="date",JSONPath=".status.atProvider.expiresAt"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,harbor}

// A Robot is a managed resource that represents a Harbor robot account (service account).
type Robot struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RobotSpec   `json:"spec"`
	Status RobotStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RobotList contains a list of Robot.
type RobotList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Robot `json:"items"`
}

// GetCondition of this Robot.
func (mg *Robot) GetCondition(ct xpv1.ConditionType) xpv1.Condition {
	return mg.Status.GetCondition(ct)
}

// GetManagementPolicies of this Robot.
func (mg *Robot) GetManagementPolicies() xpv1.ManagementPolicies {
	return mg.Spec.ManagementPolicies
}

// GetProviderConfigReference of this Robot.
func (mg *Robot) GetProviderConfigReference() *xpv1.ProviderConfigReference {
	return mg.Spec.ProviderConfigReference
}

// GetWriteConnectionSecretToReference of this Robot.
func (mg *Robot) GetWriteConnectionSecretToReference() *xpv1.LocalSecretReference {
	return mg.Spec.WriteConnectionSecretToReference
}

// SetConditions of this Robot.
func (mg *Robot) SetConditions(c ...xpv1.Condition) {
	mg.Status.SetConditions(c...)
}

// SetManagementPolicies of this Robot.
func (mg *Robot) SetManagementPolicies(r xpv1.ManagementPolicies) {
	mg.Spec.ManagementPolicies = r
}

// SetProviderConfigReference of this Robot.
func (mg *Robot) SetProviderConfigReference(r *xpv1.ProviderConfigReference) {
	mg.Spec.ProviderConfigReference = r
}

// SetWriteConnectionSecretToReference of this Robot.
func (mg *Robot) SetWriteConnectionSecretToReference(r *xpv1.LocalSecretReference) {
	mg.Spec.WriteConnectionSecretToReference = r
}
