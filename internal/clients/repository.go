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

	harborrepository "github.com/goharbor/go-client/pkg/sdk/v2.0/client/repository"
	harbormodels "github.com/goharbor/go-client/pkg/sdk/v2.0/models"
	"github.com/pkg/errors"
)

// RepositorySpec defines the desired state of a Harbor repository
type RepositorySpec struct {
	ProjectID   string  `json:"projectId"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

// RepositoryStatus represents the status of a Harbor repository
type RepositoryStatus struct {
	ID            string    `json:"id"`
	FullName      string    `json:"fullName"`
	ProjectID     string    `json:"projectId"`
	ArtifactCount int64     `json:"artifactCount"`
	CreationTime  time.Time `json:"creationTime"`
	UpdateTime    time.Time `json:"updateTime"`
	Description   string    `json:"description"`
}

// repositoryStatusFromModel converts a Harbor API Repository model to RepositoryStatus.
func repositoryStatusFromModel(r *harbormodels.Repository) *RepositoryStatus {
	if r == nil {
		return &RepositoryStatus{}
	}
	st := &RepositoryStatus{
		ID:            strconv.FormatInt(r.ID, 10),
		FullName:      r.Name,
		ProjectID:     strconv.FormatInt(r.ProjectID, 10),
		ArtifactCount: r.ArtifactCount,
		Description:   r.Description,
	}
	if r.CreationTime != nil {
		st.CreationTime = time.Time(*r.CreationTime)
	}
	if t := time.Time(r.UpdateTime); !t.IsZero() {
		st.UpdateTime = t
	}
	return st
}

// ListRepositories lists repositories in a Harbor project.
func (c *HarborClient) ListRepositories(ctx context.Context, projectID string) ([]*RepositoryStatus, error) {
	if projectID == "" {
		return nil, errors.New("project ID is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Listing Harbor repositories", "projectId", projectID)

	params := harborrepository.NewListRepositoriesParams().WithContext(ctx).WithProjectName(projectID)
	resp, err := v2Client.Repository.ListRepositories(ctx, params)
	if err != nil {
		if isHarborNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "cannot list Harbor repositories")
	}

	repos := make([]*RepositoryStatus, 0, len(resp.Payload))
	for _, r := range resp.Payload {
		if r != nil {
			repos = append(repos, repositoryStatusFromModel(r))
		}
	}
	return repos, nil
}

// GetRepository retrieves a specific Harbor repository, returning (nil, nil) when absent.
// The repository_name path parameter must be URL-encoded once by the SDK. If the name
// itself contains slashes (e.g. "a/b"), it must be encoded again by the caller so the
// final wire value is double-encoded (a/b -> a%2Fb -> a%252Fb).
func (c *HarborClient) GetRepository(ctx context.Context, projectID, repoName string) (*RepositoryStatus, error) {
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

	c.logger.Info("Retrieving Harbor repository", "projectId", projectID, "name", repoName)

	params := harborrepository.NewGetRepositoryParams().WithContext(ctx).
		WithProjectName(projectID).
		WithRepositoryName(repoName)
	resp, err := v2Client.Repository.GetRepository(ctx, params)
	if err != nil {
		if isHarborNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "cannot get Harbor repository")
	}

	return repositoryStatusFromModel(resp.Payload), nil
}

// UpdateRepository updates a Harbor repository's description.
func (c *HarborClient) UpdateRepository(ctx context.Context, projectID, repoName string, spec *RepositorySpec) (*RepositoryStatus, error) {
	if projectID == "" {
		return nil, errors.New("project ID is required")
	}
	if repoName == "" {
		return nil, errors.New("repository name is required")
	}
	if spec == nil {
		return nil, errors.New("repository spec is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Updating Harbor repository", "projectId", projectID, "name", repoName)

	repo := &harbormodels.Repository{}
	if spec.Description != nil {
		repo.Description = *spec.Description
	}

	params := harborrepository.NewUpdateRepositoryParams().WithContext(ctx).
		WithProjectName(projectID).
		WithRepositoryName(repoName).
		WithRepository(repo)
	if _, err := v2Client.Repository.UpdateRepository(ctx, params); err != nil {
		return nil, errors.Wrap(err, "cannot update Harbor repository")
	}

	return c.GetRepository(ctx, projectID, repoName)
}

// DeleteRepository deletes a Harbor repository (idempotent on not-found).
func (c *HarborClient) DeleteRepository(ctx context.Context, projectID, repoName string) error {
	if projectID == "" {
		return errors.New("project ID is required")
	}
	if repoName == "" {
		return errors.New("repository name is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Deleting Harbor repository", "projectId", projectID, "name", repoName)

	params := harborrepository.NewDeleteRepositoryParams().WithContext(ctx).
		WithProjectName(projectID).
		WithRepositoryName(repoName)
	if _, err := v2Client.Repository.DeleteRepository(ctx, params); err != nil {
		if isHarborNotFound(err) {
			return nil
		}
		return errors.Wrap(err, "cannot delete Harbor repository")
	}
	return nil
}
