// Code generated by MockGen. DO NOT EDIT.
// Source: ./volume.go

// Package mock_storage is a generated GoMock package.
package mock_storage

import (
	reflect "reflect"
	time "time"

	gomock "github.com/golang/mock/gomock"
	v1beta2 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta2"
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

// CheckVolumeState mocks base method.
func (m *MockOscVolumeInterface) CheckVolumeState(clockInsideLoop, clockLoop time.Duration, state, volumeId string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CheckVolumeState", clockInsideLoop, clockLoop, state, volumeId)
	ret0, _ := ret[0].(error)
	return ret0
}

// CheckVolumeState indicates an expected call of CheckVolumeState.
func (mr *MockOscVolumeInterfaceMockRecorder) CheckVolumeState(clockInsideLoop, clockLoop, state, volumeId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CheckVolumeState", reflect.TypeOf((*MockOscVolumeInterface)(nil).CheckVolumeState), clockInsideLoop, clockLoop, state, volumeId)
}

// CreateVolume mocks base method.
func (m *MockOscVolumeInterface) CreateVolume(spec *v1beta2.OscVolume, volumeName string) (*osc.Volume, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateVolume", spec, volumeName)
	ret0, _ := ret[0].(*osc.Volume)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateVolume indicates an expected call of CreateVolume.
func (mr *MockOscVolumeInterfaceMockRecorder) CreateVolume(spec, volumeName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateVolume", reflect.TypeOf((*MockOscVolumeInterface)(nil).CreateVolume), spec, volumeName)
}

// DeleteVolume mocks base method.
func (m *MockOscVolumeInterface) DeleteVolume(volumeId string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteVolume", volumeId)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteVolume indicates an expected call of DeleteVolume.
func (mr *MockOscVolumeInterfaceMockRecorder) DeleteVolume(volumeId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteVolume", reflect.TypeOf((*MockOscVolumeInterface)(nil).DeleteVolume), volumeId)
}

// GetVolume mocks base method.
func (m *MockOscVolumeInterface) GetVolume(volumeId string) (*osc.Volume, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetVolume", volumeId)
	ret0, _ := ret[0].(*osc.Volume)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetVolume indicates an expected call of GetVolume.
func (mr *MockOscVolumeInterfaceMockRecorder) GetVolume(volumeId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetVolume", reflect.TypeOf((*MockOscVolumeInterface)(nil).GetVolume), volumeId)
}

// LinkVolume mocks base method.
func (m *MockOscVolumeInterface) LinkVolume(volumeId, vmId, deviceName string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LinkVolume", volumeId, vmId, deviceName)
	ret0, _ := ret[0].(error)
	return ret0
}

// LinkVolume indicates an expected call of LinkVolume.
func (mr *MockOscVolumeInterfaceMockRecorder) LinkVolume(volumeId, vmId, deviceName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LinkVolume", reflect.TypeOf((*MockOscVolumeInterface)(nil).LinkVolume), volumeId, vmId, deviceName)
}

// UnlinkVolume mocks base method.
func (m *MockOscVolumeInterface) UnlinkVolume(volumeId string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UnlinkVolume", volumeId)
	ret0, _ := ret[0].(error)
	return ret0
}

// UnlinkVolume indicates an expected call of UnlinkVolume.
func (mr *MockOscVolumeInterfaceMockRecorder) UnlinkVolume(volumeId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UnlinkVolume", reflect.TypeOf((*MockOscVolumeInterface)(nil).UnlinkVolume), volumeId)
}

// ValidateVolumeIds mocks base method.
func (m *MockOscVolumeInterface) ValidateVolumeIds(volumeIds []string) ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ValidateVolumeIds", volumeIds)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ValidateVolumeIds indicates an expected call of ValidateVolumeIds.
func (mr *MockOscVolumeInterfaceMockRecorder) ValidateVolumeIds(volumeIds interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ValidateVolumeIds", reflect.TypeOf((*MockOscVolumeInterface)(nil).ValidateVolumeIds), volumeIds)
}
