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
	"strings"
	"time"

	"github.com/go-openapi/strfmt"
	harborscanner "github.com/goharbor/go-client/pkg/sdk/v2.0/client/scanner"
	harbormodels "github.com/goharbor/go-client/pkg/sdk/v2.0/models"
	"github.com/pkg/errors"
)

// ScannerSpec defines the desired state of a Harbor scanner registration
type ScannerSpec struct {
	Name             string  `json:"name"`
	Description      *string `json:"description,omitempty"`
	URL              string  `json:"url"`
	Auth             *string `json:"auth,omitempty"`
	AccessCredential *string `json:"access_credential,omitempty"`
}

// ScannerStatus represents the status of a Harbor scanner registration
type ScannerStatus struct {
	UUID             string    `json:"uuid"`
	Name             string    `json:"name"`
	Description      *string   `json:"description,omitempty"`
	URL              string    `json:"url"`
	Auth             *string   `json:"auth,omitempty"`
	AccessCredential *string   `json:"access_credential,omitempty"`
	CreateTime       time.Time `json:"create_time"`
	UpdateTime       time.Time `json:"update_time"`
}

// scannerStatusFromModel converts a Harbor ScannerRegistration model into our ScannerStatus.
func scannerStatusFromModel(s *harbormodels.ScannerRegistration) *ScannerStatus {
	if s == nil {
		return &ScannerStatus{}
	}
	st := &ScannerStatus{
		UUID: s.UUID,
		Name: s.Name,
		URL:  string(s.URL),
		Auth: &s.Auth,
	}
	if s.Description != "" {
		st.Description = &s.Description
	}
	if s.AccessCredential != "" {
		st.AccessCredential = &s.AccessCredential
	}
	if t := time.Time(s.CreateTime); !t.IsZero() {
		st.CreateTime = t
	}
	if t := time.Time(s.UpdateTime); !t.IsZero() {
		st.UpdateTime = t
	}
	return st
}

// CreateScannerRegistration creates a new Harbor scanner registration.
// Harbor returns the UUID via the Location header; we re-read to get full state.
func (c *HarborClient) CreateScannerRegistration(ctx context.Context, spec *ScannerSpec) (*ScannerStatus, error) {
	if spec == nil {
		return nil, errors.New("scanner spec is required")
	}
	if spec.Name == "" {
		return nil, errors.New("scanner name is required")
	}
	if spec.URL == "" {
		return nil, errors.New("scanner URL is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Creating Harbor scanner registration", "name", spec.Name, "url", spec.URL)

	scannerURL := strfmt.URI(spec.URL)
	req := &harbormodels.ScannerRegistrationReq{
		Name: &spec.Name,
		URL:  &scannerURL,
	}
	if spec.Description != nil {
		req.Description = *spec.Description
	}
	if spec.Auth != nil {
		req.Auth = *spec.Auth
	}
	if spec.AccessCredential != nil {
		req.AccessCredential = *spec.AccessCredential
	}

	createParams := harborscanner.NewCreateScannerParams().WithContext(ctx).WithRegistration(req)
	resp, err := v2Client.Scanner.CreateScanner(ctx, createParams)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create Harbor scanner registration")
	}

	// The UUID is embedded in the Location header path: /api/v2.0/scanners/{uuid}
	location := resp.Location
	uuid := location[strings.LastIndex(location, "/")+1:]

	// Re-read to get authoritative state.
	st, err := c.GetScannerRegistration(ctx, uuid)
	if err != nil {
		return nil, err
	}
	if st == nil {
		return nil, errors.New("Harbor scanner created but not yet observable")
	}
	return st, nil
}

// GetScannerRegistration retrieves a Harbor scanner registration by its UUID.
// Returns (nil, nil) when the registration is absent.
func (c *HarborClient) GetScannerRegistration(ctx context.Context, scannerID string) (*ScannerStatus, error) {
	if scannerID == "" {
		return nil, errors.New("scanner ID is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Retrieving Harbor scanner registration", "id", scannerID)

	params := harborscanner.NewGetScannerParams().WithContext(ctx).WithRegistrationID(scannerID)
	resp, err := v2Client.Scanner.GetScanner(ctx, params)
	if err != nil {
		if isHarborNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "cannot get Harbor scanner registration")
	}

	return scannerStatusFromModel(resp.Payload), nil
}

// UpdateScannerRegistration updates an existing Harbor scanner registration.
func (c *HarborClient) UpdateScannerRegistration(ctx context.Context, scannerID string, spec *ScannerSpec) (*ScannerStatus, error) {
	if scannerID == "" {
		return nil, errors.New("scanner ID is required")
	}
	if spec == nil {
		return nil, errors.New("scanner spec is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Updating Harbor scanner registration", "id", scannerID, "name", spec.Name)

	updaterURL := strfmt.URI(spec.URL)
	req := &harbormodels.ScannerRegistrationReq{
		Name: &spec.Name,
		URL:  &updaterURL,
	}
	if spec.Description != nil {
		req.Description = *spec.Description
	}
	if spec.Auth != nil {
		req.Auth = *spec.Auth
	}
	if spec.AccessCredential != nil {
		req.AccessCredential = *spec.AccessCredential
	}

	updateParams := harborscanner.NewUpdateScannerParams().WithContext(ctx).
		WithRegistrationID(scannerID).
		WithRegistration(req)
	if _, err := v2Client.Scanner.UpdateScanner(ctx, updateParams); err != nil {
		return nil, errors.Wrap(err, "cannot update Harbor scanner registration")
	}

	return c.GetScannerRegistration(ctx, scannerID)
}

// DeleteScannerRegistration deletes a Harbor scanner registration. Idempotent on 404.
func (c *HarborClient) DeleteScannerRegistration(ctx context.Context, scannerID string) error {
	if scannerID == "" {
		return errors.New("scanner ID is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Deleting Harbor scanner registration", "id", scannerID)

	params := harborscanner.NewDeleteScannerParams().WithContext(ctx).WithRegistrationID(scannerID)
	if _, err := v2Client.Scanner.DeleteScanner(ctx, params); err != nil {
		if isHarborNotFound(err) {
			return nil
		}
		return errors.Wrap(err, "cannot delete Harbor scanner registration")
	}
	return nil
}

// ListScannerRegistrations lists Harbor scanner registrations.
func (c *HarborClient) ListScannerRegistrations(ctx context.Context) ([]*ScannerStatus, error) {
	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Listing Harbor scanner registrations")

	params := harborscanner.NewListScannersParams().WithContext(ctx)
	resp, err := v2Client.Scanner.ListScanners(ctx, params)
	if err != nil {
		return nil, errors.Wrap(err, "cannot list Harbor scanner registrations")
	}

	out := make([]*ScannerStatus, 0, len(resp.Payload))
	for _, s := range resp.Payload {
		if s != nil {
			out = append(out, scannerStatusFromModel(s))
		}
	}
	return out, nil
}
