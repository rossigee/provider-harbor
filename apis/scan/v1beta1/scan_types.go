/*
Copyright 2024 Crossplane Harbor Provider.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

type ScanParameters struct {
	ProjectID      string `json:"projectId"`
	RepositoryName string `json:"repositoryName"`
	Reference      string `json:"reference"`
}

type ScanObservation struct {
	ID             *string      `json:"id,omitempty"`
	Status         *string      `json:"status,omitempty"`
	CriticalCount  *int64       `json:"criticalCount,omitempty"`
	HighCount      *int64       `json:"highCount,omitempty"`
	MediumCount    *int64       `json:"mediumCount,omitempty"`
	LowCount       *int64       `json:"lowCount,omitempty"`
	StartTime      *metav1.Time `json:"startTime,omitempty"`
	EndTime        *metav1.Time `json:"endTime,omitempty"`
}

type ScanSpec struct {
	xpv1.ManagedResourceSpec `json:",inline"`
	ForProvider              ScanParameters `json:"forProvider"`
}

type ScanStatus struct {
	xpv1.ConditionedStatus `json:",inline"`
	AtProvider             ScanObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="STATUS",type="string",JSONPath=".status.atProvider.status"
// +kubebuilder:printcolumn:name="CRITICAL",type="integer",JSONPath=".status.atProvider.criticalCount"
// +kubebuilder:printcolumn:name="HIGH",type="integer",JSONPath=".status.atProvider.highCount"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,harbor}

type Scan struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ScanSpec   `json:"spec"`
	Status            ScanStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type ScanList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Scan `json:"items"`
}
