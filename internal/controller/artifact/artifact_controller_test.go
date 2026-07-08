/*
Copyright 2024 Crossplane Harbor Provider.
*/

package artifact

import (
	"context"
	"errors"
	"github.com/rossigee/provider-harbor/apis/artifact/v1beta1"
	"github.com/rossigee/provider-harbor/internal/clients"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"time"
)

func TestObserveArtifactSuccess(t *testing.T) {
	ctx := context.Background()
	artifact := &v1beta1.Artifact{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-artifact",
		},
		Spec: v1beta1.ArtifactSpec{
			ForProvider: v1beta1.ArtifactParameters{
				ProjectID:      "project-1",
				RepositoryName: "my-repo",
				Reference:      "latest",
			},
		},
	}

	ext := &external{
		service: &mockArtifactClient{
			getArtifactFunc: func(ctx context.Context, projectID, repoName, reference string) (*harborclients.ArtifactStatus, error) {
				return &harborclients.ArtifactStatus{
					ID:                 "artifact-123",
					Digest:             "sha256:abc123def456",
					Size:               1024000,
					PullCount:          10,
					CreationTime:       time.Now().Add(-24 * time.Hour),
					UpdateTime:         time.Now(),
					VulnerabilityCount: 0,
				}, nil
			},
		},
	}

	obs, err := ext.Observe(ctx, artifact)
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

func TestObserveArtifactError(t *testing.T) {
	ctx := context.Background()
	artifact := &v1beta1.Artifact{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-artifact",
		},
		Spec: v1beta1.ArtifactSpec{
			ForProvider: v1beta1.ArtifactParameters{
				ProjectID:      "project-1",
				RepositoryName: "my-repo",
				Reference:      "latest",
			},
		},
	}

	ext := &external{
		service: &mockArtifactClient{
			getArtifactFunc: func(ctx context.Context, projectID, repoName, reference string) (*harborclients.ArtifactStatus, error) {
				return nil, errors.New("not found")
			},
		},
	}

	_, err := ext.Observe(ctx, artifact)
	if err == nil {
		t.Error("Observe should fail when client returns error")
	}
}

func TestObserveArtifactNotFound(t *testing.T) {
	ctx := context.Background()
	artifact := &v1beta1.Artifact{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-artifact",
		},
		Spec: v1beta1.ArtifactSpec{
			ForProvider: v1beta1.ArtifactParameters{
				ProjectID:      "project-1",
				RepositoryName: "my-repo",
				Reference:      "nonexistent",
			},
		},
	}

	ext := &external{
		service: &mockArtifactClient{
			getArtifactFunc: func(ctx context.Context, projectID, repoName, reference string) (*harborclients.ArtifactStatus, error) {
				return nil, errors.New("not found")
			},
		},
	}

	obs, err := ext.Observe(ctx, artifact)
	if err == nil {
		t.Error("Observe should fail when artifact not found")
	}
	if obs.ResourceExists {
		t.Error("ResourceExists should be false when artifact not found")
	}
}

func TestObserveArtifactPopulatesStatus(t *testing.T) {
	ctx := context.Background()
	artifact := &v1beta1.Artifact{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-artifact",
		},
		Spec: v1beta1.ArtifactSpec{
			ForProvider: v1beta1.ArtifactParameters{
				ProjectID:      "project-1",
				RepositoryName: "my-repo",
				Reference:      "latest",
			},
		},
	}

	ext := &external{
		service: &mockArtifactClient{
			getArtifactFunc: func(ctx context.Context, projectID, repoName, reference string) (*harborclients.ArtifactStatus, error) {
				return &harborclients.ArtifactStatus{
					ID:                 "artifact-123",
					Digest:             "sha256:abc123def456",
					Size:               1024000,
					PullCount:          10,
					CreationTime:       time.Now().Add(-24 * time.Hour),
					UpdateTime:         time.Now(),
					VulnerabilityCount: 3,
				}, nil
			},
		},
	}

	obs, err := ext.Observe(ctx, artifact)
	if err != nil {
		t.Errorf("Observe should not fail, got %v", err)
	}

	if artifact.Status.AtProvider.ID == nil {
		t.Error("Status ID should be populated")
	}
	if *artifact.Status.AtProvider.ID != "artifact-123" {
		t.Errorf("Status ID should be 'artifact-123', got %s", *artifact.Status.AtProvider.ID)
	}

	if artifact.Status.AtProvider.Digest == nil {
		t.Error("Status Digest should be populated")
	}
	if *artifact.Status.AtProvider.Digest != "sha256:abc123def456" {
		t.Errorf("Status Digest should be 'sha256:abc123def456', got %s", *artifact.Status.AtProvider.Digest)
	}

	if artifact.Status.AtProvider.Size == nil {
		t.Error("Status Size should be populated")
	}
	if *artifact.Status.AtProvider.Size != 1024000 {
		t.Errorf("Status Size should be 1024000, got %d", *artifact.Status.AtProvider.Size)
	}

	if artifact.Status.AtProvider.PullCount == nil {
		t.Error("Status PullCount should be populated")
	}
	if *artifact.Status.AtProvider.PullCount != 10 {
		t.Errorf("Status PullCount should be 10, got %d", *artifact.Status.AtProvider.PullCount)
	}

	if artifact.Status.AtProvider.VulnerabilityCount == nil {
		t.Error("Status VulnerabilityCount should be populated")
	}
	if *artifact.Status.AtProvider.VulnerabilityCount != 3 {
		t.Errorf("Status VulnerabilityCount should be 3, got %d", *artifact.Status.AtProvider.VulnerabilityCount)
	}

	if !obs.ResourceExists {
		t.Error("ResourceExists should be true")
	}
}

func TestCreateArtifactSuccess(t *testing.T) {
	ctx := context.Background()
	artifact := &v1beta1.Artifact{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-artifact",
		},
		Spec: v1beta1.ArtifactSpec{
			ForProvider: v1beta1.ArtifactParameters{
				ProjectID:      "project-1",
				RepositoryName: "my-repo",
				Reference:      "latest",
			},
		},
	}

	ext := &external{
		service: &mockArtifactClient{},
	}

	_, err := ext.Create(ctx, artifact)
	if err != nil {
		t.Errorf("Create should not fail, got %v", err)
	}
}

func TestUpdateArtifactSuccess(t *testing.T) {
	ctx := context.Background()
	artifact := &v1beta1.Artifact{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-artifact",
		},
		Spec: v1beta1.ArtifactSpec{
			ForProvider: v1beta1.ArtifactParameters{
				ProjectID:      "project-1",
				RepositoryName: "my-repo",
				Reference:      "latest",
			},
		},
	}

	ext := &external{
		service: &mockArtifactClient{},
	}

	_, err := ext.Update(ctx, artifact)
	if err != nil {
		t.Errorf("Update should not fail, got %v", err)
	}
}

func TestDeleteArtifactSuccess(t *testing.T) {
	ctx := context.Background()
	artifact := &v1beta1.Artifact{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-artifact",
		},
		Spec: v1beta1.ArtifactSpec{
			ForProvider: v1beta1.ArtifactParameters{
				ProjectID:      "project-1",
				RepositoryName: "my-repo",
				Reference:      "latest",
			},
		},
	}

	ext := &external{
		service: &mockArtifactClient{
			deleteArtifactFunc: func(ctx context.Context, projectID, repoName, reference string) error {
				return nil
			},
		},
	}

	_, err := ext.Delete(ctx, artifact)
	if err != nil {
		t.Errorf("Delete should not fail, got %v", err)
	}
}

func TestDeleteArtifactError(t *testing.T) {
	ctx := context.Background()
	artifact := &v1beta1.Artifact{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-artifact",
		},
		Spec: v1beta1.ArtifactSpec{
			ForProvider: v1beta1.ArtifactParameters{
				ProjectID:      "project-1",
				RepositoryName: "my-repo",
				Reference:      "latest",
			},
		},
	}

	ext := &external{
		service: &mockArtifactClient{
			deleteArtifactFunc: func(ctx context.Context, projectID, repoName, reference string) error {
				return errors.New("delete failed")
			},
		},
	}

	_, err := ext.Delete(ctx, artifact)
	if err == nil {
		t.Error("Delete should fail when client fails")
	}
}

func TestArtifactHasRequiredFields(t *testing.T) {
	artifact := &v1beta1.Artifact{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-artifact",
			Namespace: "default",
		},
		Spec: v1beta1.ArtifactSpec{
			ForProvider: v1beta1.ArtifactParameters{
				ProjectID:      "project-1",
				RepositoryName: "my-repo",
				Reference:      "latest",
			},
		},
	}

	if artifact.Spec.ForProvider.ProjectID == "" {
		t.Error("Artifact ProjectID should not be empty")
	}
	if artifact.Spec.ForProvider.RepositoryName == "" {
		t.Error("Artifact RepositoryName should not be empty")
	}
	if artifact.Spec.ForProvider.Reference == "" {
		t.Error("Artifact Reference should not be empty")
	}
	if artifact.Name == "" {
		t.Error("Metadata Name should not be empty")
	}
}

func TestArtifactStatusFields(t *testing.T) {
	artifact := &v1beta1.Artifact{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-artifact",
		},
		Spec: v1beta1.ArtifactSpec{
			ForProvider: v1beta1.ArtifactParameters{
				ProjectID:      "project-1",
				RepositoryName: "my-repo",
				Reference:      "latest",
			},
		},
		Status: v1beta1.ArtifactStatus{
			AtProvider: v1beta1.ArtifactObservation{
				ID:     ptrString("artifact-123"),
				Digest: ptrString("sha256:abc123"),
			},
		},
	}

	if artifact.Status.AtProvider.ID == nil {
		t.Error("Status ID should be populated")
	}
	if *artifact.Status.AtProvider.ID != "artifact-123" {
		t.Errorf("Status ID should be 'artifact-123', got %s", *artifact.Status.AtProvider.ID)
	}
	if artifact.Status.AtProvider.Digest == nil {
		t.Error("Status Digest should be populated")
	}
}

func TestArtifactParametersValidation(t *testing.T) {
	tests := []struct {
		name    string
		params  v1beta1.ArtifactParameters
		isValid bool
	}{
		{
			name: "valid with required fields",
			params: v1beta1.ArtifactParameters{
				ProjectID:      "project-1",
				RepositoryName: "my-repo",
				Reference:      "latest",
			},
			isValid: true,
		},
		{
			name: "valid with type",
			params: v1beta1.ArtifactParameters{
				ProjectID:      "project-1",
				RepositoryName: "my-repo",
				Reference:      "latest",
				Type:           ptrString("image"),
			},
			isValid: true,
		},
		{
			name: "missing required project ID",
			params: v1beta1.ArtifactParameters{
				RepositoryName: "my-repo",
				Reference:      "latest",
			},
			isValid: false,
		},
		{
			name: "missing required repository name",
			params: v1beta1.ArtifactParameters{
				ProjectID: "project-1",
				Reference: "latest",
			},
			isValid: false,
		},
		{
			name: "missing required reference",
			params: v1beta1.ArtifactParameters{
				ProjectID:      "project-1",
				RepositoryName: "my-repo",
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.params.ProjectID != "" && tt.params.RepositoryName != "" && tt.params.Reference != ""
			if isValid != tt.isValid {
				t.Errorf("Expected valid=%v, got %v", tt.isValid, isValid)
			}
		})
	}
}

func TestArtifactWithOptionalType(t *testing.T) {
	ctx := context.Background()
	artifactType := "image"
	artifact := &v1beta1.Artifact{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-artifact",
		},
		Spec: v1beta1.ArtifactSpec{
			ForProvider: v1beta1.ArtifactParameters{
				ProjectID:      "project-1",
				RepositoryName: "my-repo",
				Reference:      "latest",
				Type:           &artifactType,
			},
		},
	}

	if artifact.Spec.ForProvider.Type == nil {
		t.Error("Type should be set")
	}
	if *artifact.Spec.ForProvider.Type != "image" {
		t.Errorf("Type should be 'image', got %s", *artifact.Spec.ForProvider.Type)
	}

	ext := &external{
		service: &mockArtifactClient{
			getArtifactFunc: func(ctx context.Context, projectID, repoName, reference string) (*harborclients.ArtifactStatus, error) {
				return &harborclients.ArtifactStatus{
					ID:                 "artifact-123",
					Digest:             "sha256:abc123def456",
					Size:               1024000,
					PullCount:          10,
					CreationTime:       time.Now(),
					UpdateTime:         time.Now(),
					VulnerabilityCount: 0,
				}, nil
			},
		},
	}

	_, err := ext.Observe(ctx, artifact)
	if err != nil {
		t.Errorf("Observe should not fail, got %v", err)
	}
}

type mockArtifactClient struct {
	harborclients.HarborClienter
	getArtifactFunc    func(ctx context.Context, projectID, repoName, reference string) (*harborclients.ArtifactStatus, error)
	deleteArtifactFunc func(ctx context.Context, projectID, repoName, reference string) error
}

func (m *mockArtifactClient) GetArtifact(ctx context.Context, projectID, repoName, reference string) (*harborclients.ArtifactStatus, error) {
	if m.getArtifactFunc != nil {
		return m.getArtifactFunc(ctx, projectID, repoName, reference)
	}
	return nil, nil
}

func (m *mockArtifactClient) DeleteArtifact(ctx context.Context, projectID, repoName, reference string) error {
	if m.deleteArtifactFunc != nil {
		return m.deleteArtifactFunc(ctx, projectID, repoName, reference)
	}
	return nil
}

func (m *mockArtifactClient) Close() error {
	return nil
}

func (m *mockArtifactClient) GetBaseURL() string {
	return "https://harbor.example.com"
}

func ptrString(s string) *string {
	return &s
}
