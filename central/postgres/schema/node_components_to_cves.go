// Code generated by pg-bindings generator. DO NOT EDIT.

package schema

import (
	"fmt"
	"reflect"

	"github.com/stackrox/rox/central/globaldb"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/search"
)

var (
	// CreateTableNodeComponentsToCvesStmt holds the create statement for table `node_components_to_cves`.
	CreateTableNodeComponentsToCvesStmt = &postgres.CreateStmts{
		Table: `
               create table if not exists node_components_to_cves (
                   Id varchar,
                   IsFixable bool,
                   FixedBy varchar,
                   ComponentId varchar,
                   CveId varchar,
                   serialized bytea,
                   PRIMARY KEY(Id, ComponentId, CveId),
                   CONSTRAINT fk_parent_table_0 FOREIGN KEY (ComponentId) REFERENCES node_components(Id) ON DELETE CASCADE
               )
               `,
		Indexes:  []string{},
		Children: []*postgres.CreateStmts{},
	}

	// NodeComponentsToCvesSchema is the go schema for table `node_components_to_cves`.
	NodeComponentsToCvesSchema = func() *walker.Schema {
		schema := globaldb.GetSchemaForTable("node_components_to_cves")
		if schema != nil {
			return schema
		}
		schema = walker.Walk(reflect.TypeOf((*storage.NodeComponentCVEEdge)(nil)), "node_components_to_cves")
		referencedSchemas := map[string]*walker.Schema{
			"storage.ImageComponent": NodeComponentsSchema,
			"storage.CVE":            NodeCvesSchema,
		}

		schema.ResolveReferences(func(messageTypeName string) *walker.Schema {
			return referencedSchemas[fmt.Sprintf("storage.%s", messageTypeName)]
		})
		schema.SetOptionsMap(search.Walk(v1.SearchCategory_NODE_COMPONENT_CVE_EDGE, "node_components_to_cves", (*storage.NodeComponentCVEEdge)(nil)))
		globaldb.RegisterTable(schema)
		return schema
	}()
)
