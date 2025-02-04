// Code generated by pg-bindings generator. DO NOT EDIT.

package schema

import (
	"reflect"

	"github.com/stackrox/rox/central/globaldb"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/search"
)

var (
	// CreateTableReportconfigsStmt holds the create statement for table `reportconfigs`.
	CreateTableReportconfigsStmt = &postgres.CreateStmts{
		Table: `
               create table if not exists reportconfigs (
                   Id varchar,
                   Name varchar,
                   Type integer,
                   serialized bytea,
                   PRIMARY KEY(Id)
               )
               `,
		Indexes:  []string{},
		Children: []*postgres.CreateStmts{},
	}

	// ReportconfigsSchema is the go schema for table `reportconfigs`.
	ReportconfigsSchema = func() *walker.Schema {
		schema := globaldb.GetSchemaForTable("reportconfigs")
		if schema != nil {
			return schema
		}
		schema = walker.Walk(reflect.TypeOf((*storage.ReportConfiguration)(nil)), "reportconfigs")
		schema.SetOptionsMap(search.Walk(v1.SearchCategory_REPORT_CONFIGURATIONS, "reportconfigs", (*storage.ReportConfiguration)(nil)))
		globaldb.RegisterTable(schema)
		return schema
	}()
)
