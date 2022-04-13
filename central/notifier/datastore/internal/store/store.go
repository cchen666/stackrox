package store

import (
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/bolthelper"
	bolt "go.etcd.io/bbolt"
)

var notifierBucket = []byte("notifiers")

// Store provides storage functionality for alerts.
//go:generate mockgen-wrapper
type Store interface {
	GetNotifier(id string) (*storage.Notifier, bool, error)
	GetNotifiers(request *v1.GetNotifiersRequest) ([]*storage.Notifier, error)
	AddNotifier(notifier *storage.Notifier) (string, error)
	UpdateNotifier(notifier *storage.Notifier) error
	RemoveNotifier(id string) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, notifierBucket)
	return &storeImpl{
		DB: db,
	}
}
