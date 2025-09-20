/*
Copyright 2025 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

// ScannerRegistrationParameters defines the desired state of ScannerRegistration
type ScannerRegistrationParameters struct {
	// Name is the name of this scanner registration
	// +kubebuilder:validation:Required
	Name *string `json:"name"`

	// URL is the base URL of the scanner adapter
	// +kubebuilder:validation:Required
	URL *string `json:"url"`

	// Description is an optional description of this registration
	// +optional
	Description *string `json:"description,omitempty"`

	// Auth specifies the authentication approach for HTTP communications
	// Supported types: "Basic", "Bearer", "X-ScannerAdapter-API-Key"
	// +optional
	Auth *string `json:"auth,omitempty"`

	// AccessCredential is an optional value of the HTTP Authorization header
	// sent with each request to the Scanner Adapter API
	// +optional
	AccessCredential *string `json:"accessCredential,omitempty"`
}

// ScannerRegistrationObservation defines the observed state of ScannerRegistration
type ScannerRegistrationObservation struct {
	// Name is the name of this scanner registration
	Name *string `json:"name,omitempty"`

	// URL is the base URL of the scanner adapter
	URL *string `json:"url,omitempty"`

	// Description is the description of this registration
	Description *string `json:"description,omitempty"`

	// Auth is the authentication approach for HTTP communications
	Auth *string `json:"auth,omitempty"`

	// AccessCredential is the HTTP Authorization header value
	AccessCredential *string `json:"accessCredential,omitempty"`

	// ID is the unique identifier of the scanner registration
	ID *string `json:"id,omitempty"`

	// UUID is the UUID of this scanner registration
	UUID *string `json:"uuid,omitempty"`

	// CreateTime is the date and time the scanner registration was created
	CreateTime *string `json:"createTime,omitempty"`

	// UpdateTime is the date and time the scanner registration was last updated
	UpdateTime *string `json:"updateTime,omitempty"`
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

// A ScannerRegistration is a managed resource that represents a Harbor scanner registration.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,harbor}
type ScannerRegistration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ScannerRegistrationSpec   `json:"spec"`
	Status ScannerRegistrationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ScannerRegistrationList contains a list of ScannerRegistration
type ScannerRegistrationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ScannerRegistration `json:"items"`
}

// ScannerRegistration type metadata.
var (
	ScannerRegistration_Kind             = "ScannerRegistration"
	ScannerRegistration_GroupKind        = schema.GroupKind{Group: Group, Kind: ScannerRegistration_Kind}.String()
	ScannerRegistration_KindAPIVersion   = ScannerRegistration_Kind + "." + CRDGroupVersion.String()
	ScannerRegistration_GroupVersionKind = CRDGroupVersion.WithKind(ScannerRegistration_Kind)
)

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

// GetPublishConnectionDetailsTo of this ScannerRegistration.
func (mg *ScannerRegistration) GetPublishConnectionDetailsTo() *xpv1.PublishConnectionDetailsTo {
	return mg.Spec.PublishConnectionDetailsTo
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

// SetPublishConnectionDetailsTo of this ScannerRegistration.
func (mg *ScannerRegistration) SetPublishConnectionDetailsTo(r *xpv1.PublishConnectionDetailsTo) {
	mg.Spec.PublishConnectionDetailsTo = r
}

// SetWriteConnectionSecretToReference of this ScannerRegistration.
func (mg *ScannerRegistration) SetWriteConnectionSecretToReference(r *xpv1.SecretReference) {
	mg.Spec.WriteConnectionSecretToReference = r
}

func init() {
	SchemeBuilder.Register(&ScannerRegistration{}, &ScannerRegistrationList{})
}
