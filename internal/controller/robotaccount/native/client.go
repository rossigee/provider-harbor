package native

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	harborpkg "github.com/goharbor/go-client/pkg/harbor"
	"github.com/goharbor/go-client/pkg/sdk/v2.0/client"
	"github.com/goharbor/go-client/pkg/sdk/v2.0/client/robot"
	"github.com/goharbor/go-client/pkg/sdk/v2.0/models"
	"github.com/pkg/errors"
)

// NewHarborClient creates a new Harbor client with the provided credentials
func NewHarborClient(ctx context.Context, endpoint, username, password string, insecure bool) (*HarborClient, error) {
	_, err := url.Parse(endpoint)
	if err != nil {
		return nil, errors.Wrap(err, "invalid endpoint URL")
	}

	// Create Harbor client set configuration
	cfg := &harborpkg.ClientSetConfig{
		URL:      endpoint,
		Username: username,
		Password: password,
		Insecure: insecure,
	}

	// Create Harbor client set
	cs, err := harborpkg.NewClientSet(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create Harbor client set")
	}

	return &HarborClient{
		client: cs.V2(),
		ctx:    ctx,
	}, nil
}

// HarborClient wraps the Harbor API client for robot account operations
type HarborClient struct {
	client *client.HarborAPI
	ctx    context.Context
}

// RobotAccountSpec represents the desired state of a robot account
type RobotAccountSpec struct {
	Name        string
	Description string
	Duration    int64
	Level       string
	Permissions []Permission
}

// Permission represents a robot account permission
type Permission struct {
	Kind      string
	Namespace string
	Access    []Access
}

// Access represents an access rule
type Access struct {
	Resource string
	Action   string
}

// CreateRobotAccount creates a new robot account in Harbor
func (c *HarborClient) CreateRobotAccount(spec RobotAccountSpec) (*models.Robot, error) {
	// Convert permissions to Harbor format
	perms := make([]*models.RobotPermission, len(spec.Permissions))
	for i, p := range spec.Permissions {
		access := make([]*models.Access, len(p.Access))
		for j, a := range p.Access {
			access[j] = &models.Access{
				Resource: a.Resource,
				Action:   a.Action,
			}
		}
		perms[i] = &models.RobotPermission{
			Kind:      p.Kind,
			Namespace: p.Namespace,
			Access:    access,
		}
	}

	// Create robot account request
	robotCreate := &models.RobotCreate{
		Name:        spec.Name,
		Description: spec.Description,
		Duration:    spec.Duration,
		Level:       spec.Level,
		Permissions: perms,
	}

	params := robot.NewCreateRobotParams().
		WithContext(c.ctx).
		WithRobot(robotCreate)

	// Create the robot account
	resp, err := c.client.Robot.CreateRobot(c.ctx, params)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create robot account")
	}

	// The response contains a RobotCreated which has the robot info
	if resp.Payload == nil {
		return nil, errors.New("empty response from Harbor API")
	}
	
	// RobotCreated has minimal fields, construct a Robot response
	robot := &models.Robot{
		ID:        resp.Payload.ID,
		Name:      resp.Payload.Name,
		Secret:    resp.Payload.Secret,
		ExpiresAt: resp.Payload.ExpiresAt,
		// These fields are from the input spec
		Level:       spec.Level,
		Description: spec.Description,
		Disable:     false, // Newly created robots are enabled
	}

	return robot, nil
}

// GetRobotAccount retrieves a robot account by ID
func (c *HarborClient) GetRobotAccount(robotID int64) (*models.Robot, error) {
	params := robot.NewGetRobotByIDParams().
		WithContext(c.ctx).
		WithRobotID(robotID)

	resp, err := c.client.Robot.GetRobotByID(c.ctx, params)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get robot account")
	}

	return resp.Payload, nil
}

// UpdateRobotAccount updates an existing robot account
func (c *HarborClient) UpdateRobotAccount(robotID int64, spec RobotAccountSpec) error {
	// Harbor API doesn't support updating robot accounts directly
	// We need to delete and recreate
	return errors.New("robot account updates not supported by Harbor API")
}

// DeleteRobotAccount deletes a robot account
func (c *HarborClient) DeleteRobotAccount(robotID int64) error {
	params := robot.NewDeleteRobotParams().
		WithContext(c.ctx).
		WithRobotID(robotID)

	_, err := c.client.Robot.DeleteRobot(c.ctx, params)
	if err != nil {
		return errors.Wrap(err, "failed to delete robot account")
	}

	return nil
}

// ListRobotAccounts lists all robot accounts
func (c *HarborClient) ListRobotAccounts() ([]*models.Robot, error) {
	params := robot.NewListRobotParams().
		WithContext(c.ctx)

	resp, err := c.client.Robot.ListRobot(c.ctx, params)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list robot accounts")
	}

	return resp.Payload, nil
}

// GetRobotAccountByName finds a robot account by name
func (c *HarborClient) GetRobotAccountByName(name string) (*models.Robot, error) {
	robots, err := c.ListRobotAccounts()
	if err != nil {
		return nil, err
	}

	for _, r := range robots {
		if r.Name == name {
			return r, nil
		}
	}

	return nil, fmt.Errorf("robot account %s not found", name)
}

// ExtractRobotID extracts the robot ID from the external name
// External names are in format "/robots/123"
func ExtractRobotID(externalName string) (int64, error) {
	parts := strings.Split(externalName, "/")
	if len(parts) != 3 || parts[1] != "robots" {
		return 0, fmt.Errorf("invalid external name format: %s", externalName)
	}

	var id int64
	_, err := fmt.Sscanf(parts[2], "%d", &id)
	if err != nil {
		return 0, errors.Wrap(err, "failed to parse robot ID")
	}

	return id, nil
}

