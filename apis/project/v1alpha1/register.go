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
	ProjectKind = "Project"
)

func init() {
	SchemeBuilder.Register(&Project{}, &ProjectList{})
}

// Project type metadata.
var (
	ProjectKindValue        = reflect.TypeOf(Project{}).Name()
	ProjectGroupKind        = schema.GroupKind{Group: Group, Kind: ProjectKind}.String()
	ProjectKindAPIVersion   = ProjectKind + "." + SchemeGroupVersion.String()
	ProjectGroupVersionKind = SchemeGroupVersion.WithKind(ProjectKind)
)