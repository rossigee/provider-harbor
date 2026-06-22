/*
Copyright 2024 Crossplane Harbor Provider.
*/

package clients

import (
	"context"
	"time"
)

// HarborClienter defines the interface for Harbor client operations
// This allows for easy mocking in tests
type HarborClienter interface {
	// Base client methods
	GetBaseURL() string
	Close() error
	TestConnection(ctx context.Context) error
	GetVersion(ctx context.Context) (string, error)
	GetMemoryFootprint() string

	// Project operations
	GetProject(ctx context.Context, projectName string) (*ProjectStatus, error)
	CreateProject(ctx context.Context, spec *ProjectSpec) (*ProjectStatus, error)
	UpdateProject(ctx context.Context, projectID string, spec *ProjectSpec) (*ProjectStatus, error)
	DeleteProject(ctx context.Context, projectID string) error
	ListProjects(ctx context.Context) ([]*ProjectStatus, error)

	// Scanner operations
	CreateScannerRegistration(ctx context.Context, spec *ScannerSpec) (*ScannerStatus, error)
	GetScannerRegistration(ctx context.Context, scannerID string) (*ScannerStatus, error)
	UpdateScannerRegistration(ctx context.Context, scannerID string, spec *ScannerSpec) (*ScannerStatus, error)
	DeleteScannerRegistration(ctx context.Context, scannerID string) error
	ListScannerRegistrations(ctx context.Context) ([]*ScannerStatus, error)

	// User operations
	GetUser(ctx context.Context, username string) (*UserStatus, error)
	CreateUser(ctx context.Context, spec *UserSpec) (*UserStatus, error)
	UpdateUser(ctx context.Context, username string, spec *UserSpec) (*UserStatus, error)
	DeleteUser(ctx context.Context, username string) error

	// Registry operations
	CreateRegistry(ctx context.Context, spec *RegistrySpec) (*RegistryStatus, error)
	GetRegistry(ctx context.Context, registryName string) (*RegistryStatus, error)
	UpdateRegistry(ctx context.Context, registryName string, spec *RegistrySpec) (*RegistryStatus, error)
	DeleteRegistry(ctx context.Context, registryName string) error

	// Repository operations
	ListRepositories(ctx context.Context, projectID string) ([]*RepositoryStatus, error)
	GetRepository(ctx context.Context, projectID, repoName string) (*RepositoryStatus, error)
	UpdateRepository(ctx context.Context, projectID, repoName string, spec *RepositorySpec) (*RepositoryStatus, error)
	DeleteRepository(ctx context.Context, projectID, repoName string) error

	// Artifact operations
	ListArtifacts(ctx context.Context, projectID, repoName string) ([]*ArtifactStatus, error)
	GetArtifact(ctx context.Context, projectID, repoName, reference string) (*ArtifactStatus, error)
	DeleteArtifact(ctx context.Context, projectID, repoName, reference string) error
	GetArtifactVulnerabilities(ctx context.Context, projectID, repoName, reference string) (*ArtifactStatus, error)

	// Member operations.
	//
	// The username-keyed AddProjectMember/GetProjectMember/UpdateProjectMember/
	// DeleteProjectMember set serves the deprecated catch-all Member kind (user
	// members only). The id-keyed AddProjectUserMember/AddProjectGroupMember/
	// GetProjectMemberByID/UpdateProjectMemberByID/DeleteProjectMemberByID set —
	// plus FindProjectMember — serves the single-responsibility UserMember and
	// GroupMember kinds, which key the external resource on the Harbor member id.
	AddProjectMember(ctx context.Context, projectID, username, role string) error
	ListProjectMembers(ctx context.Context, projectID string) ([]*MemberStatus, error)
	GetProjectMember(ctx context.Context, projectID, username string) (*MemberStatus, error)
	UpdateProjectMember(ctx context.Context, projectID, username, role string) error
	DeleteProjectMember(ctx context.Context, projectID, username string) error

	// AddProjectUserMember adds a user member and returns its Harbor member id.
	AddProjectUserMember(ctx context.Context, projectID, username, role string) (string, error)
	// AddProjectGroupMember adds a group member and returns its Harbor member id.
	AddProjectGroupMember(ctx context.Context, projectID, groupName string, groupType int64, role string) (string, error)
	// FindProjectMember matches an existing member by entity type ("u"/"g") and
	// name, returning (nil, nil) when absent. Used to adopt a member when the
	// external-name (member id) is not yet known.
	FindProjectMember(ctx context.Context, projectID, entityType, entityName string) (*MemberStatus, error)
	// GetProjectMemberByID retrieves a member by its Harbor member id, returning
	// (nil, nil) when absent.
	GetProjectMemberByID(ctx context.Context, projectID, memberID string) (*MemberStatus, error)
	// UpdateProjectMemberByID updates a member's role by its Harbor member id.
	UpdateProjectMemberByID(ctx context.Context, projectID, memberID, role string) error
	// DeleteProjectMemberByID removes a member by its Harbor member id (idempotent).
	DeleteProjectMemberByID(ctx context.Context, projectID, memberID string) error

	// Scan operations
	TriggerScan(ctx context.Context, projectID, repoName, reference string) error
	ListScans(ctx context.Context, projectID, repoName string) ([]*ScanStatus, error)
	GetScan(ctx context.Context, projectID, repoName, reference string) (*ScanStatus, error)
	StopScan(ctx context.Context, projectID, repoName, reference string) error

	// Robot operations
	CreateRobot(ctx context.Context, spec *RobotSpec) (*RobotStatus, error)
	ListRobots(ctx context.Context, projectID *string) ([]*RobotStatus, error)
	GetRobot(ctx context.Context, robotID string) (*RobotStatus, error)
	UpdateRobot(ctx context.Context, robotID string, spec *RobotSpec) (*RobotStatus, error)
	DeleteRobot(ctx context.Context, robotID string) error

	// Webhook operations
	CreateWebhook(ctx context.Context, spec *WebhookSpec) (*WebhookStatus, error)
	ListWebhooks(ctx context.Context, projectID string) ([]*WebhookStatus, error)
	GetWebhook(ctx context.Context, projectID, webhookID string) (*WebhookStatus, error)
	UpdateWebhook(ctx context.Context, projectID, webhookID string, spec *WebhookSpec) (*WebhookStatus, error)
	DeleteWebhook(ctx context.Context, projectID, webhookID string) error

	// Replication operations
	CreateReplicationPolicy(ctx context.Context, spec *ReplicationPolicySpec) (*ReplicationPolicyStatus, error)
	ListReplicationPolicies(ctx context.Context) ([]*ReplicationPolicyStatus, error)
	GetReplicationPolicy(ctx context.Context, policyID string) (*ReplicationPolicyStatus, error)
	UpdateReplicationPolicy(ctx context.Context, policyID string, spec *ReplicationPolicySpec) (*ReplicationPolicyStatus, error)
	DeleteReplicationPolicy(ctx context.Context, policyID string) error
	TriggerReplication(ctx context.Context, policyID string) (*ReplicationExecution, error)
	ListReplicationExecutions(ctx context.Context, policyID string) ([]*ReplicationExecution, error)

	// Retention operations
	CreateRetentionPolicy(ctx context.Context, spec *RetentionPolicySpec) (*RetentionPolicyStatus, error)
	ListRetentionPolicies(ctx context.Context, projectID string) ([]*RetentionPolicyStatus, error)
	GetRetentionPolicy(ctx context.Context, projectID, policyID string) (*RetentionPolicyStatus, error)
	UpdateRetentionPolicy(ctx context.Context, projectID, policyID string, spec *RetentionPolicySpec) (*RetentionPolicyStatus, error)
	DeleteRetentionPolicy(ctx context.Context, projectID, policyID string) error

	// UserGroup operations
	CreateUserGroup(ctx context.Context, spec *UserGroupSpec) (*UserGroupStatus, error)
	ListUserGroups(ctx context.Context) ([]*UserGroupStatus, error)
	GetUserGroup(ctx context.Context, groupID int64) (*UserGroupStatus, error)
	GetUserGroupByName(ctx context.Context, name string) (*UserGroupStatus, error)
	UpdateUserGroup(ctx context.Context, groupID int64, spec *UserGroupSpec) (*UserGroupStatus, error)
	DeleteUserGroup(ctx context.Context, groupID int64) error
}

// Ensure HarborClient implements HarborClienter
var _ HarborClienter = (*HarborClient)(nil)

// MockHarborClient implements HarborClienter for testing
type MockHarborClient struct {
	// Base client methods
	GetBaseURLFunc         func() string
	CloseFunc              func() error
	TestConnectionFunc     func(ctx context.Context) error
	GetVersionFunc         func(ctx context.Context) (string, error)
	GetMemoryFootprintFunc func() string

	// Project operations
	GetProjectFunc    func(ctx context.Context, projectName string) (*ProjectStatus, error)
	CreateProjectFunc func(ctx context.Context, spec *ProjectSpec) (*ProjectStatus, error)
	UpdateProjectFunc func(ctx context.Context, projectID string, spec *ProjectSpec) (*ProjectStatus, error)
	DeleteProjectFunc func(ctx context.Context, projectID string) error
	ListProjectsFunc  func(ctx context.Context) ([]*ProjectStatus, error)

	// Scanner operations
	CreateScannerRegistrationFunc func(ctx context.Context, spec *ScannerSpec) (*ScannerStatus, error)
	GetScannerRegistrationFunc    func(ctx context.Context, scannerID string) (*ScannerStatus, error)
	UpdateScannerRegistrationFunc func(ctx context.Context, scannerID string, spec *ScannerSpec) (*ScannerStatus, error)
	DeleteScannerRegistrationFunc func(ctx context.Context, scannerID string) error
	ListScannerRegistrationsFunc  func(ctx context.Context) ([]*ScannerStatus, error)

	// User operations
	GetUserFunc    func(ctx context.Context, username string) (*UserStatus, error)
	CreateUserFunc func(ctx context.Context, spec *UserSpec) (*UserStatus, error)
	UpdateUserFunc func(ctx context.Context, username string, spec *UserSpec) (*UserStatus, error)
	DeleteUserFunc func(ctx context.Context, username string) error

	// Registry operations
	CreateRegistryFunc func(ctx context.Context, spec *RegistrySpec) (*RegistryStatus, error)
	GetRegistryFunc    func(ctx context.Context, registryName string) (*RegistryStatus, error)
	UpdateRegistryFunc func(ctx context.Context, registryName string, spec *RegistrySpec) (*RegistryStatus, error)
	DeleteRegistryFunc func(ctx context.Context, registryName string) error

	// Repository operations
	ListRepositoriesFunc func(ctx context.Context, projectID string) ([]*RepositoryStatus, error)
	GetRepositoryFunc    func(ctx context.Context, projectID, repoName string) (*RepositoryStatus, error)
	UpdateRepositoryFunc func(ctx context.Context, projectID, repoName string, spec *RepositorySpec) (*RepositoryStatus, error)
	DeleteRepositoryFunc func(ctx context.Context, projectID, repoName string) error

	// Artifact operations
	ListArtifactsFunc              func(ctx context.Context, projectID, repoName string) ([]*ArtifactStatus, error)
	GetArtifactFunc                func(ctx context.Context, projectID, repoName, reference string) (*ArtifactStatus, error)
	DeleteArtifactFunc             func(ctx context.Context, projectID, repoName, reference string) error
	GetArtifactVulnerabilitiesFunc func(ctx context.Context, projectID, repoName, reference string) (*ArtifactStatus, error)

	// Member operations
	AddProjectMemberFunc        func(ctx context.Context, projectID, username, role string) error
	ListProjectMembersFunc      func(ctx context.Context, projectID string) ([]*MemberStatus, error)
	GetProjectMemberFunc        func(ctx context.Context, projectID, username string) (*MemberStatus, error)
	UpdateProjectMemberFunc     func(ctx context.Context, projectID, username, role string) error
	DeleteProjectMemberFunc     func(ctx context.Context, projectID, username string) error
	AddProjectUserMemberFunc    func(ctx context.Context, projectID, username, role string) (string, error)
	AddProjectGroupMemberFunc   func(ctx context.Context, projectID, groupName string, groupType int64, role string) (string, error)
	FindProjectMemberFunc       func(ctx context.Context, projectID, entityType, entityName string) (*MemberStatus, error)
	GetProjectMemberByIDFunc    func(ctx context.Context, projectID, memberID string) (*MemberStatus, error)
	UpdateProjectMemberByIDFunc func(ctx context.Context, projectID, memberID, role string) error
	DeleteProjectMemberByIDFunc func(ctx context.Context, projectID, memberID string) error

	// Scan operations
	TriggerScanFunc func(ctx context.Context, projectID, repoName, reference string) error
	ListScansFunc   func(ctx context.Context, projectID, repoName string) ([]*ScanStatus, error)
	GetScanFunc     func(ctx context.Context, projectID, repoName, reference string) (*ScanStatus, error)
	StopScanFunc    func(ctx context.Context, projectID, repoName, reference string) error

	// Robot operations
	CreateRobotFunc func(ctx context.Context, spec *RobotSpec) (*RobotStatus, error)
	ListRobotsFunc  func(ctx context.Context, projectID *string) ([]*RobotStatus, error)
	GetRobotFunc    func(ctx context.Context, robotID string) (*RobotStatus, error)
	UpdateRobotFunc func(ctx context.Context, robotID string, spec *RobotSpec) (*RobotStatus, error)
	DeleteRobotFunc func(ctx context.Context, robotID string) error

	// Webhook operations
	CreateWebhookFunc func(ctx context.Context, spec *WebhookSpec) (*WebhookStatus, error)
	ListWebhooksFunc  func(ctx context.Context, projectID string) ([]*WebhookStatus, error)
	GetWebhookFunc    func(ctx context.Context, projectID, webhookID string) (*WebhookStatus, error)
	UpdateWebhookFunc func(ctx context.Context, projectID, webhookID string, spec *WebhookSpec) (*WebhookStatus, error)
	DeleteWebhookFunc func(ctx context.Context, projectID, webhookID string) error

	// Replication operations
	CreateReplicationPolicyFunc   func(ctx context.Context, spec *ReplicationPolicySpec) (*ReplicationPolicyStatus, error)
	ListReplicationPoliciesFunc   func(ctx context.Context) ([]*ReplicationPolicyStatus, error)
	GetReplicationPolicyFunc      func(ctx context.Context, policyID string) (*ReplicationPolicyStatus, error)
	UpdateReplicationPolicyFunc   func(ctx context.Context, policyID string, spec *ReplicationPolicySpec) (*ReplicationPolicyStatus, error)
	DeleteReplicationPolicyFunc   func(ctx context.Context, policyID string) error
	TriggerReplicationFunc        func(ctx context.Context, policyID string) (*ReplicationExecution, error)
	ListReplicationExecutionsFunc func(ctx context.Context, policyID string) ([]*ReplicationExecution, error)

	// Retention operations
	CreateRetentionPolicyFunc func(ctx context.Context, spec *RetentionPolicySpec) (*RetentionPolicyStatus, error)
	ListRetentionPoliciesFunc func(ctx context.Context, projectID string) ([]*RetentionPolicyStatus, error)
	GetRetentionPolicyFunc    func(ctx context.Context, projectID, policyID string) (*RetentionPolicyStatus, error)
	UpdateRetentionPolicyFunc func(ctx context.Context, projectID, policyID string, spec *RetentionPolicySpec) (*RetentionPolicyStatus, error)
	DeleteRetentionPolicyFunc func(ctx context.Context, projectID, policyID string) error

	// UserGroup operations
	CreateUserGroupFunc    func(ctx context.Context, spec *UserGroupSpec) (*UserGroupStatus, error)
	ListUserGroupsFunc     func(ctx context.Context) ([]*UserGroupStatus, error)
	GetUserGroupFunc       func(ctx context.Context, groupID int64) (*UserGroupStatus, error)
	GetUserGroupByNameFunc func(ctx context.Context, name string) (*UserGroupStatus, error)
	UpdateUserGroupFunc    func(ctx context.Context, groupID int64, spec *UserGroupSpec) (*UserGroupStatus, error)
	DeleteUserGroupFunc    func(ctx context.Context, groupID int64) error
}

// GetBaseURL calls GetBaseURLFunc
func (m *MockHarborClient) GetBaseURL() string {
	if m.GetBaseURLFunc != nil {
		return m.GetBaseURLFunc()
	}
	return "https://harbor.example.com"
}

// Close calls CloseFunc
func (m *MockHarborClient) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

// TestConnection calls TestConnectionFunc
func (m *MockHarborClient) TestConnection(ctx context.Context) error {
	if m.TestConnectionFunc != nil {
		return m.TestConnectionFunc(ctx)
	}
	return nil
}

// GetVersion calls GetVersionFunc
func (m *MockHarborClient) GetVersion(ctx context.Context) (string, error) {
	if m.GetVersionFunc != nil {
		return m.GetVersionFunc(ctx)
	}
	return "v2.8.0", nil
}

// GetMemoryFootprint calls GetMemoryFootprintFunc
func (m *MockHarborClient) GetMemoryFootprint() string {
	if m.GetMemoryFootprintFunc != nil {
		return m.GetMemoryFootprintFunc()
	}
	return "mock-memory-footprint"
}

// GetUser calls GetUserFunc
func (m *MockHarborClient) GetUser(ctx context.Context, username string) (*UserStatus, error) {
	if m.GetUserFunc != nil {
		return m.GetUserFunc(ctx, username)
	}
	return nil, nil
}

// CreateUser calls CreateUserFunc
func (m *MockHarborClient) CreateUser(ctx context.Context, spec *UserSpec) (*UserStatus, error) {
	if m.CreateUserFunc != nil {
		return m.CreateUserFunc(ctx, spec)
	}
	return &UserStatus{
		Username:  spec.Username,
		Email:     spec.Email,
		AdminFlag: spec.AdminFlag,
		CreatedAt: time.Now(),
	}, nil
}

// UpdateUser calls UpdateUserFunc
func (m *MockHarborClient) UpdateUser(ctx context.Context, username string, spec *UserSpec) (*UserStatus, error) {
	if m.UpdateUserFunc != nil {
		return m.UpdateUserFunc(ctx, username, spec)
	}
	return &UserStatus{
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
func (m *MockHarborClient) GetProject(ctx context.Context, projectName string) (*ProjectStatus, error) {
	if m.GetProjectFunc != nil {
		return m.GetProjectFunc(ctx, projectName)
	}
	return nil, nil
}

// CreateProject calls CreateProjectFunc
func (m *MockHarborClient) CreateProject(ctx context.Context, spec *ProjectSpec) (*ProjectStatus, error) {
	if m.CreateProjectFunc != nil {
		return m.CreateProjectFunc(ctx, spec)
	}
	return &ProjectStatus{
		Name:   spec.Name,
		Public: spec.Public,
	}, nil
}

// UpdateProject calls UpdateProjectFunc
func (m *MockHarborClient) UpdateProject(ctx context.Context, projectID string, spec *ProjectSpec) (*ProjectStatus, error) {
	if m.UpdateProjectFunc != nil {
		return m.UpdateProjectFunc(ctx, projectID, spec)
	}
	return &ProjectStatus{
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

// ListProjects calls ListProjectsFunc
func (m *MockHarborClient) ListProjects(ctx context.Context) ([]*ProjectStatus, error) {
	if m.ListProjectsFunc != nil {
		return m.ListProjectsFunc(ctx)
	}
	return nil, nil
}

// CreateScannerRegistration calls CreateScannerRegistrationFunc
func (m *MockHarborClient) CreateScannerRegistration(ctx context.Context, spec *ScannerSpec) (*ScannerStatus, error) {
	if m.CreateScannerRegistrationFunc != nil {
		return m.CreateScannerRegistrationFunc(ctx, spec)
	}
	return &ScannerStatus{
		UUID:       "mock-scanner-uuid",
		Name:       spec.Name,
		URL:        spec.URL,
		CreateTime: time.Now(),
		UpdateTime: time.Now(),
	}, nil
}

// GetScannerRegistration calls GetScannerRegistrationFunc
func (m *MockHarborClient) GetScannerRegistration(ctx context.Context, scannerID string) (*ScannerStatus, error) {
	if m.GetScannerRegistrationFunc != nil {
		return m.GetScannerRegistrationFunc(ctx, scannerID)
	}
	return nil, nil
}

// UpdateScannerRegistration calls UpdateScannerRegistrationFunc
func (m *MockHarborClient) UpdateScannerRegistration(ctx context.Context, scannerID string, spec *ScannerSpec) (*ScannerStatus, error) {
	if m.UpdateScannerRegistrationFunc != nil {
		return m.UpdateScannerRegistrationFunc(ctx, scannerID, spec)
	}
	return &ScannerStatus{
		UUID:       scannerID,
		Name:       spec.Name,
		URL:        spec.URL,
		CreateTime: time.Now(),
		UpdateTime: time.Now(),
	}, nil
}

// DeleteScannerRegistration calls DeleteScannerRegistrationFunc
func (m *MockHarborClient) DeleteScannerRegistration(ctx context.Context, scannerID string) error {
	if m.DeleteScannerRegistrationFunc != nil {
		return m.DeleteScannerRegistrationFunc(ctx, scannerID)
	}
	return nil
}

// ListScannerRegistrations calls ListScannerRegistrationsFunc
func (m *MockHarborClient) ListScannerRegistrations(ctx context.Context) ([]*ScannerStatus, error) {
	if m.ListScannerRegistrationsFunc != nil {
		return m.ListScannerRegistrationsFunc(ctx)
	}
	return nil, nil
}

// CreateRegistry calls CreateRegistryFunc
func (m *MockHarborClient) CreateRegistry(ctx context.Context, spec *RegistrySpec) (*RegistryStatus, error) {
	if m.CreateRegistryFunc != nil {
		return m.CreateRegistryFunc(ctx, spec)
	}
	return &RegistryStatus{
		Name:      spec.Name,
		Type:      spec.Type,
		URL:       spec.URL,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

// GetRegistry calls GetRegistryFunc
func (m *MockHarborClient) GetRegistry(ctx context.Context, registryName string) (*RegistryStatus, error) {
	if m.GetRegistryFunc != nil {
		return m.GetRegistryFunc(ctx, registryName)
	}
	return nil, nil
}

// UpdateRegistry calls UpdateRegistryFunc
func (m *MockHarborClient) UpdateRegistry(ctx context.Context, registryName string, spec *RegistrySpec) (*RegistryStatus, error) {
	if m.UpdateRegistryFunc != nil {
		return m.UpdateRegistryFunc(ctx, registryName, spec)
	}
	return &RegistryStatus{
		Name:      spec.Name,
		Type:      spec.Type,
		URL:       spec.URL,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

// DeleteRegistry calls DeleteRegistryFunc
func (m *MockHarborClient) DeleteRegistry(ctx context.Context, registryName string) error {
	if m.DeleteRegistryFunc != nil {
		return m.DeleteRegistryFunc(ctx, registryName)
	}
	return nil
}

// ListRepositories calls ListRepositoriesFunc
func (m *MockHarborClient) ListRepositories(ctx context.Context, projectID string) ([]*RepositoryStatus, error) {
	if m.ListRepositoriesFunc != nil {
		return m.ListRepositoriesFunc(ctx, projectID)
	}
	return nil, nil
}

// GetRepository calls GetRepositoryFunc
func (m *MockHarborClient) GetRepository(ctx context.Context, projectID, repoName string) (*RepositoryStatus, error) {
	if m.GetRepositoryFunc != nil {
		return m.GetRepositoryFunc(ctx, projectID, repoName)
	}
	return nil, nil
}

// UpdateRepository calls UpdateRepositoryFunc
func (m *MockHarborClient) UpdateRepository(ctx context.Context, projectID, repoName string, spec *RepositorySpec) (*RepositoryStatus, error) {
	if m.UpdateRepositoryFunc != nil {
		return m.UpdateRepositoryFunc(ctx, projectID, repoName, spec)
	}
	return &RepositoryStatus{
		ID:            "1",
		FullName:      projectID + "/" + repoName,
		ProjectID:     projectID,
		ArtifactCount: 0,
		CreationTime:  time.Now(),
		UpdateTime:    time.Now(),
	}, nil
}

// DeleteRepository calls DeleteRepositoryFunc
func (m *MockHarborClient) DeleteRepository(ctx context.Context, projectID, repoName string) error {
	if m.DeleteRepositoryFunc != nil {
		return m.DeleteRepositoryFunc(ctx, projectID, repoName)
	}
	return nil
}

// ListArtifacts calls ListArtifactsFunc
func (m *MockHarborClient) ListArtifacts(ctx context.Context, projectID, repoName string) ([]*ArtifactStatus, error) {
	if m.ListArtifactsFunc != nil {
		return m.ListArtifactsFunc(ctx, projectID, repoName)
	}
	return nil, nil
}

// GetArtifact calls GetArtifactFunc
func (m *MockHarborClient) GetArtifact(ctx context.Context, projectID, repoName, reference string) (*ArtifactStatus, error) {
	if m.GetArtifactFunc != nil {
		return m.GetArtifactFunc(ctx, projectID, repoName, reference)
	}
	return nil, nil
}

// DeleteArtifact calls DeleteArtifactFunc
func (m *MockHarborClient) DeleteArtifact(ctx context.Context, projectID, repoName, reference string) error {
	if m.DeleteArtifactFunc != nil {
		return m.DeleteArtifactFunc(ctx, projectID, repoName, reference)
	}
	return nil
}

// GetArtifactVulnerabilities calls GetArtifactVulnerabilitiesFunc
func (m *MockHarborClient) GetArtifactVulnerabilities(ctx context.Context, projectID, repoName, reference string) (*ArtifactStatus, error) {
	if m.GetArtifactVulnerabilitiesFunc != nil {
		return m.GetArtifactVulnerabilitiesFunc(ctx, projectID, repoName, reference)
	}
	return nil, nil
}

// AddProjectMember calls AddProjectMemberFunc
func (m *MockHarborClient) AddProjectMember(ctx context.Context, projectID, username, role string) error {
	if m.AddProjectMemberFunc != nil {
		return m.AddProjectMemberFunc(ctx, projectID, username, role)
	}
	return nil
}

// ListProjectMembers calls ListProjectMembersFunc
func (m *MockHarborClient) ListProjectMembers(ctx context.Context, projectID string) ([]*MemberStatus, error) {
	if m.ListProjectMembersFunc != nil {
		return m.ListProjectMembersFunc(ctx, projectID)
	}
	return nil, nil
}

// GetProjectMember calls GetProjectMemberFunc
func (m *MockHarborClient) GetProjectMember(ctx context.Context, projectID, username string) (*MemberStatus, error) {
	if m.GetProjectMemberFunc != nil {
		return m.GetProjectMemberFunc(ctx, projectID, username)
	}
	return nil, nil
}

// UpdateProjectMember calls UpdateProjectMemberFunc
func (m *MockHarborClient) UpdateProjectMember(ctx context.Context, projectID, username, role string) error {
	if m.UpdateProjectMemberFunc != nil {
		return m.UpdateProjectMemberFunc(ctx, projectID, username, role)
	}
	return nil
}

// DeleteProjectMember calls DeleteProjectMemberFunc
func (m *MockHarborClient) DeleteProjectMember(ctx context.Context, projectID, username string) error {
	if m.DeleteProjectMemberFunc != nil {
		return m.DeleteProjectMemberFunc(ctx, projectID, username)
	}
	return nil
}

// AddProjectUserMember calls AddProjectUserMemberFunc
func (m *MockHarborClient) AddProjectUserMember(ctx context.Context, projectID, username, role string) (string, error) {
	if m.AddProjectUserMemberFunc != nil {
		return m.AddProjectUserMemberFunc(ctx, projectID, username, role)
	}
	return "", nil
}

// AddProjectGroupMember calls AddProjectGroupMemberFunc
func (m *MockHarborClient) AddProjectGroupMember(ctx context.Context, projectID, groupName string, groupType int64, role string) (string, error) {
	if m.AddProjectGroupMemberFunc != nil {
		return m.AddProjectGroupMemberFunc(ctx, projectID, groupName, groupType, role)
	}
	return "", nil
}

// FindProjectMember calls FindProjectMemberFunc
func (m *MockHarborClient) FindProjectMember(ctx context.Context, projectID, entityType, entityName string) (*MemberStatus, error) {
	if m.FindProjectMemberFunc != nil {
		return m.FindProjectMemberFunc(ctx, projectID, entityType, entityName)
	}
	return nil, nil
}

// GetProjectMemberByID calls GetProjectMemberByIDFunc
func (m *MockHarborClient) GetProjectMemberByID(ctx context.Context, projectID, memberID string) (*MemberStatus, error) {
	if m.GetProjectMemberByIDFunc != nil {
		return m.GetProjectMemberByIDFunc(ctx, projectID, memberID)
	}
	return nil, nil
}

// UpdateProjectMemberByID calls UpdateProjectMemberByIDFunc
func (m *MockHarborClient) UpdateProjectMemberByID(ctx context.Context, projectID, memberID, role string) error {
	if m.UpdateProjectMemberByIDFunc != nil {
		return m.UpdateProjectMemberByIDFunc(ctx, projectID, memberID, role)
	}
	return nil
}

// DeleteProjectMemberByID calls DeleteProjectMemberByIDFunc
func (m *MockHarborClient) DeleteProjectMemberByID(ctx context.Context, projectID, memberID string) error {
	if m.DeleteProjectMemberByIDFunc != nil {
		return m.DeleteProjectMemberByIDFunc(ctx, projectID, memberID)
	}
	return nil
}

// TriggerScan calls TriggerScanFunc
func (m *MockHarborClient) TriggerScan(ctx context.Context, projectID, repoName, reference string) error {
	if m.TriggerScanFunc != nil {
		return m.TriggerScanFunc(ctx, projectID, repoName, reference)
	}
	return nil
}

// ListScans calls ListScansFunc
func (m *MockHarborClient) ListScans(ctx context.Context, projectID, repoName string) ([]*ScanStatus, error) {
	if m.ListScansFunc != nil {
		return m.ListScansFunc(ctx, projectID, repoName)
	}
	return nil, nil
}

// GetScan calls GetScanFunc
func (m *MockHarborClient) GetScan(ctx context.Context, projectID, repoName, reference string) (*ScanStatus, error) {
	if m.GetScanFunc != nil {
		return m.GetScanFunc(ctx, projectID, repoName, reference)
	}
	return nil, nil
}

// StopScan calls StopScanFunc
func (m *MockHarborClient) StopScan(ctx context.Context, projectID, repoName, reference string) error {
	if m.StopScanFunc != nil {
		return m.StopScanFunc(ctx, projectID, repoName, reference)
	}
	return nil
}

// CreateRobot calls CreateRobotFunc
func (m *MockHarborClient) CreateRobot(ctx context.Context, spec *RobotSpec) (*RobotStatus, error) {
	if m.CreateRobotFunc != nil {
		return m.CreateRobotFunc(ctx, spec)
	}
	return &RobotStatus{
		ID:           "mock-robot-id",
		Name:         spec.Name,
		Description:  spec.Description,
		ProjectID:    spec.ProjectID,
		Secret:       "mock-secret-token",
		CreationTime: time.Now(),
		UpdateTime:   time.Now(),
	}, nil
}

// ListRobots calls ListRobotsFunc
func (m *MockHarborClient) ListRobots(ctx context.Context, projectID *string) ([]*RobotStatus, error) {
	if m.ListRobotsFunc != nil {
		return m.ListRobotsFunc(ctx, projectID)
	}
	return nil, nil
}

// GetRobot calls GetRobotFunc
func (m *MockHarborClient) GetRobot(ctx context.Context, robotID string) (*RobotStatus, error) {
	if m.GetRobotFunc != nil {
		return m.GetRobotFunc(ctx, robotID)
	}
	return nil, nil
}

// UpdateRobot calls UpdateRobotFunc
func (m *MockHarborClient) UpdateRobot(ctx context.Context, robotID string, spec *RobotSpec) (*RobotStatus, error) {
	if m.UpdateRobotFunc != nil {
		return m.UpdateRobotFunc(ctx, robotID, spec)
	}
	return &RobotStatus{
		ID:           robotID,
		Name:         spec.Name,
		Description:  spec.Description,
		ProjectID:    spec.ProjectID,
		CreationTime: time.Now(),
		UpdateTime:   time.Now(),
	}, nil
}

// DeleteRobot calls DeleteRobotFunc
func (m *MockHarborClient) DeleteRobot(ctx context.Context, robotID string) error {
	if m.DeleteRobotFunc != nil {
		return m.DeleteRobotFunc(ctx, robotID)
	}
	return nil
}

// CreateWebhook calls CreateWebhookFunc
func (m *MockHarborClient) CreateWebhook(ctx context.Context, spec *WebhookSpec) (*WebhookStatus, error) {
	if m.CreateWebhookFunc != nil {
		return m.CreateWebhookFunc(ctx, spec)
	}
	return &WebhookStatus{
		ID:           "mock-webhook-id",
		ProjectID:    spec.ProjectID,
		Name:         spec.Name,
		Description:  spec.Description,
		URL:          spec.URL,
		EventTypes:   spec.EventTypes,
		CreationTime: time.Now(),
		UpdateTime:   time.Now(),
	}, nil
}

// ListWebhooks calls ListWebhooksFunc
func (m *MockHarborClient) ListWebhooks(ctx context.Context, projectID string) ([]*WebhookStatus, error) {
	if m.ListWebhooksFunc != nil {
		return m.ListWebhooksFunc(ctx, projectID)
	}
	return nil, nil
}

// GetWebhook calls GetWebhookFunc
func (m *MockHarborClient) GetWebhook(ctx context.Context, projectID, webhookID string) (*WebhookStatus, error) {
	if m.GetWebhookFunc != nil {
		return m.GetWebhookFunc(ctx, projectID, webhookID)
	}
	return nil, nil
}

// UpdateWebhook calls UpdateWebhookFunc
func (m *MockHarborClient) UpdateWebhook(ctx context.Context, projectID, webhookID string, spec *WebhookSpec) (*WebhookStatus, error) {
	if m.UpdateWebhookFunc != nil {
		return m.UpdateWebhookFunc(ctx, projectID, webhookID, spec)
	}
	return &WebhookStatus{
		ID:           webhookID,
		ProjectID:    projectID,
		Name:         spec.Name,
		Description:  spec.Description,
		URL:          spec.URL,
		EventTypes:   spec.EventTypes,
		CreationTime: time.Now(),
		UpdateTime:   time.Now(),
	}, nil
}

// DeleteWebhook calls DeleteWebhookFunc
func (m *MockHarborClient) DeleteWebhook(ctx context.Context, projectID, webhookID string) error {
	if m.DeleteWebhookFunc != nil {
		return m.DeleteWebhookFunc(ctx, projectID, webhookID)
	}
	return nil
}

// CreateReplicationPolicy calls CreateReplicationPolicyFunc
func (m *MockHarborClient) CreateReplicationPolicy(ctx context.Context, spec *ReplicationPolicySpec) (*ReplicationPolicyStatus, error) {
	if m.CreateReplicationPolicyFunc != nil {
		return m.CreateReplicationPolicyFunc(ctx, spec)
	}
	return &ReplicationPolicyStatus{
		ID:           "mock-policy-id",
		Name:         spec.Name,
		Description:  spec.Description,
		Enabled:      spec.Enabled != nil && *spec.Enabled,
		CreationTime: time.Now(),
		UpdateTime:   time.Now(),
	}, nil
}

// ListReplicationPolicies calls ListReplicationPoliciesFunc
func (m *MockHarborClient) ListReplicationPolicies(ctx context.Context) ([]*ReplicationPolicyStatus, error) {
	if m.ListReplicationPoliciesFunc != nil {
		return m.ListReplicationPoliciesFunc(ctx)
	}
	return nil, nil
}

// GetReplicationPolicy calls GetReplicationPolicyFunc
func (m *MockHarborClient) GetReplicationPolicy(ctx context.Context, policyID string) (*ReplicationPolicyStatus, error) {
	if m.GetReplicationPolicyFunc != nil {
		return m.GetReplicationPolicyFunc(ctx, policyID)
	}
	return nil, nil
}

// UpdateReplicationPolicy calls UpdateReplicationPolicyFunc
func (m *MockHarborClient) UpdateReplicationPolicy(ctx context.Context, policyID string, spec *ReplicationPolicySpec) (*ReplicationPolicyStatus, error) {
	if m.UpdateReplicationPolicyFunc != nil {
		return m.UpdateReplicationPolicyFunc(ctx, policyID, spec)
	}
	return &ReplicationPolicyStatus{
		ID:           policyID,
		Name:         spec.Name,
		Description:  spec.Description,
		Enabled:      spec.Enabled != nil && *spec.Enabled,
		CreationTime: time.Now(),
		UpdateTime:   time.Now(),
	}, nil
}

// DeleteReplicationPolicy calls DeleteReplicationPolicyFunc
func (m *MockHarborClient) DeleteReplicationPolicy(ctx context.Context, policyID string) error {
	if m.DeleteReplicationPolicyFunc != nil {
		return m.DeleteReplicationPolicyFunc(ctx, policyID)
	}
	return nil
}

// TriggerReplication calls TriggerReplicationFunc
func (m *MockHarborClient) TriggerReplication(ctx context.Context, policyID string) (*ReplicationExecution, error) {
	if m.TriggerReplicationFunc != nil {
		return m.TriggerReplicationFunc(ctx, policyID)
	}
	return &ReplicationExecution{
		ID:        "mock-execution-id",
		PolicyID:  policyID,
		Status:    "pending",
		StartTime: time.Now(),
	}, nil
}

// ListReplicationExecutions calls ListReplicationExecutionsFunc
func (m *MockHarborClient) ListReplicationExecutions(ctx context.Context, policyID string) ([]*ReplicationExecution, error) {
	if m.ListReplicationExecutionsFunc != nil {
		return m.ListReplicationExecutionsFunc(ctx, policyID)
	}
	return nil, nil
}

// CreateRetentionPolicy calls CreateRetentionPolicyFunc
func (m *MockHarborClient) CreateRetentionPolicy(ctx context.Context, spec *RetentionPolicySpec) (*RetentionPolicyStatus, error) {
	if m.CreateRetentionPolicyFunc != nil {
		return m.CreateRetentionPolicyFunc(ctx, spec)
	}
	return &RetentionPolicyStatus{
		ID:           "mock-retention-id",
		ProjectID:    spec.ProjectID,
		Description:  spec.Description,
		Enabled:      spec.Enabled != nil && *spec.Enabled,
		CreationTime: time.Now(),
		UpdateTime:   time.Now(),
	}, nil
}

// ListRetentionPolicies calls ListRetentionPoliciesFunc
func (m *MockHarborClient) ListRetentionPolicies(ctx context.Context, projectID string) ([]*RetentionPolicyStatus, error) {
	if m.ListRetentionPoliciesFunc != nil {
		return m.ListRetentionPoliciesFunc(ctx, projectID)
	}
	return nil, nil
}

// GetRetentionPolicy calls GetRetentionPolicyFunc
func (m *MockHarborClient) GetRetentionPolicy(ctx context.Context, projectID, policyID string) (*RetentionPolicyStatus, error) {
	if m.GetRetentionPolicyFunc != nil {
		return m.GetRetentionPolicyFunc(ctx, projectID, policyID)
	}
	return nil, nil
}

// UpdateRetentionPolicy calls UpdateRetentionPolicyFunc
func (m *MockHarborClient) UpdateRetentionPolicy(ctx context.Context, projectID, policyID string, spec *RetentionPolicySpec) (*RetentionPolicyStatus, error) {
	if m.UpdateRetentionPolicyFunc != nil {
		return m.UpdateRetentionPolicyFunc(ctx, projectID, policyID, spec)
	}
	return &RetentionPolicyStatus{
		ID:           policyID,
		ProjectID:    projectID,
		Description:  spec.Description,
		Enabled:      spec.Enabled != nil && *spec.Enabled,
		CreationTime: time.Now(),
		UpdateTime:   time.Now(),
	}, nil
}

// DeleteRetentionPolicy calls DeleteRetentionPolicyFunc
func (m *MockHarborClient) DeleteRetentionPolicy(ctx context.Context, projectID, policyID string) error {
	if m.DeleteRetentionPolicyFunc != nil {
		return m.DeleteRetentionPolicyFunc(ctx, projectID, policyID)
	}
	return nil
}

// CreateUserGroup calls CreateUserGroupFunc
func (m *MockHarborClient) CreateUserGroup(ctx context.Context, spec *UserGroupSpec) (*UserGroupStatus, error) {
	if m.CreateUserGroupFunc != nil {
		return m.CreateUserGroupFunc(ctx, spec)
	}
	return &UserGroupStatus{
		ID:          1,
		GroupName:   spec.GroupName,
		GroupType:   spec.GroupType,
		LdapGroupDn: "",
	}, nil
}

// ListUserGroups calls ListUserGroupsFunc
func (m *MockHarborClient) ListUserGroups(ctx context.Context) ([]*UserGroupStatus, error) {
	if m.ListUserGroupsFunc != nil {
		return m.ListUserGroupsFunc(ctx)
	}
	return nil, nil
}

// GetUserGroupByName calls GetUserGroupByNameFunc
func (m *MockHarborClient) GetUserGroupByName(ctx context.Context, name string) (*UserGroupStatus, error) {
	if m.GetUserGroupByNameFunc != nil {
		return m.GetUserGroupByNameFunc(ctx, name)
	}
	return nil, nil
}

// GetUserGroup calls GetUserGroupFunc
func (m *MockHarborClient) GetUserGroup(ctx context.Context, groupID int64) (*UserGroupStatus, error) {
	if m.GetUserGroupFunc != nil {
		return m.GetUserGroupFunc(ctx, groupID)
	}
	return nil, nil
}

// UpdateUserGroup calls UpdateUserGroupFunc
func (m *MockHarborClient) UpdateUserGroup(ctx context.Context, groupID int64, spec *UserGroupSpec) (*UserGroupStatus, error) {
	if m.UpdateUserGroupFunc != nil {
		return m.UpdateUserGroupFunc(ctx, groupID, spec)
	}
	return &UserGroupStatus{
		ID:          groupID,
		GroupName:   spec.GroupName,
		GroupType:   spec.GroupType,
		LdapGroupDn: "",
	}, nil
}

// DeleteUserGroup calls DeleteUserGroupFunc
func (m *MockHarborClient) DeleteUserGroup(ctx context.Context, groupID int64) error {
	if m.DeleteUserGroupFunc != nil {
		return m.DeleteUserGroupFunc(ctx, groupID)
	}
	return nil
}
