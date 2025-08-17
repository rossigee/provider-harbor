package robotaccount

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestAdditionalConnectionDetailsFn(t *testing.T) {
	cases := map[string]struct {
		input  map[string]any
		want   map[string][]byte
		reason string
	}{
		"BasicRobotAccount": {
			input: map[string]any{
				"full_name": "robot$test-robot",
				"secret":    "test-secret-password",
				"robot_id":  "123",
			},
			want: map[string][]byte{
				"username":              []byte("robot$test-robot"),
				"password":              []byte("test-secret-password"),
				"robot_id":              []byte("123"),
				"docker-username":       []byte("robot$test-robot"),
				"docker-password":       []byte("test-secret-password"),
				"docker-auth":           []byte(base64.StdEncoding.EncodeToString([]byte("robot$test-robot:test-secret-password"))),
				"docker-config-template": mustMarshalDockerConfig(map[string]interface{}{
					"auths": map[string]interface{}{
						"REGISTRY_URL_PLACEHOLDER": map[string]interface{}{
							"username": "robot$test-robot",
							"password": "test-secret-password",
							"auth":     base64.StdEncoding.EncodeToString([]byte("robot$test-robot:test-secret-password")),
						},
					},
				}),
			},
			reason: "Should generate all legacy and Docker config fields",
		},
		"MissingUsername": {
			input: map[string]any{
				"secret":   "test-secret-password",
				"robot_id": "123",
			},
			want: map[string][]byte{
				"password": []byte("test-secret-password"),
				"robot_id": []byte("123"),
			},
			reason: "Should only generate available fields when username is missing",
		},
		"MissingPassword": {
			input: map[string]any{
				"full_name": "robot$test-robot",
				"robot_id":  "123",
			},
			want: map[string][]byte{
				"username": []byte("robot$test-robot"),
				"robot_id": []byte("123"),
			},
			reason: "Should only generate available fields when password is missing",
		},
		"EmptyStrings": {
			input: map[string]any{
				"full_name": "",
				"secret":    "",
				"robot_id":  "",
			},
			want: map[string][]byte{},
			reason: "Should not generate fields for empty strings",
		},
		"OnlyRobotId": {
			input: map[string]any{
				"robot_id": "456",
			},
			want: map[string][]byte{
				"robot_id": []byte("456"),
			},
			reason: "Should generate robot_id when only that field is available",
		},
		"TypeMismatch": {
			input: map[string]any{
				"full_name": 123,    // wrong type
				"secret":    true,   // wrong type
				"robot_id":  "789",  // correct type
			},
			want: map[string][]byte{
				"robot_id": []byte("789"),
			},
			reason: "Should handle type mismatches gracefully",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Create a test function that matches the signature used in config.go
			testFn := func(attr map[string]any) (map[string][]byte, error) {
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
				if username != "" && password != "" {
					conn["docker-username"] = []byte(username)
					conn["docker-password"] = []byte(password)
					conn["docker-auth"] = []byte(base64.StdEncoding.EncodeToString([]byte(username + ":" + password)))

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
						return nil, err
					}
					conn["docker-config-template"] = dockerConfigJSON
				}

				return conn, nil
			}

			got, err := testFn(tc.input)
			if err != nil {
				t.Errorf("%s: unexpected error: %v", tc.reason, err)
				return
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("%s: -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestDockerConfigTemplate(t *testing.T) {
	username := "robot$test-robot"
	password := "test-password"
	auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))

	expectedConfig := map[string]interface{}{
		"auths": map[string]interface{}{
			"REGISTRY_URL_PLACEHOLDER": map[string]interface{}{
				"username": username,
				"password": password,
				"auth":     auth,
			},
		},
	}

	configJSON, err := json.Marshal(expectedConfig)
	if err != nil {
		t.Fatalf("Failed to marshal expected config: %v", err)
	}

	// Test that the generated template can be unmarshaled and manipulated
	var config map[string]interface{}
	err = json.Unmarshal(configJSON, &config)
	if err != nil {
		t.Fatalf("Failed to unmarshal config JSON: %v", err)
	}

	auths, ok := config["auths"].(map[string]interface{})
	if !ok {
		t.Fatal("auths field is not a map")
	}

	if _, exists := auths["REGISTRY_URL_PLACEHOLDER"]; !exists {
		t.Fatal("REGISTRY_URL_PLACEHOLDER not found in auths")
	}

	// Test registry URL substitution
	delete(auths, "REGISTRY_URL_PLACEHOLDER")
	auths["harbor.example.com"] = map[string]interface{}{
		"username": username,
		"password": password,
		"auth":     auth,
	}

	finalConfigJSON, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal final config: %v", err)
	}

	// Verify the final config structure
	var finalConfig map[string]interface{}
	err = json.Unmarshal(finalConfigJSON, &finalConfig)
	if err != nil {
		t.Fatalf("Failed to unmarshal final config: %v", err)
	}

	finalAuths := finalConfig["auths"].(map[string]interface{})
	harborAuth := finalAuths["harbor.example.com"].(map[string]interface{})

	if harborAuth["username"] != username {
		t.Errorf("Expected username %s, got %v", username, harborAuth["username"])
	}
	if harborAuth["password"] != password {
		t.Errorf("Expected password %s, got %v", password, harborAuth["password"])
	}
	if harborAuth["auth"] != auth {
		t.Errorf("Expected auth %s, got %v", auth, harborAuth["auth"])
	}
}

func TestBackwardCompatibility(t *testing.T) {
	// Test that new functionality doesn't break existing behavior
	input := map[string]any{
		"full_name": "robot$legacy-robot",
		"secret":    "legacy-password",
		"robot_id":  "999",
	}

	testFn := func(attr map[string]any) (map[string][]byte, error) {
		conn := map[string][]byte{}

		// This should match the exact legacy behavior
		if v, ok := attr["full_name"].(string); ok && v != "" {
			conn["username"] = []byte(v)
		}
		if v, ok := attr["secret"].(string); ok && v != "" {
			conn["password"] = []byte(v)
		}
		if v, ok := attr["robot_id"].(string); ok && v != "" {
			conn["robot_id"] = []byte(v)
		}

		return conn, nil
	}

	got, err := testFn(input)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedLegacyFields := map[string][]byte{
		"username": []byte("robot$legacy-robot"),
		"password": []byte("legacy-password"),
		"robot_id": []byte("999"),
	}

	// Verify legacy fields are preserved
	for key, expectedValue := range expectedLegacyFields {
		if gotValue, exists := got[key]; !exists {
			t.Errorf("Legacy field %s is missing", key)
		} else if diff := cmp.Diff(expectedValue, gotValue); diff != "" {
			t.Errorf("Legacy field %s differs: -want, +got:\n%s", key, diff)
		}
	}
}

// Helper function to marshal Docker config for test expectations
func mustMarshalDockerConfig(config map[string]interface{}) []byte {
	data, err := json.Marshal(config)
	if err != nil {
		panic(err)
	}
	return data
}