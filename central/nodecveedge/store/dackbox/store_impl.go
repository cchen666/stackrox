package dackbox

import (
	"time"

	"github.com/stackrox/stackrox/central/metrics"
	edgeDackBox "github.com/stackrox/stackrox/central/nodecveedge/dackbox"
	"github.com/stackrox/stackrox/central/nodecveedge/store"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/dackbox"
	"github.com/stackrox/stackrox/pkg/dackbox/crud"
	ops "github.com/stackrox/stackrox/pkg/metrics"
)

const (
	typ = "NodeCVEEdge"
)

type storeImpl struct {
	dacky *dackbox.DackBox

	reader   crud.Reader
	upserter crud.Upserter
	deleter  crud.Deleter
}

// New returns a new Store instance.
func New(dacky *dackbox.DackBox) store.Store {
	return &storeImpl{
		dacky:    dacky,
		reader:   edgeDackBox.Reader,
		upserter: edgeDackBox.Upserter,
		deleter:  edgeDackBox.Deleter,
	}
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
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.Count, typ)

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

func (b *storeImpl) GetAll() ([]*storage.NodeCVEEdge, error) {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.GetAll, typ)

	dackTxn, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return nil, err
	}
	defer dackTxn.Discard()

	msgs, err := b.reader.ReadAllIn(edgeDackBox.Bucket, dackTxn)
	if err != nil {
		return nil, err
	}
	ret := make([]*storage.NodeCVEEdge, 0, len(msgs))
	for _, msg := range msgs {
		ret = append(ret, msg.(*storage.NodeCVEEdge))
	}

	return ret, nil
}

func (b *storeImpl) Get(id string) (*storage.NodeCVEEdge, bool, error) {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.Get, typ)

	dackTxn, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return nil, false, err
	}
	defer dackTxn.Discard()

	msg, err := b.reader.ReadIn(edgeDackBox.BucketHandler.GetKey(id), dackTxn)
	if err != nil || msg == nil {
		return nil, false, err
	}

	return msg.(*storage.NodeCVEEdge), true, err
}

func (b *storeImpl) GetBatch(ids []string) ([]*storage.NodeCVEEdge, []int, error) {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.GetMany, typ)

	dackTxn, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return nil, nil, err
	}
	defer dackTxn.Discard()

	ret := make([]*storage.NodeCVEEdge, 0, len(ids))
	var missing []int
	for idx, id := range ids {
		msg, err := b.reader.ReadIn(edgeDackBox.BucketHandler.GetKey(id), dackTxn)
		if err != nil {
			return nil, nil, err
		}
		if msg != nil {
			ret = append(ret, msg.(*storage.NodeCVEEdge))
		} else {
			missing = append(missing, idx)
		}
	}

	return ret, missing, nil
}
