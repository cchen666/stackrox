// Code generated by MockGen. DO NOT EDIT.
// Source: cluster.go

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	store "github.com/stackrox/rox/central/networkgraph/flow/datastore/internal/store"
)

// MockClusterStore is a mock of ClusterStore interface.
type MockClusterStore struct {
	ctrl     *gomock.Controller
	recorder *MockClusterStoreMockRecorder
}

// MockClusterStoreMockRecorder is the mock recorder for MockClusterStore.
type MockClusterStoreMockRecorder struct {
	mock *MockClusterStore
}

// NewMockClusterStore creates a new mock instance.
func NewMockClusterStore(ctrl *gomock.Controller) *MockClusterStore {
	mock := &MockClusterStore{ctrl: ctrl}
	mock.recorder = &MockClusterStoreMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockClusterStore) EXPECT() *MockClusterStoreMockRecorder {
	return m.recorder
}

// GetFlowStore mocks base method.
func (m *MockClusterStore) GetFlowStore(clusterID string) store.FlowStore {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFlowStore", clusterID)
	ret0, _ := ret[0].(store.FlowStore)
	return ret0
}

// GetFlowStore indicates an expected call of GetFlowStore.
func (mr *MockClusterStoreMockRecorder) GetFlowStore(clusterID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFlowStore", reflect.TypeOf((*MockClusterStore)(nil).GetFlowStore), clusterID)
}
