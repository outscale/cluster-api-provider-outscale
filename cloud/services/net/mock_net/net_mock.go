// Code generated by MockGen. DO NOT EDIT.
// Source: ./net.go

// Package mock_net is a generated GoMock package.
package mock_net

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	v1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	osc "github.com/outscale/osc-sdk-go/v2"
)

// MockOscNetInterface is a mock of OscNetInterface interface.
type MockOscNetInterface struct {
	ctrl     *gomock.Controller
	recorder *MockOscNetInterfaceMockRecorder
}

// MockOscNetInterfaceMockRecorder is the mock recorder for MockOscNetInterface.
type MockOscNetInterfaceMockRecorder struct {
	mock *MockOscNetInterface
}

// NewMockOscNetInterface creates a new mock instance.
func NewMockOscNetInterface(ctrl *gomock.Controller) *MockOscNetInterface {
	mock := &MockOscNetInterface{ctrl: ctrl}
	mock.recorder = &MockOscNetInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockOscNetInterface) EXPECT() *MockOscNetInterfaceMockRecorder {
	return m.recorder
}

// CreateNet mocks base method.
func (m *MockOscNetInterface) CreateNet(spec *v1beta1.OscNet, clusterName, netName string) (*osc.Net, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateNet", spec, clusterName, netName)
	ret0, _ := ret[0].(*osc.Net)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateNet indicates an expected call of CreateNet.
func (mr *MockOscNetInterfaceMockRecorder) CreateNet(spec, clusterName, netName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateNet", reflect.TypeOf((*MockOscNetInterface)(nil).CreateNet), spec, clusterName, netName)
}

// DeleteNet mocks base method.
func (m *MockOscNetInterface) DeleteNet(netId string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteNet", netId)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteNet indicates an expected call of DeleteNet.
func (mr *MockOscNetInterfaceMockRecorder) DeleteNet(netId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteNet", reflect.TypeOf((*MockOscNetInterface)(nil).DeleteNet), netId)
}

// GetNet mocks base method.
func (m *MockOscNetInterface) GetNet(netId string) (*osc.Net, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNet", netId)
	ret0, _ := ret[0].(*osc.Net)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetNet indicates an expected call of GetNet.
func (mr *MockOscNetInterfaceMockRecorder) GetNet(netId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNet", reflect.TypeOf((*MockOscNetInterface)(nil).GetNet), netId)
}
