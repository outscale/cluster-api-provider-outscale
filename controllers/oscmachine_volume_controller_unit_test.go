/*
Copyright 2022 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"time"

	"testing"

	"github.com/golang/mock/gomock"
	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/storage/mock_storage"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag/mock_tag"
	osc "github.com/outscale/osc-sdk-go/v2"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2/klogr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

var (
	defaultClusterInitialize = infrastructurev1beta1.OscClusterSpec{
		Network: infrastructurev1beta1.OscNetwork{
			Net: infrastructurev1beta1.OscNet{
				Name:    "test-net",
				IpRange: "10.0.0.0/16",
			},
		},
	}
	defaultClusterReconcile = infrastructurev1beta1.OscClusterSpec{
		Network: infrastructurev1beta1.OscNetwork{
			Net: infrastructurev1beta1.OscNet{
				Name:       "test-net",
				IpRange:    "10.0.0.0/16",
				ResourceId: "vpc-test-net-uid",
			},
		},
	}
	defaultVolumeInitialize = infrastructurev1beta1.OscMachineSpec{
		Node: infrastructurev1beta1.OscNode{
			Volumes: []*infrastructurev1beta1.OscVolume{
				{
					Name:          "test-volume",
					Iops:          1000,
					Size:          50,
					VolumeType:    "io1",
					SubregionName: "eu-west-2a",
				},
			},
		},
	}
	defaultVolumeReconcile = infrastructurev1beta1.OscMachineSpec{
		Node: infrastructurev1beta1.OscNode{
			Volumes: []*infrastructurev1beta1.OscVolume{
				{
					Name:          "test-volume",
					Iops:          1000,
					Size:          50,
					VolumeType:    "io1",
					SubregionName: "eu-west-2a",
					ResourceId:    "volume-test-volume-uid",
				},
			},
		},
	}
	defaultMultiVolumeInitialize = infrastructurev1beta1.OscMachineSpec{
		Node: infrastructurev1beta1.OscNode{
			Volumes: []*infrastructurev1beta1.OscVolume{
				{
					Name:          "test-volume-first",
					Iops:          1000,
					Size:          30,
					VolumeType:    "io1",
					SubregionName: "eu-west-2a",
				},
				{
					Name:          "test-volume-second",
					Iops:          1000,
					Size:          30,
					VolumeType:    "io1",
					SubregionName: "eu-west-2a",
				},
			},
		},
	}

	defaultMultiVolumeReconcile = infrastructurev1beta1.OscMachineSpec{
		Node: infrastructurev1beta1.OscNode{
			Volumes: []*infrastructurev1beta1.OscVolume{
				{
					Name:          "test-volume-first",
					Iops:          1000,
					Size:          30,
					VolumeType:    "io1",
					SubregionName: "eu-west-2a",
					ResourceId:    "volume-test-volume-first-uid",
				},
				{
					Name:          "test-volume-second",
					Iops:          1000,
					Size:          30,
					VolumeType:    "io1",
					SubregionName: "eu-west-2a",
					ResourceId:    "volume-test-volume-second-uid",
				},
			},
		},
	}
)

// Setup set osccluster, oscmachine, machineScope and clusterScope
func SetupMachine(t *testing.T, name string, clusterSpec infrastructurev1beta1.OscClusterSpec, machineSpec infrastructurev1beta1.OscMachineSpec) (clusterScope *scope.ClusterScope, machineScope *scope.MachineScope) {
	t.Logf("Validate to %s", name)

	oscCluster := infrastructurev1beta1.OscCluster{
		Spec: clusterSpec,
		ObjectMeta: metav1.ObjectMeta{
			UID:       "uid",
			Name:      "test-osc",
			Namespace: "test-system",
		},
	}
	oscMachine := infrastructurev1beta1.OscMachine{
		Spec: machineSpec,
		ObjectMeta: metav1.ObjectMeta{
			UID:       "uid",
			Name:      "test-osc",
			Namespace: "test-system",
		},
	}

	log := klogr.New()
	clusterScope = &scope.ClusterScope{
		Logger: log,
		Cluster: &clusterv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				UID:       "uid",
				Name:      "test-osc",
				Namespace: "test-system",
			},
		},
		OscCluster: &oscCluster,
	}
	machineScope = &scope.MachineScope{
		Logger: log,
		Cluster: &clusterv1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				UID:       "uid",
				Name:      "test-osc",
				Namespace: "test-system",
			},
		},
		Machine: &clusterv1.Machine{
			ObjectMeta: metav1.ObjectMeta{
				UID:       "uid",
				Name:      "test-osc",
				Namespace: "test-system",
			},
		},
		OscCluster: &oscCluster,
		OscMachine: &oscMachine,
	}
	return clusterScope, machineScope
}

// SetupWithVolumeMock set publicIpMock with clusterScope, machineScope and oscmachine
func SetupWithVolumeMock(t *testing.T, name string, clusterSpec infrastructurev1beta1.OscClusterSpec, machineSpec infrastructurev1beta1.OscMachineSpec) (clusterScope *scope.ClusterScope, machineScope *scope.MachineScope, ctx context.Context, mockOscVolumeInterface *mock_storage.MockOscVolumeInterface, mockOscTagInterface *mock_tag.MockOscTagInterface) {
	clusterScope, machineScope = SetupMachine(t, name, clusterSpec, machineSpec)
	mockCtrl := gomock.NewController(t)
	mockOscVolumeInterface = mock_storage.NewMockOscVolumeInterface(mockCtrl)
	mockOscTagInterface = mock_tag.NewMockOscTagInterface(mockCtrl)
	ctx = context.Background()
	return clusterScope, machineScope, ctx, mockOscVolumeInterface, mockOscTagInterface
}

// TestGetVolumeResourceId has several tests to cover the code of the function getVolumeResourceId
func TestGetVolumeResourceId(t *testing.T) {
	volumeTestCases := []struct {
		name                      string
		clusterSpec               infrastructurev1beta1.OscClusterSpec
		machineSpec               infrastructurev1beta1.OscMachineSpec
		expVolumeFound            bool
		expGetVolumeResourceIdErr error
	}{
		{
			name:                      "get VolumeId",
			clusterSpec:               defaultClusterInitialize,
			machineSpec:               defaultVolumeInitialize,
			expVolumeFound:            true,
			expGetVolumeResourceIdErr: nil,
		},
		{
			name:                      "can not get VolumeId",
			clusterSpec:               defaultClusterInitialize,
			machineSpec:               defaultVolumeInitialize,
			expVolumeFound:            false,
			expGetVolumeResourceIdErr: fmt.Errorf("test-volume-uid does not exist"),
		},
	}
	for _, vtc := range volumeTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			_, machineScope := SetupMachine(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			volumesSpec := vtc.machineSpec.Node.Volumes
			for _, volumeSpec := range volumesSpec {
				volumeName := volumeSpec.Name + "-uid"
				volumeId := "volume-" + volumeName
				if vtc.expVolumeFound {
					volumeRef := machineScope.GetVolumeRef()
					volumeRef.ResourceMap = make(map[string]string)
					volumeRef.ResourceMap[volumeName] = volumeId
				}
				volumeResourceId, err := getVolumeResourceId(volumeName, machineScope)
				if err != nil {
					assert.Equal(t, vtc.expGetVolumeResourceIdErr, err, "getVolumeResourceId() should return the same error")
				} else {
					assert.Nil(t, vtc.expGetVolumeResourceIdErr)
				}
				t.Logf("Find volumeResourceId %s\n", volumeResourceId)
			}
		})
	}
}

// TestCheckVolumeOscDuplicateName has several tests to cover the code of the function checkVolumeOscDuplicateNam
func TestCheckVolumeOscDuplicateName(t *testing.T) {
	volumeTestCases := []struct {
		name                              string
		clusterSpec                       infrastructurev1beta1.OscClusterSpec
		machineSpec                       infrastructurev1beta1.OscMachineSpec
		expCheckVolumeOscDuplicateNameErr error
	}{
		{
			name:                              "get separate name",
			clusterSpec:                       defaultClusterInitialize,
			machineSpec:                       defaultMultiVolumeInitialize,
			expCheckVolumeOscDuplicateNameErr: nil,
		},
		{
			name:        "get duplicate Name",
			clusterSpec: defaultClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Volumes: []*infrastructurev1beta1.OscVolume{
						{
							Name:          "test-volume",
							Iops:          1000,
							Size:          30,
							VolumeType:    "io1",
							SubregionName: "eu-west-2a",
						},
						{
							Name:          "test-volume",
							Iops:          1000,
							Size:          30,
							VolumeType:    "io1",
							SubregionName: "eu-west-2a",
						},
					},
				},
			},
			expCheckVolumeOscDuplicateNameErr: fmt.Errorf("test-volume already exist"),
		},
	}
	for _, vtc := range volumeTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			_, machineScope := SetupMachine(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			duplicateResourceVolumeErr := checkVolumeOscDuplicateName(machineScope)
			if duplicateResourceVolumeErr != nil {
				assert.Equal(t, vtc.expCheckVolumeOscDuplicateNameErr, duplicateResourceVolumeErr, "checkVolumeOscDuplicateName() should return the same error")
			} else {
				assert.Nil(t, vtc.expCheckVolumeOscDuplicateNameErr)
			}
		})
	}
}

/*
// TestReconcileVolumeResourceId has several tests to cover the code of the function reconcileVolume
func TestReconcileVolumeResourceId(t *testing.T) {
	vmTestCases := []struct {
		name                              string
		clusterSpec                       infrastructurev1beta1.OscClusterSpec
		machineSpec                       infrastructurev1beta1.OscMachineSpec
		expVolumeFound                    bool
		expSubnetFound                    bool
		expTagFound                       bool
		expPublicIpFound                  bool
		expLinkPublicIpFound              bool
		expSecurityGroupFound             bool
		expLoadBalancerSecurityGroupFound bool
		expReadTagErr                     error
		expReconcileVmErr                 error
	}{
		{
			name:                              "Volume does not exist ",
			clusterSpec:                       defaultVmClusterInitialize,
			machineSpec:                       defaultVmVolumeInitialize,
			expVolumeFound:                    false,
			expSubnetFound:                    true,
			expPublicIpFound:                  true,
			expLinkPublicIpFound:              true,
			expSecurityGroupFound:             true,
			expLoadBalancerSecurityGroupFound: true,
			expTagFound:                       false,
			expReadTagErr:                     nil,
			expReconcileVmErr:                 fmt.Errorf("test-volume-uid does not exist"),
		},
		{
			name:                              "Volume does not exist ",
			clusterSpec:                       defaultVmClusterInitialize,
			machineSpec:                       defaultVmInitialize,
			expVolumeFound:                    true,
			expSubnetFound:                    false,
			expPublicIpFound:                  true,
			expLinkPublicIpFound:              true,
			expSecurityGroupFound:             true,
			expLoadBalancerSecurityGroupFound: true,
			expTagFound:                       false,
			expReadTagErr:                     nil,
			expReconcileVmErr:                 fmt.Errorf("failed to get subnet ID for test-subnet-uid: test-subnet-uid does not exist"),
		},
		{
			name:                              "PublicIp does not exist ",
			clusterSpec:                       defaultVmClusterInitialize,
			machineSpec:                       defaultVmInitialize,
			expVolumeFound:                    true,
			expSubnetFound:                    true,
			expPublicIpFound:                  false,
			expLinkPublicIpFound:              false,
			expSecurityGroupFound:             true,
			expLoadBalancerSecurityGroupFound: true,
			expTagFound:                       false,
			expReadTagErr:                     nil,
			expReconcileVmErr:                 fmt.Errorf("test-publicip-uid does not exist"),
		},
		{
			name:                              "SecurityGroup does not exist ",
			clusterSpec:                       defaultVmClusterInitialize,
			machineSpec:                       defaultVmInitialize,
			expVolumeFound:                    true,
			expSubnetFound:                    true,
			expPublicIpFound:                  true,
			expLinkPublicIpFound:              true,
			expSecurityGroupFound:             false,
			expLoadBalancerSecurityGroupFound: false,
			expTagFound:                       false,
			expReadTagErr:                     nil,
			expReconcileVmErr:                 fmt.Errorf("test-securitygroup-uid does not exist (yet)"),
		},
		{
			name:                              "failed to get tag",
			clusterSpec:                       defaultVmClusterInitialize,
			machineSpec:                       defaultVmInitialize,
			expVolumeFound:                    true,
			expSubnetFound:                    true,
			expPublicIpFound:                  true,
			expLinkPublicIpFound:              true,
			expSecurityGroupFound:             true,
			expLoadBalancerSecurityGroupFound: true,
			expTagFound:                       true,
			expReadTagErr:                     fmt.Errorf("ReadTag generic error"),
			expReconcileVmErr:                 fmt.Errorf("ReadTag generic error Can not get tag for OscMachine test-system/test-osc"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope, ctx, mockOscVmInterface, mockOscVolumeInterface, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface, mockOscTagInterface := SetupWithVmMock(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			vmName := vtc.machineSpec.Node.Vm.Name + "-uid"
			vmId := "i-" + vmName
			volumeName := vtc.machineSpec.Node.Vm.VolumeName + "-uid"
			volumeId := "vol-" + volumeName
			volumeRef := machineScope.GetVolumeRef()
			volumeRef.ResourceMap = make(map[string]string)
			if vtc.expVolumeFound {
				volumeRef.ResourceMap[volumeName] = volumeId
			}

			subnetName := vtc.machineSpec.Node.Vm.SubnetName + "-uid"
			subnetId := "subnet-" + subnetName
			subnetRef := clusterScope.GetSubnetRef()
			subnetRef.ResourceMap = make(map[string]string)
			if vtc.expSubnetFound {
				subnetRef.ResourceMap[subnetName] = subnetId
			}

			publicIpName := vtc.machineSpec.Node.Vm.PublicIpName + "-uid"
			publicIpId := "eipalloc-" + publicIpName
			publicIpRef := clusterScope.GetPublicIpRef()
			publicIpRef.ResourceMap = make(map[string]string)
			if vtc.expPublicIpFound {
				publicIpRef.ResourceMap[publicIpName] = publicIpId
			}

			linkPublicIpId := "eipassoc-" + publicIpName
			linkPublicIpRef := machineScope.GetLinkPublicIpRef()
			linkPublicIpRef.ResourceMap = make(map[string]string)
			if vtc.expLinkPublicIpFound {
				linkPublicIpRef.ResourceMap[vmName] = linkPublicIpId
			}

			var privateIps []string
			vmPrivateIps := machineScope.GetVmPrivateIps()
			for _, vmPrivateIp := range *vmPrivateIps {
				privateIp := vmPrivateIp.PrivateIp
				privateIps = append(privateIps, privateIp)
			}

			tag := osc.Tag{
				ResourceId: &vmId,
			}
			if vtc.expVolumeFound && vtc.expSubnetFound && vtc.expPublicIpFound && vtc.expLinkPublicIpFound && vtc.expSecurityGroupFound {
				if vtc.expTagFound {
					mockOscTagInterface.
						EXPECT().
						ReadTag(gomock.Eq("Name"), gomock.Eq(vmName)).
						Return(&tag, vtc.expReadTagErr)
				} else {
					mockOscTagInterface.
						EXPECT().
						ReadTag(gomock.Eq("Name"), gomock.Eq(vmName)).
						Return(nil, vtc.expReadTagErr)
				}
			}

			var securityGroupIds []string
			vmSecurityGroups := machineScope.GetVmSecurityGroups()
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			for _, vmSecurityGroup := range *vmSecurityGroups {
				securityGroupName := vmSecurityGroup.Name + "-uid"
				securityGroupId := "sg-" + securityGroupName
				if vtc.expSecurityGroupFound {
					securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
				}
				securityGroupIds = append(securityGroupIds, securityGroupId)
			}

			loadBalancerSpec := clusterScope.GetLoadBalancer()
			loadBalancerSpec.SetDefaultValue()
			loadBalancerSecurityGroupName := loadBalancerSpec.SecurityGroupName
			loadBalancerSecurityGroupClusterScopeName := loadBalancerSecurityGroupName + "-uid"
			loadBalancerSecurityGroupId := "sg-" + loadBalancerSecurityGroupClusterScopeName
			if vtc.expLoadBalancerSecurityGroupFound {
				securityGroupsRef.ResourceMap[loadBalancerSecurityGroupClusterScopeName] = loadBalancerSecurityGroupId
			}

			reconcileVm, err := reconcileVm(ctx, clusterScope, machineScope, mockOscVmInterface, mockOscVolumeInterface, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface, mockOscTagInterface)
			if err != nil {
				assert.Equal(t, vtc.expReconcileVmErr.Error(), err.Error(), "reconcileVm() should return the same error")
			} else {
				assert.Nil(t, vtc.expReconcileVmErr)
			}
			t.Logf("find reconcileVm %v\n", reconcileVm)
		})
	}
}*/

// TestReconcileVolumeCreate has several tests to cover the code of the function reconcileVolume
func TestReconcileVolumeCreate(t *testing.T) {
	volumeTestCases := []struct {
		name                              string
		clusterSpec                       infrastructurev1beta1.OscClusterSpec
		machineSpec                       infrastructurev1beta1.OscMachineSpec
		expVolumeFound                    bool
		expCheckVolumeStateAvailableFound bool
		expCreateVolumeFound              bool
		expUserDeleteVolumeFound          bool
		expTagFound                       bool
		expCreateVolumeErr                error
		expCheckVolumeStateAvailableErr   error
		expValidateVolumeIdsErr           error
		expReadTagErr                     error
		expReconcileVolumeErr             error
	}{
		{
			name:                              "create volume (first time reconcile loop)",
			clusterSpec:                       defaultClusterInitialize,
			machineSpec:                       defaultVolumeInitialize,
			expVolumeFound:                    false,
			expUserDeleteVolumeFound:          false,
			expTagFound:                       false,
			expCheckVolumeStateAvailableFound: true,
			expValidateVolumeIdsErr:           nil,
			expCreateVolumeFound:              true,
			expCreateVolumeErr:                nil,
			expCheckVolumeStateAvailableErr:   nil,
			expReadTagErr:                     nil,
			expReconcileVolumeErr:             nil,
		},
		{
			name:                              "create two volumes (first time reconcile loop)",
			clusterSpec:                       defaultClusterInitialize,
			machineSpec:                       defaultMultiVolumeInitialize,
			expVolumeFound:                    false,
			expUserDeleteVolumeFound:          false,
			expCheckVolumeStateAvailableFound: true,
			expCreateVolumeFound:              true,
			expTagFound:                       false,
			expValidateVolumeIdsErr:           nil,
			expCreateVolumeErr:                nil,
			expCheckVolumeStateAvailableErr:   nil,
			expReadTagErr:                     nil,
			expReconcileVolumeErr:             nil,
		},
		{
			name:                              "failed to create volume",
			clusterSpec:                       defaultClusterInitialize,
			machineSpec:                       defaultVolumeInitialize,
			expVolumeFound:                    false,
			expUserDeleteVolumeFound:          false,
			expCheckVolumeStateAvailableFound: false,
			expCreateVolumeFound:              false,
			expTagFound:                       false,
			expValidateVolumeIdsErr:           nil,
			expCreateVolumeErr:                fmt.Errorf("CreateVolume generic error"),
			expCheckVolumeStateAvailableErr:   nil,
			expReadTagErr:                     nil,
			expReconcileVolumeErr:             fmt.Errorf("CreateVolume generic error Can not create volume for OscMachine test-system/test-osc"),
		},
		{
			name:                              "user delete volume without cluster-api",
			clusterSpec:                       defaultClusterInitialize,
			machineSpec:                       defaultVolumeReconcile,
			expVolumeFound:                    false,
			expCheckVolumeStateAvailableFound: true,
			expUserDeleteVolumeFound:          true,
			expCreateVolumeFound:              true,
			expTagFound:                       false,
			expValidateVolumeIdsErr:           nil,
			expCreateVolumeErr:                nil,
			expCheckVolumeStateAvailableErr:   nil,
			expReadTagErr:                     nil,
			expReconcileVolumeErr:             nil,
		},
		{
			name:                              "failed get vmVolumeState",
			clusterSpec:                       defaultClusterInitialize,
			machineSpec:                       defaultVolumeReconcile,
			expVolumeFound:                    false,
			expCheckVolumeStateAvailableFound: true,
			expUserDeleteVolumeFound:          true,
			expValidateVolumeIdsErr:           nil,
			expCreateVolumeFound:              true,
			expCreateVolumeErr:                nil,
			expCheckVolumeStateAvailableErr:   fmt.Errorf("CheckVolumeStateAvailable generic error"),
			expReadTagErr:                     nil,
			expReconcileVolumeErr:             fmt.Errorf("CheckVolumeStateAvailable generic error Can not get volume available for OscMachine test-system/test-osc"),
		},
	}
	for _, vtc := range volumeTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			_, machineScope, ctx, mockOscVolumeInterface, mockOscTagInterface := SetupWithVolumeMock(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			volumesSpec := vtc.machineSpec.Node.Volumes
			var volumesIds []string
			var clockInsideLoop time.Duration = 5
			var clockLoop time.Duration = 60
			volumeStateAvailable := "available"
			for index, volumeSpec := range volumesSpec {
				volumeName := volumeSpec.Name + "-uid"
				volumeId := "volume-" + volumeName
				tag := osc.Tag{
					ResourceId: &volumeId,
				}
				if vtc.expTagFound {
					mockOscTagInterface.
						EXPECT().
						ReadTag(gomock.Eq("Name"), gomock.Eq(volumeName)).
						Return(&tag, vtc.expReadTagErr)
				} else {
					mockOscTagInterface.
						EXPECT().
						ReadTag(gomock.Eq("Name"), gomock.Eq(volumeName)).
						Return(nil, vtc.expReadTagErr)
				}
				volumesIds = append(volumesIds, volumeId)
				volume := osc.CreateVolumeResponse{
					Volume: &osc.Volume{
						VolumeId: &volumeId,
					},
				}
				volumeRef := machineScope.GetVolumeRef()
				volumeRef.ResourceMap = make(map[string]string)
				if vtc.expCreateVolumeFound {
					volumeRef.ResourceMap[volumeName] = volumeId
					if !vtc.expUserDeleteVolumeFound {
						volumesIds[index] = ""
					}
					mockOscVolumeInterface.
						EXPECT().
						CreateVolume(gomock.Eq(volumeSpec), gomock.Eq(volumeName)).
						Return(volume.Volume, vtc.expCreateVolumeErr)
				} else {
					mockOscVolumeInterface.
						EXPECT().
						CreateVolume(gomock.Eq(volumeSpec), gomock.Eq(volumeName)).
						Return(nil, vtc.expCreateVolumeErr)
				}
				if vtc.expCheckVolumeStateAvailableFound {
					mockOscVolumeInterface.
						EXPECT().
						CheckVolumeState(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(volumeStateAvailable), gomock.Eq(volumeId)).
						Return(vtc.expCheckVolumeStateAvailableErr)
				}

			}
			if vtc.expUserDeleteVolumeFound {
				mockOscVolumeInterface.
					EXPECT().
					ValidateVolumeIds(gomock.Eq(volumesIds)).
					Return([]string{""}, vtc.expValidateVolumeIdsErr)
			} else {
				if vtc.expVolumeFound {
					mockOscVolumeInterface.
						EXPECT().
						ValidateVolumeIds(gomock.Eq(volumesIds)).
						Return(volumesIds, vtc.expValidateVolumeIdsErr)
				} else {
					mockOscVolumeInterface.
						EXPECT().
						ValidateVolumeIds(gomock.Eq(volumesIds)).
						Return(nil, vtc.expValidateVolumeIdsErr)
				}
			}

			reconcileVolume, err := reconcileVolume(ctx, machineScope, mockOscVolumeInterface, mockOscTagInterface)
			if err != nil {
				assert.Equal(t, vtc.expReconcileVolumeErr.Error(), err.Error(), "reconcileVolume should return the same error")
			} else {
				assert.Nil(t, vtc.expReconcileVolumeErr)
			}
			t.Logf("Find reconcileVolume %v\n", reconcileVolume)
		})
	}
}

// TestReconcileVolumeGet has several tests to cover the code of the function reconcileVolume
func TestReconcileVolumeGet(t *testing.T) {
	volumeTestCases := []struct {
		name                    string
		clusterSpec             infrastructurev1beta1.OscClusterSpec
		machineSpec             infrastructurev1beta1.OscMachineSpec
		expVolumeFound          bool
		expTagFound             bool
		expValidateVolumeIdsErr error
		expReadTagErr           error
		expReconcileVolumeErr   error
	}{
		{
			name:                    "check volume exist (second time reconcile loop)",
			clusterSpec:             defaultClusterReconcile,
			machineSpec:             defaultVolumeReconcile,
			expVolumeFound:          true,
			expTagFound:             true,
			expValidateVolumeIdsErr: nil,
			expReadTagErr:           nil,
			expReconcileVolumeErr:   nil,
		},
		{
			name:                    "check two volumes exist (second time reconcile loop)",
			clusterSpec:             defaultClusterReconcile,
			machineSpec:             defaultMultiVolumeReconcile,
			expVolumeFound:          true,
			expTagFound:             true,
			expValidateVolumeIdsErr: nil,
			expReadTagErr:           nil,
			expReconcileVolumeErr:   nil,
		},
		{
			name:                    "failed to validate volume",
			clusterSpec:             defaultClusterReconcile,
			machineSpec:             defaultVolumeReconcile,
			expVolumeFound:          false,
			expTagFound:             true,
			expValidateVolumeIdsErr: fmt.Errorf("ValidateVolumeIds generic error"),
			expReadTagErr:           nil,
			expReconcileVolumeErr:   fmt.Errorf("ValidateVolumeIds generic error"),
		},
		{
			name:                    "failed to get tag",
			clusterSpec:             defaultClusterReconcile,
			machineSpec:             defaultVolumeReconcile,
			expVolumeFound:          true,
			expTagFound:             false,
			expValidateVolumeIdsErr: nil,
			expReadTagErr:           fmt.Errorf("ReadTag generic error"),
			expReconcileVolumeErr:   fmt.Errorf("ReadTag generic error Can not get tag for OscMachine test-system/test-osc"),
		},
	}
	for _, vtc := range volumeTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			_, machineScope, ctx, mockOscVolumeInterface, mockOscTagInterface := SetupWithVolumeMock(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			volumesSpec := vtc.machineSpec.Node.Volumes
			var volumesIds []string
			for _, volumeSpec := range volumesSpec {
				volumeName := volumeSpec.Name + "-uid"
				volumeId := "volume-" + volumeName
				tag := osc.Tag{
					ResourceId: &volumeId,
				}
				if vtc.expVolumeFound {
					if vtc.expTagFound {
						mockOscTagInterface.
							EXPECT().
							ReadTag(gomock.Eq("Name"), gomock.Eq(volumeName)).
							Return(&tag, vtc.expReadTagErr)
					} else {
						mockOscTagInterface.
							EXPECT().
							ReadTag(gomock.Eq("Name"), gomock.Eq(volumeName)).
							Return(nil, vtc.expReadTagErr)
					}
				}
				volumesIds = append(volumesIds, volumeId)
				volumeRef := machineScope.GetVolumeRef()
				volumeRef.ResourceMap = make(map[string]string)
				volumeRef.ResourceMap[volumeName] = volumeId

			}
			if vtc.expVolumeFound {
				mockOscVolumeInterface.
					EXPECT().
					ValidateVolumeIds(gomock.Eq(volumesIds)).
					Return(volumesIds, vtc.expValidateVolumeIdsErr)
			} else {
				mockOscVolumeInterface.
					EXPECT().
					ValidateVolumeIds(gomock.Eq(volumesIds)).
					Return(nil, vtc.expValidateVolumeIdsErr)
			}

			reconcileVolume, err := reconcileVolume(ctx, machineScope, mockOscVolumeInterface, mockOscTagInterface)
			if err != nil {
				assert.Equal(t, vtc.expReconcileVolumeErr.Error(), err.Error(), "reconcileVolume should return the same error")
			} else {
				assert.Nil(t, vtc.expReconcileVolumeErr)
			}
			t.Logf("Find reconcileVolume %v\n", reconcileVolume)
		})
	}
}

// TestReconcileDeleteVolumeDelete has several tests to cover the code of the function reconcileDeleteVolume
func TestReconcileDeleteVolumeDelete(t *testing.T) {
	volumeTestCases := []struct {
		name                            string
		clusterSpec                     infrastructurev1beta1.OscClusterSpec
		machineSpec                     infrastructurev1beta1.OscMachineSpec
		expVolumeFound                  bool
		expValidateVolumeIdsErr         error
		expCheckVolumeStateAvailableErr error
		expUnlinkVolumeErr              error
		expCheckVolumeStateUseErr       error
		expDeleteVolumeErr              error
		expReconcileDeleteVolumeErr     error
	}{
		{
			name:                            "delete volume (first time reconcile loop)",
			clusterSpec:                     defaultClusterReconcile,
			machineSpec:                     defaultVolumeReconcile,
			expVolumeFound:                  true,
			expDeleteVolumeErr:              nil,
			expValidateVolumeIdsErr:         nil,
			expCheckVolumeStateAvailableErr: nil,
			expUnlinkVolumeErr:              nil,
			expCheckVolumeStateUseErr:       nil,
			expReconcileDeleteVolumeErr:     nil,
		},
		{
			name:                            "delete two volumes (first time reconcile loop)",
			clusterSpec:                     defaultClusterReconcile,
			machineSpec:                     defaultVolumeReconcile,
			expVolumeFound:                  true,
			expValidateVolumeIdsErr:         nil,
			expDeleteVolumeErr:              nil,
			expCheckVolumeStateAvailableErr: nil,
			expUnlinkVolumeErr:              nil,
			expCheckVolumeStateUseErr:       nil,
			expReconcileDeleteVolumeErr:     nil,
		},
		{
			name:                            "failed to delete volume",
			clusterSpec:                     defaultClusterReconcile,
			machineSpec:                     defaultVolumeReconcile,
			expVolumeFound:                  true,
			expValidateVolumeIdsErr:         nil,
			expDeleteVolumeErr:              fmt.Errorf("DeleteVolume generic error"),
			expCheckVolumeStateAvailableErr: nil,
			expUnlinkVolumeErr:              nil,
			expCheckVolumeStateUseErr:       nil,
			expReconcileDeleteVolumeErr:     fmt.Errorf("DeleteVolume generic error Can not delete volume for OscMachine test-system/test-osc"),
		},
	}
	for _, vtc := range volumeTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			_, machineScope, ctx, mockOscVolumeInterface, _ := SetupWithVolumeMock(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			volumesSpec := vtc.machineSpec.Node.Volumes
			volumeStateAvailable := "available"
			volumeStateUse := "in-use"

			var clockInsideLoop time.Duration = 5
			var clockLoop time.Duration = 60
			var volumesIds []string
			for _, volumeSpec := range volumesSpec {
				volumeName := volumeSpec.Name + "-uid"
				volumeId := "volume-" + volumeName
				volumesIds = append(volumesIds, volumeId)
				mockOscVolumeInterface.
					EXPECT().
					DeleteVolume(gomock.Eq(volumeId)).
					Return(vtc.expDeleteVolumeErr)
				mockOscVolumeInterface.
					EXPECT().
					CheckVolumeState(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(volumeStateUse), gomock.Eq(volumeId)).
					Return(vtc.expCheckVolumeStateUseErr)

				mockOscVolumeInterface.
					EXPECT().
					UnlinkVolume(gomock.Eq(volumeId)).
					Return(vtc.expUnlinkVolumeErr)

				mockOscVolumeInterface.
					EXPECT().
					CheckVolumeState(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(volumeStateAvailable), gomock.Eq(volumeId)).
					Return(vtc.expCheckVolumeStateAvailableErr)
			}
			if vtc.expVolumeFound {
				mockOscVolumeInterface.
					EXPECT().
					ValidateVolumeIds(gomock.Eq(volumesIds)).
					Return(volumesIds, vtc.expValidateVolumeIdsErr)
			} else {
				if len(volumesIds) == 0 {
					volumesIds = []string{""}
				}
				mockOscVolumeInterface.
					EXPECT().
					ValidateVolumeIds(gomock.Eq(volumesIds)).
					Return(nil, vtc.expValidateVolumeIdsErr)
			}

			reconcileDeleteVolume, err := reconcileDeleteVolume(ctx, machineScope, mockOscVolumeInterface)
			if err != nil {
				assert.Equal(t, vtc.expReconcileDeleteVolumeErr.Error(), err.Error(), "reconcileDeleteVolume() should return the same error")
			} else {
				assert.Nil(t, vtc.expReconcileDeleteVolumeErr)
			}
			t.Logf("Find reconcileDeleteVolume %v\n", reconcileDeleteVolume)
		})
	}
}

// TestReconcileDeleteVolumeGet has several tests to cover the code of the function reconcileDeleteVolume
func TestReconcileDeleteVolumeGet(t *testing.T) {
	volumeTestCases := []struct {
		name                        string
		clusterSpec                 infrastructurev1beta1.OscClusterSpec
		machineSpec                 infrastructurev1beta1.OscMachineSpec
		expVolumeFound              bool
		expValidateVolumeIdsErr     error
		expReconcileDeleteVolumeErr error
	}{
		{
			name:        "check work without volume spec (with default values)",
			clusterSpec: defaultClusterReconcile,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{},
			},
			expVolumeFound:              false,
			expValidateVolumeIdsErr:     nil,
			expReconcileDeleteVolumeErr: nil,
		},
		{
			name:                        "failed to validate volume",
			clusterSpec:                 defaultClusterReconcile,
			machineSpec:                 defaultVolumeReconcile,
			expVolumeFound:              true,
			expValidateVolumeIdsErr:     fmt.Errorf("ValidateVolumeIds generic errors"),
			expReconcileDeleteVolumeErr: fmt.Errorf("ValidateVolumeIds generic errors"),
		},
		{
			name:                        "remove finalizer (user delete volume without cluster-api)",
			clusterSpec:                 defaultClusterReconcile,
			machineSpec:                 defaultVolumeReconcile,
			expVolumeFound:              false,
			expValidateVolumeIdsErr:     nil,
			expReconcileDeleteVolumeErr: nil,
		},
	}

	for _, vtc := range volumeTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			_, machineScope, ctx, mockOscVolumeInterface, _ := SetupWithVolumeMock(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			volumesSpec := vtc.machineSpec.Node.Volumes
			var volumesIds []string
			for _, volumeSpec := range volumesSpec {
				volumeName := volumeSpec.Name + "-uid"
				volumeId := "volume-" + volumeName
				volumesIds = append(volumesIds, volumeId)
			}
			if vtc.expVolumeFound {
				mockOscVolumeInterface.
					EXPECT().
					ValidateVolumeIds(gomock.Eq(volumesIds)).
					Return(volumesIds, vtc.expValidateVolumeIdsErr)
			} else {
				if len(volumesIds) == 0 {
					volumesIds = []string{""}
				}
				mockOscVolumeInterface.
					EXPECT().
					ValidateVolumeIds(gomock.Eq(volumesIds)).
					Return(nil, vtc.expValidateVolumeIdsErr)
			}

			reconcileDeleteVolume, err := reconcileDeleteVolume(ctx, machineScope, mockOscVolumeInterface)
			if err != nil {
				assert.Equal(t, vtc.expReconcileDeleteVolumeErr.Error(), err.Error(), "reconcileDeleteVolume() should return the same error")
			} else {
				assert.Nil(t, vtc.expReconcileDeleteVolumeErr)
			}
			t.Logf("Find reconcileDeleteVolume %v\n", reconcileDeleteVolume)
		})
	}
}

// TestReconcileDeleteVolumeWithoutSpec has several tests to cover the code of the function reconcileDeleteVolume
func TestReconcileDeleteVolumeWithoutSpec(t *testing.T) {
	volumeTestCases := []struct {
		name                            string
		clusterSpec                     infrastructurev1beta1.OscClusterSpec
		machineSpec                     infrastructurev1beta1.OscMachineSpec
		expValidateVolumeIdsErr         error
		expDeleteVolumeErr              error
		expCheckVolumeStateAvailableErr error
		expUnlinkVolumeErr              error
		expCheckVolumeStateUseErr       error
		expReconcileDeleteVolumeErr     error
	}{
		{
			name:                            "delete volume without spec",
			clusterSpec:                     defaultClusterReconcile,
			machineSpec:                     defaultVolumeReconcile,
			expDeleteVolumeErr:              nil,
			expCheckVolumeStateAvailableErr: nil,
			expUnlinkVolumeErr:              nil,
			expCheckVolumeStateUseErr:       nil,
			expReconcileDeleteVolumeErr:     nil,
		},
	}

	for _, vtc := range volumeTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			_, machineScope, ctx, mockOscVolumeInterface, _ := SetupWithVolumeMock(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			var volumesIds []string
			var clockInsideLoop time.Duration = 5
			var clockLoop time.Duration = 60
			volumeStateUse := "in-use"

			volumeStateAvailable := "available"
			volumeName := "cluster-api-volume-uid"
			volumeId := "volume-" + volumeName
			volumesIds = append(volumesIds, volumeId)
			mockOscVolumeInterface.
				EXPECT().
				DeleteVolume(gomock.Eq(volumeId)).
				Return(vtc.expDeleteVolumeErr)
			mockOscVolumeInterface.
				EXPECT().
				ValidateVolumeIds(gomock.Eq(volumesIds)).
				Return(volumesIds, vtc.expValidateVolumeIdsErr)

			mockOscVolumeInterface.
				EXPECT().
				CheckVolumeState(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(volumeStateUse), gomock.Eq(volumeId)).
				Return(vtc.expCheckVolumeStateUseErr)

			mockOscVolumeInterface.
				EXPECT().
				UnlinkVolume(gomock.Eq(volumeId)).
				Return(vtc.expUnlinkVolumeErr)

			mockOscVolumeInterface.
				EXPECT().
				CheckVolumeState(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(volumeStateAvailable), gomock.Eq(volumeId)).
				Return(vtc.expCheckVolumeStateAvailableErr)

			nodeSpec := vtc.machineSpec.Node
			nodeSpec.SetVolumeDefaultValue()
			machineScope.OscMachine.Spec.Node.Volumes[0].ResourceId = volumeId
			reconcileDeleteVolume, err := reconcileDeleteVolume(ctx, machineScope, mockOscVolumeInterface)
			if err != nil {
				assert.Equal(t, vtc.expReconcileDeleteVolumeErr.Error(), err.Error(), "reconcileDeleteVolume() should return the same error")
			} else {
				assert.Nil(t, vtc.expReconcileDeleteVolumeErr)
			}
			t.Logf("Find reconcileDeleteVolume %v\n", reconcileDeleteVolume)
		})
	}
}

// TestReconcileDeleteVolumeUnlink has several tests to cover the code of the function reconcileDeleteVolume
func TestReconcileDeleteVolumeUnlink(t *testing.T) {
	volumeTestCases := []struct {
		name                              string
		clusterSpec                       infrastructurev1beta1.OscClusterSpec
		machineSpec                       infrastructurev1beta1.OscMachineSpec
		expVolumeFound                    bool
		expValidateVolumeIdsErr           error
		expCheckVolumeStateAvailableFound bool
		expUnlinkVolumeFound              bool
		expCheckVolumeStateUseFound       bool
		expCheckVolumeStateAvailableErr   error
		expUnlinkVolumeErr                error
		expCheckVolumeStateUseErr         error
		expDeleteVolumeErr                error
		expDeleteVolumeFound              bool
		expReconcileDeleteVolumeErr       error
	}{
		{
			name:        "failed VmVolumeStateUse",
			clusterSpec: defaultClusterReconcile,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Volumes: []*infrastructurev1beta1.OscVolume{
						{
							Name:          "test-volume",
							Iops:          1000,
							Size:          50,
							VolumeType:    "io1",
							SubregionName: "eu-west-2a",
							ResourceId:    "volume-test-volume-uid",
						},
					},
				},
			},
			expVolumeFound:                    true,
			expValidateVolumeIdsErr:           nil,
			expDeleteVolumeErr:                fmt.Errorf("DeleteVolume generic error"),
			expCheckVolumeStateAvailableErr:   nil,
			expCheckVolumeStateAvailableFound: false,
			expUnlinkVolumeErr:                nil,
			expUnlinkVolumeFound:              false,
			expCheckVolumeStateUseErr:         fmt.Errorf("VolumeState generic error"),
			expCheckVolumeStateUseFound:       true,
			expDeleteVolumeFound:              false,
			expReconcileDeleteVolumeErr:       fmt.Errorf("VolumeState generic error Can not get volume volume-test-volume-uid in use for OscMachine test-system/test-osc"),
		},
		{
			name:        "failed to unlink volume",
			clusterSpec: defaultClusterReconcile,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Volumes: []*infrastructurev1beta1.OscVolume{
						{
							Name:          "test-volume",
							Iops:          1000,
							Size:          50,
							VolumeType:    "io1",
							SubregionName: "eu-west-2a",
							ResourceId:    "volume-test-volume-uid",
						},
					},
				},
			},
			expVolumeFound:                    true,
			expValidateVolumeIdsErr:           nil,
			expDeleteVolumeErr:                nil,
			expCheckVolumeStateAvailableErr:   nil,
			expCheckVolumeStateAvailableFound: false,
			expUnlinkVolumeErr:                fmt.Errorf("UnlinkVolume generic error"),
			expUnlinkVolumeFound:              true,
			expCheckVolumeStateUseErr:         nil,
			expCheckVolumeStateUseFound:       true,
			expDeleteVolumeFound:              false,
			expReconcileDeleteVolumeErr:       fmt.Errorf("UnlinkVolume generic error Can not unlink volume volume-test-volume-uid in use for OscMachine test-system/test-osc"),
		},
		{
			name:        "failed to delete volume",
			clusterSpec: defaultClusterReconcile,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Volumes: []*infrastructurev1beta1.OscVolume{
						{
							Name:          "test-volume",
							Iops:          1000,
							Size:          50,
							VolumeType:    "io1",
							SubregionName: "eu-west-2a",
							ResourceId:    "volume-test-volume-uid",
						},
					},
				},
			},
			expVolumeFound:                    true,
			expValidateVolumeIdsErr:           nil,
			expDeleteVolumeErr:                fmt.Errorf("DeleteVolume generic error"),
			expCheckVolumeStateAvailableErr:   fmt.Errorf("VolumeState generic error"),
			expCheckVolumeStateAvailableFound: true,
			expUnlinkVolumeErr:                nil,
			expUnlinkVolumeFound:              true,
			expCheckVolumeStateUseErr:         nil,
			expCheckVolumeStateUseFound:       true,
			expDeleteVolumeFound:              false,
			expReconcileDeleteVolumeErr:       fmt.Errorf("VolumeState generic error Can not get volume volume-test-volume-uid available for OscMachine test-system/test-osc"),
		},
	}

	for _, vtc := range volumeTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			_, machineScope, ctx, mockOscVolumeInterface, _ := SetupWithVolumeMock(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			volumesSpec := vtc.machineSpec.Node.Volumes
			volumeStateAvailable := "available"
			volumeStateUse := "in-use"

			var clockInsideLoop time.Duration = 5
			var clockLoop time.Duration = 60
			var volumesIds []string
			for _, volumeSpec := range volumesSpec {
				volumeName := volumeSpec.Name + "-uid"
				volumeId := "volume-" + volumeName
				volumesIds = append(volumesIds, volumeId)
				if vtc.expDeleteVolumeFound {
					mockOscVolumeInterface.
						EXPECT().
						DeleteVolume(gomock.Eq(volumeId)).
						Return(vtc.expDeleteVolumeErr)
				}
				if vtc.expCheckVolumeStateUseFound {
					mockOscVolumeInterface.
						EXPECT().
						CheckVolumeState(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(volumeStateUse), gomock.Eq(volumeId)).
						Return(vtc.expCheckVolumeStateUseErr)
				}
				if vtc.expUnlinkVolumeFound {
					mockOscVolumeInterface.
						EXPECT().
						UnlinkVolume(gomock.Eq(volumeId)).
						Return(vtc.expUnlinkVolumeErr)
				}
				if vtc.expCheckVolumeStateAvailableFound {
					mockOscVolumeInterface.
						EXPECT().
						CheckVolumeState(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(volumeStateAvailable), gomock.Eq(volumeId)).
						Return(vtc.expCheckVolumeStateAvailableErr)
				}
				if vtc.expVolumeFound {
					mockOscVolumeInterface.
						EXPECT().
						ValidateVolumeIds(gomock.Eq(volumesIds)).
						Return(volumesIds, vtc.expValidateVolumeIdsErr)
				} else {
					if len(volumesIds) == 0 {
						volumesIds = []string{""}
					}
					mockOscVolumeInterface.
						EXPECT().
						ValidateVolumeIds(gomock.Eq(volumesIds)).
						Return(nil, vtc.expValidateVolumeIdsErr)
				}

				reconcileDeleteVolume, err := reconcileDeleteVolume(ctx, machineScope, mockOscVolumeInterface)
				if err != nil {
					assert.Equal(t, vtc.expReconcileDeleteVolumeErr.Error(), err.Error(), "reconcileDeleteVolume() should return the same error")
				} else {
					assert.Nil(t, vtc.expReconcileDeleteVolumeErr)
				}
				t.Logf("Find reconcileDeleteVolume %v\n", reconcileDeleteVolume)
			}
		})
	}
}
