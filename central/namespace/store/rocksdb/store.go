// Code generated by rocksdb-bindings generator. DO NOT EDIT.

package rocksdb

import (
	"context"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/db"
	"github.com/stackrox/rox/pkg/rocksdb"
	generic "github.com/stackrox/rox/pkg/rocksdb/crud"
)

var (
	log = logging.LoggerForModule()

	bucket = []byte("namespaces")
)

type Store interface {
	Count(ctx context.Context) (int, error)
	Exists(ctx context.Context, id string) (bool, error)
	GetIDs(ctx context.Context) ([]string, error)
	Get(ctx context.Context, id string) (*storage.NamespaceMetadata, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.NamespaceMetadata, []int, error)
	Upsert(ctx context.Context, obj *storage.NamespaceMetadata) error
	UpsertMany(ctx context.Context, objs []*storage.NamespaceMetadata) error
	Delete(ctx context.Context, id string) error
	DeleteMany(ctx context.Context, ids []string) error
	Walk(ctx context.Context, fn func(obj *storage.NamespaceMetadata) error) error
	AckKeysIndexed(ctx context.Context, keys ...string) error
	GetKeysToIndex(ctx context.Context) ([]string, error)
}

type storeImpl struct {
	crud db.Crud
}

func alloc() proto.Message {
	return &storage.NamespaceMetadata{}
}

func keyFunc(msg proto.Message) []byte {
	return []byte(msg.(*storage.NamespaceMetadata).GetId())
}

// New returns a new Store instance using the provided rocksdb instance.
func New(db *rocksdb.RocksDB) Store {
	globaldb.RegisterBucket(bucket, "NamespaceMetadata")
	return &storeImpl{
		crud: generic.NewCRUD(db, bucket, keyFunc, alloc, false),
	}
}

// Count returns the number of objects in the store
func (b *storeImpl) Count(_ context.Context) (int, error) {
	defer metrics.SetRocksDBOperationDurationTime(time.Now(), ops.Count, "NamespaceMetadata")

	return b.crud.Count()
}

// Exists returns if the id exists in the store
func (b *storeImpl) Exists(_ context.Context, id string) (bool, error) {
	defer metrics.SetRocksDBOperationDurationTime(time.Now(), ops.Exists, "NamespaceMetadata")

	return b.crud.Exists(id)
}

// GetIDs returns all the IDs for the store
func (b *storeImpl) GetIDs(_ context.Context) ([]string, error) {
	defer metrics.SetRocksDBOperationDurationTime(time.Now(), ops.GetAll, "NamespaceMetadataIDs")

	return b.crud.GetKeys()
}

// Get returns the object, if it exists from the store
func (b *storeImpl) Get(_ context.Context, id string) (*storage.NamespaceMetadata, bool, error) {
	defer metrics.SetRocksDBOperationDurationTime(time.Now(), ops.Get, "NamespaceMetadata")

	msg, exists, err := b.crud.Get(id)
	if err != nil || !exists {
		return nil, false, err
	}
	return msg.(*storage.NamespaceMetadata), true, nil
}

// GetMany returns the objects specified by the IDs or the index in the missing indices slice
func (b *storeImpl) GetMany(_ context.Context, ids []string) ([]*storage.NamespaceMetadata, []int, error) {
	defer metrics.SetRocksDBOperationDurationTime(time.Now(), ops.GetMany, "NamespaceMetadata")

	msgs, missingIndices, err := b.crud.GetMany(ids)
	if err != nil {
		return nil, nil, err
	}
	objs := make([]*storage.NamespaceMetadata, 0, len(msgs))
	for _, m := range msgs {
		objs = append(objs, m.(*storage.NamespaceMetadata))
	}
	return objs, missingIndices, nil
}

// Upsert inserts the object into the DB
func (b *storeImpl) Upsert(_ context.Context, obj *storage.NamespaceMetadata) error {
	defer metrics.SetRocksDBOperationDurationTime(time.Now(), ops.Add, "NamespaceMetadata")

	return b.crud.Upsert(obj)
}

// UpsertMany batches objects into the DB
func (b *storeImpl) UpsertMany(_ context.Context, objs []*storage.NamespaceMetadata) error {
	defer metrics.SetRocksDBOperationDurationTime(time.Now(), ops.AddMany, "NamespaceMetadata")

	msgs := make([]proto.Message, 0, len(objs))
	for _, o := range objs {
		msgs = append(msgs, o)
    }

	return b.crud.UpsertMany(msgs)
}

// Delete removes the specified ID from the store
func (b *storeImpl) Delete(_ context.Context, id string) error {
	defer metrics.SetRocksDBOperationDurationTime(time.Now(), ops.Remove, "NamespaceMetadata")

	return b.crud.Delete(id)
}

// Delete removes the specified IDs from the store
func (b *storeImpl) DeleteMany(_ context.Context, ids []string) error {
	defer metrics.SetRocksDBOperationDurationTime(time.Now(), ops.RemoveMany, "NamespaceMetadata")

	return b.crud.DeleteMany(ids)
}

// Walk iterates over all of the objects in the store and applies the closure
func (b *storeImpl) Walk(_ context.Context, fn func(obj *storage.NamespaceMetadata) error) error {
	return b.crud.Walk(func(msg proto.Message) error {
		return fn(msg.(*storage.NamespaceMetadata))
	})
}

// AckKeysIndexed acknowledges the passed keys were indexed
func (b *storeImpl) AckKeysIndexed(_ context.Context, keys ...string) error {
	return b.crud.AckKeysIndexed(keys...)
}

// GetKeysToIndex returns the keys that need to be indexed
func (b *storeImpl) GetKeysToIndex(_ context.Context) ([]string, error) {
	return b.crud.GetKeysToIndex()
}
