package dackbox

import (
	"testing"

	"github.com/stackrox/stackrox/central/imagecomponent/store"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/concurrency"
	"github.com/stackrox/stackrox/pkg/dackbox"
	"github.com/stackrox/stackrox/pkg/rocksdb"
	"github.com/stackrox/stackrox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
)

func TestImageStore(t *testing.T) {
	suite.Run(t, new(ImageStoreTestSuite))
}

type ImageStoreTestSuite struct {
	suite.Suite

	db    *rocksdb.RocksDB
	dacky *dackbox.DackBox

	store store.Store
}

func (suite *ImageStoreTestSuite) SetupSuite() {
	var err error
	suite.db, err = rocksdb.NewTemp("reference")
	if err != nil {
		suite.FailNowf("failed to create DB: %+v", err.Error())
	}
	suite.dacky, err = dackbox.NewRocksDBDackBox(suite.db, nil, []byte("graph"), []byte("dirty"), []byte("valid"))
	if err != nil {
		suite.FailNowf("failed to create counter: %+v", err.Error())
	}
	suite.store, err = New(suite.dacky, concurrency.NewKeyFence())
	if err != nil {
		suite.FailNowf("failed to create counter: %+v", err.Error())
	}
}

func (suite *ImageStoreTestSuite) TearDownSuite() {
	rocksdbtest.TearDownRocksDB(suite.db)
}

func (suite *ImageStoreTestSuite) TestImages() {
	components := []*storage.ImageComponent{
		{
			Id:   "derp",
			Name: "postgress",
		},
		{
			Id:   "jerp",
			Name: "esqueueell",
		},
	}

	// Test Add
	for _, d := range components {
		suite.NoError(suite.store.Upsert(d))
	}

	for _, d := range components {
		got, exists, err := suite.store.Get(d.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, d)
	}

	// Test Update
	for _, d := range components {
		d.Name += "1"
	}

	suite.NoError(suite.store.Upsert(components...))

	for _, d := range components {
		got, exists, err := suite.store.Get(d.GetId())
		suite.NoError(err)
		suite.True(exists)
		suite.Equal(got, d)
	}

	// Test Count
	count, err := suite.store.Count()
	suite.NoError(err)
	suite.Equal(len(components), count)
}
