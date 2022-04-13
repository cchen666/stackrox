package dtr

import (
	"github.com/pkg/errors"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/registries/docker"
	"github.com/stackrox/stackrox/pkg/registries/types"
)

// Creator provides the type and registries.Creator to add to the registries Registry.
func Creator() (string, func(integration *storage.ImageIntegration) (types.Registry, error)) {
	return "dtr", func(integration *storage.ImageIntegration) (types.Registry, error) {
		reg, err := newRegistry(integration)
		return reg, err
	}
}

func newRegistry(integration *storage.ImageIntegration) (*docker.Registry, error) {
	dtrConfig, ok := integration.IntegrationConfig.(*storage.ImageIntegration_Dtr)
	if !ok {
		return nil, errors.New("DTR configuration required")
	}
	cfg := docker.Config{
		Username: dtrConfig.Dtr.GetUsername(),
		Password: dtrConfig.Dtr.GetPassword(),
		Endpoint: dtrConfig.Dtr.GetEndpoint(),
		Insecure: dtrConfig.Dtr.GetInsecure(),
	}
	return docker.NewDockerRegistryWithConfig(cfg, integration)
}
