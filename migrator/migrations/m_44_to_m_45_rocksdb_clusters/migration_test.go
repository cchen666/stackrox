package m44tom45

import (
	"reflect"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/migrator/bolthelpers"
	"github.com/stackrox/stackrox/migrator/rockshelper"
	dbTypes "github.com/stackrox/stackrox/migrator/types"
	"github.com/stackrox/stackrox/pkg/rocksdb"
	"github.com/stackrox/stackrox/pkg/testutils"
	"github.com/stackrox/stackrox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
	"github.com/tecbot/gorocksdb"
	bolt "go.etcd.io/bbolt"
)

func TestMigration(t *testing.T) {
	suite.Run(t, new(clusterRocksDBMigrationTestSuite))
}

type clusterRocksDBMigrationTestSuite struct {
	suite.Suite

	db        *rocksdb.RocksDB
	databases *dbTypes.Databases
}

func (suite *clusterRocksDBMigrationTestSuite) SetupTest() {
	db, err := bolthelpers.NewTemp(testutils.DBFileName(suite))
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}
	suite.NoError(db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(clusterBucketName); err != nil {
			return err
		}

		if _, err := tx.CreateBucketIfNotExists(clusterStatusBucketName); err != nil {
			return err
		}

		_, err := tx.CreateBucketIfNotExists(clusterLastContactTimeBucketName)
		return err
	}))

	rocksDB, err := rocksdb.NewTemp(suite.T().Name())
	suite.NoError(err)

	suite.db = rocksDB
	suite.databases = &dbTypes.Databases{BoltDB: db, RocksDB: rocksDB.DB}
}

func (suite *clusterRocksDBMigrationTestSuite) TearDownTest() {
	testutils.TearDownDB(suite.databases.BoltDB)
	rocksdbtest.TearDownRocksDB(suite.db)
}

func insertMessage(bucket bolthelpers.BucketRef, id string, pb proto.Message) error {
	if pb == nil {
		return nil
	}
	if reflect.TypeOf(pb).Kind() == reflect.Ptr && reflect.ValueOf(pb).IsNil() {
		return nil
	}
	return bucket.Update(func(b *bolt.Bucket) error {
		bytes, err := proto.Marshal(pb)
		if err != nil {
			return err
		}
		return b.Put([]byte(id), bytes)
	})
}

func (suite *clusterRocksDBMigrationTestSuite) TestClusterRocksDBMigration() {
	ts1 := types.TimestampNow()
	ts2 := ts1.Clone()
	ts2.Seconds = ts2.Seconds - 120
	ts3 := ts1.Clone()
	ts3.Seconds = ts3.Seconds - 300

	cases := []struct {
		boltCluster       *storage.Cluster
		boltStatus        *storage.ClusterStatus
		boltLastContact   *types.Timestamp
		rocksDBCluster    *storage.Cluster
		rocksDBHealth     *storage.ClusterHealthStatus
		healthShouldExist bool
	}{
		{
			boltCluster: &storage.Cluster{
				Id:   "1",
				Name: "cluster1",
			},
			boltStatus: &storage.ClusterStatus{
				SensorVersion: "s1",
			},
			boltLastContact: ts1,
			rocksDBCluster: &storage.Cluster{
				Id:   "1",
				Name: "cluster1",
				Status: &storage.ClusterStatus{
					SensorVersion: "s1",
				},
			},
			rocksDBHealth: &storage.ClusterHealthStatus{
				SensorHealthStatus:    storage.ClusterHealthStatus_HEALTHY,
				CollectorHealthStatus: storage.ClusterHealthStatus_UNAVAILABLE,
				OverallHealthStatus:   storage.ClusterHealthStatus_HEALTHY,
				LastContact:           ts1,
			},
			healthShouldExist: true,
		},
		{
			boltCluster: &storage.Cluster{
				Id:   "2",
				Name: "cluster2",
			},
			boltStatus: &storage.ClusterStatus{
				SensorVersion: "s2",
			},
			boltLastContact: ts2,
			rocksDBCluster: &storage.Cluster{
				Id:   "2",
				Name: "cluster2",
				Status: &storage.ClusterStatus{
					SensorVersion: "s2",
				},
			},
			rocksDBHealth: &storage.ClusterHealthStatus{
				SensorHealthStatus:    storage.ClusterHealthStatus_DEGRADED,
				CollectorHealthStatus: storage.ClusterHealthStatus_UNAVAILABLE,
				OverallHealthStatus:   storage.ClusterHealthStatus_DEGRADED,
				LastContact:           ts2,
			},
			healthShouldExist: true,
		},
		{
			boltCluster: &storage.Cluster{
				Id:   "3",
				Name: "cluster3",
			},
			boltStatus: &storage.ClusterStatus{
				SensorVersion: "s3",
			},
			boltLastContact: ts3,
			rocksDBCluster: &storage.Cluster{
				Id:   "3",
				Name: "cluster3",
				Status: &storage.ClusterStatus{
					SensorVersion: "s3",
				},
			},
			rocksDBHealth: &storage.ClusterHealthStatus{
				SensorHealthStatus:    storage.ClusterHealthStatus_UNHEALTHY,
				CollectorHealthStatus: storage.ClusterHealthStatus_UNAVAILABLE,
				OverallHealthStatus:   storage.ClusterHealthStatus_UNHEALTHY,
				LastContact:           ts3,
			},
			healthShouldExist: true,
		},
		{
			boltCluster: &storage.Cluster{
				Id:   "4",
				Name: "cluster4",
			},
			rocksDBCluster: &storage.Cluster{
				Id:   "4",
				Name: "cluster4",
			},
			healthShouldExist: false,
		},
	}

	clusterBucket := bolthelpers.TopLevelRef(suite.databases.BoltDB, clusterBucketName)
	clusterStatusBucket := bolthelpers.TopLevelRef(suite.databases.BoltDB, clusterStatusBucketName)
	clusterLastContactBucket := bolthelpers.TopLevelRef(suite.databases.BoltDB, clusterLastContactTimeBucketName)

	for _, c := range cases {
		suite.NoError(insertMessage(clusterBucket, c.boltCluster.GetId(), c.boltCluster))
		suite.NoError(insertMessage(clusterStatusBucket, c.boltCluster.GetId(), c.boltStatus))
		suite.NoError(insertMessage(clusterLastContactBucket, c.boltCluster.GetId(), c.boltLastContact))
	}

	suite.NoError(migrateClusterBuckets(suite.databases))

	readOpts := gorocksdb.NewDefaultReadOptions()
	for _, c := range cases {
		cluster, exists, err := rockshelper.ReadFromRocksDB(suite.databases.RocksDB, readOpts, &storage.Cluster{}, clusterBucketName, []byte(c.boltCluster.GetId()))
		suite.NoError(err)
		suite.True(exists)
		suite.EqualValues(c.rocksDBCluster, cluster.(*storage.Cluster))
	}

	for _, c := range cases {
		health, exists, err := rockshelper.ReadFromRocksDB(suite.databases.RocksDB, readOpts, &storage.ClusterHealthStatus{}, clusterHealthStatusBucketName, []byte(c.boltCluster.GetId()))
		suite.NoError(err)
		suite.Equal(c.healthShouldExist, exists)
		if exists {
			suite.EqualValues(c.rocksDBHealth, health.(*storage.ClusterHealthStatus))
		}
	}
}
