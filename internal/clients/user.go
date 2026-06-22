/*
Copyright 2025 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package clients

import (
	"context"
	"time"

	harboruser "github.com/goharbor/go-client/pkg/sdk/v2.0/client/user"
	harbormodels "github.com/goharbor/go-client/pkg/sdk/v2.0/models"
	"github.com/pkg/errors"
)

// UserSpec defines the desired state of a Harbor user
type UserSpec struct {
	Username  string `json:"username"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	AdminFlag bool   `json:"admin_flag"`
}

// UserStatus represents the status of a Harbor user
type UserStatus struct {
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	AdminFlag bool      `json:"admin_flag"`
	CreatedAt time.Time `json:"created_at"`
}

// userStatusFromModel converts a Harbor UserResp model into our UserStatus.
func userStatusFromModel(u *harbormodels.UserResp) *UserStatus {
	if u == nil {
		return &UserStatus{}
	}
	st := &UserStatus{
		Username:  u.Username,
		Email:     u.Email,
		AdminFlag: u.SysadminFlag,
	}
	if t := time.Time(u.CreationTime); !t.IsZero() {
		st.CreatedAt = t
	}
	return st
}

// findUserByUsername locates a Harbor user by username using ListUsers with an
// exact-match query filter. Harbor's GetUser requires a numeric user_id, but the
// CR addresses by username, so we list-and-match. Returns (nil, nil) when absent.
func (c *HarborClient) findUserByUsername(ctx context.Context, username string) (*harbormodels.UserResp, error) {
	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	// The q param supports "username=<value>" for an exact match.
	q := "username=" + username
	params := harboruser.NewListUsersParams().WithContext(ctx).WithQ(&q)
	resp, err := v2Client.User.ListUsers(ctx, params)
	if err != nil {
		if isHarborNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "cannot list Harbor users")
	}
	for _, u := range resp.Payload {
		if u != nil && u.Username == username {
			return u, nil
		}
	}
	return nil, nil
}

// CreateUser creates a new Harbor user
func (c *HarborClient) CreateUser(ctx context.Context, spec *UserSpec) (*UserStatus, error) {
	if spec == nil {
		return nil, errors.New("user spec is required")
	}
	if spec.Username == "" {
		return nil, errors.New("username is required")
	}
	if spec.Email == "" {
		return nil, errors.New("email is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Creating Harbor user", "username", spec.Username, "email", spec.Email)

	req := &harbormodels.UserCreationReq{
		Username: spec.Username,
		Email:    spec.Email,
		Password: spec.Password,
		Realname: spec.Username,
	}
	createParams := harboruser.NewCreateUserParams().WithContext(ctx).WithUserReq(req)
	if _, err := v2Client.User.CreateUser(ctx, createParams); err != nil {
		return nil, errors.Wrap(err, "cannot create Harbor user")
	}

	// If the user should be a sysadmin, set that flag now (separate API call).
	if spec.AdminFlag {
		u, err := c.findUserByUsername(ctx, spec.Username)
		if err != nil {
			return nil, errors.Wrap(err, "cannot find user after creation")
		}
		if u != nil {
			sysAdminParams := harboruser.NewSetUserSysAdminParams().WithContext(ctx).
				WithUserID(u.UserID).
				WithSysadminFlag(&harbormodels.UserSysAdminFlag{SysadminFlag: true})
			if _, err := v2Client.User.SetUserSysAdmin(ctx, sysAdminParams); err != nil {
				return nil, errors.Wrap(err, "cannot set Harbor user sysadmin flag")
			}
		}
	}

	// Re-read to get authoritative state.
	st, err := c.GetUser(ctx, spec.Username)
	if err != nil {
		return nil, err
	}
	if st == nil {
		return nil, errors.New("Harbor user created but not yet observable")
	}
	return st, nil
}

// GetUser retrieves a Harbor user by username.
// Harbor's GetUser API requires a numeric user_id; we locate the user via a
// filtered ListUsers call and return (nil, nil) when no matching user is found.
func (c *HarborClient) GetUser(ctx context.Context, username string) (*UserStatus, error) {
	if username == "" {
		return nil, errors.New("username is required")
	}

	c.logger.Info("Retrieving Harbor user", "username", username)

	u, err := c.findUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, nil
	}
	return userStatusFromModel(u), nil
}

// UpdateUser updates an existing Harbor user's profile and sysadmin flag.
func (c *HarborClient) UpdateUser(ctx context.Context, username string, spec *UserSpec) (*UserStatus, error) {
	if username == "" {
		return nil, errors.New("username is required")
	}
	if spec == nil {
		return nil, errors.New("user spec is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Updating Harbor user", "username", username, "email", spec.Email)

	// Find the numeric user id required by the update API.
	u, err := c.findUserByUsername(ctx, username)
	if err != nil {
		return nil, errors.Wrap(err, "cannot find user for update")
	}
	if u == nil {
		return nil, errors.Errorf("Harbor user %q not found", username)
	}

	profileParams := harboruser.NewUpdateUserProfileParams().WithContext(ctx).
		WithUserID(u.UserID).
		WithProfile(&harbormodels.UserProfile{Email: spec.Email})
	if _, err := v2Client.User.UpdateUserProfile(ctx, profileParams); err != nil {
		return nil, errors.Wrap(err, "cannot update Harbor user profile")
	}

	// Update sysadmin flag separately.
	sysAdminParams := harboruser.NewSetUserSysAdminParams().WithContext(ctx).
		WithUserID(u.UserID).
		WithSysadminFlag(&harbormodels.UserSysAdminFlag{SysadminFlag: spec.AdminFlag})
	if _, err := v2Client.User.SetUserSysAdmin(ctx, sysAdminParams); err != nil {
		return nil, errors.Wrap(err, "cannot update Harbor user sysadmin flag")
	}

	return c.GetUser(ctx, username)
}

// DeleteUser deletes a Harbor user. Idempotent: absent user is already-gone.
func (c *HarborClient) DeleteUser(ctx context.Context, username string) error {
	if username == "" {
		return errors.New("username is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Deleting Harbor user", "username", username)

	u, err := c.findUserByUsername(ctx, username)
	if err != nil {
		return errors.Wrap(err, "cannot find user for deletion")
	}
	if u == nil {
		// Already absent — idempotent.
		return nil
	}

	delParams := harboruser.NewDeleteUserParams().WithContext(ctx).WithUserID(u.UserID)
	if _, err := v2Client.User.DeleteUser(ctx, delParams); err != nil {
		if isHarborNotFound(err) {
			return nil
		}
		return errors.Wrap(err, "cannot delete Harbor user")
	}
	return nil
}
