/*
Copyright 2024 Crossplane Harbor Provider.
*/

package controller

import (
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
)

const (
	// ExternalNameAnnotation is the annotation key used to store the external identifier
	ExternalNameAnnotation = "crossplane.io/external-name"
)

// GetExternalName retrieves the external name from resource annotations
func GetExternalName(mg resource.Managed) string {
	return mg.GetAnnotations()[ExternalNameAnnotation]
}

// SetExternalName sets the external name in resource annotations
func SetExternalName(mg resource.Managed, name string) {
	if mg.GetAnnotations() == nil {
		mg.SetAnnotations(make(map[string]string))
	}
	annotations := mg.GetAnnotations()
	annotations[ExternalNameAnnotation] = name
	mg.SetAnnotations(annotations)
}

// HasExternalName checks if a resource has an external name set
func HasExternalName(mg resource.Managed) bool {
	return GetExternalName(mg) != ""
}
