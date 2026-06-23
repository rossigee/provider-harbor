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

	// Test get-by-id (robots are managed by external-name only; no list/adoption).
	mock.GetRobotFunc = func(ctx context.Context, id string) (*clients.RobotStatus, error) {
		if id == robotID {
			return status, nil
		}
		return nil, nil
	}

	got, err := mock.GetRobot(ctx, robotID)
	if err != nil {
		t.Fatalf("GetRobot failed: %v", err)
	}
	if got == nil || got.Name != robotName {
		t.Errorf("Expected to find robot %s by id", robotName)
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
