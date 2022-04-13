package cache

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/stackrox/central/deployment/store/mocks"
	"github.com/stackrox/stackrox/central/deployment/store/types"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestDeploymentCache(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	baseStore := mocks.NewMockStore(mockCtrl)
	cacheStore := NewCachedStore(baseStore)

	dep1 := fixtures.GetDeployment()
	listDep1 := types.ConvertDeploymentToDeploymentList(dep1)

	dep2 := fixtures.GetDeployment()
	dep2.Id = "id2"
	listDep2 := types.ConvertDeploymentToDeploymentList(dep2)

	baseStore.EXPECT().ListDeployment(dep1.GetId()).Return(nil, false, nil)
	listDep, exists, err := cacheStore.ListDeployment(dep1.GetId())
	assert.NoError(t, err)
	assert.False(t, exists)
	assert.Nil(t, listDep)

	// Call Get and return dep1 as if it already exists in the store
	// This should fill the cache
	baseStore.EXPECT().GetDeployment(dep1.GetId()).Return(dep1, true, nil)
	dep, exists, err := cacheStore.GetDeployment(dep1.GetId())
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, dep1, dep)

	baseStore.EXPECT().UpsertDeployment(dep1).Return(nil)
	assert.NoError(t, cacheStore.UpsertDeployment(dep1))
	baseStore.EXPECT().UpsertDeployment(dep2).Return(nil)
	assert.NoError(t, cacheStore.UpsertDeployment(dep2))

	// Don't expect this to hit the underlying store
	dep, exists, err = cacheStore.GetDeployment(dep1.GetId())
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, dep1, dep)

	deployments, missing, err := cacheStore.GetDeploymentsWithIDs(dep1.GetId(), dep2.GetId())
	assert.NoError(t, err)
	assert.Empty(t, missing)
	assert.Equal(t, deployments, []*storage.Deployment{dep1, dep2})

	baseStore.EXPECT().GetDeployment("noid").Return(nil, false, nil)
	deployments, missing, err = cacheStore.GetDeploymentsWithIDs(dep1.GetId(), "noid", dep2.GetId())
	assert.NoError(t, err)
	assert.Equal(t, []int{1}, missing)
	assert.Equal(t, deployments, []*storage.Deployment{dep1, dep2})

	baseStore.EXPECT().ListDeployment("noid").Return(nil, false, nil)
	listDeployments, missing, err := cacheStore.ListDeploymentsWithIDs(dep1.GetId(), "noid", dep2.GetId())
	assert.NoError(t, err)
	assert.Equal(t, []int{1}, missing)
	assert.Equal(t, listDeployments, []*storage.ListDeployment{listDep1, listDep2})

	listImage, exists, err := cacheStore.ListDeployment(dep1.GetId())
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, types.ConvertDeploymentToDeploymentList(dep1), listImage)

	listImage, exists, err = cacheStore.ListDeployment(dep1.GetId())
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, types.ConvertDeploymentToDeploymentList(dep1), listImage)

	dep, exists, err = cacheStore.GetDeployment(dep2.GetId())
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, dep2, dep)

	baseStore.EXPECT().RemoveDeployment(dep1.GetId()).Return(nil)
	assert.NoError(t, cacheStore.RemoveDeployment(dep1.GetId()))

	// Expect the cache to be hit with a tombstone and the DB will not be hit
	dep, exists, err = cacheStore.GetDeployment(dep1.GetId())
	assert.NoError(t, err)
	assert.False(t, exists)
	assert.Nil(t, dep)

	// Test acknowledgements
	baseStore.EXPECT().AckKeysIndexed(dep1.GetId(), dep2.GetId()).Return(nil)
	assert.NoError(t, cacheStore.AckKeysIndexed(dep1.GetId(), dep2.GetId()))
}
