package dackbox

import (
	"time"

	"github.com/gogo/protobuf/proto"
	edgeDackBox "github.com/stackrox/stackrox/central/componentcveedge/dackbox"
	"github.com/stackrox/stackrox/central/componentcveedge/store"
	"github.com/stackrox/stackrox/central/metrics"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/dackbox"
	"github.com/stackrox/stackrox/pkg/dackbox/crud"
	ops "github.com/stackrox/stackrox/pkg/metrics"
)

type storeImpl struct {
	dacky *dackbox.DackBox

	reader   crud.Reader
	upserter crud.Upserter
	deleter  crud.Deleter
}

// New returns a new Store instance.
func New(dacky *dackbox.DackBox) (store.Store, error) {
	return &storeImpl{
		dacky:    dacky,
		reader:   edgeDackBox.Reader,
		upserter: edgeDackBox.Upserter,
		deleter:  edgeDackBox.Deleter,
	}, nil
}

func (b *storeImpl) Exists(id string) (bool, error) {
	dackTxn, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return false, err
	}
	defer dackTxn.Discard()

	exists, err := b.reader.ExistsIn(edgeDackBox.BucketHandler.GetKey(id), dackTxn)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (b *storeImpl) Count() (int, error) {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.Count, "ComponentCVEEdge")

	dackTxn, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return 0, err
	}
	defer dackTxn.Discard()

	count, err := b.reader.CountIn(edgeDackBox.Bucket, dackTxn)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (b *storeImpl) GetAll() ([]*storage.ComponentCVEEdge, error) {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.GetAll, "ComponentCVEEdge")

	dackTxn, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return nil, err
	}
	defer dackTxn.Discard()

	msgs, err := b.reader.ReadAllIn(edgeDackBox.Bucket, dackTxn)
	if err != nil {
		return nil, err
	}
	ret := make([]*storage.ComponentCVEEdge, 0, len(msgs))
	for _, msg := range msgs {
		ret = append(ret, msg.(*storage.ComponentCVEEdge))
	}

	return ret, nil
}

func (b *storeImpl) Get(id string) (edges *storage.ComponentCVEEdge, exists bool, err error) {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.Get, "ComponentCVEEdge")

	dackTxn, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return nil, false, err
	}
	defer dackTxn.Discard()

	msg, err := b.reader.ReadIn(edgeDackBox.BucketHandler.GetKey(id), dackTxn)
	if err != nil || msg == nil {
		return nil, false, err
	}

	return msg.(*storage.ComponentCVEEdge), msg != nil, err
}

func (b *storeImpl) GetBatch(ids []string) ([]*storage.ComponentCVEEdge, []int, error) {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.GetMany, "ComponentCVEEdge")

	dackTxn, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return nil, nil, err
	}
	defer dackTxn.Discard()

	msgs := make([]proto.Message, 0, len(ids)/2)
	missing := make([]int, 0, len(ids)/2)
	for idx, id := range ids {
		msg, err := b.reader.ReadIn(edgeDackBox.BucketHandler.GetKey(id), dackTxn)
		if err != nil {
			return nil, nil, err
		}
		if msg != nil {
			msgs = append(msgs, msg)
		} else {
			missing = append(missing, idx)
		}
	}

	ret := make([]*storage.ComponentCVEEdge, 0, len(msgs))
	for _, msg := range msgs {
		ret = append(ret, msg.(*storage.ComponentCVEEdge))
	}

	return ret, missing, nil
}
