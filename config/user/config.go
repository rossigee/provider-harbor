package user

import (
	"github.com/crossplane/upjet/pkg/config"
)

// Configure harbor_user resource
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("harbor_user", func(r *config.Resource) {
		r.ShortGroup = "user"
		r.Kind = "User"
		
		// Use the standard IdentifierFromProvider approach for Harbor users
		// This lets the Terraform provider handle ID management properly
		// The external name will be the username, but the provider manages numeric IDs internally
	})
}
