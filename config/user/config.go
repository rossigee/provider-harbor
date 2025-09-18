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
		r.ExternalName = config.IdentifierFromProvider

		// NOTE: For now, the password generation functionality is documented as an annotation-based approach
		// Future enhancement: Integrate with upjet's lifecycle hooks when they provide better support
		// Users can implement password generation using the provided helper functions and examples

		// Example usage documented in examples/user/generated-password-user.yaml:
		// metadata:
		//   annotations:
		//     harbor.crossplane.io/generated-password-secret: "my-harbor-password"
		//     harbor.crossplane.io/generated-password-namespace: "default"
		//     harbor.crossplane.io/generated-password-key: "password"
	})
}
