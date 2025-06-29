// Code generated by MockGen. DO NOT EDIT.
// Source: ./internal/store/codebase/codebase_store.go

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	io "io"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	codebase "github.com/zgsm-ai/codebase-indexer/internal/store/codebase"
	types "github.com/zgsm-ai/codebase-indexer/internal/types"
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

// Add mocks base method.
func (m *MockStore) Add(ctx context.Context, codebasePath string, source io.Reader, target string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Add", ctx, codebasePath, source, target)
	ret0, _ := ret[0].(error)
	return ret0
}

// Add indicates an expected call of Add.
func (mr *MockStoreMockRecorder) Add(ctx, codebasePath, source, target interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Add", reflect.TypeOf((*MockStore)(nil).Add), ctx, codebasePath, source, target)
}

// BatchDelete mocks base method.
func (m *MockStore) BatchDelete(ctx context.Context, codebasePath string, paths []string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BatchDelete", ctx, codebasePath, paths)
	ret0, _ := ret[0].(error)
	return ret0
}

// BatchDelete indicates an expected call of BatchDelete.
func (mr *MockStoreMockRecorder) BatchDelete(ctx, codebasePath, paths interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BatchDelete", reflect.TypeOf((*MockStore)(nil).BatchDelete), ctx, codebasePath, paths)
}

// Delete mocks base method.
func (m *MockStore) Delete(ctx context.Context, codebasePath, path string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteByCodebase", ctx, codebasePath, path)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockStoreMockRecorder) Delete(ctx, codebasePath, path interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteByCodebase", reflect.TypeOf((*MockStore)(nil).Delete), ctx, codebasePath, path)
}

// DeleteAll mocks base method.
func (m *MockStore) DeleteAll(ctx context.Context, codebasePath string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteAll", ctx, codebasePath)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteAll indicates an expected call of DeleteAll.
func (mr *MockStoreMockRecorder) DeleteAll(ctx, codebasePath interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteAll", reflect.TypeOf((*MockStore)(nil).DeleteAll), ctx, codebasePath)
}

// Exists mocks base method.
func (m *MockStore) Exists(ctx context.Context, codebasePath, path string) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Exists", ctx, codebasePath, path)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Exists indicates an expected call of Exists.
func (mr *MockStoreMockRecorder) Exists(ctx, codebasePath, path interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Exists", reflect.TypeOf((*MockStore)(nil).Exists), ctx, codebasePath, path)
}

// GetSyncFileListCollapse mocks base method.
func (m *MockStore) GetSyncFileListCollapse(ctx context.Context, codebasePath string) (map[string]string, []string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSyncFileListCollapse", ctx, codebasePath)
	ret0, _ := ret[0].(map[string]string)
	ret1, _ := ret[1].([]string)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// GetSyncFileListCollapse indicates an expected call of GetSyncFileListCollapse.
func (mr *MockStoreMockRecorder) GetSyncFileListCollapse(ctx, codebasePath interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSyncFileListCollapse", reflect.TypeOf((*MockStore)(nil).GetSyncFileListCollapse), ctx, codebasePath)
}

// Init mocks base method.
func (m *MockStore) Init(ctx context.Context, clientId, clientPath string) (*types.Codebase, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Init", ctx, clientId, clientPath)
	ret0, _ := ret[0].(*types.Codebase)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Init indicates an expected call of Init.
func (mr *MockStoreMockRecorder) Init(ctx, clientId, clientPath interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Init", reflect.TypeOf((*MockStore)(nil).Init), ctx, clientId, clientPath)
}

// List mocks base method.
func (m *MockStore) List(ctx context.Context, codebasePath, dir string, option types.ListOptions) ([]*types.FileInfo, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", ctx, codebasePath, dir, option)
	ret0, _ := ret[0].([]*types.FileInfo)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockStoreMockRecorder) List(ctx, codebasePath, dir, option interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockStore)(nil).List), ctx, codebasePath, dir, option)
}

// MkDirs mocks base method.
func (m *MockStore) MkDirs(ctx context.Context, codebasePath, path string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MkDirs", ctx, codebasePath, path)
	ret0, _ := ret[0].(error)
	return ret0
}

// MkDirs indicates an expected call of MkDirs.
func (mr *MockStoreMockRecorder) MkDirs(ctx, codebasePath, path interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MkDirs", reflect.TypeOf((*MockStore)(nil).MkDirs), ctx, codebasePath, path)
}

// Open mocks base method.
func (m *MockStore) Open(ctx context.Context, codebasePath, filePath string) (io.ReadSeekCloser, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Open", ctx, codebasePath, filePath)
	ret0, _ := ret[0].(io.ReadSeekCloser)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Open indicates an expected call of Open.
func (mr *MockStoreMockRecorder) Open(ctx, codebasePath, filePath interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Open", reflect.TypeOf((*MockStore)(nil).Open), ctx, codebasePath, filePath)
}

// Read mocks base method.
func (m *MockStore) Read(ctx context.Context, codebasePath, filePath string, option types.ReadOptions) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Read", ctx, codebasePath, filePath, option)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Read indicates an expected call of Read.
func (mr *MockStoreMockRecorder) Read(ctx, codebasePath, filePath, option interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Read", reflect.TypeOf((*MockStore)(nil).Read), ctx, codebasePath, filePath, option)
}

// Stat mocks base method.
func (m *MockStore) Stat(ctx context.Context, codebasePath, path string) (*types.FileInfo, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Stat", ctx, codebasePath, path)
	ret0, _ := ret[0].(*types.FileInfo)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Stat indicates an expected call of Stat.
func (mr *MockStoreMockRecorder) Stat(ctx, codebasePath, path interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stat", reflect.TypeOf((*MockStore)(nil).Stat), ctx, codebasePath, path)
}

// Tree mocks base method.
func (m *MockStore) Tree(ctx context.Context, codebasePath, dir string, option types.TreeOptions) ([]*types.TreeNode, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Tree", ctx, codebasePath, dir, option)
	ret0, _ := ret[0].([]*types.TreeNode)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Tree indicates an expected call of Tree.
func (mr *MockStoreMockRecorder) Tree(ctx, codebasePath, dir, option interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Tree", reflect.TypeOf((*MockStore)(nil).Tree), ctx, codebasePath, dir, option)
}

// Unzip mocks base method.
func (m *MockStore) Unzip(ctx context.Context, codebasePath string, source io.Reader) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Unzip", ctx, codebasePath, source)
	ret0, _ := ret[0].(error)
	return ret0
}

// Unzip indicates an expected call of Unzip.
func (mr *MockStoreMockRecorder) Unzip(ctx, codebasePath, source interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Unzip", reflect.TypeOf((*MockStore)(nil).Unzip), ctx, codebasePath, source)
}

// Walk mocks base method.
func (m *MockStore) Walk(ctx context.Context, codebasePath, dir string, walkFn codebase.WalkFunc, walkOpts codebase.WalkOptions) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Walk", ctx, codebasePath, dir, walkFn, walkOpts)
	ret0, _ := ret[0].(error)
	return ret0
}

// Walk indicates an expected call of Walk.
func (mr *MockStoreMockRecorder) Walk(ctx, codebasePath, dir, walkFn, walkOpts interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Walk", reflect.TypeOf((*MockStore)(nil).Walk), ctx, codebasePath, dir, walkFn, walkOpts)
}
