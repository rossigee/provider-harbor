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

	harborartifact "github.com/goharbor/go-client/pkg/sdk/v2.0/client/artifact"
	harbormodels "github.com/goharbor/go-client/pkg/sdk/v2.0/models"
	"github.com/pkg/errors"
)

// ArtifactSpec defines the desired state of a Harbor artifact
type ArtifactSpec struct {
	// ProjectID is the numeric Harbor project id (a project name is also accepted
	// for backward compat). Harbor's artifact endpoints address the project by
	// name in the path, so it is resolved id -> name at the API boundary.
	ProjectID      string
	RepositoryName string
	Reference      string
	Type           *string
}

// ArtifactStatus represents the status of a Harbor artifact
type ArtifactStatus struct {
	ID                 string
	Digest             string
	Size               int64
	PullCount          int64
	CreationTime       time.Time
	UpdateTime         time.Time
	VulnerabilityCount int64
}

// artifactStatusFromModel converts a Harbor SDK Artifact model into ArtifactStatus.
// VulnerabilityCount is the total from the first scan_overview entry found
// (ScanOverview is a map keyed by MIME type); 0 when no scan data exists.
func artifactStatusFromModel(a *harbormodels.Artifact) *ArtifactStatus {
	if a == nil {
		return nil
	}
	st := &ArtifactStatus{
		ID:           strconv.FormatInt(a.ID, 10),
		Digest:       a.Digest,
		Size:         a.Size,
		CreationTime: time.Time(a.PushTime),
		UpdateTime:   time.Time(a.PullTime),
	}
	for _, rep := range a.ScanOverview {
		if rep.Summary != nil {
			for _, v := range rep.Summary.Summary {
				st.VulnerabilityCount += v
			}
		}
		break
	}
	return st
}

// ListArtifacts lists artifacts in a Harbor repository.
func (c *HarborClient) ListArtifacts(ctx context.Context, projectID, repoName string) ([]*ArtifactStatus, error) {
	if projectID == "" {
		return nil, errors.New("project ID is required")
	}
	if repoName == "" {
		return nil, errors.New("repository name is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Listing Harbor artifacts", "projectId", projectID, "repo", repoName)

	projectName, err := c.resolveProjectName(ctx, projectID)
	if err != nil {
		return nil, errors.Wrap(err, "cannot resolve project for artifact listing")
	}

	withScan := true
	params := harborartifact.NewListArtifactsParams().WithContext(ctx).
		WithProjectName(projectName).
		WithRepositoryName(repoName).
		WithWithScanOverview(&withScan)
	resp, err := v2Client.Artifact.ListArtifacts(ctx, params)
	if err != nil {
		if isHarborNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "cannot list Harbor artifacts")
	}

	out := make([]*ArtifactStatus, 0, len(resp.Payload))
	for _, a := range resp.Payload {
		if a != nil {
			out = append(out, artifactStatusFromModel(a))
		}
	}
	return out, nil
}

// GetArtifact retrieves a specific Harbor artifact by project, repository, and
// reference (tag or digest). Returns (nil, nil) if the artifact is not found.
func (c *HarborClient) GetArtifact(ctx context.Context, projectID, repoName, reference string) (*ArtifactStatus, error) {
	if projectID == "" {
		return nil, errors.New("project ID is required")
	}
	if repoName == "" {
		return nil, errors.New("repository name is required")
	}
	if reference == "" {
		return nil, errors.New("reference is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Retrieving Harbor artifact", "projectId", projectID, "repo", repoName, "reference", reference)

	projectName, err := c.resolveProjectName(ctx, projectID)
	if err != nil {
		return nil, errors.Wrap(err, "cannot resolve project for artifact")
	}

	withScan := true
	params := harborartifact.NewGetArtifactParams().WithContext(ctx).
		WithProjectName(projectName).
		WithRepositoryName(repoName).
		WithReference(reference).
		WithWithScanOverview(&withScan)
	resp, err := v2Client.Artifact.GetArtifact(ctx, params)
	if err != nil {
		if isHarborNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "cannot get Harbor artifact")
	}

	return artifactStatusFromModel(resp.Payload), nil
}

// DeleteArtifact deletes a Harbor artifact. Idempotent: 404 is treated as success.
func (c *HarborClient) DeleteArtifact(ctx context.Context, projectID, repoName, reference string) error {
	if projectID == "" {
		return errors.New("project ID is required")
	}
	if repoName == "" {
		return errors.New("repository name is required")
	}
	if reference == "" {
		return errors.New("reference is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Deleting Harbor artifact", "projectId", projectID, "repo", repoName, "reference", reference)

	projectName, err := c.resolveProjectName(ctx, projectID)
	if err != nil {
		return errors.Wrap(err, "cannot resolve project for artifact")
	}

	params := harborartifact.NewDeleteArtifactParams().WithContext(ctx).
		WithProjectName(projectName).
		WithRepositoryName(repoName).
		WithReference(reference)
	if _, err := v2Client.Artifact.DeleteArtifact(ctx, params); err != nil {
		if isHarborNotFound(err) {
			return nil
		}
		return errors.Wrap(err, "cannot delete Harbor artifact")
	}
	return nil
}

// GetArtifactVulnerabilities retrieves an artifact with its scan/vulnerability
// overview, delegating to GetArtifact which already requests with_scan_overview.
// Returns (nil, nil) if the artifact is not found.
func (c *HarborClient) GetArtifactVulnerabilities(ctx context.Context, projectID, repoName, reference string) (*ArtifactStatus, error) {
	return c.GetArtifact(ctx, projectID, repoName, reference)
}
