/*
Copyright 2024 Crossplane Harbor Provider.
*/

package v1alpha1

import (
	"reflect"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Package type metadata.
const (
	RegistryKind = "Registry"
)

func init() {
	SchemeBuilder.Register(&Registry{}, &RegistryList{})
}

// Registry type metadata.
var (
	RegistryKindValue        = reflect.TypeOf(Registry{}).Name()
	RegistryGroupKind        = schema.GroupKind{Group: Group, Kind: RegistryKind}.String()
	RegistryKindAPIVersion   = RegistryKind + "." + SchemeGroupVersion.String()
	RegistryGroupVersionKind = SchemeGroupVersion.WithKind(RegistryKind)
)