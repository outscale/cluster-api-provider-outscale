package controllers_test

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/osc-sdk-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const defaultImageId = "ami-foo"

func mockImageFoundByName(name string) mockFunc {
	return func(s *MockCloudServices) {
		s.ImageMock.
			EXPECT().
			GetImageId(gomock.Any(), gomock.Eq(name)).
			Return(defaultImageId, nil)
		s.ImageMock.
			EXPECT().
			GetImage(gomock.Any(), gomock.Eq(defaultImageId)).
			Return(&osc.Image{}, nil)
	}
}

func mockKeyPairFound(name string) mockFunc {
	return func(s *MockCloudServices) {
		s.KeyPairMock.
			EXPECT().
			GetKeyPair(gomock.Any(), gomock.Eq(name)).
			Return(&osc.Keypair{}, nil)
	}
}

func mockNoVmFoundByName(name string) mockFunc {
	return func(s *MockCloudServices) {
		s.VMMock.
			EXPECT().
			GetVmListFromTag(gomock.Any(), gomock.Eq("Name"), gomock.Eq(name)).
			Return(nil, nil)
	}
}

const (
	defaultVmId           = "i-foo"
	defaultPrivateDnsName = "ip-10-0-3-144.eu-west-2.compute.internal"
	defaultPrivateIp      = "10.0.3.144"
)

func mockCreateVm(subnetId string, securityGroupIds, privateIps []string, vmName string, vmTags map[string]string) mockFunc {
	return func(s *MockCloudServices) {
		s.VMMock.
			EXPECT().
			CreateVm(gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Eq(subnetId), gomock.Eq(securityGroupIds), gomock.Eq(privateIps), gomock.Eq(vmName), gomock.Eq(vmTags)).
			Return(&osc.Vm{VmId: ptr.To(defaultVmId)}, nil)
	}
}

func mockGetVm() mockFunc {
	return func(s *MockCloudServices) {
		s.VMMock.
			EXPECT().
			GetVm(gomock.Any(), gomock.Eq(defaultVmId)).
			Return(&osc.Vm{
				VmId:           ptr.To(defaultVmId),
				PrivateDnsName: ptr.To(defaultPrivateDnsName),
				PrivateIp:      ptr.To(defaultPrivateIp),
			}, nil)
	}
}

func mockGetVmState(state string) mockFunc {
	return func(s *MockCloudServices) {
		s.VMMock.
			EXPECT().
			GetVmState(gomock.Any(), gomock.Eq(defaultVmId)).
			Return(state, nil)
	}
}

func mockLinkLoadBalancer(lb string) mockFunc {
	return func(s *MockCloudServices) {
		s.LoadBalancerMock.EXPECT().
			LinkLoadBalancerBackendMachines(gomock.Any(), []string{defaultVmId}, lb).
			Return(nil)
	}
}

func mockVmReadEmptyCCMTag() mockFunc {
	return func(s *MockCloudServices) {
		s.TagMock.EXPECT().
			ReadTag(gomock.Any(), gomock.Eq("OscK8sNodeName"), gomock.Eq(defaultPrivateDnsName))
	}
}

func mockVmSetCCMTag(clusterName string) mockFunc {
	return func(s *MockCloudServices) {
		s.VMMock.EXPECT().
			AddCcmTag(gomock.Any(), clusterName, gomock.Eq(defaultPrivateDnsName), gomock.Eq(defaultVmId))
	}
}

func patchVmExists(state v1beta1.VmState, ready bool) patchOSCMachineFunc {
	return func(m *v1beta1.OscMachine) {
		m.Spec.Node.Vm.ResourceId = defaultVmId
		m.Status.VmState = &state
		m.Status.Ready = ready
	}
}

func assertVmExists(state v1beta1.VmState, ready bool) assertOSCMachineFunc {
	return func(t *testing.T, m *v1beta1.OscMachine) {
		assert.Equal(t, defaultVmId, m.Spec.Node.Vm.ResourceId)
		require.NotNil(t, m.Status.VmState)
		assert.Equal(t, state, *m.Status.VmState)
		assert.Equal(t, ready, m.Status.Ready)
	}
}

func assertHasFinalizer() assertOSCMachineFunc {
	return func(t *testing.T, m *v1beta1.OscMachine) {
		assert.True(t, controllerutil.ContainsFinalizer(m, "oscmachine.infrastructure.cluster.x-k8s.io"))
	}
}
