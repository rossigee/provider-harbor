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
	"net/http"
	"strconv"
	"time"

	harborrobot "github.com/goharbor/go-client/pkg/sdk/v2.0/client/robot"
	harbormodels "github.com/goharbor/go-client/pkg/sdk/v2.0/models"
	"github.com/pkg/errors"
	"k8s.io/utils/ptr"
)

// RobotSpec defines the desired state of a Harbor robot account
type RobotSpec struct {
	Name string
	// Level is the robot level: "project" (default) or "system".
	Level       string
	Description *string
	// ProjectID is the numeric Harbor project id a project-level robot is scoped
	// to. It is resolved to the project name at the API boundary because Harbor's
	// robot permission namespace is addressed by project name. A non-numeric value
	// is treated as a project name directly (backward compat). It is optional (and
	// typically empty) for system-level robots.
	ProjectID   *string
	ExpiresIn   *int64
	Permissions []RobotPermission
}

// RobotPermission defines permissions for a robot account
type RobotPermission struct {
	// Namespace is the resource type the access applies to (e.g. "repository").
	Namespace string
	Access    []string
	// Kind and Scope only apply to system-level robots: Kind is the permission's
	// own scope kind ("project" or "system") and Scope is the namespace it is
	// scoped to (a project name for kind "project", or "/" for kind "system").
	// They are ignored for project-level robots.
	Kind  *string
	Scope *string
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

const (
	robotLevelProject = "project"
	robotLevelSystem  = "system"
	// systemScopeAll is Harbor's namespace value for a system-kind permission.
	systemScopeAll = "/"
)

// accessFrom maps one CR permission's actions onto Harbor per-action Access
// entries ({resource, action}), where the resource is the CR Namespace.
func accessFrom(p RobotPermission) []*harbormodels.Access {
	access := make([]*harbormodels.Access, 0, len(p.Access))
	for _, a := range p.Access {
		access = append(access, &harbormodels.Access{
			Resource: p.Namespace,
			Action:   a,
		})
	}
	return access
}

// projectRobotPermissions maps the CR's permission shape onto Harbor's for a
// project-level robot. All CR permissions are collapsed into a single
// project-scoped RobotPermission whose Namespace is the project name.
func projectRobotPermissions(projectName string, perms []RobotPermission) []*harbormodels.RobotPermission {
	if len(perms) == 0 {
		return nil
	}
	access := make([]*harbormodels.Access, 0)
	for _, p := range perms {
		access = append(access, accessFrom(p)...)
	}
	return []*harbormodels.RobotPermission{{
		Kind:      robotLevelProject,
		Namespace: projectName,
		Access:    access,
	}}
}

// systemRobotPermissions maps the CR's permission shape onto Harbor's for a
// system-level robot. Each permission honours its own scope: Kind defaults to
// "system" (Namespace "/") and may instead be "project" with Scope naming the
// project. Permissions sharing a (kind, namespace) are grouped into one Harbor
// RobotPermission.
func systemRobotPermissions(perms []RobotPermission) []*harbormodels.RobotPermission {
	if len(perms) == 0 {
		return nil
	}
	type key struct{ kind, namespace string }
	order := make([]key, 0)
	grouped := make(map[key][]*harbormodels.Access)
	for _, p := range perms {
		kind := ptr.Deref(p.Kind, robotLevelSystem)
		ns := ptr.Deref(p.Scope, "")
		if ns == "" {
			if kind == robotLevelSystem {
				ns = systemScopeAll
			} else {
				ns = ptr.Deref(p.Scope, "")
			}
		}
		k := key{kind: kind, namespace: ns}
		if _, ok := grouped[k]; !ok {
			order = append(order, k)
		}
		grouped[k] = append(grouped[k], accessFrom(p)...)
	}
	out := make([]*harbormodels.RobotPermission, 0, len(order))
	for _, k := range order {
		out = append(out, &harbormodels.RobotPermission{
			Kind:      k.kind,
			Namespace: k.namespace,
			Access:    grouped[k],
		})
	}
	return out
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

// CreateRobot creates a new robot account at the level given by spec.Level
// ("project" by default, or "system"). The returned secret is only ever
// available here (Harbor never returns it again), so the controller must publish
// it as connection details on Create.
//
// Robots are NOT importable: Harbor discloses the secret only at creation, so a
// pre-existing robot of the same name cannot be adopted. A 409 from Harbor is
// surfaced as an actionable error instructing an operator to delete the existing
// robot; the controller never deletes or recreates automatically.
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

	level := spec.Level
	if level == "" {
		level = robotLevelProject
	}

	c.logger.Info("Creating Harbor robot account", "name", spec.Name, "level", level, "projectId", spec.ProjectID)

	var permissions []*harbormodels.RobotPermission
	switch level {
	case robotLevelSystem:
		permissions = systemRobotPermissions(spec.Permissions)
	default:
		// projectId is the numeric Harbor project id (per the field contract); the
		// robot permission namespace needs the project NAME, so resolve id -> name.
		projectName, err := c.resolveProjectName(ctx, ptr.Deref(spec.ProjectID, ""))
		if err != nil {
			return nil, errors.Wrap(err, "cannot resolve project for robot")
		}
		permissions = projectRobotPermissions(projectName, spec.Permissions)
	}

	req := &harbormodels.RobotCreate{
		Name:        spec.Name,
		Description: ptr.Deref(spec.Description, ""),
		Level:       level,
		Duration:    robotDuration(spec.ExpiresIn),
		Permissions: permissions,
	}

	params := harborrobot.NewCreateRobotParams().WithContext(ctx).WithRobot(req)
	resp, err := v2Client.Robot.CreateRobot(ctx, params)
	if err != nil {
		// A robot cannot be imported: Harbor only returns the secret at creation, so
		// a name clash with a pre-existing robot is unrecoverable by the controller.
		// Surface an actionable error rather than silently looping or deleting.
		if isHarborCode(err, http.StatusConflict) {
			return nil, errors.Errorf("robot %q already exists in Harbor and cannot be imported (Harbor discloses the secret only at creation); delete the existing robot to let this resource manage it", spec.Name)
		}
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

	level := spec.Level
	if level == "" {
		level = robotLevelProject
	}

	c.logger.Info("Updating Harbor robot account", "robotId", robotID, "name", spec.Name, "level", level)

	var permissions []*harbormodels.RobotPermission
	switch level {
	case robotLevelSystem:
		permissions = systemRobotPermissions(spec.Permissions)
	default:
		// projectId is the numeric Harbor project id; resolve to the project name for
		// the robot permission namespace.
		projectName, perr := c.resolveProjectName(ctx, ptr.Deref(spec.ProjectID, ""))
		if perr != nil {
			return nil, errors.Wrap(perr, "cannot resolve project for robot")
		}
		permissions = projectRobotPermissions(projectName, spec.Permissions)
	}

	duration := robotDuration(spec.ExpiresIn)
	req := &harbormodels.Robot{
		ID:          id,
		Name:        spec.Name,
		Description: ptr.Deref(spec.Description, ""),
		Level:       level,
		Duration:    &duration,
		Permissions: permissions,
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
