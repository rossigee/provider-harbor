/*
Copyright 2024 Crossplane Harbor Provider.
*/

package controller

import (
	"context"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/rossigee/provider-harbor/internal/clients"
)

// MockHarborClient is a mock implementation of HarborClient for testing
type MockHarborClient struct {
	GetProjectFn func(ctx context.Context, projectName string) (*clients.ProjectStatus, error)
	CreateProjectFn func(ctx context.Context, spec *clients.ProjectSpec) (*clients.ProjectStatus, error)
	UpdateProjectFn func(ctx context.Context, projectName string, spec *clients.ProjectSpec) (*clients.ProjectStatus, error)
	DeleteProjectFn func(ctx context.Context, projectName string) error
	ListProjectsFn func(ctx context.Context) ([]*clients.ProjectStatus, error)
	CloseFn func() error
}

func (m *MockHarborClient) GetProject(ctx context.Context, projectName string) (*clients.ProjectStatus, error) {
	if m.GetProjectFn != nil {
		return m.GetProjectFn(ctx, projectName)
	}
	return nil, errors.New("not implemented")
}

func (m *MockHarborClient) CreateProject(ctx context.Context, spec *clients.ProjectSpec) (*clients.ProjectStatus, error) {
	if m.CreateProjectFn != nil {
		return m.CreateProjectFn(ctx, spec)
	}
	return nil, errors.New("not implemented")
}

func (m *MockHarborClient) UpdateProject(ctx context.Context, projectName string, spec *clients.ProjectSpec) (*clients.ProjectStatus, error) {
	if m.UpdateProjectFn != nil {
		return m.UpdateProjectFn(ctx, projectName, spec)
	}
	return nil, errors.New("not implemented")
}

func (m *MockHarborClient) DeleteProject(ctx context.Context, projectName string) error {
	if m.DeleteProjectFn != nil {
		return m.DeleteProjectFn(ctx, projectName)
	}
	return errors.New("not implemented")
}

func (m *MockHarborClient) ListProjects(ctx context.Context) ([]*clients.ProjectStatus, error) {
	if m.ListProjectsFn != nil {
		return m.ListProjectsFn(ctx)
	}
	return nil, errors.New("not implemented")
}

func (m *MockHarborClient) Close() error {
	if m.CloseFn != nil {
		return m.CloseFn()
	}
	return nil
}

// MockKubeClient is a mock Kubernetes client for testing
type MockKubeClient struct {
	GetFn func(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error
	CreateFn func(ctx context.Context, obj client.Object, opts ...client.CreateOption) error
	UpdateFn func(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error
	DeleteFn func(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error
}

func (m *MockKubeClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	if m.GetFn != nil {
		return m.GetFn(ctx, key, obj, opts...)
	}
	return nil
}

func (m *MockKubeClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, obj, opts...)
	}
	return nil
}

func (m *MockKubeClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, obj, opts...)
	}
	return nil
}

func (m *MockKubeClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, obj, opts...)
	}
	return nil
}

// TestLogger returns a test logger for use in tests
func TestLogger(t *testing.T) logging.Logger {
	return logging.NewNopLogger()
}

// NewTestManagedResource creates a simple test object
// Note: This is a stub for testing error handling paths
func NewTestManagedResource(name string) interface{} {
	return struct{ name string }{name: name}
}

// PointerTo returns a pointer to the given value
func PointerTo[T any](v T) *T {
	return &v
}

// AssertError checks if an error matches the expected message
func AssertError(t *testing.T, err error, expectedMsg string) {
	if err == nil && expectedMsg == "" {
		return
	}
	if err == nil {
		t.Errorf("Expected error with message '%s', got nil", expectedMsg)
	}
	if err != nil && expectedMsg != "" && err.Error() != expectedMsg {
		t.Errorf("Expected error '%s', got '%s'", expectedMsg, err.Error())
	}
}
