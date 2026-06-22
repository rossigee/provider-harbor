/*
Copyright 2024 Crossplane Harbor Provider.
*/

package member

import (
	"context"
	"errors"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rossigee/provider-harbor/apis/member/v1beta1"
	harborclients "github.com/rossigee/provider-harbor/internal/clients"
)

// ---- UserMember ----

func TestUserMemberObserveAbsent(t *testing.T) {
	ctx := context.Background()
	cr := &v1beta1.UserMember{
		ObjectMeta: metav1.ObjectMeta{Name: "um"},
		Spec:       v1beta1.UserMemberSpec{ForProvider: v1beta1.UserMemberParameters{ProjectID: "1", Username: "alice", Role: "developer"}},
	}
	ext := &userMemberExternal{service: &harborclients.MockHarborClient{
		FindProjectMemberFunc: func(ctx context.Context, projectID, entityType, entityName string) (*harborclients.MemberStatus, error) {
			return nil, nil
		},
	}}
	obs, err := ext.Observe(ctx, cr)
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if obs.ResourceExists {
		t.Error("ResourceExists should be false when member absent")
	}
}

func TestUserMemberObserveAdopts(t *testing.T) {
	ctx := context.Background()
	cr := &v1beta1.UserMember{
		ObjectMeta: metav1.ObjectMeta{Name: "um"},
		Spec:       v1beta1.UserMemberSpec{ForProvider: v1beta1.UserMemberParameters{ProjectID: "1", Username: "alice", Role: "developer"}},
	}
	ext := &userMemberExternal{service: &harborclients.MockHarborClient{
		FindProjectMemberFunc: func(ctx context.Context, projectID, entityType, entityName string) (*harborclients.MemberStatus, error) {
			if entityType != "u" {
				t.Errorf("expected entity type u, got %q", entityType)
			}
			return &harborclients.MemberStatus{ID: "7", MemberName: "alice", MemberType: "user", Role: "developer"}, nil
		},
	}}
	obs, err := ext.Observe(ctx, cr)
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if !obs.ResourceExists || !obs.ResourceUpToDate {
		t.Errorf("expected exists+upToDate, got %+v", obs)
	}
	if meta.GetExternalName(cr) != "7" {
		t.Errorf("Observe should adopt external-name 7, got %q", meta.GetExternalName(cr))
	}
}

func TestUserMemberObserveByIDDrift(t *testing.T) {
	ctx := context.Background()
	cr := &v1beta1.UserMember{
		ObjectMeta: metav1.ObjectMeta{Name: "um"},
		Spec:       v1beta1.UserMemberSpec{ForProvider: v1beta1.UserMemberParameters{ProjectID: "1", Username: "alice", Role: "maintainer"}},
	}
	meta.SetExternalName(cr, "7")
	ext := &userMemberExternal{service: &harborclients.MockHarborClient{
		GetProjectMemberByIDFunc: func(ctx context.Context, projectID, memberID string) (*harborclients.MemberStatus, error) {
			if memberID != "7" {
				t.Errorf("expected get by id 7, got %q", memberID)
			}
			return &harborclients.MemberStatus{ID: "7", MemberName: "alice", MemberType: "user", Role: "developer"}, nil
		},
	}}
	obs, err := ext.Observe(ctx, cr)
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if !obs.ResourceExists {
		t.Error("expected exists")
	}
	if obs.ResourceUpToDate {
		t.Error("expected drift (spec maintainer vs observed developer)")
	}
}

func TestUserMemberCreateSetsExternalName(t *testing.T) {
	ctx := context.Background()
	cr := &v1beta1.UserMember{
		ObjectMeta: metav1.ObjectMeta{Name: "um"},
		Spec:       v1beta1.UserMemberSpec{ForProvider: v1beta1.UserMemberParameters{ProjectID: "1", Username: "alice", Role: "developer"}},
	}
	called := false
	ext := &userMemberExternal{service: &harborclients.MockHarborClient{
		AddProjectUserMemberFunc: func(ctx context.Context, projectID, username, role string) (string, error) {
			called = true
			if username != "alice" {
				t.Errorf("expected username alice, got %q", username)
			}
			return "9", nil
		},
	}}
	if _, err := ext.Create(ctx, cr); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !called {
		t.Error("AddProjectUserMember not called")
	}
	if meta.GetExternalName(cr) != "9" {
		t.Errorf("Create should set external-name 9, got %q", meta.GetExternalName(cr))
	}
}

func TestUserMemberUpdateByID(t *testing.T) {
	ctx := context.Background()
	cr := &v1beta1.UserMember{
		ObjectMeta: metav1.ObjectMeta{Name: "um"},
		Spec:       v1beta1.UserMemberSpec{ForProvider: v1beta1.UserMemberParameters{ProjectID: "1", Username: "alice", Role: "maintainer"}},
	}
	meta.SetExternalName(cr, "9")
	ext := &userMemberExternal{service: &harborclients.MockHarborClient{
		UpdateProjectMemberByIDFunc: func(ctx context.Context, projectID, memberID, role string) error {
			if memberID != "9" || role != "maintainer" {
				t.Errorf("expected update id 9 role maintainer, got id=%q role=%q", memberID, role)
			}
			return nil
		},
	}}
	if _, err := ext.Update(ctx, cr); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func TestUserMemberDeleteByID(t *testing.T) {
	ctx := context.Background()
	cr := &v1beta1.UserMember{
		ObjectMeta: metav1.ObjectMeta{Name: "um"},
		Spec:       v1beta1.UserMemberSpec{ForProvider: v1beta1.UserMemberParameters{ProjectID: "1", Username: "alice", Role: "developer"}},
	}
	meta.SetExternalName(cr, "9")
	deleted := false
	ext := &userMemberExternal{service: &harborclients.MockHarborClient{
		DeleteProjectMemberByIDFunc: func(ctx context.Context, projectID, memberID string) error {
			deleted = true
			return nil
		},
	}}
	if _, err := ext.Delete(ctx, cr); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !deleted {
		t.Error("DeleteProjectMemberByID not called")
	}
}

func TestUserMemberNotUserMember(t *testing.T) {
	ext := &userMemberExternal{}
	if _, err := ext.Observe(context.Background(), nil); err == nil {
		t.Error("expected error for nil resource")
	}
}

// ---- GroupMember ----

func TestGroupMemberObserveAdopts(t *testing.T) {
	ctx := context.Background()
	cr := &v1beta1.GroupMember{
		ObjectMeta: metav1.ObjectMeta{Name: "gm"},
		Spec:       v1beta1.GroupMemberSpec{ForProvider: v1beta1.GroupMemberParameters{ProjectID: "1", GroupName: "admins", Role: "guest"}},
	}
	ext := &groupMemberExternal{service: &harborclients.MockHarborClient{
		FindProjectMemberFunc: func(ctx context.Context, projectID, entityType, entityName string) (*harborclients.MemberStatus, error) {
			if entityType != "g" {
				t.Errorf("expected entity type g, got %q", entityType)
			}
			return &harborclients.MemberStatus{ID: "5", MemberName: "admins", MemberType: "group", Role: "guest"}, nil
		},
	}}
	obs, err := ext.Observe(ctx, cr)
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if !obs.ResourceExists || !obs.ResourceUpToDate {
		t.Errorf("expected exists+upToDate, got %+v", obs)
	}
	if meta.GetExternalName(cr) != "5" {
		t.Errorf("Observe should adopt external-name 5, got %q", meta.GetExternalName(cr))
	}
}

func TestGroupMemberCreateDefaultsTypeAndSetsExternalName(t *testing.T) {
	ctx := context.Background()
	cr := &v1beta1.GroupMember{
		ObjectMeta: metav1.ObjectMeta{Name: "gm"},
		Spec:       v1beta1.GroupMemberSpec{ForProvider: v1beta1.GroupMemberParameters{ProjectID: "1", GroupName: "admins", Role: "guest"}}, // GroupType nil
	}
	ext := &groupMemberExternal{service: &harborclients.MockHarborClient{
		AddProjectGroupMemberFunc: func(ctx context.Context, projectID, groupName string, gt int64, role string) (string, error) {
			if groupName != "admins" {
				t.Errorf("expected group admins, got %q", groupName)
			}
			if gt != 3 {
				t.Errorf("expected default group type 3 (OIDC), got %d", gt)
			}
			return "11", nil
		},
	}}
	if _, err := ext.Create(ctx, cr); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if meta.GetExternalName(cr) != "11" {
		t.Errorf("Create should set external-name 11, got %q", meta.GetExternalName(cr))
	}
}

func TestGroupMemberCreateHonoursGroupType(t *testing.T) {
	ctx := context.Background()
	gt := int64(1) // LDAP
	cr := &v1beta1.GroupMember{
		ObjectMeta: metav1.ObjectMeta{Name: "gm"},
		Spec:       v1beta1.GroupMemberSpec{ForProvider: v1beta1.GroupMemberParameters{ProjectID: "1", GroupName: "admins", Role: "guest", GroupType: &gt}},
	}
	ext := &groupMemberExternal{service: &harborclients.MockHarborClient{
		AddProjectGroupMemberFunc: func(ctx context.Context, projectID, groupName string, groupType int64, role string) (string, error) {
			if groupType != 1 {
				t.Errorf("expected group type 1 (LDAP), got %d", groupType)
			}
			return "12", nil
		},
	}}
	if _, err := ext.Create(ctx, cr); err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestGroupMemberDeleteByID(t *testing.T) {
	ctx := context.Background()
	cr := &v1beta1.GroupMember{
		ObjectMeta: metav1.ObjectMeta{Name: "gm"},
		Spec:       v1beta1.GroupMemberSpec{ForProvider: v1beta1.GroupMemberParameters{ProjectID: "1", GroupName: "admins", Role: "guest"}},
	}
	meta.SetExternalName(cr, "12")
	ext := &groupMemberExternal{service: &harborclients.MockHarborClient{
		DeleteProjectMemberByIDFunc: func(ctx context.Context, projectID, memberID string) error {
			if memberID != "12" {
				t.Errorf("expected delete id 12, got %q", memberID)
			}
			return nil
		},
	}}
	if _, err := ext.Delete(ctx, cr); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestGroupMemberObserveError(t *testing.T) {
	ctx := context.Background()
	cr := &v1beta1.GroupMember{
		ObjectMeta: metav1.ObjectMeta{Name: "gm"},
		Spec:       v1beta1.GroupMemberSpec{ForProvider: v1beta1.GroupMemberParameters{ProjectID: "1", GroupName: "admins", Role: "guest"}},
	}
	ext := &groupMemberExternal{service: &harborclients.MockHarborClient{
		FindProjectMemberFunc: func(ctx context.Context, projectID, entityType, entityName string) (*harborclients.MemberStatus, error) {
			return nil, errors.New("boom")
		},
	}}
	if _, err := ext.Observe(ctx, cr); err == nil {
		t.Error("expected Observe error to surface")
	}
}
