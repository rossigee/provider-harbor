/*
Copyright 2024 Crossplane Harbor Provider.
*/

package v1beta1

import (
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WebhookParameters defines the desired state of a Webhook
type WebhookParameters struct {
	// ProjectID is the ID of the project this webhook belongs to
	// +kubebuilder:validation:Required
	ProjectID string `json:"projectId"`

	// Name is the name of the webhook
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Description of the webhook
	// +kubebuilder:validation:Optional
	Description *string `json:"description,omitempty"`

	// URL is the endpoint to send webhook events to
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern="^https?://"
	URL string `json:"url"`

	// EventTypes is a list of Harbor events to subscribe to
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Items:Enum=PUSH_ARTIFACT;PULL_ARTIFACT;DELETE_ARTIFACT;SCANNING_COMPLETED;SCANNING_FAILED
	EventTypes []string `json:"eventTypes"`

	// AuthHeader is the optional authentication header value
	// +kubebuilder:validation:Optional
	AuthHeader *string `json:"authHeader,omitempty"`

	// SkipCertVerify skips HTTPS certificate verification (not recommended)
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	SkipCertVerify *bool `json:"skipCertVerify,omitempty"`

	// Enabled controls whether this webhook is active
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	Enabled *bool `json:"enabled,omitempty"`
}

// WebhookObservation defines the observed state of a Webhook
type WebhookObservation struct {
	// ID is the unique identifier of the webhook
	ID *string `json:"id,omitempty"`

	// CreationTime is when the webhook was created
	CreationTime *metav1.Time `json:"creationTime,omitempty"`

	// UpdateTime is when the webhook was last updated
	UpdateTime *metav1.Time `json:"updateTime,omitempty"`

	// Status indicates the current status of the webhook
	Status *string `json:"status,omitempty"`
}

// A WebhookSpec defines the desired state of a Webhook.
type WebhookSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              WebhookParameters `json:"forProvider"`
}

// A WebhookStatus represents the observed state of a Webhook.
type WebhookStatus struct {
	xpv1.ConditionedStatus `json:",inline"`
	AtProvider             WebhookObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="PROJECT",type="string",JSONPath=".spec.forProvider.projectId"
// +kubebuilder:printcolumn:name="WEBHOOK",type="string",JSONPath=".spec.forProvider.name"
// +kubebuilder:printcolumn:name="URL",type="string",JSONPath=".spec.forProvider.url"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,harbor}

// A Webhook is a managed resource that represents a Harbor webhook for event notifications.
type Webhook struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WebhookSpec   `json:"spec"`
	Status WebhookStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// WebhookList contains a list of Webhook.
type WebhookList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Webhook `json:"items"`
}

// GetCondition of this Webhook.
func (mg *Webhook) GetCondition(ct xpv1.ConditionType) xpv1.Condition {
	return mg.Status.GetCondition(ct)
}

// GetManagementPolicies of this Webhook.
func (mg *Webhook) GetManagementPolicies() xpv1.ManagementPolicies {
	return mg.Spec.ManagementPolicies
}

// GetProviderConfigReference of this Webhook.
func (mg *Webhook) GetProviderConfigReference() *xpv1.ProviderConfigReference {
	return mg.Spec.ProviderConfigReference
}

// GetWriteConnectionSecretToReference of this Webhook.
func (mg *Webhook) GetWriteConnectionSecretToReference() *xpv1.LocalSecretReference {
	return mg.Spec.WriteConnectionSecretToReference
}

// SetConditions of this Webhook.
func (mg *Webhook) SetConditions(c ...xpv1.Condition) {
	mg.Status.SetConditions(c...)
}

// SetManagementPolicies of this Webhook.
func (mg *Webhook) SetManagementPolicies(r xpv1.ManagementPolicies) {
	mg.Spec.ManagementPolicies = r
}

// SetProviderConfigReference of this Webhook.
func (mg *Webhook) SetProviderConfigReference(r *xpv1.ProviderConfigReference) {
	mg.Spec.ProviderConfigReference = r
}

// SetWriteConnectionSecretToReference of this Webhook.
func (mg *Webhook) SetWriteConnectionSecretToReference(r *xpv1.LocalSecretReference) {
	mg.Spec.WriteConnectionSecretToReference = r
}
