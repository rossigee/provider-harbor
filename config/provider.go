/*
Copyright 2021 Upbound Inc.
*/

package config

import (
	// (lornest) embedding schema and metadata files
	_ "embed"

	ujconfig "github.com/crossplane/upjet/pkg/config"

	"github.com/rossigee/provider-harbor/config/configauth"
	configsystem "github.com/rossigee/provider-harbor/config/configsecurity"
	configsecurity "github.com/rossigee/provider-harbor/config/configsystem"
	"github.com/rossigee/provider-harbor/config/garbagecollection"
	"github.com/rossigee/provider-harbor/config/group"
	"github.com/rossigee/provider-harbor/config/immutabletagrule"
	"github.com/rossigee/provider-harbor/config/interrogationservices"
	"github.com/rossigee/provider-harbor/config/label"
	"github.com/rossigee/provider-harbor/config/membergroup"
	"github.com/rossigee/provider-harbor/config/memberuser"
	"github.com/rossigee/provider-harbor/config/preheatinstance"
	"github.com/rossigee/provider-harbor/config/project"
	"github.com/rossigee/provider-harbor/config/purgeauditlog"
	"github.com/rossigee/provider-harbor/config/registry"
	"github.com/rossigee/provider-harbor/config/replication"
	"github.com/rossigee/provider-harbor/config/retentionpolicy"
	"github.com/rossigee/provider-harbor/config/robotaccount"
	"github.com/rossigee/provider-harbor/config/scanner"
	"github.com/rossigee/provider-harbor/config/tasks"
	"github.com/rossigee/provider-harbor/config/user"
	"github.com/rossigee/provider-harbor/config/webhook"
)

const (
	resourcePrefix = "harbor"
	modulePath     = "github.com/rossigee/provider-harbor"
)

//go:embed schema.json
var providerSchema string

//go:embed provider-metadata.yaml
var providerMetadata string

// GetProvider returns provider configuration
func GetProvider() *ujconfig.Provider {
	pc := ujconfig.NewProvider(
		[]byte(providerSchema),
		resourcePrefix,
		modulePath,
		[]byte(providerMetadata),
		ujconfig.WithRootGroup("harbor.crossplane.io"),
		ujconfig.WithIncludeList(ExternalNameConfigured()),
		ujconfig.WithFeaturesPackage("internal/features"),
		ujconfig.WithDefaultResourceOptions(
			ExternalNameConfigurations(),
		),
	)

	for _, configure := range []func(provider *ujconfig.Provider){
		// add custom config functions
		configauth.Configure,
		configsecurity.Configure,
		configsystem.Configure,
		garbagecollection.Configure,
		group.Configure,
		immutabletagrule.Configure,
		interrogationservices.Configure,
		label.Configure,
		preheatinstance.Configure,
		project.Configure,
		membergroup.Configure,
		memberuser.Configure,
		webhook.Configure,
		purgeauditlog.Configure,
		registry.Configure,
		replication.Configure,
		retentionpolicy.Configure,
		robotaccount.Configure,
		scanner.Configure,
		tasks.Configure,
		user.Configure,
	} {
		configure(pc)
	}

	pc.ConfigureResources()
	return pc
}
