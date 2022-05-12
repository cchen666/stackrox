// Code generated by pg-bindings generator. DO NOT EDIT.
package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	metrics "github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	storage "github.com/stackrox/rox/generated/storage"
	ops "github.com/stackrox/rox/pkg/metrics"
	search "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
	"github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/search/postgres/mapping"
)

func init() {
	mapping.RegisterCategoryToTable(v1.SearchCategory_PROCESS_INDICATORS, schema)
}

// NewIndexer returns new indexer for `storage.ProcessIndicator`.
func NewIndexer(db *pgxpool.Pool) *indexerImpl {
	return &indexerImpl{
		db: db,
	}
}

type indexerImpl struct {
	db *pgxpool.Pool
}

func (b *indexerImpl) Count(ctx context.Context, q *v1.Query, opts ...blevesearch.SearchOption) (int, error) {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Count, "ProcessIndicator")

	return postgres.RunCountRequest(v1.SearchCategory_PROCESS_INDICATORS, q, b.db)
}

func (b *indexerImpl) Search(ctx context.Context, q *v1.Query, opts ...blevesearch.SearchOption) ([]search.Result, error) {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Search, "ProcessIndicator")

	return postgres.RunSearchRequest(v1.SearchCategory_PROCESS_INDICATORS, q, b.db)
}

//// Stubs for satisfying interfaces

func (b *indexerImpl) AddProcessIndicator(deployment *storage.ProcessIndicator) error {
	return nil
}

func (b *indexerImpl) AddProcessIndicators(_ []*storage.ProcessIndicator) error {
	return nil
}

func (b *indexerImpl) DeleteProcessIndicator(id string) error {
	return nil
}

func (b *indexerImpl) DeleteProcessIndicators(_ []string) error {
	return nil
}

func (b *indexerImpl) MarkInitialIndexingComplete() error {
	return nil
}

func (b *indexerImpl) NeedsInitialIndexing() (bool, error) {
	return false, nil
}
