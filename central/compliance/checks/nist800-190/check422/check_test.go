package check422

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/stackrox/central/compliance/framework"
	complianceMocks "github.com/stackrox/stackrox/central/compliance/framework/mocks"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/booleanpolicy/policyversion"
	"github.com/stackrox/stackrox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testCluster = &storage.Cluster{
		Id: uuid.NewV4().String(),
	}

	testDeployments = []*storage.Deployment{
		{
			Id: uuid.NewV4().String(),
		},
		{
			Id: uuid.NewV4().String(),
		},
	}

	testNodes = []*storage.Node{
		{
			Id: uuid.NewV4().String(),
		},
		{
			Id: uuid.NewV4().String(),
		},
	}

	domain = framework.NewComplianceDomain(testCluster, testNodes, testDeployments, nil, nil)

	latestTagEnabledAndEnforced = &storage.Policy{
		Id:   uuid.NewV4().String(),
		Name: "Foo",
		Fields: &storage.PolicyFields{
			ImageName: &storage.ImageNamePolicy{
				Tag: "latest",
			},
		},
		Disabled:           false,
		EnforcementActions: []storage.EnforcementAction{storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT},
	}

	imageAgePolicyEnabledAndEnforced = &storage.Policy{
		Id:   uuid.NewV4().String(),
		Name: "Bar",
		Fields: &storage.PolicyFields{
			SetImageAgeDays: &storage.PolicyFields_ImageAgeDays{
				ImageAgeDays: 30,
			},
		},
		Disabled:           false,
		EnforcementActions: []storage.EnforcementAction{storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT},
	}

	randomPolicy = &storage.Policy{
		Id:       uuid.NewV4().String(),
		Name:     "Random",
		Disabled: false,
		Fields: &storage.PolicyFields{
			ProcessPolicy: &storage.ProcessPolicy{
				Name: "sshd",
			},
		},
		EnforcementActions: []storage.EnforcementAction{storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT},
	}
)

func getPolicies(t *testing.T, policies ...*storage.Policy) map[string]*storage.Policy {
	m := make(map[string]*storage.Policy, len(policies))
	for _, p := range policies {
		require.NoError(t, policyversion.EnsureConvertedToLatest(p))
		m[p.GetName()] = p
	}
	return m
}

func TestNIST422_Success(t *testing.T) {
	registry := framework.RegistrySingleton()
	check := registry.Lookup(standardID)
	require.NotNil(t, check)

	policies := getPolicies(t, latestTagEnabledAndEnforced, imageAgePolicyEnabledAndEnforced)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	data := complianceMocks.NewMockComplianceDataRepository(mockCtrl)
	data.EXPECT().Cluster().AnyTimes().Return(testCluster)
	data.EXPECT().Policies().AnyTimes().Return(policies)

	run, err := framework.NewComplianceRun(check)
	require.NoError(t, err)
	err = run.Run(context.Background(), "standard", domain, data)
	require.NoError(t, err)

	results := run.GetAllResults()
	checkResults := results[standardID]
	require.NotNil(t, checkResults)

	require.Len(t, checkResults.Evidence(), 2)
	assert.Equal(t, framework.PassStatus, checkResults.Evidence()[0].Status)
	assert.Equal(t, framework.PassStatus, checkResults.Evidence()[1].Status)
}

func TestNIST422_Fail(t *testing.T) {
	registry := framework.RegistrySingleton()
	check := registry.Lookup(standardID)
	require.NotNil(t, check)

	policies := getPolicies(t, randomPolicy)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	data := complianceMocks.NewMockComplianceDataRepository(mockCtrl)
	data.EXPECT().Cluster().AnyTimes().Return(testCluster)
	data.EXPECT().Policies().AnyTimes().Return(policies)

	run, err := framework.NewComplianceRun(check)
	require.NoError(t, err)
	err = run.Run(context.Background(), "standard", domain, data)
	require.NoError(t, err)

	results := run.GetAllResults()
	checkResults := results[standardID]
	require.NotNil(t, checkResults)

	require.Len(t, checkResults.Evidence(), 2)
	assert.Equal(t, framework.FailStatus, checkResults.Evidence()[0].Status)
	assert.Equal(t, framework.FailStatus, checkResults.Evidence()[1].Status)
}
