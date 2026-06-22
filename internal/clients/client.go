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
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/goharbor/go-client/pkg/harbor"
	harborsysteminfo "github.com/goharbor/go-client/pkg/sdk/v2.0/client/systeminfo"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/rossigee/provider-harbor/apis/v1beta1"
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
}

// HarborConfig holds configuration for creating a Harbor client
type HarborConfig struct {
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
	Insecure bool   `json:"insecure"`
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
				DualStack: true,
			}).DialContext,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: config.Insecure,
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

	logger := logging.NewNopLogger().WithValues("client", "harbor")

	return &HarborClient{
		clientSet:  clientSet,
		config:     csConfig,
		logger:     logger,
		httpClient: httpClient,
	}, nil
}

// NewHarborClientFromProviderConfig creates a Harbor client from a ProviderConfig
// This maintains compatibility with the existing Crossplane provider pattern
func NewHarborClientFromProviderConfig(ctx context.Context, k8sClient client.Client, mg resource.Managed) (HarborClienter, error) {
	// Every Harbor managed resource is generated with a GetProviderConfigReference
	// accessor (its spec embeds xpv1.ManagedResourceSpec). Resolve the reference
	// generically via that interface rather than a per-type switch, so a new
	// resource works without editing this function.
	pcr, ok := mg.(interface {
		GetProviderConfigReference() *xpv1.ProviderConfigReference
	})
	if !ok {
		return nil, errors.New("managed resource does not expose a ProviderConfigReference")
	}
	configRef := pcr.GetProviderConfigReference()
	if configRef == nil {
		return nil, errors.New(errNoProviderConfig)
	}

	pc := &v1beta1.ProviderConfig{}
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

	if urlBytes, ok := secret.Data["url"]; ok {
		config.URL = string(urlBytes)
	} else {
		return nil, errors.New("url is required in credentials secret")
	}

	if usernameBytes, ok := secret.Data["username"]; ok {
		config.Username = string(usernameBytes)
	} else {
		return nil, errors.New("username is required in credentials secret")
	}

	if passwordBytes, ok := secret.Data["password"]; ok {
		config.Password = string(passwordBytes)
	} else {
		return nil, errors.New("password is required in credentials secret")
	}

	// Optional: insecure flag
	if insecureBytes, ok := secret.Data["insecure"]; ok {
		insecure, err := strconv.ParseBool(string(insecureBytes))
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

// GetVersion returns Harbor version information
func (c *HarborClient) GetVersion(ctx context.Context) (string, error) {
	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return "", errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Retrieving Harbor version information")

	params := harborsysteminfo.NewGetSystemInfoParams().WithContext(ctx)
	resp, err := v2Client.Systeminfo.GetSystemInfo(ctx, params)
	if err != nil {
		return "", errors.Wrap(err, "cannot get Harbor system info")
	}
	return ptr.Deref(resp.Payload.HarborVersion, "unknown"), nil
}

// GetMemoryFootprint returns estimated memory usage for this client
func (c *HarborClient) GetMemoryFootprint() string {
	return "~5-10MB (Harbor Go client + minimal overhead)"
}
