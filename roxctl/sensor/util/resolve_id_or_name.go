package util

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	pkgCommon "github.com/stackrox/stackrox/pkg/roxctl/common"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/uuid"
	"github.com/stackrox/stackrox/roxctl/common"
)

// ResolveClusterID returns the cluster ID corresponding to the given id or name,
// or an error if no matching cluster was found.
func ResolveClusterID(idOrName string, timeout time.Duration) (string, error) {
	if _, err := uuid.FromString(idOrName); err == nil {
		return idOrName, nil
	}

	conn, err := common.GetGRPCConnection()
	if err != nil {
		return "", err
	}

	service := v1.NewClustersServiceClient(conn)

	ctx, cancel := context.WithTimeout(pkgCommon.Context(), timeout)
	defer cancel()

	clusters, err := service.GetClusters(ctx, &v1.GetClustersRequest{
		Query: fmt.Sprintf("%s:%s", search.Cluster, idOrName),
	})
	if err != nil {
		return "", err
	}

	for _, cluster := range clusters.GetClusters() {
		if cluster.GetName() == idOrName {
			return cluster.GetId(), nil
		}
	}
	return "", errors.Errorf("no cluster with name %q found", idOrName)
}
