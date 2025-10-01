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
	CRDGroup   = "project.harbor.m.crossplane.io"
	CRDVersion = "v1beta1"
)

var (
	// CRDGroupVersion is the API Group Version used to register the objects
	CRDGroupVersion = schema.GroupVersion{Group: CRDGroup, Version: CRDVersion}
)

// Project type metadata.
var (
	ProjectKind             = reflect.TypeOf(Project{}).Name()
	ProjectGroupKind        = schema.GroupKind{Group: CRDGroup, Kind: ProjectKind}
	ProjectKindAPIVersion   = ProjectKind + "." + CRDGroupVersion.String()
	ProjectGroupVersionKind = CRDGroupVersion.WithKind(ProjectKind)
)

func init() {
	SchemeBuilder.Register(&Project{}, &ProjectList{})
}