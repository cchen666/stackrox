// Code generated by MockGen. DO NOT EDIT.
// Source: store.go

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	v1 "github.com/stackrox/rox/generated/api/v1"
	storage "github.com/stackrox/rox/generated/storage"
)

// MockStore is a mock of Store interface.
type MockStore struct {
	ctrl     *gomock.Controller
	recorder *MockStoreMockRecorder
}

// MockStoreMockRecorder is the mock recorder for MockStore.
type MockStoreMockRecorder struct {
	mock *MockStore
}

// NewMockStore creates a new mock instance.
func NewMockStore(ctrl *gomock.Controller) *MockStore {
	mock := &MockStore{ctrl: ctrl}
	mock.recorder = &MockStoreMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStore) EXPECT() *MockStoreMockRecorder {
	return m.recorder
}

// AckKeysIndexed mocks base method.
func (m *MockStore) AckKeysIndexed(ctx context.Context, keys ...string) error {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx}
	for _, a := range keys {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "AckKeysIndexed", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// AckKeysIndexed indicates an expected call of AckKeysIndexed.
func (mr *MockStoreMockRecorder) AckKeysIndexed(ctx interface{}, keys ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx}, keys...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AckKeysIndexed", reflect.TypeOf((*MockStore)(nil).AckKeysIndexed), varargs...)
}

// Delete mocks base method.
func (m *MockStore) Delete(ctx context.Context, id string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockStoreMockRecorder) Delete(ctx, id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockStore)(nil).Delete), ctx, id)
}

// DeleteMany mocks base method.
func (m *MockStore) DeleteMany(ctx context.Context, ids []string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteMany", ctx, ids)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteMany indicates an expected call of DeleteMany.
func (mr *MockStoreMockRecorder) DeleteMany(ctx, ids interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteMany", reflect.TypeOf((*MockStore)(nil).DeleteMany), ctx, ids)
}

// DeletePolicyCategory mocks base method.
func (m *MockStore) DeletePolicyCategory(request *v1.DeletePolicyCategoryRequest) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeletePolicyCategory", request)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeletePolicyCategory indicates an expected call of DeletePolicyCategory.
func (mr *MockStoreMockRecorder) DeletePolicyCategory(request interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeletePolicyCategory", reflect.TypeOf((*MockStore)(nil).DeletePolicyCategory), request)
}

// Get mocks base method.
func (m *MockStore) Get(ctx context.Context, id string) (*storage.Policy, bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", ctx, id)
	ret0, _ := ret[0].(*storage.Policy)
	ret1, _ := ret[1].(bool)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// Get indicates an expected call of Get.
func (mr *MockStoreMockRecorder) Get(ctx, id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockStore)(nil).Get), ctx, id)
}

// GetAll mocks base method.
func (m *MockStore) GetAll(ctx context.Context) ([]*storage.Policy, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAll", ctx)
	ret0, _ := ret[0].([]*storage.Policy)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAll indicates an expected call of GetAll.
func (mr *MockStoreMockRecorder) GetAll(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAll", reflect.TypeOf((*MockStore)(nil).GetAll), ctx)
}

// GetIDs mocks base method.
func (m *MockStore) GetIDs(ctx context.Context) ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetIDs", ctx)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetIDs indicates an expected call of GetIDs.
func (mr *MockStoreMockRecorder) GetIDs(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetIDs", reflect.TypeOf((*MockStore)(nil).GetIDs), ctx)
}

// GetKeysToIndex mocks base method.
func (m *MockStore) GetKeysToIndex(ctx context.Context) ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetKeysToIndex", ctx)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetKeysToIndex indicates an expected call of GetKeysToIndex.
func (mr *MockStoreMockRecorder) GetKeysToIndex(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetKeysToIndex", reflect.TypeOf((*MockStore)(nil).GetKeysToIndex), ctx)
}

// GetMany mocks base method.
func (m *MockStore) GetMany(ctx context.Context, ids ...string) ([]*storage.Policy, []int, []error, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx}
	for _, a := range ids {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "GetMany", varargs...)
	ret0, _ := ret[0].([]*storage.Policy)
	ret1, _ := ret[1].([]int)
	ret2, _ := ret[2].([]error)
	ret3, _ := ret[3].(error)
	return ret0, ret1, ret2, ret3
}

// GetMany indicates an expected call of GetMany.
func (mr *MockStoreMockRecorder) GetMany(ctx interface{}, ids ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx}, ids...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMany", reflect.TypeOf((*MockStore)(nil).GetMany), varargs...)
}

// RenamePolicyCategory mocks base method.
func (m *MockStore) RenamePolicyCategory(request *v1.RenamePolicyCategoryRequest) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RenamePolicyCategory", request)
	ret0, _ := ret[0].(error)
	return ret0
}

// RenamePolicyCategory indicates an expected call of RenamePolicyCategory.
func (mr *MockStoreMockRecorder) RenamePolicyCategory(request interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RenamePolicyCategory", reflect.TypeOf((*MockStore)(nil).RenamePolicyCategory), request)
}

// Upsert mocks base method.
func (m *MockStore) Upsert(ctx context.Context, policy *storage.Policy) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Upsert", ctx, policy)
	ret0, _ := ret[0].(error)
	return ret0
}

// Upsert indicates an expected call of Upsert.
func (mr *MockStoreMockRecorder) Upsert(ctx, policy interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Upsert", reflect.TypeOf((*MockStore)(nil).Upsert), ctx, policy)
}

// UpsertMany mocks base method.
func (m *MockStore) UpsertMany(ctx context.Context, objs []*storage.Policy) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpsertMany", ctx, objs)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpsertMany indicates an expected call of UpsertMany.
func (mr *MockStoreMockRecorder) UpsertMany(ctx, objs interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpsertMany", reflect.TypeOf((*MockStore)(nil).UpsertMany), ctx, objs)
}
