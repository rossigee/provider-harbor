package controller

import (
	ctrl "sigs.k8s.io/controller-runtime"

	xpcontroller "github.com/crossplane/crossplane-runtime/pkg/controller"
	tjcontroller "github.com/crossplane/upjet/pkg/controller"

	// Native controllers
	robotaccountnative "github.com/globallogicuki/provider-harbor/internal/controller/robotaccount/native"
	usernative "github.com/globallogicuki/provider-harbor/internal/controller/user/native"

	// Provider config (not terraform-based)
	providerconfig "github.com/globallogicuki/provider-harbor/internal/controller/providerconfig"

	// Terraform controllers (to be replaced)
	configauth "github.com/globallogicuki/provider-harbor/internal/controller/config/configauth"
	configsecurity "github.com/globallogicuki/provider-harbor/internal/controller/config/configsecurity"
	configsystem "github.com/globallogicuki/provider-harbor/internal/controller/config/configsystem"
	garbagecollection "github.com/globallogicuki/provider-harbor/internal/controller/garbagecollection/garbagecollection"
	group "github.com/globallogicuki/provider-harbor/internal/controller/group/group"
	interrogationservices "github.com/globallogicuki/provider-harbor/internal/controller/interrogationservices/interrogationservices"
	label "github.com/globallogicuki/provider-harbor/internal/controller/label/label"
	preheatinstance "github.com/globallogicuki/provider-harbor/internal/controller/preheatinstance/preheatinstance"
	immutabletagrule "github.com/globallogicuki/provider-harbor/internal/controller/project/immutabletagrule"
	membergroup "github.com/globallogicuki/provider-harbor/internal/controller/project/membergroup"
	memberuser "github.com/globallogicuki/provider-harbor/internal/controller/project/memberuser"
	project "github.com/globallogicuki/provider-harbor/internal/controller/project/project"
	retentionpolicy "github.com/globallogicuki/provider-harbor/internal/controller/project/retentionpolicy"
	webhook "github.com/globallogicuki/provider-harbor/internal/controller/project/webhook"
	purgeauditlog "github.com/globallogicuki/provider-harbor/internal/controller/purgeauditlog/purgeauditlog"
	registry "github.com/globallogicuki/provider-harbor/internal/controller/registry/registry"
	replication "github.com/globallogicuki/provider-harbor/internal/controller/registry/replication"
	task "github.com/globallogicuki/provider-harbor/internal/controller/tasks/task"
)

// SetupMixed sets up both native and terraform controllers
func SetupMixed(mgr ctrl.Manager, nativeOpts xpcontroller.Options, terraformOpts tjcontroller.Options) error {
	// Setup native controllers
	for _, setup := range []func(ctrl.Manager, xpcontroller.Options) error{
		robotaccountnative.Setup,
		usernative.Setup,
	} {
		if err := setup(mgr, nativeOpts); err != nil {
			return err
		}
	}

	// Setup terraform-based controllers 
	for _, setup := range []func(ctrl.Manager, tjcontroller.Options) error{
		providerconfig.Setup,
		configauth.Setup,
		configsecurity.Setup,
		configsystem.Setup,
		garbagecollection.Setup,
		group.Setup,
		interrogationservices.Setup,
		label.Setup,
		preheatinstance.Setup,
		immutabletagrule.Setup,
		membergroup.Setup,
		memberuser.Setup,
		project.Setup,
		retentionpolicy.Setup,
		webhook.Setup,
		purgeauditlog.Setup,
		registry.Setup,
		replication.Setup,
		task.Setup,
	} {
		if err := setup(mgr, terraformOpts); err != nil {
			return err
		}
	}
	return nil
}