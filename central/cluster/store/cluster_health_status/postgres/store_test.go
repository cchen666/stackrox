// Code generated by pg-bindings generator. DO NOT EDIT.

//go:build sql_integration

package postgres

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/suite"
)

type ClusterHealthStatusStoreSuite struct {
	suite.Suite
	envIsolator *envisolator.EnvIsolator
}

func TestClusterHealthStatusStore(t *testing.T) {
	suite.Run(t, new(ClusterHealthStatusStoreSuite))
}

func (s *ClusterHealthStatusStoreSuite) SetupTest() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
	s.envIsolator.Setenv(features.PostgresDatastore.EnvVar(), "true")

	if !features.PostgresDatastore.Enabled() {
		s.T().Skip("Skip postgres store tests")
		s.T().SkipNow()
	}
}

func (s *ClusterHealthStatusStoreSuite) TearDownTest() {
	s.envIsolator.RestoreAll()
}

func (s *ClusterHealthStatusStoreSuite) TestStore() {
	ctx := context.Background()

	source := pgtest.GetConnectionString(s.T())
	config, err := pgxpool.ParseConfig(source)
	s.Require().NoError(err)
	pool, err := pgxpool.ConnectConfig(ctx, config)
	s.NoError(err)
	defer pool.Close()

	Destroy(ctx, pool)
	store := New(ctx, pool)

	clusterHealthStatus := &storage.ClusterHealthStatus{}
	s.NoError(testutils.FullInit(clusterHealthStatus, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))

	foundClusterHealthStatus, exists, err := store.Get(ctx, clusterHealthStatus.GetId())
	s.NoError(err)
	s.False(exists)
	s.Nil(foundClusterHealthStatus)

	s.NoError(store.Upsert(ctx, clusterHealthStatus))
	foundClusterHealthStatus, exists, err = store.Get(ctx, clusterHealthStatus.GetId())
	s.NoError(err)
	s.True(exists)
	s.Equal(clusterHealthStatus, foundClusterHealthStatus)

	clusterHealthStatusCount, err := store.Count(ctx)
	s.NoError(err)
	s.Equal(clusterHealthStatusCount, 1)

	clusterHealthStatusExists, err := store.Exists(ctx, clusterHealthStatus.GetId())
	s.NoError(err)
	s.True(clusterHealthStatusExists)
	s.NoError(store.Upsert(ctx, clusterHealthStatus))

	foundClusterHealthStatus, exists, err = store.Get(ctx, clusterHealthStatus.GetId())
	s.NoError(err)
	s.True(exists)
	s.Equal(clusterHealthStatus, foundClusterHealthStatus)

	s.NoError(store.Delete(ctx, clusterHealthStatus.GetId()))
	foundClusterHealthStatus, exists, err = store.Get(ctx, clusterHealthStatus.GetId())
	s.NoError(err)
	s.False(exists)
	s.Nil(foundClusterHealthStatus)

	var clusterHealthStatuss []*storage.ClusterHealthStatus
	for i := 0; i < 200; i++ {
		clusterHealthStatus := &storage.ClusterHealthStatus{}
		s.NoError(testutils.FullInit(clusterHealthStatus, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
		clusterHealthStatuss = append(clusterHealthStatuss, clusterHealthStatus)
	}

	s.NoError(store.UpsertMany(ctx, clusterHealthStatuss))

	clusterHealthStatusCount, err = store.Count(ctx)
	s.NoError(err)
	s.Equal(clusterHealthStatusCount, 200)
}
