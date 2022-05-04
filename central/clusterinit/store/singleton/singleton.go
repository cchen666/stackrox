package singleton

import (
	"context"

	"github.com/stackrox/rox/central/clusterinit/store"
	"github.com/stackrox/rox/central/clusterinit/store/postgres"
	"github.com/stackrox/rox/central/clusterinit/store/rocksdb"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	instance     store.Store
	instanceInit sync.Once
)

// Singleton returns the singleton data store for cluster init bundles.
func Singleton() store.Store {
	instanceInit.Do(func() {
		var underlying store.UnderlyingStore
		if features.PostgresDatastore.Enabled() {
			var err error
			underlying, err = rocksdb.New(globaldb.GetRocksDB())
			utils.CrashOnError(err)
		} else {
			underlying = postgres.New(context.TODO(), globaldb.GetPostgres())
		}
		instance = store.NewStore(underlying)
	})
	return instance
}
