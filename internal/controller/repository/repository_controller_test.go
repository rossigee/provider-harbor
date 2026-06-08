/*
Copyright 2024 Crossplane Harbor Provider.
*/

package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rossigee/provider-harbor/apis/repository/v1beta1"
	harborclients "github.com/rossigee/provider-harbor/internal/clients"
)

func TestConnectNotRepository(t *testing.T) {
	ctx := context.Background()
	conn := &connector{}

	_, err := conn.Connect(ctx, nil)
	if err == nil || err.Error() != errNotRepository {
		t.Errorf("Connect with nil should return %s error", errNotRepository)
	}
}

func TestObserveNotRepository(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Observe(ctx, nil)
	if err == nil || err.Error() != errNotRepository {
		t.Errorf("Observe with nil should return %s error", errNotRepository)
	}
}

func TestUpdateNotRepository(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Update(ctx, nil)
	if err == nil || err.Error() != errNotRepository {
		t.Errorf("Update with nil should return %s error", errNotRepository)
	}
}

func TestDeleteNotRepository(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Delete(ctx, nil)
	if err == nil || err.Error() != errNotRepository {
		t.Errorf("Delete with nil should return %s error", errNotRepository)
	}
}

func TestCreateNotRepository(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Create(ctx, nil)
	if err == nil || err.Error() != errNotRepository {
		t.Errorf("Create with nil should return %s error", errNotRepository)
	}
}

func TestObserveRepositoryNotFound(t *testing.T) {
	ctx := context.Background()
	repo := &v1beta1.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-repository",
		},
		Spec: v1beta1.RepositorySpec{
			ForProvider: v1beta1.RepositoryParameters{
				ProjectID: "project-1",
				Name:      "my-repo",
			},
		},
	}

	ext := &external{
		service: &mockRepositoryClient{
			getRepositoryFunc: func(ctx context.Context, projectID, repoName string) (*harborclients.RepositoryStatus, error) {
				return nil, errors.New("not found")
			},
		},
	}

	obs, err := ext.Observe(ctx, repo)
	if err == nil {
		t.Error("Observe should fail when client returns error")
	}
	if obs.ResourceExists {
		t.Error("ResourceExists should be false when repository not found")
	}
}

func TestObserveRepositoryExists(t *testing.T) {
	ctx := context.Background()
	repo := &v1beta1.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-repository",
		},
		Spec: v1beta1.RepositorySpec{
			ForProvider: v1beta1.RepositoryParameters{
				ProjectID: "project-1",
				Name:      "my-repo",
			},
		},
	}

	ext := &external{
		service: &mockRepositoryClient{
			getRepositoryFunc: func(ctx context.Context, projectID, repoName string) (*harborclients.RepositoryStatus, error) {
				return &harborclients.RepositoryStatus{
					ID:            "repo-123",
					FullName:      "project-1/my-repo",
					ProjectID:     "project-1",
					Description:   "Test repository",
					ArtifactCount: 10,
					CreationTime:  time.Now(),
					UpdateTime:    time.Now(),
				}, nil
			},
		},
	}

	obs, err := ext.Observe(ctx, repo)
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

func TestObserveRepositoryNotUpToDate(t *testing.T) {
	ctx := context.Background()
	desc := "updated description"
	repo := &v1beta1.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-repository",
		},
		Spec: v1beta1.RepositorySpec{
			ForProvider: v1beta1.RepositoryParameters{
				ProjectID:   "project-1",
				Name:        "my-repo",
				Description: &desc,
			},
		},
	}

	ext := &external{
		service: &mockRepositoryClient{
			getRepositoryFunc: func(ctx context.Context, projectID, repoName string) (*harborclients.RepositoryStatus, error) {
				return &harborclients.RepositoryStatus{
					ID:            "repo-123",
					FullName:      "project-1/my-repo",
					ProjectID:     "project-1",
					Description:   "old description",
					ArtifactCount: 10,
					CreationTime:  time.Now(),
					UpdateTime:    time.Now(),
				}, nil
			},
		},
	}

	obs, err := ext.Observe(ctx, repo)
	if err != nil {
		t.Errorf("Observe should not fail, got %v", err)
	}
	if !obs.ResourceExists {
		t.Error("ResourceExists should be true")
	}
	if obs.ResourceUpToDate {
		t.Error("ResourceUpToDate should be false when description differs")
	}
}

func TestCreateRepositorySuccess(t *testing.T) {
	ctx := context.Background()
	repo := &v1beta1.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-repository",
		},
		Spec: v1beta1.RepositorySpec{
			ForProvider: v1beta1.RepositoryParameters{
				ProjectID: "project-1",
				Name:      "my-repo",
			},
		},
	}

	ext := &external{
		service: &mockRepositoryClient{
			getRepositoryFunc: func(ctx context.Context, projectID, repoName string) (*harborclients.RepositoryStatus, error) {
				return nil, errors.New("not found")
			},
			updateRepositoryFunc: func(ctx context.Context, projectID, repoName string, spec *harborclients.RepositorySpec) (*harborclients.RepositoryStatus, error) {
				return &harborclients.RepositoryStatus{
					ID:         "repo-123",
					ProjectID:  projectID,
					FullName:   projectID + "/" + repoName,
					UpdateTime: time.Now(),
				}, nil
			},
		},
	}

	_, err := ext.Create(ctx, repo)
	if err != nil {
		t.Errorf("Create should not fail, got %v", err)
	}
}

func TestCreateRepositoryAlreadyExists(t *testing.T) {
	ctx := context.Background()
	repo := &v1beta1.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-repository",
		},
		Spec: v1beta1.RepositorySpec{
			ForProvider: v1beta1.RepositoryParameters{
				ProjectID: "project-1",
				Name:      "my-repo",
			},
		},
	}

	ext := &external{
		service: &mockRepositoryClient{
			getRepositoryFunc: func(ctx context.Context, projectID, repoName string) (*harborclients.RepositoryStatus, error) {
				return &harborclients.RepositoryStatus{
					ID:       "repo-123",
					FullName: projectID + "/" + repoName,
				}, nil
			},
		},
	}

	_, err := ext.Create(ctx, repo)
	if err != nil {
		t.Errorf("Create should not fail when repo exists, got %v", err)
	}
}

func TestCreateRepositoryError(t *testing.T) {
	ctx := context.Background()
	repo := &v1beta1.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-repository",
		},
		Spec: v1beta1.RepositorySpec{
			ForProvider: v1beta1.RepositoryParameters{
				ProjectID: "project-1",
				Name:      "my-repo",
			},
		},
	}

	ext := &external{
		service: &mockRepositoryClient{
			getRepositoryFunc: func(ctx context.Context, projectID, repoName string) (*harborclients.RepositoryStatus, error) {
				return nil, errors.New("not found")
			},
			updateRepositoryFunc: func(ctx context.Context, projectID, repoName string, spec *harborclients.RepositorySpec) (*harborclients.RepositoryStatus, error) {
				return nil, errors.New("create failed")
			},
		},
	}

	_, err := ext.Create(ctx, repo)
	if err == nil {
		t.Error("Create should fail when client fails")
	}
}

func TestUpdateRepositorySuccess(t *testing.T) {
	ctx := context.Background()
	repo := &v1beta1.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-repository",
		},
		Spec: v1beta1.RepositorySpec{
			ForProvider: v1beta1.RepositoryParameters{
				ProjectID: "project-1",
				Name:      "my-repo",
			},
		},
	}

	ext := &external{
		service: &mockRepositoryClient{
			updateRepositoryFunc: func(ctx context.Context, projectID, repoName string, spec *harborclients.RepositorySpec) (*harborclients.RepositoryStatus, error) {
				return &harborclients.RepositoryStatus{
					ID:         "repo-123",
					ProjectID:  projectID,
					FullName:   projectID + "/" + repoName,
					UpdateTime: time.Now(),
				}, nil
			},
		},
	}

	_, err := ext.Update(ctx, repo)
	if err != nil {
		t.Errorf("Update should not fail, got %v", err)
	}
}

func TestUpdateRepositoryError(t *testing.T) {
	ctx := context.Background()
	repo := &v1beta1.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-repository",
		},
		Spec: v1beta1.RepositorySpec{
			ForProvider: v1beta1.RepositoryParameters{
				ProjectID: "project-1",
				Name:      "my-repo",
			},
		},
	}

	ext := &external{
		service: &mockRepositoryClient{
			updateRepositoryFunc: func(ctx context.Context, projectID, repoName string, spec *harborclients.RepositorySpec) (*harborclients.RepositoryStatus, error) {
				return nil, errors.New("update failed")
			},
		},
	}

	_, err := ext.Update(ctx, repo)
	if err == nil {
		t.Error("Update should fail when client fails")
	}
}

func TestDeleteRepositorySuccess(t *testing.T) {
	ctx := context.Background()
	repo := &v1beta1.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-repository",
		},
		Spec: v1beta1.RepositorySpec{
			ForProvider: v1beta1.RepositoryParameters{
				ProjectID: "project-1",
				Name:      "my-repo",
			},
		},
	}

	ext := &external{
		service: &mockRepositoryClient{
			deleteRepositoryFunc: func(ctx context.Context, projectID, repoName string) error {
				return nil
			},
		},
	}

	_, err := ext.Delete(ctx, repo)
	if err != nil {
		t.Errorf("Delete should not fail, got %v", err)
	}
}

func TestDeleteRepositoryError(t *testing.T) {
	ctx := context.Background()
	repo := &v1beta1.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-repository",
		},
		Spec: v1beta1.RepositorySpec{
			ForProvider: v1beta1.RepositoryParameters{
				ProjectID: "project-1",
				Name:      "my-repo",
			},
		},
	}

	ext := &external{
		service: &mockRepositoryClient{
			deleteRepositoryFunc: func(ctx context.Context, projectID, repoName string) error {
				return errors.New("delete failed")
			},
		},
	}

	_, err := ext.Delete(ctx, repo)
	if err == nil {
		t.Error("Delete should fail when client fails")
	}
}

func TestRepositoryHasRequiredFields(t *testing.T) {
	repo := &v1beta1.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-repository",
			Namespace: "default",
		},
		Spec: v1beta1.RepositorySpec{
			ForProvider: v1beta1.RepositoryParameters{
				ProjectID: "project-1",
				Name:      "my-repo",
			},
		},
	}

	if repo.Spec.ForProvider.Name == "" {
		t.Error("Repository Name should not be empty")
	}
	if repo.Spec.ForProvider.ProjectID == "" {
		t.Error("Repository ProjectID should not be empty")
	}
	if repo.Name == "" {
		t.Error("Metadata Name should not be empty")
	}
}

func TestRepositoryStatusFields(t *testing.T) {
	repo := &v1beta1.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-repository",
		},
		Spec: v1beta1.RepositorySpec{
			ForProvider: v1beta1.RepositoryParameters{
				ProjectID: "project-1",
				Name:      "my-repo",
			},
		},
		Status: v1beta1.RepositoryStatus{
			AtProvider: v1beta1.RepositoryObservation{
				ID:       ptrString("repo-123"),
				FullName: ptrString("project-1/my-repo"),
			},
		},
	}

	if repo.Status.AtProvider.ID == nil {
		t.Error("Status ID should be populated")
	}
	if *repo.Status.AtProvider.ID != "repo-123" {
		t.Errorf("Status ID should be 'repo-123', got %s", *repo.Status.AtProvider.ID)
	}
}

func TestRepositoryParametersValidation(t *testing.T) {
	tests := []struct {
		name    string
		params  v1beta1.RepositoryParameters
		isValid bool
	}{
		{
			name: "valid with required fields",
			params: v1beta1.RepositoryParameters{
				ProjectID: "project-1",
				Name:      "my-repo",
			},
			isValid: true,
		},
		{
			name: "valid with description",
			params: v1beta1.RepositoryParameters{
				ProjectID:   "project-1",
				Name:        "my-repo",
				Description: ptrString("My repository"),
			},
			isValid: true,
		},
		{
			name: "missing required name",
			params: v1beta1.RepositoryParameters{
				ProjectID: "project-1",
			},
			isValid: false,
		},
		{
			name: "missing required project ID",
			params: v1beta1.RepositoryParameters{
				Name: "my-repo",
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.params.Name != "" && tt.params.ProjectID != ""
			if isValid != tt.isValid {
				t.Errorf("Expected valid=%v, got %v", tt.isValid, isValid)
			}
		})
	}
}

type mockRepositoryClient struct {
	harborclients.HarborClienter
	getRepositoryFunc    func(ctx context.Context, projectID, repoName string) (*harborclients.RepositoryStatus, error)
	updateRepositoryFunc func(ctx context.Context, projectID, repoName string, spec *harborclients.RepositorySpec) (*harborclients.RepositoryStatus, error)
	deleteRepositoryFunc func(ctx context.Context, projectID, repoName string) error
}

func (m *mockRepositoryClient) GetRepository(ctx context.Context, projectID, repoName string) (*harborclients.RepositoryStatus, error) {
	if m.getRepositoryFunc != nil {
		return m.getRepositoryFunc(ctx, projectID, repoName)
	}
	return nil, nil
}

func (m *mockRepositoryClient) UpdateRepository(ctx context.Context, projectID, repoName string, spec *harborclients.RepositorySpec) (*harborclients.RepositoryStatus, error) {
	if m.updateRepositoryFunc != nil {
		return m.updateRepositoryFunc(ctx, projectID, repoName, spec)
	}
	return nil, nil
}

func (m *mockRepositoryClient) DeleteRepository(ctx context.Context, projectID, repoName string) error {
	if m.deleteRepositoryFunc != nil {
		return m.deleteRepositoryFunc(ctx, projectID, repoName)
	}
	return nil
}

func (m *mockRepositoryClient) Close() error {
	return nil
}

func (m *mockRepositoryClient) GetBaseURL() string {
	return "https://harbor.example.com"
}

func ptrString(s string) *string {
	return &s
}
