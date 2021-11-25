// Code generated by MockGen. DO NOT EDIT.
// Source: messages.go

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	sql "database/sql"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockSQLService is a mock of SQLService interface.
type MockSQLService struct {
	ctrl     *gomock.Controller
	recorder *MockSQLServiceMockRecorder
}

// MockSQLServiceMockRecorder is the mock recorder for MockSQLService.
type MockSQLServiceMockRecorder struct {
	mock *MockSQLService
}

// NewMockSQLService creates a new mock instance.
func NewMockSQLService(ctrl *gomock.Controller) *MockSQLService {
	mock := &MockSQLService{ctrl: ctrl}
	mock.recorder = &MockSQLServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSQLService) EXPECT() *MockSQLServiceMockRecorder {
	return m.recorder
}

// ExecContext mocks base method.
func (m *MockSQLService) ExecContext(ctx context.Context, q string) (sql.Result, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ExecContext", ctx, q)
	ret0, _ := ret[0].(sql.Result)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ExecContext indicates an expected call of ExecContext.
func (mr *MockSQLServiceMockRecorder) ExecContext(ctx, q interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ExecContext", reflect.TypeOf((*MockSQLService)(nil).ExecContext), ctx, q)
}
