/*
Copyright 2024 Crossplane Harbor Provider.
*/

package testing

import (
	"context"
	"time"

	harborclients "github.com/rossigee/provider-harbor/internal/clients"
)

// MockHarborClient is a mock implementation of the Harbor client for testing
type MockHarborClient struct {
	// User operations
	GetUserFunc    func(ctx context.Context, username string) (*harborclients.UserStatus, error)
	CreateUserFunc func(ctx context.Context, spec *harborclients.UserSpec) (*harborclients.UserStatus, error)
	UpdateUserFunc func(ctx context.Context, username string, spec *harborclients.UserSpec) (*harborclients.UserStatus, error)
	DeleteUserFunc func(ctx context.Context, username string) error

	// Project operations
	GetProjectFunc    func(ctx context.Context, projectName string) (*harborclients.ProjectStatus, error)
	CreateProjectFunc func(ctx context.Context, spec *harborclients.ProjectSpec) (*harborclients.ProjectStatus, error)
	UpdateProjectFunc func(ctx context.Context, projectID string, spec *harborclients.ProjectSpec) (*harborclients.ProjectStatus, error)
	DeleteProjectFunc func(ctx context.Context, projectID string) error
}

// GetUser calls GetUserFunc
func (m *MockHarborClient) GetUser(ctx context.Context, username string) (*harborclients.UserStatus, error) {
	if m.GetUserFunc != nil {
		return m.GetUserFunc(ctx, username)
	}
	return nil, nil
}

// CreateUser calls CreateUserFunc
func (m *MockHarborClient) CreateUser(ctx context.Context, spec *harborclients.UserSpec) (*harborclients.UserStatus, error) {
	if m.CreateUserFunc != nil {
		return m.CreateUserFunc(ctx, spec)
	}
	return &harborclients.UserStatus{
		Username:  spec.Username,
		Email:     spec.Email,
		AdminFlag: spec.AdminFlag,
		CreatedAt: time.Now(),
	}, nil
}

// UpdateUser calls UpdateUserFunc
func (m *MockHarborClient) UpdateUser(ctx context.Context, username string, spec *harborclients.UserSpec) (*harborclients.UserStatus, error) {
	if m.UpdateUserFunc != nil {
		return m.UpdateUserFunc(ctx, username, spec)
	}
	return &harborclients.UserStatus{
		Username:  spec.Username,
		Email:     spec.Email,
		AdminFlag: spec.AdminFlag,
	}, nil
}

// DeleteUser calls DeleteUserFunc
func (m *MockHarborClient) DeleteUser(ctx context.Context, username string) error {
	if m.DeleteUserFunc != nil {
		return m.DeleteUserFunc(ctx, username)
	}
	return nil
}

// GetProject calls GetProjectFunc
func (m *MockHarborClient) GetProject(ctx context.Context, projectName string) (*harborclients.ProjectStatus, error) {
	if m.GetProjectFunc != nil {
		return m.GetProjectFunc(ctx, projectName)
	}
	return nil, nil
}

// CreateProject calls CreateProjectFunc
func (m *MockHarborClient) CreateProject(ctx context.Context, spec *harborclients.ProjectSpec) (*harborclients.ProjectStatus, error) {
	if m.CreateProjectFunc != nil {
		return m.CreateProjectFunc(ctx, spec)
	}
	return &harborclients.ProjectStatus{
		Name:   spec.Name,
		Public: spec.Public,
	}, nil
}

// UpdateProject calls UpdateProjectFunc
func (m *MockHarborClient) UpdateProject(ctx context.Context, projectID string, spec *harborclients.ProjectSpec) (*harborclients.ProjectStatus, error) {
	if m.UpdateProjectFunc != nil {
		return m.UpdateProjectFunc(ctx, projectID, spec)
	}
	return &harborclients.ProjectStatus{
		Name:   spec.Name,
		Public: spec.Public,
	}, nil
}

// DeleteProject calls DeleteProjectFunc
func (m *MockHarborClient) DeleteProject(ctx context.Context, projectID string) error {
	if m.DeleteProjectFunc != nil {
		return m.DeleteProjectFunc(ctx, projectID)
	}
	return nil
}
