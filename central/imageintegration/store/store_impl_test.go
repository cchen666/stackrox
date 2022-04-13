package store

import (
	"strings"
	"testing"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/bolthelper"
	"github.com/stackrox/stackrox/pkg/testutils"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

func TestImageIntegrationStore(t *testing.T) {
	suite.Run(t, new(ImageIntegrationStoreTestSuite))
}

type ImageIntegrationStoreTestSuite struct {
	suite.Suite

	db *bolt.DB

	store Store
}

func (suite *ImageIntegrationStoreTestSuite) SetupTest() {
	db, err := bolthelper.NewTemp(testutils.DBFileName(suite))
	if err != nil {
		suite.FailNow("failure: "+suite.T().Name(), err.Error())
	}

	suite.db = db
	suite.store = New(db)
}

func (suite *ImageIntegrationStoreTestSuite) TearDownTest() {
	testutils.TearDownDB(suite.db)
}

func (suite *ImageIntegrationStoreTestSuite) TestIntegrations() {
	integration := []*storage.ImageIntegration{
		{
			Name: "registry1",
			IntegrationConfig: &storage.ImageIntegration_Docker{
				Docker: &storage.DockerConfig{
					Endpoint: "https://endpoint1",
				},
			},
		},
		{
			Name: "registry2",
			IntegrationConfig: &storage.ImageIntegration_Docker{
				Docker: &storage.DockerConfig{
					Endpoint: "https://endpoint2",
				},
			},
		},
	}

	// Test Add
	for _, r := range integration {
		id, err := suite.store.AddImageIntegration(r)
		suite.NoError(err)
		suite.NotEmpty(id)
	}

	for _, r := range integration {
		got, exists, err := suite.store.GetImageIntegration(r.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, r)
	}

	// Test Update
	for _, r := range integration {
		r.Name += "-ext"
	}
	for _, r := range integration {
		suite.NoError(suite.store.UpdateImageIntegration(r))
	}
	for _, r := range integration {
		r.Name = strings.TrimSuffix(r.Name, "-ext")
	}
	for _, r := range integration {
		suite.NoError(suite.store.UpdateImageIntegration(r))
	}

	for _, r := range integration {
		got, exists, err := suite.store.GetImageIntegration(r.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, r)
	}

	// Test Remove
	for _, r := range integration {
		suite.NoError(suite.store.RemoveImageIntegration(r.GetId()))
	}

	for _, r := range integration {
		_, exists, err := suite.store.GetImageIntegration(r.GetId())
		suite.NoError(err)
		suite.False(exists)
	}
}
