/*
Copyright 2024 Crossplane Harbor Provider.
*/

// Package v1beta1 contains the v1beta1 API of the harbor scan provider.
// +kubebuilder:object:generate=true
// +groupName=scan.harbor.m.crossplane.io
// +versionName=v1beta1
package v1beta1

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	Group   = "scan.harbor.m.crossplane.io"
	Version = "v1beta1"
)

var (
	SchemeGroupVersion = schema.GroupVersion{Group: Group, Version: Version}
	SchemeBuilder      = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme        = SchemeBuilder.AddToScheme
)

func addKnownTypes(s *runtime.Scheme) error {
	return nil
}
