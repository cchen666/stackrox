package store

import (
	"github.com/stackrox/stackrox/generated/storage"
)

// Store provides storage functionality.
//go:generate mockgen-wrapper
type Store interface {
	ListDeployment(id string) (*storage.ListDeployment, bool, error)
	ListDeploymentsWithIDs(ids ...string) ([]*storage.ListDeployment, []int, error)

	GetDeployment(id string) (*storage.Deployment, bool, error)
	GetDeploymentsWithIDs(ids ...string) ([]*storage.Deployment, []int, error)

	CountDeployments() (int, error)
	UpsertDeployment(deployment *storage.Deployment) error
	RemoveDeployment(id string) error

	AckKeysIndexed(keys ...string) error
	GetKeysToIndex() ([]string, error)

	GetDeploymentIDs() ([]string, error)
}
