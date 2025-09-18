package robotaccount

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/crossplane/upjet/pkg/config"
)

// Configure harbor_robot_account resource
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("harbor_robot_account", func(r *config.Resource) {
		r.ShortGroup = "robotaccount"
		r.Kind = "RobotAccount"
		r.References["permissions.namespace"] = config.Reference{
			TerraformName: "harbor_project",
			Extractor:     `github.com/crossplane/upjet/pkg/resource.ExtractParamPath("name",true)`,
		}

		// Configure connection details to support both legacy and Docker config JSON formats
		r.Sensitive.AdditionalConnectionDetailsFn = func(attr map[string]any) (map[string][]byte, error) {
			conn := map[string][]byte{}

			// Always provide individual fields (backward compatibility)
			username := ""
			password := ""

			if v, ok := attr["full_name"].(string); ok && v != "" {
				conn["username"] = []byte(v)
				username = v
			}
			if v, ok := attr["secret"].(string); ok && v != "" {
				conn["password"] = []byte(v)
				password = v
			}
			if v, ok := attr["robot_id"].(string); ok && v != "" {
				conn["robot_id"] = []byte(v)
			}

			// Generate Docker config JSON if both username and password are available
			// The consumer can use this in their publishConnectionDetailsTo metadata
			// by setting the appropriate secret type and labels
			if username != "" && password != "" {
				// This creates a generic docker config that can be customized with registry URL
				// in the publishConnectionDetailsTo metadata
				conn["docker-username"] = []byte(username)
				conn["docker-password"] = []byte(password)
				conn["docker-auth"] = []byte(base64.StdEncoding.EncodeToString([]byte(username + ":" + password)))

				// Generate a basic docker config template - consumers can customize the registry URL
				dockerConfig := map[string]interface{}{
					"auths": map[string]interface{}{
						"REGISTRY_URL_PLACEHOLDER": map[string]interface{}{
							"username": username,
							"password": password,
							"auth":     base64.StdEncoding.EncodeToString([]byte(username + ":" + password)),
						},
					},
				}

				dockerConfigJSON, err := json.Marshal(dockerConfig)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal docker config JSON template: %w", err)
				}
				conn["docker-config-template"] = dockerConfigJSON
			}

			return conn, nil
		}
	})
}
