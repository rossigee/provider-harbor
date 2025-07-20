package controller

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/crossplane/crossplane-runtime/pkg/controller"

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

// Setup creates all controllers with the supplied logger and adds them to
// the supplied manager.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		// Provider config
		providerconfig.Setup,

		// Native controllers
		robotaccountnative.Setup,
		usernative.Setup,

		// Terraform controllers (to be replaced)
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
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}