package singleton

import (
	"github.com/stackrox/stackrox/central/deploymentenvs"
	"github.com/stackrox/stackrox/central/license/datastore"
	"github.com/stackrox/stackrox/central/license/manager"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	instance     manager.LicenseManager
	instanceInit sync.Once
)

// ManagerSingleton returns the license manager singleton instance
func ManagerSingleton() manager.LicenseManager {
	instanceInit.Do(func() {
		instance = manager.New(datastore.Singleton(), validatorSingleton(), deploymentenvs.ManagerSingleton())
	})
	return instance
}
