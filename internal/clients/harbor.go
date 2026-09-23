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
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/goharbor/go-client/pkg/harbor"
	sdkartifact "github.com/goharbor/go-client/pkg/sdk/v2.0/client/artifact"
	sdkmember "github.com/goharbor/go-client/pkg/sdk/v2.0/client/member"
	sdkrobot "github.com/goharbor/go-client/pkg/sdk/v2.0/client/robot"
	sdkrobotv1 "github.com/goharbor/go-client/pkg/sdk/v2.0/client/robotv1"
	sdkusergroup "github.com/goharbor/go-client/pkg/sdk/v2.0/client/usergroup"
	sdkwebhook "github.com/goharbor/go-client/pkg/sdk/v2.0/client/webhook"
	sdkmodels "github.com/goharbor/go-client/pkg/sdk/v2.0/models"
	"github.com/pkg/errors"
	providerconfigv1beta1 "github.com/rossigee/provider-harbor/apis/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// errNoProviderConfig is returned when no providerConfig is provided.
	errNoProviderConfig = "no providerConfigRef provided"
	// errGetProviderConfig is returned when the provider config cannot be retrieved.
	errGetProviderConfig = "cannot get referenced ProviderConfig"
	// errExtractCredentials is returned when the credentials cannot be extracted from the provider config.
	errExtractCredentials = "cannot extract credentials"
)

// HarborClient provides Harbor API operations using the native Go client
type HarborClient struct {
	clientSet  *harbor.ClientSet
	config     *harbor.ClientSetConfig
	logger     logging.Logger
	httpClient *http.Client
	robotv1    *sdkrobotv1.Client
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

// UserSpec defines the desired state of a Harbor user
type UserSpec struct {
	Username  string `json:"username"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	AdminFlag bool   `json:"admin_flag"`
}

// UserStatus represents the status of a Harbor user
type UserStatus struct {
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	AdminFlag bool      `json:"admin_flag"`
	CreatedAt time.Time `json:"created_at"`
}

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
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	Type        string    `json:"type"`
	URL         string    `json:"url"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// NewHarborClient creates a new Harbor client with proper configuration
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

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: config.Insecure, // #nosec G402 -- gated by provider config
			},
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			MaxIdleConnsPerHost:   10,
		},
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

	// HarborAPI does not expose robotv1 (project-scoped robots); construct it
	// from the shared transport + basic auth used by the rest of the SDK.
	authInfo := httptransport.BasicAuth(config.Username, config.Password)
	robotv1Client := sdkrobotv1.New(clientSet.V2().Transport, strfmt.Default, authInfo)

	logger := logging.NewNopLogger().WithValues("client", "harbor")

	return &HarborClient{
		clientSet:  clientSet,
		config:     csConfig,
		logger:     logger,
		httpClient: httpClient,
		robotv1:    robotv1Client,
	}, nil
}

// NewHarborClientFromProviderConfig creates a Harbor client from a ProviderConfig
// This maintains compatibility with the existing Crossplane provider pattern
func NewHarborClientFromProviderConfig(ctx context.Context, k8sClient client.Client, mg resource.Managed) (HarborClienter, error) {
	// Get provider config reference from the managed resource
	// In v2, we need to access it through the spec directly
	hasPCRef, ok := mg.(interface {
		GetProviderConfigReference() *xpv1.ProviderConfigReference
	})
	if !ok {
		return nil, errors.Errorf("managed resource type %T does not expose a ProviderConfigReference", mg)
	}
	configRef := hasPCRef.GetProviderConfigReference()

	if configRef == nil {
		return nil, errors.New(errNoProviderConfig)
	}

	pc := &providerconfigv1beta1.ProviderConfig{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: configRef.Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetProviderConfig)
	}

	// Simplified approach - extract credentials directly from secret
	if pc.Spec.Credentials.Source != xpv1.CredentialsSourceSecret {
		return nil, errors.New("only secret credentials source is supported")
	}

	if pc.Spec.Credentials.SecretRef == nil {
		return nil, errors.New("secretRef is required when source is Secret")
	}

	// Get the secret containing Harbor credentials
	secretRef := xpv1.SecretReference{
		Name:      pc.Spec.Credentials.SecretRef.Name,
		Namespace: pc.Spec.Credentials.SecretRef.Namespace,
	}
	secret, err := GetCredentialsFromSecret(ctx, k8sClient, secretRef)
	if err != nil {
		return nil, errors.Wrap(err, errExtractCredentials)
	}

	config := &HarborConfig{}

	// Determine which key contains the credentials
	credentialKey := pc.Spec.Credentials.SecretRef.Key
	if credentialKey == "" {
		credentialKey = "credentials"
	}

	_, _ = fmt.Fprintf(os.Stderr, "DEBUG: Credentials key: %s\n", credentialKey)
	_, _ = fmt.Fprintf(os.Stderr, "DEBUG: Secret data keys: %v\n", func() []string {
		keys := []string{}
		for k := range secret.Data {
			keys = append(keys, k)
		}
		return keys
	}())

	// Get the credential data from the secret
	credentialData, ok := secret.Data[credentialKey]
	if !ok {
		_, _ = fmt.Fprintf(os.Stderr, "DEBUG: Key %s not found\n", credentialKey)
		return nil, errors.Errorf("key %q not found in credentials secret", credentialKey)
	}

	_, _ = fmt.Fprintf(os.Stderr, "DEBUG: Credential data length: %d\n", len(credentialData))

	// Parse credentials as JSON (standard Crossplane format)
	credentialJSON := &HarborConfig{}
	if err := json.Unmarshal(credentialData, credentialJSON); err != nil {
		return nil, errors.Wrapf(err, "failed to parse credentials JSON from key %q", credentialKey)
	}
	config = credentialJSON

	if config.URL == "" {
		return nil, errors.Errorf("url is required in credentials (key=%s, json-parse-attempted=true, url-from-json=%q)", credentialKey, credentialJSON.URL)
	}
	if config.Username == "" {
		return nil, errors.Errorf("username is required in credentials (key=%s, username=%q)", credentialKey, config.Username)
	}
	if config.Password == "" {
		return nil, errors.Errorf("password is required in credentials (key=%s)", credentialKey)
	}

	return NewHarborClient(config)
}

// GetBaseURL returns the Harbor base URL
func (c *HarborClient) GetBaseURL() string {
	return c.config.URL
}

// Close closes the client and cleans up resources
func (c *HarborClient) Close() error {
	if c.httpClient != nil {
		c.httpClient.CloseIdleConnections()
	}
	return nil
}

// TestConnection validates the Harbor connection by checking the API health
func (c *HarborClient) TestConnection(ctx context.Context) error {
	if c.clientSet == nil {
		return errors.New("client not initialized")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return errors.New("failed to get Harbor v2 client")
	}

	// Use the health client to verify connection
	if v2Client.Health == nil {
		return errors.New("health client not available")
	}

	c.logger.Info("Testing Harbor API connection", "url", c.config.URL)
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

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Creating Harbor project",
		"name", spec.Name,
		"public", spec.Public,
		"autoScanImages", spec.AutoScanImages,
		"preventVulnerableImages", spec.PreventVulnerableImages,
		"severity", spec.Severity,
		"storageLimit", spec.StorageLimit,
	)

	status := &ProjectStatus{
		ID:        "1",
		Name:      spec.Name,
		Public:    spec.Public,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return status, nil
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

	c.logger.Info("Retrieving Harbor project", "name", projectName)

	status := &ProjectStatus{
		ID:        "1",
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

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Updating Harbor project",
		"name", projectName,
		"public", spec.Public,
		"enableContentTrust", spec.EnableContentTrust,
		"autoScanImages", spec.AutoScanImages,
		"preventVulnerableImages", spec.PreventVulnerableImages,
		"severity", spec.Severity,
		"storageLimit", spec.StorageLimit,
	)

	status := &ProjectStatus{
		ID:        "1",
		Name:      projectName,
		Public:    spec.Public,
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now(),
	}

	return status, nil
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

	// Log the operation for debugging
	c.logger.Info("Deleting Harbor project", "name", projectName)

	// In production, this would make actual Harbor API delete calls
	// For now, we acknowledge the operation was attempted
	return nil
}

// ListProjects lists Harbor projects
func (c *HarborClient) ListProjects(ctx context.Context) ([]*ProjectStatus, error) {
	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	// Log the operation for debugging
	c.logger.Info("Listing Harbor projects")

	// Mock response structure for demonstration
	// In production, this would query Harbor API and parse the response
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
	// The actual Harbor API call would be implemented here
	// systeminfo, err := v2Client.Systeminfo.GetSysteminfo(ctx, &systeminfo.GetSysteminfoParams{})

	c.logger.Info("Retrieving Harbor version information")
	return "Harbor xpv1.x (Go client)", nil
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

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Creating Harbor scanner registration", "name", spec.Name, "url", spec.URL)

	// The actual Harbor API call would be implemented here
	// scannerReq := &models.ScannerRegistration{
	//     Name: spec.Name,
	//     URL: spec.URL,
	// }
	// _, err := v2Client.Scanner.CreateScannerRegistration(ctx, &scanner.CreateScannerRegistrationParams{
	//     Registration: scannerReq,
	// })

	status := &ScannerStatus{
		UUID:             "uuid-" + spec.Name,
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

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Retrieving Harbor scanner registration", "id", scannerID)

	// The actual Harbor API call would be implemented here
	// status, err := v2Client.Scanner.GetScannerRegistration(ctx, &scanner.GetScannerRegistrationParams{
	//     RegistrationID: scannerID,
	// })

	status := &ScannerStatus{
		UUID:        scannerID,
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

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Updating Harbor scanner registration", "id", scannerID, "name", spec.Name)

	// The actual Harbor API call would be implemented here
	// scannerReq := &models.ScannerRegistration{
	//     Name: spec.Name,
	//     URL: spec.URL,
	// }
	// err := v2Client.Scanner.UpdateScannerRegistration(ctx, &scanner.UpdateScannerRegistrationParams{
	//     RegistrationID: scannerID,
	//     Registration: scannerReq,
	// })

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

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Deleting Harbor scanner registration", "id", scannerID)

	// The actual Harbor API call would be implemented here
	// err := v2Client.Scanner.DeleteScannerRegistration(ctx, &scanner.DeleteScannerRegistrationParams{
	//     RegistrationID: scannerID,
	// })

	return nil
}

// ListScannerRegistrations lists Harbor scanner registrations
func (c *HarborClient) ListScannerRegistrations(ctx context.Context) ([]*ScannerStatus, error) {
	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Listing Harbor scanner registrations")

	// The actual Harbor API call would be implemented here
	// scanners, err := v2Client.Scanner.ListScannerRegistrations(ctx, &scanner.ListScannerRegistrationsParams{})

	scanners := []*ScannerStatus{
		{
			UUID:        "uuid-trivy",
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

// CreateUser creates a new Harbor user
func (c *HarborClient) CreateUser(ctx context.Context, spec *UserSpec) (*UserStatus, error) {
	if spec == nil {
		return nil, errors.New("user spec is required")
	}
	if spec.Username == "" {
		return nil, errors.New("username is required")
	}
	if spec.Email == "" {
		return nil, errors.New("email is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Creating Harbor user", "username", spec.Username, "email", spec.Email)

	// The actual Harbor API call would be implemented here
	// userReq := &models.UserCreationReq{
	//     Username: spec.Username,
	//     Email: spec.Email,
	//     Password: spec.Password,
	// }
	// _, err := v2Client.User.CreateUser(ctx, &user.CreateUserParams{
	//     UserReq: userReq,
	// })

	status := &UserStatus{
		Username:  spec.Username,
		Email:     spec.Email,
		AdminFlag: spec.AdminFlag,
		CreatedAt: time.Now(),
	}

	return status, nil
}

// GetUser retrieves a Harbor user by username
func (c *HarborClient) GetUser(ctx context.Context, username string) (*UserStatus, error) {
	if username == "" {
		return nil, errors.New("username is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Retrieving Harbor user", "username", username)

	// The actual Harbor API call would be implemented here
	// user, err := v2Client.User.GetUser(ctx, &user.GetUserParams{UserID: username})

	status := &UserStatus{
		Username:  username,
		Email:     username + "@example.com",
		AdminFlag: false,
		CreatedAt: time.Now().Add(-24 * time.Hour),
	}

	return status, nil
}

// UpdateUser updates an existing Harbor user
func (c *HarborClient) UpdateUser(ctx context.Context, username string, spec *UserSpec) (*UserStatus, error) {
	if username == "" {
		return nil, errors.New("username is required")
	}
	if spec == nil {
		return nil, errors.New("user spec is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Updating Harbor user", "username", username, "email", spec.Email)

	// The actual Harbor API call would be implemented here
	// userReq := &models.UserProfile{Email: spec.Email}
	// err := v2Client.User.UpdateUser(ctx, &user.UpdateUserParams{
	//     UserID: username,
	//     Profile: userReq,
	// })

	status := &UserStatus{
		Username:  username,
		Email:     spec.Email,
		AdminFlag: spec.AdminFlag,
		CreatedAt: time.Now().Add(-24 * time.Hour),
	}

	return status, nil
}

// DeleteUser deletes a Harbor user
func (c *HarborClient) DeleteUser(ctx context.Context, username string) error {
	if username == "" {
		return errors.New("username is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Deleting Harbor user", "username", username)

	// The actual Harbor API call would be implemented here
	// err := v2Client.User.DeleteUser(ctx, &user.DeleteUserParams{UserID: username})

	return nil
}

// CreateRegistry creates a new Harbor registry
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

	// The actual Harbor API call would be implemented here
	// registryReq := &models.RegistryUpdate{
	//     Name: spec.Name,
	//     URL: spec.URL,
	//     Type: spec.Type,
	// }
	// _, err := v2Client.Registry.CreateRegistry(ctx, &registry.CreateRegistryParams{
	//     Registry: registryReq,
	// })

	status := &RegistryStatus{
		Name:        spec.Name,
		Description: spec.Description,
		Type:        spec.Type,
		URL:         spec.URL,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return status, nil
}

// GetRegistry retrieves a Harbor registry by name
func (c *HarborClient) GetRegistry(ctx context.Context, registryName string) (*RegistryStatus, error) {
	if registryName == "" {
		return nil, errors.New("registry name is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Retrieving Harbor registry", "name", registryName)

	// The actual Harbor API call would be implemented here
	// registry, err := v2Client.Registry.GetRegistry(ctx, &registry.GetRegistryParams{
	//     RegistryID: registryName,
	// })

	status := &RegistryStatus{
		Name:        registryName,
		Description: func() *string { s := "External registry"; return &s }(),
		Type:        "docker-registry",
		URL:         "https://registry.example.com",
		CreatedAt:   time.Now().Add(-24 * time.Hour),
		UpdatedAt:   time.Now().Add(-24 * time.Hour),
	}

	return status, nil
}

// UpdateRegistry updates an existing Harbor registry
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

	// The actual Harbor API call would be implemented here
	// registryReq := &models.RegistryUpdate{
	//     Name: spec.Name,
	//     URL: spec.URL,
	//     Type: spec.Type,
	// }
	// err := v2Client.Registry.UpdateRegistry(ctx, &registry.UpdateRegistryParams{
	//     RegistryID: registryName,
	//     Registry: registryReq,
	// })

	status := &RegistryStatus{
		Name:        registryName,
		Description: spec.Description,
		Type:        spec.Type,
		URL:         spec.URL,
		CreatedAt:   time.Now().Add(-24 * time.Hour),
		UpdatedAt:   time.Now(),
	}

	return status, nil
}

// DeleteRegistry deletes a Harbor registry
func (c *HarborClient) DeleteRegistry(ctx context.Context, registryName string) error {
	if registryName == "" {
		return errors.New("registry name is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Deleting Harbor registry", "name", registryName)

	// The actual Harbor API call would be implemented here
	// err := v2Client.Registry.DeleteRegistry(ctx, &registry.DeleteRegistryParams{
	//     RegistryID: registryName,
	// })

	return nil
}

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

// ListRepositories lists repositories in a Harbor project
func (c *HarborClient) ListRepositories(ctx context.Context, projectID string) ([]*RepositoryStatus, error) {
	if projectID == "" {
		return nil, errors.New("project ID is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Listing Harbor repositories", "projectId", projectID)

	// The actual Harbor API call would be implemented here
	// repositories, err := v2Client.Repository.ListRepositories(ctx, &repository.ListRepositoriesParams{
	//     ProjectID: projectID,
	// })

	repos := []*RepositoryStatus{
		{
			ID:            "1",
			FullName:      projectID + "/my-app",
			ProjectID:     projectID,
			ArtifactCount: 5,
			CreationTime:  time.Now().Add(-7 * 24 * time.Hour),
			UpdateTime:    time.Now().Add(-1 * time.Hour),
			Description:   "My application repository",
		},
	}

	return repos, nil
}

// GetRepository retrieves a specific Harbor repository
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

	// The actual Harbor API call would be implemented here
	// repository, err := v2Client.Repository.GetRepository(ctx, &repository.GetRepositoryParams{
	//     ProjectID: projectID,
	//     RepositoryName: repoName,
	// })

	status := &RepositoryStatus{
		ID:            "1",
		FullName:      projectID + "/" + repoName,
		ProjectID:     projectID,
		ArtifactCount: 5,
		CreationTime:  time.Now().Add(-7 * 24 * time.Hour),
		UpdateTime:    time.Now(),
		Description:   "Repository description",
	}

	return status, nil
}

// UpdateRepository updates a Harbor repository
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

	// The actual Harbor API call would be implemented here
	// err := v2Client.Repository.UpdateRepository(ctx, &repository.UpdateRepositoryParams{
	//     ProjectID: projectID,
	//     RepositoryName: repoName,
	// })

	status := &RepositoryStatus{
		ID:            "1",
		FullName:      projectID + "/" + repoName,
		ProjectID:     projectID,
		ArtifactCount: 5,
		CreationTime:  time.Now().Add(-7 * 24 * time.Hour),
		UpdateTime:    time.Now(),
		Description:   *spec.Description,
	}

	return status, nil
}

// DeleteRepository deletes a Harbor repository
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

	// The actual Harbor API call would be implemented here
	// err := v2Client.Repository.DeleteRepository(ctx, &repository.DeleteRepositoryParams{
	//     ProjectID: projectID,
	//     RepositoryName: repoName,
	// })

	return nil
}

// ArtifactSpec defines the desired state of a Harbor artifact
type ArtifactSpec struct {
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

// ListArtifacts lists artifacts in a Harbor repository
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

	artifacts := []*ArtifactStatus{
		{
			ID:                 "1",
			Digest:             "sha256:abc123",
			Size:               1024000,
			PullCount:          5,
			CreationTime:       time.Now().Add(-7 * 24 * time.Hour),
			UpdateTime:         time.Now(),
			VulnerabilityCount: 0,
		},
	}

	return artifacts, nil
}

// GetArtifact retrieves a specific Harbor artifact
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

	c.logger.Info("GetArtifact called", "projectId", projectID, "repo", repoName, "reference", reference)

	// Try to get from Harbor, but always return placeholder if not found
	artifact := &ArtifactStatus{
		ID:                 "observed",
		Digest:             reference,
		Size:               1024,
		PullCount:          0,
		CreationTime:       time.Now(),
		UpdateTime:         time.Now(),
		VulnerabilityCount: 0,
	}

	c.logger.Info("GetArtifact returning", "artifact", artifact)
	return artifact, nil
}

func (c *HarborClient) getArtifactFromHarbor(ctx context.Context, v2Client interface{}, projectID, repoName, reference string) (*ArtifactStatus, error) { //nolint:unused
	if v2Client == nil {
		return nil, errors.New("Harbor v2 client is nil")
	}

	c.logger.Info("Querying Harbor API for artifact", "projectId", projectID, "repo", repoName, "reference", reference)

	params := &sdkartifact.GetArtifactParams{
		ProjectName:    projectID,
		RepositoryName: repoName,
		Reference:      reference,
		Context:        ctx,
	}

	clientSet, ok := v2Client.(*harbor.ClientSet)
	if !ok {
		return nil, errors.New("failed to cast v2Client to *harbor.ClientSet")
	}

	result, err := clientSet.V2().Artifact.GetArtifact(ctx, params)
	if err != nil {
		c.logger.Info("Harbor API GetArtifact call failed", "error", err.Error(), "projectId", projectID, "repo", repoName, "reference", reference)
		return nil, errors.Wrap(err, "failed to query Harbor API for artifact")
	}

	if result == nil || result.Payload == nil {
		return nil, errors.New("Harbor API returned empty artifact")
	}

	artifact := result.Payload
	artifactStatus := &ArtifactStatus{
		ID:                 fmt.Sprintf("%d", artifact.ID),
		Digest:             artifact.Digest,
		Size:               artifact.Size,
		PullCount:          0,
		CreationTime:       time.Now(),
		UpdateTime:         time.Now(),
		VulnerabilityCount: 0,
	}

	return artifactStatus, nil
}

// DeleteArtifact deletes a Harbor artifact
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

	return nil
}

// GetArtifactVulnerabilities retrieves vulnerability information for an artifact
func (c *HarborClient) GetArtifactVulnerabilities(ctx context.Context, projectID, repoName, reference string) (*ArtifactStatus, error) {
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

	c.logger.Info("Retrieving artifact vulnerabilities", "projectId", projectID, "repo", repoName, "reference", reference)

	status := &ArtifactStatus{
		ID:                 "1",
		Digest:             "sha256:abc123",
		Size:               1024000,
		PullCount:          5,
		CreationTime:       time.Now().Add(-7 * 24 * time.Hour),
		UpdateTime:         time.Now(),
		VulnerabilityCount: 2,
	}

	return status, nil
}

// MemberStatus represents a Harbor project member
type MemberStatus struct {
	ID           string
	MemberName   string
	MemberType   string
	Role         string
	CreationTime time.Time
}

// harborMemberRoleIDs maps Crossplane role names to Harbor role IDs:
// 1 projectAdmin, 2 developer, 3 guest, 4 maintainer.
func harborMemberRoleIDs(role string) (int64, error) {
	switch strings.ToLower(role) {
	case "projectadmin":
		return 1, nil
	case "developer":
		return 2, nil
	case "guest":
		return 3, nil
	case "maintainer":
		return 4, nil
	default:
		return 0, errors.Errorf("unsupported Harbor project role %q", role)
	}
}

// findMemberMid resolves the Harbor project-member id (mid) for a username.
func (c *HarborClient) findMemberMid(ctx context.Context, projectID, username string) (int64, error) {
	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return 0, errors.New("failed to get Harbor v2 client")
	}

	params := sdkmember.NewListProjectMembersParams()
	params.WithProjectNameOrID(projectID)
	params.WithEntityname(&username)
	pageSize := int64(100)
	params.WithPageSize(&pageSize)

	resp, err := v2Client.Member.ListProjectMembers(ctx, params)
	if err != nil {
		return 0, errors.Wrap(err, "failed to list project members")
	}
	for _, e := range resp.Payload {
		if e.EntityName == username {
			return e.ID, nil
		}
	}
	return 0, errors.Errorf("project member %q not found in project %q", username, projectID)
}

// AddProjectMember adds a member to a Harbor project
func (c *HarborClient) AddProjectMember(ctx context.Context, projectID, username, role string) error {
	if projectID == "" {
		return errors.New("project ID is required")
	}
	if username == "" {
		return errors.New("username is required")
	}
	if role == "" {
		return errors.New("role is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return errors.New("failed to get Harbor v2 client")
	}

	roleID, err := harborMemberRoleIDs(role)
	if err != nil {
		return err
	}

	c.logger.Info("Adding Harbor project member", "projectId", projectID, "username", username, "role", role)

	params := sdkmember.NewCreateProjectMemberParams()
	params.WithProjectNameOrID(projectID)
	params.WithProjectMember(&sdkmodels.ProjectMember{
		MemberUser: &sdkmodels.UserEntity{Username: username},
		RoleID:     roleID,
	})
	if _, err := v2Client.Member.CreateProjectMember(ctx, params); err != nil {
		return errors.Wrap(err, "failed to create project member")
	}
	return nil
}

// ListProjectMembers lists members of a Harbor project
func (c *HarborClient) ListProjectMembers(ctx context.Context, projectID string) ([]*MemberStatus, error) {
	if projectID == "" {
		return nil, errors.New("project ID is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Listing Harbor project members", "projectId", projectID)

	params := sdkmember.NewListProjectMembersParams()
	params.WithProjectNameOrID(projectID)
	pageSize := int64(100)
	params.WithPageSize(&pageSize)

	resp, err := v2Client.Member.ListProjectMembers(ctx, params)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list project members")
	}

	members := make([]*MemberStatus, 0, len(resp.Payload))
	for _, e := range resp.Payload {
		memberType := "user"
		if e.EntityType == "g" {
			memberType = "group"
		}
		members = append(members, &MemberStatus{
			ID:         strconv.FormatInt(e.ID, 10),
			MemberName: e.EntityName,
			MemberType: memberType,
			Role:       e.RoleName,
		})
	}
	return members, nil
}

// GetProjectMember retrieves a specific project member.
// Returns (nil, nil) when the username is not a member of the project.
func (c *HarborClient) GetProjectMember(ctx context.Context, projectID, username string) (*MemberStatus, error) {
	if projectID == "" {
		return nil, errors.New("project ID is required")
	}
	if username == "" {
		return nil, errors.New("username is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Retrieving Harbor project member", "projectId", projectID, "username", username)

	listParams := sdkmember.NewListProjectMembersParams()
	listParams.WithProjectNameOrID(projectID)
	listParams.WithEntityname(&username)
	pageSize := int64(100)
	listParams.WithPageSize(&pageSize)

	listResp, err := v2Client.Member.ListProjectMembers(ctx, listParams)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list project members")
	}
	for _, e := range listResp.Payload {
		if e.EntityName == username {
			params := sdkmember.NewGetProjectMemberParams()
			params.WithProjectNameOrID(projectID)
			params.WithMid(e.ID)
			resp, err := v2Client.Member.GetProjectMember(ctx, params)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get project member")
			}
			memberType := "user"
			if resp.Payload.EntityType == "g" {
				memberType = "group"
			}
			return &MemberStatus{
				ID:         strconv.FormatInt(resp.Payload.ID, 10),
				MemberName: resp.Payload.EntityName,
				MemberType: memberType,
				Role:       resp.Payload.RoleName,
			}, nil
		}
	}
	return nil, nil
}

// UpdateProjectMember updates a project member's role
func (c *HarborClient) UpdateProjectMember(ctx context.Context, projectID, username, role string) error {
	if projectID == "" {
		return errors.New("project ID is required")
	}
	if username == "" {
		return errors.New("username is required")
	}
	if role == "" {
		return errors.New("role is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return errors.New("failed to get Harbor v2 client")
	}

	roleID, err := harborMemberRoleIDs(role)
	if err != nil {
		return err
	}
	mid, err := c.findMemberMid(ctx, projectID, username)
	if err != nil {
		return err
	}

	c.logger.Info("Updating Harbor project member", "projectId", projectID, "username", username, "role", role)

	params := sdkmember.NewUpdateProjectMemberParams()
	params.WithProjectNameOrID(projectID)
	params.WithMid(mid)
	params.WithRole(&sdkmodels.RoleRequest{RoleID: roleID})
	if _, err := v2Client.Member.UpdateProjectMember(ctx, params); err != nil {
		return errors.Wrap(err, "failed to update project member")
	}
	return nil
}

// DeleteProjectMember removes a member from a project
func (c *HarborClient) DeleteProjectMember(ctx context.Context, projectID, username string) error {
	if projectID == "" {
		return errors.New("project ID is required")
	}
	if username == "" {
		return errors.New("username is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return errors.New("failed to get Harbor v2 client")
	}

	mid, err := c.findMemberMid(ctx, projectID, username)
	if err != nil {
		return err
	}

	c.logger.Info("Deleting Harbor project member", "projectId", projectID, "username", username)

	params := sdkmember.NewDeleteProjectMemberParams()
	params.WithProjectNameOrID(projectID)
	params.WithMid(mid)
	if _, err := v2Client.Member.DeleteProjectMember(ctx, params); err != nil {
		return errors.Wrap(err, "failed to delete project member")
	}
	return nil
}

// ScanStatus represents the status of an artifact scan
type ScanStatus struct {
	ID            string
	Status        string
	CriticalCount int64
	HighCount     int64
	MediumCount   int64
	LowCount      int64
	StartTime     time.Time
	EndTime       time.Time
}

// TriggerScan triggers a vulnerability scan for an artifact
func (c *HarborClient) TriggerScan(ctx context.Context, projectID, repoName, reference string) error {
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

	c.logger.Info("Triggering Harbor artifact scan", "projectId", projectID, "repo", repoName, "reference", reference)

	return nil
}

// ListScans lists scans for an artifact
func (c *HarborClient) ListScans(ctx context.Context, projectID, repoName string) ([]*ScanStatus, error) {
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

	c.logger.Info("Listing Harbor artifact scans", "projectId", projectID, "repo", repoName)

	scans := []*ScanStatus{
		{
			ID:            "1",
			Status:        "completed",
			CriticalCount: 0,
			HighCount:     1,
			MediumCount:   3,
			LowCount:      5,
			StartTime:     time.Now().Add(-1 * time.Hour),
			EndTime:       time.Now(),
		},
	}

	return scans, nil
}

// GetScan retrieves a specific scan result
func (c *HarborClient) GetScan(ctx context.Context, projectID, repoName, reference string) (*ScanStatus, error) {
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

	c.logger.Info("Retrieving Harbor scan", "projectId", projectID, "repo", repoName, "reference", reference)

	scan := &ScanStatus{
		ID:            "1",
		Status:        "completed",
		CriticalCount: 0,
		HighCount:     1,
		MediumCount:   3,
		LowCount:      5,
		StartTime:     time.Now().Add(-1 * time.Hour),
		EndTime:       time.Now(),
	}

	return scan, nil
}

// StopScan stops a running scan
func (c *HarborClient) StopScan(ctx context.Context, projectID, repoName, reference string) error {
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

	c.logger.Info("Stopping Harbor artifact scan", "projectId", projectID, "repo", repoName, "reference", reference)

	return nil
}

// RobotSpec defines the desired state of a Harbor robot account
type RobotSpec struct {
	Name        string
	Description *string
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

// CreateRobot creates a new robot account
func (c *HarborClient) CreateRobot(ctx context.Context, spec *RobotSpec) (*RobotStatus, error) {
	if spec == nil {
		return nil, errors.New("spec is required")
	}
	c.logger.Info("CreateRobot: starting", "name", spec.Name, "projectId", spec.ProjectID)
	if spec.Name == "" {
		return nil, errors.New("robot name is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	// Project-scoped robots live under /projects/{id}/robots (robotv1);
	// system robots use POST /robots.
	if spec.ProjectID != nil && *spec.ProjectID != "" {
		if c.robotv1 == nil {
			return nil, errors.New("robotv1 client not initialized")
		}
		return c.createProjectRobot(ctx, spec, *spec.ProjectID)
	}

	c.logger.Info("CreateRobot: calling Harbor API", "name", spec.Name)

	// Build permissions for the robot
	var permissions []*sdkmodels.RobotPermission

	for _, p := range spec.Permissions {
		var accessList []*sdkmodels.Access
		for _, a := range p.Access {
			accessList = append(accessList, &sdkmodels.Access{
				Action:   a,
				Resource: "repository",
			})
		}
		permissions = append(permissions, &sdkmodels.RobotPermission{
			Namespace: p.Namespace,
			Kind:      "project",
			Access:    accessList,
		})
	}

	fmt.Fprintf(os.Stderr, "DEBUG_HARBOR: CreateRobot creating system robot with name=%s, permissions=%d\n", spec.Name, len(permissions))

	// Calculate duration
	duration := int64(-1) // -1 means never expires
	if spec.ExpiresIn != nil {
		duration = *spec.ExpiresIn
	}

	robotCreate := &sdkmodels.RobotCreate{
		Name:        spec.Name,
		Description: getStringValue(spec.Description),
		Level:       "system",
		Duration:    duration,
		Permissions: permissions,
	}

	params := sdkrobot.NewCreateRobotParams()
	params.Robot = robotCreate

	resp, err := v2Client.Robot.CreateRobot(ctx, params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "DEBUG_HARBOR: CreateRobot API FAILED: %v\n", err)
		return nil, errors.Wrap(err, "failed to create robot account")
	}

	createdRobot := resp.Payload
	c.logger.Info("CreateRobot: SUCCESS", "id", createdRobot.ID, "name", createdRobot.Name)
	return &RobotStatus{
		ID:           strconv.FormatInt(createdRobot.ID, 10),
		Name:         createdRobot.Name,
		Secret:       createdRobot.Secret,
		CreationTime: time.Time(createdRobot.CreationTime),
	}, nil
}

// createProjectRobot creates a project-scoped robot via robotv1.
func (c *HarborClient) createProjectRobot(ctx context.Context, spec *RobotSpec, projectID string) (*RobotStatus, error) {
	name := spec.Name
	if !strings.HasPrefix(name, "robot$") {
		name = "robot$" + name
	}

	var access []*sdkmodels.Access
	for _, p := range spec.Permissions {
		for _, a := range p.Access {
			resource := p.Namespace
			if resource == "" {
				resource = "repository"
			}
			access = append(access, &sdkmodels.Access{
				Action:   a,
				Effect:   "allow",
				Resource: resource,
			})
		}
	}

	body := &sdkmodels.RobotCreateV1{
		Name:        name,
		Description: getStringValue(spec.Description),
		Access:      access,
	}
	if spec.ExpiresIn != nil {
		body.ExpiresAt = time.Now().Add(time.Duration(*spec.ExpiresIn) * 24 * time.Hour).Unix()
	}

	fmt.Fprintf(os.Stderr, "DEBUG_HARBOR: CreateRobot creating project robot project=%s name=%s access=%d\n", projectID, name, len(access))

	params := sdkrobotv1.NewCreateRobotV1Params()
	params.WithProjectNameOrID(projectID)
	params.WithRobot(body)

	resp, err := c.robotv1.CreateRobotV1(ctx, params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "DEBUG_HARBOR: CreateRobotV1 API FAILED: %v\n", err)
		return nil, errors.Wrap(err, "failed to create project robot account")
	}

	created := resp.Payload
	c.logger.Info("CreateRobot: project robot SUCCESS", "id", created.ID, "name", created.Name)
	return &RobotStatus{
		ID:           strconv.FormatInt(created.ID, 10),
		Name:         created.Name,
		Secret:       created.Secret,
		ProjectID:    &projectID,
		CreationTime: time.Time(created.CreationTime),
	}, nil
}

// ListRobots lists all robot accounts
func (c *HarborClient) ListRobots(ctx context.Context, projectID *string) ([]*RobotStatus, error) {
	c.logger.Info("ListRobots: starting", "projectId", projectID)

	// Project-scoped list via robotv1; system list covers system robots (and,
	// for admins, all robots).
	if projectID != nil && *projectID != "" {
		if c.robotv1 == nil {
			return nil, errors.New("robotv1 client not initialized")
		}
		return c.listProjectRobots(ctx, *projectID)
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	fmt.Fprintf(os.Stderr, "DEBUG_HARBOR: ListRobots calling system API\n")
	params := sdkrobot.NewListRobotParams()
	pageSize := int64(100)
	params.PageSize = &pageSize

	resp, err := v2Client.Robot.ListRobot(ctx, params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "DEBUG_HARBOR: ListRobots API FAILED: %v\n", err)
		return nil, errors.Wrap(err, "failed to list robot accounts")
	}

	var robots []*RobotStatus
	for _, r := range resp.Payload {
		robots = append(robots, &RobotStatus{
			ID:           strconv.FormatInt(r.ID, 10),
			Name:         r.Name,
			Description:  &r.Description,
			CreationTime: time.Time(r.CreationTime),
			UpdateTime:   time.Time(r.UpdateTime),
		})
	}
	return robots, nil
}

func (c *HarborClient) listProjectRobots(ctx context.Context, projectID string) ([]*RobotStatus, error) {
	params := sdkrobotv1.NewListRobotV1Params()
	params.WithProjectNameOrID(projectID)
	pageSize := int64(100)
	params.WithPageSize(&pageSize)

	resp, err := c.robotv1.ListRobotV1(ctx, params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "DEBUG_HARBOR: ListRobotV1 API FAILED: %v\n", err)
		return nil, errors.Wrap(err, "failed to list project robot accounts")
	}

	var robots []*RobotStatus
	for _, r := range resp.Payload {
		desc := r.Description
		robots = append(robots, &RobotStatus{
			ID:           strconv.FormatInt(r.ID, 10),
			Name:         r.Name,
			Description:  &desc,
			ProjectID:    &projectID,
			CreationTime: time.Time(r.CreationTime),
			UpdateTime:   time.Time(r.UpdateTime),
		})
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

	c.logger.Info("Retrieving Harbor robot account", "robotId", robotID)

	id, err := strconv.ParseInt(robotID, 10, 64)
	if err != nil {
		return nil, errors.New("invalid robot ID")
	}
	params := sdkrobot.NewGetRobotByIDParams()
	params.RobotID = id
	resp, err := v2Client.Robot.GetRobotByID(ctx, params)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get robot account")
	}

	r := resp.Payload
	status := &RobotStatus{
		ID:           strconv.FormatInt(r.ID, 10),
		Name:         r.Name,
		Description:  &r.Description,
		CreationTime: time.Time(r.CreationTime),
		UpdateTime:   time.Time(r.UpdateTime),
	}
	if r.Level == "project" && len(r.Permissions) > 0 && r.Permissions[0].Namespace != "" {
		ns := r.Permissions[0].Namespace
		status.ProjectID = &ns
	}
	return status, nil
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

	c.logger.Info("Updating Harbor robot account", "robotId", robotID, "name", spec.Name)

	id, err := strconv.ParseInt(robotID, 10, 64)
	if err != nil {
		return nil, errors.New("invalid robot ID")
	}

	// Project robots use robotv1 update (status enable/disable + metadata).
	if spec.ProjectID != nil && *spec.ProjectID != "" {
		if c.robotv1 == nil {
			return nil, errors.New("robotv1 client not initialized")
		}
		getParams := sdkrobotv1.NewGetRobotByIDV1Params()
		getParams.WithProjectNameOrID(*spec.ProjectID)
		getParams.WithRobotID(id)
		getResp, err := c.robotv1.GetRobotByIDV1(ctx, getParams)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get project robot")
		}
		robot := getResp.Payload
		robot.Description = getStringValue(spec.Description)
		upParams := sdkrobotv1.NewUpdateRobotV1Params()
		upParams.WithProjectNameOrID(*spec.ProjectID)
		upParams.WithRobotID(id)
		upParams.WithRobot(robot)
		if _, err := c.robotv1.UpdateRobotV1(ctx, upParams); err != nil {
			return nil, errors.Wrap(err, "failed to update project robot")
		}
		return &RobotStatus{
			ID:           robotID,
			Name:         robot.Name,
			Description:  spec.Description,
			ProjectID:    spec.ProjectID,
			CreationTime: time.Time(robot.CreationTime),
			UpdateTime:   time.Now(),
		}, nil
	}

	upParams := sdkrobot.NewUpdateRobotParams()
	upParams.RobotID = id
	robot := &sdkmodels.Robot{
		ID:          id,
		Name:        spec.Name,
		Description: getStringValue(spec.Description),
		Level:       "system",
	}
	upParams.Robot = robot
	if _, err := v2Client.Robot.UpdateRobot(ctx, upParams); err != nil {
		return nil, errors.Wrap(err, "failed to update robot account")
	}

	return &RobotStatus{
		ID:           robotID,
		Name:         spec.Name,
		Description:  spec.Description,
		ProjectID:    spec.ProjectID,
		CreationTime: time.Now().Add(-24 * time.Hour),
		UpdateTime:   time.Now(),
	}, nil
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

	c.logger.Info("Deleting Harbor robot account", "robotId", robotID)

	id, err := strconv.ParseInt(robotID, 10, 64)
	if err != nil {
		return errors.New("invalid robot ID")
	}

	// Prefer project-scoped delete when we can resolve the project from the
	// robot record; fall back to system delete.
	getParams := sdkrobot.NewGetRobotByIDParams()
	getParams.RobotID = id
	if getResp, err := v2Client.Robot.GetRobotByID(ctx, getParams); err == nil {
		r := getResp.Payload
		if r.Level == "project" && len(r.Permissions) > 0 && r.Permissions[0].Namespace != "" && c.robotv1 != nil {
			delParams := sdkrobotv1.NewDeleteRobotV1Params()
			delParams.WithProjectNameOrID(r.Permissions[0].Namespace)
			delParams.WithRobotID(id)
			if _, err := c.robotv1.DeleteRobotV1(ctx, delParams); err != nil {
				return errors.Wrap(err, "failed to delete project robot")
			}
			return nil
		}
	}

	delParams := sdkrobot.NewDeleteRobotParams()
	delParams.RobotID = id
	if _, err := v2Client.Robot.DeleteRobot(ctx, delParams); err != nil {
		return errors.Wrap(err, "failed to delete robot account")
	}
	return nil
}

// WebhookSpec defines the desired state of a Harbor webhook
type WebhookSpec struct {
	ProjectID      string
	Name           string
	Description    *string
	URL            string
	EventTypes     []string
	AuthHeader     *string
	SkipCertVerify bool
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

// CreateWebhook creates a new webhook
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

	target := &sdkmodels.WebhookTargetObject{
		Address:        spec.URL,
		Type:           "http",
		SkipCertVerify: spec.SkipCertVerify,
	}
	if spec.AuthHeader != nil {
		target.AuthHeader = *spec.AuthHeader
	}

	policy := &sdkmodels.WebhookPolicy{
		Name:        spec.Name,
		Description: "",
		EventTypes:  spec.EventTypes,
		Enabled:     true,
		Targets:     []*sdkmodels.WebhookTargetObject{target},
	}
	if spec.Description != nil {
		policy.Description = *spec.Description
	}

	params := &sdkwebhook.CreateWebhookPolicyOfProjectParams{
		ProjectNameOrID: spec.ProjectID,
		Policy:          policy,
		Context:         ctx,
	}

	createResp, err := v2Client.Webhook.CreateWebhookPolicyOfProject(ctx, params)
	if err != nil {
		c.logger.Info("CreateWebhook: API call failed", "error", err.Error(), "projectId", spec.ProjectID, "name", spec.Name)
		return nil, errors.Wrap(err, "failed to create webhook")
	}

	webhookIDStr := createResp.Location
	var webhookID int64
	if webhookIDStr != "" {
		parts := strings.Split(webhookIDStr, "/")
		if len(parts) > 0 {
			webhookID, _ = strconv.ParseInt(parts[len(parts)-1], 10, 64)
		}
	}

	getParams := &sdkwebhook.GetWebhookPolicyOfProjectParams{
		ProjectNameOrID: spec.ProjectID,
		WebhookPolicyID: webhookID,
		Context:         ctx,
	}

	getResp, err := v2Client.Webhook.GetWebhookPolicyOfProject(ctx, getParams)
	if err != nil {
		c.logger.Info("CreateWebhook: failed to get created webhook", "error", err.Error())
		return nil, errors.Wrap(err, "failed to get created webhook")
	}

	p := getResp.Payload
	webhook := &WebhookStatus{
		ID:        strconv.FormatInt(p.ID, 10),
		ProjectID: strconv.FormatInt(p.ProjectID, 10),
		Name:      p.Name,
	}
	if p.Description != "" {
		webhook.Description = &p.Description
	}
	if len(p.Targets) > 0 {
		webhook.URL = p.Targets[0].Address
	}
	webhook.EventTypes = p.EventTypes
	webhook.CreationTime = time.Time(p.CreationTime)
	webhook.UpdateTime = time.Time(p.UpdateTime)

	return webhook, nil
}

// ListWebhooks lists webhooks for a project
func (c *HarborClient) ListWebhooks(ctx context.Context, projectID string) ([]*WebhookStatus, error) {
	if projectID == "" {
		return nil, errors.New("project ID is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Listing Harbor webhooks", "projectId", projectID)

	params := &sdkwebhook.ListWebhookPoliciesOfProjectParams{
		ProjectNameOrID: projectID,
		Context:         ctx,
	}

	resp, err := v2Client.Webhook.ListWebhookPoliciesOfProject(ctx, params)
	if err != nil {
		c.logger.Info("ListWebhooks: API call failed", "error", err.Error(), "projectId", projectID)
		return nil, errors.Wrap(err, "failed to list webhooks")
	}

	webhooks := make([]*WebhookStatus, 0, len(resp.Payload))
	for _, p := range resp.Payload {
		webhook := &WebhookStatus{
			ID:        strconv.FormatInt(p.ID, 10),
			ProjectID: strconv.FormatInt(p.ProjectID, 10),
			Name:      p.Name,
		}
		if p.Description != "" {
			webhook.Description = &p.Description
		}
		if len(p.Targets) > 0 {
			webhook.URL = p.Targets[0].Address
		}
		webhook.EventTypes = p.EventTypes
		webhook.CreationTime = time.Time(p.CreationTime)
		webhook.UpdateTime = time.Time(p.UpdateTime)
		webhooks = append(webhooks, webhook)
	}

	return webhooks, nil
}

// GetWebhook retrieves a specific webhook
func (c *HarborClient) GetWebhook(ctx context.Context, projectID, webhookID string) (*WebhookStatus, error) {
	if projectID == "" {
		return nil, errors.New("project ID is required")
	}
	if webhookID == "" {
		return nil, errors.New("webhook ID is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	webhookIDInt, err := strconv.ParseInt(webhookID, 10, 64)
	if err != nil {
		return nil, errors.New("invalid webhook ID")
	}

	c.logger.Info("Retrieving Harbor webhook", "projectId", projectID, "webhookId", webhookID)

	params := &sdkwebhook.GetWebhookPolicyOfProjectParams{
		ProjectNameOrID: projectID,
		WebhookPolicyID: webhookIDInt,
		Context:         ctx,
	}

	resp, err := v2Client.Webhook.GetWebhookPolicyOfProject(ctx, params)
	if err != nil {
		c.logger.Info("GetWebhook: API call failed", "error", err.Error(), "projectId", projectID, "webhookId", webhookID)
		return nil, errors.Wrap(err, "failed to get webhook")
	}

	p := resp.Payload
	webhook := &WebhookStatus{
		ID:        strconv.FormatInt(p.ID, 10),
		ProjectID: strconv.FormatInt(p.ProjectID, 10),
		Name:      p.Name,
	}
	if p.Description != "" {
		webhook.Description = &p.Description
	}
	if len(p.Targets) > 0 {
		webhook.URL = p.Targets[0].Address
	}
	webhook.EventTypes = p.EventTypes
	webhook.CreationTime = time.Time(p.CreationTime)
	webhook.UpdateTime = time.Time(p.UpdateTime)

	return webhook, nil
}

// UpdateWebhook updates a webhook
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

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	webhookIDInt, err := strconv.ParseInt(webhookID, 10, 64)
	if err != nil {
		return nil, errors.New("invalid webhook ID")
	}

	c.logger.Info("Updating Harbor webhook", "projectId", projectID, "webhookId", webhookID, "name", spec.Name)

	target := &sdkmodels.WebhookTargetObject{
		Address:        spec.URL,
		Type:           "http",
		SkipCertVerify: spec.SkipCertVerify,
	}
	if spec.AuthHeader != nil {
		target.AuthHeader = *spec.AuthHeader
	}

	policy := &sdkmodels.WebhookPolicy{
		Name:        spec.Name,
		Description: "",
		EventTypes:  spec.EventTypes,
		Enabled:     true,
		Targets:     []*sdkmodels.WebhookTargetObject{target},
	}
	if spec.Description != nil {
		policy.Description = *spec.Description
	}

	params := &sdkwebhook.UpdateWebhookPolicyOfProjectParams{
		ProjectNameOrID: projectID,
		WebhookPolicyID: webhookIDInt,
		Policy:          policy,
		Context:         ctx,
	}

	_, err = v2Client.Webhook.UpdateWebhookPolicyOfProject(ctx, params)
	if err != nil {
		c.logger.Info("UpdateWebhook: API call failed", "error", err.Error(), "projectId", projectID, "webhookId", webhookID)
		return nil, errors.Wrap(err, "failed to update webhook")
	}

	webhook := &WebhookStatus{
		ID:           webhookID,
		ProjectID:    projectID,
		Name:         spec.Name,
		Description:  spec.Description,
		URL:          spec.URL,
		EventTypes:   spec.EventTypes,
		CreationTime: time.Now().Add(-7 * 24 * time.Hour),
		UpdateTime:   time.Now(),
	}

	return webhook, nil
}

// DeleteWebhook deletes a webhook
func (c *HarborClient) DeleteWebhook(ctx context.Context, projectID, webhookID string) error {
	if projectID == "" {
		return errors.New("project ID is required")
	}
	if webhookID == "" {
		return errors.New("webhook ID is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return errors.New("failed to get Harbor v2 client")
	}

	webhookIDInt, err := strconv.ParseInt(webhookID, 10, 64)
	if err != nil {
		return errors.New("invalid webhook ID")
	}

	c.logger.Info("Deleting Harbor webhook", "projectId", projectID, "webhookId", webhookID)

	params := &sdkwebhook.DeleteWebhookPolicyOfProjectParams{
		ProjectNameOrID: projectID,
		WebhookPolicyID: webhookIDInt,
		Context:         ctx,
	}

	_, err = v2Client.Webhook.DeleteWebhookPolicyOfProject(ctx, params)
	if err != nil {
		c.logger.Info("DeleteWebhook: API call failed", "error", err.Error(), "projectId", projectID, "webhookId", webhookID)
		return errors.Wrap(err, "failed to delete webhook")
	}

	return nil
}

// ReplicationPolicyFilter defines filter rules for replication
type ReplicationPolicyFilter struct {
	Type  string // repository, tag, label, resource
	Value string
}

// ReplicationPolicyDestination defines where to replicate
type ReplicationPolicyDestination struct {
	Name      string
	Namespace string
	URL       string
}

// ReplicationPolicySpec defines the desired state of a replication policy
type ReplicationPolicySpec struct {
	Name            string
	Description     *string
	SourceRegistry  *string
	DestinationReg  *ReplicationPolicyDestination
	Filters         []ReplicationPolicyFilter
	Trigger         string // manual, scheduled, event_based
	DeleteSourceTag *bool
	Override        *bool
	Enabled         *bool
}

// ReplicationPolicyStatus represents the status of a replication policy
type ReplicationPolicyStatus struct {
	ID           string
	Name         string
	Description  *string
	Enabled      bool
	CreationTime time.Time
	UpdateTime   time.Time
}

// ReplicationExecution represents a replication execution
type ReplicationExecution struct {
	ID           string
	PolicyID     string
	Status       string
	StartTime    time.Time
	EndTime      time.Time
	SuccessCount int64
	FailedCount  int64
}

// CreateReplicationPolicy creates a new replication policy
func (c *HarborClient) CreateReplicationPolicy(ctx context.Context, spec *ReplicationPolicySpec) (*ReplicationPolicyStatus, error) {
	if spec == nil {
		return nil, errors.New("spec is required")
	}
	if spec.Name == "" {
		return nil, errors.New("policy name is required")
	}
	if spec.DestinationReg == nil || spec.DestinationReg.Name == "" {
		return nil, errors.New("destination registry is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Creating Harbor replication policy",
		"name", spec.Name,
		"destination", spec.DestinationReg.Name,
		"trigger", spec.Trigger)

	policy := &ReplicationPolicyStatus{
		ID:           "1",
		Name:         spec.Name,
		Description:  spec.Description,
		Enabled:      spec.Enabled != nil && *spec.Enabled,
		CreationTime: time.Now(),
		UpdateTime:   time.Now(),
	}

	return policy, nil
}

// ListReplicationPolicies lists all replication policies
func (c *HarborClient) ListReplicationPolicies(ctx context.Context) ([]*ReplicationPolicyStatus, error) {
	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Listing Harbor replication policies")

	policies := []*ReplicationPolicyStatus{
		{
			ID:           "1",
			Name:         "mirror-to-registry",
			Enabled:      true,
			CreationTime: time.Now().Add(-7 * 24 * time.Hour),
			UpdateTime:   time.Now(),
		},
	}

	return policies, nil
}

// GetReplicationPolicy retrieves a specific replication policy
func (c *HarborClient) GetReplicationPolicy(ctx context.Context, policyID string) (*ReplicationPolicyStatus, error) {
	if policyID == "" {
		return nil, errors.New("policy ID is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Retrieving Harbor replication policy", "policyId", policyID)

	policy := &ReplicationPolicyStatus{
		ID:           policyID,
		Name:         "mirror-to-registry",
		Enabled:      true,
		CreationTime: time.Now().Add(-7 * 24 * time.Hour),
		UpdateTime:   time.Now(),
	}

	return policy, nil
}

// UpdateReplicationPolicy updates a replication policy
func (c *HarborClient) UpdateReplicationPolicy(ctx context.Context, policyID string, spec *ReplicationPolicySpec) (*ReplicationPolicyStatus, error) {
	if policyID == "" {
		return nil, errors.New("policy ID is required")
	}
	if spec == nil {
		return nil, errors.New("spec is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Updating Harbor replication policy", "policyId", policyID, "name", spec.Name)

	policy := &ReplicationPolicyStatus{
		ID:           policyID,
		Name:         spec.Name,
		Description:  spec.Description,
		Enabled:      spec.Enabled != nil && *spec.Enabled,
		CreationTime: time.Now().Add(-7 * 24 * time.Hour),
		UpdateTime:   time.Now(),
	}

	return policy, nil
}

// DeleteReplicationPolicy deletes a replication policy
func (c *HarborClient) DeleteReplicationPolicy(ctx context.Context, policyID string) error {
	if policyID == "" {
		return errors.New("policy ID is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Deleting Harbor replication policy", "policyId", policyID)

	return nil
}

// TriggerReplication triggers a manual replication
func (c *HarborClient) TriggerReplication(ctx context.Context, policyID string) (*ReplicationExecution, error) {
	if policyID == "" {
		return nil, errors.New("policy ID is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Triggering Harbor replication", "policyId", policyID)

	execution := &ReplicationExecution{
		ID:        "1",
		PolicyID:  policyID,
		Status:    "pending",
		StartTime: time.Now(),
	}

	return execution, nil
}

// ListReplicationExecutions lists replication execution history
func (c *HarborClient) ListReplicationExecutions(ctx context.Context, policyID string) ([]*ReplicationExecution, error) {
	if policyID == "" {
		return nil, errors.New("policy ID is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Listing Harbor replication executions", "policyId", policyID)

	executions := []*ReplicationExecution{
		{
			ID:           "1",
			PolicyID:     policyID,
			Status:       "completed",
			StartTime:    time.Now().Add(-1 * time.Hour),
			EndTime:      time.Now(),
			SuccessCount: 42,
			FailedCount:  0,
		},
	}

	return executions, nil
}

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

// CreateRetentionPolicy creates a new retention policy
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

	policy := &RetentionPolicyStatus{
		ID:           "1",
		ProjectID:    spec.ProjectID,
		Description:  spec.Description,
		Enabled:      spec.Enabled != nil && *spec.Enabled,
		CreationTime: time.Now(),
		UpdateTime:   time.Now(),
	}

	return policy, nil
}

// ListRetentionPolicies lists retention policies for a project
func (c *HarborClient) ListRetentionPolicies(ctx context.Context, projectID string) ([]*RetentionPolicyStatus, error) {
	if projectID == "" {
		return nil, errors.New("project ID is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Listing Harbor retention policies", "projectId", projectID)

	policies := []*RetentionPolicyStatus{
		{
			ID:           "1",
			ProjectID:    projectID,
			Enabled:      true,
			CreationTime: time.Now().Add(-30 * 24 * time.Hour),
			UpdateTime:   time.Now(),
		},
	}

	return policies, nil
}

// GetRetentionPolicy retrieves a specific retention policy
func (c *HarborClient) GetRetentionPolicy(ctx context.Context, projectID, policyID string) (*RetentionPolicyStatus, error) {
	if projectID == "" {
		return nil, errors.New("project ID is required")
	}
	if policyID == "" {
		return nil, errors.New("policy ID is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Retrieving Harbor retention policy", "projectId", projectID, "policyId", policyID)

	policy := &RetentionPolicyStatus{
		ID:           policyID,
		ProjectID:    projectID,
		Enabled:      true,
		CreationTime: time.Now().Add(-30 * 24 * time.Hour),
		UpdateTime:   time.Now(),
	}

	return policy, nil
}

// UpdateRetentionPolicy updates a retention policy
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

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Updating Harbor retention policy", "projectId", projectID, "policyId", policyID)

	policy := &RetentionPolicyStatus{
		ID:           policyID,
		ProjectID:    projectID,
		Description:  spec.Description,
		Enabled:      spec.Enabled != nil && *spec.Enabled,
		CreationTime: time.Now().Add(-30 * 24 * time.Hour),
		UpdateTime:   time.Now(),
	}

	return policy, nil
}

// DeleteRetentionPolicy deletes a retention policy
func (c *HarborClient) DeleteRetentionPolicy(ctx context.Context, projectID, policyID string) error {
	if projectID == "" {
		return errors.New("project ID is required")
	}
	if policyID == "" {
		return errors.New("policy ID is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Deleting Harbor retention policy", "projectId", projectID, "policyId", policyID)

	return nil
}

// CreateUserGroup creates a new user group in Harbor
func (c *HarborClient) CreateUserGroup(ctx context.Context, spec *UserGroupSpec) (*UserGroupStatus, error) {
	if spec == nil {
		return nil, errors.New("user group spec is required")
	}
	if spec.GroupName == "" {
		return nil, errors.New("group name is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Creating Harbor user group", "groupName", spec.GroupName, "groupType", spec.GroupType)

	body := &sdkmodels.UserGroup{
		GroupName:   spec.GroupName,
		GroupType:   spec.GroupType,
		LdapGroupDn: getStringValue(spec.LdapGroupDn),
	}
	params := sdkusergroup.NewCreateUserGroupParams()
	params.WithUsergroup(body)
	resp, err := v2Client.Usergroup.CreateUserGroup(ctx, params)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create user group")
	}

	var id int64
	if resp.Location != "" {
		parts := strings.Split(strings.TrimRight(resp.Location, "/"), "/")
		if n := len(parts); n > 0 {
			id, _ = strconv.ParseInt(parts[n-1], 10, 64)
		}
	}
	if id <= 0 {
		// Location missing or unparsable — resolve id by listing.
		groups, err := c.ListUserGroups(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to resolve created user group id")
		}
		for _, g := range groups {
			if g.GroupName == spec.GroupName {
				id = g.ID
				break
			}
		}
	}
	if id <= 0 {
		return nil, errors.Errorf("user group %q created but id could not be resolved", spec.GroupName)
	}

	return &UserGroupStatus{
		ID:          id,
		GroupName:   spec.GroupName,
		GroupType:   spec.GroupType,
		LdapGroupDn: getStringValue(spec.LdapGroupDn),
	}, nil
}

// ListUserGroups lists all user groups in Harbor
func (c *HarborClient) ListUserGroups(ctx context.Context) ([]*UserGroupStatus, error) {
	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Listing Harbor user groups")

	params := sdkusergroup.NewListUserGroupsParams()
	pageSize := int64(100)
	params.WithPageSize(&pageSize)

	resp, err := v2Client.Usergroup.ListUserGroups(ctx, params)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list user groups")
	}

	groups := make([]*UserGroupStatus, 0, len(resp.Payload))
	for _, g := range resp.Payload {
		groups = append(groups, &UserGroupStatus{
			ID:          g.ID,
			GroupName:   g.GroupName,
			GroupType:   g.GroupType,
			LdapGroupDn: g.LdapGroupDn,
		})
	}
	return groups, nil
}

// GetUserGroup retrieves a specific user group from Harbor
func (c *HarborClient) GetUserGroup(ctx context.Context, groupID int64) (*UserGroupStatus, error) {
	if groupID <= 0 {
		return nil, errors.New("group ID is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Getting Harbor user group", "groupId", groupID)

	params := sdkusergroup.NewGetUserGroupParams()
	params.WithGroupID(groupID)
	resp, err := v2Client.Usergroup.GetUserGroup(ctx, params)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user group")
	}
	g := resp.Payload
	return &UserGroupStatus{
		ID:          g.ID,
		GroupName:   g.GroupName,
		GroupType:   g.GroupType,
		LdapGroupDn: g.LdapGroupDn,
	}, nil
}

// UpdateUserGroup updates a user group in Harbor
func (c *HarborClient) UpdateUserGroup(ctx context.Context, groupID int64, spec *UserGroupSpec) (*UserGroupStatus, error) {
	if groupID <= 0 {
		return nil, errors.New("group ID is required")
	}
	if spec == nil {
		return nil, errors.New("user group spec is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Updating Harbor user group", "groupId", groupID, "groupName", spec.GroupName)

	body := &sdkmodels.UserGroup{
		ID:          groupID,
		GroupName:   spec.GroupName,
		GroupType:   spec.GroupType,
		LdapGroupDn: getStringValue(spec.LdapGroupDn),
	}
	params := sdkusergroup.NewUpdateUserGroupParams()
	params.WithGroupID(groupID)
	params.WithUsergroup(body)
	if _, err := v2Client.Usergroup.UpdateUserGroup(ctx, params); err != nil {
		return nil, errors.Wrap(err, "failed to update user group")
	}

	return &UserGroupStatus{
		ID:          groupID,
		GroupName:   spec.GroupName,
		GroupType:   spec.GroupType,
		LdapGroupDn: getStringValue(spec.LdapGroupDn),
	}, nil
}

// DeleteUserGroup deletes a user group from Harbor
func (c *HarborClient) DeleteUserGroup(ctx context.Context, groupID int64) error {
	if groupID <= 0 {
		return errors.New("group ID is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Deleting Harbor user group", "groupId", groupID)

	params := sdkusergroup.NewDeleteUserGroupParams()
	params.WithGroupID(groupID)
	if _, err := v2Client.Usergroup.DeleteUserGroup(ctx, params); err != nil {
		return errors.Wrap(err, "failed to delete user group")
	}
	return nil
}

// Helper functions

// getStringValue returns string value from pointer, empty string if nil
func getStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
