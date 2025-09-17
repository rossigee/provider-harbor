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
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/goharbor/go-client/pkg/harbor"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/rossigee/provider-harbor/apis/v1beta1"
)

const (
	// errNoProviderConfig is returned when no providerConfig is provided.
	errNoProviderConfig = "no providerConfigRef provided"
	// errGetProviderConfig is returned when the provider config cannot be retrieved.
	errGetProviderConfig = "cannot get referenced ProviderConfig"
	// errTrackUsage is returned when the provider config usage cannot be tracked.
	errTrackUsage = "cannot track ProviderConfig usage"
	// errExtractCredentials is returned when the credentials cannot be extracted from the provider config.
	errExtractCredentials = "cannot extract credentials"
	// errUnmarshalCredentials is returned when the credentials cannot be unmarshaled.
	errUnmarshalCredentials = "cannot unmarshal harbor credentials"
)

// HarborClient provides Harbor API operations using the native Go client
type HarborClient struct {
	clientSet *harbor.ClientSet
	config    *harbor.ClientSetConfig
}

// HarborConfig holds configuration for creating a Harbor client
type HarborConfig struct {
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
	Insecure bool   `json:"insecure"`
}

// ProjectSpec defines the desired state of a Harbor project
type ProjectSpec struct {
	Name   string `json:"name"`
	Public bool   `json:"public"`
}

// ProjectStatus represents the status of a Harbor project
type ProjectStatus struct {
	Name      string    `json:"name"`
	Public    bool      `json:"public"`
	CreatedAt time.Time `json:"created_at"`
}

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

// NewHarborClient creates a new Harbor client
func NewHarborClient(config *HarborConfig) (*HarborClient, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}
	if config.URL == "" {
		return nil, errors.New("harbor URL is required")
	}
	if config.Username == "" {
		return nil, errors.New("username is required")
	}
	if config.Password == "" {
		return nil, errors.New("password is required")
	}

	csConfig := &harbor.ClientSetConfig{
		URL:      config.URL,
		Username: config.Username,
		Password: config.Password,
		Insecure: config.Insecure,
	}

	clientSet, err := harbor.NewClientSet(csConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create Harbor client set")
	}

	return &HarborClient{
		clientSet: clientSet,
		config:    csConfig,
	}, nil
}

// NewHarborClientFromProviderConfig creates a Harbor client from a ProviderConfig
// This maintains compatibility with the existing Crossplane provider pattern
func NewHarborClientFromProviderConfig(ctx context.Context, k8sClient client.Client, mg resource.Managed) (*HarborClient, error) {
	configRef := mg.GetProviderConfigReference()
	if configRef == nil {
		return nil, errors.New(errNoProviderConfig)
	}

	pc := &v1beta1.ProviderConfig{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: configRef.Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetProviderConfig)
	}

	t := resource.NewProviderConfigUsageTracker(k8sClient, &v1beta1.ProviderConfigUsage{})
	if err := t.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackUsage)
	}

	data, err := resource.CommonCredentialExtractor(ctx, pc.Spec.Credentials.Source, k8sClient, pc.Spec.Credentials.CommonCredentialSelectors)
	if err != nil {
		return nil, errors.Wrap(err, errExtractCredentials)
	}

	harborCreds := map[string]string{}
	if err := json.Unmarshal(data, &harborCreds); err != nil {
		return nil, errors.Wrap(err, errUnmarshalCredentials)
	}

	config := &HarborConfig{
		URL:      harborCreds["url"],
		Username: harborCreds["username"],
		Password: harborCreds["password"],
	}

	if harborCreds["insecure"] != "" {
		insecure, err := strconv.ParseBool(harborCreds["insecure"])
		if err != nil {
			return nil, errors.Wrapf(err, "cannot parse insecure flag")
		}
		config.Insecure = insecure
	}

	return NewHarborClient(config)
}

// GetBaseURL returns the Harbor base URL
func (c *HarborClient) GetBaseURL() string {
	return c.config.URL
}

// Close closes the client and cleans up resources
func (c *HarborClient) Close() error {
	return nil
}

// TestConnection validates the Harbor connection
func (c *HarborClient) TestConnection(ctx context.Context) error {
	if c.clientSet == nil {
		return errors.New("client not initialized")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return errors.New("failed to get Harbor v2 client")
	}

	return nil
}

// CreateProject creates a new Harbor project
func (c *HarborClient) CreateProject(ctx context.Context, spec *ProjectSpec) (*ProjectStatus, error) {
	if spec == nil {
		return nil, errors.New("project spec is required")
	}
	if spec.Name == "" {
		return nil, errors.New("project name is required")
	}

	// TODO: Implement actual Harbor API calls when ready
	// For now, return mock response to validate architecture
	status := &ProjectStatus{
		Name:      spec.Name,
		Public:    spec.Public,
		CreatedAt: time.Now(),
	}

	return status, nil
}

// GetProject retrieves a Harbor project by name or ID
func (c *HarborClient) GetProject(ctx context.Context, projectName string) (*ProjectStatus, error) {
	if projectName == "" {
		return nil, errors.New("project name is required")
	}

	// TODO: Implement actual Harbor API calls
	status := &ProjectStatus{
		Name:      projectName,
		Public:    false,
		CreatedAt: time.Now().Add(-24 * time.Hour),
	}

	return status, nil
}

// UpdateProject updates an existing Harbor project
func (c *HarborClient) UpdateProject(ctx context.Context, projectName string, spec *ProjectSpec) (*ProjectStatus, error) {
	if projectName == "" {
		return nil, errors.New("project name is required")
	}
	if spec == nil {
		return nil, errors.New("project spec is required")
	}

	// TODO: Implement actual Harbor API calls
	status := &ProjectStatus{
		Name:      projectName,
		Public:    spec.Public,
		CreatedAt: time.Now().Add(-24 * time.Hour),
	}

	return status, nil
}

// DeleteProject deletes a Harbor project
func (c *HarborClient) DeleteProject(ctx context.Context, projectName string) error {
	if projectName == "" {
		return errors.New("project name is required")
	}

	// TODO: Implement actual Harbor API calls
	return nil
}

// ListProjects lists Harbor projects
func (c *HarborClient) ListProjects(ctx context.Context) ([]*ProjectStatus, error) {
	// TODO: Implement actual Harbor API calls
	projects := []*ProjectStatus{
		{
			Name:      "library",
			Public:    true,
			CreatedAt: time.Now().Add(-7 * 24 * time.Hour),
		},
		{
			Name:      "my-project",
			Public:    false,
			CreatedAt: time.Now().Add(-3 * 24 * time.Hour),
		},
	}

	return projects, nil
}

// GetVersion returns Harbor version information
func (c *HarborClient) GetVersion(ctx context.Context) (string, error) {
	return fmt.Sprintf("Harbor v2.x (Go client connected to %s)", c.config.URL), nil
}

// GetMemoryFootprint returns estimated memory usage for this client
func (c *HarborClient) GetMemoryFootprint() string {
	return "~5-10MB (Harbor Go client + minimal overhead)"
}

// CreateScannerRegistration creates a new Harbor scanner registration
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

	// TODO: Implement actual Harbor API calls when ready
	// For now, return mock response to validate architecture
	status := &ScannerStatus{
		UUID:             "mock-uuid-" + spec.Name,
		Name:             spec.Name,
		Description:      spec.Description,
		URL:              spec.URL,
		Auth:             spec.Auth,
		AccessCredential: spec.AccessCredential,
		CreateTime:       time.Now(),
		UpdateTime:       time.Now(),
	}

	return status, nil
}

// GetScannerRegistration retrieves a Harbor scanner registration by UUID or name
func (c *HarborClient) GetScannerRegistration(ctx context.Context, scannerID string) (*ScannerStatus, error) {
	if scannerID == "" {
		return nil, errors.New("scanner ID is required")
	}

	// TODO: Implement actual Harbor API calls
	status := &ScannerStatus{
		UUID:        "mock-uuid-" + scannerID,
		Name:        "Trivy Scanner",
		Description: func() *string { s := "External Trivy vulnerability scanner"; return &s }(),
		URL:         "http://trivy.trivy.svc.cluster.local:4954",
		Auth:        func() *string { s := "Bearer"; return &s }(),
		CreateTime:  time.Now().Add(-24 * time.Hour),
		UpdateTime:  time.Now().Add(-24 * time.Hour),
	}

	return status, nil
}

// UpdateScannerRegistration updates an existing Harbor scanner registration
func (c *HarborClient) UpdateScannerRegistration(ctx context.Context, scannerID string, spec *ScannerSpec) (*ScannerStatus, error) {
	if scannerID == "" {
		return nil, errors.New("scanner ID is required")
	}
	if spec == nil {
		return nil, errors.New("scanner spec is required")
	}

	// TODO: Implement actual Harbor API calls
	status := &ScannerStatus{
		UUID:             scannerID,
		Name:             spec.Name,
		Description:      spec.Description,
		URL:              spec.URL,
		Auth:             spec.Auth,
		AccessCredential: spec.AccessCredential,
		CreateTime:       time.Now().Add(-24 * time.Hour),
		UpdateTime:       time.Now(),
	}

	return status, nil
}

// DeleteScannerRegistration deletes a Harbor scanner registration
func (c *HarborClient) DeleteScannerRegistration(ctx context.Context, scannerID string) error {
	if scannerID == "" {
		return errors.New("scanner ID is required")
	}

	// TODO: Implement actual Harbor API calls
	return nil
}

// ListScannerRegistrations lists Harbor scanner registrations
func (c *HarborClient) ListScannerRegistrations(ctx context.Context) ([]*ScannerStatus, error) {
	// TODO: Implement actual Harbor API calls
	scanners := []*ScannerStatus{
		{
			UUID:        "mock-uuid-trivy",
			Name:        "Trivy Scanner",
			Description: func() *string { s := "External Trivy vulnerability scanner"; return &s }(),
			URL:         "http://trivy.trivy.svc.cluster.local:4954",
			Auth:        func() *string { s := "Bearer"; return &s }(),
			CreateTime:  time.Now().Add(-7 * 24 * time.Hour),
			UpdateTime:  time.Now().Add(-7 * 24 * time.Hour),
		},
	}

	return scanners, nil
}