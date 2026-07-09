/*
Copyright 2024 Crossplane Harbor Provider.
*/

package registry

import (
	"context"
	"errors"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/rossigee/provider-harbor/apis/registry/v1beta1"
	harborclients "github.com/rossigee/provider-harbor/internal/clients"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
	"time"
)

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

func TestCreateNotRegistry(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Create(ctx, nil)
	if err == nil || err.Error() != errNotRegistry {
		t.Errorf("Create with nil should return %s error", errNotRegistry)
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

func TestObserveRegistryNotFound(t *testing.T) {
	ctx := context.Background()
	registry := &v1beta1.Registry{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-registry",
		},
		Spec: v1beta1.RegistrySpec{
			ForProvider: v1beta1.RegistryParameters{
				Name: "docker-hub",
				Type: "docker-hub",
				URL:  "https://docker.io",
			},
		},
	}

	ext := &external{
		service: &mockRegistryClient{
			getRegistryFunc: func(ctx context.Context, registryName string) (*harborclients.RegistryStatus, error) {
				return nil, errors.New("not found")
			},
		},
	}

	obs, err := ext.Observe(ctx, registry)
	if err != nil {
		t.Errorf("Observe should not fail, got %v", err)
	}
	if obs.ResourceExists {
		t.Error("ResourceExists should be false when registry not found")
	}
}

func TestObserveRegistryExists(t *testing.T) {
	ctx := context.Background()
	desc := "Test description"
	registry := &v1beta1.Registry{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-registry",
		},
		Spec: v1beta1.RegistrySpec{
			ForProvider: v1beta1.RegistryParameters{
				Name:        "docker-hub",
				Type:        "docker-hub",
				URL:         "https://docker.io",
				Description: &desc,
			},
		},
	}

	ext := &external{
		service: &mockRegistryClient{
			getRegistryFunc: func(ctx context.Context, registryName string) (*harborclients.RegistryStatus, error) {
				return &harborclients.RegistryStatus{
					Name:        "docker-hub",
					Type:        "docker-hub",
					URL:         "https://docker.io",
					Description: &desc,
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}, nil
			},
		},
	}

	obs, err := ext.Observe(ctx, registry)
	if err != nil {
		t.Errorf("Observe should not fail, got %v", err)
	}
	if !obs.ResourceExists {
		t.Error("ResourceExists should be true")
	}
	if !obs.ResourceUpToDate {
		t.Error("ResourceUpToDate should be true")
	}
}

func TestObserveRegistryNotUpToDate(t *testing.T) {
	ctx := context.Background()
	newDesc := "New description"
	registry := &v1beta1.Registry{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-registry",
		},
		Spec: v1beta1.RegistrySpec{
			ForProvider: v1beta1.RegistryParameters{
				Name:        "docker-hub",
				Type:        "docker-hub",
				URL:         "https://docker.io",
				Description: &newDesc,
			},
		},
	}

	oldDesc := "Old description"
	ext := &external{
		service: &mockRegistryClient{
			getRegistryFunc: func(ctx context.Context, registryName string) (*harborclients.RegistryStatus, error) {
				return &harborclients.RegistryStatus{
					Name:        "docker-hub",
					Type:        "docker-hub",
					URL:         "https://docker.io",
					Description: &oldDesc,
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}, nil
			},
		},
	}

	obs, err := ext.Observe(ctx, registry)
	if err != nil {
		t.Errorf("Observe should not fail, got %v", err)
	}
	if !obs.ResourceExists {
		t.Error("ResourceExists should be true")
	}
	if obs.ResourceUpToDate {
		t.Error("ResourceUpToDate should be false when descriptions differ")
	}
}

func TestCreateRegistrySuccess(t *testing.T) {
	ctx := context.Background()
	registry := &v1beta1.Registry{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-registry",
		},
		Spec: v1beta1.RegistrySpec{
			ForProvider: v1beta1.RegistryParameters{
				Name: "docker-hub",
				Type: "docker-hub",
				URL:  "https://docker.io",
			},
		},
	}

	ext := &external{
		service: &mockRegistryClient{
			createRegistryFunc: func(ctx context.Context, spec *harborclients.RegistrySpec) (*harborclients.RegistryStatus, error) {
				return &harborclients.RegistryStatus{
					Name:      spec.Name,
					Type:      spec.Type,
					URL:       spec.URL,
					CreatedAt: time.Now(),
				}, nil
			},
		},
	}

	_, err := ext.Create(ctx, registry)
	if err != nil {
		t.Errorf("Create should not fail, got %v", err)
	}
}

func TestCreateRegistryError(t *testing.T) {
	ctx := context.Background()
	registry := &v1beta1.Registry{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-registry",
		},
		Spec: v1beta1.RegistrySpec{
			ForProvider: v1beta1.RegistryParameters{
				Name: "docker-hub",
				Type: "docker-hub",
				URL:  "https://docker.io",
			},
		},
	}

	ext := &external{
		service: &mockRegistryClient{
			createRegistryFunc: func(ctx context.Context, spec *harborclients.RegistrySpec) (*harborclients.RegistryStatus, error) {
				return nil, errors.New("create failed")
			},
		},
	}

	_, err := ext.Create(ctx, registry)
	if err == nil {
		t.Error("Create should fail when client fails")
	}
}

func TestUpdateRegistrySuccess(t *testing.T) {
	ctx := context.Background()
	registry := &v1beta1.Registry{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-registry",
		},
		Spec: v1beta1.RegistrySpec{
			ForProvider: v1beta1.RegistryParameters{
				Name: "docker-hub",
				Type: "docker-hub",
				URL:  "https://docker.io",
			},
		},
	}

	ext := &external{
		service: &mockRegistryClient{
			updateRegistryFunc: func(ctx context.Context, registryName string, spec *harborclients.RegistrySpec) (*harborclients.RegistryStatus, error) {
				return &harborclients.RegistryStatus{
					Name:      spec.Name,
					Type:      spec.Type,
					URL:       spec.URL,
					UpdatedAt: time.Now(),
				}, nil
			},
		},
	}

	_, err := ext.Update(ctx, registry)
	if err != nil {
		t.Errorf("Update should not fail, got %v", err)
	}
}

func TestDeleteRegistrySuccess(t *testing.T) {
	ctx := context.Background()
	registry := &v1beta1.Registry{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-registry",
		},
		Spec: v1beta1.RegistrySpec{
			ForProvider: v1beta1.RegistryParameters{
				Name: "docker-hub",
				Type: "docker-hub",
				URL:  "https://docker.io",
			},
		},
	}

	ext := &external{
		service: &mockRegistryClient{
			deleteRegistryFunc: func(ctx context.Context, registryName string) error {
				return nil
			},
		},
	}

	_, err := ext.Delete(ctx, registry)
	if err != nil {
		t.Errorf("Delete should not fail, got %v", err)
	}
}

func TestDeleteRegistryError(t *testing.T) {
	ctx := context.Background()
	registry := &v1beta1.Registry{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-registry",
		},
		Spec: v1beta1.RegistrySpec{
			ForProvider: v1beta1.RegistryParameters{
				Name: "docker-hub",
				Type: "docker-hub",
				URL:  "https://docker.io",
			},
		},
	}

	ext := &external{
		service: &mockRegistryClient{
			deleteRegistryFunc: func(ctx context.Context, registryName string) error {
				return errors.New("delete failed")
			},
		},
	}

	_, err := ext.Delete(ctx, registry)
	if err == nil {
		t.Error("Delete should fail when client fails")
	}
}

func TestHelperFunctions(t *testing.T) {
	intVal := int64(42)
	result := getInt64Ptr(intVal)
	if result == nil || *result != intVal {
		t.Errorf("getInt64Ptr failed")
	}

	strVal := "test"
	resultStr := getStringPtr(strVal)
	if resultStr == nil || *resultStr != strVal {
		t.Errorf("getStringPtr failed")
	}
}

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
		t.Error("Name should not be empty")
	}
	if registry.Spec.ForProvider.Type == "" {
		t.Error("Type should not be empty")
	}
	if registry.Spec.ForProvider.URL == "" {
		t.Error("URL should not be empty")
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

func TestRegistryStatus(t *testing.T) {
	registry := &v1beta1.Registry{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-registry",
		},
		Spec: v1beta1.RegistrySpec{
			ForProvider: v1beta1.RegistryParameters{
				Name: "docker-hub",
				Type: "docker-hub",
				URL:  "https://docker.io",
			},
		},
		Status: v1beta1.RegistryStatus{
			AtProvider: v1beta1.RegistryObservation{
				ID: ptrInt64(123),
			},
		},
	}

	if registry.Status.AtProvider.ID == nil {
		t.Error("Status ID should be populated")
	}
	if *registry.Status.AtProvider.ID != 123 {
		t.Errorf("Status ID should be 123, got %d", *registry.Status.AtProvider.ID)
	}
}

func TestRegistryCredential(t *testing.T) {
	credType := "basic"
	accessKey := "my-key"
	registry := &v1beta1.Registry{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-registry",
		},
		Spec: v1beta1.RegistrySpec{
			ForProvider: v1beta1.RegistryParameters{
				Name: "docker-hub",
				Type: "docker-hub",
				URL:  "https://docker.io",
				Credential: &v1beta1.RegistryCredential{
					Type:      &credType,
					AccessKey: &accessKey,
				},
			},
		},
	}

	if registry.Spec.ForProvider.Credential == nil {
		t.Error("Credential should not be nil")
	}
	if *registry.Spec.ForProvider.Credential.AccessKey != "my-key" {
		t.Error("AccessKey should be set")
	}
}

func TestCreateRegistryWithEmptyURL(t *testing.T) {
	ctx := context.Background()
	registry := &v1beta1.Registry{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-registry",
		},
		Spec: v1beta1.RegistrySpec{
			ForProvider: v1beta1.RegistryParameters{
				Name: "my-registry",
				Type: "docker-hub",
				URL:  "",
			},
		},
	}

	ext := &external{
		service: &mockRegistryClient{
			createRegistryFunc: func(ctx context.Context, spec *harborclients.RegistrySpec) (*harborclients.RegistryStatus, error) {
				if spec.URL == "" {
					return nil, errors.New("URL is required")
				}
				return nil, nil
			},
		},
	}

	_, err := ext.Create(ctx, registry)
	if err == nil {
		t.Error("Create should fail when URL is empty")
	}
}

func TestCreateRegistryWithCredentials(t *testing.T) {
	ctx := context.Background()
	credType := "basic"
	accessKey := "my-access-key"

	registry := &v1beta1.Registry{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-registry",
		},
		Spec: v1beta1.RegistrySpec{
			ForProvider: v1beta1.RegistryParameters{
				Name: "private-registry",
				Type: "harbor",
				URL:  "https://harbor.private.com",
				Credential: &v1beta1.RegistryCredential{
					Type:      &credType,
					AccessKey: &accessKey,
				},
			},
		},
	}

	ext := &external{
		service: &mockRegistryClient{
			createRegistryFunc: func(ctx context.Context, spec *harborclients.RegistrySpec) (*harborclients.RegistryStatus, error) {
				if spec.Credential == nil {
					return nil, errors.New("credential is required")
				}
				return &harborclients.RegistryStatus{
					Name:      spec.Name,
					Type:      spec.Type,
					URL:       spec.URL,
					CreatedAt: time.Now(),
				}, nil
			},
		},
	}

	_, err := ext.Create(ctx, registry)
	if err != nil {
		t.Errorf("Create with credentials should not fail, got %v", err)
	}
}

func TestUpdateRegistryWithEmptyDescription(t *testing.T) {
	ctx := context.Background()
	emptyDesc := ""
	registry := &v1beta1.Registry{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-registry",
		},
		Spec: v1beta1.RegistrySpec{
			ForProvider: v1beta1.RegistryParameters{
				Name:        "docker-hub",
				Type:        "docker-hub",
				URL:         "https://docker.io",
				Description: &emptyDesc,
			},
		},
	}

	ext := &external{
		service: &mockRegistryClient{
			updateRegistryFunc: func(ctx context.Context, registryName string, spec *harborclients.RegistrySpec) (*harborclients.RegistryStatus, error) {
				return &harborclients.RegistryStatus{
					Name:        spec.Name,
					Type:        spec.Type,
					URL:         spec.URL,
					Description: spec.Description,
					UpdatedAt:   time.Now(),
				}, nil
			},
		},
	}

	_, err := ext.Update(ctx, registry)
	if err != nil {
		t.Errorf("Update with empty description should not fail, got %v", err)
	}
}

func TestObserveRegistryWithNilDescription(t *testing.T) {
	ctx := context.Background()
	registry := &v1beta1.Registry{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-registry",
		},
		Spec: v1beta1.RegistrySpec{
			ForProvider: v1beta1.RegistryParameters{
				Name: "docker-hub",
				Type: "docker-hub",
				URL:  "https://docker.io",
			},
		},
	}

	ext := &external{
		service: &mockRegistryClient{
			getRegistryFunc: func(ctx context.Context, registryName string) (*harborclients.RegistryStatus, error) {
				return &harborclients.RegistryStatus{
					Name:        "docker-hub",
					Type:        "docker-hub",
					URL:         "https://docker.io",
					Description: nil,
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}, nil
			},
		},
	}

	obs, err := ext.Observe(ctx, registry)
	if err != nil {
		t.Errorf("Observe with nil description should not fail, got %v", err)
	}
	if !obs.ResourceExists {
		t.Error("ResourceExists should be true")
	}
	if !obs.ResourceUpToDate {
		t.Error("ResourceUpToDate should be true when description is nil in spec")
	}
}

func TestGetInt64PtrHelper(t *testing.T) {
	result := getInt64Ptr(1000)
	if result == nil || *result != 1000 {
		t.Error("getInt64Ptr should work correctly")
	}

	resultZero := getInt64Ptr(0)
	if resultZero == nil || *resultZero != 0 {
		t.Error("getInt64Ptr with 0 should work correctly")
	}
}

func TestGetStringPtrHelper(t *testing.T) {
	result := getStringPtr("test-value")
	if result == nil || *result != "test-value" {
		t.Error("getStringPtr should work correctly")
	}

	resultEmpty := getStringPtr("")
	if resultEmpty == nil || *resultEmpty != "" {
		t.Error("getStringPtr with empty should work correctly")
	}
}

func TestCreateRegistryWithInsecureFlag(t *testing.T) {
	ctx := context.Background()
	insecure := true

	registry := &v1beta1.Registry{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-registry",
		},
		Spec: v1beta1.RegistrySpec{
			ForProvider: v1beta1.RegistryParameters{
				Name:     "test-registry",
				Type:     "harbor",
				URL:      "https://harbor.local",
				Insecure: &insecure,
			},
		},
	}

	ext := &external{
		service: &mockRegistryClient{
			createRegistryFunc: func(ctx context.Context, spec *harborclients.RegistrySpec) (*harborclients.RegistryStatus, error) {
				return &harborclients.RegistryStatus{
					Name:      spec.Name,
					Type:      spec.Type,
					URL:       spec.URL,
					CreatedAt: time.Now(),
				}, nil
			},
		},
	}

	_, err := ext.Create(ctx, registry)
	if err != nil {
		t.Errorf("Create with insecure flag should not fail, got %v", err)
	}
}

func TestObserveRegistryStatusPopulation(t *testing.T) {
	ctx := context.Background()
	registry := &v1beta1.Registry{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-registry",
		},
		Spec: v1beta1.RegistrySpec{
			ForProvider: v1beta1.RegistryParameters{
				Name: "docker-hub",
				Type: "docker-hub",
				URL:  "https://docker.io",
			},
		},
	}

	ext := &external{
		service: &mockRegistryClient{
			getRegistryFunc: func(ctx context.Context, registryName string) (*harborclients.RegistryStatus, error) {
				return &harborclients.RegistryStatus{
					Name:      "docker-hub",
					Type:      "docker-hub",
					URL:       "https://docker.io",
					CreatedAt: time.Now().Add(-24 * time.Hour),
					UpdatedAt: time.Now(),
				}, nil
			},
		},
	}

	obs, err := ext.Observe(ctx, registry)
	if err != nil {
		t.Errorf("Observe should not fail, got %v", err)
	}
	if obs.ConnectionDetails == nil {
		t.Error("ConnectionDetails should be populated")
	}
}

func TestUpdateRegistryWithNilCredential(t *testing.T) {
	ctx := context.Background()
	registry := &v1beta1.Registry{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-registry",
		},
		Spec: v1beta1.RegistrySpec{
			ForProvider: v1beta1.RegistryParameters{
				Name:       "docker-hub",
				Type:       "docker-hub",
				URL:        "https://docker.io",
				Credential: nil,
			},
		},
	}

	ext := &external{
		service: &mockRegistryClient{
			updateRegistryFunc: func(ctx context.Context, registryName string, spec *harborclients.RegistrySpec) (*harborclients.RegistryStatus, error) {
				return &harborclients.RegistryStatus{
					Name:      spec.Name,
					Type:      spec.Type,
					URL:       spec.URL,
					UpdatedAt: time.Now(),
				}, nil
			},
		},
	}

	_, err := ext.Update(ctx, registry)
	if err != nil {
		t.Errorf("Update with nil credential should not fail, got %v", err)
	}
}

func TestDisconnectRegistry(t *testing.T) {
	ctx := context.Background()
	ext := &external{
		service: &mockRegistryClient{},
	}

	err := ext.Disconnect(ctx)
	if err != nil {
		t.Errorf("Disconnect should not fail, got %v", err)
	}
}

func TestObserveRegistryConnectionDetails(t *testing.T) {
	ctx := context.Background()
	registry := &v1beta1.Registry{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-registry",
		},
		Spec: v1beta1.RegistrySpec{
			ForProvider: v1beta1.RegistryParameters{
				Name: "docker-hub",
				Type: "docker-hub",
				URL:  "https://docker.io",
			},
		},
	}

	ext := &external{
		service: &mockRegistryClient{
			getRegistryFunc: func(ctx context.Context, registryName string) (*harborclients.RegistryStatus, error) {
				return &harborclients.RegistryStatus{
					Name:      "docker-hub",
					Type:      "docker-hub",
					URL:       "https://docker.io",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}, nil
			},
		},
	}

	obs, err := ext.Observe(ctx, registry)
	if err != nil {
		t.Errorf("Observe should not fail, got %v", err)
	}

	if obs.ConnectionDetails == nil {
		t.Error("ConnectionDetails should not be nil")
	}
}

func TestCreateRegistryConnectionDetails(t *testing.T) {
	ctx := context.Background()
	registry := &v1beta1.Registry{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-registry",
		},
		Spec: v1beta1.RegistrySpec{
			ForProvider: v1beta1.RegistryParameters{
				Name: "docker-hub",
				Type: "docker-hub",
				URL:  "https://docker.io",
			},
		},
	}

	ext := &external{
		service: &mockRegistryClient{
			createRegistryFunc: func(ctx context.Context, spec *harborclients.RegistrySpec) (*harborclients.RegistryStatus, error) {
				return &harborclients.RegistryStatus{
					Name:      spec.Name,
					Type:      spec.Type,
					URL:       spec.URL,
					CreatedAt: time.Now(),
				}, nil
			},
		},
	}

	creation, err := ext.Create(ctx, registry)
	if err != nil {
		t.Errorf("Create should not fail, got %v", err)
	}

	if creation.ConnectionDetails == nil {
		t.Error("ConnectionDetails should not be nil")
	}
}

func TestUpdateRegistryConnectionDetails(t *testing.T) {
	ctx := context.Background()
	registry := &v1beta1.Registry{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-registry",
		},
		Spec: v1beta1.RegistrySpec{
			ForProvider: v1beta1.RegistryParameters{
				Name: "docker-hub",
				Type: "docker-hub",
				URL:  "https://docker.io",
			},
		},
	}

	ext := &external{
		service: &mockRegistryClient{
			updateRegistryFunc: func(ctx context.Context, registryName string, spec *harborclients.RegistrySpec) (*harborclients.RegistryStatus, error) {
				return &harborclients.RegistryStatus{
					Name:      spec.Name,
					Type:      spec.Type,
					URL:       spec.URL,
					UpdatedAt: time.Now(),
				}, nil
			},
		},
	}

	update, err := ext.Update(ctx, registry)
	if err != nil {
		t.Errorf("Update should not fail, got %v", err)
	}

	if update.ConnectionDetails == nil {
		t.Error("ConnectionDetails should not be nil")
	}
}

func TestObserveRegistryTypeValidation(t *testing.T) {
	registryTypes := []string{"docker-hub", "harbor", "artifactory", "azure", "aws-ecr", "gcp-gcr"}

	for _, regType := range registryTypes {
		t.Run(regType, func(t *testing.T) {
			ctx := context.Background()
			registry := &v1beta1.Registry{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-registry",
				},
				Spec: v1beta1.RegistrySpec{
					ForProvider: v1beta1.RegistryParameters{
						Name: regType,
						Type: regType,
						URL:  "https://registry.example.com",
					},
				},
			}

			ext := &external{
				service: &mockRegistryClient{
					getRegistryFunc: func(ctx context.Context, registryName string) (*harborclients.RegistryStatus, error) {
						return &harborclients.RegistryStatus{
							Name:      regType,
							Type:      regType,
							URL:       "https://registry.example.com",
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
						}, nil
					},
				},
			}

			obs, err := ext.Observe(ctx, registry)
			if err != nil {
				t.Errorf("Observe should not fail, got %v", err)
			}
			if !obs.ResourceExists {
				t.Error("ResourceExists should be true")
			}
		})
	}
}

func TestConnectRegistrySuccess(t *testing.T) {
	ctx := context.Background()
	conn := &connector{
		kube: nil,
		newServiceFn: func(ctx context.Context, kube client.Client, mg resource.Managed) (harborclients.HarborClienter, error) {
			return &mockRegistryClient{}, nil
		},
	}

	_, err := conn.Connect(ctx, &v1beta1.Registry{})
	if err != nil {
		t.Errorf("Connect should not fail, got %v", err)
	}
}

func TestUpdateRegistryWithAllFields(t *testing.T) {
	ctx := context.Background()
	desc := "Updated description"
	insecure := false

	registry := &v1beta1.Registry{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-registry",
		},
		Spec: v1beta1.RegistrySpec{
			ForProvider: v1beta1.RegistryParameters{
				Name:        "docker-hub",
				Type:        "docker-hub",
				URL:         "https://docker.io",
				Description: &desc,
				Insecure:    &insecure,
			},
		},
	}

	ext := &external{
		service: &mockRegistryClient{
			updateRegistryFunc: func(ctx context.Context, registryName string, spec *harborclients.RegistrySpec) (*harborclients.RegistryStatus, error) {
				return &harborclients.RegistryStatus{
					Name:        spec.Name,
					Type:        spec.Type,
					URL:         spec.URL,
					Description: spec.Description,
					UpdatedAt:   time.Now(),
				}, nil
			},
		},
		kube: nil,
	}

	_, err := ext.Update(ctx, registry)
	if err != nil {
		t.Errorf("Update with all fields should not fail, got %v", err)
	}
}

func TestCreateRegistryWithoutCredentials(t *testing.T) {
	ctx := context.Background()
	registry := &v1beta1.Registry{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-registry",
		},
		Spec: v1beta1.RegistrySpec{
			ForProvider: v1beta1.RegistryParameters{
				Name: "docker-hub",
				Type: "docker-hub",
				URL:  "https://docker.io",
			},
		},
	}

	ext := &external{
		service: &mockRegistryClient{
			createRegistryFunc: func(ctx context.Context, spec *harborclients.RegistrySpec) (*harborclients.RegistryStatus, error) {
				return &harborclients.RegistryStatus{
					Name:      spec.Name,
					Type:      spec.Type,
					URL:       spec.URL,
					CreatedAt: time.Now(),
				}, nil
			},
		},
		kube: nil,
	}

	_, err := ext.Create(ctx, registry)
	if err != nil {
		t.Errorf("Create without credentials should not fail, got %v", err)
	}
}

// mockRegistryClient implements HarborClienter for registry tests
type mockRegistryClient struct {
	harborclients.HarborClienter
	getRegistryFunc    func(ctx context.Context, registryName string) (*harborclients.RegistryStatus, error)
	createRegistryFunc func(ctx context.Context, spec *harborclients.RegistrySpec) (*harborclients.RegistryStatus, error)
	updateRegistryFunc func(ctx context.Context, registryName string, spec *harborclients.RegistrySpec) (*harborclients.RegistryStatus, error)
	deleteRegistryFunc func(ctx context.Context, registryName string) error
}

func (m *mockRegistryClient) GetRegistry(ctx context.Context, registryName string) (*harborclients.RegistryStatus, error) {
	if m.getRegistryFunc != nil {
		return m.getRegistryFunc(ctx, registryName)
	}
	return nil, nil
}

func (m *mockRegistryClient) CreateRegistry(ctx context.Context, spec *harborclients.RegistrySpec) (*harborclients.RegistryStatus, error) {
	if m.createRegistryFunc != nil {
		return m.createRegistryFunc(ctx, spec)
	}
	return nil, nil
}

func (m *mockRegistryClient) UpdateRegistry(ctx context.Context, registryName string, spec *harborclients.RegistrySpec) (*harborclients.RegistryStatus, error) {
	if m.updateRegistryFunc != nil {
		return m.updateRegistryFunc(ctx, registryName, spec)
	}
	return nil, nil
}

func (m *mockRegistryClient) DeleteRegistry(ctx context.Context, registryName string) error {
	if m.deleteRegistryFunc != nil {
		return m.deleteRegistryFunc(ctx, registryName)
	}
	return nil
}

func (m *mockRegistryClient) Close() error {
	return nil
}

func (m *mockRegistryClient) GetBaseURL() string {
	return "https://harbor.example.com"
}

func ptrInt64(i int64) *int64 {
	return &i
}
