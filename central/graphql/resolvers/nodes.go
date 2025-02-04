package resolvers

import (
	"context"
	"strings"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	complianceStandards "github.com/stackrox/rox/central/compliance/standards"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddQuery("node(id:ID!): Node"),
		schema.AddQuery("nodes(query: String, pagination: Pagination): [Node!]!"),
		schema.AddQuery("nodeCount(query: String): Int!"),
		schema.AddExtraResolver("Node", "complianceResults(query: String): [ControlResult!]!"),
		schema.AddType("ComplianceControlCount", []string{"failingCount: Int!", "passingCount: Int!", "unknownCount: Int!"}),
		schema.AddExtraResolver("Node", "nodeComplianceControlCount(query: String) : ComplianceControlCount!"),
		schema.AddExtraResolver("Node", "controlStatus(query: String): String!"),
		schema.AddExtraResolver("Node", "failingControls(query: String): [ComplianceControl!]!"),
		schema.AddExtraResolver("Node", "passingControls(query: String): [ComplianceControl!]!"),
		schema.AddExtraResolver("Node", "controls(query: String): [ComplianceControl!]!"),
		schema.AddExtraResolver("Node", "cluster: Cluster!"),
		schema.AddExtraResolver("Node", "vulns(query: String, scopeQuery: String, pagination: Pagination): [EmbeddedVulnerability]!"),
		schema.AddExtraResolver("Node", "vulnerabilities(query: String, scopeQuery: String, pagination: Pagination): [NodeVulnerability]!"),
		schema.AddExtraResolver("Node", `unusedVarSink(query: String): Int`),
		schema.AddExtraResolver("Node", "nodeStatus(query: String): String!"),
		schema.AddExtraResolver("Node", "topVuln(query: String): EmbeddedVulnerability"),
		schema.AddExtraResolver("Node", "topVulnerability(query: String): NodeVulnerability"),
		schema.AddExtraResolver("Node", "vulnCount(query: String): Int!"),
		schema.AddExtraResolver("Node", "vulnCounter(query: String): VulnerabilityCounter!"),
		schema.AddExtraResolver("Node", "plottedVulns(query: String): PlottedVulnerabilities!"),
		schema.AddExtraResolver("Node", "components(query: String, pagination: Pagination): [EmbeddedImageScanComponent!]!"),
		schema.AddExtraResolver("Node", `componentCount(query: String): Int!`),
	)
}

// Node returns a resolver for a matching node, or nil if no node is found in any cluster
func (resolver *Resolver) Node(ctx context.Context, args struct{ graphql.ID }) (*nodeResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Node")
	if err := readNodes(ctx); err != nil {
		return nil, err
	}

	nodeLoader, err := loaders.GetNodeLoader(ctx)
	if err != nil {
		return nil, err
	}
	node, err := nodeLoader.FromID(ctx, string(args.ID))
	return resolver.wrapNode(node, node != nil, err)
}

// Nodes returns resolvers for a matching nodes, or nil if no node is found in any cluster
func (resolver *Resolver) Nodes(ctx context.Context, args PaginatedQuery) ([]*nodeResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Nodes")
	if err := readNodes(ctx); err != nil {
		return nil, err
	}

	q, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	nodeLoader, err := loaders.GetNodeLoader(ctx)
	if err != nil {
		return nil, err
	}
	return resolver.wrapNodes(nodeLoader.FromQuery(ctx, q))
}

// NodeCount returns count of nodes across clusters
func (resolver *Resolver) NodeCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "NodeCount")
	if err := readNodes(ctx); err != nil {
		return 0, err
	}
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return 0, err
	}
	results, err := resolver.NodeGlobalDataStore.Search(ctx, query)
	if err != nil {
		return 0, err
	}
	return int32(len(results)), nil
}

func (resolver *nodeResolver) Cluster(ctx context.Context) (*clusterResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Nodes, "Cluster")
	if err := readClusters(ctx); err != nil {
		return nil, err
	}
	return resolver.root.wrapCluster(resolver.root.ClusterDataStore.GetCluster(ctx, resolver.data.GetClusterId()))
}

func (resolver *nodeResolver) ComplianceResults(ctx context.Context, args RawQuery) ([]*controlResultResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Nodes, "ComplianceResults")
	if err := readCompliance(ctx); err != nil {
		return nil, err
	}

	runResults, err := resolver.root.ComplianceAggregator.GetResultsWithEvidence(ctx, args.String())
	if err != nil {
		return nil, err
	}
	output := newBulkControlResults()
	nodeID := resolver.data.GetId()
	output.addNodeData(resolver.root, runResults, func(node *storage.Node, _ *v1.ComplianceControl) bool {
		return node.GetId() == nodeID
	})
	return *output, nil
}

func (resolver *nodeResolver) NodeComplianceControlCount(ctx context.Context, args RawQuery) (*complianceControlCountResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Nodes, "NodeComplianceControlCount")
	if err := readCompliance(ctx); err != nil {
		return nil, err
	}
	scope := []storage.ComplianceAggregation_Scope{storage.ComplianceAggregation_NODE, storage.ComplianceAggregation_CONTROL}
	results, err := resolver.getNodeLastSuccessfulComplianceRunAggregatedResult(ctx, scope, args)
	if err != nil {
		return nil, err
	}
	if results == nil {
		return &complianceControlCountResolver{}, nil
	}
	return getComplianceControlCountFromAggregationResults(results), nil
}

func (resolver *nodeResolver) ControlStatus(ctx context.Context, args RawQuery) (string, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Nodes, "ControlStatus")
	if err := readCompliance(ctx); err != nil {
		return "Fail", err
	}
	r, err := resolver.getNodeLastSuccessfulComplianceRunAggregatedResult(ctx, []storage.ComplianceAggregation_Scope{storage.ComplianceAggregation_NODE}, args)
	if err != nil || r == nil {
		return "Fail", err
	}
	if len(r) != 1 {
		return "Fail", errors.Errorf("unexpected node aggregation results length: expected: 1, actual: %d", len(r))
	}
	return getControlStatusFromAggregationResult(r[0]), nil
}

func (resolver *nodeResolver) getNodeLastSuccessfulComplianceRunAggregatedResult(ctx context.Context, scope []storage.ComplianceAggregation_Scope, args RawQuery) ([]*storage.ComplianceAggregation_Result, error) {
	if err := readCompliance(ctx); err != nil {
		return nil, err
	}
	standardIDs, err := getStandardIDs(ctx, resolver.root.ComplianceStandardStore)
	if err != nil {
		return nil, err
	}
	hasComplianceSuccessfullyRun, err := resolver.root.ComplianceDataStore.IsComplianceRunSuccessfulOnCluster(ctx, resolver.data.GetClusterId(), standardIDs)
	if err != nil || !hasComplianceSuccessfullyRun {
		return nil, err
	}
	query, err := search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetClusterId()).
		AddExactMatches(search.NodeID, resolver.data.GetId()).RawQuery()
	if err != nil {
		return nil, err
	}
	if args.Query != nil {
		query = strings.Join([]string{query, args.String()}, "+")
	}
	r, _, _, err := resolver.root.ComplianceAggregator.Aggregate(ctx, query, scope, storage.ComplianceAggregation_CONTROL)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (resolver *nodeResolver) FailingControls(ctx context.Context, args RawQuery) ([]*complianceControlResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Nodes, "FailingControls")
	if err := readCompliance(ctx); err != nil {
		return nil, err
	}
	scope := []storage.ComplianceAggregation_Scope{storage.ComplianceAggregation_NODE, storage.ComplianceAggregation_CONTROL}
	results, err := resolver.getNodeLastSuccessfulComplianceRunAggregatedResult(ctx, scope, args)
	if err != nil {
		return nil, err
	}
	resolvers, err := resolver.root.wrapComplianceControls(getComplianceControlsFromAggregationResults(results, failing, resolver.root.ComplianceStandardStore))
	if err != nil {
		return nil, err
	}
	return resolvers, nil
}

func (resolver *nodeResolver) PassingControls(ctx context.Context, args RawQuery) ([]*complianceControlResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Nodes, "PassingControls")
	if err := readCompliance(ctx); err != nil {
		return nil, err
	}
	scope := []storage.ComplianceAggregation_Scope{storage.ComplianceAggregation_NODE, storage.ComplianceAggregation_CONTROL}
	results, err := resolver.getNodeLastSuccessfulComplianceRunAggregatedResult(ctx, scope, args)
	if err != nil {
		return nil, err
	}
	resolvers, err := resolver.root.wrapComplianceControls(getComplianceControlsFromAggregationResults(results, passing, resolver.root.ComplianceStandardStore))
	if err != nil {
		return nil, err
	}
	return resolvers, nil
}

func (resolver *nodeResolver) Controls(ctx context.Context, args RawQuery) ([]*complianceControlResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Nodes, "Controls")
	if err := readCompliance(ctx); err != nil {
		return nil, err
	}
	scope := []storage.ComplianceAggregation_Scope{storage.ComplianceAggregation_NODE, storage.ComplianceAggregation_CONTROL}
	results, err := resolver.getNodeLastSuccessfulComplianceRunAggregatedResult(ctx, scope, args)
	if err != nil {
		return nil, err
	}
	resolvers, err := resolver.root.wrapComplianceControls(getComplianceControlsFromAggregationResults(results, any, resolver.root.ComplianceStandardStore))
	if err != nil {
		return nil, err
	}
	return resolvers, nil
}

func getComplianceControlsFromAggregationResults(results []*storage.ComplianceAggregation_Result, controlType resultType, cs complianceStandards.Repository) ([]*v1.ComplianceControl, error) {
	if cs == nil {
		return nil, errors.New("empty compliance standards store encountered: argument cs is nil")
	}
	var controls []*v1.ComplianceControl
	for _, r := range results {
		if (controlType == passing && r.GetNumPassing() == 0) || (controlType == failing && r.GetNumFailing() == 0) {
			continue
		}
		controlID, err := getScopeIDFromAggregationResult(r, storage.ComplianceAggregation_CONTROL)
		if err != nil {
			continue
		}
		control := cs.Control(controlID)
		if control == nil {
			continue
		}
		controls = append(controls, control)
	}
	return controls, nil
}

func getComplianceControlCountFromAggregationResults(results []*storage.ComplianceAggregation_Result) *complianceControlCountResolver {
	ret := &complianceControlCountResolver{}
	for _, r := range results {
		if r.GetNumFailing() != 0 {
			ret.failingCount++
		} else if r.GetNumPassing() != 0 {
			ret.passingCount++
		} else {
			ret.unknownCount++
		}
	}
	return ret
}

type complianceControlCountResolver struct {
	failingCount int32
	passingCount int32
	unknownCount int32
}

func (resolver *complianceControlCountResolver) FailingCount() int32 {
	return resolver.failingCount
}

func (resolver *complianceControlCountResolver) PassingCount() int32 {
	return resolver.passingCount
}

func (resolver *complianceControlCountResolver) UnknownCount() int32 {
	return resolver.unknownCount
}

func (resolver *nodeResolver) NodeStatus(ctx context.Context, args RawQuery) (string, error) {
	return "active", nil
}

// Compoonents returns all of the components in the node.
func (resolver *nodeResolver) Components(ctx context.Context, args PaginatedQuery) ([]ComponentResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Nodes, "NodeComponents")
	if err := readNodes(ctx); err != nil {
		return nil, err
	}
	query := search.AddRawQueriesAsConjunction(args.String(), resolver.getNodeRawQuery())

	return resolver.root.componentsV2(resolver.nodeScopeContext(ctx), PaginatedQuery{Query: &query, Pagination: args.Pagination})
}

// ComponentCount returns the number of components in the node
func (resolver *nodeResolver) ComponentCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Nodes, "ComponentCount")
	if err := readNodes(ctx); err != nil {
		return 0, err
	}

	query := search.AddRawQueriesAsConjunction(args.String(), resolver.getNodeRawQuery())

	return resolver.root.componentCountV2(resolver.nodeScopeContext(ctx), RawQuery{Query: &query})
}

// TopVuln returns the first vulnerability with the top CVSS score.
func (resolver *nodeResolver) TopVuln(ctx context.Context, args RawQuery) (VulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Nodes, "TopVulnerability")
	if err := readNodes(ctx); err != nil {
		return nil, err
	}

	return resolver.unwrappedTopVulnQuery(ctx, args)
}

// TopVulnerability returns the first node vulnerability with the top CVSS score.
func (resolver *nodeResolver) TopVulnerability(ctx context.Context, args RawQuery) (NodeVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Nodes, "TopVulnerability")
	if err := readNodes(ctx); err != nil {
		return nil, err
	}

	if !features.PostgresDatastore.Enabled() {
		return resolver.unwrappedTopVulnQuery(ctx, args)
	}
	// TODO : Add postgres support
	return nil, errors.New("Sub-resolver TopVuln in nodeResolver does not support postgres yet")
}

func (resolver *nodeResolver) unwrappedTopVulnQuery(ctx context.Context, args RawQuery) (*cVEResolver, error) {
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	if resolver.data.GetSetTopCvss() == nil {
		return nil, nil
	}

	query = search.ConjunctionQuery(query, resolver.getNodeQuery())
	query.Pagination = &v1.QueryPagination{
		SortOptions: []*v1.QuerySortOption{
			{
				Field:    search.CVSS.String(),
				Reversed: true,
			},
			{
				Field:    search.CVE.String(),
				Reversed: true,
			},
		},
		Limit:  1,
		Offset: 0,
	}

	vulnLoader, err := loaders.GetCVELoader(ctx)
	if err != nil {
		return nil, err
	}
	vulns, err := vulnLoader.FromQuery(ctx, query)
	if err != nil {
		return nil, err
	} else if len(vulns) == 0 {
		return nil, err
	} else if len(vulns) > 1 {
		return nil, errors.New("multiple vulnerabilities matched for top node vulnerability")
	}
	return &cVEResolver{root: resolver.root, data: vulns[0]}, nil
}

func (resolver *nodeResolver) getNodeRawQuery() string {
	return search.NewQueryBuilder().AddExactMatches(search.NodeID, resolver.data.GetId()).Query()
}

func (resolver *nodeResolver) getNodeQuery() *v1.Query {
	return search.NewQueryBuilder().AddExactMatches(search.NodeID, resolver.data.GetId()).ProtoQuery()
}

// Vulns returns all of the vulnerabilities in the node.
func (resolver *nodeResolver) Vulns(ctx context.Context, args PaginatedQuery) ([]VulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Nodes, "Vulnerabilities")
	if err := readNodes(ctx); err != nil {
		return nil, err
	}
	query := search.AddRawQueriesAsConjunction(args.String(), resolver.getNodeRawQuery())

	return resolver.root.vulnerabilitiesV2(resolver.nodeScopeContext(ctx), PaginatedQuery{Query: &query, Pagination: args.Pagination})
}

// Vulnerabilities returns all of the vulnerabilities in the node.
func (resolver *nodeResolver) Vulnerabilities(ctx context.Context, args PaginatedQuery) ([]NodeVulnerabilityResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Nodes, "Vulnerabilities")
	if err := readNodes(ctx); err != nil {
		return nil, err
	}

	if !features.PostgresDatastore.Enabled() {
		return resolver.root.NodeVulnerabilities(resolver.nodeScopeContext(ctx), args)
	}
	// TODO : Add postgres support
	return nil, errors.New("Sub-resolver Vulns in nodeResolver does not support postgres yet")
}

// VulnCount returns the number of vulnerabilities the node has.
func (resolver *nodeResolver) VulnCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Nodes, "VulnerabilityCount")
	if err := readNodes(ctx); err != nil {
		return 0, err
	}

	if !features.PostgresDatastore.Enabled() {
		return resolver.root.NodeVulnerabilityCount(resolver.nodeScopeContext(ctx), args)
	}
	// TODO : Add postgres support
	return 0, errors.New("Sub-resolver VulnCount in nodeResolver does not support postgres yet")
}

// VulnCounter resolves the number of different types of vulnerabilities contained in a node.
func (resolver *nodeResolver) VulnCounter(ctx context.Context, args RawQuery) (*VulnerabilityCounterResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Nodes, "VulnCounter")
	if err := readNodes(ctx); err != nil {
		return nil, err
	}

	if !features.PostgresDatastore.Enabled() {
		return resolver.root.NodeVulnCounter(resolver.nodeScopeContext(ctx), args)
	}
	// TODO : Add postgres support
	return nil, errors.New("Sub-resolver VulnCounter in nodeResolver does not support postgres yet")
}

// PlottedVulns returns the data required by top risky entity scatter-plot on vuln mgmt dashboard
func (resolver *nodeResolver) PlottedVulns(ctx context.Context, args RawQuery) (*PlottedVulnerabilitiesResolver, error) {
	query := search.AddRawQueriesAsConjunction(args.String(), resolver.getNodeRawQuery())
	return newPlottedVulnerabilitiesResolver(ctx, resolver.root, RawQuery{Query: &query})
}

func (resolver *nodeResolver) UnusedVarSink(ctx context.Context, args RawQuery) *int32 {
	return nil
}

func (resolver *nodeResolver) nodeScopeContext(ctx context.Context) context.Context {
	return scoped.Context(ctx, scoped.Scope{
		Level: v1.SearchCategory_NODES,
		ID:    resolver.data.GetId(),
	})
}
