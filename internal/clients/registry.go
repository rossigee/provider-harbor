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

	harborregistry "github.com/goharbor/go-client/pkg/sdk/v2.0/client/registry"
	harbormodels "github.com/goharbor/go-client/pkg/sdk/v2.0/models"
	"github.com/pkg/errors"
	"k8s.io/utils/ptr"
)

// RegistrySpec defines the desired state of a Harbor registry
type RegistrySpec struct {
	Name        string              `json:"name"`
	Description *string             `json:"description,omitempty"`
	Type        string              `json:"type"`
	URL         string              `json:"url"`
	Insecure    bool                `json:"insecure"`
	Credential  *RegistryCredential `json:"credential,omitempty"`
}

// RegistryCredential represents registry authentication credentials
type RegistryCredential struct {
	Type         string `json:"type"`
	AccessKey    string `json:"access_key"`
	AccessSecret string `json:"access_secret"`
}

// RegistryStatus represents the status of a Harbor registry
type RegistryStatus struct {
	ID          int64     `json:"id,omitempty"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	Type        string    `json:"type"`
	URL         string    `json:"url"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// registryStatusFromModel converts a Harbor API Registry model to RegistryStatus.
func registryStatusFromModel(r *harbormodels.Registry) *RegistryStatus {
	if r == nil {
		return &RegistryStatus{}
	}
	st := &RegistryStatus{
		ID:   r.ID,
		Name: r.Name,
		Type: r.Type,
		URL:  r.URL,
	}
	if r.Description != "" {
		st.Description = ptr.To(r.Description)
	}
	if t := time.Time(r.CreationTime); !t.IsZero() {
		st.CreatedAt = t
	}
	if t := time.Time(r.UpdateTime); !t.IsZero() {
		st.UpdatedAt = t
	}
	return st
}

// findRegistryByName lists registries filtered by name and returns the first
// exact match. Harbor's registry API is keyed by numeric ID; we use the list
// endpoint (which accepts a name query) to resolve name -> numeric ID.
// Returns (nil, nil) when no registry matches the name.
func (c *HarborClient) findRegistryByName(ctx context.Context, name string) (*harbormodels.Registry, error) {
	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}
	params := harborregistry.NewListRegistriesParams().WithContext(ctx).WithName(ptr.To(name))
	resp, err := v2Client.Registry.ListRegistries(ctx, params)
	if err != nil {
		if isHarborNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "cannot list Harbor registries")
	}
	for _, r := range resp.Payload {
		if r != nil && r.Name == name {
			return r, nil
		}
	}
	return nil, nil
}

// CreateRegistry creates a new Harbor registry.
// Harbor returns only a Location header on create, so we re-read by name after.
func (c *HarborClient) CreateRegistry(ctx context.Context, spec *RegistrySpec) (*RegistryStatus, error) {
	if spec == nil {
		return nil, errors.New("registry spec is required")
	}
	if spec.Name == "" {
		return nil, errors.New("registry name is required")
	}
	if spec.URL == "" {
		return nil, errors.New("registry URL is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Creating Harbor registry", "name", spec.Name, "url", spec.URL, "type", spec.Type)

	req := &harbormodels.Registry{
		Name:     spec.Name,
		Type:     spec.Type,
		URL:      spec.URL,
		Insecure: spec.Insecure,
	}
	if spec.Description != nil {
		req.Description = *spec.Description
	}
	if spec.Credential != nil {
		req.Credential = &harbormodels.RegistryCredential{
			Type:         spec.Credential.Type,
			AccessKey:    spec.Credential.AccessKey,
			AccessSecret: spec.Credential.AccessSecret,
		}
	}

	params := harborregistry.NewCreateRegistryParams().WithContext(ctx).WithRegistry(req)
	if _, err := v2Client.Registry.CreateRegistry(ctx, params); err != nil {
		return nil, errors.Wrap(err, "cannot create Harbor registry")
	}

	// Re-read to get the authoritative ID assigned by Harbor.
	st, err := c.GetRegistry(ctx, spec.Name)
	if err != nil {
		return nil, err
	}
	if st == nil {
		return nil, errors.New("Harbor registry created but not yet observable")
	}
	return st, nil
}

// GetRegistry retrieves a Harbor registry by name, returning (nil, nil) when absent.
// Harbor's registry API is keyed by numeric ID; we list+match by name to resolve.
func (c *HarborClient) GetRegistry(ctx context.Context, registryName string) (*RegistryStatus, error) {
	if registryName == "" {
		return nil, errors.New("registry name is required")
	}

	c.logger.Info("Retrieving Harbor registry", "name", registryName)

	r, err := c.findRegistryByName(ctx, registryName)
	if err != nil {
		return nil, errors.Wrap(err, "cannot get Harbor registry")
	}
	if r == nil {
		return nil, nil
	}
	return registryStatusFromModel(r), nil
}

// UpdateRegistry updates an existing Harbor registry identified by name.
// The numeric ID is resolved via list+match before calling PUT /registries/{id}.
func (c *HarborClient) UpdateRegistry(ctx context.Context, registryName string, spec *RegistrySpec) (*RegistryStatus, error) {
	if registryName == "" {
		return nil, errors.New("registry name is required")
	}
	if spec == nil {
		return nil, errors.New("registry spec is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Updating Harbor registry", "name", registryName, "url", spec.URL, "type", spec.Type)

	r, err := c.findRegistryByName(ctx, registryName)
	if err != nil {
		return nil, errors.Wrap(err, "cannot resolve registry ID for update")
	}
	if r == nil {
		return nil, errors.Errorf("Harbor registry %q not found", registryName)
	}

	upd := &harbormodels.RegistryUpdate{
		Name: ptr.To(spec.Name),
		URL:  ptr.To(spec.URL),
	}
	if spec.Description != nil {
		upd.Description = spec.Description
	}
	upd.Insecure = ptr.To(spec.Insecure)
	if spec.Credential != nil {
		upd.CredentialType = ptr.To(spec.Credential.Type)
		upd.AccessKey = ptr.To(spec.Credential.AccessKey)
		upd.AccessSecret = ptr.To(spec.Credential.AccessSecret)
	}

	params := harborregistry.NewUpdateRegistryParams().WithContext(ctx).WithID(r.ID).WithRegistry(upd)
	if _, err := v2Client.Registry.UpdateRegistry(ctx, params); err != nil {
		return nil, errors.Wrap(err, "cannot update Harbor registry")
	}

	return c.GetRegistry(ctx, registryName)
}

// DeleteRegistry deletes a Harbor registry (idempotent on not-found).
// The numeric ID is resolved via list+match before calling DELETE /registries/{id}.
func (c *HarborClient) DeleteRegistry(ctx context.Context, registryName string) error {
	if registryName == "" {
		return errors.New("registry name is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Deleting Harbor registry", "name", registryName)

	r, err := c.findRegistryByName(ctx, registryName)
	if err != nil {
		return errors.Wrap(err, "cannot resolve registry ID for delete")
	}
	if r == nil {
		// Already gone — idempotent.
		return nil
	}

	params := harborregistry.NewDeleteRegistryParams().WithContext(ctx).WithID(r.ID)
	if _, err := v2Client.Registry.DeleteRegistry(ctx, params); err != nil {
		if isHarborNotFound(err) {
			return nil
		}
		return errors.Wrap(err, "cannot delete Harbor registry")
	}
	return nil
}
