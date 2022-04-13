package store

import (
	"fmt"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/central/metrics"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/bolthelper"
	"github.com/stackrox/stackrox/pkg/dberrors"
	ops "github.com/stackrox/stackrox/pkg/metrics"
	"github.com/stackrox/stackrox/pkg/secondarykey"
	bolt "go.etcd.io/bbolt"
)

type storeImpl struct {
	db *bolt.DB
}

// GetAuthProviders retrieves authProviders from bolt
func (s *storeImpl) GetAllAuthProviders() ([]*storage.AuthProvider, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetAll, "AuthProvider")

	var authProviders []*storage.AuthProvider
	err := s.db.View(func(tx *bolt.Tx) error {
		provB := tx.Bucket(authProviderBucket)

		return provB.ForEach(func(k, v []byte) error {
			var authProvider storage.AuthProvider
			if err := proto.Unmarshal(v, &authProvider); err != nil {
				return err
			}

			authProviders = append(authProviders, &authProvider)
			return nil
		})
	})
	return authProviders, err
}

// AddAuthProvider adds an auth provider into bolt
func (s *storeImpl) AddAuthProvider(authProvider *storage.AuthProvider) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Add, "AuthProvider")

	if authProvider.GetId() == "" || authProvider.GetName() == "" {
		return errors.New("auth provider is missing required fields")
	}

	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(authProviderBucket)
		if bolthelper.Exists(bucket, authProvider.GetId()) {
			return fmt.Errorf("AuthProvider %v (%v) cannot be added because it already exists", authProvider.GetId(), authProvider.GetName())
		}
		if err := secondarykey.CheckUniqueKeyExistsAndInsert(tx, authProviderBucket, authProvider.GetId(), authProvider.GetName()); err != nil {
			return errors.Wrap(err, "Could not add AuthProvider due to name validation")
		}
		bytes, err := proto.Marshal(authProvider)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(authProvider.GetId()), bytes)
	})
}

// UpdateAuthProvider upserts an auth provider into bolt
func (s *storeImpl) UpdateAuthProvider(authProvider *storage.AuthProvider) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Update, "AuthProvider")

	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(authProviderBucket)
		// If the update is changing the name, check if the name has already been taken
		if val, _ := secondarykey.GetCurrentUniqueKey(tx, authProviderBucket, authProvider.GetId()); val != authProvider.GetName() {
			if err := secondarykey.UpdateUniqueKey(tx, authProviderBucket, authProvider.GetId(), authProvider.GetName()); err != nil {
				return errors.Wrap(err, "Could not update auth provider due to name validation")
			}
		}
		bytes, err := proto.Marshal(authProvider)
		if err != nil {
			return err
		}
		return b.Put([]byte(authProvider.GetId()), bytes)
	})
}

// RemoveAuthProvider removes an auth provider from bolt
func (s *storeImpl) RemoveAuthProvider(id string) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "AuthProvider")

	return s.db.Update(func(tx *bolt.Tx) error {
		ab := tx.Bucket(authProviderBucket)
		key := []byte(id)
		if exists := ab.Get(key) != nil; !exists {
			return dberrors.ErrNotFound{Type: "Auth Provider", ID: id}
		}
		if err := secondarykey.RemoveUniqueKey(tx, authProviderBucket, id); err != nil {
			return err
		}
		return ab.Delete(key)
	})
}
