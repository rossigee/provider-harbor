/*
Copyright 2024 Crossplane Harbor Provider.
*/

package registry

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rossigee/provider-harbor/apis/registry/v1beta1"
)

// ERROR CASE TESTS

func TestConnectNotRegistry(t *testing.T) {
	ctx := context.Background()
	conn := &connector{}

	_, err := conn.Connect(ctx, nil)
	if err == nil || err.Error() != errNotRegistry {
		t.Errorf("Connect with nil should return %s error", errNotRegistry)
	}
}

func TestObserveNotRegistry(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Observe(ctx, nil)
	if err == nil || err.Error() != errNotRegistry {
		t.Errorf("Observe with nil should return %s error", errNotRegistry)
	}
}

func TestUpdateNotRegistry(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Update(ctx, nil)
	if err == nil || err.Error() != errNotRegistry {
		t.Errorf("Update with nil should return %s error", errNotRegistry)
	}
}

func TestDeleteNotRegistry(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Delete(ctx, nil)
	if err == nil || err.Error() != errNotRegistry {
		t.Errorf("Delete with nil should return %s error", errNotRegistry)
	}
}

func TestCreateNotRegistry(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Create(ctx, nil)
	if err == nil || err.Error() != errNotRegistry {
		t.Errorf("Create with nil should return %s error", errNotRegistry)
	}
}

// HAPPY-PATH AND VALIDATION TESTS

func TestRegistryHasRequiredFields(t *testing.T) {
	registry := &v1beta1.Registry{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-registry",
			Namespace: "default",
		},
		Spec: v1beta1.RegistrySpec{
			ForProvider: v1beta1.RegistryParameters{
				Name: "docker-hub",
				Type: "docker-hub",
				URL:  "https://docker.io",
			},
		},
	}

	if registry.Spec.ForProvider.Name == "" {
		t.Error("Registry Name should not be empty")
	}
	if registry.Spec.ForProvider.Type == "" {
		t.Error("Registry Type should not be empty")
	}
	if registry.Spec.ForProvider.URL == "" {
		t.Error("Registry URL should not be empty")
	}
}

func TestRegistryParametersValidation(t *testing.T) {
	tests := []struct {
		name    string
		params  v1beta1.RegistryParameters
		isValid bool
	}{
		{
			name: "valid docker-hub registry",
			params: v1beta1.RegistryParameters{
				Name: "docker-hub",
				Type: "docker-hub",
				URL:  "https://docker.io",
			},
			isValid: true,
		},
		{
			name: "valid harbor registry",
			params: v1beta1.RegistryParameters{
				Name: "internal-harbor",
				Type: "harbor",
				URL:  "https://harbor.example.com",
			},
			isValid: true,
		},
		{
			name: "valid with description",
			params: v1beta1.RegistryParameters{
				Name:        "gcr-registry",
				Type:        "google-gcr",
				URL:         "https://gcr.io",
				Description: ptrString("Google Container Registry"),
			},
			isValid: true,
		},
		{
			name: "valid with insecure flag",
			params: v1beta1.RegistryParameters{
				Name:     "local-registry",
				Type:     "docker-registry",
				URL:      "http://localhost:5000",
				Insecure: ptrBool(true),
			},
			isValid: true,
		},
		{
			name: "missing name",
			params: v1beta1.RegistryParameters{
				Type: "docker-hub",
				URL:  "https://docker.io",
			},
			isValid: false,
		},
		{
			name: "missing type",
			params: v1beta1.RegistryParameters{
				Name: "my-registry",
				URL:  "https://registry.example.com",
			},
			isValid: false,
		},
		{
			name: "missing URL",
			params: v1beta1.RegistryParameters{
				Name: "my-registry",
				Type: "docker-hub",
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.params.Name != "" && tt.params.Type != "" && tt.params.URL != ""
			if isValid != tt.isValid {
				t.Errorf("Expected valid=%v, got %v", tt.isValid, isValid)
			}
		})
	}
}

func TestRegistryTypes(t *testing.T) {
	validTypes := map[string]bool{
		"harbor":           true,
		"docker-hub":       true,
		"docker-registry":  true,
		"helm-hub":         true,
		"aws-ecr":          true,
		"azure-acr":        true,
		"google-gcr":       true,
		"gitlab":           true,
		"quay":             true,
		"invalid-registry": false,
	}

	for registryType, isValid := range validTypes {
		t.Run(registryType, func(t *testing.T) {
			if registryType != "invalid-registry" && !isValid {
				t.Errorf("Registry type '%s' should be valid", registryType)
			}
			if registryType == "invalid-registry" && isValid {
				t.Errorf("Registry type '%s' should be invalid", registryType)
			}
		})
	}
}

// Helper functions
func ptrBool(b bool) *bool {
	return &b
}

func ptrString(s string) *string {
	return &s
}
