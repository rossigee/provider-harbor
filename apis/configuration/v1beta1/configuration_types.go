/*
Copyright 2026 Crossplane Harbor Provider.
*/

package v1beta1

import (
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OIDCSettings are the OpenID Connect settings Harbor uses when auth_mode is
// oidc_auth. Each field maps onto the matching oidc_* entry in Harbor's
// system configuration.
type OIDCSettings struct {
	// Name is the label shown on the Harbor login page for the OIDC provider.
	// +kubebuilder:validation:Optional
	Name *string `json:"name,omitempty"`

	// Endpoint is the OIDC issuer URL, for example
	// https://sso.example.com/auth/realms/Realm.
	// +kubebuilder:validation:Optional
	Endpoint *string `json:"endpoint,omitempty"`

	// ClientID is the OIDC client identifier Harbor authenticates as.
	// +kubebuilder:validation:Optional
	ClientID *string `json:"clientId,omitempty"`

	// ClientSecretRef references the OIDC client secret. Harbor never returns
	// the secret, so changes are detected by comparing a hash of the value.
	// +kubebuilder:validation:Optional
	ClientSecretRef *xpv1.SecretKeySelector `json:"clientSecretRef,omitempty"`

	// Scope is the space-separated list of OIDC scopes to request.
	// +kubebuilder:validation:Optional
	Scope *string `json:"scope,omitempty"`

	// UserClaim is the ID token claim used as the Harbor username.
	// +kubebuilder:validation:Optional
	UserClaim *string `json:"userClaim,omitempty"`

	// GroupsClaim is the ID token claim that lists the user's groups.
	// +kubebuilder:validation:Optional
	GroupsClaim *string `json:"groupsClaim,omitempty"`

	// AdminGroup is the group whose members become Harbor system administrators.
	// Set to an empty string to clear it.
	// +kubebuilder:validation:Optional
	AdminGroup *string `json:"adminGroup,omitempty"`

	// GroupFilter is a regular expression that limits which groups are imported.
	// +kubebuilder:validation:Optional
	GroupFilter *string `json:"groupFilter,omitempty"`

	// VerifyCert controls whether Harbor verifies the OIDC provider's TLS
	// certificate.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	VerifyCert *bool `json:"verifyCert,omitempty"`

	// AutoOnboard creates Harbor users automatically on first OIDC login.
	// +kubebuilder:validation:Optional
	AutoOnboard *bool `json:"autoOnboard,omitempty"`

	// Logout makes Harbor's logout also end the OIDC session.
	// +kubebuilder:validation:Optional
	Logout *bool `json:"logout,omitempty"`
}

// ConfigurationParameters define the desired system-wide authentication
// settings of a Harbor instance. A Configuration manages one Harbor instance,
// selected by the ProviderConfig it references.
type ConfigurationParameters struct {
	// AuthMode is Harbor's primary authentication mode.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=db_auth;ldap_auth;http_auth;oidc_auth;uaa_auth
	AuthMode *string `json:"authMode,omitempty"`

	// OIDC holds the OpenID Connect settings. Only applied when they are set.
	// +kubebuilder:validation:Optional
	OIDC *OIDCSettings `json:"oidc,omitempty"`
}

// ConfigurationObservation are the observed Harbor settings.
type ConfigurationObservation struct {
	// AuthMode is the authentication mode Harbor currently uses.
	AuthMode *string `json:"authMode,omitempty"`

	// OIDCClientSecretHash is the SHA-256 (hex) of the OIDC client secret last
	// written to Harbor. It is used to detect rotation of the referenced secret.
	OIDCClientSecretHash *string `json:"oidcClientSecretHash,omitempty"`
}

// A ConfigurationSpec defines the desired state of a Configuration.
type ConfigurationSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              ConfigurationParameters `json:"forProvider"`
}

// A ConfigurationStatus represents the observed state of a Configuration.
type ConfigurationStatus struct {
	xpv1.ConditionedStatus `json:",inline"`
	AtProvider             ConfigurationObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="AUTH-MODE",type="string",JSONPath=".spec.forProvider.authMode"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,harbor}
//
// A Configuration manages the system-wide authentication settings of the Harbor
// instance selected by its ProviderConfig. Create one per instance.
type Configuration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConfigurationSpec   `json:"spec"`
	Status ConfigurationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type ConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Configuration `json:"items"`
}

// GetCondition of this Configuration.
func (mg *Configuration) GetCondition(ct xpv1.ConditionType) xpv1.Condition {
	return mg.Status.GetCondition(ct)
}

// GetManagementPolicies of this Configuration.
func (mg *Configuration) GetManagementPolicies() xpv1.ManagementPolicies {
	return mg.Spec.ManagementPolicies
}

// GetProviderConfigReference of this Configuration.
func (mg *Configuration) GetProviderConfigReference() *xpv1.ProviderConfigReference {
	return mg.Spec.ProviderConfigReference
}

// GetWriteConnectionSecretToReference of this Configuration.
func (mg *Configuration) GetWriteConnectionSecretToReference() *xpv1.LocalSecretReference {
	return mg.Spec.WriteConnectionSecretToReference
}

// SetConditions of this Configuration.
func (mg *Configuration) SetConditions(c ...xpv1.Condition) {
	mg.Status.SetConditions(c...)
}

// SetManagementPolicies of this Configuration.
func (mg *Configuration) SetManagementPolicies(r xpv1.ManagementPolicies) {
	mg.Spec.ManagementPolicies = r
}

// SetProviderConfigReference of this Configuration.
func (mg *Configuration) SetProviderConfigReference(r *xpv1.ProviderConfigReference) {
	mg.Spec.ProviderConfigReference = r
}

// SetWriteConnectionSecretToReference of this Configuration.
func (mg *Configuration) SetWriteConnectionSecretToReference(r *xpv1.LocalSecretReference) {
	mg.Spec.WriteConnectionSecretToReference = r
}
