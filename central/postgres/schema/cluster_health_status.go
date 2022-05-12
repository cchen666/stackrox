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
	// CreateTableClusterHealthStatusStmt holds the create statement for table `cluster_health_status`.
	CreateTableClusterHealthStatusStmt = &postgres.CreateStmts{
		Table: `
               create table if not exists cluster_health_status (
                   Id varchar,
                   SensorHealthStatus integer,
                   CollectorHealthStatus integer,
                   OverallHealthStatus integer,
                   AdmissionControlHealthStatus integer,
                   ScannerHealthStatus integer,
                   serialized bytea,
                   PRIMARY KEY(Id),
                   CONSTRAINT fk_parent_table_0 FOREIGN KEY (Id) REFERENCES clusters(Id) ON DELETE CASCADE
               )
               `,
		Indexes:  []string{},
		Children: []*postgres.CreateStmts{},
	}

	// ClusterHealthStatusSchema is the go schema for table `cluster_health_status`.
	ClusterHealthStatusSchema = func() *walker.Schema {
		schema := globaldb.GetSchemaForTable("cluster_health_status")
		if schema != nil {
			return schema
		}
		schema = walker.Walk(reflect.TypeOf((*storage.ClusterHealthStatus)(nil)), "cluster_health_status")
		referencedSchemas := map[string]*walker.Schema{
			"storage.Cluster": ClustersSchema,
		}

		schema.ResolveReferences(func(messageTypeName string) *walker.Schema {
			return referencedSchemas[fmt.Sprintf("storage.%s", messageTypeName)]
		})
		schema.SetOptionsMap(search.Walk(v1.SearchCategory_CLUSTER_HEALTH_STATUS, "cluster_health_status", (*storage.ClusterHealthStatus)(nil)))
		globaldb.RegisterTable(schema)
		return schema
	}()
)
