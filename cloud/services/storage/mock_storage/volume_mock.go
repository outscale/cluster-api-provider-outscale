// Code generated by MockGen. DO NOT EDIT.
// Source: ./volume.go

// Package mock_storage is a generated GoMock package.
package mock_storage

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	v1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	osc "github.com/outscale/osc-sdk-go/v2"
)

// MockOscVolumeInterface is a mock of OscVolumeInterface interface.
type MockOscVolumeInterface struct {
	ctrl     *gomock.Controller
	recorder *MockOscVolumeInterfaceMockRecorder
}

// MockOscVolumeInterfaceMockRecorder is the mock recorder for MockOscVolumeInterface.
type MockOscVolumeInterfaceMockRecorder struct {
	mock *MockOscVolumeInterface
}

// NewMockOscVolumeInterface creates a new mock instance.
func NewMockOscVolumeInterface(ctrl *gomock.Controller) *MockOscVolumeInterface {
	mock := &MockOscVolumeInterface{ctrl: ctrl}
	mock.recorder = &MockOscVolumeInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockOscVolumeInterface) EXPECT() *MockOscVolumeInterfaceMockRecorder {
	return m.recorder
}

// CreateVolume mocks base method.
func (m *MockOscVolumeInterface) CreateVolume(ctx context.Context, spec *v1beta1.OscVolume, volumeName, subregionName string) (*osc.Volume, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateVolume", ctx, spec, volumeName, subregionName)
	ret0, _ := ret[0].(*osc.Volume)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateVolume indicates an expected call of CreateVolume.
func (mr *MockOscVolumeInterfaceMockRecorder) CreateVolume(ctx, spec, volumeName, subregionName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateVolume", reflect.TypeOf((*MockOscVolumeInterface)(nil).CreateVolume), ctx, spec, volumeName, subregionName)
}

// DeleteVolume mocks base method.
func (m *MockOscVolumeInterface) DeleteVolume(ctx context.Context, volumeId string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteVolume", ctx, volumeId)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteVolume indicates an expected call of DeleteVolume.
func (mr *MockOscVolumeInterfaceMockRecorder) DeleteVolume(ctx, volumeId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteVolume", reflect.TypeOf((*MockOscVolumeInterface)(nil).DeleteVolume), ctx, volumeId)
}

// GetVolume mocks base method.
func (m *MockOscVolumeInterface) GetVolume(ctx context.Context, volumeId string) (*osc.Volume, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetVolume", ctx, volumeId)
	ret0, _ := ret[0].(*osc.Volume)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetVolume indicates an expected call of GetVolume.
func (mr *MockOscVolumeInterfaceMockRecorder) GetVolume(ctx, volumeId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetVolume", reflect.TypeOf((*MockOscVolumeInterface)(nil).GetVolume), ctx, volumeId)
}

// LinkVolume mocks base method.
func (m *MockOscVolumeInterface) LinkVolume(ctx context.Context, volumeId, vmId, deviceName string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LinkVolume", ctx, volumeId, vmId, deviceName)
	ret0, _ := ret[0].(error)
	return ret0
}

// LinkVolume indicates an expected call of LinkVolume.
func (mr *MockOscVolumeInterfaceMockRecorder) LinkVolume(ctx, volumeId, vmId, deviceName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LinkVolume", reflect.TypeOf((*MockOscVolumeInterface)(nil).LinkVolume), ctx, volumeId, vmId, deviceName)
}

// UnlinkVolume mocks base method.
func (m *MockOscVolumeInterface) UnlinkVolume(ctx context.Context, volumeId string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UnlinkVolume", ctx, volumeId)
	ret0, _ := ret[0].(error)
	return ret0
}

// UnlinkVolume indicates an expected call of UnlinkVolume.
func (mr *MockOscVolumeInterfaceMockRecorder) UnlinkVolume(ctx, volumeId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UnlinkVolume", reflect.TypeOf((*MockOscVolumeInterface)(nil).UnlinkVolume), ctx, volumeId)
}
