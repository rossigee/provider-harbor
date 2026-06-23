/*
Copyright 2024 Crossplane Harbor Provider.
*/

package controller

import (
	"testing"

	xpcontroller "github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"

	"github.com/rossigee/provider-harbor/internal/features"
)

// TestReconcilerOptions asserts the feature gate decides whether Management
// Policies support is wired into a controller's reconciler. With the feature
// enabled a managed resource carrying a non-default spec.managementPolicies
// (e.g. [Observe,Create,Update,LateInitialize] = manage-but-never-delete) is
// honoured instead of being rejected with "feature is not enabled".
func TestReconcilerOptions(t *testing.T) {
	cases := map[string]struct {
		features  *feature.Flags
		wantCount int
	}{
		"FeatureEnabled": {
			features: func() *feature.Flags {
				f := &feature.Flags{}
				f.Enable(features.EnableBetaManagementPolicies)
				return f
			}(),
			wantCount: 1,
		},
		"FeatureDisabled": {
			features:  &feature.Flags{},
			wantCount: 0,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			o := xpcontroller.Options{Features: tc.features}
			got := ReconcilerOptions(o)
			if len(got) != tc.wantCount {
				t.Errorf("ReconcilerOptions(): want %d option(s), got %d", tc.wantCount, len(got))
			}
		})
	}
}
