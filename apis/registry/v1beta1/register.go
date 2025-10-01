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
	CRDGroup   = "registry.harbor.m.crossplane.io"
	CRDVersion = "v1beta1"
)

var (
	// CRDGroupVersion is the API Group Version used to register the objects
	CRDGroupVersion = schema.GroupVersion{Group: CRDGroup, Version: CRDVersion}
)

// Registry type metadata.
var (
	RegistryKind             = reflect.TypeOf(Registry{}).Name()
	RegistryGroupKind        = schema.GroupKind{Group: CRDGroup, Kind: RegistryKind}
	RegistryKindAPIVersion   = RegistryKind + "." + CRDGroupVersion.String()
	RegistryGroupVersionKind = CRDGroupVersion.WithKind(RegistryKind)
)

func init() {
	SchemeBuilder.Register(&Registry{}, &RegistryList{})
}