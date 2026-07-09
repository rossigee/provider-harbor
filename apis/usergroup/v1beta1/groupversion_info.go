/*
Copyright 2024 Crossplane Harbor Provider.
*/

// Package v1beta1 contains the v1beta1 API of the harbor usergroup provider.
// +kubebuilder:object:generate=true
// +groupName=usergroup.harbor.m.crossplane.io
// +versionName=v1beta1
package v1beta1

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	Group   = "usergroup.harbor.m.crossplane.io"
	Version = "v1beta1"
)

var (
	SchemeGroupVersion = schema.GroupVersion{Group: Group, Version: Version}
	SchemeBuilder      = runtime.NewSchemeBuilder(addKnownTypes)
)

func addKnownTypes(s *runtime.Scheme) error {
	return nil
}
