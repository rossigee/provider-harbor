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
	"strconv"
	"time"

	harborrobot "github.com/goharbor/go-client/pkg/sdk/v2.0/client/robot"
	harbormodels "github.com/goharbor/go-client/pkg/sdk/v2.0/models"
	"github.com/pkg/errors"
	"k8s.io/utils/ptr"
)

// RobotSpec defines the desired state of a Harbor robot account
type RobotSpec struct {
	Name        string
	Description *string
	// ProjectID is the numeric Harbor project id the robot is scoped to. It is
	// resolved to the project name at the API boundary because Harbor's robot
	// permission namespace is addressed by project name. A non-numeric value is
	// treated as a project name directly (backward compat).
	ProjectID   *string
	ExpiresIn   *int64
	Permissions []RobotPermission
}

// RobotPermission defines permissions for a robot account
type RobotPermission struct {
	Namespace string
	Access    []string
}

// RobotStatus represents the status of a Harbor robot account
type RobotStatus struct {
	ID           string
	Name         string
	Description  *string
	ProjectID    *string
	Secret       string
	ExpiresAt    *time.Time
	CreationTime time.Time
	UpdateTime   time.Time
}

const robotLevelProject = "project"

// robotPermissions maps the CR's permission shape onto Harbor's. Each CR
// permission groups one resource namespace (its Namespace field, e.g.
// "repository") with a set of actions (its Access field, e.g. "pull","push");
// Harbor models these as per-action Access entries ({resource, action}) under a
// single project-scoped RobotPermission whose Namespace is the project name.
func robotPermissions(projectName string, perms []RobotPermission) []*harbormodels.RobotPermission {
	if len(perms) == 0 {
		return nil
	}
	access := make([]*harbormodels.Access, 0)
	for _, p := range perms {
		for _, a := range p.Access {
			access = append(access, &harbormodels.Access{
				Resource: p.Namespace,
				Action:   a,
			})
		}
	}
	return []*harbormodels.RobotPermission{{
		Kind:      robotLevelProject,
		Namespace: projectName,
		Access:    access,
	}}
}

func robotStatusFromModel(r *harbormodels.Robot) *RobotStatus {
	st := &RobotStatus{
		ID:           strconv.FormatInt(r.ID, 10),
		Name:         r.Name,
		Secret:       r.Secret,
		CreationTime: time.Time(r.CreationTime),
		UpdateTime:   time.Time(r.UpdateTime),
	}
	if r.Description != "" {
		st.Description = ptr.To(r.Description)
	}
	if r.ExpiresAt > 0 {
		t := time.Unix(r.ExpiresAt, 0)
		st.ExpiresAt = &t
	}
	// A project robot's permission namespace is its project NAME (Harbor stores
	// the name there, not the numeric id). ProjectID therefore observes the name;
	// callers that compare against the numeric-id spec must resolve first.
	if len(r.Permissions) > 0 && r.Permissions[0] != nil && r.Permissions[0].Namespace != "" {
		st.ProjectID = ptr.To(r.Permissions[0].Namespace)
	}
	return st
}

func robotDuration(expiresIn *int64) int64 {
	if expiresIn != nil {
		return *expiresIn
	}
	return -1 // Harbor: -1 means never expires.
}

// CreateRobot creates a new project-level robot account. The returned secret is
// only ever available here (Harbor never returns it again), so the controller
// must publish it as connection details on Create.
func (c *HarborClient) CreateRobot(ctx context.Context, spec *RobotSpec) (*RobotStatus, error) {
	if spec == nil {
		return nil, errors.New("spec is required")
	}
	if spec.Name == "" {
		return nil, errors.New("robot name is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Creating Harbor robot account", "name", spec.Name, "projectId", spec.ProjectID)

	// projectId is the numeric Harbor project id (per the field contract); the
	// robot permission namespace needs the project NAME, so resolve id -> name.
	projectName, err := c.resolveProjectName(ctx, ptr.Deref(spec.ProjectID, ""))
	if err != nil {
		return nil, errors.Wrap(err, "cannot resolve project for robot")
	}
	req := &harbormodels.RobotCreate{
		Name:        spec.Name,
		Description: ptr.Deref(spec.Description, ""),
		Level:       robotLevelProject,
		Duration:    robotDuration(spec.ExpiresIn),
		Permissions: robotPermissions(projectName, spec.Permissions),
	}

	params := harborrobot.NewCreateRobotParams().WithContext(ctx).WithRobot(req)
	resp, err := v2Client.Robot.CreateRobot(ctx, params)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create Harbor robot")
	}

	created := resp.Payload
	st := &RobotStatus{
		ID:           strconv.FormatInt(created.ID, 10),
		Name:         created.Name,
		Secret:       created.Secret,
		Description:  spec.Description,
		ProjectID:    spec.ProjectID,
		CreationTime: time.Time(created.CreationTime),
		UpdateTime:   time.Time(created.CreationTime),
	}
	if created.ExpiresAt > 0 {
		t := time.Unix(created.ExpiresAt, 0)
		st.ExpiresAt = &t
	}
	return st, nil
}

// ListRobots lists all robot accounts
func (c *HarborClient) ListRobots(ctx context.Context, projectID *string) ([]*RobotStatus, error) {
	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Listing Harbor robot accounts", "projectId", projectID)

	// Observed robots carry the project NAME in their permission namespace, while
	// the spec projectId is the numeric id; resolve to a name to filter by project.
	var wantProject string
	if projectID != nil && *projectID != "" {
		name, err := c.resolveProjectName(ctx, *projectID)
		if err != nil {
			return nil, errors.Wrap(err, "cannot resolve project for robot listing")
		}
		wantProject = name
	}

	pageSize := int64(100)
	params := harborrobot.NewListRobotParams().WithContext(ctx).WithPageSize(&pageSize)
	resp, err := v2Client.Robot.ListRobot(ctx, params)
	if err != nil {
		return nil, errors.Wrap(err, "cannot list Harbor robots")
	}

	robots := make([]*RobotStatus, 0, len(resp.Payload))
	for _, r := range resp.Payload {
		if r == nil {
			continue
		}
		st := robotStatusFromModel(r)
		// When scoped to a project, drop robots from other projects (match by name).
		if wantProject != "" && st.ProjectID != nil && *st.ProjectID != wantProject {
			continue
		}
		robots = append(robots, st)
	}
	return robots, nil
}

// GetRobot retrieves a specific robot account
func (c *HarborClient) GetRobot(ctx context.Context, robotID string) (*RobotStatus, error) {
	if robotID == "" {
		return nil, errors.New("robot ID is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	id, err := strconv.ParseInt(robotID, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "invalid robot ID")
	}

	c.logger.Info("Retrieving Harbor robot account", "robotId", robotID)

	params := harborrobot.NewGetRobotByIDParams().WithContext(ctx).WithRobotID(id)
	resp, err := v2Client.Robot.GetRobotByID(ctx, params)
	if err != nil {
		// A missing robot is reported as (nil, nil), not an error.
		if isHarborNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "cannot get Harbor robot")
	}

	return robotStatusFromModel(resp.Payload), nil
}

// UpdateRobot updates a robot account
func (c *HarborClient) UpdateRobot(ctx context.Context, robotID string, spec *RobotSpec) (*RobotStatus, error) {
	if robotID == "" {
		return nil, errors.New("robot ID is required")
	}
	if spec == nil {
		return nil, errors.New("spec is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	id, err := strconv.ParseInt(robotID, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "invalid robot ID")
	}

	c.logger.Info("Updating Harbor robot account", "robotId", robotID, "name", spec.Name)

	// projectId is the numeric Harbor project id; resolve to the project name for
	// the robot permission namespace.
	projectName, err := c.resolveProjectName(ctx, ptr.Deref(spec.ProjectID, ""))
	if err != nil {
		return nil, errors.Wrap(err, "cannot resolve project for robot")
	}
	duration := robotDuration(spec.ExpiresIn)
	req := &harbormodels.Robot{
		ID:          id,
		Name:        spec.Name,
		Description: ptr.Deref(spec.Description, ""),
		Level:       robotLevelProject,
		Duration:    &duration,
		Permissions: robotPermissions(projectName, spec.Permissions),
	}

	params := harborrobot.NewUpdateRobotParams().WithContext(ctx).WithRobotID(id).WithRobot(req)
	if _, err := v2Client.Robot.UpdateRobot(ctx, params); err != nil {
		return nil, errors.Wrap(err, "cannot update Harbor robot")
	}

	return c.GetRobot(ctx, robotID)
}

// DeleteRobot deletes a robot account
func (c *HarborClient) DeleteRobot(ctx context.Context, robotID string) error {
	if robotID == "" {
		return errors.New("robot ID is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return errors.New("failed to get Harbor v2 client")
	}

	id, err := strconv.ParseInt(robotID, 10, 64)
	if err != nil {
		return errors.Wrap(err, "invalid robot ID")
	}

	c.logger.Info("Deleting Harbor robot account", "robotId", robotID)

	params := harborrobot.NewDeleteRobotParams().WithContext(ctx).WithRobotID(id)
	if _, err := v2Client.Robot.DeleteRobot(ctx, params); err != nil {
		// Already gone is success (idempotent delete).
		if isHarborNotFound(err) {
			return nil
		}
		return errors.Wrap(err, "cannot delete Harbor robot")
	}
	return nil
}
