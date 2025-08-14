/*
Copyright 2022 Upbound Inc.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	v1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

// UserWithGeneratedPasswordInitParameters defines custom parameters for User initialization with generated password
type UserWithGeneratedPasswordInitParameters struct {
	UserInitParameters `json:",inline"`

	// Generate a secure random password and store it in the specified secret
	// This is mutually exclusive with passwordSecretRef
	// +kubebuilder:validation:Optional
	GeneratePasswordInSecret *GeneratePasswordConfig `json:"generatePasswordInSecret,omitempty"`
}

// UserWithGeneratedPasswordParameters defines custom parameters for User with generated password
type UserWithGeneratedPasswordParameters struct {
	UserParameters `json:",inline"`

	// Generate a secure random password and store it in the specified secret  
	// This is mutually exclusive with passwordSecretRef
	// +kubebuilder:validation:Optional
	GeneratePasswordInSecret *GeneratePasswordConfig `json:"generatePasswordInSecret,omitempty"`
}

// GeneratePasswordConfig specifies how to generate and store a password
type GeneratePasswordConfig struct {
	// Name of the secret to create with the generated password
	Name string `json:"name"`
	
	// Namespace where the secret should be created (defaults to same as user resource)
	// +kubebuilder:validation:Optional
	Namespace string `json:"namespace,omitempty"`
	
	// Key within the secret to store the password (defaults to "password")
	// +kubebuilder:validation:Optional
	Key string `json:"key,omitempty"`
	
	// Length of the generated password (defaults to 16, minimum 8)
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=8
	// +kubebuilder:validation:Maximum=128
	Length *int `json:"length,omitempty"`
}

// UserWithGeneratedPasswordSpec defines the desired state of User with generated password support
type UserWithGeneratedPasswordSpec struct {
	v1.ResourceSpec `json:",inline"`
	ForProvider     UserWithGeneratedPasswordParameters     `json:"forProvider"`
	InitProvider    UserWithGeneratedPasswordInitParameters `json:"initProvider,omitempty"`
}

// UserWithGeneratedPassword is the Schema for the Users API with password generation support
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,harbor}
type UserWithGeneratedPassword struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// +kubebuilder:validation:XValidation:rule="!('*' in self.managementPolicies || 'Create' in self.managementPolicies || 'Update' in self.managementPolicies) || has(self.forProvider.email) || (has(self.initProvider) && has(self.initProvider.email))",message="spec.forProvider.email is a required parameter"
	// +kubebuilder:validation:XValidation:rule="!('*' in self.managementPolicies || 'Create' in self.managementPolicies || 'Update' in self.managementPolicies) || has(self.forProvider.fullName) || (has(self.initProvider) && has(self.initProvider.fullName))",message="spec.forProvider.fullName is a required parameter"
	// +kubebuilder:validation:XValidation:rule="!('*' in self.managementPolicies || 'Create' in self.managementPolicies || 'Update' in self.managementPolicies) || has(self.forProvider.passwordSecretRef) || has(self.forProvider.generatePasswordInSecret)",message="either spec.forProvider.passwordSecretRef or spec.forProvider.generatePasswordInSecret is required"
	// +kubebuilder:validation:XValidation:rule="!('*' in self.managementPolicies || 'Create' in self.managementPolicies || 'Update' in self.managementPolicies) || has(self.forProvider.username) || (has(self.initProvider) && has(self.initProvider.username))",message="spec.forProvider.username is a required parameter"
	Spec   UserWithGeneratedPasswordSpec `json:"spec"`
	Status UserStatus                    `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// UserWithGeneratedPasswordList contains a list of Users with generated password support
type UserWithGeneratedPasswordList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []UserWithGeneratedPassword `json:"items"`
}

// Repository type metadata.
var (
	UserWithGeneratedPassword_Kind             = "UserWithGeneratedPassword"
	UserWithGeneratedPassword_GroupKind        = schema.GroupKind{Group: CRDGroup, Kind: UserWithGeneratedPassword_Kind}.String()
	UserWithGeneratedPassword_KindAPIVersion   = UserWithGeneratedPassword_Kind + "." + CRDGroupVersion.String()
	UserWithGeneratedPassword_GroupVersionKind = CRDGroupVersion.WithKind(UserWithGeneratedPassword_Kind)
)

func init() {
	SchemeBuilder.Register(&UserWithGeneratedPassword{}, &UserWithGeneratedPasswordList{})
}