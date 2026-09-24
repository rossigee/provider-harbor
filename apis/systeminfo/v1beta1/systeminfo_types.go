/*
Copyright 2025 Crossplane Harbor Provider.
*/

package v1beta1

import (
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SystemInfoObservation defines the observed state of Harbor system information
type SystemInfoObservation struct {
	// HarborVersion is the version of Harbor
	// +kubebuilder:validation:Optional
	HarborVersion *string `json:"harborVersion,omitempty"`

	// RegistryStorageProviderName is the storage provider type
	// +kubebuilder:validation:Optional
	RegistryStorageProviderName *string `json:"registryStorageProviderName,omitempty"`

	// DatabaseType is the type of database (postgresql, mysql, etc.)
	// +kubebuilder:validation:Optional
	DatabaseType *string `json:"databaseType,omitempty"`

	// AuthMode is the authentication mode (db_auth, ldap_auth, oidc_auth, etc.)
	// +kubebuilder:validation:Optional
	AuthMode *string `json:"authMode,omitempty"`

	// SelfRegistration indicates if self-registration is enabled
	// +kubebuilder:validation:Optional
	SelfRegistration *bool `json:"selfRegistration,omitempty"`

	// HasCARoot indicates if a custom CA certificate is installed
	// +kubebuilder:validation:Optional
	HasCARoot *bool `json:"hasCARoot,omitempty"`

	// CertificateExpiration is the Unix timestamp of certificate expiration
	// +kubebuilder:validation:Optional
	CertificateExpiration *int64 `json:"certificateExpiration,omitempty"`

	// NotaryEnabled indicates if Notary is enabled
	// +kubebuilder:validation:Optional
	NotaryEnabled *bool `json:"notaryEnabled,omitempty"`

	// TraceEnabled indicates if tracing is enabled
	// +kubebuilder:validation:Optional
	TraceEnabled *bool `json:"traceEnabled,omitempty"`

	// ObservedAt is the timestamp when this information was last observed
	// +kubebuilder:validation:Optional
	ObservedAt *metav1.Time `json:"observedAt,omitempty"`
}

// SystemInfoParameters defines the desired state of SystemInfo (typically empty for read-only)
type SystemInfoParameters struct {
	// This resource is read-only. No parameters are typically provided.
}

// A SystemInfoSpec defines the desired state of a SystemInfo resource.
type SystemInfoSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              SystemInfoParameters `json:"forProvider"`
}

// A SystemInfoStatus represents the observed state of SystemInfo.
type SystemInfoStatus struct {
	xpv1.ConditionedStatus `json:",inline"`
	AtProvider             SystemInfoObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="VERSION",type="string",JSONPath=".status.atProvider.harborVersion"
// +kubebuilder:printcolumn:name="STORAGE",type="string",JSONPath=".status.atProvider.registryStorageProviderName"
// +kubebuilder:printcolumn:name="AUTH-MODE",type="string",JSONPath=".status.atProvider.authMode"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,harbor}
type SystemInfo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SystemInfoSpec   `json:"spec"`
	Status SystemInfoStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type SystemInfoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SystemInfo `json:"items"`
}
