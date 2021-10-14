// Code generated by pg-bindings generator. DO NOT EDIT.
package postgres

import (
	mappings "github.com/stackrox/rox/central/rbac/k8srole/mappings"
	metrics "github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	storage "github.com/stackrox/rox/generated/storage"
	ops "github.com/stackrox/rox/pkg/metrics"
	search "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres/mapping"
	"github.com/stackrox/rox/pkg/search/postgres"
	"time"
	"database/sql"
	"github.com/stackrox/rox/pkg/search/blevesearch"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

const table = "k8sroles"

func init() {
	mapping.RegisterCategoryToTable(v1.SearchCategory_ROLES, table)
}

func NewIndexer(db *sql.DB) *indexerImpl {
	return &indexerImpl {
		db: db,
	}
}

type indexerImpl struct {
	db *sql.DB
}

func (b *indexerImpl) AddK8SRole(deployment *storage.K8SRole) error {
	// Added as a part of normal DB op
	return nil
}

func (b *indexerImpl) AddK8SRoles(_ []*storage.K8SRole) error {
	// Added as a part of normal DB op
	return nil
}

func (b *indexerImpl) DeleteK8SRole(id string) error {
	// Removed as a part of normal DB op
	return nil
}

func (b *indexerImpl) DeleteK8SRoles(_ []string) error {
	// Added as a part of normal DB op
	return nil
}

func (b *indexerImpl) MarkInitialIndexingComplete() error {
	return nil
}

func (b *indexerImpl) NeedsInitialIndexing() (bool, error) {
	return false, nil
}

func (b *indexerImpl) Count(q *v1.Query, opts ...blevesearch.SearchOption) (int, error) {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Count, "K8SRole")
	return postgres.RunCountRequest(v1.SearchCategory_ROLES, q, b.db, mappings.OptionsMap)
}

func (b *indexerImpl) Search(q *v1.Query, opts ...blevesearch.SearchOption) ([]search.Result, error) {
	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Search, "K8SRole")
	return postgres.RunSearchRequest(v1.SearchCategory_ROLES, q, b.db, mappings.OptionsMap)
}
