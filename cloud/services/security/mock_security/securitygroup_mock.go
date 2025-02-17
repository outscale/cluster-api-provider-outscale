// Code generated by MockGen. DO NOT EDIT.
// Source: ./securitygroup.go

// Package mock_security is a generated GoMock package.
package mock_security

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	osc "github.com/outscale/osc-sdk-go/v2"
)

// MockOscSecurityGroupInterface is a mock of OscSecurityGroupInterface interface.
type MockOscSecurityGroupInterface struct {
	ctrl     *gomock.Controller
	recorder *MockOscSecurityGroupInterfaceMockRecorder
}

// MockOscSecurityGroupInterfaceMockRecorder is the mock recorder for MockOscSecurityGroupInterface.
type MockOscSecurityGroupInterfaceMockRecorder struct {
	mock *MockOscSecurityGroupInterface
}

// NewMockOscSecurityGroupInterface creates a new mock instance.
func NewMockOscSecurityGroupInterface(ctrl *gomock.Controller) *MockOscSecurityGroupInterface {
	mock := &MockOscSecurityGroupInterface{ctrl: ctrl}
	mock.recorder = &MockOscSecurityGroupInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockOscSecurityGroupInterface) EXPECT() *MockOscSecurityGroupInterfaceMockRecorder {
	return m.recorder
}

// CreateSecurityGroup mocks base method.
func (m *MockOscSecurityGroupInterface) CreateSecurityGroup(ctx context.Context, netId, clusterName, securityGroupName, securityGroupDescription, securityGroupTag string) (*osc.SecurityGroup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateSecurityGroup", ctx, netId, clusterName, securityGroupName, securityGroupDescription, securityGroupTag)
	ret0, _ := ret[0].(*osc.SecurityGroup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateSecurityGroup indicates an expected call of CreateSecurityGroup.
func (mr *MockOscSecurityGroupInterfaceMockRecorder) CreateSecurityGroup(ctx, netId, clusterName, securityGroupName, securityGroupDescription, securityGroupTag interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateSecurityGroup", reflect.TypeOf((*MockOscSecurityGroupInterface)(nil).CreateSecurityGroup), ctx, netId, clusterName, securityGroupName, securityGroupDescription, securityGroupTag)
}

// CreateSecurityGroupRule mocks base method.
func (m *MockOscSecurityGroupInterface) CreateSecurityGroupRule(ctx context.Context, securityGroupId, flow, ipProtocol, ipRange, securityGroupMemberId string, fromPortRange, toPortRange int32) (*osc.SecurityGroup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateSecurityGroupRule", ctx, securityGroupId, flow, ipProtocol, ipRange, securityGroupMemberId, fromPortRange, toPortRange)
	ret0, _ := ret[0].(*osc.SecurityGroup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateSecurityGroupRule indicates an expected call of CreateSecurityGroupRule.
func (mr *MockOscSecurityGroupInterfaceMockRecorder) CreateSecurityGroupRule(ctx, securityGroupId, flow, ipProtocol, ipRange, securityGroupMemberId, fromPortRange, toPortRange interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateSecurityGroupRule", reflect.TypeOf((*MockOscSecurityGroupInterface)(nil).CreateSecurityGroupRule), ctx, securityGroupId, flow, ipProtocol, ipRange, securityGroupMemberId, fromPortRange, toPortRange)
}

// DeleteSecurityGroup mocks base method.
func (m *MockOscSecurityGroupInterface) DeleteSecurityGroup(ctx context.Context, securityGroupId string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteSecurityGroup", ctx, securityGroupId)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteSecurityGroup indicates an expected call of DeleteSecurityGroup.
func (mr *MockOscSecurityGroupInterfaceMockRecorder) DeleteSecurityGroup(ctx, securityGroupId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteSecurityGroup", reflect.TypeOf((*MockOscSecurityGroupInterface)(nil).DeleteSecurityGroup), ctx, securityGroupId)
}

// DeleteSecurityGroupRule mocks base method.
func (m *MockOscSecurityGroupInterface) DeleteSecurityGroupRule(ctx context.Context, securityGroupId, flow, ipProtocol, ipRange, securityGroupMemberId string, fromPortRange, toPortRange int32) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteSecurityGroupRule", ctx, securityGroupId, flow, ipProtocol, ipRange, securityGroupMemberId, fromPortRange, toPortRange)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteSecurityGroupRule indicates an expected call of DeleteSecurityGroupRule.
func (mr *MockOscSecurityGroupInterfaceMockRecorder) DeleteSecurityGroupRule(ctx, securityGroupId, flow, ipProtocol, ipRange, securityGroupMemberId, fromPortRange, toPortRange interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteSecurityGroupRule", reflect.TypeOf((*MockOscSecurityGroupInterface)(nil).DeleteSecurityGroupRule), ctx, securityGroupId, flow, ipProtocol, ipRange, securityGroupMemberId, fromPortRange, toPortRange)
}

// GetSecurityGroup mocks base method.
func (m *MockOscSecurityGroupInterface) GetSecurityGroup(ctx context.Context, securityGroupId string) (*osc.SecurityGroup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSecurityGroup", ctx, securityGroupId)
	ret0, _ := ret[0].(*osc.SecurityGroup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSecurityGroup indicates an expected call of GetSecurityGroup.
func (mr *MockOscSecurityGroupInterfaceMockRecorder) GetSecurityGroup(ctx, securityGroupId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSecurityGroup", reflect.TypeOf((*MockOscSecurityGroupInterface)(nil).GetSecurityGroup), ctx, securityGroupId)
}

// GetSecurityGroupFromSecurityGroupRule mocks base method.
func (m *MockOscSecurityGroupInterface) GetSecurityGroupFromSecurityGroupRule(ctx context.Context, securityGroupId, Flow, IpProtocols, IpRanges, securityGroupMemberId string, FromPortRanges, ToPortRanges int32) (*osc.SecurityGroup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSecurityGroupFromSecurityGroupRule", ctx, securityGroupId, Flow, IpProtocols, IpRanges, securityGroupMemberId, FromPortRanges, ToPortRanges)
	ret0, _ := ret[0].(*osc.SecurityGroup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSecurityGroupFromSecurityGroupRule indicates an expected call of GetSecurityGroupFromSecurityGroupRule.
func (mr *MockOscSecurityGroupInterfaceMockRecorder) GetSecurityGroupFromSecurityGroupRule(ctx, securityGroupId, Flow, IpProtocols, IpRanges, securityGroupMemberId, FromPortRanges, ToPortRanges interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSecurityGroupFromSecurityGroupRule", reflect.TypeOf((*MockOscSecurityGroupInterface)(nil).GetSecurityGroupFromSecurityGroupRule), ctx, securityGroupId, Flow, IpProtocols, IpRanges, securityGroupMemberId, FromPortRanges, ToPortRanges)
}

// GetSecurityGroupIdsFromNetIds mocks base method.
func (m *MockOscSecurityGroupInterface) GetSecurityGroupIdsFromNetIds(ctx context.Context, netId string) ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSecurityGroupIdsFromNetIds", ctx, netId)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSecurityGroupIdsFromNetIds indicates an expected call of GetSecurityGroupIdsFromNetIds.
func (mr *MockOscSecurityGroupInterfaceMockRecorder) GetSecurityGroupIdsFromNetIds(ctx, netId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSecurityGroupIdsFromNetIds", reflect.TypeOf((*MockOscSecurityGroupInterface)(nil).GetSecurityGroupIdsFromNetIds), ctx, netId)
}
