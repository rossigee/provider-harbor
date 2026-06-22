/*
Copyright 2024 Crossplane Harbor Provider.
*/

package usergroup

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	"github.com/rossigee/provider-harbor/apis/usergroup/v1beta1"
	harborclients "github.com/rossigee/provider-harbor/internal/clients"
)

// mockUserGroupClient is a minimal mock that satisfies the HarborClienter
// interface only for the methods exercised by the usergroup controller.
type mockUserGroupClient struct {
	harborclients.MockHarborClient
	listFunc      func(ctx context.Context) ([]*harborclients.UserGroupStatus, error)
	getByNameFunc func(ctx context.Context, name string) (*harborclients.UserGroupStatus, error)
	createFunc    func(ctx context.Context, spec *harborclients.UserGroupSpec) (*harborclients.UserGroupStatus, error)
	updateFunc    func(ctx context.Context, id int64, spec *harborclients.UserGroupSpec) (*harborclients.UserGroupStatus, error)
	deleteFunc    func(ctx context.Context, id int64) error
}

func (m *mockUserGroupClient) GetUserGroupByName(ctx context.Context, name string) (*harborclients.UserGroupStatus, error) {
	if m.getByNameFunc != nil {
		return m.getByNameFunc(ctx, name)
	}
	return nil, nil
}

func (m *mockUserGroupClient) ListUserGroups(ctx context.Context) ([]*harborclients.UserGroupStatus, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx)
	}
	return nil, nil
}

func (m *mockUserGroupClient) CreateUserGroup(ctx context.Context, spec *harborclients.UserGroupSpec) (*harborclients.UserGroupStatus, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, spec)
	}
	return &harborclients.UserGroupStatus{ID: 1, GroupName: spec.GroupName, GroupType: spec.GroupType}, nil
}

func (m *mockUserGroupClient) UpdateUserGroup(ctx context.Context, id int64, spec *harborclients.UserGroupSpec) (*harborclients.UserGroupStatus, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, id, spec)
	}
	return &harborclients.UserGroupStatus{ID: id, GroupName: spec.GroupName, GroupType: spec.GroupType}, nil
}

func (m *mockUserGroupClient) DeleteUserGroup(ctx context.Context, id int64) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *mockUserGroupClient) GetUserGroup(ctx context.Context, id int64) (*harborclients.UserGroupStatus, error) {
	return nil, nil
}

// newTestUserGroup builds a minimal UserGroup CR for testing.
func newTestUserGroup(name string, groupType int64) *v1beta1.UserGroup {
	return &v1beta1.UserGroup{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: v1beta1.UserGroupSpec{
			ForProvider: v1beta1.UserGroupParameters{
				GroupName: name,
				GroupType: groupType,
			},
		},
	}
}

func TestObserveUserGroupNotFound(t *testing.T) {
	ctx := context.Background()
	ext := &external{
		service: &mockUserGroupClient{
			getByNameFunc: func(ctx context.Context, name string) (*harborclients.UserGroupStatus, error) {
				return nil, nil
			},
		},
	}
	cr := newTestUserGroup("devs", 3)
	obs, err := ext.Observe(ctx, cr)
	if err != nil {
		t.Fatalf("Observe should not fail on not-found, got: %v", err)
	}
	if obs.ResourceExists {
		t.Error("ResourceExists should be false when group not found")
	}
}

func TestObserveUserGroupAvailableWhenUpToDate(t *testing.T) {
	ctx := context.Background()
	gid := int64(42)
	ext := &external{
		service: &mockUserGroupClient{
			getByNameFunc: func(ctx context.Context, name string) (*harborclients.UserGroupStatus, error) {
				return &harborclients.UserGroupStatus{ID: gid, GroupName: "devs", GroupType: 3}, nil
			},
		},
	}
	cr := newTestUserGroup("devs", 3)
	obs, err := ext.Observe(ctx, cr)
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if !obs.ResourceExists {
		t.Fatal("expected ResourceExists=true")
	}
	if !obs.ResourceUpToDate {
		t.Error("expected ResourceUpToDate=true when spec matches")
	}

	// Verify xpv1.Available() was set.
	cond := cr.GetCondition(xpv1.TypeReady)
	if cond.Status != "True" {
		t.Errorf("expected Ready=True (Available) after Observe up-to-date, got %v", cond.Status)
	}

	// Verify status.atProvider.ID was set.
	if cr.Status.AtProvider.ID == nil || *cr.Status.AtProvider.ID != gid {
		t.Errorf("expected atProvider.ID=%d, got %v", gid, cr.Status.AtProvider.ID)
	}
}

func TestObserveUserGroupNotUpToDate(t *testing.T) {
	ctx := context.Background()
	ext := &external{
		service: &mockUserGroupClient{
			getByNameFunc: func(ctx context.Context, name string) (*harborclients.UserGroupStatus, error) {
				return &harborclients.UserGroupStatus{ID: 5, GroupName: "devs", GroupType: 1}, nil // LDAP, but spec wants OIDC
			},
		},
	}
	cr := newTestUserGroup("devs", 3) // spec: OIDC
	obs, err := ext.Observe(ctx, cr)
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if !obs.ResourceExists {
		t.Fatal("expected ResourceExists=true")
	}
	if obs.ResourceUpToDate {
		t.Error("expected ResourceUpToDate=false when group type drifted")
	}
	// A drifted-but-existing group stays Available — drift is signalled only by
	// ResourceUpToDate=false (which drives Update), not by withholding Ready.
	cond := cr.GetCondition(xpv1.TypeReady)
	if cond.Status != "True" {
		t.Error("expected Ready=True (Available) for an existing group, even when drifted")
	}
}

func TestCreateUserGroupSetsCreating(t *testing.T) {
	ctx := context.Background()
	gid := int64(10)
	ext := &external{
		service: &mockUserGroupClient{
			createFunc: func(ctx context.Context, spec *harborclients.UserGroupSpec) (*harborclients.UserGroupStatus, error) {
				return &harborclients.UserGroupStatus{ID: gid, GroupName: spec.GroupName, GroupType: spec.GroupType}, nil
			},
		},
	}
	cr := newTestUserGroup("devs", 3)
	_, err := ext.Create(ctx, cr)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	// Verify Creating() condition was set.
	cond := cr.GetCondition(xpv1.TypeReady)
	if cond.Reason != xpv1.ReasonCreating {
		t.Errorf("expected Ready reason Creating, got %v", cond.Reason)
	}
	// Verify atProvider.ID set from result.
	if cr.Status.AtProvider.ID == nil || *cr.Status.AtProvider.ID != gid {
		t.Errorf("expected atProvider.ID=%d after Create, got %v", gid, cr.Status.AtProvider.ID)
	}
}

func TestDeleteUserGroupSetsDeleting(t *testing.T) {
	ctx := context.Background()
	deleted := false
	ext := &external{
		service: &mockUserGroupClient{
			deleteFunc: func(ctx context.Context, id int64) error {
				deleted = true
				return nil
			},
		},
	}
	gid := int64(10)
	cr := newTestUserGroup("devs", 3)
	cr.Status.AtProvider.ID = &gid

	_, err := ext.Delete(ctx, cr)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !deleted {
		t.Error("expected DeleteUserGroup to be called")
	}
	// Verify Deleting() condition was set.
	cond := cr.GetCondition(xpv1.TypeReady)
	if cond.Reason != xpv1.ReasonDeleting {
		t.Errorf("expected Ready reason Deleting, got %v", cond.Reason)
	}
}
