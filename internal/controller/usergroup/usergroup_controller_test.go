/*
Copyright 2024 Crossplane Harbor Provider.
*/

package usergroup

import (
	"context"
	"errors"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rossigee/provider-harbor/apis/usergroup/v1beta1"
	harborclients "github.com/rossigee/provider-harbor/internal/clients"
)

func TestConnectNotUserGroup(t *testing.T) {
	ctx := context.Background()
	conn := &connector{}

	_, err := conn.Connect(ctx, nil)
	if err == nil || err.Error() != errNotUserGroup {
		t.Errorf("Connect with nil should return %s error", errNotUserGroup)
	}
}

func TestObserveNotUserGroup(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Observe(ctx, nil)
	if err == nil || err.Error() != errNotUserGroup {
		t.Errorf("Observe with nil should return %s error", errNotUserGroup)
	}
}

func TestCreateNotUserGroup(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Create(ctx, nil)
	if err == nil || err.Error() != errNotUserGroup {
		t.Errorf("Create with nil should return %s error", errNotUserGroup)
	}
}

func TestUpdateNotUserGroup(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Update(ctx, nil)
	if err == nil || err.Error() != errNotUserGroup {
		t.Errorf("Update with nil should return %s error", errNotUserGroup)
	}
}

func TestDeleteNotUserGroup(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Delete(ctx, nil)
	if err == nil || err.Error() != errNotUserGroup {
		t.Errorf("Delete with nil should return %s error", errNotUserGroup)
	}
}

func TestObserveUserGroupNotFound(t *testing.T) {
	ctx := context.Background()

	ug := &v1beta1.UserGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ug",
		},
		Spec: v1beta1.UserGroupSpec{
			ForProvider: v1beta1.UserGroupParameters{
				GroupName: "testgroup",
				GroupType: int64(1),
			},
		},
	}

	ext := &external{
		service: &mockUserGroupClient{
			listUserGroupsFunc: func(ctx context.Context) ([]*harborclients.UserGroupStatus, error) {
				return []*harborclients.UserGroupStatus{}, nil
			},
		},
		kube: nil,
	}

	obs, err := ext.Observe(ctx, ug)
	if err != nil {
		t.Errorf("Observe returned error: %v", err)
	}

	if obs.ResourceExists {
		t.Errorf("Observe should return ResourceExists=false for not found")
	}
}

func TestObserveUserGroupExists(t *testing.T) {
	ctx := context.Background()

	ug := &v1beta1.UserGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ug",
		},
		Spec: v1beta1.UserGroupSpec{
			ForProvider: v1beta1.UserGroupParameters{
				GroupName: "testgroup",
				GroupType: int64(1),
			},
		},
	}

	ext := &external{
		service: &mockUserGroupClient{
			listUserGroupsFunc: func(ctx context.Context) ([]*harborclients.UserGroupStatus, error) {
				return []*harborclients.UserGroupStatus{
					{
						ID:        123,
						GroupName: "testgroup",
						GroupType: int64(1),
					},
				}, nil
			},
		},
		kube: nil,
	}

	obs, err := ext.Observe(ctx, ug)
	if err != nil {
		t.Errorf("Observe returned error: %v", err)
	}

	if !obs.ResourceExists {
		t.Errorf("Observe should return ResourceExists=true for found group")
	}

	if !obs.ResourceUpToDate {
		t.Errorf("Observe should return ResourceUpToDate=true when spec matches")
	}
}

func TestObserveUserGroupNotUpToDate(t *testing.T) {
	ctx := context.Background()

	ug := &v1beta1.UserGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ug",
		},
		Spec: v1beta1.UserGroupSpec{
			ForProvider: v1beta1.UserGroupParameters{
				GroupName: "testgroup",
				GroupType: int64(2), // HTTP - different from LDAP (1)
			},
		},
	}

	ext := &external{
		service: &mockUserGroupClient{
			listUserGroupsFunc: func(ctx context.Context) ([]*harborclients.UserGroupStatus, error) {
				return []*harborclients.UserGroupStatus{
					{
						ID:        123,
						GroupName: "testgroup",
						GroupType: int64(1),
					},
				}, nil
			},
		},
		kube: nil,
	}

	obs, err := ext.Observe(ctx, ug)
	if err != nil {
		t.Errorf("Observe returned error: %v", err)
	}

	if !obs.ResourceExists {
		t.Errorf("Observe should return ResourceExists=true")
	}

	if obs.ResourceUpToDate {
		t.Errorf("Observe should return ResourceUpToDate=false when spec differs")
	}
}

func TestObserveUserGroupListError(t *testing.T) {
	ctx := context.Background()

	ug := &v1beta1.UserGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ug",
		},
		Spec: v1beta1.UserGroupSpec{
			ForProvider: v1beta1.UserGroupParameters{
				GroupName: "testgroup",
				GroupType: int64(1),
			},
		},
	}

	ext := &external{
		service: &mockUserGroupClient{
			listUserGroupsFunc: func(ctx context.Context) ([]*harborclients.UserGroupStatus, error) {
				return nil, errors.New("api error")
			},
		},
		kube: nil,
	}

	_, err := ext.Observe(ctx, ug)
	if err == nil {
		t.Errorf("Observe should return error when ListUserGroups fails")
	}
}

func TestCreateUserGroupSuccess(t *testing.T) {
	ctx := context.Background()

	ug := &v1beta1.UserGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ug",
		},
		Spec: v1beta1.UserGroupSpec{
			ForProvider: v1beta1.UserGroupParameters{
				GroupName: "testgroup",
				GroupType: int64(1),
			},
		},
	}

	ext := &external{
		service: &mockUserGroupClient{
			createUserGroupFunc: func(ctx context.Context, spec *harborclients.UserGroupSpec) (*harborclients.UserGroupStatus, error) {
				return &harborclients.UserGroupStatus{
					ID:        123,
					GroupName: "testgroup",
					GroupType: int64(1),
				}, nil
			},
		},
		kube: nil,
	}

	_, err := ext.Create(ctx, ug)
	if err != nil {
		t.Errorf("Create should succeed: %v", err)
	}
}

func TestCreateUserGroupError(t *testing.T) {
	ctx := context.Background()

	ug := &v1beta1.UserGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ug",
		},
		Spec: v1beta1.UserGroupSpec{
			ForProvider: v1beta1.UserGroupParameters{
				GroupName: "testgroup",
				GroupType: int64(1),
			},
		},
	}

	ext := &external{
		service: &mockUserGroupClient{
			createUserGroupFunc: func(ctx context.Context, spec *harborclients.UserGroupSpec) (*harborclients.UserGroupStatus, error) {
				return nil, errors.New("api error")
			},
		},
		kube: nil,
	}

	_, err := ext.Create(ctx, ug)
	if err == nil {
		t.Errorf("Create should return error when CreateUserGroup fails")
	}
}

func TestUpdateUserGroupSuccess(t *testing.T) {
	ctx := context.Background()

	groupID := int64(123)
	ug := &v1beta1.UserGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ug",
		},
		Spec: v1beta1.UserGroupSpec{
			ForProvider: v1beta1.UserGroupParameters{
				GroupName: "testgroup",
				GroupType: int64(1),
			},
		},
		Status: v1beta1.UserGroupStatus{
			AtProvider: v1beta1.UserGroupObservation{
				ID: &groupID,
			},
		},
	}

	ext := &external{
		service: &mockUserGroupClient{
			updateUserGroupFunc: func(ctx context.Context, groupID int64, spec *harborclients.UserGroupSpec) (*harborclients.UserGroupStatus, error) {
				return &harborclients.UserGroupStatus{
					ID:        groupID,
					GroupName: "testgroup",
					GroupType: int64(1),
				}, nil
			},
		},
		kube: nil,
	}

	_, err := ext.Update(ctx, ug)
	if err != nil {
		t.Errorf("Update should succeed: %v", err)
	}
}

func TestUpdateUserGroupError(t *testing.T) {
	ctx := context.Background()

	groupID := int64(123)
	ug := &v1beta1.UserGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ug",
		},
		Spec: v1beta1.UserGroupSpec{
			ForProvider: v1beta1.UserGroupParameters{
				GroupName: "testgroup",
				GroupType: int64(1),
			},
		},
		Status: v1beta1.UserGroupStatus{
			AtProvider: v1beta1.UserGroupObservation{
				ID: &groupID,
			},
		},
	}

	ext := &external{
		service: &mockUserGroupClient{
			updateUserGroupFunc: func(ctx context.Context, groupID int64, spec *harborclients.UserGroupSpec) (*harborclients.UserGroupStatus, error) {
				return nil, errors.New("api error")
			},
		},
		kube: nil,
	}

	_, err := ext.Update(ctx, ug)
	if err == nil {
		t.Errorf("Update should return error when UpdateUserGroup fails")
	}
}

func TestDeleteUserGroupSuccess(t *testing.T) {
	ctx := context.Background()

	groupID := int64(123)
	ug := &v1beta1.UserGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ug",
		},
		Status: v1beta1.UserGroupStatus{
			AtProvider: v1beta1.UserGroupObservation{
				ID: &groupID,
			},
		},
	}

	ext := &external{
		service: &mockUserGroupClient{
			deleteUserGroupFunc: func(ctx context.Context, groupID int64) error {
				return nil
			},
		},
		kube: nil,
	}

	_, err := ext.Delete(ctx, ug)
	if err != nil {
		t.Errorf("Delete should succeed: %v", err)
	}
}

func TestDeleteUserGroupError(t *testing.T) {
	ctx := context.Background()

	groupID := int64(123)
	ug := &v1beta1.UserGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ug",
		},
		Status: v1beta1.UserGroupStatus{
			AtProvider: v1beta1.UserGroupObservation{
				ID: &groupID,
			},
		},
	}

	ext := &external{
		service: &mockUserGroupClient{
			deleteUserGroupFunc: func(ctx context.Context, groupID int64) error {
				return errors.New("api error")
			},
		},
		kube: nil,
	}

	_, err := ext.Delete(ctx, ug)
	if err == nil {
		t.Errorf("Delete should return error when DeleteUserGroup fails")
	}
}

func TestUserGroupHasRequiredFields(t *testing.T) {
	ug := &v1beta1.UserGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ug",
		},
		Spec: v1beta1.UserGroupSpec{
			ForProvider: v1beta1.UserGroupParameters{
				GroupName: "testgroup",
				GroupType: int64(1),
			},
		},
	}

	if ug.Spec.ForProvider.GroupName == "" {
		t.Errorf("UserGroup should have GroupName")
	}

	if ug.Spec.ForProvider.GroupType == 0 {
		t.Errorf("UserGroup should have GroupType")
	}
}

func TestUserGroupStatusFields(t *testing.T) {
	groupID := int64(123)
	ug := &v1beta1.UserGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ug",
		},
		Status: v1beta1.UserGroupStatus{
			AtProvider: v1beta1.UserGroupObservation{
				ID: &groupID,
			},
		},
	}

	if ug.Status.AtProvider.ID == nil {
		t.Errorf("UserGroup status should have ID field")
	}

	if *ug.Status.AtProvider.ID != 123 {
		t.Errorf("UserGroup status ID should be 123, got %d", *ug.Status.AtProvider.ID)
	}
}

// mockUserGroupClient implements HarborClienter for UserGroup tests
type mockUserGroupClient struct {
	harborclients.HarborClienter
	listUserGroupsFunc  func(ctx context.Context) ([]*harborclients.UserGroupStatus, error)
	createUserGroupFunc func(ctx context.Context, spec *harborclients.UserGroupSpec) (*harborclients.UserGroupStatus, error)
	updateUserGroupFunc func(ctx context.Context, groupID int64, spec *harborclients.UserGroupSpec) (*harborclients.UserGroupStatus, error)
	deleteUserGroupFunc func(ctx context.Context, groupID int64) error
}

func (m *mockUserGroupClient) ListUserGroups(ctx context.Context) ([]*harborclients.UserGroupStatus, error) {
	if m.listUserGroupsFunc != nil {
		return m.listUserGroupsFunc(ctx)
	}
	return nil, nil
}

func (m *mockUserGroupClient) CreateUserGroup(ctx context.Context, spec *harborclients.UserGroupSpec) (*harborclients.UserGroupStatus, error) {
	if m.createUserGroupFunc != nil {
		return m.createUserGroupFunc(ctx, spec)
	}
	return nil, nil
}

func (m *mockUserGroupClient) UpdateUserGroup(ctx context.Context, groupID int64, spec *harborclients.UserGroupSpec) (*harborclients.UserGroupStatus, error) {
	if m.updateUserGroupFunc != nil {
		return m.updateUserGroupFunc(ctx, groupID, spec)
	}
	return nil, nil
}

func (m *mockUserGroupClient) DeleteUserGroup(ctx context.Context, groupID int64) error {
	if m.deleteUserGroupFunc != nil {
		return m.deleteUserGroupFunc(ctx, groupID)
	}
	return nil
}

func (m *mockUserGroupClient) Close() error {
	return nil
}

func (m *mockUserGroupClient) GetBaseURL() string {
	return "https://harbor.example.com"
}
