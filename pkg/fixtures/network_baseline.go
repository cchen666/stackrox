package fixtures

import (
	"github.com/stackrox/stackrox/generated/storage"
)

// GetNetworkBaseline returns a mock network baseline
func GetNetworkBaseline() *storage.NetworkBaseline {
	return &storage.NetworkBaseline{
		DeploymentId:         GetDeployment().GetId(),
		ClusterId:            "prod cluster",
		Namespace:            "stackrox",
		Peers:                nil,
		ForbiddenPeers:       nil,
		ObservationPeriodEnd: nil,
		Locked:               false,
		DeploymentName:       GetDeployment().GetName(),
	}
}
