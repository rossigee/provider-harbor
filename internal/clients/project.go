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
	"strings"
	"time"

	"github.com/go-openapi/runtime"
	harborproject "github.com/goharbor/go-client/pkg/sdk/v2.0/client/project"
	harbormodels "github.com/goharbor/go-client/pkg/sdk/v2.0/models"
	"github.com/pkg/errors"
	"k8s.io/utils/ptr"
)

// CreateProject creates a new Harbor project
// projectMetadata maps the spec's boolean/string settings onto Harbor's
// string-typed ProjectMetadata (Harbor stores these as "true"/"false").
func projectMetadata(spec *ProjectSpec) *harbormodels.ProjectMetadata {
	md := &harbormodels.ProjectMetadata{Public: strconv.FormatBool(spec.Public)}
	if spec.AutoScanImages != nil {
		md.AutoScan = ptr.To(strconv.FormatBool(*spec.AutoScanImages))
	}
	if spec.EnableContentTrust != nil {
		md.EnableContentTrust = ptr.To(strconv.FormatBool(*spec.EnableContentTrust))
	}
	if spec.EnableContentTrustCosign != nil {
		md.EnableContentTrustCosign = ptr.To(strconv.FormatBool(*spec.EnableContentTrustCosign))
	}
	if spec.PreventVulnerableImages != nil {
		md.PreventVul = ptr.To(strconv.FormatBool(*spec.PreventVulnerableImages))
	}
	if spec.Severity != nil {
		md.Severity = spec.Severity
	}
	return md
}

// projectStatusFromModel converts a Harbor API project into our ProjectStatus.
func projectStatusFromModel(p *harbormodels.Project) *ProjectStatus {
	if p == nil {
		return &ProjectStatus{}
	}
	st := &ProjectStatus{
		ID:        strconv.Itoa(int(p.ProjectID)),
		Name:      p.Name,
		OwnerID:   int64(p.OwnerID),
		OwnerName: p.OwnerName,
		RepoCount: p.RepoCount,
	}
	if p.Metadata != nil {
		st.Public = strings.EqualFold(p.Metadata.Public, "true")
	}
	if t := time.Time(p.CreationTime); !t.IsZero() {
		st.CreatedAt = t
	}
	if t := time.Time(p.UpdateTime); !t.IsZero() {
		st.UpdatedAt = t
	}
	return st
}

func (c *HarborClient) CreateProject(ctx context.Context, spec *ProjectSpec) (*ProjectStatus, error) {
	if spec == nil {
		return nil, errors.New("project spec is required")
	}
	if spec.Name == "" {
		return nil, errors.New("project name is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Creating Harbor project", "name", spec.Name, "public", spec.Public)

	public := spec.Public
	req := &harbormodels.ProjectReq{
		ProjectName:  spec.Name,
		Public:       &public,
		Metadata:     projectMetadata(spec),
		StorageLimit: spec.StorageLimit,
		RegistryID:   spec.RegistryID,
	}
	if len(spec.CVEAllowlist) > 0 {
		items := make([]*harbormodels.CVEAllowlistItem, 0, len(spec.CVEAllowlist))
		for _, id := range spec.CVEAllowlist {
			items = append(items, &harbormodels.CVEAllowlistItem{CVEID: id})
		}
		req.CVEAllowlist = &harbormodels.CVEAllowlist{Items: items}
	}

	params := harborproject.NewCreateProjectParams().WithContext(ctx)
	params.Project = req
	if _, err := v2Client.Project.CreateProject(ctx, params); err != nil {
		return nil, errors.Wrap(err, "cannot create Harbor project")
	}

	// Re-read to capture the authoritative project ID and observed state.
	st, err := c.GetProject(ctx, spec.Name)
	if err != nil {
		return nil, err
	}
	if st == nil {
		return nil, errors.New("Harbor project created but not yet observable")
	}
	return st, nil
}

// GetProject retrieves a Harbor project by name or ID
func (c *HarborClient) GetProject(ctx context.Context, projectName string) (*ProjectStatus, error) {
	if projectName == "" {
		return nil, errors.New("project name is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	params := harborproject.NewGetProjectParams().WithContext(ctx)
	params.ProjectNameOrID = projectName
	resp, err := v2Client.Project.GetProject(ctx, params)
	if err != nil {
		// A missing project is reported as (nil, nil), not an error: Harbor's
		// GetProject has no typed 404 in its swagger, so a 404 surfaces as a
		// generic *runtime.APIError. Anything else is a real failure.
		var apiErr *runtime.APIError
		if errors.As(err, &apiErr) && apiErr.Code == http.StatusNotFound {
			return nil, nil
		}
		return nil, errors.Wrap(err, "cannot get Harbor project")
	}

	return projectStatusFromModel(resp.Payload), nil
}

// UpdateProject updates an existing Harbor project
func (c *HarborClient) UpdateProject(ctx context.Context, projectName string, spec *ProjectSpec) (*ProjectStatus, error) {
	if projectName == "" {
		return nil, errors.New("project name is required")
	}
	if spec == nil {
		return nil, errors.New("project spec is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Updating Harbor project", "name", projectName, "public", spec.Public)

	public := spec.Public
	req := &harbormodels.ProjectReq{
		Public:       &public,
		Metadata:     projectMetadata(spec),
		StorageLimit: spec.StorageLimit,
	}

	params := harborproject.NewUpdateProjectParams().WithContext(ctx)
	params.ProjectNameOrID = projectName
	params.Project = req
	if _, err := v2Client.Project.UpdateProject(ctx, params); err != nil {
		return nil, errors.Wrap(err, "cannot update Harbor project")
	}

	return c.GetProject(ctx, projectName)
}

// DeleteProject deletes a Harbor project
func (c *HarborClient) DeleteProject(ctx context.Context, projectName string) error {
	if projectName == "" {
		return errors.New("project name is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Deleting Harbor project", "name", projectName)

	params := harborproject.NewDeleteProjectParams().WithContext(ctx)
	params.ProjectNameOrID = projectName
	if _, err := v2Client.Project.DeleteProject(ctx, params); err != nil {
		// Already gone is success (idempotent delete).
		var apiErr *runtime.APIError
		if errors.As(err, &apiErr) && apiErr.Code == http.StatusNotFound {
			return nil
		}
		return errors.Wrap(err, "cannot delete Harbor project")
	}
	return nil
}

// ListProjects lists Harbor projects
func (c *HarborClient) ListProjects(ctx context.Context) ([]*ProjectStatus, error) {
	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	params := harborproject.NewListProjectsParams().WithContext(ctx)
	resp, err := v2Client.Project.ListProjects(ctx, params)
	if err != nil {
		return nil, errors.Wrap(err, "cannot list Harbor projects")
	}

	out := make([]*ProjectStatus, 0, len(resp.Payload))
	for _, p := range resp.Payload {
		out = append(out, projectStatusFromModel(p))
	}
	return out, nil
}

// ProjectSpec defines the desired state of a Harbor project
type ProjectSpec struct {
	Name                     string            `json:"name"`
	Public                   bool              `json:"public"`
	EnableContentTrust       *bool             `json:"enableContentTrust,omitempty"`
	EnableContentTrustCosign *bool             `json:"enableContentTrustCosign,omitempty"`
	AutoScanImages           *bool             `json:"autoScanImages,omitempty"`
	PreventVulnerableImages  *bool             `json:"preventVulnerableImages,omitempty"`
	Severity                 *string           `json:"severity,omitempty"`
	CVEAllowlist             []string          `json:"cveAllowlist,omitempty"`
	RegistryID               *int64            `json:"registryId,omitempty"`
	StorageLimit             *int64            `json:"storageLimit,omitempty"`
	Metadata                 map[string]string `json:"metadata,omitempty"`
}

// ProjectStatus represents the status of a Harbor project
type ProjectStatus struct {
	ID                  string    `json:"id,omitempty"`
	Name                string    `json:"name"`
	Public              bool      `json:"public"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at,omitempty"`
	OwnerID             int64     `json:"owner_id,omitempty"`
	OwnerName           string    `json:"owner_name,omitempty"`
	RepoCount           int64     `json:"repo_count,omitempty"`
	ChartCount          int64     `json:"chart_count,omitempty"`
	CurrentStorageUsage int64     `json:"current_storage_usage,omitempty"`
}
