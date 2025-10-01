/*
Copyright 2024 Crossplane Harbor Provider.
*/

package v1beta1

import (
	"reflect"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Package type metadata.
const (
	CRDGroup   = "scanner.harbor.m.crossplane.io"
	CRDVersion = "v1beta1"
)

var (
	// CRDGroupVersion is the API Group Version used to register the objects
	CRDGroupVersion = schema.GroupVersion{Group: CRDGroup, Version: CRDVersion}
)

// ScannerRegistration type metadata.
var (
	ScannerRegistrationKind             = reflect.TypeOf(ScannerRegistration{}).Name()
	ScannerRegistrationGroupKind        = schema.GroupKind{Group: CRDGroup, Kind: ScannerRegistrationKind}
	ScannerRegistrationKindAPIVersion   = ScannerRegistrationKind + "." + CRDGroupVersion.String()
	ScannerRegistrationGroupVersionKind = CRDGroupVersion.WithKind(ScannerRegistrationKind)
)

func init() {
	SchemeBuilder.Register(&ScannerRegistration{}, &ScannerRegistrationList{})
}