package datastore

import (
	"github.com/stackrox/stackrox/central/notifier/datastore/internal/store"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	bolt "go.etcd.io/bbolt"
)

// Scrub scrubs sensitive notifier information from a DB.
func Scrub(db *bolt.DB) error {
	store := store.New(db)

	notifiers, err := store.GetNotifiers(&v1.GetNotifiersRequest{})
	if err != nil {
		return err
	}
	for _, n := range notifiers {
		n.Config = nil
		if err := store.UpdateNotifier(n); err != nil {
			return err
		}
	}
	return nil
}
