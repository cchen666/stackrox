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
	// CreateTableDeploymentsStmt holds the create statement for table `deployments`.
	CreateTableDeploymentsStmt = &postgres.CreateStmts{
		Table: `
               create table if not exists deployments (
                   Id varchar,
                   Name varchar,
                   Type varchar,
                   Namespace varchar,
                   NamespaceId varchar,
                   OrchestratorComponent bool,
                   Labels jsonb,
                   PodLabels jsonb,
                   Created timestamp,
                   ClusterId varchar,
                   ClusterName varchar,
                   Annotations jsonb,
                   Priority integer,
                   ImagePullSecrets text[],
                   ServiceAccount varchar,
                   ServiceAccountPermissionLevel integer,
                   RiskScore numeric,
                   ProcessTags text[],
                   serialized bytea,
                   PRIMARY KEY(Id)
               )
               `,
		Indexes: []string{},
		Children: []*postgres.CreateStmts{
			&postgres.CreateStmts{
				Table: `
               create table if not exists deployments_Containers (
                   deployments_Id varchar,
                   idx integer,
                   Image_Id varchar,
                   Image_Name_Registry varchar,
                   Image_Name_Remote varchar,
                   Image_Name_Tag varchar,
                   Image_Name_FullName varchar,
                   SecurityContext_Privileged bool,
                   SecurityContext_DropCapabilities text[],
                   SecurityContext_AddCapabilities text[],
                   SecurityContext_ReadOnlyRootFilesystem bool,
                   Resources_CpuCoresRequest numeric,
                   Resources_CpuCoresLimit numeric,
                   Resources_MemoryMbRequest numeric,
                   Resources_MemoryMbLimit numeric,
                   PRIMARY KEY(deployments_Id, idx),
                   CONSTRAINT fk_parent_table_0 FOREIGN KEY (deployments_Id) REFERENCES deployments(Id) ON DELETE CASCADE
               )
               `,
				Indexes: []string{
					"create index if not exists deploymentsContainers_idx on deployments_Containers using btree(idx)",
				},
				Children: []*postgres.CreateStmts{
					&postgres.CreateStmts{
						Table: `
               create table if not exists deployments_Containers_Env (
                   deployments_Id varchar,
                   deployments_Containers_idx integer,
                   idx integer,
                   Key varchar,
                   Value varchar,
                   EnvVarSource integer,
                   PRIMARY KEY(deployments_Id, deployments_Containers_idx, idx),
                   CONSTRAINT fk_parent_table_0 FOREIGN KEY (deployments_Id, deployments_Containers_idx) REFERENCES deployments_Containers(deployments_Id, idx) ON DELETE CASCADE
               )
               `,
						Indexes: []string{
							"create index if not exists deploymentsContainersEnv_idx on deployments_Containers_Env using btree(idx)",
						},
						Children: []*postgres.CreateStmts{},
					},
					&postgres.CreateStmts{
						Table: `
               create table if not exists deployments_Containers_Volumes (
                   deployments_Id varchar,
                   deployments_Containers_idx integer,
                   idx integer,
                   Name varchar,
                   Source varchar,
                   Destination varchar,
                   ReadOnly bool,
                   Type varchar,
                   PRIMARY KEY(deployments_Id, deployments_Containers_idx, idx),
                   CONSTRAINT fk_parent_table_0 FOREIGN KEY (deployments_Id, deployments_Containers_idx) REFERENCES deployments_Containers(deployments_Id, idx) ON DELETE CASCADE
               )
               `,
						Indexes: []string{
							"create index if not exists deploymentsContainersVolumes_idx on deployments_Containers_Volumes using btree(idx)",
						},
						Children: []*postgres.CreateStmts{},
					},
					&postgres.CreateStmts{
						Table: `
               create table if not exists deployments_Containers_Secrets (
                   deployments_Id varchar,
                   deployments_Containers_idx integer,
                   idx integer,
                   Name varchar,
                   Path varchar,
                   PRIMARY KEY(deployments_Id, deployments_Containers_idx, idx),
                   CONSTRAINT fk_parent_table_0 FOREIGN KEY (deployments_Id, deployments_Containers_idx) REFERENCES deployments_Containers(deployments_Id, idx) ON DELETE CASCADE
               )
               `,
						Indexes: []string{
							"create index if not exists deploymentsContainersSecrets_idx on deployments_Containers_Secrets using btree(idx)",
						},
						Children: []*postgres.CreateStmts{},
					},
				},
			},
			&postgres.CreateStmts{
				Table: `
               create table if not exists deployments_Ports (
                   deployments_Id varchar,
                   idx integer,
                   ContainerPort integer,
                   Protocol varchar,
                   Exposure integer,
                   PRIMARY KEY(deployments_Id, idx),
                   CONSTRAINT fk_parent_table_0 FOREIGN KEY (deployments_Id) REFERENCES deployments(Id) ON DELETE CASCADE
               )
               `,
				Indexes: []string{
					"create index if not exists deploymentsPorts_idx on deployments_Ports using btree(idx)",
				},
				Children: []*postgres.CreateStmts{
					&postgres.CreateStmts{
						Table: `
               create table if not exists deployments_Ports_ExposureInfos (
                   deployments_Id varchar,
                   deployments_Ports_idx integer,
                   idx integer,
                   Level integer,
                   ServiceName varchar,
                   ServicePort integer,
                   NodePort integer,
                   ExternalIps text[],
                   ExternalHostnames text[],
                   PRIMARY KEY(deployments_Id, deployments_Ports_idx, idx),
                   CONSTRAINT fk_parent_table_0 FOREIGN KEY (deployments_Id, deployments_Ports_idx) REFERENCES deployments_Ports(deployments_Id, idx) ON DELETE CASCADE
               )
               `,
						Indexes: []string{
							"create index if not exists deploymentsPortsExposureInfos_idx on deployments_Ports_ExposureInfos using btree(idx)",
						},
						Children: []*postgres.CreateStmts{},
					},
				},
			},
		},
	}

	// DeploymentsSchema is the go schema for table `deployments`.
	DeploymentsSchema = func() *walker.Schema {
		schema := globaldb.GetSchemaForTable("deployments")
		if schema != nil {
			return schema
		}
		schema = walker.Walk(reflect.TypeOf((*storage.Deployment)(nil)), "deployments")
		referencedSchemas := map[string]*walker.Schema{
			"storage.Image":             ImagesSchema,
			"storage.NamespaceMetadata": NamespacesSchema,
		}

		schema.ResolveReferences(func(messageTypeName string) *walker.Schema {
			return referencedSchemas[fmt.Sprintf("storage.%s", messageTypeName)]
		})
		schema.SetOptionsMap(search.Walk(v1.SearchCategory_DEPLOYMENTS, "deployments", (*storage.Deployment)(nil)))
		globaldb.RegisterTable(schema)
		return schema
	}()
)
