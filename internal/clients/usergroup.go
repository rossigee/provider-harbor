/*
Copyright 2024 Crossplane Harbor Provider.
*/

package clients

import (
	"context"

	harborusergroup "github.com/goharbor/go-client/pkg/sdk/v2.0/client/usergroup"
	harbormodels "github.com/goharbor/go-client/pkg/sdk/v2.0/models"
	"github.com/pkg/errors"
)

// UserGroupSpec defines the desired state of a UserGroup
type UserGroupSpec struct {
	GroupName   string  `json:"groupName"`
	GroupType   int64   `json:"groupType"`
	LdapGroupDn *string `json:"ldapGroupDn,omitempty"`
}

// UserGroupStatus represents the observed state of a UserGroup
type UserGroupStatus struct {
	ID          int64
	GroupName   string
	GroupType   int64
	LdapGroupDn string
}

// userGroupStatusFromModel converts a Harbor API UserGroup model to our UserGroupStatus.
func userGroupStatusFromModel(g *harbormodels.UserGroup) *UserGroupStatus {
	if g == nil {
		return nil
	}
	return &UserGroupStatus{
		ID:          g.ID,
		GroupName:   g.GroupName,
		GroupType:   g.GroupType,
		LdapGroupDn: g.LdapGroupDn,
	}
}

// CreateUserGroup creates a new user group in Harbor.
// Harbor returns only a Location header on 201; we parse the ID from that URL
// and re-read to return the authoritative observed state.
func (c *HarborClient) CreateUserGroup(ctx context.Context, spec *UserGroupSpec) (*UserGroupStatus, error) {
	if spec == nil {
		return nil, errors.New("user group spec is required")
	}
	if spec.GroupName == "" {
		return nil, errors.New("group name is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Creating Harbor user group", "groupName", spec.GroupName, "groupType", spec.GroupType)

	ldapDn := ""
	if spec.LdapGroupDn != nil {
		ldapDn = *spec.LdapGroupDn
	}
	req := &harbormodels.UserGroup{
		GroupName:   spec.GroupName,
		GroupType:   spec.GroupType,
		LdapGroupDn: ldapDn,
	}
	params := harborusergroup.NewCreateUserGroupParams().WithContext(ctx).WithUsergroup(req)
	resp, err := v2Client.Usergroup.CreateUserGroup(ctx, params)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create Harbor user group")
	}

	// Resolve the new group's id. Prefer the Location header when present
	// (/api/v2.0/usergroups/42); Harbor does not always populate it, so fall back
	// to a name lookup. The name lookup is also the resilient path in OIDC mode,
	// where many groups exist and an unfiltered/paged list may not contain ours.
	if gid, lerr := idFromLocation(resp.Location); lerr == nil && gid > 0 {
		st, err := c.GetUserGroup(ctx, gid)
		if err != nil {
			return nil, err
		}
		if st != nil && st.ID > 0 {
			return st, nil
		}
	}

	st, err := c.GetUserGroupByName(ctx, spec.GroupName)
	if err != nil {
		return nil, err
	}
	if st == nil {
		return nil, errors.New("Harbor user group created but not found by name")
	}
	return st, nil
}

// GetUserGroupByName finds a Harbor user group by exact name using Harbor's
// group_name filter (fuzzy server-side; we exact-match the result). Returns
// (nil, nil) when no group with that name exists.
func (c *HarborClient) GetUserGroupByName(ctx context.Context, name string) (*UserGroupStatus, error) {
	if name == "" {
		return nil, errors.New("group name is required")
	}
	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	params := harborusergroup.NewListUserGroupsParams().WithContext(ctx).WithGroupName(&name)
	resp, err := v2Client.Usergroup.ListUserGroups(ctx, params)
	if err != nil {
		return nil, errors.Wrap(err, "cannot search Harbor user groups")
	}
	for _, g := range resp.Payload {
		if g != nil && g.GroupName == name {
			return userGroupStatusFromModel(g), nil
		}
	}
	return nil, nil
}

// ListUserGroups lists all user groups in Harbor.
func (c *HarborClient) ListUserGroups(ctx context.Context) ([]*UserGroupStatus, error) {
	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Listing Harbor user groups")

	params := harborusergroup.NewListUserGroupsParams().WithContext(ctx)
	resp, err := v2Client.Usergroup.ListUserGroups(ctx, params)
	if err != nil {
		return nil, errors.Wrap(err, "cannot list Harbor user groups")
	}

	out := make([]*UserGroupStatus, 0, len(resp.Payload))
	for _, g := range resp.Payload {
		if g != nil {
			out = append(out, userGroupStatusFromModel(g))
		}
	}
	return out, nil
}

// GetUserGroup retrieves a specific user group from Harbor by numeric ID.
// Returns (nil, nil) when the group does not exist (404).
func (c *HarborClient) GetUserGroup(ctx context.Context, groupID int64) (*UserGroupStatus, error) {
	if groupID <= 0 {
		return nil, errors.New("group ID is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Getting Harbor user group", "groupId", groupID)

	params := harborusergroup.NewGetUserGroupParams().WithContext(ctx).WithGroupID(groupID)
	resp, err := v2Client.Usergroup.GetUserGroup(ctx, params)
	if err != nil {
		if isHarborNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "cannot get Harbor user group")
	}
	return userGroupStatusFromModel(resp.Payload), nil
}

// UpdateUserGroup updates a user group in Harbor.
func (c *HarborClient) UpdateUserGroup(ctx context.Context, groupID int64, spec *UserGroupSpec) (*UserGroupStatus, error) {
	if groupID <= 0 {
		return nil, errors.New("group ID is required")
	}
	if spec == nil {
		return nil, errors.New("user group spec is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Updating Harbor user group", "groupId", groupID, "groupName", spec.GroupName)

	ldapDn := ""
	if spec.LdapGroupDn != nil {
		ldapDn = *spec.LdapGroupDn
	}
	req := &harbormodels.UserGroup{
		GroupName:   spec.GroupName,
		GroupType:   spec.GroupType,
		LdapGroupDn: ldapDn,
	}
	params := harborusergroup.NewUpdateUserGroupParams().WithContext(ctx).
		WithGroupID(groupID).
		WithUsergroup(req)
	if _, err := v2Client.Usergroup.UpdateUserGroup(ctx, params); err != nil {
		return nil, errors.Wrap(err, "cannot update Harbor user group")
	}

	return c.GetUserGroup(ctx, groupID)
}

// DeleteUserGroup deletes a user group from Harbor. Idempotent on 404.
func (c *HarborClient) DeleteUserGroup(ctx context.Context, groupID int64) error {
	if groupID <= 0 {
		return errors.New("group ID is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Deleting Harbor user group", "groupId", groupID)

	params := harborusergroup.NewDeleteUserGroupParams().WithContext(ctx).WithGroupID(groupID)
	if _, err := v2Client.Usergroup.DeleteUserGroup(ctx, params); err != nil {
		if isHarborNotFound(err) {
			return nil
		}
		return errors.Wrap(err, "cannot delete Harbor user group")
	}
	return nil
}
