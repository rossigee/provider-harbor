package replication

import (
	"github.com/crossplane/upjet/pkg/config"
	"github.com/rossigee/provider-harbor/config/common"
)

// Configure harbor_replication resource
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("harbor_replication", func(r *config.Resource) {
		r.ShortGroup = "registry"
		r.Kind = "Replication"
		r.References["registry_id"] = config.Reference{
			TerraformName: "harbor_registry",
			Extractor:     common.RegistryIDExtractor,
		}
	})
}
