/*
Copyright 2024 Crossplane Harbor Provider.
*/

// Package v1beta1 contains the v1beta1 API of the harbor project provider.
// +kubebuilder:object:generate=true
// +groupName=project.harbor.m.crossplane.io
// +versionName=v1beta1
package v1beta1

import (
	"k8s.io/apimachinery/pkg/runtime"
)

// Package type metadata.
const (
	Group   = "project.harbor.m.crossplane.io"
	Version = "v1beta1"
)

var (
	// SchemeGroupVersion is group version used to register these objects
	SchemeGroupVersion = schema.GroupVersion{Group: Group, Version: Version}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

func addKnownTypes(s *runtime.Scheme) error {
	s.AddKnownTypes(SchemeGroupVersion,
		&Project{},
		&ProjectList{},
	)
	return nil
}
