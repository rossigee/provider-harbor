/*
Copyright 2026 Crossplane Harbor Provider.
*/

package clients

import (
	"context"

	sdkconfigure "github.com/goharbor/go-client/pkg/sdk/v2.0/client/configure"
	sdkmodels "github.com/goharbor/go-client/pkg/sdk/v2.0/models"
	"github.com/pkg/errors"
)

// GetConfigurations returns Harbor's system-wide configuration.
func (c *HarborClient) GetConfigurations(ctx context.Context) (*sdkmodels.ConfigurationsResponse, error) {
	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Getting Harbor system configuration")

	resp, err := v2Client.Configure.GetConfigurations(ctx, sdkconfigure.NewGetConfigurationsParams())
	if err != nil {
		return nil, errors.Wrap(err, "failed to get configurations")
	}

	if resp.Payload == nil {
		return nil, errors.New("configurations response is empty")
	}

	return resp.Payload, nil
}

// UpdateConfigurations writes the given settings to Harbor's system
// configuration. Fields left nil are not sent and keep their current value.
func (c *HarborClient) UpdateConfigurations(ctx context.Context, cfg *sdkmodels.Configurations) error {
	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Updating Harbor system configuration")

	params := sdkconfigure.NewUpdateConfigurationsParams()
	params.Configurations = cfg

	if _, err := v2Client.Configure.UpdateConfigurations(ctx, params); err != nil {
		return errors.Wrap(err, "failed to update configurations")
	}

	return nil
}
