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

	harborproject "github.com/goharbor/go-client/pkg/sdk/v2.0/client/project"
	harborretention "github.com/goharbor/go-client/pkg/sdk/v2.0/client/retention"
	harbormodels "github.com/goharbor/go-client/pkg/sdk/v2.0/models"
	"github.com/pkg/errors"
	"k8s.io/utils/ptr"
)

// RetentionPolicyRule defines a retention rule
type RetentionPolicyRule struct {
	RuleType     string // always, latestPushedK, latestPulledN
	TagSelectors []string
	Parameters   map[string]interface{}
}

// RetentionPolicySpec defines the desired state of a retention policy
type RetentionPolicySpec struct {
	ProjectID   string
	Description *string
	Rules       []RetentionPolicyRule
	Trigger     string // manual, scheduled
	Enabled     *bool
}

// RetentionPolicyStatus represents the status of a retention policy
type RetentionPolicyStatus struct {
	ID           string
	ProjectID    string
	Description  *string
	Enabled      bool
	CreationTime time.Time
	UpdateTime   time.Time
}

func retentionPolicyModel(spec *RetentionPolicySpec, projectIDInt int64) *harbormodels.RetentionPolicy {
	rules := make([]*harbormodels.RetentionRule, 0, len(spec.Rules))
	for _, r := range spec.Rules {
		rule := &harbormodels.RetentionRule{
			Template: r.RuleType,
			Action:   "retain",
		}
		if len(r.TagSelectors) > 0 {
			rule.TagSelectors = make([]*harbormodels.RetentionSelector, len(r.TagSelectors))
			for i, ts := range r.TagSelectors {
				rule.TagSelectors[i] = &harbormodels.RetentionSelector{
					Kind:       "doublestar",
					Decoration: "matches",
					Pattern:    ts,
				}
			}
		}
		if len(r.Parameters) > 0 {
			rule.Params = r.Parameters
		}
		rules = append(rules, rule)
	}

	p := &harbormodels.RetentionPolicy{
		Algorithm: "or",
		Rules:     rules,
		Scope: &harbormodels.RetentionPolicyScope{
			Level: "project",
			Ref:   projectIDInt,
		},
		Trigger: &harbormodels.RetentionRuleTrigger{Kind: "Schedule"},
	}
	return p
}

// retentionPolicyStatusFromModel converts a Harbor SDK RetentionPolicy to our
// internal RetentionPolicyStatus.
// Caveat: Harbor's RetentionPolicy model has no Description, Enabled, CreationTime,
// or UpdateTime — these are synthesised from spec/reconcile context by the caller.
func retentionPolicyStatusFromModel(projectID string, p *harbormodels.RetentionPolicy) *RetentionPolicyStatus {
	if p == nil {
		return &RetentionPolicyStatus{ProjectID: projectID}
	}
	return &RetentionPolicyStatus{
		ID:        strconv.FormatInt(p.ID, 10),
		ProjectID: projectID,
	}
}

// getRetentionPolicyByID fetches a retention policy by its numeric Harbor ID.
// Returns (nil, nil) on 404.
func (c *HarborClient) getRetentionPolicyByID(ctx context.Context, projectID string, id int64) (*RetentionPolicyStatus, error) {
	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	params := harborretention.NewGetRetentionParams().WithContext(ctx).WithID(id)
	resp, err := v2Client.Retention.GetRetention(ctx, params)
	if err != nil {
		if isHarborNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "cannot get Harbor retention policy")
	}
	return retentionPolicyStatusFromModel(projectID, resp.Payload), nil
}

// CreateRetentionPolicy creates a new retention policy using the Harbor SDK.
// Re-reads via the Location header to capture the authoritative numeric ID.
func (c *HarborClient) CreateRetentionPolicy(ctx context.Context, spec *RetentionPolicySpec) (*RetentionPolicyStatus, error) {
	if spec == nil {
		return nil, errors.New("spec is required")
	}
	if spec.ProjectID == "" {
		return nil, errors.New("project ID is required")
	}
	if len(spec.Rules) == 0 {
		return nil, errors.New("at least one rule is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Creating Harbor retention policy",
		"projectId", spec.ProjectID,
		"rulesCount", len(spec.Rules))

	projectIDInt, err := c.resolveProjectID(ctx, spec.ProjectID)
	if err != nil {
		return nil, errors.Wrap(err, "invalid project for retention policy")
	}
	model := retentionPolicyModel(spec, projectIDInt)

	params := harborretention.NewCreateRetentionParams().WithContext(ctx).WithPolicy(model)
	resp, err := v2Client.Retention.CreateRetention(ctx, params)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create Harbor retention policy")
	}

	id, err := idFromLocation(resp.Location)
	if err != nil {
		return nil, errors.Wrap(err, "cannot parse retention policy ID from location header")
	}
	st, err := c.getRetentionPolicyByID(ctx, spec.ProjectID, id)
	if err != nil {
		return nil, err
	}
	if st == nil {
		return nil, errors.New("retention policy created but not yet observable")
	}
	if spec.Enabled != nil {
		st.Enabled = *spec.Enabled
	}
	st.Description = spec.Description
	return st, nil
}

// ListRetentionPolicies returns the retention policy bound to a project, if any.
// Harbor allows at most one retention policy per project; the binding is stored
// as "retention_id" in project metadata. Returns empty slice when none is bound.
func (c *HarborClient) ListRetentionPolicies(ctx context.Context, projectID string) ([]*RetentionPolicyStatus, error) {
	if projectID == "" {
		return nil, errors.New("project ID is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Listing Harbor retention policies", "projectId", projectID)

	ref, isName := projectRef(projectID)
	projParams := harborproject.NewGetProjectParams().WithContext(ctx).WithProjectNameOrID(ref)
	if isName != nil {
		projParams = projParams.WithXIsResourceName(isName)
	}
	projResp, err := v2Client.Project.GetProject(ctx, projParams)
	if err != nil {
		if isHarborNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "cannot get project for retention lookup")
	}

	if projResp.Payload == nil || projResp.Payload.Metadata == nil {
		return nil, nil
	}
	retentionIDStr := ptr.Deref(projResp.Payload.Metadata.RetentionID, "")
	if retentionIDStr == "" {
		return nil, nil
	}

	retentionID, err := strconv.ParseInt(retentionIDStr, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "cannot parse retention policy ID from project metadata")
	}

	st, err := c.getRetentionPolicyByID(ctx, projectID, retentionID)
	if err != nil {
		return nil, err
	}
	if st == nil {
		return nil, nil
	}
	return []*RetentionPolicyStatus{st}, nil
}

// GetRetentionPolicy retrieves a retention policy by numeric ID string.
// Returns (nil, nil) on 404.
func (c *HarborClient) GetRetentionPolicy(ctx context.Context, projectID, policyID string) (*RetentionPolicyStatus, error) {
	if projectID == "" {
		return nil, errors.New("project ID is required")
	}
	if policyID == "" {
		return nil, errors.New("policy ID is required")
	}

	id, err := strconv.ParseInt(policyID, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "invalid retention policy ID")
	}
	return c.getRetentionPolicyByID(ctx, projectID, id)
}

// UpdateRetentionPolicy updates an existing retention policy. Re-reads and
// returns the updated state.
func (c *HarborClient) UpdateRetentionPolicy(ctx context.Context, projectID, policyID string, spec *RetentionPolicySpec) (*RetentionPolicyStatus, error) {
	if projectID == "" {
		return nil, errors.New("project ID is required")
	}
	if policyID == "" {
		return nil, errors.New("policy ID is required")
	}
	if spec == nil {
		return nil, errors.New("spec is required")
	}

	id, err := strconv.ParseInt(policyID, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "invalid retention policy ID")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Updating Harbor retention policy", "projectId", projectID, "policyId", policyID)

	projectIDInt, err := c.resolveProjectID(ctx, spec.ProjectID)
	if err != nil {
		return nil, errors.Wrap(err, "invalid project for retention policy")
	}
	model := retentionPolicyModel(spec, projectIDInt)
	model.ID = id

	params := harborretention.NewUpdateRetentionParams().WithContext(ctx).WithID(id).WithPolicy(model)
	if _, err := v2Client.Retention.UpdateRetention(ctx, params); err != nil {
		return nil, errors.Wrap(err, "cannot update Harbor retention policy")
	}

	st, err := c.getRetentionPolicyByID(ctx, projectID, id)
	if err != nil {
		return nil, err
	}
	if st != nil {
		if spec.Enabled != nil {
			st.Enabled = *spec.Enabled
		}
		st.Description = spec.Description
	}
	return st, nil
}

// DeleteRetentionPolicy deletes a retention policy. Idempotent: 404 is success.
func (c *HarborClient) DeleteRetentionPolicy(ctx context.Context, projectID, policyID string) error {
	if projectID == "" {
		return errors.New("project ID is required")
	}
	if policyID == "" {
		return errors.New("policy ID is required")
	}

	id, err := strconv.ParseInt(policyID, 10, 64)
	if err != nil {
		return errors.Wrap(err, "invalid retention policy ID")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Deleting Harbor retention policy", "projectId", projectID, "policyId", policyID)

	// NOTE: DELETE /retentions/{id} currently panics server-side in Harbor (500),
	// which we hit via this provider during e2e. Tracked upstream:
	//   https://github.com/goharbor/harbor/issues/23345  (fix: PR #23407)
	// The request below is correct; until the fix ships, retention delete fails
	// and the retention e2e is disabled (see examples/e2e/retention.yaml.disabled).
	params := harborretention.NewDeleteRetentionParams().WithContext(ctx).WithID(id)
	if _, err := v2Client.Retention.DeleteRetention(ctx, params); err != nil {
		if isHarborNotFound(err) {
			return nil
		}
		return errors.Wrap(err, "cannot delete Harbor retention policy")
	}
	return nil
}
