package datastore

import (
	"context"

	sacFilters "github.com/stackrox/stackrox/central/nodecveedge/sac"
	"github.com/stackrox/stackrox/central/nodecveedge/store"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/dackbox/graph"
	"github.com/stackrox/stackrox/pkg/search/filtered"
)

type datastoreImpl struct {
	storage       store.Store
	graphProvider graph.Provider
}

func (ds *datastoreImpl) Get(ctx context.Context, id string) (*storage.NodeCVEEdge, bool, error) {
	filteredIDs, err := ds.filterReadable(ctx, []string{id})
	if err != nil || len(filteredIDs) != 1 {
		return nil, false, err
	}

	edge, found, err := ds.storage.Get(id)
	if err != nil || !found {
		return nil, false, err
	}
	return edge, true, nil
}

func (ds *datastoreImpl) filterReadable(ctx context.Context, ids []string) ([]string, error) {
	var filteredIDs []string
	var err error
	graph.Context(ctx, ds.graphProvider, func(graphContext context.Context) {
		filteredIDs, err = filtered.ApplySACFilter(graphContext, ids, sacFilters.GetSACFilter())
	})
	return filteredIDs, err
}
