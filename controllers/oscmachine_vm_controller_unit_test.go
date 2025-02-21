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
	"errors"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/compute/mock_compute"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/security/mock_security"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/service/mock_service"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/storage/mock_storage"
	"github.com/outscale/cluster-api-provider-outscale/cloud/tag/mock_tag"
	osc "github.com/outscale/osc-sdk-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	failureDomainSubnet        = "test-failure-domain-subnet"
	failureDomainSubregion     = "test-failure-domain-subregion"
	defaultVmClusterInitialize = infrastructurev1beta1.OscClusterSpec{
		Network: infrastructurev1beta1.OscNetwork{
			Net: infrastructurev1beta1.OscNet{
				Name:        "test-net",
				IpRange:     "10.0.0.0/16",
				ClusterName: "test-cluster",
			},
			Subnets: []*infrastructurev1beta1.OscSubnet{
				{
					Name:          "test-subnet",
					IpSubnetRange: "10.0.0.0/24",
					SubregionName: "eu-west-2a",
				},
			},
			SecurityGroups: []*infrastructurev1beta1.OscSecurityGroup{
				{
					Name:        "test-securitygroup",
					Description: "test securitygroup",
					SecurityGroupRules: []infrastructurev1beta1.OscSecurityGroupRule{
						{
							Name:          "test-securitygrouprule",
							Flow:          "Inbound",
							IpProtocol:    "tcp",
							IpRange:       "0.0.0.0/0",
							FromPortRange: 6443,
							ToPortRange:   6443,
						},
					},
				},
			},
			LoadBalancer: infrastructurev1beta1.OscLoadBalancer{
				LoadBalancerName:  "test-loadbalancer",
				LoadBalancerType:  "internet-facing",
				SubnetName:        "test-subnet",
				SecurityGroupName: "test-securitygroup",
			},
			PublicIps: []*infrastructurev1beta1.OscPublicIp{
				{
					Name: "test-publicip",
				},
			},
		},
	}

	defaultVmClusterReconcile = infrastructurev1beta1.OscClusterSpec{
		Network: infrastructurev1beta1.OscNetwork{
			Net: infrastructurev1beta1.OscNet{
				Name:        "test-net",
				IpRange:     "10.0.0.0/16",
				ClusterName: "test-cluster",
				ResourceId:  "vpc-test-net-uid",
			},
			Subnets: []*infrastructurev1beta1.OscSubnet{
				{
					Name:          "test-subnet",
					IpSubnetRange: "10.0.0.0/24",
					SubregionName: "eu-west-2a",
					ResourceId:    "subnet-test-subnet-uid",
				},
			},
			SecurityGroups: []*infrastructurev1beta1.OscSecurityGroup{
				{
					Name:        "test-securitygroup",
					Description: "test securitygroup",
					ResourceId:  "sg-test-securitygroup-uid",
					SecurityGroupRules: []infrastructurev1beta1.OscSecurityGroupRule{
						{
							Name:          "test-securitygrouprule",
							Flow:          "Inbound",
							IpProtocol:    "tcp",
							IpRange:       "0.0.0.0/0",
							FromPortRange: 6443,
							ToPortRange:   6443,
						},
					},
				},
			},
			LoadBalancer: infrastructurev1beta1.OscLoadBalancer{
				LoadBalancerName:  "test-loadbalancer",
				LoadBalancerType:  "internet-facing",
				SubnetName:        "test-subnet",
				SecurityGroupName: "test-securitygroup",
			},
			PublicIps: []*infrastructurev1beta1.OscPublicIp{
				{
					Name:       "test-publicip",
					ResourceId: "test-publicip-uid",
				},
			},
		},
	}
	failureDomainClusterInitialize = infrastructurev1beta1.OscClusterSpec{
		Network: infrastructurev1beta1.OscNetwork{
			Net: infrastructurev1beta1.OscNet{
				Name:        "test-net",
				IpRange:     "10.0.0.0/16",
				ClusterName: "test-cluster",
			},
			Subnets: []*infrastructurev1beta1.OscSubnet{
				{
					Name:          "test-subnet",
					IpSubnetRange: "10.0.0.0/24",
					SubregionName: "eu-west-2a",
				},
				{
					Name:          failureDomainSubnet,
					IpSubnetRange: "10.0.0.0/24",
					SubregionName: failureDomainSubregion,
				},
			},
			SecurityGroups: []*infrastructurev1beta1.OscSecurityGroup{
				{
					Name:        "test-securitygroup",
					Description: "test securitygroup",
					SecurityGroupRules: []infrastructurev1beta1.OscSecurityGroupRule{
						{
							Name:          "test-securitygrouprule",
							Flow:          "Inbound",
							IpProtocol:    "tcp",
							IpRange:       "0.0.0.0/0",
							FromPortRange: 6443,
							ToPortRange:   6443,
						},
					},
				},
			},
			LoadBalancer: infrastructurev1beta1.OscLoadBalancer{
				LoadBalancerName:  "test-loadbalancer",
				LoadBalancerType:  "internet-facing",
				SubnetName:        "test-subnet",
				SecurityGroupName: "test-securitygroup",
			},
			PublicIps: []*infrastructurev1beta1.OscPublicIp{
				{
					Name: "test-publicip",
				},
			},
		},
	}
	defaultVmVolumeInitialize = infrastructurev1beta1.OscMachineSpec{
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
			Vm: infrastructurev1beta1.OscVm{
				Name:             "test-vm",
				ImageId:          "ami-00000000",
				Role:             "controlplane",
				DeviceName:       "/dev/sda1",
				VolumeName:       "test-volume",
				VolumeDeviceName: "/dev/xvdb",
				RootDisk: infrastructurev1beta1.OscRootDisk{
					RootDiskSize: 300,
					RootDiskIops: 1500,
					RootDiskType: "io1",
				},
				KeypairName:      "rke",
				SubregionName:    "eu-west-2a",
				SubnetName:       "test-subnet",
				LoadBalancerName: "test-loadbalancer",
				PublicIpName:     "test-publicip",
				VmType:           "tinav3.c2r4p2",
				Replica:          1,
				SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
					{
						Name: "test-securitygroup",
					},
				},
				PrivateIps: []infrastructurev1beta1.OscPrivateIpElement{
					{
						Name:      "test-privateip",
						PrivateIp: "10.0.0.17",
					},
				},
			},
		},
	}
	defaultVmVolumeNotAvaiInitialize = infrastructurev1beta1.OscMachineSpec{
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
			Vm: infrastructurev1beta1.OscVm{
				Name:             "test-vm",
				ImageId:          "ami-00000000",
				Role:             "controlplane",
				VolumeName:       "test-volume",
				KeypairName:      "rke",
				SubregionName:    "eu-west-2a",
				SubnetName:       "test-subnet",
				LoadBalancerName: "test-loadbalancer",
				PublicIpName:     "test-publicip",
				VmType:           "tinav3.c2r4p2",
				Replica:          1,
				SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
					{
						Name: "test-securitygroup",
					},
				},
				PrivateIps: []infrastructurev1beta1.OscPrivateIpElement{
					{
						Name:      "test-privateip",
						PrivateIp: "10.0.0.17",
					},
				},
			},
		},
	}
	defaultVmInitialize = infrastructurev1beta1.OscMachineSpec{
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
			Vm: infrastructurev1beta1.OscVm{
				ClusterName: "test-cluster",
				Name:        "test-vm",
				ImageId:     "ami-00000000",
				Role:        "controlplane",
				DeviceName:  "/dev/sda1",
				RootDisk: infrastructurev1beta1.OscRootDisk{
					RootDiskSize: 30,
					RootDiskIops: 1500,
					RootDiskType: "gp2",
				},
				KeypairName:      "rke",
				SubregionName:    "eu-west-2a",
				SubnetName:       "test-subnet",
				LoadBalancerName: "test-loadbalancer",
				PublicIpName:     "test-publicip",
				VmType:           "tinav3.c2r4p2",
				Tags:             map[string]string{"key1": "value1"},
				Replica:          1,
				SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
					{
						Name: "test-securitygroup",
					},
				},
				PrivateIps: []infrastructurev1beta1.OscPrivateIpElement{
					{
						Name:      "test-privateip",
						PrivateIp: "10.0.0.17",
					},
				},
			},
		},
	}
	defaultFailureDomainVmInitialize = infrastructurev1beta1.OscMachineSpec{
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
			Vm: infrastructurev1beta1.OscVm{
				ClusterName: "test-cluster",
				Name:        "test-vm",
				ImageId:     "ami-00000000",
				Role:        "controlplane",
				DeviceName:  "/dev/sda1",
				RootDisk: infrastructurev1beta1.OscRootDisk{
					RootDiskSize: 30,
					RootDiskIops: 1500,
					RootDiskType: "gp2",
				},
				KeypairName:      "rke",
				SubregionName:    "",
				SubnetName:       "",
				LoadBalancerName: "test-loadbalancer",
				PublicIpName:     "test-publicip",
				VmType:           "tinav3.c2r4p2",
				Replica:          1,
				SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
					{
						Name: "test-securitygroup",
					},
				},
				PrivateIps: []infrastructurev1beta1.OscPrivateIpElement{
					{
						Name:      "test-privateip",
						PrivateIp: "10.0.0.17",
					},
				},
			},
		},
	}

	defaultVmInitializeWithoutPublicIp = infrastructurev1beta1.OscMachineSpec{
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
			Vm: infrastructurev1beta1.OscVm{
				ClusterName: "test-cluster",
				Name:        "test-vm",
				ImageId:     "ami-00000000",
				Role:        "controlplane",
				DeviceName:  "/dev/sda1",
				RootDisk: infrastructurev1beta1.OscRootDisk{
					RootDiskSize: 30,
					RootDiskIops: 1500,
					RootDiskType: "gp2",
				},
				KeypairName:      "rke",
				SubregionName:    "eu-west-2a",
				SubnetName:       "test-subnet",
				LoadBalancerName: "test-loadbalancer",
				VmType:           "tinav3.c2r4p2",
				Replica:          1,
				SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
					{
						Name: "test-securitygroup",
					},
				},
				PrivateIps: []infrastructurev1beta1.OscPrivateIpElement{
					{
						Name:      "test-privateip",
						PrivateIp: "10.0.0.17",
					},
				},
			},
		},
	}

	defaultMultiVmInitialize = infrastructurev1beta1.OscMachineSpec{
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
			Vm: infrastructurev1beta1.OscVm{
				ClusterName: "test-cluster",
				Name:        "test-vm",
				ImageId:     "ami-00000000",
				Role:        "controlplane",
				DeviceName:  "/dev/sda1",
				KeypairName: "rke",
				RootDisk: infrastructurev1beta1.OscRootDisk{
					RootDiskSize: 30,
					RootDiskIops: 1500,
					RootDiskType: "io1",
				},
				SubregionName:    "eu-west-2a",
				SubnetName:       "test-subnet",
				LoadBalancerName: "test-loadbalancer",
				VmType:           "tinav3.c2r4p2",
				PublicIpName:     "test-publicip",
				Replica:          1,
				SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
					{
						Name: "test-securitygroup",
					},
				},
				PrivateIps: []infrastructurev1beta1.OscPrivateIpElement{
					{
						Name:      "test-privateip-first",
						PrivateIp: "10.0.0.17",
					},
					{
						Name:      "test-privateip-second",
						PrivateIp: "10.0.0.18",
					},
				},
			},
		},
	}
	defaultVmReconcile = infrastructurev1beta1.OscMachineSpec{
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
			Vm: infrastructurev1beta1.OscVm{
				ClusterName: "test-cluster",
				Name:        "test-vm",
				ImageId:     "ami-00000000",
				Role:        "controlplane",
				DeviceName:  "/dev/xvdb",
				KeypairName: "rke",
				RootDisk: infrastructurev1beta1.OscRootDisk{

					RootDiskSize: 30,
					RootDiskIops: 1500,
					RootDiskType: "io1",
				},
				SubregionName:    "eu-west-2a",
				SubnetName:       "test-subnet",
				LoadBalancerName: "test-loadbalancer",
				VmType:           "tinav3.c2r4p2",
				ResourceId:       "i-test-vm-uid",
				PublicIpName:     "test-publicip",
				Replica:          1,
				SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
					{
						Name: "test-securitygroup",
					},
				},
				PrivateIps: []infrastructurev1beta1.OscPrivateIpElement{
					{
						Name:      "test-privateip",
						PrivateIp: "10.0.0.17",
					},
				},
			},
		},
	}
	defaultVmReconcileWithDedicatedIp = infrastructurev1beta1.OscMachineSpec{
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
			Vm: infrastructurev1beta1.OscVm{
				ClusterName: "test-cluster",
				Name:        "test-vm",
				ImageId:     "ami-00000000",
				Role:        "controlplane",
				DeviceName:  "/dev/xvdb",
				KeypairName: "rke",
				RootDisk: infrastructurev1beta1.OscRootDisk{

					RootDiskSize: 30,
					RootDiskIops: 1500,
					RootDiskType: "io1",
				},
				SubregionName:    "eu-west-2a",
				SubnetName:       "test-subnet",
				LoadBalancerName: "test-loadbalancer",
				VmType:           "tinav3.c2r4p2",
				ResourceId:       "i-test-vm-uid",
				PublicIp:         true,
				Replica:          1,
				SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
					{
						Name: "test-securitygroup",
					},
				},
				PrivateIps: []infrastructurev1beta1.OscPrivateIpElement{
					{
						Name:      "test-privateip",
						PrivateIp: "10.0.0.17",
					},
				},
			},
		},
	}
)

// SetupWithVmMock set vmMock with clusterScope, machineScope and oscmachine
func SetupWithVmMock(t *testing.T, name string, clusterSpec infrastructurev1beta1.OscClusterSpec, machineSpec infrastructurev1beta1.OscMachineSpec) (clusterScope *scope.ClusterScope, machineScope *scope.MachineScope, ctx context.Context, mockOscVmInterface *mock_compute.MockOscVmInterface, mockOscVolumeInterface *mock_storage.MockOscVolumeInterface, mockOscPublicIpInterface *mock_security.MockOscPublicIpInterface, mockOscLoadBalancerInterface *mock_service.MockOscLoadBalancerInterface, mockOscSecurityGroupInterface *mock_security.MockOscSecurityGroupInterface, mockOscTagInterface *mock_tag.MockOscTagInterface) {
	clusterScope, machineScope = SetupMachine(t, name, clusterSpec, machineSpec)
	mockCtrl := gomock.NewController(t)
	mockOscVmInterface = mock_compute.NewMockOscVmInterface(mockCtrl)
	mockOscVolumeInterface = mock_storage.NewMockOscVolumeInterface(mockCtrl)
	mockOscPublicIpInterface = mock_security.NewMockOscPublicIpInterface(mockCtrl)
	mockOscLoadBalancerInterface = mock_service.NewMockOscLoadBalancerInterface(mockCtrl)
	mockOscSecurityGroupInterface = mock_security.NewMockOscSecurityGroupInterface(mockCtrl)
	mockOscTagInterface = mock_tag.NewMockOscTagInterface(mockCtrl)
	ctx = context.Background()
	return clusterScope, machineScope, ctx, mockOscVmInterface, mockOscVolumeInterface, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface, mockOscTagInterface
}

// TestGetVmResourceId has several tests to cover the code of the function getVmResourceId
func TestGetVmResourceId(t *testing.T) {
	vmTestCases := []struct {
		name                  string
		clusterSpec           infrastructurev1beta1.OscClusterSpec
		machineSpec           infrastructurev1beta1.OscMachineSpec
		expVmFound            bool
		expGetVmResourceIdErr error
	}{
		{
			name:                  "get vm",
			clusterSpec:           defaultVmClusterInitialize,
			machineSpec:           defaultVmInitialize,
			expVmFound:            true,
			expGetVmResourceIdErr: nil,
		},
		{
			name:                  "can not get vm",
			clusterSpec:           defaultVmClusterInitialize,
			machineSpec:           defaultVmInitialize,
			expVmFound:            false,
			expGetVmResourceIdErr: errors.New("test-vm-uid does not exist"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			_, machineScope := SetupMachine(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			vmName := vtc.machineSpec.Node.Vm.Name + "-uid"
			vmId := "i-" + vmName
			vmRef := machineScope.GetVmRef()
			vmRef.ResourceMap = make(map[string]string)
			if vtc.expVmFound {
				vmRef.ResourceMap[vmName] = vmId
			}
			vmResourceId, err := getVmResourceId(vmName, machineScope)
			if vtc.expGetVmResourceIdErr != nil {
				require.EqualError(t, err, vtc.expGetVmResourceIdErr.Error(), "GetVmResourceId() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("find netResourceId %s", vmResourceId)
		})
	}
}

// TestCheckVmVolumeOscAssociateResourceName has several tests to cover the code of the function checkVmVolumeOscAssociateResourceName
func TestCheckVmVolumeOscAssociateResourceName(t *testing.T) {
	vmTestCases := []struct {
		name                                        string
		clusterSpec                                 infrastructurev1beta1.OscClusterSpec
		machineSpec                                 infrastructurev1beta1.OscMachineSpec
		expCheckVmVolumeOscAssociateResourceNameErr error
	}{
		{
			name:        "check volume associate with vm",
			clusterSpec: defaultVmClusterInitialize,
			machineSpec: defaultVmVolumeInitialize,
			expCheckVmVolumeOscAssociateResourceNameErr: nil,
		},
		{
			name:        "check work without vm spec (with default values)",
			clusterSpec: defaultVmClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						VolumeName: "test-volume",
					},
				},
			},
			expCheckVmVolumeOscAssociateResourceNameErr: errors.New("test-volume-uid volume does not exist in vm"),
		},
		{
			name:        "check Bad name volume",
			clusterSpec: defaultVmClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
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
					Vm: infrastructurev1beta1.OscVm{
						Name:        "test-vm",
						ImageId:     "ami-00000000",
						Role:        "controlplane",
						VolumeName:  "test-volume@test",
						DeviceName:  "/dev/sda1",
						KeypairName: "rke",
						RootDisk: infrastructurev1beta1.OscRootDisk{

							RootDiskSize: 30,
							RootDiskIops: 1500,
							RootDiskType: "io1",
						},
						SubregionName:    "eu-west-2a",
						SubnetName:       "test-subnet",
						LoadBalancerName: "test-loadbalancer",
						VmType:           "tinav3.c2r4p2",
						PublicIpName:     "test-publicip",
						SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
							{
								Name: "test-securitygroup",
							},
						},
						PrivateIps: []infrastructurev1beta1.OscPrivateIpElement{
							{
								Name:      "test-privateip-first",
								PrivateIp: "10.0.0.17",
							},
							{
								Name:      "test-privateip-second",
								PrivateIp: "10.0.0.18",
							},
						},
					},
				},
			},
			expCheckVmVolumeOscAssociateResourceNameErr: errors.New("test-volume@test-uid volume does not exist in vm"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			_, machineScope := SetupMachine(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			err := checkVmVolumeOscAssociateResourceName(machineScope)
			if vtc.expCheckVmVolumeOscAssociateResourceNameErr != nil {
				require.EqualError(t, err, vtc.expCheckVmVolumeOscAssociateResourceNameErr.Error(), "checkVmVolumeOscAssociateResourceName() should return the same eror")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestCheckVmLoadBalancerOscAssociateResourceName has several tests to cover the code of the function checkVmLoadBalancerOscAssociateResourceName
func TestCheckVmLoadBalancerOscAssociateResourceName(t *testing.T) {
	vmTestCases := []struct {
		name                                              string
		clusterSpec                                       infrastructurev1beta1.OscClusterSpec
		machineSpec                                       infrastructurev1beta1.OscMachineSpec
		expCheckVmLoadBalancerOscAssociateResourceNameErr error
	}{
		{
			name:        "check loadbalancer association with vm",
			clusterSpec: defaultVmClusterInitialize,
			machineSpec: defaultVmInitialize,
			expCheckVmLoadBalancerOscAssociateResourceNameErr: nil,
		},
		{
			name:        "check work without vm spec (with default value)",
			clusterSpec: defaultVmClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						Role: "controlplane",
					},
				},
			},
			expCheckVmLoadBalancerOscAssociateResourceNameErr: errors.New("OscClusterApi-1-uid loadBalancer does not exist in vm"),
		},
		{
			name:        "check Bad loadBalancer name",
			clusterSpec: defaultVmClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
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
					Vm: infrastructurev1beta1.OscVm{
						Name:             "test-vm",
						ImageId:          "ami-00000000",
						Role:             "controlplane",
						VolumeName:       "test-volume",
						DeviceName:       "/dev/sda1",
						KeypairName:      "rke",
						SubregionName:    "eu-west-2a",
						SubnetName:       "test-subnet",
						LoadBalancerName: "test-loadbalancer@test",
						RootDisk: infrastructurev1beta1.OscRootDisk{

							RootDiskSize: 30,
							RootDiskIops: 1500,
							RootDiskType: "io1",
						},
						VmType:       "tinav3.c2r4p2",
						PublicIpName: "test-publicip",
						SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
							{
								Name: "test-securitygroup",
							},
						},
						PrivateIps: []infrastructurev1beta1.OscPrivateIpElement{
							{
								Name:      "test-privateip-first",
								PrivateIp: "10.0.0.17",
							},
							{
								Name:      "test-privateip-second",
								PrivateIp: "10.0.0.18",
							},
						},
					},
				},
			},
			expCheckVmLoadBalancerOscAssociateResourceNameErr: errors.New("test-loadbalancer@test-uid loadBalancer does not exist in vm"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope := SetupMachine(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			err := checkVmLoadBalancerOscAssociateResourceName(machineScope, clusterScope)
			if vtc.expCheckVmLoadBalancerOscAssociateResourceNameErr != nil {
				require.EqualError(t, err, vtc.expCheckVmLoadBalancerOscAssociateResourceNameErr.Error(), "checkVmLoadBalancerOscAssociateResourceName() should return the same erroor")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestCheckVmSecurityGroupOscAssociateResourceName has several tests to cover the code of the function checkVmSecurityGroupOscAssociateResourceNam
func TestCheckVmSecurityGroupOscAssociateResourceName(t *testing.T) {
	vmTestCases := []struct {
		name                                               string
		clusterSpec                                        infrastructurev1beta1.OscClusterSpec
		machineSpec                                        infrastructurev1beta1.OscMachineSpec
		expCheckVmSecurityGroupOscAssociateResourceNameErr error
	}{
		{
			name:        "check securitygroup association with vm",
			clusterSpec: defaultVmClusterInitialize,
			machineSpec: defaultVmInitialize,
			expCheckVmSecurityGroupOscAssociateResourceNameErr: nil,
		},
		{
			name:        "check work without vm spec (with default value)",
			clusterSpec: defaultVmClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{},
			},
			expCheckVmSecurityGroupOscAssociateResourceNameErr: errors.New("cluster-api-securitygroup-kw-uid securityGroup does not exist in vm"),
		},
		{
			name:        "check Bad security group name",
			clusterSpec: defaultVmClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
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
					Vm: infrastructurev1beta1.OscVm{
						Name:          "test-vm",
						ImageId:       "ami-00000000",
						Role:          "controlplane",
						VolumeName:    "test-volume",
						DeviceName:    "/dev/sda1",
						KeypairName:   "rke",
						SubregionName: "eu-west-2a",
						RootDisk: infrastructurev1beta1.OscRootDisk{

							RootDiskSize: 30,
							RootDiskIops: 1500,
							RootDiskType: "io1",
						},
						SubnetName:       "test-subnet",
						LoadBalancerName: "test-loadbalancer",
						VmType:           "tinav3.c2r4p2",
						PublicIpName:     "test-publicip",
						SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
							{
								Name: "test-securitygroup@test",
							},
						},
						PrivateIps: []infrastructurev1beta1.OscPrivateIpElement{
							{
								Name:      "test-privateip-first",
								PrivateIp: "10.0.0.17",
							},
							{
								Name:      "test-privateip-second",
								PrivateIp: "10.0.0.18",
							},
						},
					},
				},
			},
			expCheckVmSecurityGroupOscAssociateResourceNameErr: errors.New("test-securitygroup@test-uid securityGroup does not exist in vm"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope := SetupMachine(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			err := checkVmSecurityGroupOscAssociateResourceName(machineScope, clusterScope)
			if vtc.expCheckVmSecurityGroupOscAssociateResourceNameErr != nil {
				require.EqualError(t, err, vtc.expCheckVmSecurityGroupOscAssociateResourceNameErr.Error(), "checkVmSecurityGroupOscAssociateResourceName() should return the same error")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestCheckVmPublicIpOscAssociateResourceName has several tests to cover the code of the function checkVmPublicIpOscAssociateResourceName
func TestCheckVmPublicIpOscAssociateResourceName(t *testing.T) {
	vmTestCases := []struct {
		name                                          string
		clusterSpec                                   infrastructurev1beta1.OscClusterSpec
		machineSpec                                   infrastructurev1beta1.OscMachineSpec
		expCheckVmPublicIpOscAssociateResourceNameErr error
	}{
		{
			name:        "check publicip association with vm",
			clusterSpec: defaultVmClusterInitialize,
			machineSpec: defaultVmInitialize,
			expCheckVmPublicIpOscAssociateResourceNameErr: nil,
		},
		{
			name:        "check work without vm spec (with default values)",
			clusterSpec: defaultVmClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						PublicIpName: "cluster-api-publicip",
					},
				},
			},
			expCheckVmPublicIpOscAssociateResourceNameErr: errors.New("cluster-api-publicip-uid publicIp does not exist in vm"),
		},
		{
			name:        "check Bad name publicIp",
			clusterSpec: defaultVmClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
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
					Vm: infrastructurev1beta1.OscVm{
						Name:       "test-vm",
						ImageId:    "ami-00000000",
						Role:       "controlplane",
						VolumeName: "test-volume",
						DeviceName: "/dev/sda1",
						RootDisk: infrastructurev1beta1.OscRootDisk{

							RootDiskSize: 30,
							RootDiskIops: 1500,
							RootDiskType: "io1",
						},
						KeypairName:      "rke",
						SubregionName:    "eu-west-2a",
						SubnetName:       "test-subnet",
						LoadBalancerName: "test-loadbalancer",
						VmType:           "tinav3.c2r4p2",
						PublicIpName:     "test-publicip@test",
						SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
							{
								Name: "test-securitygroup",
							},
						},
						PrivateIps: []infrastructurev1beta1.OscPrivateIpElement{
							{
								Name:      "test-privateip-first",
								PrivateIp: "10.0.0.17",
							},
							{
								Name:      "test-privateip-second",
								PrivateIp: "10.0.0.18",
							},
						},
					},
				},
			},
			expCheckVmPublicIpOscAssociateResourceNameErr: errors.New("test-publicip@test-uid publicIp does not exist in vm"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope := SetupMachine(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			err := checkVmPublicIpOscAssociateResourceName(machineScope, clusterScope)
			if vtc.expCheckVmPublicIpOscAssociateResourceNameErr != nil {
				require.EqualError(t, err, vtc.expCheckVmPublicIpOscAssociateResourceNameErr.Error(), "checkVmPublicIpOscAssociateResourceName() should return the same error")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestCheckVmFormatParameters has several tests to cover the code of the function checkVmFormatParameter
func TestCheckVmFormatParameters(t *testing.T) {
	vmTestCases := []struct {
		name                          string
		clusterSpec                   infrastructurev1beta1.OscClusterSpec
		machineSpec                   infrastructurev1beta1.OscMachineSpec
		expCheckVmFormatParametersErr error
	}{
		{
			name:                          "check vm format",
			clusterSpec:                   defaultVmClusterInitialize,
			machineSpec:                   defaultVmInitialize,
			expCheckVmFormatParametersErr: nil,
		},
		{
			name:        "check Bad name vm",
			clusterSpec: defaultVmClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
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
					Vm: infrastructurev1beta1.OscVm{
						Name:       "test-vm@test",
						ImageId:    "ami-00000000",
						Role:       "controlplane",
						VolumeName: "test-volume",
						DeviceName: "/dev/sda1",
						RootDisk: infrastructurev1beta1.OscRootDisk{

							RootDiskSize: 30,
							RootDiskIops: 1500,
							RootDiskType: "io1",
						},
						KeypairName:      "rke",
						SubregionName:    "eu-west-2a",
						SubnetName:       "test-subnet",
						LoadBalancerName: "test-loadbalancer",
						VmType:           "tinav3.c2r4p2",
						PublicIpName:     "test-publicip",
						SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
							{
								Name: "test-securitygroup",
							},
						},
						PrivateIps: []infrastructurev1beta1.OscPrivateIpElement{
							{
								Name:      "test-privateip-first",
								PrivateIp: "10.0.0.17",
							},
						},
					},
				},
			},
			expCheckVmFormatParametersErr: errors.New("Invalid Tag Name"),
		},
		{
			name:        "check Bad name volumeDeviceName",
			clusterSpec: defaultVmClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
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
					Vm: infrastructurev1beta1.OscVm{
						Name:             "test-vm",
						ImageId:          "ami-00000000",
						Role:             "controlplane",
						VolumeName:       "test-volume",
						VolumeDeviceName: "/dev/xvdaa",
						RootDisk: infrastructurev1beta1.OscRootDisk{

							RootDiskSize: 30,
							RootDiskIops: 1500,
							RootDiskType: "io1",
						},
						KeypairName:      "rke",
						SubregionName:    "eu-west-2a",
						SubnetName:       "test-subnet",
						LoadBalancerName: "test-loadbalancer",
						VmType:           "tinav3.c2r4p2",
						PublicIpName:     "test-publicip",
						SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
							{
								Name: "test-securitygroup",
							},
						},
						PrivateIps: []infrastructurev1beta1.OscPrivateIpElement{
							{
								Name:      "test-privateip-first",
								PrivateIp: "10.0.0.17",
							},
						},
					},
				},
			},
			expCheckVmFormatParametersErr: errors.New("Invalid deviceName"),
		},
		{
			name:        "check Bad imageId",
			clusterSpec: defaultVmClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
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
					Vm: infrastructurev1beta1.OscVm{
						Name:       "test-vm",
						ImageId:    "omi-00000000",
						Role:       "controlplane",
						VolumeName: "test-volume",
						DeviceName: "/dev/sda1",
						RootDisk: infrastructurev1beta1.OscRootDisk{

							RootDiskSize: 30,
							RootDiskIops: 1500,
							RootDiskType: "io1",
						},
						KeypairName:      "rke",
						SubregionName:    "eu-west-2a",
						SubnetName:       "test-subnet",
						LoadBalancerName: "test-loadbalancer",
						VmType:           "tinav3.c2r4p2",
						PublicIpName:     "test-publicip",
						SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
							{
								Name: "test-securitygroup",
							},
						},
						PrivateIps: []infrastructurev1beta1.OscPrivateIpElement{
							{
								Name:      "test-privateip-first",
								PrivateIp: "10.0.0.17",
							},
						},
					},
				},
			},
			expCheckVmFormatParametersErr: errors.New("Invalid imageId"),
		},
		{
			name:        "check bad image Name",
			clusterSpec: defaultVmClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
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
					Image: infrastructurev1beta1.OscImage{
						Name: "!test-image@Name",
					},
					Vm: infrastructurev1beta1.OscVm{
						Name:       "test-vm",
						Role:       "controlplane",
						VolumeName: "test-volume",
						DeviceName: "/dev/sda1",
						RootDisk: infrastructurev1beta1.OscRootDisk{

							RootDiskSize: 30,
							RootDiskIops: 1500,
							RootDiskType: "io1",
						},
						KeypairName:      "rke",
						SubregionName:    "eu-west-2a",
						SubnetName:       "test-subnet",
						LoadBalancerName: "test-loadbalancer",
						VmType:           "tinav3.c2r4p2",
						PublicIpName:     "test-publicip",
						SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
							{
								Name: "test-securitygroup",
							},
						},
						PrivateIps: []infrastructurev1beta1.OscPrivateIpElement{
							{
								Name:      "test-privateip-first",
								PrivateIp: "10.0.0.17",
							},
						},
					},
				},
			},
			expCheckVmFormatParametersErr: errors.New("Invalid Image Name"),
		},
		{
			name:        "check Bad keypairname",
			clusterSpec: defaultVmClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
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
					Vm: infrastructurev1beta1.OscVm{
						Name:       "test-vm",
						ImageId:    "ami-00000000",
						Role:       "controlplane",
						VolumeName: "test-volume",
						DeviceName: "/dev/sda1",
						RootDisk: infrastructurev1beta1.OscRootDisk{

							RootDiskSize: 30,
							RootDiskIops: 1500,
							RootDiskType: "io1",
						},
						KeypairName:      "rke λ",
						SubregionName:    "eu-west-2a",
						SubnetName:       "test-subnet",
						LoadBalancerName: "test-loadbalancer",
						VmType:           "tinav3.c2r4p2",
						PublicIpName:     "test-publicip",
						SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
							{
								Name: "test-securitygroup",
							},
						},
						PrivateIps: []infrastructurev1beta1.OscPrivateIpElement{
							{
								Name:      "test-privateip-first",
								PrivateIp: "10.0.0.17",
							},
						},
					},
				},
			},
			expCheckVmFormatParametersErr: errors.New("Invalid KeypairName"),
		},
		{
			name:        "check empty imageId",
			clusterSpec: defaultVmClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
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
					Image: infrastructurev1beta1.OscImage{
						Name: "omi-000",
					},
					Vm: infrastructurev1beta1.OscVm{
						Name:       "test-vm",
						Role:       "controlplane",
						VolumeName: "test-volume",
						DeviceName: "/dev/sda1",
						RootDisk: infrastructurev1beta1.OscRootDisk{

							RootDiskSize: 300,
							RootDiskIops: 1500,
							RootDiskType: "gp2",
						},
						KeypairName:      "rke",
						SubregionName:    "eu-west-2a",
						SubnetName:       "test-subnet",
						LoadBalancerName: "test-loadbalancer",
						VmType:           "tinav3.c2r4p2",
						PublicIpName:     "test-publicip",
						SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
							{
								Name: "test-securitygroup",
							},
						},
						PrivateIps: []infrastructurev1beta1.OscPrivateIpElement{
							{
								Name:      "test-privateip-first",
								PrivateIp: "10.0.0.17",
							},
						},
					},
				},
			},
			expCheckVmFormatParametersErr: nil,
		},
		{
			name:        "check Bad device name",
			clusterSpec: defaultVmClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
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
					Vm: infrastructurev1beta1.OscVm{
						Name:       "test-vm",
						ImageId:    "ami-00000000",
						Role:       "controlplane",
						VolumeName: "test-volume",
						DeviceName: "/dev/sdab1",
						RootDisk: infrastructurev1beta1.OscRootDisk{

							RootDiskSize: 30,
							RootDiskIops: 1500,
							RootDiskType: "gp2",
						},
						KeypairName:      "rke",
						SubregionName:    "eu-west-2a",
						SubnetName:       "test-subnet",
						LoadBalancerName: "test-loadbalancer",
						VmType:           "tinav3.c2r4p2",
						PublicIpName:     "test-publicip",
						SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
							{
								Name: "test-securitygroup",
							},
						},
						PrivateIps: []infrastructurev1beta1.OscPrivateIpElement{
							{
								Name:      "test-privateip-first",
								PrivateIp: "10.0.0.17",
							},
						},
					},
				},
			},
			expCheckVmFormatParametersErr: errors.New("Invalid deviceName"),
		},
		{
			name:        "check empty device name",
			clusterSpec: defaultVmClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
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
					Vm: infrastructurev1beta1.OscVm{
						Name:       "test-vm",
						ImageId:    "ami-00000000",
						Role:       "controlplane",
						VolumeName: "test-volume",
						RootDisk: infrastructurev1beta1.OscRootDisk{

							RootDiskSize: 300,
							RootDiskIops: 1500,
							RootDiskType: "gp2",
						},
						KeypairName:      "rke",
						SubregionName:    "eu-west-2a",
						SubnetName:       "test-subnet",
						LoadBalancerName: "test-loadbalancer",
						VmType:           "tinav3.c2r4p2",
						PublicIpName:     "test-publicip",
						SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
							{
								Name: "test-securitygroup",
							},
						},
						PrivateIps: []infrastructurev1beta1.OscPrivateIpElement{
							{
								Name:      "test-privateip-first",
								PrivateIp: "10.0.0.17",
							},
						},
					},
				},
			},
			expCheckVmFormatParametersErr: nil,
		},
		{
			name:        "Check Bad VmType",
			clusterSpec: defaultVmClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
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
					Vm: infrastructurev1beta1.OscVm{
						Name:       "test-vm",
						ImageId:    "ami-00000000",
						Role:       "controlplane",
						VolumeName: "test-volume",
						DeviceName: "/dev/sda1",
						RootDisk: infrastructurev1beta1.OscRootDisk{

							RootDiskSize: 30,
							RootDiskIops: 1500,
							RootDiskType: "io1",
						},
						KeypairName:      "rke",
						SubregionName:    "eu-west-2a",
						SubnetName:       "test-subnet",
						LoadBalancerName: "test-loadbalancer",
						VmType:           "awsv4.c2r4p2",
						PublicIpName:     "test-publicip",
						SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
							{
								Name: "test-securitygroup",
							},
						},
						PrivateIps: []infrastructurev1beta1.OscPrivateIpElement{
							{
								Name:      "test-privateip-first",
								PrivateIp: "10.0.0.17",
							},
						},
					},
				},
			},
			expCheckVmFormatParametersErr: errors.New("Invalid vmType"),
		},
		{
			name:        "Check Bad IpAddr",
			clusterSpec: defaultVmClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
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
					Vm: infrastructurev1beta1.OscVm{
						Name:       "test-vm",
						ImageId:    "ami-00000000",
						Role:       "controlplane",
						VolumeName: "test-volume",
						DeviceName: "/dev/sda1",
						RootDisk: infrastructurev1beta1.OscRootDisk{

							RootDiskSize: 30,
							RootDiskIops: 1500,
							RootDiskType: "io1",
						},
						KeypairName:      "rke",
						SubregionName:    "eu-west-2a",
						SubnetName:       "test-subnet",
						LoadBalancerName: "test-loadbalancer",
						VmType:           "tinav3.c2r4p2",
						PublicIpName:     "test-publicip",
						SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
							{
								Name: "test-securitygroup",
							},
						},
						PrivateIps: []infrastructurev1beta1.OscPrivateIpElement{
							{
								Name:      "test-privateip-first",
								PrivateIp: "10.245.0.17",
							},
						},
					},
				},
			},
			expCheckVmFormatParametersErr: errors.New("Invalid ip in cidr"),
		},
		{
			name:        "Check Bad subregionname",
			clusterSpec: defaultVmClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
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
					Vm: infrastructurev1beta1.OscVm{
						Name:       "test-vm",
						ImageId:    "ami-00000000",
						Role:       "controlplane",
						VolumeName: "test-volume",
						DeviceName: "/dev/sda1",
						RootDisk: infrastructurev1beta1.OscRootDisk{

							RootDiskSize: 30,
							RootDiskIops: 1500,
							RootDiskType: "io1",
						},
						KeypairName:      "rke",
						SubregionName:    "eu-west-2d",
						SubnetName:       "test-subnet",
						LoadBalancerName: "test-loadbalancer",
						VmType:           "tinav3.c2r4p2",
						PublicIpName:     "test-publicip",
						SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
							{
								Name: "test-securitygroup",
							},
						},
						PrivateIps: []infrastructurev1beta1.OscPrivateIpElement{
							{
								Name:      "test-privateip-first",
								PrivateIp: "10.0.0.17",
							},
						},
					},
				},
			},
			expCheckVmFormatParametersErr: errors.New("Invalid subregionName"),
		},
		{
			name:        "Check Bad root device size",
			clusterSpec: defaultVmClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
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
					Vm: infrastructurev1beta1.OscVm{
						Name:       "test-vm",
						ImageId:    "ami-00000000",
						Role:       "controlplane",
						VolumeName: "test-volume",
						DeviceName: "/dev/sda1",
						RootDisk: infrastructurev1beta1.OscRootDisk{

							RootDiskSize: -30,
							RootDiskIops: 1500,
							RootDiskType: "io1",
						},
						KeypairName:      "rke",
						SubregionName:    "eu-west-2a",
						SubnetName:       "test-subnet",
						LoadBalancerName: "test-loadbalancer",
						VmType:           "tinav3.c2r4p2",
						PublicIpName:     "test-publicip",
						SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
							{
								Name: "test-securitygroup",
							},
						},
						PrivateIps: []infrastructurev1beta1.OscPrivateIpElement{
							{
								Name:      "test-privateip-first",
								PrivateIp: "10.0.0.17",
							},
						},
					},
				},
			},
			expCheckVmFormatParametersErr: errors.New("Invalid size"),
		},
		{
			name:        "Check Bad rootDeviceIops",
			clusterSpec: defaultVmClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
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
					Vm: infrastructurev1beta1.OscVm{
						Name:       "test-vm",
						ImageId:    "ami-00000000",
						Role:       "controlplane",
						VolumeName: "test-volume",
						DeviceName: "/dev/sda1",
						RootDisk: infrastructurev1beta1.OscRootDisk{

							RootDiskSize: 30,
							RootDiskIops: -15,
							RootDiskType: "io1",
						},
						KeypairName:      "rke",
						SubregionName:    "eu-west-2a",
						SubnetName:       "test-subnet",
						LoadBalancerName: "test-loadbalancer",
						VmType:           "tinav3.c2r4p2",
						PublicIpName:     "test-publicip",
						SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
							{
								Name: "test-securitygroup",
							},
						},
						PrivateIps: []infrastructurev1beta1.OscPrivateIpElement{
							{
								Name:      "test-privateip-first",
								PrivateIp: "10.0.0.17",
							},
						},
					},
				},
			},
			expCheckVmFormatParametersErr: errors.New("Invalid iops"),
		},
		{
			name:        "Check bad rootDiskType",
			clusterSpec: defaultVmClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
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
					Vm: infrastructurev1beta1.OscVm{
						Name:       "test-vm",
						ImageId:    "ami-00000000",
						Role:       "controlplane",
						VolumeName: "test-volume",
						DeviceName: "/dev/sda1",
						RootDisk: infrastructurev1beta1.OscRootDisk{

							RootDiskSize: 30,
							RootDiskIops: 1500,
							RootDiskType: "gp3",
						},
						KeypairName:      "rke",
						SubregionName:    "eu-west-2a",
						SubnetName:       "test-subnet",
						LoadBalancerName: "test-loadbalancer",
						VmType:           "tinav3.c2r4p2",
						PublicIpName:     "test-publicip",
						SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
							{
								Name: "test-securitygroup",
							},
						},
						PrivateIps: []infrastructurev1beta1.OscPrivateIpElement{
							{
								Name:      "test-privateip-first",
								PrivateIp: "10.0.0.17",
							},
						},
					},
				},
			},
			expCheckVmFormatParametersErr: errors.New("Invalid volumeType"),
		},
		{
			name:        "Check Bad ratio root disk Iops Size",
			clusterSpec: defaultVmClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
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
					Vm: infrastructurev1beta1.OscVm{
						Name:       "test-vm",
						ImageId:    "ami-00000000",
						Role:       "controlplane",
						VolumeName: "test-volume",
						DeviceName: "/dev/sda1",
						RootDisk: infrastructurev1beta1.OscRootDisk{

							RootDiskSize: 10,
							RootDiskIops: 3500,
							RootDiskType: "io1",
						},
						KeypairName:      "rke",
						SubregionName:    "eu-west-2a",
						SubnetName:       "test-subnet",
						LoadBalancerName: "test-loadbalancer",
						VmType:           "tinav3.c2r4p2",
						PublicIpName:     "test-publicip",
						SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
							{
								Name: "test-securitygroup",
							},
						},
						PrivateIps: []infrastructurev1beta1.OscPrivateIpElement{
							{
								Name:      "test-privateip-first",
								PrivateIp: "10.0.0.17",
							},
						},
					},
				},
			},
			expCheckVmFormatParametersErr: errors.New("Invalid ratio Iops size that exceed 300"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope := SetupMachine(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			subnetName := vtc.machineSpec.Node.Vm.SubnetName + "-uid"
			subnetId := "subnet-" + subnetName
			subnetRef := clusterScope.GetSubnetRef()
			subnetRef.ResourceMap = make(map[string]string)
			subnetRef.ResourceMap[subnetName] = subnetId
			vmName, err := checkVmFormatParameters(machineScope, clusterScope)
			if vtc.expCheckVmFormatParametersErr != nil {
				require.EqualError(t, err, vtc.expCheckVmFormatParametersErr.Error(), "checkVmFormatParameters() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("find vmName %s\n", vmName)
		})
	}
}

// TestCheckVmSubnetAssociateResourceName has several tests to cover the code of the function checkVmSubnetAssociateResourceName
func TestCheckVmSubnetAssociateResourceName(t *testing.T) {
	vmTestCases := []struct {
		name                                     string
		clusterSpec                              infrastructurev1beta1.OscClusterSpec
		machineSpec                              infrastructurev1beta1.OscMachineSpec
		expCheckVmSubnetAssociateResourceNameErr error
	}{
		{
			name:                                     "check subnet associate with vm",
			clusterSpec:                              defaultVmClusterInitialize,
			machineSpec:                              defaultVmInitialize,
			expCheckVmSubnetAssociateResourceNameErr: nil,
		},
		{
			name:        "check work without vm spec (with default value)",
			clusterSpec: defaultVmClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{},
			},
			expCheckVmSubnetAssociateResourceNameErr: errors.New("cluster-api-subnet-kw-uid subnet does not exist in vm"),
		},
		{
			name:        "check Bad subnet name",
			clusterSpec: defaultVmClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
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
					Vm: infrastructurev1beta1.OscVm{
						Name:       "test-vm",
						ImageId:    "ami-00000000",
						Role:       "controlplane",
						VolumeName: "test-volume",
						DeviceName: "/dev/sda1",
						RootDisk: infrastructurev1beta1.OscRootDisk{

							RootDiskSize: 30,
							RootDiskIops: 1500,
							RootDiskType: "io1",
						},
						KeypairName:      "rke",
						SubregionName:    "eu-west-2a",
						SubnetName:       "test-subnet@test",
						LoadBalancerName: "test-loadbalancer",
						VmType:           "tinav3.c2r4p2",
						PublicIpName:     "test-publicip",
						SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
							{
								Name: "test-securitygroup",
							},
						},
						PrivateIps: []infrastructurev1beta1.OscPrivateIpElement{
							{
								Name:      "test-privateip-first",
								PrivateIp: "10.0.0.17",
							},
							{
								Name:      "test-privateip-second",
								PrivateIp: "10.0.0.18",
							},
						},
					},
				},
			},
			expCheckVmSubnetAssociateResourceNameErr: errors.New("test-subnet@test-uid subnet does not exist in vm"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope := SetupMachine(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			err := checkVmSubnetOscAssociateResourceName(machineScope, clusterScope)
			if vtc.expCheckVmSubnetAssociateResourceNameErr != nil {
				require.EqualError(t, err, vtc.expCheckVmSubnetAssociateResourceNameErr.Error(), "checkVmSubnetOscAssociateResourceName() should return the same error")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestCheckVmPrivateIpsOscDuplicateName has several tests to cover the code of the function checkVmPrivateIpsOscDuplicateName
func TestCheckVmPrivateIpsOscDuplicateName(t *testing.T) {
	vmTestCases := []struct {
		name                                    string
		clusterSpec                             infrastructurev1beta1.OscClusterSpec
		machineSpec                             infrastructurev1beta1.OscMachineSpec
		expCheckVmPrivateIpsOscDuplicateNameErr error
	}{
		{
			name:                                    "get separate name",
			clusterSpec:                             defaultVmClusterInitialize,
			machineSpec:                             defaultMultiVmInitialize,
			expCheckVmPrivateIpsOscDuplicateNameErr: nil,
		},
		{
			name:        "get duplicate name",
			clusterSpec: defaultVmClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
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
					Vm: infrastructurev1beta1.OscVm{
						Name:       "test-vm",
						ImageId:    "ami-00000000",
						Role:       "controlplane",
						VolumeName: "test-volume",
						DeviceName: "/dev/sda1",
						RootDisk: infrastructurev1beta1.OscRootDisk{

							RootDiskSize: 30,
							RootDiskIops: 1500,
							RootDiskType: "io1",
						},
						KeypairName:      "rke",
						SubregionName:    "eu-west-2a",
						SubnetName:       "test-subnet",
						LoadBalancerName: "test-loadbalancer",
						VmType:           "tinav3.c2r4p2",
						PublicIpName:     "test-publicip",
						SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
							{
								Name: "test-securitygroup",
							},
						},
						PrivateIps: []infrastructurev1beta1.OscPrivateIpElement{
							{
								Name:      "test-privateip-first",
								PrivateIp: "10.0.0.17",
							},
							{
								Name:      "test-privateip-first",
								PrivateIp: "10.0.0.17",
							},
						},
					},
				},
			},
			expCheckVmPrivateIpsOscDuplicateNameErr: errors.New("test-privateip-first already exist"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			_, machineScope := SetupMachine(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			err := checkVmPrivateIpOscDuplicateName(machineScope)
			if vtc.expCheckVmPrivateIpsOscDuplicateNameErr != nil {
				require.EqualError(t, err, vtc.expCheckVmPrivateIpsOscDuplicateNameErr.Error(), "checkVmPrivateIpsOscDuplicateName() should return the same error")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestCheckVmVolumeSubregionName has several tests to cover the code of the function checkVmVolumeSubregionName
func TestCheckVmVolumeSubregionName(t *testing.T) {
	vmTestCases := []struct {
		name                             string
		clusterSpec                      infrastructurev1beta1.OscClusterSpec
		machineSpec                      infrastructurev1beta1.OscMachineSpec
		expCheckVmVolumeSubregionNameErr error
	}{
		{
			name:        "get the same volume and vm subregion name",
			clusterSpec: defaultVmClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
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
					Vm: infrastructurev1beta1.OscVm{
						Name:       "test-vm",
						ImageId:    "ami-00000000",
						Role:       "controlplane",
						VolumeName: "test-volume",
						DeviceName: "/dev/sda1",
						RootDisk: infrastructurev1beta1.OscRootDisk{

							RootDiskSize: 30,
							RootDiskIops: 1500,
							RootDiskType: "io1",
						},
						KeypairName:      "rke",
						SubregionName:    "eu-west-2a",
						SubnetName:       "test-subnet",
						LoadBalancerName: "test-loadbalancer",
						VmType:           "tinav3.c2r4p2",
						PublicIpName:     "test-publicip",
						SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
							{
								Name: "test-securitygroup",
							},
						},
						PrivateIps: []infrastructurev1beta1.OscPrivateIpElement{
							{
								Name:      "test-privateip-first",
								PrivateIp: "10.0.0.17",
							},
						},
					},
				},
			},
			expCheckVmVolumeSubregionNameErr: nil,
		},
		{
			name:        "can not get the same volume and vm subregion name",
			clusterSpec: defaultVmClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
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
					Vm: infrastructurev1beta1.OscVm{
						Name:       "test-vm",
						ImageId:    "ami-00000000",
						Role:       "controlplane",
						VolumeName: "test-volume",
						DeviceName: "/dev/sda1",
						RootDisk: infrastructurev1beta1.OscRootDisk{

							RootDiskSize: 30,
							RootDiskIops: 1500,
							RootDiskType: "io1",
						},
						KeypairName:      "rke",
						SubregionName:    "eu-west-2b",
						SubnetName:       "test-subnet",
						LoadBalancerName: "test-loadbalancer",
						VmType:           "tinav3.c2r4p2",
						PublicIpName:     "test-publicip",
						SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
							{
								Name: "test-securitygroup",
							},
						},
						PrivateIps: []infrastructurev1beta1.OscPrivateIpElement{
							{
								Name:      "test-privateip-first",
								PrivateIp: "10.0.0.17",
							},
						},
					},
				},
			},
			expCheckVmVolumeSubregionNameErr: errors.New("volume test-volume and vm test-vm are not in the same subregion eu-west-2b"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			_, machineScope := SetupMachine(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			err := checkVmVolumeSubregionName(machineScope)
			if vtc.expCheckVmVolumeSubregionNameErr != nil {
				require.EqualError(t, err, vtc.expCheckVmVolumeSubregionNameErr.Error(), "checkVmVolumeSubregionName() should return the same error")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestUseFailureDomain(t *testing.T) {
	vmTestCases := []struct {
		name                string
		clusterSpec         infrastructurev1beta1.OscClusterSpec
		machineSec          clusterv1.MachineSpec
		oscMachineSpec      infrastructurev1beta1.OscMachineSpec
		expFailureDomainSet bool
	}{
		{
			name:                "Default cluster without FailureDomain",
			clusterSpec:         defaultVmClusterInitialize,
			machineSec:          clusterv1.MachineSpec{},
			oscMachineSpec:      defaultVmInitialize,
			expFailureDomainSet: false,
		},
		{
			name:                "Cluster with FailureDomain not used (Subnet and Subregion provided)",
			clusterSpec:         failureDomainClusterInitialize,
			machineSec:          clusterv1.MachineSpec{},
			oscMachineSpec:      defaultVmInitialize,
			expFailureDomainSet: false,
		},
		{
			name:        "Cluster with FailureDomain used (Subnet and Subregion not provided)",
			clusterSpec: failureDomainClusterInitialize,
			machineSec: clusterv1.MachineSpec{
				FailureDomain: &failureDomainSubnet,
			},
			oscMachineSpec:      defaultFailureDomainVmInitialize,
			expFailureDomainSet: true,
		},
	}

	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope := SetupMachine(t, vtc.name, vtc.clusterSpec, vtc.oscMachineSpec)
			machineScope.Machine.Spec = vtc.machineSec
			UseFailureDomain(clusterScope, machineScope)

			if vtc.expFailureDomainSet {
				assert.Equal(t, failureDomainSubnet, machineScope.GetVm().SubnetName)
				assert.Equal(t, failureDomainSubregion, machineScope.GetVm().SubregionName)
			} else {
				assert.NotEqual(t, failureDomainSubnet, machineScope.GetVm().SubnetName)
			}
		})
	}
}

/*
// TestReconcileVm has several tests to cover the code of the function reconcileVm
func TestReconcileVm(t *testing.T) {
	vmTestCases := []struct {
		name                                    string
		clusterSpec                             infrastructurev1beta1.OscClusterSpec
		machineSpec                             infrastructurev1beta1.OscMachineSpec
		expCreateVmFound                        bool
		expLinkPublicIpFound                    bool
		expCreateInboundSecurityGroupRuleFound  bool
		expCreateOutboundSecurityGroupRuleFound bool
		expGetOutboundSecurityGroupRuleFound    bool
		expGetInboundSecurityGroupRuleFound     bool
		expTagFound                             bool
		expCreateVmErr                          error
		expReconcileVmErr                       error
		expCheckVmStateBootErr                  error
		expCheckVolumeStateAvailableErr         error
		expLinkVolumeErr                        error
		expCheckVolumeStateUseErr               error
		expCheckVmStateVolumeErr                error
		expCreateInboundSecurityGroupRuleErr    error
		expGetInboundSecurityGroupRuleErr       error
		expCreateOutboundSecurityGroupRuleErr   error
		expGetOutboundSecurityGroupRuleErr      error
		expLinkPublicIpErr                      error
		expCheckVmStatePublicIpErr              error
		expReadTagErr                           error
		expLinkLoadBalancerBackendMachineErr    error
	}{
		{
			name:                                    "create vm (first time reconcile loop)",
			clusterSpec:                             defaultVmClusterInitialize,
			machineSpec:                             defaultVmInitialize,
			expCreateVmFound:                        true,
			expLinkPublicIpFound:                    true,
			expCreateInboundSecurityGroupRuleFound:  true,
			expGetInboundSecurityGroupRuleFound:     false,
			expCreateOutboundSecurityGroupRuleFound: true,
			expGetOutboundSecurityGroupRuleFound:    false,
			expTagFound:                             false,
			expCreateVmErr:                          nil,
			expCheckVmStateBootErr:                  nil,
			expCheckVolumeStateAvailableErr:         nil,
			expLinkVolumeErr:                        nil,
			expCheckVolumeStateUseErr:               nil,
			expCheckVmStateVolumeErr:                nil,
			expLinkPublicIpErr:                      nil,
			expCheckVmStatePublicIpErr:              nil,
			expLinkLoadBalancerBackendMachineErr:    nil,
			expCreateInboundSecurityGroupRuleErr:    nil,
			expGetInboundSecurityGroupRuleErr:       nil,
			expCreateOutboundSecurityGroupRuleErr:   nil,
			expGetOutboundSecurityGroupRuleErr:      nil,
			expReadTagErr:                           nil,
			expReconcileVmErr:                       nil,
		},
		{
			name:                                    "create two vms (first time reconcile loop)",
			clusterSpec:                             defaultVmClusterInitialize,
			machineSpec:                             defaultMultiVmInitialize,
			expCreateVmFound:                        true,
			expLinkPublicIpFound:                    true,
			expCreateInboundSecurityGroupRuleFound:  true,
			expGetInboundSecurityGroupRuleFound:     false,
			expCreateOutboundSecurityGroupRuleFound: true,
			expGetOutboundSecurityGroupRuleFound:    false,
			expTagFound:                             false,
			expCreateVmErr:                          nil,
			expCheckVmStateBootErr:                  nil,
			expCheckVolumeStateAvailableErr:         nil,
			expLinkVolumeErr:                        nil,
			expCheckVolumeStateUseErr:               nil,
			expCheckVmStateVolumeErr:                nil,
			expLinkPublicIpErr:                      nil,
			expCheckVmStatePublicIpErr:              nil,
			expLinkLoadBalancerBackendMachineErr:    nil,
			expCreateInboundSecurityGroupRuleErr:    nil,
			expGetInboundSecurityGroupRuleErr:       nil,
			expCreateOutboundSecurityGroupRuleErr:   nil,
			expGetOutboundSecurityGroupRuleErr:      nil,
			expReadTagErr:                           nil,
			expReconcileVmErr:                       nil,
		},
		{
			name:                                    "user delete vm without cluster-api",
			clusterSpec:                             defaultVmClusterInitialize,
			machineSpec:                             defaultVmInitialize,
			expCreateVmFound:                        true,
			expLinkPublicIpFound:                    true,
			expCreateInboundSecurityGroupRuleFound:  true,
			expGetInboundSecurityGroupRuleFound:     false,
			expCreateOutboundSecurityGroupRuleFound: true,
			expGetOutboundSecurityGroupRuleFound:    false,
			expTagFound:                             false,
			expCreateVmErr:                          nil,
			expCheckVmStateBootErr:                  nil,
			expCheckVolumeStateAvailableErr:         nil,
			expLinkVolumeErr:                        nil,
			expCheckVolumeStateUseErr:               nil,
			expCheckVmStateVolumeErr:                nil,
			expLinkPublicIpErr:                      nil,
			expCheckVmStatePublicIpErr:              nil,
			expLinkLoadBalancerBackendMachineErr:    nil,
			expCreateInboundSecurityGroupRuleErr:    nil,
			expGetInboundSecurityGroupRuleErr:       nil,
			expCreateOutboundSecurityGroupRuleErr:   nil,
			expGetOutboundSecurityGroupRuleErr:      nil,
			expReadTagErr:                           nil,
			expReconcileVmErr:                       nil,
		},
		{
			name:                                    "failed to create outbound securityGroupRule",
			clusterSpec:                             defaultVmClusterInitialize,
			machineSpec:                             defaultVmInitialize,
			expCreateVmFound:                        true,
			expLinkPublicIpFound:                    true,
			expCreateInboundSecurityGroupRuleFound:  false,
			expGetInboundSecurityGroupRuleFound:     false,
			expCreateOutboundSecurityGroupRuleFound: true,
			expGetOutboundSecurityGroupRuleFound:    false,
			expTagFound:                             false,
			expCreateVmErr:                          nil,
			expCheckVmStateBootErr:                  nil,
			expCheckVolumeStateAvailableErr:         nil,
			expLinkVolumeErr:                        nil,
			expCheckVolumeStateUseErr:               nil,
			expCheckVmStateVolumeErr:                nil,
			expLinkPublicIpErr:                      nil,
			expCheckVmStatePublicIpErr:              nil,
			expLinkLoadBalancerBackendMachineErr:    nil,
			expCreateOutboundSecurityGroupRuleErr:   errors.New("CreateSecurityGroupRule generic error"),
			expGetOutboundSecurityGroupRuleErr:      nil,
			expCreateInboundSecurityGroupRuleErr:    nil,
			expGetInboundSecurityGroupRuleErr:       nil,
			expReadTagErr:                           nil,
			expReconcileVmErr:                       errors.New("CreateSecurityGroupRule generic error Can not create outbound securityGroupRule for OscCluster test-system/test-osc"),
		},
		{
			name:                                    "failed to get outbound securityGroupRule",
			clusterSpec:                             defaultVmClusterInitialize,
			machineSpec:                             defaultVmInitialize,
			expCreateVmFound:                        true,
			expLinkPublicIpFound:                    true,
			expCreateInboundSecurityGroupRuleFound:  false,
			expGetInboundSecurityGroupRuleFound:     false,
			expCreateOutboundSecurityGroupRuleFound: false,
			expGetOutboundSecurityGroupRuleFound:    true,
			expTagFound:                             false,
			expCreateVmErr:                          nil,
			expCheckVmStateBootErr:                  nil,
			expCheckVolumeStateAvailableErr:         nil,
			expLinkVolumeErr:                        nil,
			expCheckVolumeStateUseErr:               nil,
			expCheckVmStateVolumeErr:                nil,
			expLinkPublicIpErr:                      nil,
			expCheckVmStatePublicIpErr:              nil,
			expLinkLoadBalancerBackendMachineErr:    nil,
			expCreateOutboundSecurityGroupRuleErr:   nil,
			expGetOutboundSecurityGroupRuleErr:      errors.New("GetSecurityGroupRule generic error"),
			expCreateInboundSecurityGroupRuleErr:    nil,
			expGetInboundSecurityGroupRuleErr:       nil,
			expReadTagErr:                           nil,
			expReconcileVmErr:                       errors.New("GetSecurityGroupRule generic error Can not get outbound securityGroupRule for OscCluster test-system/test-osc"),
		},
		{
			name:                                    "failed to create inbound securityGroupRule",
			clusterSpec:                             defaultVmClusterInitialize,
			machineSpec:                             defaultVmInitialize,
			expCreateVmFound:                        true,
			expLinkPublicIpFound:                    true,
			expCreateInboundSecurityGroupRuleFound:  false,
			expGetInboundSecurityGroupRuleFound:     false,
			expCreateOutboundSecurityGroupRuleFound: false,
			expGetOutboundSecurityGroupRuleFound:    false,
			expTagFound:                             false,
			expCreateVmErr:                          nil,
			expCheckVmStateBootErr:                  nil,
			expCheckVolumeStateAvailableErr:         nil,
			expLinkVolumeErr:                        nil,
			expCheckVolumeStateUseErr:               nil,
			expCheckVmStateVolumeErr:                nil,
			expLinkPublicIpErr:                      nil,
			expCheckVmStatePublicIpErr:              nil,
			expLinkLoadBalancerBackendMachineErr:    nil,
			expCreateInboundSecurityGroupRuleErr:    errors.New("CreateSecurityGroupRule generic error"),
			expGetInboundSecurityGroupRuleErr:       nil,
			expCreateOutboundSecurityGroupRuleErr:   nil,
			expGetOutboundSecurityGroupRuleErr:      nil,
			expReadTagErr:                           nil,
			expReconcileVmErr:                       errors.New("CreateSecurityGroupRule generic error Can not create inbound securityGroupRule for OscCluster test-system/test-osc"),
		},
		{
			name:                                    "failed to get inbound securityGroupRule",
			clusterSpec:                             defaultVmClusterInitialize,
			machineSpec:                             defaultVmInitialize,
			expCreateVmFound:                        true,
			expLinkPublicIpFound:                    true,
			expCreateInboundSecurityGroupRuleFound:  false,
			expGetInboundSecurityGroupRuleFound:     true,
			expCreateOutboundSecurityGroupRuleFound: false,
			expGetOutboundSecurityGroupRuleFound:    true,
			expCreateVmErr:                          nil,
			expCheckVmStateBootErr:                  nil,
			expCheckVolumeStateAvailableErr:         nil,
			expLinkVolumeErr:                        nil,
			expCheckVolumeStateUseErr:               nil,
			expCheckVmStateVolumeErr:                nil,
			expLinkPublicIpErr:                      nil,
			expCheckVmStatePublicIpErr:              nil,
			expLinkLoadBalancerBackendMachineErr:    nil,
			expCreateInboundSecurityGroupRuleErr:    nil,
			expGetInboundSecurityGroupRuleErr:       errors.New("GetSecurityGroupRule generic error"),
			expCreateOutboundSecurityGroupRuleErr:   nil,
			expGetOutboundSecurityGroupRuleErr:      nil,
			expReadTagErr:                           nil,
			expReconcileVmErr:                       errors.New("GetSecurityGroupRule generic error Can not get inbound securityGroupRule for OscCluster test-system/test-osc"),
		},
		{
			name:                                   "linkPublicIp does not exist",
			clusterSpec:                            defaultVmClusterInitialize,
			machineSpec:                            defaultVmInitialize,
			expCreateVmFound:                       true,
			expTagFound:                            false,
			expLinkPublicIpFound:                   false,
			expCreateInboundSecurityGroupRuleFound: true,
			expGetInboundSecurityGroupRuleFound:    false,

			expCreateOutboundSecurityGroupRuleFound: true,
			expGetOutboundSecurityGroupRuleFound:    false,
			expCreateVmErr:                          nil,
			expCheckVmStateBootErr:                  nil,
			expCheckVolumeStateAvailableErr:         nil,
			expLinkVolumeErr:                        nil,
			expCheckVolumeStateUseErr:               nil,
			expCheckVmStateVolumeErr:                nil,
			expLinkPublicIpErr:                      nil,
			expCheckVmStatePublicIpErr:              nil,
			expLinkLoadBalancerBackendMachineErr:    nil,
			expCreateInboundSecurityGroupRuleErr:    nil,
			expGetInboundSecurityGroupRuleErr:       nil,
			expCreateOutboundSecurityGroupRuleErr:   nil,
			expGetOutboundSecurityGroupRuleErr:      nil,
			expReadTagErr:                           nil,
			expReconcileVmErr:                       nil,
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope, ctx, mockOscVmInterface, mockOscVolumeInterface, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface, mockOscTagInterface := SetupWithVmMock(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			vmSpec := machineScope.GetVm()
			vmName := vtc.machineSpec.Node.Vm.Name + "-uid"
			vmId := "i-" + vmName
			vmTags := vtc.machineSpec.Node.Vm.Tags
			vmRef := machineScope.GetVmRef()
			vmRef.ResourceMap = make(map[string]string)
			vmRef.ResourceMap[vmName] = vmId

			volumeName := vtc.machineSpec.Node.Vm.VolumeName + "-uid"
			volumeId := "vol-" + volumeName
			volumeRef := machineScope.GetVolumeRef()
			volumeRef.ResourceMap = make(map[string]string)
			volumeRef.ResourceMap[volumeName] = volumeId
			volumeStateAvailable := "available"
			volumeStateUse := "in-use"

			subnetName := vtc.machineSpec.Node.Vm.SubnetName + "-uid"
			subnetId := "subnet-" + subnetName
			subnetRef := clusterScope.GetSubnetRef()
			subnetRef.ResourceMap = make(map[string]string)
			subnetRef.ResourceMap[subnetName] = subnetId

			publicIpName := vtc.machineSpec.Node.Vm.PublicIpName + "-uid"
			publicIpId := "eipalloc-" + publicIpName
			publicIpRef := clusterScope.GetPublicIpRef()
			publicIpRef.ResourceMap = make(map[string]string)
			publicIpRef.ResourceMap[publicIpName] = publicIpId

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

			// Populate SecurityGroupRef
			var securityGroupIds []string
			vmSecurityGroups := machineScope.GetVmSecurityGroups()
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			for _, vmSecurityGroup := range *vmSecurityGroups {
				securityGroupName := vmSecurityGroup.Name + "-uid"
				securityGroupId := "sg-" + securityGroupName
				securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
				securityGroupIds = append(securityGroupIds, securityGroupId)
			}
			deviceName := vtc.machineSpec.Node.Vm.DeviceName
			//vmSpec := vtc.machineSpec.Node.Vm
			var clockInsideLoop time.Duration = 20
			var clockLoop time.Duration = 240
			loadBalancerName := vtc.machineSpec.Node.Vm.LoadBalancerName
			loadBalancerSpec := clusterScope.GetLoadBalancer()
			loadBalancerSpec.SetDefaultValue()
			loadBalancerSecurityGroupName := loadBalancerSpec.SecurityGroupName
			ipProtocol := strings.ToLower(loadBalancerSpec.Listener.BackendProtocol)
			fromPortRange := loadBalancerSpec.Listener.BackendPort
			toPortRange := loadBalancerSpec.Listener.BackendPort
			loadBalancerSecurityGroupClusterScopeName := loadBalancerSecurityGroupName + "-uid"
			loadBalancerSecurityGroupId := "sg-" + loadBalancerSecurityGroupClusterScopeName
			securityGroupsRef.ResourceMap[loadBalancerSecurityGroupClusterScopeName] = loadBalancerSecurityGroupId
			associateSecurityGroupId := securityGroupsRef.ResourceMap[loadBalancerSecurityGroupClusterScopeName]
			createVms := osc.CreateVmsResponse{
				Vms: &[]osc.Vm{
					{
						VmId: &vmId,
					},
				},
			}

			createVm := *createVms.Vms
			tag := osc.Tag{
				ResourceId: &vmId,
			}
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

			linkPublicIp := osc.LinkPublicIpResponse{
				LinkPublicIpId: &linkPublicIpId,
			}
			securityGroupRule := osc.CreateSecurityGroupRuleResponse{
				SecurityGroup: &osc.SecurityGroup{
					SecurityGroupId: &loadBalancerSecurityGroupId,
				},
			}

			readSecurityGroups := osc.ReadSecurityGroupsResponse{
				SecurityGroups: &[]osc.SecurityGroup{
					*securityGroupRule.SecurityGroup,
				},
			}
			readSecurityGroup := *readSecurityGroups.SecurityGroups
			vm := &createVm[0]
			if vtc.expCreateVmFound {
				mockOscVmInterface.
					EXPECT().
					CreateVm(gomock.Eq(machineScope), gomock.Eq(&vmSpec), gomock.Eq(subnetId), gomock.Eq(securityGroupIds), gomock.Eq(privateIps), gomock.Eq(vmName), gomock.Eq(vmTags)).
					Return(vm, vtc.expCreateVmErr)
			} else {
				mockOscVmInterface.
					EXPECT().
					CreateVm(gomock.Eq(machineScope), gomock.Eq(vmSpec), gomock.Eq(subnetId), gomock.Eq(securityGroupIds), gomock.Eq(privateIps), gomock.Eq(vmName), gomock.Eq(vmTags)).
					Return(nil, vtc.expCreateVmErr)
			}

			if vtc.machineSpec.Node.Vm.VolumeName != "" {
				mockOscVolumeInterface.
					EXPECT().
					CheckVolumeState(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(volumeStateAvailable), gomock.Eq(volumeId)).
					Return(vtc.expCheckVolumeStateAvailableErr)
				mockOscVolumeInterface.
					EXPECT().
					LinkVolume(gomock.Eq(volumeId), gomock.Eq(vmId), gomock.Eq(deviceName)).
					Return(vtc.expLinkVolumeErr)

				mockOscVolumeInterface.
					EXPECT().
					CheckVolumeState(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(volumeStateUse), gomock.Eq(volumeId)).
					Return(vtc.expCheckVolumeStateUseErr)

			}

			if vtc.expLinkPublicIpFound {
				mockOscPublicIpInterface.
					EXPECT().
					LinkPublicIp(gomock.Eq(publicIpId), gomock.Eq(vmId)).
					Return(*linkPublicIp.LinkPublicIpId, vtc.expLinkPublicIpErr)
			} else {
				mockOscPublicIpInterface.
					EXPECT().
					LinkPublicIp(gomock.Eq(publicIpId), gomock.Eq(vmId)).
					Return("", vtc.expLinkPublicIpErr)
			}

			vmIds := []string{vmId}
			mockOscLoadBalancerInterface.
				EXPECT().
				LinkLoadBalancerBackendMachines(gomock.Eq(vmIds), gomock.Eq(loadBalancerName)).
				Return(vtc.expLinkLoadBalancerBackendMachineErr)

			if vtc.expGetOutboundSecurityGroupRuleFound {
				mockOscSecurityGroupInterface.
					EXPECT().
					GetSecurityGroupFromSecurityGroupRule(gomock.Eq(associateSecurityGroupId), gomock.Eq("Outbound"), gomock.Eq(ipProtocol), "", gomock.Eq(securityGroupIds[0]), gomock.Eq(fromPortRange), gomock.Eq(toPortRange)).
					Return(&readSecurityGroup[0], vtc.expGetOutboundSecurityGroupRuleErr)

			} else {
				mockOscSecurityGroupInterface.
					EXPECT().
					GetSecurityGroupFromSecurityGroupRule(gomock.Eq(associateSecurityGroupId), gomock.Eq("Outbound"), gomock.Eq(ipProtocol), "", gomock.Eq(securityGroupIds[0]), gomock.Eq(fromPortRange), gomock.Eq(toPortRange)).
					Return(nil, vtc.expGetOutboundSecurityGroupRuleErr)
			}

			if vtc.expCreateOutboundSecurityGroupRuleFound {
				mockOscSecurityGroupInterface.
					EXPECT().
					CreateSecurityGroupRule(gomock.Eq(associateSecurityGroupId), gomock.Eq("Outbound"), gomock.Eq(ipProtocol), "", gomock.Eq(securityGroupIds[0]), gomock.Eq(fromPortRange), gomock.Eq(toPortRange)).
					Return(securityGroupRule.SecurityGroup, vtc.expCreateOutboundSecurityGroupRuleErr)
			} else if vtc.expGetOutboundSecurityGroupRuleErr != nil || vtc.expGetInboundSecurityGroupRuleErr != nil {
			} else {
				mockOscSecurityGroupInterface.
					EXPECT().
					CreateSecurityGroupRule(gomock.Eq(associateSecurityGroupId), gomock.Eq("Outbound"), gomock.Eq(ipProtocol), "", gomock.Eq(securityGroupIds[0]), gomock.Eq(fromPortRange), gomock.Eq(toPortRange)).
					Return(nil, vtc.expCreateOutboundSecurityGroupRuleErr)
			}

			if vtc.expGetInboundSecurityGroupRuleFound {
				mockOscSecurityGroupInterface.
					EXPECT().
					GetSecurityGroupFromSecurityGroupRule(gomock.Eq(associateSecurityGroupId), gomock.Eq("Inbound"), gomock.Eq(ipProtocol), "", gomock.Eq(securityGroupIds[0]), gomock.Eq(fromPortRange), gomock.Eq(toPortRange)).
					Return(&readSecurityGroup[0], vtc.expGetInboundSecurityGroupRuleErr)
			} else if vtc.expCreateOutboundSecurityGroupRuleErr != nil || vtc.expGetOutboundSecurityGroupRuleErr != nil {

			} else {
				mockOscSecurityGroupInterface.
					EXPECT().
					GetSecurityGroupFromSecurityGroupRule(gomock.Eq(associateSecurityGroupId), gomock.Eq("Inbound"), gomock.Eq(ipProtocol), "", gomock.Eq(securityGroupIds[0]), gomock.Eq(fromPortRange), gomock.Eq(toPortRange)).
					Return(nil, vtc.expGetInboundSecurityGroupRuleErr)

			}

			if vtc.expCreateInboundSecurityGroupRuleFound {
				mockOscSecurityGroupInterface.
					EXPECT().
					CreateSecurityGroupRule(gomock.Eq(associateSecurityGroupId), gomock.Eq("Inbound"), gomock.Eq(ipProtocol), "", gomock.Eq(securityGroupIds[0]), gomock.Eq(fromPortRange), gomock.Eq(toPortRange)).
					Return(securityGroupRule.SecurityGroup, vtc.expCreateInboundSecurityGroupRuleErr)

			} else if (!vtc.expCreateInboundSecurityGroupRuleFound && vtc.expCreateOutboundSecurityGroupRuleFound) || vtc.expGetOutboundSecurityGroupRuleErr != nil || vtc.expGetInboundSecurityGroupRuleErr != nil {
			} else {
				mockOscSecurityGroupInterface.
					EXPECT().
					CreateSecurityGroupRule(gomock.Eq(associateSecurityGroupId), gomock.Eq("Inbound"), gomock.Eq(ipProtocol), "", gomock.Eq(securityGroupIds[0]), gomock.Eq(fromPortRange), gomock.Eq(toPortRange)).
					Return(nil, vtc.expCreateInboundSecurityGroupRuleErr)

			}
			reconcileVm, err := reconcileVm(ctx, clusterScope, machineScope, mockOscVmInterface, mockOscVolumeInterface, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface, mockOscTagInterface)
			if err != nil {
				require.EqualError(t, vtc.expReconcileVmErr.Error(), err.Error(), "reconcileVm() should return the same error")
			} else {
				require.NoError(t, vtc.expReconcileVmErr)
			}
			t.Logf("find reconcileVm %v\n", reconcileVm)
		})
	}
}
/*
// TestReconcileVmCreate has several tests to cover the code of the function reconcileVm
func TestReconcileVmCreate(t *testing.T) {
	vmTestCases := []struct {
		name              string
		clusterSpec       infrastructurev1beta1.OscClusterSpec
		machineSpec       infrastructurev1beta1.OscMachineSpec
		expCreateVmFound  bool
		expTagFound       bool
		expCreateVmErr    error
		expReadTagErr     error
		expReconcileVmErr error
	}{
		{
			name:              "failed to create vm",
			clusterSpec:       defaultVmClusterInitialize,
			machineSpec:       defaultVmInitialize,
			expCreateVmFound:  false,
			expTagFound:       false,
			expCreateVmErr:    errors.New("CreateVm generic error"),
			expReadTagErr:     nil,
			expReconcileVmErr: errors.New("CreateVm generic error Can not create vm for OscMachine test-system/test-osc"),
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
			volumeRef.ResourceMap[volumeName] = volumeId

			subnetName := vtc.machineSpec.Node.Vm.SubnetName + "-uid"
			subnetId := "subnet-" + subnetName
			subnetRef := clusterScope.GetSubnetRef()
			subnetRef.ResourceMap = make(map[string]string)
			subnetRef.ResourceMap[subnetName] = subnetId

			publicIpName := vtc.machineSpec.Node.Vm.PublicIpName + "-uid"
			publicIpId := "eipalloc-" + publicIpName
			publicIpRef := clusterScope.GetPublicIpRef()
			publicIpRef.ResourceMap = make(map[string]string)
			publicIpRef.ResourceMap[publicIpName] = publicIpId

			linkPublicIpId := "eipassoc-" + publicIpName
			linkPublicIpRef := machineScope.GetLinkPublicIpRef()
			linkPublicIpRef.ResourceMap = make(map[string]string)
			linkPublicIpRef.ResourceMap[vmName] = linkPublicIpId

			var privateIps []string
			vmPrivateIps := machineScope.GetVmPrivateIps()
			for _, vmPrivateIp := range *vmPrivateIps {
				privateIp := vmPrivateIp.PrivateIp
				privateIps = append(privateIps, privateIp)
			}

			var securityGroupIds []string
			vmSecurityGroups := machineScope.GetVmSecurityGroups()
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			for _, vmSecurityGroup := range *vmSecurityGroups {
				securityGroupName := vmSecurityGroup.Name + "-uid"
				securityGroupId := "sg-" + securityGroupName
				securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
				securityGroupIds = append(securityGroupIds, securityGroupId)
			}
			tag := osc.Tag{
				ResourceId: &vmId,
			}
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
			vmSpec := vtc.machineSpec.Node.Vm
			createVms := osc.CreateVmsResponse{
				Vms: &[]osc.Vm{
					{
						VmId: &vmId,
					},
				},
			}

			createVm := *createVms.Vms
			vm := &createVm[0]
			if vtc.expCreateVmFound {
				mockOscVmInterface.
					EXPECT().
					CreateVm(gomock.Eq(machineScope), gomock.Eq(&vmSpec), gomock.Eq(subnetId), gomock.Eq(securityGroupIds), gomock.Eq(privateIps), gomock.Eq(vmName), gomock.Eq(vtc.machineSpec.Node.Vm.Tags)).
					Return(vm, vtc.expCreateVmErr)
			} else {
				mockOscVmInterface.
					EXPECT().
					CreateVm(gomock.Eq(machineScope), gomock.Eq(&vmSpec), gomock.Eq(subnetId), gomock.Eq(securityGroupIds), gomock.Eq(privateIps), gomock.Eq(vmName), gomock.Eq(vtc.machineSpec.Node.Vm.Tags)).
					Return(nil, vtc.expCreateVmErr)
			}

			reconcileVm, err := reconcileVm(ctx, clusterScope, machineScope, mockOscVmInterface, mockOscVolumeInterface, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface, mockOscTagInterface)
			if err != nil {
				require.EqualError(t, vtc.expReconcileVmErr.Error(), err.Error(), "reconcileVm() should return the same error")
			} else {
				require.NoError(t, vtc.expReconcileVmErr)
			}
			t.Logf("find reconcileVm %v\n", reconcileVm)
		})
	}
}

// TestReconcileVmGet has several tests to cover the code of the function reconcileVm
func TestReconcileVmGet(t *testing.T) {
	vmTestCases := []struct {
		name                   string
		clusterSpec            infrastructurev1beta1.OscClusterSpec
		machineSpec            infrastructurev1beta1.OscMachineSpec
		expGetVmFound          bool
		expGetVmStateFound     bool
		expAddCcmTagFound      bool
		expPrivateDnsNameFound bool
		expPrivateIpFound      bool
		expTagFound            bool
		expGetVmStateErr       error
		expGetVmErr            error
		expAddCcmTagErr        error
		expPrivateDnsNameErr   error
		expPrivateIpErr        error
		expReadTagErr          error
		expReconcileVmErr      error
	}{
		{
			name:                   "get vm",
			clusterSpec:            defaultVmClusterInitialize,
			machineSpec:            defaultVmReconcile,
			expGetVmFound:          true,
			expGetVmStateFound:     true,
			expAddCcmTagFound:      true,
			expTagFound:            false,
			expPrivateDnsNameFound: true,
			expPrivateIpFound:      true,
			expGetVmErr:            nil,
			expGetVmStateErr:       nil,
			expAddCcmTagErr:        nil,
			expPrivateDnsNameErr:   nil,
			expPrivateIpErr:        nil,
			expReadTagErr:          nil,
			expReconcileVmErr:      nil,
		},
		{
			name:                   "failed to get vm",
			clusterSpec:            defaultVmClusterInitialize,
			machineSpec:            defaultVmReconcile,
			expGetVmFound:          true,
			expGetVmStateFound:     false,
			expAddCcmTagFound:      false,
			expPrivateDnsNameFound: true,
			expPrivateIpFound:      true,
			expTagFound:            false,
			expGetVmErr:            errors.New("GetVm generic error"),
			expGetVmStateErr:       nil,
			expAddCcmTagErr:        nil,
			expPrivateDnsNameErr:   nil,
			expPrivateIpErr:        nil,
			expReadTagErr:          nil,
			expReconcileVmErr:      errors.New("GetVm generic error"),
		},
		{
			name:                   "failed to set vmstate",
			clusterSpec:            defaultVmClusterInitialize,
			machineSpec:            defaultVmReconcile,
			expGetVmFound:          true,
			expGetVmStateFound:     true,
			expAddCcmTagFound:      true,
			expPrivateDnsNameFound: true,
			expPrivateIpFound:      true,
			expTagFound:            false,
			expGetVmErr:            nil,
			expAddCcmTagErr:        nil,
			expGetVmStateErr:       errors.New("GetVmState generic error"),
			expPrivateDnsNameErr:   nil,
			expPrivateIpErr:        nil,
			expReadTagErr:          nil,
			expReconcileVmErr:      errors.New("GetVmState generic error Can not get vm i-test-vm-uid state for OscMachine test-system/test-osc"),
		},
		{
			name:                   "failed to add tag",
			clusterSpec:            defaultVmClusterReconcile,
			machineSpec:            defaultVmReconcile,
			expGetVmFound:          true,
			expGetVmStateFound:     false,
			expAddCcmTagFound:      true,
			expPrivateDnsNameFound: true,
			expPrivateIpFound:      true,
			expTagFound:            false,
			expGetVmErr:            nil,
			expGetVmStateErr:       nil,
			expAddCcmTagErr:        errors.New("AddCcmTag generic error"),
			expPrivateDnsNameErr:   nil,
			expPrivateIpErr:        nil,
			expReadTagErr:          nil,
			expReconcileVmErr:      errors.New("AddCcmTag generic error can not add ccm tag test-system/test-osc"),
		},
		{
			name:                   "Failed to retrieve privateDnsName",
			clusterSpec:            defaultVmClusterReconcile,
			machineSpec:            defaultVmReconcile,
			expGetVmFound:          true,
			expGetVmStateFound:     false,
			expPrivateIpFound:      true,
			expAddCcmTagFound:      false,
			expPrivateDnsNameFound: false,
			expTagFound:            false,
			expGetVmErr:            nil,
			expGetVmStateErr:       nil,
			expAddCcmTagErr:        nil,
			expPrivateIpErr:        nil,
			expReadTagErr:          nil,
			expPrivateDnsNameErr:   errors.New("GetPrivateDnsNameok generic error"),
			expReconcileVmErr:      errors.New("Can not found privateDnsName test-system/test-osc"),
		},
		{
			name:                   "Failed to retrieve privateIp",
			clusterSpec:            defaultVmClusterReconcile,
			machineSpec:            defaultVmReconcile,
			expGetVmFound:          true,
			expGetVmStateFound:     false,
			expPrivateIpFound:      false,
			expAddCcmTagFound:      false,
			expPrivateDnsNameFound: true,
			expGetVmErr:            nil,
			expGetVmStateErr:       nil,
			expAddCcmTagErr:        nil,
			expPrivateIpErr:        errors.New("GetPrivateIpOk generic error"),
			expPrivateDnsNameErr:   nil,
			expReconcileVmErr:      errors.New("Can not found privateIp test-system/test-osc"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope, ctx, mockOscVmInterface, mockOscVolumeInterface, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface, mockOscTagInterface := SetupWithVmMock(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			vmName := vtc.machineSpec.Node.Vm.Name + "-uid"
			vmId := "i-" + vmName
			vmState := "running"

			tag := osc.Tag{
				ResourceId: &vmId,
			}
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
			volumeName := vtc.machineSpec.Node.Vm.VolumeName + "-uid"
			volumeId := "vol-" + volumeName
			volumeRef := machineScope.GetVolumeRef()
			volumeRef.ResourceMap = make(map[string]string)
			volumeRef.ResourceMap[volumeName] = volumeId

			subnetName := vtc.machineSpec.Node.Vm.SubnetName + "-uid"
			subnetId := "subnet-" + subnetName
			subnetRef := clusterScope.GetSubnetRef()
			subnetRef.ResourceMap = make(map[string]string)
			subnetRef.ResourceMap[subnetName] = subnetId

			publicIpName := vtc.machineSpec.Node.Vm.PublicIpName + "-uid"
			publicIpId := "eipalloc-" + publicIpName
			publicIpRef := clusterScope.GetPublicIpRef()
			publicIpRef.ResourceMap = make(map[string]string)
			publicIpRef.ResourceMap[publicIpName] = publicIpId

			linkPublicIpId := "eipassoc-" + publicIpName
			linkPublicIpRef := machineScope.GetLinkPublicIpRef()
			linkPublicIpRef.ResourceMap = make(map[string]string)
			linkPublicIpRef.ResourceMap[vmName] = linkPublicIpId

			var privateIps []string
			vmPrivateIps := machineScope.GetVmPrivateIps()
			for _, vmPrivateIp := range *vmPrivateIps {
				privateIp := vmPrivateIp.PrivateIp
				privateIps = append(privateIps, privateIp)
			}

			var securityGroupIds []string
			vmSecurityGroups := machineScope.GetVmSecurityGroups()
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			for _, vmSecurityGroup := range *vmSecurityGroups {
				securityGroupName := vmSecurityGroup.Name + "-uid"
				securityGroupId := "sg-" + securityGroupName
				securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
				securityGroupIds = append(securityGroupIds, securityGroupId)
			}
			var privateDnsName string
			var privateIp string
			var readVms osc.ReadVmsResponse
			if vtc.expPrivateDnsNameFound {
				privateDnsName = "ip-0-0-0-0.eu-west-2.compute.internal"
				readVms = osc.ReadVmsResponse{
					Vms: &[]osc.Vm{
						{
							VmId:           &vmId,
							PrivateDnsName: &privateDnsName,
						},
					},
				}
			} else {
				readVms = osc.ReadVmsResponse{
					Vms: &[]osc.Vm{
						{
							VmId: &vmId,
						},
					},
				}
			}
			readVm := *readVms.Vms
			vm := &readVm[0]
			privateIp = "0.0.0.0"
			if vtc.expPrivateIpFound {
				vm.PrivateIp = &privateIp
			}
			if vtc.expGetVmFound {
				mockOscVmInterface.
					EXPECT().
					GetVm(gomock.Any(), gomock.Eq(vmId)).
					Return(vm, vtc.expGetVmErr)
			} else {
				mockOscVmInterface.
					EXPECT().
					GetVm(gomock.Any(), gomock.Eq(vmId)).
					Return(nil, vtc.expGetVmErr)
			}

			if vtc.expGetVmStateFound {
				mockOscVmInterface.
					EXPECT().
					GetVmState(gomock.Eq(vmId)).
					Return(vmState, vtc.expGetVmStateErr)
			}
			clusterName := vtc.clusterSpec.Network.Net.ClusterName + "-uid"
			if vtc.expAddCcmTagFound {
				mockOscVmInterface.
					EXPECT().
					AddCcmTag(gomock.Eq(clusterName), gomock.Eq(privateDnsName), gomock.Eq(vmId)).
					Return(vtc.expAddCcmTagErr)
			}
			reconcileVm, err := reconcileVm(ctx, clusterScope, machineScope, mockOscVmInterface, mockOscVolumeInterface, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface, mockOscTagInterface)
			if err != nil {
				require.EqualError(t, vtc.expReconcileVmErr.Error(), err.Error(), "reconcileVm() should return the same error")
			} else {
				require.NoError(t, vtc.expReconcileVmErr)
			}
			t.Logf("find reconcileVm %v\n", reconcileVm)

		})
	}
}
*/

// TestReconcileDeleteVm has several tests to cover the code of the function reconcileDeleteVm
func TestReconcileDeleteVm(t *testing.T) {
	vmMachine1 := defaultVmReconcile
	vmMachine1.Node.Vm.Replica = 2
	vmTestCases := []struct {
		name                                    string
		clusterSpec                             infrastructurev1beta1.OscClusterSpec
		machineSpec                             infrastructurev1beta1.OscMachineSpec
		expListMachine                          bool
		expDeleteInboundSecurityGroupRuleFound  bool
		expDeleteOutboundSecurityGroupRuleFound bool
		expDeleteDedicatedPublicIpFound         bool
		expNoResourceId                         bool
		expGetLoadBalancerErr                   error
		expGetLoadBalancerContainsVm            bool
		expUnlinkLoadBalancerBackendMachineErr  error
		expDeleteVmErr                          error
		expGetVmErr                             error
		expSecurityGroupRuleFound               bool
		expGetVmFound                           bool
		expCheckUnlinkPublicIpErr               error
		expReconcileDeleteVmErr                 error
	}{
		{
			name:                                    "delete vm",
			clusterSpec:                             defaultVmClusterReconcile,
			machineSpec:                             defaultVmReconcile,
			expGetLoadBalancerContainsVm:            true,
			expDeleteInboundSecurityGroupRuleFound:  true,
			expDeleteOutboundSecurityGroupRuleFound: true,
			expSecurityGroupRuleFound:               true,
			expGetVmFound:                           true,
		},
		{
			name:                                    "delete without resource id",
			clusterSpec:                             defaultVmClusterReconcile,
			machineSpec:                             defaultVmReconcile,
			expNoResourceId:                         true,
			expGetLoadBalancerContainsVm:            true,
			expDeleteInboundSecurityGroupRuleFound:  true,
			expDeleteOutboundSecurityGroupRuleFound: true,
			expSecurityGroupRuleFound:               true,
			expGetVmFound:                           true,
		},
		{
			name:                                    "delete vm not link to lb",
			clusterSpec:                             defaultVmClusterReconcile,
			machineSpec:                             defaultVmReconcile,
			expDeleteInboundSecurityGroupRuleFound:  true,
			expDeleteOutboundSecurityGroupRuleFound: true,
			expSecurityGroupRuleFound:               true,
			expGetVmFound:                           true,
		},
		{
			name:                      "delete first vm in group",
			clusterSpec:               defaultVmClusterReconcile,
			machineSpec:               vmMachine1,
			expListMachine:            true,
			expSecurityGroupRuleFound: true,
			expGetVmFound:             true,
		},
		{
			name:                                    "delete vm with publicIp",
			clusterSpec:                             defaultVmClusterReconcile,
			machineSpec:                             defaultVmReconcileWithDedicatedIp,
			expDeleteInboundSecurityGroupRuleFound:  true,
			expDeleteOutboundSecurityGroupRuleFound: true,
			expDeleteDedicatedPublicIpFound:         true,
			expSecurityGroupRuleFound:               true,
			expGetVmFound:                           true,
		},
		{
			name:                                    "failed to delete vm",
			clusterSpec:                             defaultVmClusterReconcile,
			machineSpec:                             defaultVmReconcile,
			expDeleteInboundSecurityGroupRuleFound:  true,
			expDeleteOutboundSecurityGroupRuleFound: true,
			expSecurityGroupRuleFound:               true,
			expDeleteVmErr:                          errors.New("DeleteVm generic error"),
			expGetVmFound:                           true,
			expReconcileDeleteVmErr:                 errors.New("cannot delete vm: DeleteVm generic error"),
		},
		{
			name:                    "failed to get lb",
			clusterSpec:             defaultVmClusterReconcile,
			machineSpec:             defaultVmReconcile,
			expGetLoadBalancerErr:   errors.New("GetLoadBalancer generic error"),
			expGetVmFound:           true,
			expReconcileDeleteVmErr: errors.New("cannot unlink loadBalancer test-loadbalancer: GetLoadBalancer generic error"),
		},
		{
			name:                                   "failed to unlink lb",
			clusterSpec:                            defaultVmClusterReconcile,
			machineSpec:                            defaultVmReconcile,
			expGetLoadBalancerContainsVm:           true,
			expUnlinkLoadBalancerBackendMachineErr: errors.New("UnlinkLoadBalancerBackendMachine generic error"),
			expGetVmFound:                          true,
			expReconcileDeleteVmErr:                errors.New("cannot unlink loadBalancer test-loadbalancer: UnlinkLoadBalancerBackendMachine generic error"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope, ctx, mockOscVmInterface, _, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface, _ := SetupWithVmMock(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			vmId := vtc.machineSpec.Node.Vm.ResourceId
			if vtc.expNoResourceId {
				vtc.machineSpec.Node.Vm.ResourceId = ""
				vmName := vtc.machineSpec.Node.Vm.Name
				vmRef := machineScope.GetVmRef()
				vmRef.ResourceMap = make(map[string]string)
				vmRef.ResourceMap[vmName] = vmId
			}
			var securityGroupIds []string
			vmSecurityGroups := machineScope.GetVmSecurityGroups()
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			for _, vmSecurityGroup := range vmSecurityGroups {
				securityGroupName := vmSecurityGroup.Name + "-uid"
				securityGroupId := "sg-" + securityGroupName
				securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
				securityGroupIds = append(securityGroupIds, securityGroupId)
			}

			if vtc.expDeleteDedicatedPublicIpFound {
				publicIpName := machineScope.GetName() + "-publicIp"
				machineScope.OscMachine.Spec.Node.Vm.PublicIpName = publicIpName
				vtc.machineSpec.Node.Vm.PublicIpName = publicIpName
				machineScope.GetPublicIpIdRef().ResourceMap = map[string]string{publicIpName + "-uid": "eipassoc-" + publicIpName + "-uid"}
			}

			publicIpName := vtc.machineSpec.Node.Vm.PublicIpName + "-uid"
			linkPublicIpId := "eipassoc-" + publicIpName
			linkPublicIpRef := machineScope.GetLinkPublicIpRef()
			linkPublicIpRef.ResourceMap = make(map[string]string)
			linkPublicIpRef.ResourceMap[publicIpName] = linkPublicIpId

			loadBalancerName := vtc.machineSpec.Node.Vm.LoadBalancerName
			loadBalancerSpec := clusterScope.GetLoadBalancer()
			loadBalancerSpec.SetDefaultValue()
			loadBalancerSecurityGroupName := loadBalancerSpec.SecurityGroupName
			ipProtocol := strings.ToLower(loadBalancerSpec.Listener.BackendProtocol)
			fromPortRange := loadBalancerSpec.Listener.BackendPort
			toPortRange := loadBalancerSpec.Listener.BackendPort
			loadBalancerSecurityGroupClusterScopeName := loadBalancerSecurityGroupName + "-uid"
			loadBalancerSecurityGroupId := "sg-" + loadBalancerSecurityGroupClusterScopeName
			securityGroupsRef.ResourceMap[loadBalancerSecurityGroupClusterScopeName] = loadBalancerSecurityGroupId
			associateSecurityGroupId := securityGroupsRef.ResourceMap[loadBalancerSecurityGroupClusterScopeName]

			vm := &osc.Vm{
				VmId: &vmId,
			}
			if vtc.expGetVmFound {
				mockOscVmInterface.
					EXPECT().
					GetVm(gomock.Any(), gomock.Eq(vmId)).
					Return(vm, vtc.expGetVmErr)
			} else {
				mockOscVmInterface.
					EXPECT().
					GetVm(gomock.Any(), gomock.Eq(vmId)).
					Return(nil, vtc.expGetVmErr)
			}

			if vtc.expListMachine {
				fakeScheme := runtime.NewScheme()
				_ = clientgoscheme.AddToScheme(fakeScheme)
				_ = clusterv1.AddToScheme(fakeScheme)
				_ = apiextensionsv1.AddToScheme(fakeScheme)
				_ = infrastructurev1beta1.AddToScheme(fakeScheme)
				clientFake := fake.NewClientBuilder().WithScheme(fakeScheme).WithObjects(
					&clusterv1.Machine{
						TypeMeta: v1.TypeMeta{
							APIVersion: "cluster.x-k8s.io/v1",
							Kind:       "Machine",
						},
						ObjectMeta: v1.ObjectMeta{Name: "Machine1"},
					},
					&clusterv1.Machine{
						TypeMeta: v1.TypeMeta{
							APIVersion: "cluster.x-k8s.io/v1",
							Kind:       "Machine",
						},
						ObjectMeta: v1.ObjectMeta{Name: "Machine2"},
					},
				).Build()
				clusterScope.Client = clientFake
			}
			mockOscPublicIpInterface.
				EXPECT().
				UnlinkPublicIp(gomock.Any(), gomock.Eq(linkPublicIpId)).
				Return(vtc.expCheckUnlinkPublicIpErr)

			if vtc.expGetLoadBalancerContainsVm {
				mockOscLoadBalancerInterface.
					EXPECT().
					GetLoadBalancer(gomock.Any(), gomock.Eq(loadBalancerName)).
					Return(&osc.LoadBalancer{
						BackendVmIds: ptr.To([]string{vmId}),
					}, vtc.expGetLoadBalancerErr)
				if vtc.expGetLoadBalancerErr == nil {
					mockOscLoadBalancerInterface.
						EXPECT().
						UnlinkLoadBalancerBackendMachines(gomock.Any(), gomock.Eq([]string{vmId}), gomock.Eq(loadBalancerName)).
						Return(vtc.expUnlinkLoadBalancerBackendMachineErr)
				}
			} else {
				mockOscLoadBalancerInterface.
					EXPECT().
					GetLoadBalancer(gomock.Any(), gomock.Eq(loadBalancerName)).
					Return(&osc.LoadBalancer{
						BackendVmIds: ptr.To([]string{}),
					}, vtc.expGetLoadBalancerErr)
			}
			if vtc.expDeleteOutboundSecurityGroupRuleFound {
				mockOscSecurityGroupInterface.
					EXPECT().
					DeleteSecurityGroupRule(gomock.Any(), gomock.Eq(associateSecurityGroupId), gomock.Eq("Outbound"), gomock.Eq(ipProtocol), "", gomock.Eq(securityGroupIds[0]), gomock.Eq(fromPortRange), gomock.Eq(toPortRange)).
					Return(nil)
			}
			if vtc.expDeleteDedicatedPublicIpFound {
				mockOscPublicIpInterface.
					EXPECT().
					DeletePublicIp(gomock.Any(), gomock.Eq("eipassoc-test-osc-publicIp-uid")).
					Return(nil)
			}

			if vtc.expDeleteInboundSecurityGroupRuleFound {
				mockOscSecurityGroupInterface.
					EXPECT().
					DeleteSecurityGroupRule(gomock.Any(), gomock.Eq(associateSecurityGroupId), gomock.Eq("Inbound"), gomock.Eq(ipProtocol), "", gomock.Eq(securityGroupIds[0]), gomock.Eq(fromPortRange), gomock.Eq(toPortRange)).
					Return(nil)
			}

			if vtc.expGetLoadBalancerErr == nil && vtc.expUnlinkLoadBalancerBackendMachineErr == nil {
				mockOscVmInterface.
					EXPECT().
					DeleteVm(gomock.Any(), gomock.Eq(vmId)).
					Return(vtc.expDeleteVmErr)
			}
			reconcileDeleteVm, err := reconcileDeleteVm(ctx, clusterScope, machineScope, mockOscVmInterface, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface)
			if vtc.expReconcileDeleteVmErr != nil {
				require.EqualError(t, err, vtc.expReconcileDeleteVmErr.Error(), "reconcileDeleteVm() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("find reconcileDeleteVm %v\n", reconcileDeleteVm)
		})
	}
}

// TestReconcileDeleteVmUnlinkPublicIp has several tests to cover the code of the function reconcileDeleteVm
func TestReconcileDeleteVmUnlinkPublicIp(t *testing.T) {
	vmTestCases := []struct {
		name                        string
		clusterSpec                 infrastructurev1beta1.OscClusterSpec
		machineSpec                 infrastructurev1beta1.OscMachineSpec
		expCheckUnlinkPublicIpFound bool
		expGetVmFound               bool
		expGetVmErr                 error

		expCheckUnlinkPublicIpErr error
		expReconcileDeleteVmErr   error
	}{
		{
			name:                        "failed unlink volume",
			clusterSpec:                 defaultVmClusterReconcile,
			machineSpec:                 defaultVmReconcile,
			expGetVmFound:               true,
			expGetVmErr:                 nil,
			expCheckUnlinkPublicIpFound: true,
			expCheckUnlinkPublicIpErr:   errors.New("CheckUnlinkPublicIp generic error"),
			expReconcileDeleteVmErr:     errors.New("cannot unlink publicIp: CheckUnlinkPublicIp generic error"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope, ctx, mockOscVmInterface, _, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface, _ := SetupWithVmMock(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			vmId := vtc.machineSpec.Node.Vm.ResourceId

			var securityGroupIds []string
			vmSecurityGroups := machineScope.GetVmSecurityGroups()
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			for _, vmSecurityGroup := range vmSecurityGroups {
				securityGroupName := vmSecurityGroup.Name + "-uid"
				securityGroupId := "sg-" + securityGroupName
				securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
				securityGroupIds = append(securityGroupIds, securityGroupId)
			}
			publicIpName := vtc.machineSpec.Node.Vm.PublicIpName + "-uid"
			linkPublicIpId := "eipassoc-" + publicIpName
			linkPublicIpRef := machineScope.GetLinkPublicIpRef()
			linkPublicIpRef.ResourceMap = make(map[string]string)
			linkPublicIpRef.ResourceMap[publicIpName] = linkPublicIpId

			vm := &osc.Vm{
				VmId: &vmId,
			}
			if vtc.expGetVmFound {
				mockOscVmInterface.
					EXPECT().
					GetVm(gomock.Any(), gomock.Eq(vmId)).
					Return(vm, vtc.expGetVmErr)
			} else {
				mockOscVmInterface.
					EXPECT().
					GetVm(gomock.Any(), gomock.Eq(vmId)).
					Return(nil, vtc.expGetVmErr)
			}

			if vtc.expCheckUnlinkPublicIpFound {
				mockOscPublicIpInterface.
					EXPECT().
					UnlinkPublicIp(gomock.Any(), gomock.Eq(linkPublicIpId)).
					Return(vtc.expCheckUnlinkPublicIpErr)
			}

			reconcileDeleteVm, err := reconcileDeleteVm(ctx, clusterScope, machineScope, mockOscVmInterface, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface)
			if vtc.expReconcileDeleteVmErr != nil {
				require.EqualError(t, err, vtc.expReconcileDeleteVmErr.Error(), "reconcileDeleteVm() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("find reconcileDeleteVm %v\n", reconcileDeleteVm)
		})
	}
}

// TestReconcileDeleteVmInitializing tests deleting a not yet initialized VM (no vmId).
func TestReconcileDeleteVmInitializing(t *testing.T) {
	clusterSpec := defaultVmClusterReconcile
	machineSpec := defaultVmInitialize
	clusterScope, machineScope, ctx, mockOscVmInterface, _, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface, _ := SetupWithVmMock(t, "delete unitialized vm", clusterSpec, machineSpec)
	reconcileDeleteVm, err := reconcileDeleteVm(ctx, clusterScope, machineScope, mockOscVmInterface, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface)
	require.NoError(t, err)
	t.Logf("find reconcileDeleteVm %v\n", reconcileDeleteVm)
}

// TestReconcileDeleteVmSecurityGroup has several tests to cover the code of the function reconcileDeleteVm
func TestReconcileDeleteVmSecurityGroup(t *testing.T) {
	vmTestCases := []struct {
		name                                    string
		clusterSpec                             infrastructurev1beta1.OscClusterSpec
		machineSpec                             infrastructurev1beta1.OscMachineSpec
		expDeleteInboundSecurityGroupRuleFound  bool
		expDeleteOutboundSecurityGroupRuleFound bool
		expDeleteInboundSecurityGroupRuleErr    error
		expDeleteOutboundSecurityGroupRuleErr   error
		expSecurityGroupRuleFound               bool
		expReconcileDeleteVmErr                 error
	}{
		{
			name:                                    "failed to delete inbound securitygroup",
			clusterSpec:                             defaultVmClusterReconcile,
			machineSpec:                             defaultVmReconcile,
			expDeleteInboundSecurityGroupRuleFound:  true,
			expDeleteOutboundSecurityGroupRuleFound: true,
			expDeleteInboundSecurityGroupRuleErr:    errors.New("DeleteSecurityGroupRule generic error"),
			expDeleteOutboundSecurityGroupRuleErr:   nil,
			expSecurityGroupRuleFound:               true,
			expReconcileDeleteVmErr:                 errors.New("cannot delete inbound securityGroupRule: DeleteSecurityGroupRule generic error"),
		},
		{
			name:                                    "failed to delete outbound securitygroup",
			clusterSpec:                             defaultVmClusterReconcile,
			machineSpec:                             defaultVmReconcile,
			expDeleteInboundSecurityGroupRuleFound:  false,
			expDeleteOutboundSecurityGroupRuleFound: true,
			expSecurityGroupRuleFound:               true,
			expDeleteInboundSecurityGroupRuleErr:    nil,
			expDeleteOutboundSecurityGroupRuleErr:   errors.New("DeleteSecurityGroupRule generic error"),
			expReconcileDeleteVmErr:                 errors.New("cannot delete outbound securityGroupRule: DeleteSecurityGroupRule generic error"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope, ctx, mockOscVmInterface, _, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface, _ := SetupWithVmMock(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			vmId := vtc.machineSpec.Node.Vm.ResourceId

			var securityGroupIds []string
			vmSecurityGroups := machineScope.GetVmSecurityGroups()
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			for _, vmSecurityGroup := range vmSecurityGroups {
				securityGroupName := vmSecurityGroup.Name + "-uid"
				securityGroupId := "sg-" + securityGroupName
				securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
				securityGroupIds = append(securityGroupIds, securityGroupId)
			}
			publicIpName := vtc.machineSpec.Node.Vm.PublicIpName + "-uid"
			linkPublicIpId := "eipassoc-" + publicIpName
			linkPublicIpRef := machineScope.GetLinkPublicIpRef()
			linkPublicIpRef.ResourceMap = make(map[string]string)
			linkPublicIpRef.ResourceMap[publicIpName] = linkPublicIpId

			loadBalancerName := vtc.machineSpec.Node.Vm.LoadBalancerName
			loadBalancerSpec := clusterScope.GetLoadBalancer()
			loadBalancerSpec.SetDefaultValue()
			loadBalancerSecurityGroupName := loadBalancerSpec.SecurityGroupName
			ipProtocol := strings.ToLower(loadBalancerSpec.Listener.BackendProtocol)
			fromPortRange := loadBalancerSpec.Listener.BackendPort
			toPortRange := loadBalancerSpec.Listener.BackendPort
			loadBalancerSecurityGroupClusterScopeName := loadBalancerSecurityGroupName + "-uid"
			loadBalancerSecurityGroupId := "sg-" + loadBalancerSecurityGroupClusterScopeName
			securityGroupsRef.ResourceMap[loadBalancerSecurityGroupClusterScopeName] = loadBalancerSecurityGroupId
			associateSecurityGroupId := securityGroupsRef.ResourceMap[loadBalancerSecurityGroupClusterScopeName]

			vm := &osc.Vm{
				VmId: &vmId,
			}
			mockOscVmInterface.
				EXPECT().
				GetVm(gomock.Any(), gomock.Eq(vmId)).
				Return(vm, nil)

			mockOscPublicIpInterface.
				EXPECT().
				UnlinkPublicIp(gomock.Any(), gomock.Eq(linkPublicIpId)).
				Return(nil)
			mockOscLoadBalancerInterface.
				EXPECT().
				GetLoadBalancer(gomock.Any(), gomock.Eq(loadBalancerName)).
				Return(&osc.LoadBalancer{
					BackendVmIds: ptr.To([]string{vmId}),
				}, nil)
			mockOscLoadBalancerInterface.
				EXPECT().
				UnlinkLoadBalancerBackendMachines(gomock.Any(), gomock.Eq([]string{vmId}), gomock.Eq(loadBalancerName)).
				Return(nil)

			if vtc.expDeleteOutboundSecurityGroupRuleFound {
				mockOscSecurityGroupInterface.
					EXPECT().
					DeleteSecurityGroupRule(gomock.Any(), gomock.Eq(associateSecurityGroupId), gomock.Eq("Outbound"), gomock.Eq(ipProtocol), "", gomock.Eq(securityGroupIds[0]), gomock.Eq(fromPortRange), gomock.Eq(toPortRange)).
					Return(vtc.expDeleteOutboundSecurityGroupRuleErr)
			}

			if vtc.expDeleteInboundSecurityGroupRuleFound {
				mockOscSecurityGroupInterface.
					EXPECT().
					DeleteSecurityGroupRule(gomock.Any(), gomock.Eq(associateSecurityGroupId), gomock.Eq("Inbound"), gomock.Eq(ipProtocol), "", gomock.Eq(securityGroupIds[0]), gomock.Eq(fromPortRange), gomock.Eq(toPortRange)).
					Return(vtc.expDeleteInboundSecurityGroupRuleErr)
			}

			reconcileDeleteVm, err := reconcileDeleteVm(ctx, clusterScope, machineScope, mockOscVmInterface, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface)
			if vtc.expReconcileDeleteVmErr != nil {
				require.EqualError(t, err, vtc.expReconcileDeleteVmErr.Error(), "reconcileDeleteVm() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("find reconcileDeleteVm %v\n", reconcileDeleteVm)
		})
	}
}
