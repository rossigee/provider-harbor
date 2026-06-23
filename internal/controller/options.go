/*
Copyright 2024 Crossplane Harbor Provider.
*/

// Package controller wires up the provider's managed-resource controllers.
package controller

import (
	xpcontroller "github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"

	"github.com/rossigee/provider-harbor/internal/features"
)

// ReconcilerOptions returns the feature-gated managed.NewReconciler options that
// every controller should append to its base option set. Currently this enables
// Management Policies support (honouring a non-default spec.managementPolicies,
// e.g. [Observe,Create,Update,LateInitialize] = manage-but-never-delete) when
// the EnableBetaManagementPolicies feature flag is set.
//
// Centralising the gate here keeps every controller's Setup DRY and consistent.
func ReconcilerOptions(o xpcontroller.Options) []managed.ReconcilerOption {
	var opts []managed.ReconcilerOption
	if o.Features.Enabled(features.EnableBetaManagementPolicies) {
		opts = append(opts, managed.WithManagementPolicies())
	}
	return opts
}
