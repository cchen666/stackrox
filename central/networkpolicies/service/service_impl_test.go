package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	cDataStoreMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	dDataStoreMocks "github.com/stackrox/rox/central/deployment/datastore/mocks"
	networkBaselineDSMocks "github.com/stackrox/rox/central/networkbaseline/datastore/mocks"
	graphConfigMocks "github.com/stackrox/rox/central/networkgraph/config/datastore/mocks"
	netEntityDSMocks "github.com/stackrox/rox/central/networkgraph/entity/datastore/mocks"
	netTreeMgrMocks "github.com/stackrox/rox/central/networkgraph/entity/networktree/mocks"
	npMocks "github.com/stackrox/rox/central/networkpolicies/datastore/mocks"
	npGraphMocks "github.com/stackrox/rox/central/networkpolicies/graph/mocks"
	nDataStoreMocks "github.com/stackrox/rox/central/notifier/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	grpcTestutils "github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stackrox/rox/pkg/networkgraph/tree"
	"github.com/stackrox/rox/pkg/protoconv/networkpolicy"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/suite"
)

const fakeClusterID = "FAKECLUSTERID"
const fakeDeploymentID = "FAKEDEPLOYMENTID"
const badYAML = `
kind: NetworkPolicy
apiVersion: networking.k8s.io/v1
metadata:
  name: first-policy
spec:
  podSelector: {}
  ingress: []
`
const fakeYAML1 = `
kind: NetworkPolicy
apiVersion: networking.k8s.io/v1
metadata:
  name: first-policy
  namespace: default
spec:
  podSelector: {}
  ingress: []
`
const fakeYAML2 = `
kind: NetworkPolicy
apiVersion: networking.k8s.io/v1
metadata:
  name: second-policy
  namespace: default
spec:
  podSelector: {}
  ingress: []
`
const combinedYAMLs = `---
kind: NetworkPolicy
apiVersion: networking.k8s.io/v1
metadata:
  name: first-policy
  namespace: default
spec:
  podSelector: {}
  ingress: []
---
kind: NetworkPolicy
apiVersion: networking.k8s.io/v1
metadata:
  name: second-policy
  namespace: default
spec:
  podSelector: {}
  ingress: []
`

func TestNetworkPolicyService(t *testing.T) {
	suite.Run(t, new(ServiceTestSuite))
}

type ServiceTestSuite struct {
	suite.Suite

	requestContext   context.Context
	clusters         *cDataStoreMocks.MockDataStore
	deployments      *dDataStoreMocks.MockDataStore
	externalSrcs     *netEntityDSMocks.MockEntityDataStore
	graphConfig      *graphConfigMocks.MockDataStore
	networkBaselines *networkBaselineDSMocks.MockDataStore
	netTreeMgr       *netTreeMgrMocks.MockManager
	networkPolicies  *npMocks.MockDataStore
	evaluator        *npGraphMocks.MockEvaluator
	notifiers        *nDataStoreMocks.MockDataStore
	tested           Service
	mockCtrl         *gomock.Controller
	envIsolator      *envisolator.EnvIsolator
}

func (suite *ServiceTestSuite) SetupTest() {
	// Since all the datastores underneath are mocked, the context of the request doesns't need any permissions.
	suite.requestContext = context.Background()

	suite.mockCtrl = gomock.NewController(suite.T())
	suite.networkPolicies = npMocks.NewMockDataStore(suite.mockCtrl)
	suite.evaluator = npGraphMocks.NewMockEvaluator(suite.mockCtrl)
	suite.clusters = cDataStoreMocks.NewMockDataStore(suite.mockCtrl)
	suite.deployments = dDataStoreMocks.NewMockDataStore(suite.mockCtrl)
	suite.externalSrcs = netEntityDSMocks.NewMockEntityDataStore(suite.mockCtrl)
	suite.graphConfig = graphConfigMocks.NewMockDataStore(suite.mockCtrl)
	suite.networkBaselines = networkBaselineDSMocks.NewMockDataStore(suite.mockCtrl)
	suite.netTreeMgr = netTreeMgrMocks.NewMockManager(suite.mockCtrl)
	suite.notifiers = nDataStoreMocks.NewMockDataStore(suite.mockCtrl)
	suite.envIsolator = envisolator.NewEnvIsolator(suite.T())
	suite.envIsolator.Setenv(features.NetworkDetectionBaselineSimulation.EnvVar(), "true")

	suite.tested = New(suite.networkPolicies, suite.deployments, suite.externalSrcs, suite.graphConfig, suite.networkBaselines, suite.netTreeMgr,
		suite.evaluator, nil, suite.clusters, suite.notifiers, nil, nil)
}

func (suite *ServiceTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *ServiceTestSuite) TestAuth() {
	grpcTestutils.AssertAuthzWorks(suite.T(), suite.tested)
}

func (suite *ServiceTestSuite) TestFailsIfClusterIsNotSet() {
	request := &v1.SimulateNetworkGraphRequest{}
	_, err := suite.tested.SimulateNetworkGraph(suite.requestContext, request)
	suite.Error(err, "expected graph generation to fail since no cluster is specified")
}

func (suite *ServiceTestSuite) TestFailsIfClusterDoesNotExist() {
	// Mock that cluster exists.
	suite.clusters.EXPECT().Exists(gomock.Any(), fakeClusterID).
		Return(false, nil)

	// Make the request to the service and check that it did not err.
	request := &v1.SimulateNetworkGraphRequest{
		ClusterId:       fakeClusterID,
		IncludeNodeDiff: true,
	}
	_, err := suite.tested.SimulateNetworkGraph(suite.requestContext, request)
	suite.Error(err, "expected graph generation to fail since cluster does not exist")
}

func (suite *ServiceTestSuite) TestRejectsYamlWithoutNamespace() {
	// Mock that cluster exists.
	suite.clusters.EXPECT().Exists(gomock.Any(), fakeClusterID).
		Return(true, nil)

	// Make the request to the service and check that it did not err.
	request := &v1.SimulateNetworkGraphRequest{
		ClusterId: fakeClusterID,
		Modification: &storage.NetworkPolicyModification{
			ApplyYaml: badYAML,
		},
		IncludeNodeDiff: true,
	}
	_, err := suite.tested.SimulateNetworkGraph(suite.requestContext, request)
	suite.Error(err, "expected graph generation to fail since input yaml has no namespace")
}

func (suite *ServiceTestSuite) TestGetNetworkGraph() {
	// Mock that cluster exists.
	suite.clusters.EXPECT().Exists(gomock.Any(), fakeClusterID).
		Return(true, nil)

	// Mock that we receive deployments for the cluster
	deps := make([]*storage.Deployment, 0)
	var networkTree tree.ReadOnlyNetworkTree
	suite.deployments.EXPECT().SearchRawDeployments(gomock.Any(), deploymentSearchIsForCluster(fakeClusterID)).
		Return(deps, nil)

	// Mock that we have network policies in effect for the cluster.
	pols := make([]*storage.NetworkPolicy, 0)
	suite.networkPolicies.EXPECT().GetNetworkPolicies(suite.requestContext, networkPolicyGetIsForCluster(fakeClusterID), "").
		Return(pols, nil)
	suite.graphConfig.EXPECT().GetNetworkGraphConfig(gomock.Any()).Return(&storage.NetworkGraphConfig{HideDefaultExternalSrcs: true}, nil)
	suite.netTreeMgr.EXPECT().GetReadOnlyNetworkTree(gomock.Any(), fakeClusterID).Return(nil)

	// Check that the evaluator gets called with our created deployment and policy set.
	expectedGraph := &v1.NetworkGraph{}
	suite.evaluator.EXPECT().GetGraph(fakeClusterID, set.NewStringSet(), deps, networkTree, pols, false).
		Return(expectedGraph)
	expectedResp := &v1.SimulateNetworkGraphResponse{
		SimulatedGraph: expectedGraph,
		Policies:       []*v1.NetworkPolicyInSimulation{},
	}

	// Make the request to the service and check that it did not err.
	request := &v1.SimulateNetworkGraphRequest{
		ClusterId:       fakeClusterID,
		IncludeNodeDiff: true,
	}
	actualResp, err := suite.tested.SimulateNetworkGraph(suite.requestContext, request)
	suite.NoError(err, "expected graph generation to succeed")
	suite.Equal(expectedResp, actualResp, "response should be output from graph generation")
}

func (suite *ServiceTestSuite) TestGetNetworkGraphWithReplacement() {
	// Mock that cluster exists.
	suite.clusters.EXPECT().Exists(gomock.Any(), fakeClusterID).
		Return(true, nil)

	// Mock that we receive deployments for the cluster
	deps := make([]*storage.Deployment, 0)
	var networkTree tree.ReadOnlyNetworkTree
	suite.deployments.EXPECT().SearchRawDeployments(gomock.Any(), deploymentSearchIsForCluster(fakeClusterID)).
		Return(deps, nil)

	// Mock that we have network policies in effect for the cluster.
	compiledPolicies, _ := networkpolicy.YamlWrap{Yaml: fakeYAML1}.ToRoxNetworkPolicies()
	pols := []*storage.NetworkPolicy{
		compiledPolicies[0],
	}
	suite.networkPolicies.EXPECT().GetNetworkPolicies(suite.requestContext, networkPolicyGetIsForCluster(fakeClusterID), "").
		Return(pols, nil)
	suite.graphConfig.EXPECT().GetNetworkGraphConfig(gomock.Any()).Return(&storage.NetworkGraphConfig{HideDefaultExternalSrcs: true}, nil)
	suite.netTreeMgr.EXPECT().GetReadOnlyNetworkTree(gomock.Any(), fakeClusterID).Return(nil)

	// Check that the evaluator gets called with our created deployment and policy set.
	expectedGraph := &v1.NetworkGraph{}
	suite.evaluator.EXPECT().GetGraph(fakeClusterID, set.NewStringSet(), deps, networkTree, checkHasPolicies("first-policy"), false).
		Return(expectedGraph)
	suite.evaluator.EXPECT().GetGraph(fakeClusterID, set.NewStringSet(), deps, networkTree, checkHasPolicies("first-policy"), false).
		Return(expectedGraph)

	// Make the request to the service and check that it did not err.
	request := &v1.SimulateNetworkGraphRequest{
		ClusterId: fakeClusterID,
		Modification: &storage.NetworkPolicyModification{
			ApplyYaml: fakeYAML1,
		},
		IncludeNodeDiff: true,
	}
	actualResp, err := suite.tested.SimulateNetworkGraph(suite.requestContext, request)
	suite.NoError(err, "expected graph generation to succeed")
	suite.Equal(expectedGraph, actualResp.GetSimulatedGraph(), "response should be output from graph generation")
	suite.Require().Len(actualResp.GetPolicies(), 1)
	suite.Equal("first-policy", actualResp.GetPolicies()[0].GetPolicy().GetName())
	suite.Equal(v1.NetworkPolicyInSimulation_MODIFIED, actualResp.GetPolicies()[0].GetStatus())
}

func (suite *ServiceTestSuite) TestGetNetworkGraphWithAddition() {
	// Mock that cluster exists.
	suite.clusters.EXPECT().Exists(gomock.Any(), fakeClusterID).
		Return(true, nil)

	// Mock that we receive deployments for the cluster
	deps := make([]*storage.Deployment, 0)
	var networkTree tree.ReadOnlyNetworkTree
	suite.deployments.EXPECT().SearchRawDeployments(gomock.Any(), deploymentSearchIsForCluster(fakeClusterID)).
		Return(deps, nil)

	// Mock that we have network policies in effect for the cluster.
	compiledPolicies, _ := networkpolicy.YamlWrap{Yaml: fakeYAML2}.ToRoxNetworkPolicies()
	suite.networkPolicies.EXPECT().GetNetworkPolicies(suite.requestContext, networkPolicyGetIsForCluster(fakeClusterID), "").
		Return(compiledPolicies, nil)
	suite.graphConfig.EXPECT().GetNetworkGraphConfig(gomock.Any()).Return(&storage.NetworkGraphConfig{HideDefaultExternalSrcs: true}, nil)
	suite.netTreeMgr.EXPECT().GetReadOnlyNetworkTree(gomock.Any(), fakeClusterID).Return(nil)

	// Check that the evaluator gets called with our created deployment and policy set.
	expectedGraph := &v1.NetworkGraph{}
	suite.evaluator.EXPECT().GetGraph(fakeClusterID, set.NewStringSet(), deps, networkTree, checkHasPolicies("first-policy", "second-policy"), false).
		Return(expectedGraph)
	suite.evaluator.EXPECT().GetGraph(fakeClusterID, set.NewStringSet(), deps, networkTree, checkHasPolicies("second-policy"), false).
		Return(expectedGraph)

	request := &v1.SimulateNetworkGraphRequest{
		ClusterId: fakeClusterID,
		Modification: &storage.NetworkPolicyModification{
			ApplyYaml: fakeYAML1,
		},
		IncludeNodeDiff: true,
	}
	actualResp, err := suite.tested.SimulateNetworkGraph(suite.requestContext, request)
	suite.NoError(err, "expected graph generation to succeed")
	suite.Equal(expectedGraph, actualResp.GetSimulatedGraph(), "response should be output from graph generation")
	suite.Require().Len(actualResp.GetPolicies(), 2)
	suite.Equal("second-policy", actualResp.GetPolicies()[0].GetPolicy().GetName())
	suite.Equal(v1.NetworkPolicyInSimulation_UNCHANGED, actualResp.GetPolicies()[0].GetStatus())
	suite.Equal("first-policy", actualResp.GetPolicies()[1].GetPolicy().GetName())
	suite.Equal(v1.NetworkPolicyInSimulation_ADDED, actualResp.GetPolicies()[1].GetStatus())
}

func (suite *ServiceTestSuite) TestGetNetworkGraphWithReplacementAndAddition() {
	// Mock that cluster exists.
	suite.clusters.EXPECT().Exists(gomock.Any(), fakeClusterID).
		Return(true, nil)

	// Mock that we receive deployments for the cluster
	deps := make([]*storage.Deployment, 0)
	var networkTree tree.ReadOnlyNetworkTree
	suite.deployments.EXPECT().SearchRawDeployments(gomock.Any(), deploymentSearchIsForCluster(fakeClusterID)).
		Return(deps, nil)

	// Mock that we have network policies in effect for the cluster.
	compiledPolicies, _ := networkpolicy.YamlWrap{Yaml: fakeYAML1}.ToRoxNetworkPolicies()
	suite.networkPolicies.EXPECT().GetNetworkPolicies(suite.requestContext, networkPolicyGetIsForCluster(fakeClusterID), "").
		Return(compiledPolicies, nil)
	suite.graphConfig.EXPECT().GetNetworkGraphConfig(gomock.Any()).Return(&storage.NetworkGraphConfig{HideDefaultExternalSrcs: true}, nil)
	suite.netTreeMgr.EXPECT().GetReadOnlyNetworkTree(gomock.Any(), fakeClusterID).Return(nil)

	// Check that the evaluator gets called with our created deployment and policy set.
	expectedGraph := &v1.NetworkGraph{}
	suite.evaluator.EXPECT().GetGraph(fakeClusterID, set.NewStringSet(), deps, networkTree, checkHasPolicies("first-policy", "second-policy"), false).
		Return(expectedGraph)
	suite.evaluator.EXPECT().GetGraph(fakeClusterID, set.NewStringSet(), deps, networkTree, checkHasPolicies("first-policy"), false).
		Return(expectedGraph)

	// Make the request to the service and check that it did not err.
	request := &v1.SimulateNetworkGraphRequest{
		ClusterId: fakeClusterID,
		Modification: &storage.NetworkPolicyModification{
			ApplyYaml: combinedYAMLs,
		},
		IncludeNodeDiff: true,
	}
	actualResp, err := suite.tested.SimulateNetworkGraph(suite.requestContext, request)
	suite.NoError(err, "expected graph generation to succeed")

	suite.Equal(expectedGraph, actualResp.GetSimulatedGraph(), "response should be output from graph generation")
	suite.Require().Len(actualResp.GetPolicies(), 2)
	suite.Equal("first-policy", actualResp.GetPolicies()[0].GetPolicy().GetName())
	suite.Equal(v1.NetworkPolicyInSimulation_MODIFIED, actualResp.GetPolicies()[0].GetStatus())
	suite.Equal("second-policy", actualResp.GetPolicies()[1].GetPolicy().GetName())
	suite.Equal(v1.NetworkPolicyInSimulation_ADDED, actualResp.GetPolicies()[1].GetStatus())
}

func (suite *ServiceTestSuite) TestGetNetworkGraphWithDeletion() {
	// Mock that cluster exists.
	suite.clusters.EXPECT().Exists(gomock.Any(), fakeClusterID).
		Return(true, nil)

	// Mock that we receive deployments for the cluster
	deps := make([]*storage.Deployment, 0)
	var networkTree tree.ReadOnlyNetworkTree
	suite.deployments.EXPECT().SearchRawDeployments(gomock.Any(), deploymentSearchIsForCluster(fakeClusterID)).
		Return(deps, nil)

	// Mock that we have network policies in effect for the cluster.
	compiledPolicies, _ := networkpolicy.YamlWrap{Yaml: fakeYAML1}.ToRoxNetworkPolicies()
	suite.networkPolicies.EXPECT().GetNetworkPolicies(suite.requestContext, networkPolicyGetIsForCluster(fakeClusterID), "").
		Return(compiledPolicies, nil)
	suite.graphConfig.EXPECT().GetNetworkGraphConfig(gomock.Any()).Return(&storage.NetworkGraphConfig{HideDefaultExternalSrcs: true}, nil)
	suite.netTreeMgr.EXPECT().GetReadOnlyNetworkTree(gomock.Any(), fakeClusterID).Return(nil)

	// Check that the evaluator gets called with our created deployment and policy set.
	expectedGraph := &v1.NetworkGraph{}
	suite.evaluator.EXPECT().GetGraph(fakeClusterID, set.NewStringSet(), deps, networkTree, checkHasPolicies(), false).
		Return(expectedGraph)
	suite.evaluator.EXPECT().GetGraph(fakeClusterID, set.NewStringSet(), deps, networkTree, checkHasPolicies("first-policy"), false).
		Return(expectedGraph)

	// Make the request to the service and check that it did not err.
	request := &v1.SimulateNetworkGraphRequest{
		ClusterId: fakeClusterID,
		Modification: &storage.NetworkPolicyModification{
			ToDelete: []*storage.NetworkPolicyReference{
				{
					Namespace: "default",
					Name:      "first-policy",
				},
			},
		},
		IncludeNodeDiff: true,
	}
	actualResp, err := suite.tested.SimulateNetworkGraph(suite.requestContext, request)
	suite.NoError(err, "expected graph generation to succeed")

	suite.Equal(expectedGraph, actualResp.GetSimulatedGraph(), "response should be output from graph generation")
	suite.Require().Len(actualResp.GetPolicies(), 1)
	suite.Equal("first-policy", actualResp.GetPolicies()[0].GetOldPolicy().GetName())
	suite.Equal(v1.NetworkPolicyInSimulation_DELETED, actualResp.GetPolicies()[0].GetStatus())
}

func (suite *ServiceTestSuite) TestGetNetworkGraphWithDeletionAndAdditionOfSame() {
	// Mock that cluster exists.
	suite.clusters.EXPECT().Exists(gomock.Any(), fakeClusterID).
		Return(true, nil)

	// Mock that we receive deployments for the cluster
	deps := make([]*storage.Deployment, 0)
	var networkTree tree.ReadOnlyNetworkTree
	suite.deployments.EXPECT().SearchRawDeployments(gomock.Any(), deploymentSearchIsForCluster(fakeClusterID)).
		Return(deps, nil)

	// Mock that we have network policies in effect for the cluster.
	compiledPolicies, _ := networkpolicy.YamlWrap{Yaml: fakeYAML2}.ToRoxNetworkPolicies()
	suite.networkPolicies.EXPECT().GetNetworkPolicies(suite.requestContext, networkPolicyGetIsForCluster(fakeClusterID), "").
		Return(compiledPolicies, nil)
	suite.graphConfig.EXPECT().GetNetworkGraphConfig(gomock.Any()).Return(&storage.NetworkGraphConfig{HideDefaultExternalSrcs: true}, nil)
	suite.netTreeMgr.EXPECT().GetReadOnlyNetworkTree(gomock.Any(), fakeClusterID).Return(nil)

	// Check that the evaluator gets called with our created deployment and policy set.
	expectedGraph := &v1.NetworkGraph{}
	suite.evaluator.EXPECT().GetGraph(fakeClusterID, set.NewStringSet(), deps, networkTree, checkHasPolicies("first-policy", "second-policy"), false).
		Return(expectedGraph)
	suite.evaluator.EXPECT().GetGraph(fakeClusterID, set.NewStringSet(), deps, networkTree, checkHasPolicies("second-policy"), false).
		Return(expectedGraph)

	request := &v1.SimulateNetworkGraphRequest{
		ClusterId: fakeClusterID,
		Modification: &storage.NetworkPolicyModification{
			ToDelete: []*storage.NetworkPolicyReference{
				{
					Namespace: "default",
					Name:      "second-policy",
				},
			},
			ApplyYaml: combinedYAMLs,
		},
		IncludeNodeDiff: true,
	}
	actualResp, err := suite.tested.SimulateNetworkGraph(suite.requestContext, request)
	suite.NoError(err, "expected graph generation to succeed")
	suite.Equal(expectedGraph, actualResp.GetSimulatedGraph(), "response should be output from graph generation")
	suite.Require().Len(actualResp.GetPolicies(), 2)
	suite.Equal("second-policy", actualResp.GetPolicies()[0].GetPolicy().GetName())
	suite.Equal(v1.NetworkPolicyInSimulation_MODIFIED, actualResp.GetPolicies()[0].GetStatus())
	suite.Equal("first-policy", actualResp.GetPolicies()[1].GetPolicy().GetName())
	suite.Equal(v1.NetworkPolicyInSimulation_ADDED, actualResp.GetPolicies()[1].GetStatus())
}

func (suite *ServiceTestSuite) TestGetNetworkGraphWithOnlyAdditions() {
	// Mock that cluster exists.
	suite.clusters.EXPECT().Exists(gomock.Any(), fakeClusterID).
		Return(true, nil)

	// Mock that we receive deployments for the cluster
	deps := make([]*storage.Deployment, 0)
	var networkTree tree.ReadOnlyNetworkTree
	suite.deployments.EXPECT().SearchRawDeployments(gomock.Any(), deploymentSearchIsForCluster(fakeClusterID)).
		Return(deps, nil)

	// Mock that we have network policies in effect for the cluster.
	suite.networkPolicies.EXPECT().GetNetworkPolicies(suite.requestContext, networkPolicyGetIsForCluster(fakeClusterID), "").
		Return(nil, nil)
	suite.graphConfig.EXPECT().GetNetworkGraphConfig(gomock.Any()).Return(&storage.NetworkGraphConfig{HideDefaultExternalSrcs: true}, nil)
	suite.netTreeMgr.EXPECT().GetReadOnlyNetworkTree(gomock.Any(), fakeClusterID).Return(nil)

	// Check that the evaluator gets called with our created deployment and policy set.
	expectedGraph := &v1.NetworkGraph{}
	suite.evaluator.EXPECT().GetGraph(fakeClusterID, set.NewStringSet(), deps, networkTree, checkHasPolicies("first-policy", "second-policy"), false).
		Return(expectedGraph)
	suite.evaluator.EXPECT().GetGraph(fakeClusterID, set.NewStringSet(), deps, networkTree, checkHasPolicies(), false).
		Return(expectedGraph)

	// Make the request to the service and check that it did not err.
	request := &v1.SimulateNetworkGraphRequest{
		ClusterId: fakeClusterID,
		Modification: &storage.NetworkPolicyModification{
			ApplyYaml: combinedYAMLs,
		},
		IncludeNodeDiff: true,
	}
	actualResp, err := suite.tested.SimulateNetworkGraph(suite.requestContext, request)
	suite.NoError(err, "expected graph generation to succeed")
	suite.Equal(expectedGraph, actualResp.GetSimulatedGraph(), "response should be output from graph generation")
	suite.Require().Len(actualResp.GetPolicies(), 2)
	suite.Equal("first-policy", actualResp.GetPolicies()[0].GetPolicy().GetName())
	suite.Equal(v1.NetworkPolicyInSimulation_ADDED, actualResp.GetPolicies()[0].GetStatus())
	suite.Equal("second-policy", actualResp.GetPolicies()[1].GetPolicy().GetName())
	suite.Equal(v1.NetworkPolicyInSimulation_ADDED, actualResp.GetPolicies()[1].GetStatus())
}

func (suite *ServiceTestSuite) TestGetNetworkPoliciesWithoutDeploymentQuery() {
	// Mock that cluster exists.
	suite.clusters.EXPECT().Exists(gomock.Any(), fakeClusterID).
		Return(true, nil)

	// Mock that we have network policies in effect for the cluster.
	neps := make([]*storage.NetworkPolicy, 0)
	suite.networkPolicies.EXPECT().GetNetworkPolicies(suite.requestContext, fakeClusterID, "").
		Return(neps, nil)

	// Make the request to the service and check that it did not err.
	request := &v1.GetNetworkPoliciesRequest{
		ClusterId: fakeClusterID,
	}
	actualResp, err := suite.tested.GetNetworkPolicies(suite.requestContext, request)

	suite.NoError(err, "expected graph generation to succeed")
	suite.Equal(neps, actualResp.GetNetworkPolicies(), "response should be policies read from store")
}

func (suite *ServiceTestSuite) TestGetNetworkPoliciesWitDeploymentQuery() {
	// Mock that cluster exists.
	suite.clusters.EXPECT().Exists(gomock.Any(), fakeClusterID).
		Return(true, nil)

	// Mock that we have network policies in effect for the cluster.
	neps := make([]*storage.NetworkPolicy, 0)
	suite.networkPolicies.EXPECT().GetNetworkPolicies(suite.requestContext, fakeClusterID, "").
		Return(neps, nil)

	// Mock that we receive deployments for the cluster
	deps := make([]*storage.Deployment, 0)
	var networkTree tree.ReadOnlyNetworkTree
	suite.deployments.EXPECT().SearchRawDeployments(gomock.Any(), testutils.PredMatcher("deployment search is for cluster", func(query *v1.Query) bool {
		// Should be a conjunction with cluster and deployment id.
		conj := query.GetConjunction()
		if len(conj.GetQueries()) != 2 {
			return false
		}
		matchCount := 0
		for _, query := range conj.GetQueries() {
			if queryIsForClusterID(query, fakeClusterID) || queryIsForDeploymentID(query, fakeDeploymentID) {
				matchCount = matchCount + 1
			}
		}
		return matchCount == 2
	})).Return(deps, nil)

	suite.graphConfig.EXPECT().GetNetworkGraphConfig(gomock.Any()).Return(&storage.NetworkGraphConfig{HideDefaultExternalSrcs: true}, nil)
	suite.netTreeMgr.EXPECT().GetReadOnlyNetworkTree(gomock.Any(), fakeClusterID).Return(nil)
	// Check that the evaluator gets called with our created deployment and policy set.
	expectedPolicies := make([]*storage.NetworkPolicy, 0)
	suite.evaluator.EXPECT().GetAppliedPolicies(deps, networkTree, neps).
		Return(expectedPolicies)

	// Make the request to the service and check that it did not err.
	request := &v1.GetNetworkPoliciesRequest{
		ClusterId:       fakeClusterID,
		DeploymentQuery: fmt.Sprintf("%s:\"%s\"", search.DeploymentID, fakeDeploymentID),
	}
	actualResp, err := suite.tested.GetNetworkPolicies(suite.requestContext, request)

	suite.NoError(err, "expected graph generation to succeed")
	suite.Equal(expectedPolicies, actualResp.GetNetworkPolicies(), "response should be policies applied to deployments")
}

func (suite *ServiceTestSuite) TestGetAllowedPeersFromCurrentPolicyForDeployment() {
	// NOTE: although the test verifies GetAllowedPeersFromCurrentPolicyForDeployment, most of the
	// dependency calls are mocked out. Thus those dependency calls' logics are not tested. This
	// only verifies the needed dependency calls are indeed getting called and also the execution logic
	// of the private functions used by GetAllowedPeersFromCurrentPolicyForDeployment.
	if !features.NetworkDetectionBaselineSimulation.Enabled() {
		return
	}
	// Prepare deployment001 - deployment004
	numDeployments := 4
	deps := make([]*storage.Deployment, 0, numDeployments)
	for i := 0; i < numDeployments; i++ {
		deps = append(deps, &storage.Deployment{
			Id:        fmt.Sprintf("deployment%03d", i),
			Name:      fmt.Sprintf("deployment%03d", i),
			Namespace: "namespace",
			ClusterId: fakeClusterID,
			PodLabels: map[string]string{"app": fmt.Sprintf("deployment%03d", i)},
		})
	}
	suite.deployments.EXPECT().SearchRawDeployments(
		gomock.Any(), deploymentSearchIsForCluster(fakeClusterID)).MinTimes(numDeployments).Return(deps, nil)

	var pols []*storage.NetworkPolicy
	suite.evaluator.EXPECT().GetAppliedPolicies(gomock.Any(), gomock.Any(), pols).MinTimes(numDeployments).Return(pols)
	suite.networkPolicies.EXPECT().GetNetworkPolicies(suite.requestContext, networkPolicyGetIsForCluster(fakeClusterID), "").MinTimes(numDeployments).Return(pols, nil)
	suite.graphConfig.EXPECT().GetNetworkGraphConfig(gomock.Any()).Return(&storage.NetworkGraphConfig{HideDefaultExternalSrcs: true}, nil).MinTimes(numDeployments)
	suite.netTreeMgr.EXPECT().GetReadOnlyNetworkTree(gomock.Any(), fakeClusterID).MinTimes(numDeployments).Return(nil)

	// Validate GetAllowedPeers
	for i, testCase := range []struct {
		expectedAllowedPeers []*v1.NetworkBaselineStatusPeer
	}{
		{
			// deployment000
			expectedAllowedPeers: []*v1.NetworkBaselineStatusPeer{
				{
					Entity: &v1.NetworkBaselinePeerEntity{
						Id:   deps[1].GetId(),
						Type: storage.NetworkEntityInfo_DEPLOYMENT,
					},
					Port:     80,
					Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
					Ingress:  true,
				},
				{
					Entity: &v1.NetworkBaselinePeerEntity{
						Id:   deps[2].GetId(),
						Type: storage.NetworkEntityInfo_DEPLOYMENT,
					},
					Port:     443,
					Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
					Ingress:  false,
				},
				{
					Entity: &v1.NetworkBaselinePeerEntity{
						Id:   deps[2].GetId(),
						Type: storage.NetworkEntityInfo_DEPLOYMENT,
					},
					Port:     80,
					Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
					Ingress:  false,
				},
			},
		},
		{
			// deployment001
			expectedAllowedPeers: []*v1.NetworkBaselineStatusPeer{
				{
					Entity: &v1.NetworkBaselinePeerEntity{
						Id:   deps[0].GetId(),
						Type: storage.NetworkEntityInfo_DEPLOYMENT,
					},
					Port:     80,
					Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
					Ingress:  false,
				},
				{
					Entity: &v1.NetworkBaselinePeerEntity{
						Id:   deps[2].GetId(),
						Type: storage.NetworkEntityInfo_DEPLOYMENT,
					},
					Port:     0,
					Protocol: storage.L4Protocol_L4_PROTOCOL_ANY,
					Ingress:  true,
				},
			},
		},
		{
			// deployment002
			expectedAllowedPeers: []*v1.NetworkBaselineStatusPeer{
				{
					Entity: &v1.NetworkBaselinePeerEntity{
						Id:   deps[0].GetId(),
						Type: storage.NetworkEntityInfo_DEPLOYMENT,
					},
					Port:     443,
					Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
					Ingress:  true,
				},
				{
					Entity: &v1.NetworkBaselinePeerEntity{
						Id:   deps[0].GetId(),
						Type: storage.NetworkEntityInfo_DEPLOYMENT,
					},
					Port:     80,
					Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
					Ingress:  true,
				},
				{
					Entity: &v1.NetworkBaselinePeerEntity{
						Id:   deps[1].GetId(),
						Type: storage.NetworkEntityInfo_DEPLOYMENT,
					},
					Port:     0,
					Protocol: storage.L4Protocol_L4_PROTOCOL_ANY,
					Ingress:  false,
				},
			},
		},
		{
			// deployment003
			expectedAllowedPeers: nil,
		},
	} {
		suite.Run(fmt.Sprintf("testing deployment%03d", i), func() {
			// Mark testing deployment node's query match to be true
			graph := suite.getSampleNetworkGraph(deps...)
			graph.Nodes[i].QueryMatch = true

			suite.evaluator.EXPECT().GetGraph(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(graph)
			suite.deployments.EXPECT().GetDeployment(gomock.Any(), gomock.Any()).Return(deps[i], true, nil)
			resp, err := suite.tested.GetAllowedPeersFromCurrentPolicyForDeployment(
				suite.requestContext,
				&v1.ResourceByID{Id: deps[0].GetId()})
			suite.NoError(err, "expected GetAllowedPeersFromCurrentPolicyForDeployment to succeed")

			suite.ElementsMatch(resp.GetAllowedPeers(), testCase.expectedAllowedPeers)
		})
	}
}

func (suite *ServiceTestSuite) TestGetUndoDeploymentRecord() {
	if !features.NetworkDetectionBaselineSimulation.Enabled() {
		return
	}
	suite.deployments.EXPECT().GetDeployment(gomock.Any(), "some-deployment").Return(
		&storage.Deployment{
			Id:        "some-deployment",
			Namespace: "some-namespace",
		},
		true,
		nil)
	suite.
		networkPolicies.
		EXPECT().
		GetUndoDeploymentRecord(gomock.Any(), "some-deployment").
		Return(
			&storage.NetworkPolicyApplicationUndoDeploymentRecord{
				DeploymentId: "some-deployment",
				UndoRecord:   &storage.NetworkPolicyApplicationUndoRecord{},
			},
			true,
			nil)
	resp, err :=
		suite.tested.GetUndoModificationForDeployment(suite.requestContext, &v1.ResourceByID{Id: "some-deployment"})
	suite.NoError(err)
	suite.Equal(
		&v1.GetUndoModificationForDeploymentResponse{UndoRecord: &storage.NetworkPolicyApplicationUndoRecord{}},
		resp)
}

func (suite *ServiceTestSuite) TestGetDiffFlows() {
	if !features.NetworkDetectionBaselineSimulation.Enabled() {
		return
	}
	// Prepare deployment001 - deployment004
	numDeployments := 4
	deps := make([]*storage.Deployment, 0, numDeployments)
	for i := 0; i < numDeployments; i++ {
		deps = append(deps, &storage.Deployment{
			Id:        fmt.Sprintf("deployment%03d", i),
			Name:      fmt.Sprintf("deployment%03d", i),
			Namespace: "namespace",
			ClusterId: fakeClusterID,
			PodLabels: map[string]string{"app": fmt.Sprintf("deployment%03d", i)},
		})
	}
	suite.deployments.EXPECT().GetDeployment(gomock.Any(), "deployment000").Return(deps[1], true, nil)
	suite.deployments.EXPECT().SearchRawDeployments(
		gomock.Any(), deploymentSearchIsForCluster(fakeClusterID)).Return(deps, nil)

	var pols []*storage.NetworkPolicy
	suite.evaluator.EXPECT().GetAppliedPolicies(gomock.Any(), gomock.Any(), pols).Return(pols)
	graph := suite.getSampleNetworkGraph(deps...)
	graph.Nodes[0].QueryMatch = true
	suite.evaluator.EXPECT().GetGraph(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(graph)
	suite.networkPolicies.EXPECT().GetNetworkPolicies(suite.requestContext, networkPolicyGetIsForCluster(fakeClusterID), "").Return(pols, nil)
	suite.graphConfig.EXPECT().GetNetworkGraphConfig(gomock.Any()).Return(&storage.NetworkGraphConfig{HideDefaultExternalSrcs: true}, nil)
	suite.netTreeMgr.EXPECT().GetReadOnlyNetworkTree(gomock.Any(), fakeClusterID).Return(nil)
	suite.networkBaselines.EXPECT().GetNetworkBaseline(gomock.Any(), "deployment000").Return(
		&storage.NetworkBaseline{
			DeploymentId: "deployment000",
			Peers: []*storage.NetworkBaselinePeer{
				{
					Entity: &storage.NetworkEntity{
						Info: &storage.NetworkEntityInfo{
							Type: storage.NetworkEntityInfo_DEPLOYMENT,
							Id:   "deployment002",
						},
					},
					Properties: []*storage.NetworkBaselineConnectionProperties{
						{
							Ingress:  false,
							Port:     443,
							Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
						},
						{
							Ingress:  true,
							Port:     22,
							Protocol: storage.L4Protocol_L4_PROTOCOL_UDP,
						},
					},
				},
				{
					Entity: &storage.NetworkEntity{
						Info: &storage.NetworkEntityInfo{
							Type: storage.NetworkEntityInfo_DEPLOYMENT,
							Id:   "deployment003",
						},
					},
					Properties: []*storage.NetworkBaselineConnectionProperties{
						{
							Ingress:  false,
							Port:     443,
							Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
						},
					},
				},
			},
		}, true, nil)
	rsp, err :=
		suite.tested.GetDiffFlowsBetweenPolicyAndBaselineForDeployment(suite.requestContext, &v1.ResourceByID{Id: "deployment000"})
	suite.NoError(err)
	suite.Equal(rsp, &v1.GetDiffFlowsResponse{
		Added: []*v1.GetDiffFlowsGroupedFlow{
			{
				Entity: &v1.NetworkBaselinePeerEntity{
					Id:   "deployment003",
					Type: storage.NetworkEntityInfo_DEPLOYMENT,
				},
				Properties: []*storage.NetworkBaselineConnectionProperties{
					{
						Ingress:  false,
						Port:     443,
						Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
					},
				},
			},
		},
		Removed: []*v1.GetDiffFlowsGroupedFlow{
			{
				Entity: &v1.NetworkBaselinePeerEntity{
					Id:   "deployment001",
					Type: storage.NetworkEntityInfo_DEPLOYMENT,
				},
				Properties: []*storage.NetworkBaselineConnectionProperties{
					{
						Ingress:  true,
						Port:     80,
						Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
					},
				},
			},
		},
		Reconciled: []*v1.GetDiffFlowsReconciledFlow{
			{
				Entity: &v1.NetworkBaselinePeerEntity{
					Id:   "deployment002",
					Type: storage.NetworkEntityInfo_DEPLOYMENT,
				},
				Added: []*storage.NetworkBaselineConnectionProperties{
					{
						Ingress:  true,
						Port:     22,
						Protocol: storage.L4Protocol_L4_PROTOCOL_UDP,
					},
				},
				Removed: []*storage.NetworkBaselineConnectionProperties{
					{
						Ingress:  false,
						Port:     80,
						Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
					},
				},
				Unchanged: []*storage.NetworkBaselineConnectionProperties{
					{
						Ingress:  false,
						Port:     443,
						Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
					},
				},
			},
		},
	})
}

// getSampleNetworkGraph requires at least 4 deployments
// This function configures a graph which has explicit edges like this:
//   - deployment001 -> deployment000 -> deployment002
// deployment003 is an "island" in this graph
// deployment001 has non-isolated ingress, and deployment002 has non-isolated egress. Thus
// there should be an implicit edge from deployment002 -> deployment001
func (suite *ServiceTestSuite) getSampleNetworkGraph(deps ...*storage.Deployment) *v1.NetworkGraph {
	suite.GreaterOrEqual(len(deps), 4)
	return &v1.NetworkGraph{
		Epoch: 0,
		Nodes: []*v1.NetworkNode{
			{
				Entity: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_DEPLOYMENT,
					Id:   deps[0].GetId(),
				},
				OutEdges: map[int32]*v1.NetworkEdgePropertiesBundle{
					2: {
						Properties: []*v1.NetworkEdgeProperties{
							{
								Port:     443,
								Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
							},
							{
								Port:     80,
								Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
							},
						},
					},
				},
			},
			{
				Entity: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_DEPLOYMENT,
					Id:   deps[1].GetId(),
				},
				OutEdges: map[int32]*v1.NetworkEdgePropertiesBundle{
					0: {
						Properties: []*v1.NetworkEdgeProperties{
							{
								Port:     80,
								Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
							},
						},
					},
				},
				NonIsolatedIngress: true,
			},
			{
				Entity: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_DEPLOYMENT,
					Id:   deps[2].GetId(),
				},
				NonIsolatedEgress: true,
			},
			{
				Entity: &storage.NetworkEntityInfo{
					Type: storage.NetworkEntityInfo_DEPLOYMENT,
					Id:   deps[3].GetId(),
				},
			},
		},
	}
}

// deploymentSearchIsForCluster returns a function that returns true if the in input ParsedSearchRequest has the
// ClusterID field set to the input clusterID.
func deploymentSearchIsForCluster(clusterID string) gomock.Matcher {
	return testutils.PredMatcher("deployment search is for cluster", func(query *v1.Query) bool {
		// Should be a single conjunction with a base string query inside.
		return query.GetBaseQuery().GetMatchFieldQuery().GetValue() == search.ExactMatchString(clusterID)
	})
}

// networkPolicyGetIsForCluster returns a function that returns true if the in input GetNetworkPolicyRequest has the
// ClusterID field set to the input clusterID.
func networkPolicyGetIsForCluster(expectedClusterID string) gomock.Matcher {
	return testutils.PredMatcher("network policy get is for cluster", func(actualClusterID string) bool {
		return actualClusterID == expectedClusterID
	})
}

func queryIsForClusterID(query *v1.Query, clusterID string) bool {
	if query.GetBaseQuery().GetMatchFieldQuery().GetField() != search.ClusterID.String() {
		return false
	}
	return query.GetBaseQuery().GetMatchFieldQuery().GetValue() == search.ExactMatchString(clusterID)
}

func queryIsForDeploymentID(query *v1.Query, deploymentID string) bool {
	if query.GetBaseQuery().GetMatchFieldQuery().GetField() != search.DeploymentID.String() {
		return false
	}
	return query.GetBaseQuery().GetMatchFieldQuery().GetValue() == search.ExactMatchString(deploymentID)
}

// checkHasPolicies returns a function that returns true if the input is a slice of network policies, containing
// exactly one policy for every input (policyNames).
func checkHasPolicies(policyNames ...string) gomock.Matcher {
	return testutils.PredMatcher("has policies", func(networkPolicies []*storage.NetworkPolicy) bool {
		if len(networkPolicies) != len(policyNames) {
			return false
		}
		for _, name := range policyNames {
			count := 0
			for _, policy := range networkPolicies {
				if policy.GetName() == name {
					count = count + 1
				}
			}
			if count != 1 {
				return false
			}
		}
		return true
	})
}
