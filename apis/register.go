/*
Copyright 2024 Crossplane Harbor Provider.
*/

// Package apis contains Kubernetes API for the native Harbor provider.
package apis

import (
	"k8s.io/apimachinery/pkg/runtime"

	// Native API groups
	projectv1alpha1 "github.com/rossigee/provider-harbor/apis/project/v1alpha1"
	registryv1alpha1 "github.com/rossigee/provider-harbor/apis/registry/v1alpha1"
	userv1alpha1 "github.com/rossigee/provider-harbor/apis/user/v1alpha1"

	// Provider config APIs
	v1alpha1apis "github.com/rossigee/provider-harbor/apis/v1alpha1"
	v1beta1 "github.com/rossigee/provider-harbor/apis/v1beta1"
)

func init() {
	// Register the native types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes,
		// Native APIs
		projectv1alpha1.SchemeBuilder.AddToScheme,
		registryv1alpha1.SchemeBuilder.AddToScheme,
		userv1alpha1.SchemeBuilder.AddToScheme,

		// Provider config APIs
		v1alpha1apis.SchemeBuilder.AddToScheme,
		v1beta1.SchemeBuilder.AddToScheme,
	)
}

// AddToSchemes may be used to add all resources defined in the project to a Scheme
var AddToSchemes runtime.SchemeBuilder

// AddToScheme adds all Resources to the Scheme
func AddToScheme(s *runtime.Scheme) error {
	return AddToSchemes.AddToScheme(s)
}