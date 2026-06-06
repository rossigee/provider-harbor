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

func (m *MockHarborClient) ListRepositories(ctx context.Context, projectID string) ([]*RepositoryStatus, error) {
	if m.ListRepositoriesFunc != nil {
		return m.ListRepositoriesFunc(ctx, projectID)
	}
	return nil, nil
}

func (m *MockHarborClient) GetRepository(ctx context.Context, projectID, repoName string) (*RepositoryStatus, error) {
	if m.GetRepositoryFunc != nil {
		return m.GetRepositoryFunc(ctx, projectID, repoName)
	}
	return nil, nil
}

func (m *MockHarborClient) UpdateRepository(ctx context.Context, projectID, repoName string, spec *RepositorySpec) (*RepositoryStatus, error) {
	if m.UpdateRepositoryFunc != nil {
		return m.UpdateRepositoryFunc(ctx, projectID, repoName, spec)
	}
	return nil, nil
}

func (m *MockHarborClient) DeleteRepository(ctx context.Context, projectID, repoName string) error {
	if m.DeleteRepositoryFunc != nil {
		return m.DeleteRepositoryFunc(ctx, projectID, repoName)
	}
	return nil
}

func (m *MockHarborClient) ListArtifacts(ctx context.Context, projectID, repoName string) ([]*ArtifactStatus, error) {
	if m.ListArtifactsFunc != nil {
		return m.ListArtifactsFunc(ctx, projectID, repoName)
	}
	return nil, nil
}

func (m *MockHarborClient) GetArtifact(ctx context.Context, projectID, repoName, reference string) (*ArtifactStatus, error) {
	if m.GetArtifactFunc != nil {
		return m.GetArtifactFunc(ctx, projectID, repoName, reference)
	}
	return nil, nil
}

func (m *MockHarborClient) DeleteArtifact(ctx context.Context, projectID, repoName, reference string) error {
	if m.DeleteArtifactFunc != nil {
		return m.DeleteArtifactFunc(ctx, projectID, repoName, reference)
	}
	return nil
}

func (m *MockHarborClient) GetArtifactVulnerabilities(ctx context.Context, projectID, repoName, reference string) (*ArtifactStatus, error) {
	if m.GetArtifactVulnerabilitiesFunc != nil {
		return m.GetArtifactVulnerabilitiesFunc(ctx, projectID, repoName, reference)
	}
	return nil, nil
}

func (m *MockHarborClient) AddProjectMember(ctx context.Context, projectID, username, role string) error {
	if m.AddProjectMemberFunc != nil {
		return m.AddProjectMemberFunc(ctx, projectID, username, role)
	}
	return nil
}

func (m *MockHarborClient) ListProjectMembers(ctx context.Context, projectID string) ([]*MemberStatus, error) {
	if m.ListProjectMembersFunc != nil {
		return m.ListProjectMembersFunc(ctx, projectID)
	}
	return nil, nil
}

func (m *MockHarborClient) GetProjectMember(ctx context.Context, projectID, username string) (*MemberStatus, error) {
	if m.GetProjectMemberFunc != nil {
		return m.GetProjectMemberFunc(ctx, projectID, username)
	}
	return nil, nil
}

func (m *MockHarborClient) UpdateProjectMember(ctx context.Context, projectID, username, role string) error {
	if m.UpdateProjectMemberFunc != nil {
		return m.UpdateProjectMemberFunc(ctx, projectID, username, role)
	}
	return nil
}

func (m *MockHarborClient) DeleteProjectMember(ctx context.Context, projectID, username string) error {
	if m.DeleteProjectMemberFunc != nil {
		return m.DeleteProjectMemberFunc(ctx, projectID, username)
	}
	return nil
}

func (m *MockHarborClient) TriggerScan(ctx context.Context, projectID, repoName, reference string) error {
	if m.TriggerScanFunc != nil {
		return m.TriggerScanFunc(ctx, projectID, repoName, reference)
	}
	return nil
}

func (m *MockHarborClient) ListScans(ctx context.Context, projectID, repoName string) ([]*ScanStatus, error) {
	if m.ListScansFunc != nil {
		return m.ListScansFunc(ctx, projectID, repoName)
	}
	return nil, nil
}

func (m *MockHarborClient) GetScan(ctx context.Context, projectID, repoName, reference string) (*ScanStatus, error) {
	if m.GetScanFunc != nil {
		return m.GetScanFunc(ctx, projectID, repoName, reference)
	}
	return nil, nil
}

func (m *MockHarborClient) StopScan(ctx context.Context, projectID, repoName, reference string) error {
	if m.StopScanFunc != nil {
		return m.StopScanFunc(ctx, projectID, repoName, reference)
	}
	return nil
}

func (m *MockHarborClient) CreateWebhook(ctx context.Context, spec *WebhookSpec) (*WebhookStatus, error) {
	if m.CreateWebhookFunc != nil {
		return m.CreateWebhookFunc(ctx, spec)
	}
	return nil, nil
}

func (m *MockHarborClient) ListWebhooks(ctx context.Context, projectID string) ([]*WebhookStatus, error) {
	if m.ListWebhooksFunc != nil {
		return m.ListWebhooksFunc(ctx, projectID)
	}
	return nil, nil
}

func (m *MockHarborClient) GetWebhook(ctx context.Context, projectID, webhookID string) (*WebhookStatus, error) {
	if m.GetWebhookFunc != nil {
		return m.GetWebhookFunc(ctx, projectID, webhookID)
	}
	return nil, nil
}

func (m *MockHarborClient) UpdateWebhook(ctx context.Context, projectID, webhookID string, spec *WebhookSpec) (*WebhookStatus, error) {
	if m.UpdateWebhookFunc != nil {
		return m.UpdateWebhookFunc(ctx, projectID, webhookID, spec)
	}
	return nil, nil
}

func (m *MockHarborClient) DeleteWebhook(ctx context.Context, projectID, webhookID string) error {
	if m.DeleteWebhookFunc != nil {
		return m.DeleteWebhookFunc(ctx, projectID, webhookID)
	}
	return nil
}

func (m *MockHarborClient) CreateReplicationPolicy(ctx context.Context, spec *ReplicationPolicySpec) (*ReplicationPolicyStatus, error) {
	if m.CreateReplicationPolicyFunc != nil {
		return m.CreateReplicationPolicyFunc(ctx, spec)
	}
	return nil, nil
}

func (m *MockHarborClient) ListReplicationPolicies(ctx context.Context) ([]*ReplicationPolicyStatus, error) {
	if m.ListReplicationPoliciesFunc != nil {
		return m.ListReplicationPoliciesFunc(ctx)
	}
	return nil, nil
}

func (m *MockHarborClient) GetReplicationPolicy(ctx context.Context, policyID string) (*ReplicationPolicyStatus, error) {
	if m.GetReplicationPolicyFunc != nil {
		return m.GetReplicationPolicyFunc(ctx, policyID)
	}
	return nil, nil
}

func (m *MockHarborClient) UpdateReplicationPolicy(ctx context.Context, policyID string, spec *ReplicationPolicySpec) (*ReplicationPolicyStatus, error) {
	if m.UpdateReplicationPolicyFunc != nil {
		return m.UpdateReplicationPolicyFunc(ctx, policyID, spec)
	}
	return nil, nil
}

func (m *MockHarborClient) DeleteReplicationPolicy(ctx context.Context, policyID string) error {
	if m.DeleteReplicationPolicyFunc != nil {
		return m.DeleteReplicationPolicyFunc(ctx, policyID)
	}
	return nil
}

func (m *MockHarborClient) TriggerReplication(ctx context.Context, policyID string) (*ReplicationExecution, error) {
	if m.TriggerReplicationFunc != nil {
		return m.TriggerReplicationFunc(ctx, policyID)
	}
	return nil, nil
}

func (m *MockHarborClient) ListReplicationExecutions(ctx context.Context, policyID string) ([]*ReplicationExecution, error) {
	if m.ListReplicationExecutionsFunc != nil {
		return m.ListReplicationExecutionsFunc(ctx, policyID)
	}
	return nil, nil
}

func (m *MockHarborClient) CreateRetentionPolicy(ctx context.Context, spec *RetentionPolicySpec) (*RetentionPolicyStatus, error) {
	if m.CreateRetentionPolicyFunc != nil {
		return m.CreateRetentionPolicyFunc(ctx, spec)
	}
	return nil, nil
}

func (m *MockHarborClient) ListRetentionPolicies(ctx context.Context, projectID string) ([]*RetentionPolicyStatus, error) {
	if m.ListRetentionPoliciesFunc != nil {
		return m.ListRetentionPoliciesFunc(ctx, projectID)
	}
	return nil, nil
}

func (m *MockHarborClient) GetRetentionPolicy(ctx context.Context, projectID, policyID string) (*RetentionPolicyStatus, error) {
	if m.GetRetentionPolicyFunc != nil {
		return m.GetRetentionPolicyFunc(ctx, projectID, policyID)
	}
	return nil, nil
}

func (m *MockHarborClient) UpdateRetentionPolicy(ctx context.Context, projectID, policyID string, spec *RetentionPolicySpec) (*RetentionPolicyStatus, error) {
	if m.UpdateRetentionPolicyFunc != nil {
		return m.UpdateRetentionPolicyFunc(ctx, projectID, policyID, spec)
	}
	return nil, nil
}

func (m *MockHarborClient) DeleteRetentionPolicy(ctx context.Context, projectID, policyID string) error {
	if m.DeleteRetentionPolicyFunc != nil {
		return m.DeleteRetentionPolicyFunc(ctx, projectID, policyID)
	}
	return nil
}

func (m *MockHarborClient) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}
