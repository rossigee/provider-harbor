/*
Copyright 2024 Crossplane Harbor Provider.
*/

package member

import (
	"context"
	"errors"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rossigee/provider-harbor/apis/member/v1beta1"
	harborclients "github.com/rossigee/provider-harbor/internal/clients"
)

func TestConnectNotMember(t *testing.T) {
	ctx := context.Background()
	conn := &connector{}

	_, err := conn.Connect(ctx, nil)
	if err == nil || err.Error() != errNotMember {
		t.Errorf("Connect with nil should return %s error", errNotMember)
	}
}

func TestObserveNotMember(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Observe(ctx, nil)
	if err == nil || err.Error() != errNotMember {
		t.Errorf("Observe with nil should return %s error", errNotMember)
	}
}

func TestUpdateNotMember(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Update(ctx, nil)
	if err == nil || err.Error() != errNotMember {
		t.Errorf("Update with nil should return %s error", errNotMember)
	}
}

func TestDeleteNotMember(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Delete(ctx, nil)
	if err == nil || err.Error() != errNotMember {
		t.Errorf("Delete with nil should return %s error", errNotMember)
	}
}

func TestCreateNotMember(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Create(ctx, nil)
	if err == nil || err.Error() != errNotMember {
		t.Errorf("Create with nil should return %s error", errNotMember)
	}
}

func TestObserveMemberNotFound(t *testing.T) {
	ctx := context.Background()
	member := &v1beta1.Member{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-member",
		},
		Spec: v1beta1.MemberSpec{
			ForProvider: v1beta1.MemberParameters{
				ProjectID: "project-1",
				Username:  "testuser",
			},
		},
	}

	ext := &external{
		service: &mockMemberClient{
			getProjectMemberFunc: func(ctx context.Context, projectID, username string) (*harborclients.MemberStatus, error) {
				return nil, errors.New("not found")
			},
		},
	}

	obs, err := ext.Observe(ctx, member)
	if err == nil {
		t.Error("Observe should fail when client returns error")
	}
	if obs.ResourceExists {
		t.Error("ResourceExists should be false when member not found")
	}
}

func TestObserveMemberExists(t *testing.T) {
	ctx := context.Background()
	member := &v1beta1.Member{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-member",
		},
		Spec: v1beta1.MemberSpec{
			ForProvider: v1beta1.MemberParameters{
				ProjectID: "project-1",
				Username:  "testuser",
				Role:      "admin",
			},
		},
	}

	ext := &external{
		service: &mockMemberClient{
			getProjectMemberFunc: func(ctx context.Context, projectID, username string) (*harborclients.MemberStatus, error) {
				return &harborclients.MemberStatus{
					ID:           "member-123",
					MemberName:   "testuser",
					MemberType:   "u",
					Role:         "admin",
					CreationTime: time.Now(),
				}, nil
			},
		},
	}

	obs, err := ext.Observe(ctx, member)
	if err != nil {
		t.Errorf("Observe should not fail, got %v", err)
	}
	if !obs.ResourceExists {
		t.Error("ResourceExists should be true")
	}
	if !obs.ResourceUpToDate {
		t.Error("ResourceUpToDate should be true when role matches")
	}
}

func TestObserveMemberNotUpToDate(t *testing.T) {
	ctx := context.Background()
	member := &v1beta1.Member{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-member",
		},
		Spec: v1beta1.MemberSpec{
			ForProvider: v1beta1.MemberParameters{
				ProjectID: "project-1",
				Username:  "testuser",
				Role:      "admin",
			},
		},
	}

	ext := &external{
		service: &mockMemberClient{
			getProjectMemberFunc: func(ctx context.Context, projectID, username string) (*harborclients.MemberStatus, error) {
				return &harborclients.MemberStatus{
					ID:           "member-123",
					MemberName:   "testuser",
					MemberType:   "u",
					Role:         "developer",
					CreationTime: time.Now(),
				}, nil
			},
		},
	}

	obs, err := ext.Observe(ctx, member)
	if err != nil {
		t.Errorf("Observe should not fail, got %v", err)
	}
	if !obs.ResourceExists {
		t.Error("ResourceExists should be true")
	}
	if obs.ResourceUpToDate {
		t.Error("ResourceUpToDate should be false when role differs")
	}
}

func TestObserveMemberNoRoleInSpec(t *testing.T) {
	ctx := context.Background()
	member := &v1beta1.Member{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-member",
		},
		Spec: v1beta1.MemberSpec{
			ForProvider: v1beta1.MemberParameters{
				ProjectID: "project-1",
				Username:  "testuser",
			},
		},
	}

	ext := &external{
		service: &mockMemberClient{
			getProjectMemberFunc: func(ctx context.Context, projectID, username string) (*harborclients.MemberStatus, error) {
				return &harborclients.MemberStatus{
					ID:           "member-123",
					MemberName:   "testuser",
					MemberType:   "u",
					Role:         "developer",
					CreationTime: time.Now(),
				}, nil
			},
		},
	}

	obs, err := ext.Observe(ctx, member)
	if err != nil {
		t.Errorf("Observe should not fail, got %v", err)
	}
	if !obs.ResourceExists {
		t.Error("ResourceExists should be true")
	}
	if !obs.ResourceUpToDate {
		t.Error("ResourceUpToDate should be true when no role specified in spec")
	}
}

func TestCreateMemberSuccess(t *testing.T) {
	ctx := context.Background()
	member := &v1beta1.Member{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-member",
		},
		Spec: v1beta1.MemberSpec{
			ForProvider: v1beta1.MemberParameters{
				ProjectID: "project-1",
				Username:  "testuser",
				Role:      "admin",
			},
		},
	}

	ext := &external{
		service: &mockMemberClient{
			addProjectMemberFunc: func(ctx context.Context, projectID, username, role string) error {
				return nil
			},
		},
	}

	_, err := ext.Create(ctx, member)
	if err != nil {
		t.Errorf("Create should not fail, got %v", err)
	}
}

func TestCreateMemberError(t *testing.T) {
	ctx := context.Background()
	member := &v1beta1.Member{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-member",
		},
		Spec: v1beta1.MemberSpec{
			ForProvider: v1beta1.MemberParameters{
				ProjectID: "project-1",
				Username:  "testuser",
				Role:      "admin",
			},
		},
	}

	ext := &external{
		service: &mockMemberClient{
			addProjectMemberFunc: func(ctx context.Context, projectID, username, role string) error {
				return errors.New("create failed")
			},
		},
	}

	_, err := ext.Create(ctx, member)
	if err == nil {
		t.Error("Create should fail when client fails")
	}
}

func TestUpdateMemberSuccess(t *testing.T) {
	ctx := context.Background()
	member := &v1beta1.Member{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-member",
		},
		Spec: v1beta1.MemberSpec{
			ForProvider: v1beta1.MemberParameters{
				ProjectID: "project-1",
				Username:  "testuser",
				Role:      "developer",
			},
		},
	}

	ext := &external{
		service: &mockMemberClient{
			updateProjectMemberFunc: func(ctx context.Context, projectID, username, role string) error {
				return nil
			},
		},
	}

	_, err := ext.Update(ctx, member)
	if err != nil {
		t.Errorf("Update should not fail, got %v", err)
	}
}

func TestUpdateMemberError(t *testing.T) {
	ctx := context.Background()
	member := &v1beta1.Member{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-member",
		},
		Spec: v1beta1.MemberSpec{
			ForProvider: v1beta1.MemberParameters{
				ProjectID: "project-1",
				Username:  "testuser",
				Role:      "developer",
			},
		},
	}

	ext := &external{
		service: &mockMemberClient{
			updateProjectMemberFunc: func(ctx context.Context, projectID, username, role string) error {
				return errors.New("update failed")
			},
		},
	}

	_, err := ext.Update(ctx, member)
	if err == nil {
		t.Error("Update should fail when client fails")
	}
}

func TestDeleteMemberSuccess(t *testing.T) {
	ctx := context.Background()
	member := &v1beta1.Member{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-member",
		},
		Spec: v1beta1.MemberSpec{
			ForProvider: v1beta1.MemberParameters{
				ProjectID: "project-1",
				Username:  "testuser",
			},
		},
	}

	ext := &external{
		service: &mockMemberClient{
			deleteProjectMemberFunc: func(ctx context.Context, projectID, username string) error {
				return nil
			},
		},
	}

	_, err := ext.Delete(ctx, member)
	if err != nil {
		t.Errorf("Delete should not fail, got %v", err)
	}
}

func TestDeleteMemberError(t *testing.T) {
	ctx := context.Background()
	member := &v1beta1.Member{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-member",
		},
		Spec: v1beta1.MemberSpec{
			ForProvider: v1beta1.MemberParameters{
				ProjectID: "project-1",
				Username:  "testuser",
			},
		},
	}

	ext := &external{
		service: &mockMemberClient{
			deleteProjectMemberFunc: func(ctx context.Context, projectID, username string) error {
				return errors.New("delete failed")
			},
		},
	}

	_, err := ext.Delete(ctx, member)
	if err == nil {
		t.Error("Delete should fail when client fails")
	}
}

func TestMemberHasRequiredFields(t *testing.T) {
	member := &v1beta1.Member{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-member",
			Namespace: "default",
		},
		Spec: v1beta1.MemberSpec{
			ForProvider: v1beta1.MemberParameters{
				ProjectID: "project-1",
				Username:  "testuser",
				Role:      "admin",
			},
		},
	}

	if member.Spec.ForProvider.ProjectID == "" {
		t.Error("Member ProjectID should not be empty")
	}
	if member.Spec.ForProvider.Username == "" {
		t.Error("Member Username should not be empty")
	}
	if member.Spec.ForProvider.Role == "" {
		t.Error("Member Role should not be empty")
	}
	if member.Name == "" {
		t.Error("Metadata Name should not be empty")
	}
}

func TestMemberStatusFields(t *testing.T) {
	member := &v1beta1.Member{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-member",
		},
		Spec: v1beta1.MemberSpec{
			ForProvider: v1beta1.MemberParameters{
				ProjectID: "project-1",
				Username:  "testuser",
				Role:      "admin",
			},
		},
		Status: v1beta1.MemberStatus{
			AtProvider: v1beta1.MemberObservation{
				ID:         ptrString("member-123"),
				MemberName: ptrString("testuser"),
				MemberType: ptrString("u"),
				Role:       ptrString("admin"),
			},
		},
	}

	if member.Status.AtProvider.ID == nil {
		t.Error("Status ID should be populated")
	}
	if *member.Status.AtProvider.ID != "member-123" {
		t.Errorf("Status ID should be 'member-123', got %s", *member.Status.AtProvider.ID)
	}
}

func TestMemberParametersValidation(t *testing.T) {
	tests := []struct {
		name    string
		params  v1beta1.MemberParameters
		isValid bool
	}{
		{
			name: "valid with all required fields",
			params: v1beta1.MemberParameters{
				ProjectID: "project-1",
				Username:  "testuser",
				Role:      "admin",
			},
			isValid: true,
		},
		{
			name: "valid with developer role",
			params: v1beta1.MemberParameters{
				ProjectID: "project-1",
				Username:  "testuser",
				Role:      "developer",
			},
			isValid: true,
		},
		{
			name: "valid with guest role",
			params: v1beta1.MemberParameters{
				ProjectID: "project-1",
				Username:  "testuser",
				Role:      "guest",
			},
			isValid: true,
		},
		{
			name: "missing required project ID",
			params: v1beta1.MemberParameters{
				Username: "testuser",
				Role:     "admin",
			},
			isValid: false,
		},
		{
			name: "missing required username",
			params: v1beta1.MemberParameters{
				ProjectID: "project-1",
				Role:      "admin",
			},
			isValid: false,
		},
		{
			name: "missing required role",
			params: v1beta1.MemberParameters{
				ProjectID: "project-1",
				Username:  "testuser",
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.params.ProjectID != "" && tt.params.Username != "" && tt.params.Role != ""
			if isValid != tt.isValid {
				t.Errorf("Expected valid=%v, got %v", tt.isValid, isValid)
			}
		})
	}
}

func TestDisconnect(t *testing.T) {
	ctx := context.Background()
	ext := &external{
		service: &mockMemberClient{},
	}

	err := ext.Disconnect(ctx)
	if err != nil {
		t.Errorf("Disconnect should not fail, got %v", err)
	}
}

type mockMemberClient struct {
	harborclients.HarborClienter
	getProjectMemberFunc    func(ctx context.Context, projectID, username string) (*harborclients.MemberStatus, error)
	addProjectMemberFunc    func(ctx context.Context, projectID, username, role string) error
	updateProjectMemberFunc func(ctx context.Context, projectID, username, role string) error
	deleteProjectMemberFunc func(ctx context.Context, projectID, username string) error
	listProjectMembersFunc  func(ctx context.Context, projectID string) ([]*harborclients.MemberStatus, error)
}

func (m *mockMemberClient) GetProjectMember(ctx context.Context, projectID, username string) (*harborclients.MemberStatus, error) {
	if m.getProjectMemberFunc != nil {
		return m.getProjectMemberFunc(ctx, projectID, username)
	}
	return nil, nil
}

func (m *mockMemberClient) AddProjectMember(ctx context.Context, projectID, username, role string) error {
	if m.addProjectMemberFunc != nil {
		return m.addProjectMemberFunc(ctx, projectID, username, role)
	}
	return nil
}

func (m *mockMemberClient) UpdateProjectMember(ctx context.Context, projectID, username, role string) error {
	if m.updateProjectMemberFunc != nil {
		return m.updateProjectMemberFunc(ctx, projectID, username, role)
	}
	return nil
}

func (m *mockMemberClient) DeleteProjectMember(ctx context.Context, projectID, username string) error {
	if m.deleteProjectMemberFunc != nil {
		return m.deleteProjectMemberFunc(ctx, projectID, username)
	}
	return nil
}

func (m *mockMemberClient) ListProjectMembers(ctx context.Context, projectID string) ([]*harborclients.MemberStatus, error) {
	if m.listProjectMembersFunc != nil {
		return m.listProjectMembersFunc(ctx, projectID)
	}
	return nil, nil
}

func (m *mockMemberClient) Close() error {
	return nil
}

func (m *mockMemberClient) GetBaseURL() string {
	return "https://harbor.example.com"
}

func ptrString(s string) *string {
	return &s
}
