package store

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/bolthelper"
	protoCrud "github.com/stackrox/stackrox/pkg/bolthelper/crud/proto"
	bolt "go.etcd.io/bbolt"
)

func alloc() proto.Message {
	return new(storage.ExternalBackup)
}

func key(msg proto.Message) []byte {
	return []byte(msg.(*storage.ExternalBackup).GetId())
}

var (
	backupBucketKey = []byte("externalBackups")
)

type storeImpl struct {
	crud protoCrud.MessageCrud
}

// New returns a new Node store
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, backupBucketKey)

	crud := protoCrud.NewMessageCrudForBucket(bolthelper.TopLevelRef(db, backupBucketKey), key, alloc)
	return &storeImpl{crud: crud}
}

func (s *storeImpl) ListBackups() ([]*storage.ExternalBackup, error) {
	entries, err := s.crud.ReadAll()
	if err != nil {
		return nil, err
	}
	backups := make([]*storage.ExternalBackup, len(entries))
	for i, entry := range entries {
		backups[i] = entry.(*storage.ExternalBackup)
	}
	return backups, nil
}

func (s *storeImpl) GetBackup(id string) (*storage.ExternalBackup, error) {
	value, err := s.crud.Read(id)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, nil
	}
	return value.(*storage.ExternalBackup), nil
}

func (s *storeImpl) UpsertBackup(backup *storage.ExternalBackup) error {
	_, _, err := s.crud.Upsert(backup)
	return err
}

func (s *storeImpl) RemoveBackup(id string) error {
	_, _, err := s.crud.Delete(id)
	return err
}
