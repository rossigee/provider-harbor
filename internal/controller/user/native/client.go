package native

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/goharbor/go-client/pkg/sdk/v2.0/client/user"
	"github.com/goharbor/go-client/pkg/sdk/v2.0/models"
	harborclients "github.com/globallogicuki/provider-harbor/internal/clients"
	"github.com/pkg/errors"
)

// Client defines Harbor user operations interface
type Client interface {
	GetUser(ctx context.Context, username string) (*models.UserResp, error)
	CreateUser(ctx context.Context, username, email, realname, password string, admin bool, comment string) (int64, error)
	UpdateUser(ctx context.Context, userID int64, email, realname string, admin bool, comment string) error
	UpdateUserPassword(ctx context.Context, userID int64, newPassword string) error
	DeleteUser(ctx context.Context, userID int64) error
}

// HarborClient implements the Client interface
type HarborClient struct {
	V2Client *harborclients.HarborCLI
}

// NewHarborClient creates a new Harbor client
func NewHarborClient(v2Client *harborclients.HarborCLI) Client {
	return &HarborClient{
		V2Client: v2Client,
	}
}

// GetUser retrieves a user by username
func (c *HarborClient) GetUser(ctx context.Context, username string) (*models.UserResp, error) {
	params := &user.SearchUsersParams{
		Username: username,
		Context:  ctx,
	}

	resp, err := c.V2Client.V2Client.User.SearchUsers(params, c.V2Client.AuthInfo)
	if err != nil {
		return nil, errors.Wrap(err, "failed to search users")
	}

	// Search for exact username match
	for _, u := range resp.Payload {
		if u.Username == username {
			return u, nil
		}
	}

	return nil, errors.New("user not found")
}

// CreateUser creates a new user
func (c *HarborClient) CreateUser(ctx context.Context, username, email, realname, password string, admin bool, comment string) (int64, error) {
	userCreationReq := &models.UserCreationReq{
		Username: username,
		Email:    email,
		Realname: realname,
		Password: password,
		Comment:  comment,
	}

	params := &user.CreateUserParams{
		User:    userCreationReq,
		Context: ctx,
	}

	resp, err := c.V2Client.V2Client.User.CreateUser(params, c.V2Client.AuthInfo)
	if err != nil {
		return 0, errors.Wrap(err, "failed to create user")
	}

	// Extract user ID from Location header
	location := resp.Location
	if location == "" {
		return 0, errors.New("no location header in create response")
	}

	// Location format: /api/v2.0/users/{id}
	parts := strings.Split(location, "/")
	if len(parts) == 0 {
		return 0, errors.New("invalid location header format")
	}

	userIDStr := parts[len(parts)-1]
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		return 0, errors.Wrap(err, "failed to parse user ID from location header")
	}

	// Update admin status if needed (separate API call)
	if admin {
		sysAdminFlag := &models.UserSysAdminFlag{
			SysadminFlag: admin,
		}
		adminParams := &user.SetUserSysAdminParams{
			UserID:           userID,
			SysadminFlag:     sysAdminFlag,
			Context:          ctx,
		}
		if _, err := c.V2Client.V2Client.User.SetUserSysAdmin(adminParams, c.V2Client.AuthInfo); err != nil {
			// Try to clean up the created user
			_ = c.DeleteUser(ctx, userID)
			return 0, errors.Wrap(err, "failed to set admin flag")
		}
	}

	return userID, nil
}

// UpdateUser updates an existing user
func (c *HarborClient) UpdateUser(ctx context.Context, userID int64, email, realname string, admin bool, comment string) error {
	profile := &models.UserProfile{
		Email:    email,
		Realname: realname,
		Comment:  comment,
	}

	params := &user.UpdateUserProfileParams{
		UserID:  userID,
		Profile: profile,
		Context: ctx,
	}

	if _, err := c.V2Client.V2Client.User.UpdateUserProfile(params, c.V2Client.AuthInfo); err != nil {
		return errors.Wrap(err, "failed to update user profile")
	}

	// Update admin status separately
	sysAdminFlag := &models.UserSysAdminFlag{
		SysadminFlag: admin,
	}
	adminParams := &user.SetUserSysAdminParams{
		UserID:       userID,
		SysadminFlag: sysAdminFlag,
		Context:      ctx,
	}
	if _, err := c.V2Client.V2Client.User.SetUserSysAdmin(adminParams, c.V2Client.AuthInfo); err != nil {
		return errors.Wrap(err, "failed to update admin flag")
	}

	return nil
}

// UpdateUserPassword updates a user's password
func (c *HarborClient) UpdateUserPassword(ctx context.Context, userID int64, newPassword string) error {
	passwordReq := &models.PasswordReq{
		NewPassword: newPassword,
	}

	params := &user.UpdateUserPasswordParams{
		UserID:      userID,
		Password:    passwordReq,
		Context:     ctx,
	}

	_, err := c.V2Client.V2Client.User.UpdateUserPassword(params, c.V2Client.AuthInfo)
	if err != nil {
		// Check if it's a 400 error (password unchanged)
		if httpErr, ok := err.(*user.UpdateUserPasswordBadRequest); ok {
			if httpErr.Code() == http.StatusBadRequest {
				// Password is the same, not an error in our case
				return nil
			}
		}
		return errors.Wrap(err, "failed to update user password")
	}

	return nil
}

// DeleteUser deletes a user
func (c *HarborClient) DeleteUser(ctx context.Context, userID int64) error {
	params := &user.DeleteUserParams{
		UserID:  userID,
		Context: ctx,
	}

	_, err := c.V2Client.V2Client.User.DeleteUser(params, c.V2Client.AuthInfo)
	if err != nil {
		return errors.Wrap(err, "failed to delete user")
	}

	return nil
}