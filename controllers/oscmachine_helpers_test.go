package controllers_test

import (
	"testing"

	"github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/compute"
	"github.com/outscale/cluster-api-provider-outscale/controllers"
	"github.com/outscale/osc-sdk-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	defaultPrivateDnsName = "ip-10-0-3-144.eu-west-2.compute.internal"
	defaultPrivateIp      = "10.0.3.144"
	defaultRootVolumeId   = "vol-foo"
)

var defaultVolumes = []osc.BlockDeviceMappingCreated{{
	DeviceName: ptr.To("/dev/sda1"),
	Bsu: &osc.BsuCreated{
		VolumeId: ptr.To(string(defaultRootVolumeId)),
	},
}}

func patchVmExists(vmId string, state v1beta1.VmState, ready bool) patchOSCMachineFunc {
	return func(m *v1beta1.OscMachine) {
		m.Status.Resources.Vm = map[string]string{"default": vmId}
		m.Status.VmState = &state
		m.Status.Ready = ready
	}
}

func patchMoveMachine() patchOSCMachineFunc {
	return func(m *v1beta1.OscMachine) {
		m.UID = "foo"
		m.Status = v1beta1.OscMachineStatus{}
	}
}

func patchUsePublicIP(pool ...string) patchOSCMachineFunc {
	return func(m *v1beta1.OscMachine) {
		m.Spec.Node.Vm.PublicIp = true
		if len(pool) > 0 {
			m.Spec.Node.Vm.PublicIpPool = pool[0]
		}
	}
}

func patchPublicIPStatus(publicIpId string) patchOSCMachineFunc {
	return func(m *v1beta1.OscMachine) {
		if m.Status.Resources.PublicIPs == nil {
			m.Status.Resources.PublicIPs = map[string]string{}
		}
		m.Status.Resources.PublicIPs["default"] = publicIpId
	}
}

func patchDeleteMachine() patchOSCMachineFunc {
	return func(m *infrastructurev1beta1.OscMachine) {
		m.DeletionTimestamp = ptr.To(metav1.Now())
		if len(m.Finalizers) == 0 {
			m.Finalizers = []string{controllers.OscMachineFinalizer}
		}
	}
}

func patchUseOpenSourceOMI() patchOSCMachineFunc {
	return func(m *infrastructurev1beta1.OscMachine) {
		m.Spec.Node.Image.OutscaleOpenSource = true
	}
}

func mockImageFoundByName(name, account, imageId string) mockFunc {
	return func(s *MockCloudServices) {
		s.ImageMock.
			EXPECT().
			GetImageByName(gomock.Any(), gomock.Eq(name), gomock.Eq(account)).
			Return(&osc.Image{ImageId: ptr.To(imageId)}, nil)
	}
}

func mockOpenSourceImageFound(name, region, imageId string) mockFunc {
	return func(s *MockCloudServices) {
		s.ImageMock.
			EXPECT().
			GetImageByName(gomock.Any(), gomock.Eq(name), gomock.Eq(controllers.OutscaleOpenSourceAccounts[region])).
			Return(&osc.Image{ImageId: ptr.To(imageId)}, nil)
	}
}

func mockGetVm(vmId, state string, ccmtags bool) mockFunc {
	vm := &osc.Vm{
		VmId:                &vmId,
		PrivateDnsName:      ptr.To(defaultPrivateDnsName),
		PrivateIp:           ptr.To(defaultPrivateIp),
		State:               &state,
		BlockDeviceMappings: &defaultVolumes,
	}
	if ccmtags {
		vm.Tags = &[]osc.ResourceTag{
			{Key: compute.TagKeyNodeName, Value: defaultPrivateDnsName},
			{Key: compute.TagKeyClusterIDPrefix + "foo", Value: "owned"},
		}
	}
	return func(s *MockCloudServices) {
		s.VMMock.
			EXPECT().
			GetVm(gomock.Any(), gomock.Eq(vmId)).
			Return(vm, nil)
	}
}

func mockGetVmFromClientToken(token string, vm *osc.Vm) mockFunc {
	if vm != nil {
		vm.PrivateDnsName = ptr.To(defaultPrivateDnsName)
		vm.PrivateIp = ptr.To(defaultPrivateIp)
		if vm.BlockDeviceMappings == nil {
			vm.BlockDeviceMappings = &defaultVolumes
		}
	}
	return func(s *MockCloudServices) {
		s.VMMock.
			EXPECT().
			GetVmFromClientToken(gomock.Any(), gomock.Eq(token)).
			Return(vm, nil)
	}
}

func mockCreateVmNoVolumes(vmId, imageId, subnetId string, securityGroupIds, privateIps []string, vmName, clientToken string, vmTags map[string]string) mockFunc {
	return func(s *MockCloudServices) {
		s.VMMock.
			EXPECT().
			CreateVm(gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Eq(imageId), gomock.Eq(subnetId), gomock.Eq(securityGroupIds), gomock.Eq(privateIps), gomock.Eq(vmName), gomock.Eq(clientToken), gomock.Eq(vmTags), gomock.Any()).
			Return(&osc.Vm{
				VmId:                ptr.To(vmId),
				PrivateDnsName:      ptr.To(defaultPrivateDnsName),
				PrivateIp:           ptr.To(defaultPrivateIp),
				BlockDeviceMappings: &defaultVolumes,
				State:               ptr.To("pending"),
			}, nil)
	}
}

func mockCreateVmWithVolumes(vmId string, volumes []infrastructurev1beta1.OscVolume, volumedevices ...string) mockFunc {
	created := []osc.BlockDeviceMappingCreated{{
		DeviceName: ptr.To("/dev/sda1"),
		Bsu: &osc.BsuCreated{
			VolumeId: ptr.To(string(defaultRootVolumeId)),
		},
	}}
	for i, volume := range volumes {
		created = append(created, osc.BlockDeviceMappingCreated{
			DeviceName: &volume.Device,
			Bsu: &osc.BsuCreated{
				VolumeId: &volumedevices[i],
			},
		})
	}
	return func(s *MockCloudServices) {
		s.VMMock.
			EXPECT().
			CreateVm(gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Eq(volumes)).
			Return(&osc.Vm{
				VmId:                ptr.To(vmId),
				PrivateDnsName:      ptr.To(defaultPrivateDnsName),
				PrivateIp:           ptr.To(defaultPrivateIp),
				BlockDeviceMappings: &created,
				State:               ptr.To("pending"),
			}, nil)
	}
}

func mockLinkLoadBalancer(vmId, lb string) mockFunc {
	return func(s *MockCloudServices) {
		s.LoadBalancerMock.EXPECT().
			LinkLoadBalancerBackendMachines(gomock.Any(), []string{vmId}, lb).
			Return(nil)
	}
}

func mockVmSetCCMTag(vmId, clusterID string) mockFunc {
	return func(s *MockCloudServices) {
		s.VMMock.EXPECT().
			AddCCMTags(gomock.Any(), gomock.Eq(clusterID), gomock.Eq(defaultPrivateDnsName), gomock.Eq(vmId)).
			Return(nil)
	}
}

func assertVmExists(vmId string, state v1beta1.VmState, ready bool) assertOSCMachineFunc {
	return func(t *testing.T, m *v1beta1.OscMachine) {
		require.NotNil(t, m.Status.Resources.Vm)
		assert.Equal(t, vmId, m.Status.Resources.Vm["default"])
		require.NotNil(t, m.Status.VmState)
		assert.Equal(t, state, *m.Status.VmState)
		assert.Equal(t, ready, m.Status.Ready)
	}
}

func assertHasMachineFinalizer() assertOSCMachineFunc {
	return func(t *testing.T, m *v1beta1.OscMachine) {
		assert.True(t, controllerutil.ContainsFinalizer(m, controllers.OscMachineFinalizer))
	}
}

func assertStatusMachineResources(rsrcs infrastructurev1beta1.OscMachineResources) assertOSCMachineFunc {
	return func(t *testing.T, c *infrastructurev1beta1.OscMachine) {
		assert.Equal(t, rsrcs, c.Status.Resources)
	}
}

func assertVolumesAreConfigured(deviceAndVolume ...string) assertOSCMachineFunc {
	return func(t *testing.T, m *v1beta1.OscMachine) {
		expect := map[string]string{
			"/dev/sda1": defaultRootVolumeId,
		}
		for i := 0; i < len(deviceAndVolume); i += 2 {
			expect[deviceAndVolume[i]] = deviceAndVolume[i+1]
		}
		assert.Equal(t, expect, m.Status.Resources.Volumes)
	}
}
