// Code generated by MockGen. DO NOT EDIT.
// Source: ./publicip.go
//
// Generated by this command:
//
//	mockgen -destination mock_security/publicip_mock.go -package mock_security -source ./publicip.go
//

// Package mock_security is a generated GoMock package.
package mock_security

import (
	context "context"
	reflect "reflect"
	time "time"

	osc "github.com/outscale/osc-sdk-go/v2"
	gomock "go.uber.org/mock/gomock"
)

// MockOscPublicIpInterface is a mock of OscPublicIpInterface interface.
type MockOscPublicIpInterface struct {
	ctrl     *gomock.Controller
	recorder *MockOscPublicIpInterfaceMockRecorder
	isgomock struct{}
}

// MockOscPublicIpInterfaceMockRecorder is the mock recorder for MockOscPublicIpInterface.
type MockOscPublicIpInterfaceMockRecorder struct {
	mock *MockOscPublicIpInterface
}

// NewMockOscPublicIpInterface creates a new mock instance.
func NewMockOscPublicIpInterface(ctrl *gomock.Controller) *MockOscPublicIpInterface {
	mock := &MockOscPublicIpInterface{ctrl: ctrl}
	mock.recorder = &MockOscPublicIpInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockOscPublicIpInterface) EXPECT() *MockOscPublicIpInterfaceMockRecorder {
	return m.recorder
}

// CheckPublicIpUnlink mocks base method.
func (m *MockOscPublicIpInterface) CheckPublicIpUnlink(ctx context.Context, clockInsideLoop, clockLoop time.Duration, publicIpId string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CheckPublicIpUnlink", ctx, clockInsideLoop, clockLoop, publicIpId)
	ret0, _ := ret[0].(error)
	return ret0
}

// CheckPublicIpUnlink indicates an expected call of CheckPublicIpUnlink.
func (mr *MockOscPublicIpInterfaceMockRecorder) CheckPublicIpUnlink(ctx, clockInsideLoop, clockLoop, publicIpId any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CheckPublicIpUnlink", reflect.TypeOf((*MockOscPublicIpInterface)(nil).CheckPublicIpUnlink), ctx, clockInsideLoop, clockLoop, publicIpId)
}

// CreatePublicIp mocks base method.
func (m *MockOscPublicIpInterface) CreatePublicIp(ctx context.Context, publicIpName string) (*osc.PublicIp, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreatePublicIp", ctx, publicIpName)
	ret0, _ := ret[0].(*osc.PublicIp)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreatePublicIp indicates an expected call of CreatePublicIp.
func (mr *MockOscPublicIpInterfaceMockRecorder) CreatePublicIp(ctx, publicIpName any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreatePublicIp", reflect.TypeOf((*MockOscPublicIpInterface)(nil).CreatePublicIp), ctx, publicIpName)
}

// DeletePublicIp mocks base method.
func (m *MockOscPublicIpInterface) DeletePublicIp(ctx context.Context, publicIpId string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeletePublicIp", ctx, publicIpId)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeletePublicIp indicates an expected call of DeletePublicIp.
func (mr *MockOscPublicIpInterfaceMockRecorder) DeletePublicIp(ctx, publicIpId any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeletePublicIp", reflect.TypeOf((*MockOscPublicIpInterface)(nil).DeletePublicIp), ctx, publicIpId)
}

// GetPublicIp mocks base method.
func (m *MockOscPublicIpInterface) GetPublicIp(ctx context.Context, publicIpId string) (*osc.PublicIp, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPublicIp", ctx, publicIpId)
	ret0, _ := ret[0].(*osc.PublicIp)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetPublicIp indicates an expected call of GetPublicIp.
func (mr *MockOscPublicIpInterfaceMockRecorder) GetPublicIp(ctx, publicIpId any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPublicIp", reflect.TypeOf((*MockOscPublicIpInterface)(nil).GetPublicIp), ctx, publicIpId)
}

// LinkPublicIp mocks base method.
func (m *MockOscPublicIpInterface) LinkPublicIp(ctx context.Context, publicIpId, vmId string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LinkPublicIp", ctx, publicIpId, vmId)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// LinkPublicIp indicates an expected call of LinkPublicIp.
func (mr *MockOscPublicIpInterfaceMockRecorder) LinkPublicIp(ctx, publicIpId, vmId any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LinkPublicIp", reflect.TypeOf((*MockOscPublicIpInterface)(nil).LinkPublicIp), ctx, publicIpId, vmId)
}

// UnlinkPublicIp mocks base method.
func (m *MockOscPublicIpInterface) UnlinkPublicIp(ctx context.Context, linkPublicIpId string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UnlinkPublicIp", ctx, linkPublicIpId)
	ret0, _ := ret[0].(error)
	return ret0
}

// UnlinkPublicIp indicates an expected call of UnlinkPublicIp.
func (mr *MockOscPublicIpInterfaceMockRecorder) UnlinkPublicIp(ctx, linkPublicIpId any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UnlinkPublicIp", reflect.TypeOf((*MockOscPublicIpInterface)(nil).UnlinkPublicIp), ctx, linkPublicIpId)
}

// ValidatePublicIpIds mocks base method.
func (m *MockOscPublicIpInterface) ValidatePublicIpIds(ctx context.Context, publicIpIds []string) ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ValidatePublicIpIds", ctx, publicIpIds)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ValidatePublicIpIds indicates an expected call of ValidatePublicIpIds.
func (mr *MockOscPublicIpInterfaceMockRecorder) ValidatePublicIpIds(ctx, publicIpIds any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ValidatePublicIpIds", reflect.TypeOf((*MockOscPublicIpInterface)(nil).ValidatePublicIpIds), ctx, publicIpIds)
}
