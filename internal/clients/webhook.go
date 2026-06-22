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

	harborwebhook "github.com/goharbor/go-client/pkg/sdk/v2.0/client/webhook"
	harbormodels "github.com/goharbor/go-client/pkg/sdk/v2.0/models"
	"github.com/pkg/errors"
	"k8s.io/utils/ptr"
)

// WebhookSpec defines the desired state of a Harbor webhook
type WebhookSpec struct {
	ProjectID      string
	Name           string
	Description    *string
	URL            string
	EventTypes     []string
	AuthHeader     *string
	SkipCertVerify bool
	Enabled        *bool
}

// WebhookStatus represents the status of a Harbor webhook
type WebhookStatus struct {
	ID           string
	ProjectID    string
	Name         string
	Description  *string
	URL          string
	EventTypes   []string
	CreationTime time.Time
	UpdateTime   time.Time
}

// webhookPolicyToStatus converts a Harbor WebhookPolicy model to our WebhookStatus.
func webhookPolicyToStatus(projectID string, p *harbormodels.WebhookPolicy) *WebhookStatus {
	if p == nil {
		return nil
	}
	st := &WebhookStatus{
		ID:           strconv.FormatInt(p.ID, 10),
		ProjectID:    projectID,
		Name:         p.Name,
		EventTypes:   p.EventTypes,
		CreationTime: time.Time(p.CreationTime),
		UpdateTime:   time.Time(p.UpdateTime),
	}
	if p.Description != "" {
		st.Description = ptr.To(p.Description)
	}
	// Collect target URL from first target (Harbor supports one target per policy).
	if len(p.Targets) > 0 && p.Targets[0] != nil {
		st.URL = p.Targets[0].Address
	}
	return st
}

// webhookPolicyReq builds a WebhookPolicy request body from our WebhookSpec.
func webhookPolicyReq(spec *WebhookSpec) *harbormodels.WebhookPolicy {
	enabled := true
	if spec.Enabled != nil {
		enabled = *spec.Enabled
	}
	target := &harbormodels.WebhookTargetObject{
		Type:           "http",
		Address:        spec.URL,
		SkipCertVerify: spec.SkipCertVerify,
	}
	if spec.AuthHeader != nil {
		target.AuthHeader = *spec.AuthHeader
	}
	desc := ""
	if spec.Description != nil {
		desc = *spec.Description
	}
	return &harbormodels.WebhookPolicy{
		Name:        spec.Name,
		Description: desc,
		Enabled:     enabled,
		EventTypes:  spec.EventTypes,
		Targets:     []*harbormodels.WebhookTargetObject{target},
	}
}

// findWebhookByName lists webhook policies for the project and returns the one
// whose name matches. Returns (nil, nil) when not found.
func (c *HarborClient) findWebhookByName(ctx context.Context, projectID, name string) (*WebhookStatus, error) {
	policies, err := c.ListWebhooks(ctx, projectID)
	if err != nil {
		return nil, err
	}
	for _, p := range policies {
		if p.Name == name {
			return p, nil
		}
	}
	return nil, nil
}

// CreateWebhook creates a new webhook policy in the given project. Harbor's
// Create response carries no policy ID, so we re-read via list+match to
// capture the authoritative numeric ID.
func (c *HarborClient) CreateWebhook(ctx context.Context, spec *WebhookSpec) (*WebhookStatus, error) {
	if spec == nil {
		return nil, errors.New("spec is required")
	}
	if spec.ProjectID == "" {
		return nil, errors.New("project ID is required")
	}
	if spec.Name == "" {
		return nil, errors.New("webhook name is required")
	}
	if spec.URL == "" {
		return nil, errors.New("webhook URL is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Creating Harbor webhook", "projectId", spec.ProjectID, "name", spec.Name, "url", spec.URL)

	ref, isName := projectRef(spec.ProjectID)
	params := harborwebhook.NewCreateWebhookPolicyOfProjectParams().
		WithContext(ctx).
		WithProjectNameOrID(ref).
		WithPolicy(webhookPolicyReq(spec))
	if isName != nil {
		params = params.WithXIsResourceName(isName)
	}
	if _, err := v2Client.Webhook.CreateWebhookPolicyOfProject(ctx, params); err != nil {
		return nil, errors.Wrap(err, "cannot create Harbor webhook policy")
	}

	st, err := c.findWebhookByName(ctx, spec.ProjectID, spec.Name)
	if err != nil {
		return nil, err
	}
	if st == nil {
		return nil, errors.New("Harbor webhook created but not yet observable")
	}
	return st, nil
}

// ListWebhooks lists webhook policies for a project.
func (c *HarborClient) ListWebhooks(ctx context.Context, projectID string) ([]*WebhookStatus, error) {
	if projectID == "" {
		return nil, errors.New("project ID is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Listing Harbor webhooks", "projectId", projectID)

	ref, isName := projectRef(projectID)
	params := harborwebhook.NewListWebhookPoliciesOfProjectParams().
		WithContext(ctx).
		WithProjectNameOrID(ref)
	if isName != nil {
		params = params.WithXIsResourceName(isName)
	}
	resp, err := v2Client.Webhook.ListWebhookPoliciesOfProject(ctx, params)
	if err != nil {
		if isHarborNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "cannot list Harbor webhook policies")
	}

	out := make([]*WebhookStatus, 0, len(resp.Payload))
	for _, p := range resp.Payload {
		if p != nil {
			out = append(out, webhookPolicyToStatus(projectID, p))
		}
	}
	return out, nil
}

// GetWebhook retrieves a webhook policy by its numeric ID. Returns (nil, nil)
// when the policy does not exist.
func (c *HarborClient) GetWebhook(ctx context.Context, projectID, webhookID string) (*WebhookStatus, error) {
	if projectID == "" {
		return nil, errors.New("project ID is required")
	}
	if webhookID == "" {
		return nil, errors.New("webhook ID is required")
	}

	id, err := strconv.ParseInt(webhookID, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "invalid webhook policy ID")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Retrieving Harbor webhook", "projectId", projectID, "webhookId", webhookID)

	ref, isName := projectRef(projectID)
	params := harborwebhook.NewGetWebhookPolicyOfProjectParams().
		WithContext(ctx).
		WithProjectNameOrID(ref).
		WithWebhookPolicyID(id)
	if isName != nil {
		params = params.WithXIsResourceName(isName)
	}
	resp, err := v2Client.Webhook.GetWebhookPolicyOfProject(ctx, params)
	if err != nil {
		if isHarborNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "cannot get Harbor webhook policy")
	}

	return webhookPolicyToStatus(projectID, resp.Payload), nil
}

// UpdateWebhook updates a webhook policy by its numeric ID.
func (c *HarborClient) UpdateWebhook(ctx context.Context, projectID, webhookID string, spec *WebhookSpec) (*WebhookStatus, error) {
	if projectID == "" {
		return nil, errors.New("project ID is required")
	}
	if webhookID == "" {
		return nil, errors.New("webhook ID is required")
	}
	if spec == nil {
		return nil, errors.New("spec is required")
	}

	id, err := strconv.ParseInt(webhookID, 10, 64)
	if err != nil {
		return nil, errors.Wrap(err, "invalid webhook policy ID")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Updating Harbor webhook", "projectId", projectID, "webhookId", webhookID, "name", spec.Name)

	ref, isName := projectRef(projectID)
	params := harborwebhook.NewUpdateWebhookPolicyOfProjectParams().
		WithContext(ctx).
		WithProjectNameOrID(ref).
		WithWebhookPolicyID(id).
		WithPolicy(webhookPolicyReq(spec))
	if isName != nil {
		params = params.WithXIsResourceName(isName)
	}
	if _, err := v2Client.Webhook.UpdateWebhookPolicyOfProject(ctx, params); err != nil {
		return nil, errors.Wrap(err, "cannot update Harbor webhook policy")
	}

	return c.GetWebhook(ctx, projectID, webhookID)
}

// DeleteWebhook deletes a webhook policy (idempotent on 404).
func (c *HarborClient) DeleteWebhook(ctx context.Context, projectID, webhookID string) error {
	if projectID == "" {
		return errors.New("project ID is required")
	}
	if webhookID == "" {
		return errors.New("webhook ID is required")
	}

	id, err := strconv.ParseInt(webhookID, 10, 64)
	if err != nil {
		return errors.Wrap(err, "invalid webhook policy ID")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Deleting Harbor webhook", "projectId", projectID, "webhookId", webhookID)

	ref, isName := projectRef(projectID)
	params := harborwebhook.NewDeleteWebhookPolicyOfProjectParams().
		WithContext(ctx).
		WithProjectNameOrID(ref).
		WithWebhookPolicyID(id)
	if isName != nil {
		params = params.WithXIsResourceName(isName)
	}
	if _, err := v2Client.Webhook.DeleteWebhookPolicyOfProject(ctx, params); err != nil {
		if isHarborNotFound(err) {
			return nil
		}
		return errors.Wrap(err, "cannot delete Harbor webhook policy")
	}
	return nil
}
