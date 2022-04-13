package undostore

import (
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/bolthelper"
	"github.com/stackrox/stackrox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

func TestUndoStore(t *testing.T) {
	suite.Run(t, new(undoStoreTestSuite))
}

type undoStoreTestSuite struct {
	suite.Suite

	db    *bolt.DB
	store UndoStore
}

func (suite *undoStoreTestSuite) SetupSuite() {
	db, err := bolthelper.NewTemp(suite.T().Name() + ".db")
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}

	suite.db = db
	suite.store = New(db)
}

func (suite *undoStoreTestSuite) TearDownSuite() {
	suite.NoError(suite.db.Close())
}

func (suite *undoStoreTestSuite) TestGetOnEmptyStore() {
	clusterID := uuid.NewV4().String()

	_, exists, err := suite.store.GetUndoRecord(clusterID)
	suite.Require().NoError(err)
	suite.False(exists)
}

func (suite *undoStoreTestSuite) TestUpsertOnEmpty() {
	record := &storage.NetworkPolicyApplicationUndoRecord{
		User:           "foo",
		ApplyTimestamp: types.TimestampNow(),
		UndoModification: &storage.NetworkPolicyModification{
			ApplyYaml: "some yaml",
		},
	}

	clusterID := uuid.NewV4().String()

	err := suite.store.UpsertUndoRecord(clusterID, record)
	suite.Require().NoError(err)

	readRecord, exists, err := suite.store.GetUndoRecord(clusterID)
	suite.Require().NoError(err)
	suite.Require().True(exists)

	suite.Equal(record, readRecord)
}

func (suite *undoStoreTestSuite) TestUpsertNewer() {
	olderRecord := &storage.NetworkPolicyApplicationUndoRecord{
		User:           "foo",
		ApplyTimestamp: types.TimestampNow(),
		UndoModification: &storage.NetworkPolicyModification{
			ApplyYaml: "some yaml",
		},
	}

	newerRecord := &storage.NetworkPolicyApplicationUndoRecord{
		User:           "bar",
		ApplyTimestamp: types.TimestampNow(),
		UndoModification: &storage.NetworkPolicyModification{
			ApplyYaml: "another yaml",
		},
	}

	clusterID := uuid.NewV4().String()

	err := suite.store.UpsertUndoRecord(clusterID, olderRecord)
	suite.Require().NoError(err)

	err = suite.store.UpsertUndoRecord(clusterID, newerRecord)
	suite.Require().NoError(err)

	readRecord, exists, err := suite.store.GetUndoRecord(clusterID)
	suite.Require().NoError(err)
	suite.Require().True(exists)

	suite.Equal(newerRecord, readRecord)
}

func (suite *undoStoreTestSuite) TestUpsertOlder() {
	olderRecord := &storage.NetworkPolicyApplicationUndoRecord{
		User:           "foo",
		ApplyTimestamp: types.TimestampNow(),
		UndoModification: &storage.NetworkPolicyModification{
			ApplyYaml: "some yaml",
		},
	}

	newerRecord := &storage.NetworkPolicyApplicationUndoRecord{
		User:           "bar",
		ApplyTimestamp: types.TimestampNow(),
		UndoModification: &storage.NetworkPolicyModification{
			ApplyYaml: "another yaml",
		},
	}

	clusterID := uuid.NewV4().String()

	err := suite.store.UpsertUndoRecord(clusterID, newerRecord)
	suite.Require().NoError(err)

	err = suite.store.UpsertUndoRecord(clusterID, olderRecord)
	suite.Require().Error(err)
}
