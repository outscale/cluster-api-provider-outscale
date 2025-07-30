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
			expSecurityGroupFound:             true,
			expLoadBalancerSecurityGroupFound: true,
			expTagFound:                       false,
			expReadTagErr:                     nil,
			expReconcileVmErr:                 fmt.Errorf("test-subnet-uid does not exist"),
		},
		{
			name:                              "PublicIp does not exist ",
			clusterSpec:                       defaultVmClusterInitialize,
			machineSpec:                       defaultVmInitialize,
			expVolumeFound:                    true,
			expSubnetFound:                    true,
			expPublicIpFound:                  false,
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
			expSecurityGroupFound:             false,
			expLoadBalancerSecurityGroupFound: false,
			expTagFound:                       false,
			expReadTagErr:                     nil,
			expReconcileVmErr:                 fmt.Errorf("test-securitygroup-uid does not exist"),
		},
		{
			name:                              "failed to get tag",
			clusterSpec:                       defaultVmClusterInitialize,
			machineSpec:                       defaultVmInitialize,
			expVolumeFound:                    true,
			expSubnetFound:                    true,
			expPublicIpFound:                  true,
			expSecurityGroupFound:             true,
			expLoadBalancerSecurityGroupFound: true,
			expTagFound:                       true,
			expReadTagErr:                     fmt.Errorf("ReadTag generic error"),
			expReconcileVmErr:                 fmt.Errorf("ReadTag generic error Can not get tag for OscMachine test-system/test-osc"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope, ctx, mockOscVmInterface, _, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface, mockOscTagInterface := SetupWithVmMock(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
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

			var privateIps []string
			vmPrivateIps := machineScope.GetVmPrivateIps()
			for _, vmPrivateIp := range *vmPrivateIps {
				privateIp := vmPrivateIp.PrivateIp
				privateIps = append(privateIps, privateIp)
			}

			tag := osc.Tag{
				ResourceId: &vmId,
			}
			if vtc.expVolumeFound && vtc.expSubnetFound && vtc.expPublicIpFound && vtc.expSecurityGroupFound {
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

			vmSecurityGroups := machineScope.GetVmSecurityGroups()
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			for _, vmSecurityGroup := range *vmSecurityGroups {
				securityGroupName := vmSecurityGroup.Name + "-uid"
				securityGroupId := "sg-" + securityGroupName
				if vtc.expSecurityGroupFound {
					securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
				}
			}

			loadBalancerSpec := clusterScope.GetLoadBalancer()
			loadBalancerSpec.SetDefaultValue()
			loadBalancerSecurityGroupName := loadBalancerSpec.SecurityGroupName
			loadBalancerSecurityGroupClusterScopeName := loadBalancerSecurityGroupName + "-uid"
			loadBalancerSecurityGroupId := "sg-" + loadBalancerSecurityGroupClusterScopeName
			if vtc.expLoadBalancerSecurityGroupFound {
				securityGroupsRef.ResourceMap[loadBalancerSecurityGroupClusterScopeName] = loadBalancerSecurityGroupId
			}

			reconcileVm, err := reconcileVm(ctx, clusterScope, machineScope, mockOscVmInterface, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface, mockOscTagInterface)
			if err != nil {
				assert.Equal(t, vtc.expReconcileVmErr.Error(), err.Error(), "reconcileVm() should return the same error")
			} else {
				assert.Nil(t, vtc.expReconcileVmErr)
			}
			t.Logf("find reconcileVm %v\n", reconcileVm)
		})
	}
}
