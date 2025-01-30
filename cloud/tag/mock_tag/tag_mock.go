// Code generated by MockGen. DO NOT EDIT.
// Source: ./tag.go

// Package mock_tag is a generated GoMock package.
package mock_tag

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	osc "github.com/outscale/osc-sdk-go/v2"
)

// MockOscTagInterface is a mock of OscTagInterface interface.
type MockOscTagInterface struct {
	ctrl     *gomock.Controller
	recorder *MockOscTagInterfaceMockRecorder
}

// MockOscTagInterfaceMockRecorder is the mock recorder for MockOscTagInterface.
type MockOscTagInterfaceMockRecorder struct {
	mock *MockOscTagInterface
}

// NewMockOscTagInterface creates a new mock instance.
func NewMockOscTagInterface(ctrl *gomock.Controller) *MockOscTagInterface {
	mock := &MockOscTagInterface{ctrl: ctrl}
	mock.recorder = &MockOscTagInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockOscTagInterface) EXPECT() *MockOscTagInterfaceMockRecorder {
	return m.recorder
}

// ReadTag mocks base method.
func (m *MockOscTagInterface) ReadTag(ctx context.Context, tagKey, tagValue string) (*osc.Tag, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReadTag", ctx, tagKey, tagValue)
	ret0, _ := ret[0].(*osc.Tag)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ReadTag indicates an expected call of ReadTag.
func (mr *MockOscTagInterfaceMockRecorder) ReadTag(ctx, tagKey, tagValue interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadTag", reflect.TypeOf((*MockOscTagInterface)(nil).ReadTag), ctx, tagKey, tagValue)
}
