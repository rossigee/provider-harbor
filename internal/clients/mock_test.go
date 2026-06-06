/*
Copyright 2024 Crossplane Harbor Provider.
*/

package clients

import (
	"context"
)

// MockHarborClient is a fully mockable Harbor client for testing
type MockHarborClient struct {
	GetProjectFunc                 func(ctx context.Context, projectName string) (*ProjectStatus, error)
	CreateProjectFunc              func(ctx context.Context, spec *ProjectSpec) (*ProjectStatus, error)
	UpdateProjectFunc              func(ctx context.Context, projectName string, spec *ProjectSpec) (*ProjectStatus, error)
	DeleteProjectFunc              func(ctx context.Context, projectName string) error
	ListProjectsFunc               func(ctx context.Context) ([]*ProjectStatus, error)
	ListRepositoriesFunc           func(ctx context.Context, projectID string) ([]*RepositoryStatus, error)
	GetRepositoryFunc              func(ctx context.Context, projectID, repoName string) (*RepositoryStatus, error)
	UpdateRepositoryFunc           func(ctx context.Context, projectID, repoName string, spec *RepositorySpec) (*RepositoryStatus, error)
	DeleteRepositoryFunc           func(ctx context.Context, projectID, repoName string) error
	ListArtifactsFunc              func(ctx context.Context, projectID, repoName string) ([]*ArtifactStatus, error)
	GetArtifactFunc                func(ctx context.Context, projectID, repoName, reference string) (*ArtifactStatus, error)
	DeleteArtifactFunc             func(ctx context.Context, projectID, repoName, reference string) error
	GetArtifactVulnerabilitiesFunc func(ctx context.Context, projectID, repoName, reference string) (*ArtifactStatus, error)
	AddProjectMemberFunc           func(ctx context.Context, projectID, username, role string) error
	ListProjectMembersFunc         func(ctx context.Context, projectID string) ([]*MemberStatus, error)
	GetProjectMemberFunc           func(ctx context.Context, projectID, username string) (*MemberStatus, error)
	UpdateProjectMemberFunc        func(ctx context.Context, projectID, username, role string) error
	DeleteProjectMemberFunc        func(ctx context.Context, projectID, username string) error
	TriggerScanFunc                func(ctx context.Context, projectID, repoName, reference string) error
	ListScansFunc                  func(ctx context.Context, projectID, repoName string) ([]*ScanStatus, error)
	GetScanFunc                    func(ctx context.Context, projectID, repoName, reference string) (*ScanStatus, error)
	StopScanFunc                   func(ctx context.Context, projectID, repoName, reference string) error
	CreateRobotFunc                func(ctx context.Context, spec *RobotSpec) (*RobotStatus, error)
	ListRobotsFunc                 func(ctx context.Context, projectID *string) ([]*RobotStatus, error)
	GetRobotFunc                   func(ctx context.Context, robotID string) (*RobotStatus, error)
	UpdateRobotFunc                func(ctx context.Context, robotID string, spec *RobotSpec) (*RobotStatus, error)
	DeleteRobotFunc                func(ctx context.Context, robotID string) error
	CreateWebhookFunc              func(ctx context.Context, spec *WebhookSpec) (*WebhookStatus, error)
	ListWebhooksFunc               func(ctx context.Context, projectID string) ([]*WebhookStatus, error)
	GetWebhookFunc                 func(ctx context.Context, projectID, webhookID string) (*WebhookStatus, error)
	UpdateWebhookFunc              func(ctx context.Context, projectID, webhookID string, spec *WebhookSpec) (*WebhookStatus, error)
	DeleteWebhookFunc              func(ctx context.Context, projectID, webhookID string) error
	CreateReplicationPolicyFunc    func(ctx context.Context, spec *ReplicationPolicySpec) (*ReplicationPolicyStatus, error)
	ListReplicationPoliciesFunc    func(ctx context.Context) ([]*ReplicationPolicyStatus, error)
	GetReplicationPolicyFunc       func(ctx context.Context, policyID string) (*ReplicationPolicyStatus, error)
	UpdateReplicationPolicyFunc    func(ctx context.Context, policyID string, spec *ReplicationPolicySpec) (*ReplicationPolicyStatus, error)
	DeleteReplicationPolicyFunc    func(ctx context.Context, policyID string) error
	TriggerReplicationFunc         func(ctx context.Context, policyID string) (*ReplicationExecution, error)
	ListReplicationExecutionsFunc  func(ctx context.Context, policyID string) ([]*ReplicationExecution, error)
	CreateRetentionPolicyFunc      func(ctx context.Context, spec *RetentionPolicySpec) (*RetentionPolicyStatus, error)
	ListRetentionPoliciesFunc      func(ctx context.Context, projectID string) ([]*RetentionPolicyStatus, error)
	GetRetentionPolicyFunc         func(ctx context.Context, projectID, policyID string) (*RetentionPolicyStatus, error)
	UpdateRetentionPolicyFunc      func(ctx context.Context, projectID, policyID string, spec *RetentionPolicySpec) (*RetentionPolicyStatus, error)
	DeleteRetentionPolicyFunc      func(ctx context.Context, projectID, policyID string) error
	CloseFunc                      func() error
}

// Implement all interface methods
func (m *MockHarborClient) GetProject(ctx context.Context, projectName string) (*ProjectStatus, error) {
	if m.GetProjectFunc != nil {
		return m.GetProjectFunc(ctx, projectName)
	}
	return nil, nil
}

func (m *MockHarborClient) CreateProject(ctx context.Context, spec *ProjectSpec) (*ProjectStatus, error) {
	if m.CreateProjectFunc != nil {
		return m.CreateProjectFunc(ctx, spec)
	}
	return nil, nil
}

func (m *MockHarborClient) UpdateProject(ctx context.Context, projectName string, spec *ProjectSpec) (*ProjectStatus, error) {
	if m.UpdateProjectFunc != nil {
		return m.UpdateProjectFunc(ctx, projectName, spec)
	}
	return nil, nil
}

func (m *MockHarborClient) DeleteProject(ctx context.Context, projectName string) error {
	if m.DeleteProjectFunc != nil {
		return m.DeleteProjectFunc(ctx, projectName)
	}
	return nil
}

func (m *MockHarborClient) ListProjects(ctx context.Context) ([]*ProjectStatus, error) {
	if m.ListProjectsFunc != nil {
		return m.ListProjectsFunc(ctx)
	}
	return nil, nil
}

func (m *MockHarborClient) ListRobots(ctx context.Context, projectID *string) ([]*RobotStatus, error) {
	if m.ListRobotsFunc != nil {
		return m.ListRobotsFunc(ctx, projectID)
	}
	return nil, nil
}

func (m *MockHarborClient) GetRobot(ctx context.Context, robotID string) (*RobotStatus, error) {
	if m.GetRobotFunc != nil {
		return m.GetRobotFunc(ctx, robotID)
	}
	return nil, nil
}

func (m *MockHarborClient) CreateRobot(ctx context.Context, spec *RobotSpec) (*RobotStatus, error) {
	if m.CreateRobotFunc != nil {
		return m.CreateRobotFunc(ctx, spec)
	}
	return nil, nil
}

func (m *MockHarborClient) UpdateRobot(ctx context.Context, robotID string, spec *RobotSpec) (*RobotStatus, error) {
	if m.UpdateRobotFunc != nil {
		return m.UpdateRobotFunc(ctx, robotID, spec)
	}
	return nil, nil
}

func (m *MockHarborClient) DeleteRobot(ctx context.Context, robotID string) error {
	if m.DeleteRobotFunc != nil {
		return m.DeleteRobotFunc(ctx, robotID)
	}
	return nil
}

func (m *MockHarborClient) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}
