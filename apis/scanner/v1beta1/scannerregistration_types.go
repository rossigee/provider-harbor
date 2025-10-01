/*
Copyright 2024 Crossplane Harbor Provider.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
)

// ScannerRegistrationParameters defines the desired state of a ScannerRegistration
type ScannerRegistrationParameters struct {
	// Name is the name of the scanner
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Description is a description of the scanner
	// +kubebuilder:validation:Optional
	Description *string `json:"description,omitempty"`

	// URL is the URL of the scanner
	// +kubebuilder:validation:Required
	URL string `json:"url"`

	// Auth is the authentication method
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=Bearer;Basic;APIKey
	Auth *string `json:"auth,omitempty"`

	// AccessCredential is the access credential for the scanner
	// +kubebuilder:validation:Optional
	AccessCredential *string `json:"accessCredential,omitempty"`

	// SkipCertVerify indicates whether to skip certificate verification
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	SkipCertVerify *bool `json:"skipCertVerify,omitempty"`

	// UseInternalAddr indicates whether to use internal address
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	UseInternalAddr *bool `json:"useInternalAddr,omitempty"`

	// Disabled indicates whether the scanner is disabled
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	Disabled *bool `json:"disabled,omitempty"`

	// IsDefault indicates whether this is the default scanner
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	IsDefault *bool `json:"isDefault,omitempty"`
}

// ScannerRegistrationObservation defines the observed state of a ScannerRegistration
type ScannerRegistrationObservation struct {
	// UUID is the unique identifier of the scanner registration
	UUID *string `json:"uuid,omitempty"`

	// CreationTime is when the scanner registration was created
	CreationTime *metav1.Time `json:"creationTime,omitempty"`

	// UpdateTime is when the scanner registration was last updated
	UpdateTime *metav1.Time `json:"updateTime,omitempty"`

	// Health indicates the health status of the scanner
	Health *string `json:"health,omitempty"`

	// Adapter is the scanner adapter name
	Adapter *string `json:"adapter,omitempty"`

	// Vendor is the scanner vendor
	Vendor *string `json:"vendor,omitempty"`

	// Version is the scanner version
	Version *string `json:"version,omitempty"`
}

// A ScannerRegistrationSpec defines the desired state of a ScannerRegistration.
type ScannerRegistrationSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       ScannerRegistrationParameters `json:"forProvider"`
}

// A ScannerRegistrationStatus represents the observed state of a ScannerRegistration.
type ScannerRegistrationStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ScannerRegistrationObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="SCANNER-UUID",type="string",JSONPath=".status.atProvider.uuid"
// +kubebuilder:printcolumn:name="HEALTH",type="string",JSONPath=".status.atProvider.health"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,harbor}
type ScannerRegistration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ScannerRegistrationSpec   `json:"spec"`
	Status ScannerRegistrationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type ScannerRegistrationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ScannerRegistration `json:"items"`
}

// GetCondition of this ScannerRegistration.
func (mg *ScannerRegistration) GetCondition(ct xpv1.ConditionType) xpv1.Condition {
	return mg.Status.GetCondition(ct)
}

// GetDeletionPolicy of this ScannerRegistration.
func (mg *ScannerRegistration) GetDeletionPolicy() xpv1.DeletionPolicy {
	return mg.Spec.DeletionPolicy
}

// GetManagementPolicies of this ScannerRegistration.
func (mg *ScannerRegistration) GetManagementPolicies() xpv1.ManagementPolicies {
	return mg.Spec.ManagementPolicies
}

// GetProviderConfigReference of this ScannerRegistration.
func (mg *ScannerRegistration) GetProviderConfigReference() *xpv1.Reference {
	return mg.Spec.ProviderConfigReference
}

// GetWriteConnectionSecretToReference of this ScannerRegistration.
func (mg *ScannerRegistration) GetWriteConnectionSecretToReference() *xpv1.SecretReference {
	return mg.Spec.WriteConnectionSecretToReference
}

// SetConditions of this ScannerRegistration.
func (mg *ScannerRegistration) SetConditions(c ...xpv1.Condition) {
	mg.Status.SetConditions(c...)
}

// SetDeletionPolicy of this ScannerRegistration.
func (mg *ScannerRegistration) SetDeletionPolicy(r xpv1.DeletionPolicy) {
	mg.Spec.DeletionPolicy = r
}

// SetManagementPolicies of this ScannerRegistration.
func (mg *ScannerRegistration) SetManagementPolicies(r xpv1.ManagementPolicies) {
	mg.Spec.ManagementPolicies = r
}

// SetProviderConfigReference of this ScannerRegistration.
func (mg *ScannerRegistration) SetProviderConfigReference(r *xpv1.Reference) {
	mg.Spec.ProviderConfigReference = r
}

// SetWriteConnectionSecretToReference of this ScannerRegistration.
func (mg *ScannerRegistration) SetWriteConnectionSecretToReference(r *xpv1.SecretReference) {
	mg.Spec.WriteConnectionSecretToReference = r
}