package store

import (
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/bolthelper/crud/proto"
)

type storeImpl struct {
	crud proto.MessageCrud
}

// New returns a new Node store
func New(crud proto.MessageCrud) Store {
	return &storeImpl{crud: crud}
}

func (s *storeImpl) CountNodes() (int, error) {
	return s.crud.Count()
}

func (s *storeImpl) ListNodes() ([]*storage.Node, error) {
	entries, err := s.crud.ReadAll()
	if err != nil {
		return nil, err
	}
	nodes := make([]*storage.Node, len(entries))
	for i, entry := range entries {
		nodes[i] = entry.(*storage.Node)
	}
	return nodes, nil
}

func (s *storeImpl) GetNode(id string) (*storage.Node, error) {
	value, err := s.crud.Read(id)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, nil
	}
	return value.(*storage.Node), nil
}

func (s *storeImpl) UpsertNode(node *storage.Node) error {
	_, _, err := s.crud.Upsert(node)
	return err
}

func (s *storeImpl) RemoveNode(id string) error {
	_, _, err := s.crud.Delete(id)
	return err
}
