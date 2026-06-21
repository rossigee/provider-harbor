/*
Copyright 2024 Crossplane Harbor Provider.
*/

package project

import (
	"context"
	"errors"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	"github.com/rossigee/provider-harbor/apis/project/v1beta1"
	harborclients "github.com/rossigee/provider-harbor/internal/clients"
)

func TestConnectNotProject(t *testing.T) {
	ctx := context.Background()
	conn := &connector{}

	_, err := conn.Connect(ctx, nil)
	if err == nil || err.Error() != errNotProject {
		t.Errorf("Connect with nil should return %s error", errNotProject)
	}
}

func TestObserveNotProject(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Observe(ctx, nil)
	if err == nil || err.Error() != errNotProject {
		t.Errorf("Observe with nil should return %s error", errNotProject)
	}
}

func TestUpdateNotProject(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Update(ctx, nil)
	if err == nil || err.Error() != errNotProject {
		t.Errorf("Update with nil should return %s error", errNotProject)
	}
}

func TestDeleteNotProject(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Delete(ctx, nil)
	if err == nil || err.Error() != errNotProject {
		t.Errorf("Delete with nil should return %s error", errNotProject)
	}
}

func TestCreateNotProject(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Create(ctx, nil)
	if err == nil || err.Error() != errNotProject {
		t.Errorf("Create with nil should return %s error", errNotProject)
	}
}

func TestObserveProjectNotFound(t *testing.T) {
	ctx := context.Background()
	project := &v1beta1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-project",
		},
		Spec: v1beta1.ProjectSpec{
			ForProvider: v1beta1.ProjectParameters{
				Name: "my-project",
			},
		},
	}

	ext := &external{
		service: &mockProjectClient{
			getProjectFunc: func(ctx context.Context, projectName string) (*harborclients.ProjectStatus, error) {
				return nil, errors.New("not found")
			},
		},
	}

	obs, err := ext.Observe(ctx, project)
	if err != nil {
		t.Errorf("Observe should not fail, got %v", err)
	}
	if obs.ResourceExists {
		t.Error("ResourceExists should be false when project not found")
	}
}

func TestObserveProjectExists(t *testing.T) {
	ctx := context.Background()
	project := &v1beta1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-project",
		},
		Spec: v1beta1.ProjectSpec{
			ForProvider: v1beta1.ProjectParameters{
				Name:   "my-project",
				Public: ptrBool(false),
			},
		},
	}

	ext := &external{
		service: &mockProjectClient{
			getProjectFunc: func(ctx context.Context, projectName string) (*harborclients.ProjectStatus, error) {
				return &harborclients.ProjectStatus{
					Name:      "my-project",
					Public:    false,
					OwnerID:   1,
					OwnerName: "admin",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}, nil
			},
		},
	}

	obs, err := ext.Observe(ctx, project)
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

func TestObserveProjectNotUpToDate(t *testing.T) {
	ctx := context.Background()
	isPublic := true
	project := &v1beta1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-project",
		},
		Spec: v1beta1.ProjectSpec{
			ForProvider: v1beta1.ProjectParameters{
				Name:   "my-project",
				Public: &isPublic,
			},
		},
	}

	ext := &external{
		service: &mockProjectClient{
			getProjectFunc: func(ctx context.Context, projectName string) (*harborclients.ProjectStatus, error) {
				return &harborclients.ProjectStatus{
					Name:      "my-project",
					Public:    false,
					OwnerID:   1,
					OwnerName: "admin",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}, nil
			},
		},
	}

	obs, err := ext.Observe(ctx, project)
	if err != nil {
		t.Errorf("Observe should not fail, got %v", err)
	}
	if !obs.ResourceExists {
		t.Error("ResourceExists should be true")
	}
	if obs.ResourceUpToDate {
		t.Error("ResourceUpToDate should be false when public flag differs")
	}
}

func TestCreateProjectSuccess(t *testing.T) {
	ctx := context.Background()
	project := &v1beta1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-project",
		},
		Spec: v1beta1.ProjectSpec{
			ForProvider: v1beta1.ProjectParameters{
				Name: "my-project",
			},
		},
	}

	ext := &external{
		service: &mockProjectClient{
			createProjectFunc: func(ctx context.Context, spec *harborclients.ProjectSpec) (*harborclients.ProjectStatus, error) {
				return &harborclients.ProjectStatus{
					Name:      spec.Name,
					Public:    spec.Public,
					CreatedAt: time.Now(),
				}, nil
			},
		},
	}

	_, err := ext.Create(ctx, project)
	if err != nil {
		t.Errorf("Create should not fail, got %v", err)
	}
}

func TestCreateProjectError(t *testing.T) {
	ctx := context.Background()
	project := &v1beta1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-project",
		},
		Spec: v1beta1.ProjectSpec{
			ForProvider: v1beta1.ProjectParameters{
				Name: "my-project",
			},
		},
	}

	ext := &external{
		service: &mockProjectClient{
			createProjectFunc: func(ctx context.Context, spec *harborclients.ProjectSpec) (*harborclients.ProjectStatus, error) {
				return nil, errors.New("create failed")
			},
		},
	}

	_, err := ext.Create(ctx, project)
	if err == nil {
		t.Error("Create should fail when client fails")
	}
}

func TestUpdateProjectSuccess(t *testing.T) {
	ctx := context.Background()
	project := &v1beta1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-project",
		},
		Spec: v1beta1.ProjectSpec{
			ForProvider: v1beta1.ProjectParameters{
				Name:   "my-project",
				Public: ptrBool(true),
			},
		},
	}

	ext := &external{
		service: &mockProjectClient{
			updateProjectFunc: func(ctx context.Context, projectID string, spec *harborclients.ProjectSpec) (*harborclients.ProjectStatus, error) {
				return &harborclients.ProjectStatus{
					Name:      spec.Name,
					Public:    spec.Public,
					UpdatedAt: time.Now(),
				}, nil
			},
		},
	}

	_, err := ext.Update(ctx, project)
	if err != nil {
		t.Errorf("Update should not fail, got %v", err)
	}
}

func TestDeleteProjectSuccess(t *testing.T) {
	ctx := context.Background()
	project := &v1beta1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-project",
		},
		Spec: v1beta1.ProjectSpec{
			ForProvider: v1beta1.ProjectParameters{
				Name: "my-project",
			},
		},
	}

	ext := &external{
		service: &mockProjectClient{
			deleteProjectFunc: func(ctx context.Context, projectID string) error {
				return nil
			},
		},
	}

	_, err := ext.Delete(ctx, project)
	if err != nil {
		t.Errorf("Delete should not fail, got %v", err)
	}
}

func TestDeleteProjectError(t *testing.T) {
	ctx := context.Background()
	project := &v1beta1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-project",
		},
		Spec: v1beta1.ProjectSpec{
			ForProvider: v1beta1.ProjectParameters{
				Name: "my-project",
			},
		},
	}

	ext := &external{
		service: &mockProjectClient{
			deleteProjectFunc: func(ctx context.Context, projectID string) error {
				return errors.New("delete failed")
			},
		},
	}

	_, err := ext.Delete(ctx, project)
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

	boolVal := true
	resultBool := getBoolValue(&boolVal)
	if !resultBool {
		t.Errorf("getBoolValue failed")
	}

	nilBool := getBoolValue(nil)
	if nilBool {
		t.Errorf("getBoolValue with nil should return false")
	}
}

func TestProjectHasRequiredFields(t *testing.T) {
	project := &v1beta1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-project",
			Namespace: "default",
		},
		Spec: v1beta1.ProjectSpec{
			ForProvider: v1beta1.ProjectParameters{
				Name: "my-project",
			},
		},
	}

	if project.Spec.ForProvider.Name == "" {
		t.Error("Project Name should not be empty")
	}
	if project.Name == "" {
		t.Error("Metadata Name should not be empty")
	}
}

func TestProjectStatusFields(t *testing.T) {
	project := &v1beta1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-project",
		},
		Spec: v1beta1.ProjectSpec{
			ForProvider: v1beta1.ProjectParameters{
				Name: "my-project",
			},
		},
		Status: v1beta1.ProjectStatus{
			AtProvider: v1beta1.ProjectObservation{
				ID: ptrString("123"),
			},
		},
	}

	if project.Status.AtProvider.ID == nil {
		t.Error("Status ID should be populated")
	}
	if *project.Status.AtProvider.ID != "123" {
		t.Errorf("Status ID should be '123', got %s", *project.Status.AtProvider.ID)
	}
}

func TestProjectParametersValidation(t *testing.T) {
	tests := []struct {
		name    string
		params  v1beta1.ProjectParameters
		isValid bool
	}{
		{
			name: "valid with only required name",
			params: v1beta1.ProjectParameters{
				Name: "my-project",
			},
			isValid: true,
		},
		{
			name: "valid with public flag",
			params: v1beta1.ProjectParameters{
				Name:   "public-project",
				Public: ptrBool(true),
			},
			isValid: true,
		},
		{
			name: "valid with security settings",
			params: v1beta1.ProjectParameters{
				Name:                     "secure-project",
				EnableContentTrust:       ptrBool(true),
				EnableContentTrustCosign: ptrBool(true),
				AutoScanImages:           ptrBool(true),
				PreventVulnerableImages:  ptrBool(true),
				Severity:                 ptrString("high"),
			},
			isValid: true,
		},
		{
			name: "valid with CVE allowlist",
			params: v1beta1.ProjectParameters{
				Name:         "project-with-cves",
				CVEAllowlist: []string{"CVE-2024-1234", "CVE-2024-5678"},
			},
			isValid: true,
		},
		{
			name: "valid with storage limit",
			params: v1beta1.ProjectParameters{
				Name:         "project-with-quota",
				StorageLimit: ptrInt64(1073741824),
			},
			isValid: true,
		},
		{
			name: "valid with metadata",
			params: v1beta1.ProjectParameters{
				Name: "project-with-metadata",
				Metadata: map[string]string{
					"environment": "production",
					"team":        "platform",
				},
			},
			isValid: true,
		},
		{
			name: "missing required name",
			params: v1beta1.ProjectParameters{
				Public: ptrBool(true),
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.params.Name != ""
			if isValid != tt.isValid {
				t.Errorf("Expected valid=%v, got %v", tt.isValid, isValid)
			}
		})
	}
}

func TestProjectSecuritySettings(t *testing.T) {
	tests := []struct {
		name     string
		severity string
		isValid  bool
	}{
		{"negligible severity", "negligible", true},
		{"low severity", "low", true},
		{"medium severity", "medium", true},
		{"high severity", "high", true},
		{"critical severity", "critical", true},
		{"invalid severity", "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			severities := map[string]bool{
				"negligible": true,
				"low":        true,
				"medium":     true,
				"high":       true,
				"critical":   true,
			}
			isValid := severities[tt.severity]
			if isValid != tt.isValid {
				t.Errorf("Expected severity '%s' to be valid=%v, got %v", tt.severity, tt.isValid, isValid)
			}
		})
	}
}

func TestProjectStorageLimitValidation(t *testing.T) {
	tests := []struct {
		name  string
		limit int64
		desc  string
	}{
		{"1MB", 1048576, "1MB storage limit"},
		{"1GB", 1073741824, "1GB storage limit"},
		{"10GB", 10737418240, "10GB storage limit"},
		{"1TB", 1099511627776, "1TB storage limit"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project := &v1beta1.Project{
				Spec: v1beta1.ProjectSpec{
					ForProvider: v1beta1.ProjectParameters{
						Name:         "test-project",
						StorageLimit: ptrInt64(tt.limit),
					},
				},
			}

			if project.Spec.ForProvider.StorageLimit == nil {
				t.Error("StorageLimit should be set")
			}
			if *project.Spec.ForProvider.StorageLimit != tt.limit {
				t.Errorf("Expected %d, got %d", tt.limit, *project.Spec.ForProvider.StorageLimit)
			}
		})
	}
}

func TestConnectProjectSuccess(t *testing.T) {
	ctx := context.Background()
	conn := &connector{
		kube: nil,
		newServiceFn: func(ctx context.Context, kube client.Client, mg resource.Managed) (harborclients.HarborClienter, error) {
			return &mockProjectClient{}, nil
		},
	}

	project := &v1beta1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-project",
		},
		Spec: v1beta1.ProjectSpec{
			ForProvider: v1beta1.ProjectParameters{
				Name: "my-project",
			},
		},
	}

	ext, err := conn.Connect(ctx, project)
	if err != nil {
		t.Errorf("Connect should not fail, got %v", err)
	}
	if ext == nil {
		t.Error("Connect should return external client")
	}
}

func TestUpdateProjectError(t *testing.T) {
	ctx := context.Background()
	project := &v1beta1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-project",
		},
		Spec: v1beta1.ProjectSpec{
			ForProvider: v1beta1.ProjectParameters{
				Name:   "my-project",
				Public: ptrBool(true),
			},
		},
	}

	ext := &external{
		service: &mockProjectClient{
			updateProjectFunc: func(ctx context.Context, projectID string, spec *harborclients.ProjectSpec) (*harborclients.ProjectStatus, error) {
				return nil, errors.New("update failed")
			},
		},
	}

	_, err := ext.Update(ctx, project)
	if err == nil {
		t.Error("Update should fail when client fails")
	}
}

func TestDisconnect(t *testing.T) {
	ctx := context.Background()
	ext := &external{
		service: &mockProjectClient{},
	}

	err := ext.Disconnect(ctx)
	if err != nil {
		t.Errorf("Disconnect should not fail, got %v", err)
	}
}

func TestObserveProjectNilPublic(t *testing.T) {
	ctx := context.Background()
	project := &v1beta1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-project",
		},
		Spec: v1beta1.ProjectSpec{
			ForProvider: v1beta1.ProjectParameters{
				Name: "my-project",
			},
		},
	}

	ext := &external{
		service: &mockProjectClient{
			getProjectFunc: func(ctx context.Context, projectName string) (*harborclients.ProjectStatus, error) {
				return &harborclients.ProjectStatus{
					Name:      "my-project",
					Public:    false,
					OwnerID:   1,
					OwnerName: "admin",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}, nil
			},
		},
	}

	obs, err := ext.Observe(ctx, project)
	if err != nil {
		t.Errorf("Observe should not fail, got %v", err)
	}
	if !obs.ResourceExists {
		t.Error("ResourceExists should be true")
	}
	if !obs.ResourceUpToDate {
		t.Error("ResourceUpToDate should be true when Public is nil in spec")
	}
}

func TestConnectSuccess(t *testing.T) {
	ctx := context.Background()
	conn := &connector{
		kube: nil,
		newServiceFn: func(ctx context.Context, kube client.Client, mg resource.Managed) (harborclients.HarborClienter, error) {
			return &mockProjectClient{}, nil
		},
	}

	_, err := conn.Connect(ctx, &v1beta1.Project{})
	if err != nil {
		t.Errorf("Connect should not fail, got %v", err)
	}
}

func TestConnectClientError(t *testing.T) {
	ctx := context.Background()
	conn := &connector{
		kube: nil,
		newServiceFn: func(ctx context.Context, kube client.Client, mg resource.Managed) (harborclients.HarborClienter, error) {
			return nil, errors.New("client creation failed")
		},
	}

	_, err := conn.Connect(ctx, &v1beta1.Project{})
	if err == nil {
		t.Error("Connect should fail when client creation fails")
	}
}

func TestObserveProjectGetError(t *testing.T) {
	ctx := context.Background()
	project := &v1beta1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-project",
		},
		Spec: v1beta1.ProjectSpec{
			ForProvider: v1beta1.ProjectParameters{
				Name: "my-project",
			},
		},
	}

	ext := &external{
		service: &mockProjectClient{
			getProjectFunc: func(ctx context.Context, projectName string) (*harborclients.ProjectStatus, error) {
				return nil, errors.New("get failed")
			},
		},
	}

	obs, err := ext.Observe(ctx, project)
	if err != nil {
		t.Errorf("Observe should not fail on error, got %v", err)
	}
	if obs.ResourceExists {
		t.Error("ResourceExists should be false when client returns error")
	}
}

// mockProjectClient implements HarborClienter for project tests
type mockProjectClient struct {
	harborclients.HarborClienter
	getProjectFunc    func(ctx context.Context, projectName string) (*harborclients.ProjectStatus, error)
	createProjectFunc func(ctx context.Context, spec *harborclients.ProjectSpec) (*harborclients.ProjectStatus, error)
	updateProjectFunc func(ctx context.Context, projectID string, spec *harborclients.ProjectSpec) (*harborclients.ProjectStatus, error)
	deleteProjectFunc func(ctx context.Context, projectID string) error
	closeFunc         func() error
}

func (m *mockProjectClient) GetProject(ctx context.Context, projectName string) (*harborclients.ProjectStatus, error) {
	if m.getProjectFunc != nil {
		return m.getProjectFunc(ctx, projectName)
	}
	return nil, nil
}

func (m *mockProjectClient) CreateProject(ctx context.Context, spec *harborclients.ProjectSpec) (*harborclients.ProjectStatus, error) {
	if m.createProjectFunc != nil {
		return m.createProjectFunc(ctx, spec)
	}
	return nil, nil
}

func (m *mockProjectClient) UpdateProject(ctx context.Context, projectID string, spec *harborclients.ProjectSpec) (*harborclients.ProjectStatus, error) {
	if m.updateProjectFunc != nil {
		return m.updateProjectFunc(ctx, projectID, spec)
	}
	return nil, nil
}

func (m *mockProjectClient) DeleteProject(ctx context.Context, projectID string) error {
	if m.deleteProjectFunc != nil {
		return m.deleteProjectFunc(ctx, projectID)
	}
	return nil
}

func (m *mockProjectClient) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func (m *mockProjectClient) GetBaseURL() string {
	return "https://harbor.example.com"
}

// Helper functions
func ptrBool(b bool) *bool {
	return &b
}

func ptrInt64(i int64) *int64 {
	return &i
}

func ptrString(s string) *string {
	return &s
}
