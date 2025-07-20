package clients

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-openapi/runtime"
	harborpkg "github.com/goharbor/go-client/pkg/harbor"
	"github.com/goharbor/go-client/pkg/sdk/v2.0/client"
	"github.com/goharbor/go-client/pkg/sdk/v2.0/client/user"
	"github.com/pkg/errors"
)

// HarborCLI wraps the Harbor v2 API client
type HarborCLI struct {
	V2Client *client.HarborAPI
	AuthInfo runtime.ClientAuthInfoWriter
}

// HarborConfig represents Harbor connection configuration
type HarborConfig struct {
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
	Insecure bool   `json:"insecure,omitempty"`
}

// NewHarborCLI creates a new Harbor v2 API client from credentials
func NewHarborCLI(creds []byte) (*HarborCLI, error) {
	var config HarborConfig
	if err := json.Unmarshal(creds, &config); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal credentials")
	}

	if config.URL == "" {
		return nil, errors.New("url is required in credentials")
	}

	// Create Harbor client configuration
	cfg := &harborpkg.ClientSetConfig{
		URL:      config.URL,
		Username: config.Username,
		Password: config.Password,
		Insecure: config.Insecure,
	}

	// Create Harbor client set
	cs, err := harborpkg.NewClientSet(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create Harbor client set")
	}

	// Get v2 client
	v2Client := cs.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get v2 client")
	}

	// Note: Auth is handled by the client configuration above
	// No need for separate auth info as it's embedded in the client

	return &HarborCLI{
		V2Client: v2Client,
		AuthInfo: nil, // Auth is handled by the client
	}, nil
}

// GetSystemInfo returns Harbor system information
func (h *HarborCLI) GetSystemInfo(ctx context.Context) error {
	// This is a simple health check to verify the client is working
	params := user.NewGetCurrentUserInfoParams()
	_, err := h.V2Client.User.GetCurrentUserInfo(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to get current user info: %w", err)
	}
	return nil
}