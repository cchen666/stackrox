package datastore

import (
	"github.com/stackrox/stackrox/central/license/internal/store"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	ds   DataStore
	once sync.Once
)

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(func() {
		ds = New(store.Singleton())
	})
	return ds
}
