/*
Copyright 2024 Crossplane Harbor Provider.
*/

// Package apis contains Kubernetes API for the native Harbor provider.
package apis

import (
	"k8s.io/apimachinery/pkg/runtime"

	// V2 Native API groups (namespaced)
	projectv1beta1 "github.com/rossigee/provider-harbor/apis/project/v1beta1"
	registryv1beta1 "github.com/rossigee/provider-harbor/apis/registry/v1beta1"
	scannerv1beta1 "github.com/rossigee/provider-harbor/apis/scanner/v1beta1"
	userv1beta1 "github.com/rossigee/provider-harbor/apis/user/v1beta1"

	// Provider config APIs
	v1beta1 "github.com/rossigee/provider-harbor/apis/v1beta1"
)

func init() {
	// Register the native types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes,
		// V2 Native APIs - v1beta1 (namespaced only)
		projectv1beta1.SchemeBuilder.AddToScheme,
		registryv1beta1.SchemeBuilder.AddToScheme,
		scannerv1beta1.SchemeBuilder.AddToScheme,
		userv1beta1.SchemeBuilder.AddToScheme,

		// Provider config APIs
		v1beta1.SchemeBuilder.AddToScheme,
	)
}

// AddToSchemes may be used to add all resources defined in the project to a Scheme
var AddToSchemes runtime.SchemeBuilder

// AddToScheme adds all Resources to the Scheme
func AddToScheme(s *runtime.Scheme) error {
	return AddToSchemes.AddToScheme(s)
}