/*
Copyright 2024 Crossplane Harbor Provider.
*/

package controller

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rossigee/provider-harbor/internal/clients"
)

// TestClientMockProjectWorkflow verifies MockHarborClient with project operations
func TestClientMockProjectWorkflow(t *testing.T) {
	ctx := context.Background()
	mock := &clients.MockHarborClient{}

	projectName := "test-project"
	projectID := "1"
	now := time.Now()

	// Test create
	mock.CreateProjectFunc = func(ctx context.Context, spec *clients.ProjectSpec) (*clients.ProjectStatus, error) {
		return &clients.ProjectStatus{
			ID:        projectID,
			Name:      projectName,
			CreatedAt: now,
			UpdatedAt: now,
		}, nil
	}

	status, err := mock.CreateProject(ctx, &clients.ProjectSpec{Name: projectName})
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}
	if status.ID != projectID {
		t.Errorf("Expected ID %s, got %s", projectID, status.ID)
	}

	// Test get
	mock.GetProjectFunc = func(ctx context.Context, name string) (*clients.ProjectStatus, error) {
		if name == projectName {
			return status, nil
		}
		return nil, errors.New("not found")
	}

	retrieved, err := mock.GetProject(ctx, projectName)
	if err != nil {
		t.Fatalf("GetProject failed: %v", err)
	}
	if retrieved.Name != projectName {
		t.Errorf("Expected name %s, got %s", projectName, retrieved.Name)
	}

	// Test update
	mock.UpdateProjectFunc = func(ctx context.Context, name string, spec *clients.ProjectSpec) (*clients.ProjectStatus, error) {
		if name == projectName {
			status.UpdatedAt = time.Now()
			return status, nil
		}
		return nil, errors.New("not found")
	}

	updated, err := mock.UpdateProject(ctx, projectName, &clients.ProjectSpec{Name: projectName})
	if err != nil {
		t.Fatalf("UpdateProject failed: %v", err)
	}
	if updated.UpdatedAt.Before(now) {
		t.Error("Expected UpdatedAt to be set")
	}

	// Test delete
	mock.DeleteProjectFunc = func(ctx context.Context, name string) error {
		if name == projectName {
			return nil
		}
		return errors.New("not found")
	}

	err = mock.DeleteProject(ctx, projectName)
	if err != nil {
		t.Fatalf("DeleteProject failed: %v", err)
	}
}

// TestClientMockRobotWorkflow verifies MockHarborClient with robot operations
func TestClientMockRobotWorkflow(t *testing.T) {
	ctx := context.Background()
	mock := &clients.MockHarborClient{}

	robotID := "robot-123"
	robotName := "ci-robot"
	secret := "robot-secret-token"
	now := time.Now()

	// Test create
	mock.CreateRobotFunc = func(ctx context.Context, spec *clients.RobotSpec) (*clients.RobotStatus, error) {
		return &clients.RobotStatus{
			ID:           robotID,
			Name:         robotName,
			Secret:       secret,
			CreationTime: now,
			UpdateTime:   now,
		}, nil
	}

	status, err := mock.CreateRobot(ctx, &clients.RobotSpec{Name: robotName})
	if err != nil {
		t.Fatalf("CreateRobot failed: %v", err)
	}
	if status.Secret != secret {
		t.Errorf("Expected secret %s, got %s", secret, status.Secret)
	}

	// Test list
	mock.ListRobotsFunc = func(ctx context.Context, projectID *string) ([]*clients.RobotStatus, error) {
		return []*clients.RobotStatus{status}, nil
	}

	robots, err := mock.ListRobots(ctx, nil)
	if err != nil {
		t.Fatalf("ListRobots failed: %v", err)
	}
	if len(robots) != 1 || robots[0].Name != robotName {
		t.Errorf("Expected to find robot %s", robotName)
	}

	// Test delete
	mock.DeleteRobotFunc = func(ctx context.Context, id string) error {
		if id == robotID {
			return nil
		}
		return errors.New("not found")
	}

	err = mock.DeleteRobot(ctx, robotID)
	if err != nil {
		t.Fatalf("DeleteRobot failed: %v", err)
	}
}

// TestClientMockRepositoryWorkflow verifies MockHarborClient with repository operations
func TestClientMockRepositoryWorkflow(t *testing.T) {
	ctx := context.Background()
	mock := &clients.MockHarborClient{}

	projectID := "1"
	repoFullName := "test-project/test-app"
	now := time.Now()

	// Test list repositories
	mock.ListRepositoriesFunc = func(ctx context.Context, pid string) ([]*clients.RepositoryStatus, error) {
		if pid == projectID {
			return []*clients.RepositoryStatus{
				{
					ID:           "repo-123",
					FullName:     repoFullName,
					ProjectID:    projectID,
					CreationTime: now,
					UpdateTime:   now,
				},
			}, nil
		}
		return nil, errors.New("not found")
	}

	repos, err := mock.ListRepositories(ctx, projectID)
	if err != nil {
		t.Fatalf("ListRepositories failed: %v", err)
	}
	if len(repos) != 1 || repos[0].FullName != repoFullName {
		t.Errorf("Expected to find repository %s", repoFullName)
	}

	// Test get repository
	mock.GetRepositoryFunc = func(ctx context.Context, pid, fullname string) (*clients.RepositoryStatus, error) {
		if pid == projectID && fullname == repoFullName {
			return repos[0], nil
		}
		return nil, errors.New("not found")
	}

	repo, err := mock.GetRepository(ctx, projectID, repoFullName)
	if err != nil {
		t.Fatalf("GetRepository failed: %v", err)
	}
	if repo.FullName != repoFullName {
		t.Errorf("Expected repository %s, got %s", repoFullName, repo.FullName)
	}
}

// TestClientMockArtifactWorkflow verifies MockHarborClient with artifact operations
func TestClientMockArtifactWorkflow(t *testing.T) {
	ctx := context.Background()
	mock := &clients.MockHarborClient{}

	projectID := "1"
	repoName := "test-app"
	digest := "sha256:abc123def456"
	now := time.Now()

	// Test list artifacts
	mock.ListArtifactsFunc = func(ctx context.Context, pid, repo string) ([]*clients.ArtifactStatus, error) {
		if pid == projectID && repo == repoName {
			return []*clients.ArtifactStatus{
				{
					ID:           "artifact-123",
					Digest:       digest,
					Size:         1024,
					CreationTime: now,
					UpdateTime:   now,
				},
			}, nil
		}
		return nil, errors.New("not found")
	}

	artifacts, err := mock.ListArtifacts(ctx, projectID, repoName)
	if err != nil {
		t.Fatalf("ListArtifacts failed: %v", err)
	}
	if len(artifacts) != 1 || artifacts[0].Digest != digest {
		t.Errorf("Expected artifact with digest %s", digest)
	}

	// Test get artifact vulnerabilities
	mock.GetArtifactVulnerabilitiesFunc = func(ctx context.Context, pid, repo, ref string) (*clients.ArtifactStatus, error) {
		if pid == projectID && repo == repoName && ref == digest {
			art := artifacts[0]
			art.VulnerabilityCount = 7
			return art, nil
		}
		return nil, errors.New("not found")
	}

	vulnArtifact, err := mock.GetArtifactVulnerabilities(ctx, projectID, repoName, digest)
	if err != nil {
		t.Fatalf("GetArtifactVulnerabilities failed: %v", err)
	}
	if vulnArtifact.VulnerabilityCount != 7 {
		t.Errorf("Expected 7 vulnerabilities, got %d", vulnArtifact.VulnerabilityCount)
	}
}

// TestClientMockMemberWorkflow verifies MockHarborClient with member operations
func TestClientMockMemberWorkflow(t *testing.T) {
	ctx := context.Background()
	mock := &clients.MockHarborClient{}

	projectID := "1"
	username := "developer"
	role := "developer"
	now := time.Now()

	// Test add member
	mock.AddProjectMemberFunc = func(ctx context.Context, pid, user, r string) error {
		if pid == projectID && user == username {
			return nil
		}
		return errors.New("not found")
	}

	err := mock.AddProjectMember(ctx, projectID, username, role)
	if err != nil {
		t.Fatalf("AddProjectMember failed: %v", err)
	}

	// Test list members
	mock.ListProjectMembersFunc = func(ctx context.Context, pid string) ([]*clients.MemberStatus, error) {
		if pid == projectID {
			return []*clients.MemberStatus{
				{
					ID:           "member-123",
					MemberName:   username,
					Role:         role,
					CreationTime: now,
				},
			}, nil
		}
		return nil, errors.New("not found")
	}

	members, err := mock.ListProjectMembers(ctx, projectID)
	if err != nil {
		t.Fatalf("ListProjectMembers failed: %v", err)
	}
	if len(members) != 1 || members[0].MemberName != username {
		t.Errorf("Expected to find member %s", username)
	}

	// Test delete member
	mock.DeleteProjectMemberFunc = func(ctx context.Context, pid, user string) error {
		if pid == projectID && user == username {
			return nil
		}
		return errors.New("not found")
	}

	err = mock.DeleteProjectMember(ctx, projectID, username)
	if err != nil {
		t.Fatalf("DeleteProjectMember failed: %v", err)
	}
}
