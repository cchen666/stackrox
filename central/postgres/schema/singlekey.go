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
	// CreateTableSinglekeyStmt holds the create statement for table `singlekey`.
	CreateTableSinglekeyStmt = &postgres.CreateStmts{
		Table: `
               create table if not exists singlekey (
                   Key varchar,
                   Name varchar UNIQUE,
                   StringSlice text[],
                   Bool bool,
                   Uint64 integer,
                   Int64 integer,
                   Float numeric,
                   Labels jsonb,
                   Timestamp timestamp,
                   Enum integer,
                   Enums int[],
                   serialized bytea,
                   PRIMARY KEY(Key)
               )
               `,
		Indexes: []string{
			"create index if not exists singlekey_Key on singlekey using hash(Key)",
		},
		Children: []*postgres.CreateStmts{},
	}

	// SinglekeySchema is the go schema for table `singlekey`.
	SinglekeySchema = func() *walker.Schema {
		schema := globaldb.GetSchemaForTable("singlekey")
		if schema != nil {
			return schema
		}
		schema = walker.Walk(reflect.TypeOf((*storage.TestSingleKeyStruct)(nil)), "singlekey")
		schema.SetOptionsMap(search.Walk(v1.SearchCategory_SEARCH_UNSET, "singlekey", (*storage.TestSingleKeyStruct)(nil)))
		globaldb.RegisterTable(schema)
		return schema
	}()
)
