package scanner

import (
	"github.com/crossplane/upjet/pkg/config"
)

// Configure harbor_scanner_registration resource
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("harbor_scanner_registration", func(r *config.Resource) {
		r.ShortGroup = "scanner"
		r.Kind = "ScannerRegistration"
	})
}