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
	"strings"
	"testing"
	"time"

	osc "github.com/outscale/osc-sdk-go/v2"

	"github.com/golang/mock/gomock"
	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/compute/mock_compute"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/security/mock_security"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/service/mock_service"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/storage/mock_storage"
	"github.com/stretchr/testify/assert"
)

var (
	defaultVmClusterInitialize = infrastructurev1beta1.OscClusterSpec{
		Network: infrastructurev1beta1.OscNetwork{
			Net: infrastructurev1beta1.OscNet{
				Name:    "test-net",
				IPRange: "10.0.0.0/16",
			},
			Subnets: []*infrastructurev1beta1.OscSubnet{
				{
					Name:          "test-subnet",
					IPSubnetRange: "10.0.0.0/24",
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
							IPProtocol:    "tcp",
							IPRange:       "0.0.0.0/0",
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
			PublicIPS: []*infrastructurev1beta1.OscPublicIP{
				{
					Name: "test-publicip",
				},
			},
		},
	}

	defaultVmClusterReconcile = infrastructurev1beta1.OscClusterSpec{
		Network: infrastructurev1beta1.OscNetwork{
			Net: infrastructurev1beta1.OscNet{
				Name:       "test-net",
				IPRange:    "10.0.0.0/16",
				ResourceID: "vpc-test-net-uid",
			},
			Subnets: []*infrastructurev1beta1.OscSubnet{
				{
					Name:          "test-subnet",
					IPSubnetRange: "10.0.0.0/24",
					ResourceID:    "subnet-test-subnet-uid",
				},
			},
			SecurityGroups: []*infrastructurev1beta1.OscSecurityGroup{
				{
					Name:        "test-securitygroup",
					Description: "test securitygroup",
					ResourceID:  "sg-test-securitygroup-uid",
					SecurityGroupRules: []infrastructurev1beta1.OscSecurityGroupRule{
						{
							Name:          "test-securitygrouprule",
							Flow:          "Inbound",
							IPProtocol:    "tcp",
							IPRange:       "0.0.0.0/0",
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
			PublicIPS: []*infrastructurev1beta1.OscPublicIP{
				{
					Name:       "test-publicip",
					ResourceID: "test-publicip-uid",
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
			VM: infrastructurev1beta1.OscVM{
				Name:             "test-vm",
				ImageID:          "ami-00000000",
				Role:             "controlplane",
				VolumeName:       "test-volume",
				DeviceName:       "/dev/xvdb",
				KeypairName:      "rke",
				SubregionName:    "eu-west-2a",
				SubnetName:       "test-subnet",
				LoadBalancerName: "test-loadbalancer",
				PublicIPName:     "test-publicip",
				VMType:           "tinav4.c2r4p2",
				SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
					{
						Name: "test-securitygroup",
					},
				},
				PrivateIPS: []infrastructurev1beta1.OscPrivateIPElement{
					{
						Name:      "test-privateip",
						PrivateIP: "10.0.0.17",
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
			VM: infrastructurev1beta1.OscVM{
				Name:             "test-vm",
				ImageID:          "ami-00000000",
				Role:             "controlplane",
				VolumeName:       "test-volume",
				DeviceName:       "/dev/xvdb",
				KeypairName:      "rke",
				SubregionName:    "eu-west-2a",
				SubnetName:       "test-subnet",
				LoadBalancerName: "test-loadbalancer",
				VMType:           "tinav4.c2r4p2",
				PublicIPName:     "test-publicip",
				SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
					{
						Name: "test-securitygroup",
					},
				},
				PrivateIPS: []infrastructurev1beta1.OscPrivateIPElement{
					{
						Name:      "test-privateip-first",
						PrivateIP: "10.0.0.17",
					},
					{
						Name:      "test-privateip-second",
						PrivateIP: "10.0.0.18",
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
					ResourceID:    "volume-test-volume-uid",
				},
			},
			VM: infrastructurev1beta1.OscVM{
				Name:             "test-vm",
				ImageID:          "ami-00000000",
				Role:             "controlplane",
				VolumeName:       "test-volume",
				DeviceName:       "/dev/xvdb",
				KeypairName:      "rke",
				SubregionName:    "eu-west-2a",
				SubnetName:       "test-subnet",
				LoadBalancerName: "test-loadbalancer",
				VMType:           "tinav4.c2r4p2",
				ResourceID:       "i-test-vm-uid",
				PublicIPName:     "test-publicip",
				SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
					{
						Name: "test-securitygroup",
					},
				},
				PrivateIPS: []infrastructurev1beta1.OscPrivateIPElement{
					{
						Name:      "test-privateip",
						PrivateIP: "10.0.0.17",
					},
				},
			},
		},
	}
)

// SetupWithVmMock set vmMock with clusterScope, machineScope and oscmachine
func SetupWithVmMock(t *testing.T, name string, clusterSpec infrastructurev1beta1.OscClusterSpec, machineSpec infrastructurev1beta1.OscMachineSpec) (clusterScope *scope.ClusterScope, machineScope *scope.MachineScope, ctx context.Context, mockOscVMInterface *mock_compute.MockOscVMInterface, mockOscVolumeInterface *mock_storage.MockOscVolumeInterface, mockOscPublicIPInterface *mock_security.MockOscPublicIPInterface, mockOscLoadBalancerInterface *mock_service.MockOscLoadBalancerInterface, mockOscSecurityGroupInterface *mock_security.MockOscSecurityGroupInterface) {
	clusterScope, machineScope = SetupMachine(t, name, clusterSpec, machineSpec)
	mockCtrl := gomock.NewController(t)
	mockOscVMInterface = mock_compute.NewMockOscVMInterface(mockCtrl)
	mockOscVolumeInterface = mock_storage.NewMockOscVolumeInterface(mockCtrl)
	mockOscPublicIPInterface = mock_security.NewMockOscPublicIPInterface(mockCtrl)
	mockOscLoadBalancerInterface = mock_service.NewMockOscLoadBalancerInterface(mockCtrl)
	mockOscSecurityGroupInterface = mock_security.NewMockOscSecurityGroupInterface(mockCtrl)
	ctx = context.Background()
	return clusterScope, machineScope, ctx, mockOscVMInterface, mockOscVolumeInterface, mockOscPublicIPInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface
}

// TestGetVmResourceID has several tests to cover the code of the function getVmResourceId
func TestGetVmResourceID(t *testing.T) {
	vmTestCases := []struct {
		name                  string
		clusterSpec           infrastructurev1beta1.OscClusterSpec
		machineSpec           infrastructurev1beta1.OscMachineSpec
		expVmFound            bool
		expGetVmResourceIDErr error
	}{
		{
			name:                  "get vm",
			clusterSpec:           defaultVmClusterInitialize,
			machineSpec:           defaultVmInitialize,
			expVmFound:            true,
			expGetVmResourceIDErr: nil,
		},
		{
			name:                  "can not get vm",
			clusterSpec:           defaultVmClusterInitialize,
			machineSpec:           defaultVmInitialize,
			expVmFound:            false,
			expGetVmResourceIDErr: fmt.Errorf("test-vm-uid does not exist"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			_, machineScope := SetupMachine(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			vmName := vtc.machineSpec.Node.VM.Name + "-uid"
			vmId := "i-" + vmName
			vmRef := machineScope.GetVMRef()
			vmRef.ResourceMap = make(map[string]string)
			if vtc.expVmFound {
				vmRef.ResourceMap[vmName] = vmId
			}
			VMResourceID, err := getVmResourceId(vmName, machineScope)
			if err != nil {
				assert.Equal(t, vtc.expGetVmResourceIDErr.Error(), err.Error(), "GetVmResourceId() should return the same error")
			} else {
				assert.Nil(t, vtc.expGetVmResourceIDErr)
			}
			t.Logf("find netResourceID %s", VMResourceID)
		})
	}
}

// TestCheckVmVolumeOscAssociateResourceName has several tests to cover the code of the function checkVMVolumeOscAssociateResourceName
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
			machineSpec: defaultVmInitialize,
			expCheckVmVolumeOscAssociateResourceNameErr: nil,
		},
		{
			name:        "check work without vm spec (with default values)",
			clusterSpec: defaultVmClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{},
			},
			expCheckVmVolumeOscAssociateResourceNameErr: fmt.Errorf("cluster-api-volume-kw-uid volume does not exist in vm"),
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
					VM: infrastructurev1beta1.OscVM{
						Name:             "test-vm",
						ImageID:          "ami-00000000",
						Role:             "controlplane",
						VolumeName:       "test-volume@test",
						DeviceName:       "/dev/xvdb",
						KeypairName:      "rke",
						SubregionName:    "eu-west-2a",
						SubnetName:       "test-subnet",
						LoadBalancerName: "test-loadbalancer",
						VMType:           "tinav4.c2r4p2",
						PublicIPName:     "test-publicip",
						SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
							{
								Name: "test-securitygroup",
							},
						},
						PrivateIPS: []infrastructurev1beta1.OscPrivateIPElement{
							{
								Name:      "test-privateip-first",
								PrivateIP: "10.0.0.17",
							},
							{
								Name:      "test-privateip-second",
								PrivateIP: "10.0.0.18",
							},
						},
					},
				},
			},
			expCheckVmVolumeOscAssociateResourceNameErr: fmt.Errorf("test-volume@test-uid volume does not exist in vm"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			_, machineScope := SetupMachine(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			err := checkVMVolumeOscAssociateResourceName(machineScope)
			if err != nil {
				assert.Equal(t, vtc.expCheckVmVolumeOscAssociateResourceNameErr, err, "checkVMVolumeOscAssociateResourceName() should return the same eror")
			} else {
				assert.Nil(t, vtc.expCheckVmVolumeOscAssociateResourceNameErr)
			}
		})
	}
}

// TestCheckVmLoadBalancerOscAssociateResourceName has several tests to cover the code of the function checkVMLoadBalancerOscAssociateResourceName
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
					VM: infrastructurev1beta1.OscVM{
						Role: "controlplane",
					},
				},
			},
			expCheckVmLoadBalancerOscAssociateResourceNameErr: fmt.Errorf("OscClusterApi-1-uid loadBalancer does not exist in vm"),
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
					VM: infrastructurev1beta1.OscVM{
						Name:             "test-vm",
						ImageID:          "ami-00000000",
						Role:             "controlplane",
						VolumeName:       "test-volume",
						DeviceName:       "/dev/xvdb",
						KeypairName:      "rke",
						SubregionName:    "eu-west-2a",
						SubnetName:       "test-subnet",
						LoadBalancerName: "test-loadbalancer@test",
						VMType:           "tinav4.c2r4p2",
						PublicIPName:     "test-publicip",
						SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
							{
								Name: "test-securitygroup",
							},
						},
						PrivateIPS: []infrastructurev1beta1.OscPrivateIPElement{
							{
								Name:      "test-privateip-first",
								PrivateIP: "10.0.0.17",
							},
							{
								Name:      "test-privateip-second",
								PrivateIP: "10.0.0.18",
							},
						},
					},
				},
			},
			expCheckVmLoadBalancerOscAssociateResourceNameErr: fmt.Errorf("test-loadbalancer@test-uid loadBalancer does not exist in vm"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope := SetupMachine(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			err := checkVMLoadBalancerOscAssociateResourceName(machineScope, clusterScope)
			if err != nil {
				assert.Equal(t, vtc.expCheckVmLoadBalancerOscAssociateResourceNameErr, err, "checkVMLoadBalancerOscAssociateResourceName() should return the same erroor")
			} else {
				assert.Nil(t, vtc.expCheckVmLoadBalancerOscAssociateResourceNameErr)
			}
		})
	}
}

// TestCheckVmSecurityGroupOscAssociateResourceName has several tests to cover the code of the function checkVMSecurityGroupOscAssociateResourceNam
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
			expCheckVmSecurityGroupOscAssociateResourceNameErr: fmt.Errorf("cluster-api-securitygroup-kw-uid securityGroup does not exist in vm"),
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
					VM: infrastructurev1beta1.OscVM{
						Name:             "test-vm",
						ImageID:          "ami-00000000",
						Role:             "controlplane",
						VolumeName:       "test-volume",
						DeviceName:       "/dev/xvdb",
						KeypairName:      "rke",
						SubregionName:    "eu-west-2a",
						SubnetName:       "test-subnet",
						LoadBalancerName: "test-loadbalancer",
						VMType:           "tinav4.c2r4p2",
						PublicIPName:     "test-publicip",
						SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
							{
								Name: "test-securitygroup@test",
							},
						},
						PrivateIPS: []infrastructurev1beta1.OscPrivateIPElement{
							{
								Name:      "test-privateip-first",
								PrivateIP: "10.0.0.17",
							},
							{
								Name:      "test-privateip-second",
								PrivateIP: "10.0.0.18",
							},
						},
					},
				},
			},
			expCheckVmSecurityGroupOscAssociateResourceNameErr: fmt.Errorf("test-securitygroup@test-uid securityGroup does not exist in vm"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope := SetupMachine(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			err := checkVMSecurityGroupOscAssociateResourceName(machineScope, clusterScope)
			if err != nil {
				assert.Equal(t, vtc.expCheckVmSecurityGroupOscAssociateResourceNameErr, err, "checkVMSecurityGroupOscAssociateResourceName() should return the same error")
			} else {
				assert.Nil(t, vtc.expCheckVmSecurityGroupOscAssociateResourceNameErr)
			}
		})
	}
}

// TestCheckVmPublicIPOscAssociateResourceName has several tests to cover the code of the function checkVMPublicIpOscAssociateResourceName
func TestCheckVmPublicIPOscAssociateResourceName(t *testing.T) {
	vmTestCases := []struct {
		name                                          string
		clusterSpec                                   infrastructurev1beta1.OscClusterSpec
		machineSpec                                   infrastructurev1beta1.OscMachineSpec
		expCheckVmPublicIPOscAssociateResourceNameErr error
	}{
		{
			name:        "check publicip association with vm",
			clusterSpec: defaultVmClusterInitialize,
			machineSpec: defaultVmInitialize,
			expCheckVmPublicIPOscAssociateResourceNameErr: nil,
		},
		{
			name:        "check work without vm spec (with default values)",
			clusterSpec: defaultVmClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					VM: infrastructurev1beta1.OscVM{
						PublicIPName: "cluster-api-publicip",
					},
				},
			},
			expCheckVmPublicIPOscAssociateResourceNameErr: fmt.Errorf("cluster-api-publicip-uid publicIp does not exist in vm"),
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
					VM: infrastructurev1beta1.OscVM{
						Name:             "test-vm",
						ImageID:          "ami-00000000",
						Role:             "controlplane",
						VolumeName:       "test-volume",
						DeviceName:       "/dev/xvdb",
						KeypairName:      "rke",
						SubregionName:    "eu-west-2a",
						SubnetName:       "test-subnet",
						LoadBalancerName: "test-loadbalancer",
						VMType:           "tinav4.c2r4p2",
						PublicIPName:     "test-publicip@test",
						SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
							{
								Name: "test-securitygroup",
							},
						},
						PrivateIPS: []infrastructurev1beta1.OscPrivateIPElement{
							{
								Name:      "test-privateip-first",
								PrivateIP: "10.0.0.17",
							},
							{
								Name:      "test-privateip-second",
								PrivateIP: "10.0.0.18",
							},
						},
					},
				},
			},
			expCheckVmPublicIPOscAssociateResourceNameErr: fmt.Errorf("test-publicip@test-uid publicIp does not exist in vm"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope := SetupMachine(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			err := checkVMPublicIPOscAssociateResourceName(machineScope, clusterScope)
			if err != nil {
				assert.Equal(t, vtc.expCheckVmPublicIPOscAssociateResourceNameErr, err, "checkVMPublicIpOscAssociateResourceName() should return the same error")
			} else {
				assert.Nil(t, vtc.expCheckVmPublicIPOscAssociateResourceNameErr)
			}
		})
	}
}

// TestCheckVmFormatParameters has several tests to cover the code of the function checkVMFormatParameter
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
					VM: infrastructurev1beta1.OscVM{
						Name:             "test-vm@test",
						ImageID:          "ami-00000000",
						Role:             "controlplane",
						VolumeName:       "test-volume",
						DeviceName:       "/dev/xvdb",
						KeypairName:      "rke",
						SubregionName:    "eu-west-2a",
						SubnetName:       "test-subnet",
						LoadBalancerName: "test-loadbalancer",
						VMType:           "tinav4.c2r4p2",
						PublicIPName:     "test-publicip",
						SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
							{
								Name: "test-securitygroup",
							},
						},
						PrivateIPS: []infrastructurev1beta1.OscPrivateIPElement{
							{
								Name:      "test-privateip-first",
								PrivateIP: "10.0.0.17",
							},
						},
					},
				},
			},
			expCheckVmFormatParametersErr: fmt.Errorf("invalid Tag Name"),
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
					VM: infrastructurev1beta1.OscVM{
						Name:             "test-vm",
						ImageID:          "omi-00000000",
						Role:             "controlplane",
						VolumeName:       "test-volume",
						DeviceName:       "/dev/xvdb",
						KeypairName:      "rke",
						SubregionName:    "eu-west-2a",
						SubnetName:       "test-subnet",
						LoadBalancerName: "test-loadbalancer",
						VMType:           "tinav4.c2r4p2",
						PublicIPName:     "test-publicip",
						SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
							{
								Name: "test-securitygroup",
							},
						},
						PrivateIPS: []infrastructurev1beta1.OscPrivateIPElement{
							{
								Name:      "test-privateip-first",
								PrivateIP: "10.0.0.17",
							},
						},
					},
				},
			},
			expCheckVmFormatParametersErr: fmt.Errorf("invalid imageId"),
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
					VM: infrastructurev1beta1.OscVM{
						Name:             "test-vm",
						ImageID:          "ami-00000000",
						Role:             "controlplane",
						VolumeName:       "test-volume",
						DeviceName:       "/dev/xvdb",
						KeypairName:      "rke λ",
						SubregionName:    "eu-west-2a",
						SubnetName:       "test-subnet",
						LoadBalancerName: "test-loadbalancer",
						VMType:           "tinav4.c2r4p2",
						PublicIPName:     "test-publicip",
						SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
							{
								Name: "test-securitygroup",
							},
						},
						PrivateIPS: []infrastructurev1beta1.OscPrivateIPElement{
							{
								Name:      "test-privateip-first",
								PrivateIP: "10.0.0.17",
							},
						},
					},
				},
			},
			expCheckVmFormatParametersErr: fmt.Errorf("invalid KeypairName"),
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
					VM: infrastructurev1beta1.OscVM{
						Name:             "test-vm",
						ImageID:          "ami-00000000",
						Role:             "controlplane",
						VolumeName:       "test-volume",
						DeviceName:       "/dev/xvab",
						KeypairName:      "rke",
						SubregionName:    "eu-west-2a",
						SubnetName:       "test-subnet",
						LoadBalancerName: "test-loadbalancer",
						VMType:           "tinav4.c2r4p2",
						PublicIPName:     "test-publicip",
						SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
							{
								Name: "test-securitygroup",
							},
						},
						PrivateIPS: []infrastructurev1beta1.OscPrivateIPElement{
							{
								Name:      "test-privateip-first",
								PrivateIP: "10.0.0.17",
							},
						},
					},
				},
			},
			expCheckVmFormatParametersErr: fmt.Errorf("invalid deviceName"),
		},
		{
			name:        "Check Bad VMType",
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
					VM: infrastructurev1beta1.OscVM{
						Name:             "test-vm",
						ImageID:          "ami-00000000",
						Role:             "controlplane",
						VolumeName:       "test-volume",
						DeviceName:       "/dev/xvaidb",
						KeypairName:      "rke",
						SubregionName:    "eu-west-2a",
						SubnetName:       "test-subnet",
						LoadBalancerName: "test-loadbalancer",
						VMType:           "awsv4.c2r4p2",
						PublicIPName:     "test-publicip",
						SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
							{
								Name: "test-securitygroup",
							},
						},
						PrivateIPS: []infrastructurev1beta1.OscPrivateIPElement{
							{
								Name:      "test-privateip-first",
								PrivateIP: "10.0.0.17",
							},
						},
					},
				},
			},
			expCheckVmFormatParametersErr: fmt.Errorf("invalid vmType"),
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
					VM: infrastructurev1beta1.OscVM{
						Name:             "test-vm",
						ImageID:          "ami-00000000",
						Role:             "controlplane",
						VolumeName:       "test-volume",
						DeviceName:       "/dev/xvdb",
						KeypairName:      "rke",
						SubregionName:    "eu-west-2a",
						SubnetName:       "test-subnet",
						LoadBalancerName: "test-loadbalancer",
						VMType:           "tinav4.c2r4p2",
						PublicIPName:     "test-publicip",
						SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
							{
								Name: "test-securitygroup",
							},
						},
						PrivateIPS: []infrastructurev1beta1.OscPrivateIPElement{
							{
								Name:      "test-privateip-first",
								PrivateIP: "10.245.0.17",
							},
						},
					},
				},
			},
			expCheckVmFormatParametersErr: fmt.Errorf("invalid ip in cidr"),
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
					VM: infrastructurev1beta1.OscVM{
						Name:             "test-vm",
						ImageID:          "ami-00000000",
						Role:             "controlplane",
						VolumeName:       "test-volume",
						DeviceName:       "/dev/xvdb",
						KeypairName:      "rke",
						SubregionName:    "eu-west-2c",
						SubnetName:       "test-subnet",
						LoadBalancerName: "test-loadbalancer",
						VMType:           "tinav4.c2r4p2",
						PublicIPName:     "test-publicip",
						SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
							{
								Name: "test-securitygroup",
							},
						},
						PrivateIPS: []infrastructurev1beta1.OscPrivateIPElement{
							{
								Name:      "test-privateip-first",
								PrivateIP: "10.0.0.17",
							},
						},
					},
				},
			},
			expCheckVmFormatParametersErr: fmt.Errorf("invalid subregionName"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope := SetupMachine(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			subnetName := vtc.machineSpec.Node.VM.SubnetName + "-uid"
			subnetId := "subnet-" + subnetName
			subnetRef := clusterScope.GetSubnetRef()
			subnetRef.ResourceMap = make(map[string]string)
			subnetRef.ResourceMap[subnetName] = subnetId
			vmName, err := checkVMFormatParameters(machineScope, clusterScope)
			if err != nil {
				assert.Equal(t, vtc.expCheckVmFormatParametersErr, err, "checkVMFormatParameters() should return the same error")
			} else {
				assert.Nil(t, vtc.expCheckVmFormatParametersErr)
			}
			t.Logf("find vmName %s\n", vmName)
		})
	}

}

// TestCheckVmSubnetAssociateResourceName has several tests to cover the code of the function checkVMSubnetAssociateResourceName
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
			expCheckVmSubnetAssociateResourceNameErr: fmt.Errorf("cluster-api-subnet-kw-uid subnet does not exist in vm"),
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
					VM: infrastructurev1beta1.OscVM{
						Name:             "test-vm",
						ImageID:          "ami-00000000",
						Role:             "controlplane",
						VolumeName:       "test-volume",
						DeviceName:       "/dev/xvdb",
						KeypairName:      "rke",
						SubregionName:    "eu-west-2a",
						SubnetName:       "test-subnet@test",
						LoadBalancerName: "test-loadbalancer",
						VMType:           "tinav4.c2r4p2",
						PublicIPName:     "test-publicip",
						SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
							{
								Name: "test-securitygroup",
							},
						},
						PrivateIPS: []infrastructurev1beta1.OscPrivateIPElement{
							{
								Name:      "test-privateip-first",
								PrivateIP: "10.0.0.17",
							},
							{
								Name:      "test-privateip-second",
								PrivateIP: "10.0.0.18",
							},
						},
					},
				},
			},
			expCheckVmSubnetAssociateResourceNameErr: fmt.Errorf("test-subnet@test-uid subnet does not exist in vm"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope := SetupMachine(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			err := checkVMSubnetOscAssociateResourceName(machineScope, clusterScope)
			if err != nil {
				assert.Equal(t, vtc.expCheckVmSubnetAssociateResourceNameErr, err, "checkVMSubnetOscAssociateResourceName() should return the same error")
			} else {
				assert.Nil(t, vtc.expCheckVmSubnetAssociateResourceNameErr)
			}
		})
	}
}

// TestCheckVmPrivateIPSOscDuplicateName has several tests to cover the code of the function checkVMPrivateIPsOscDuplicateName
func TestCheckVmPrivateIPSOscDuplicateName(t *testing.T) {
	vmTestCases := []struct {
		name                                    string
		clusterSpec                             infrastructurev1beta1.OscClusterSpec
		machineSpec                             infrastructurev1beta1.OscMachineSpec
		expCheckVmPrivateIPSOscDuplicateNameErr error
	}{
		{
			name:                                    "get separate name",
			clusterSpec:                             defaultVmClusterInitialize,
			machineSpec:                             defaultMultiVmInitialize,
			expCheckVmPrivateIPSOscDuplicateNameErr: nil,
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
					VM: infrastructurev1beta1.OscVM{
						Name:             "test-vm",
						ImageID:          "ami-00000000",
						Role:             "controlplane",
						VolumeName:       "test-volume",
						DeviceName:       "/dev/xvdb",
						KeypairName:      "rke",
						SubregionName:    "eu-west-2a",
						SubnetName:       "test-subnet",
						LoadBalancerName: "test-loadbalancer",
						VMType:           "tinav4.c2r4p2",
						PublicIPName:     "test-publicip",
						SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
							{
								Name: "test-securitygroup",
							},
						},
						PrivateIPS: []infrastructurev1beta1.OscPrivateIPElement{
							{
								Name:      "test-privateip-first",
								PrivateIP: "10.0.0.17",
							},
							{
								Name:      "test-privateip-first",
								PrivateIP: "10.0.0.17",
							},
						},
					},
				},
			},
			expCheckVmPrivateIPSOscDuplicateNameErr: fmt.Errorf("test-privateip-first already exist"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			_, machineScope := SetupMachine(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			duplicateResourceVmPrivateIPSErr := checkVMPrivateIPOscDuplicateName(machineScope)
			if duplicateResourceVmPrivateIPSErr != nil {
				assert.Equal(t, vtc.expCheckVmPrivateIPSOscDuplicateNameErr, duplicateResourceVmPrivateIPSErr, "checkVMPrivateIPsOscDuplicateName() should return the same error")
			} else {
				assert.Nil(t, vtc.expCheckVmPrivateIPSOscDuplicateNameErr)
			}
		})
	}
}

// TestCheckVmVolumeSubregionName has several tests to cover the code of the function checkVMVolumeSubregionName
func TestCheckVmVolumeSubregionName(t *testing.T) {
	vmTestCases := []struct {
		name                             string
		clusterSpec                      infrastructurev1beta1.OscClusterSpec
		machineSpec                      infrastructurev1beta1.OscMachineSpec
		expCheckVmVolumeSubregionNameErr error
	}{
		{
			name:                             "get the same volume and vm subregion name",
			clusterSpec:                      defaultVmClusterInitialize,
			machineSpec:                      defaultVmInitialize,
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
					VM: infrastructurev1beta1.OscVM{
						Name:             "test-vm",
						ImageID:          "ami-00000000",
						Role:             "controlplane",
						VolumeName:       "test-volume",
						DeviceName:       "/dev/xvdb",
						KeypairName:      "rke",
						SubregionName:    "eu-west-2b",
						SubnetName:       "test-subnet",
						LoadBalancerName: "test-loadbalancer",
						VMType:           "tinav4.c2r4p2",
						PublicIPName:     "test-publicip",
						SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
							{
								Name: "test-securitygroup",
							},
						},
						PrivateIPS: []infrastructurev1beta1.OscPrivateIPElement{
							{
								Name:      "test-privateip-first",
								PrivateIP: "10.0.0.17",
							},
						},
					},
				},
			},
			expCheckVmVolumeSubregionNameErr: fmt.Errorf("volume test-volume and vm test-vm are not in the same subregion eu-west-2b"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			_, machineScope := SetupMachine(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			err := checkVMVolumeSubregionName(machineScope)
			if err != nil {
				assert.Equal(t, vtc.expCheckVmVolumeSubregionNameErr, err, "checkVMVolumeSubregionName() should return the same error")
			} else {
				assert.Nil(t, vtc.expCheckVmVolumeSubregionNameErr)
			}
		})
	}
}

// TestReconcileVm has several tests to cover the code of the function reconcileVm
func TestReconcileVm(t *testing.T) {
	vmTestCases := []struct {
		name                                    string
		clusterSpec                             infrastructurev1beta1.OscClusterSpec
		machineSpec                             infrastructurev1beta1.OscMachineSpec
		expCreateVMFound                        bool
		expLinkPublicIPFound                    bool
		expCreateInboundSecurityGroupRuleFound  bool
		expCreateOutboundSecurityGroupRuleFound bool
		expCreateVMErr                          error
		expReconcileVmErr                       error
		expCheckVMStateBootErr                  error
		expCheckVolumeStateAvailableErr         error
		expLinkVolumeErr                        error
		expCheckVolumeStateUseErr               error
		expCheckVMStateVolumeErr                error
		expCreateInboundSecurityGroupRuleErr    error
		expCreateOutboundSecurityGroupRuleErr   error
		expLinkPublicIPErr                      error
		expCheckVMStatePublicIPErr              error
		expLinkLoadBalancerBackendMachineErr    error
	}{
		{
			name:                                    "create vm (first time reconcile loop)",
			clusterSpec:                             defaultVmClusterInitialize,
			machineSpec:                             defaultVmInitialize,
			expCreateVMFound:                        true,
			expLinkPublicIPFound:                    true,
			expCreateInboundSecurityGroupRuleFound:  true,
			expCreateOutboundSecurityGroupRuleFound: true,
			expCreateVMErr:                          nil,
			expCheckVMStateBootErr:                  nil,
			expCheckVolumeStateAvailableErr:         nil,
			expLinkVolumeErr:                        nil,
			expCheckVolumeStateUseErr:               nil,
			expCheckVMStateVolumeErr:                nil,
			expLinkPublicIPErr:                      nil,
			expCheckVMStatePublicIPErr:              nil,
			expLinkLoadBalancerBackendMachineErr:    nil,
			expCreateInboundSecurityGroupRuleErr:    nil,
			expCreateOutboundSecurityGroupRuleErr:   nil,
			expReconcileVmErr:                       nil,
		},
		{
			name:                                    "create two vms (first time reconcile loop)",
			clusterSpec:                             defaultVmClusterInitialize,
			machineSpec:                             defaultMultiVmInitialize,
			expCreateVMFound:                        true,
			expLinkPublicIPFound:                    true,
			expCreateInboundSecurityGroupRuleFound:  true,
			expCreateOutboundSecurityGroupRuleFound: true,
			expCreateVMErr:                          nil,
			expCheckVMStateBootErr:                  nil,
			expCheckVolumeStateAvailableErr:         nil,
			expLinkVolumeErr:                        nil,
			expCheckVolumeStateUseErr:               nil,
			expCheckVMStateVolumeErr:                nil,
			expLinkPublicIPErr:                      nil,
			expCheckVMStatePublicIPErr:              nil,
			expLinkLoadBalancerBackendMachineErr:    nil,
			expCreateInboundSecurityGroupRuleErr:    nil,
			expCreateOutboundSecurityGroupRuleErr:   nil,
			expReconcileVmErr:                       nil,
		},
		{
			name:                                    "user delete vm without cluster-api",
			clusterSpec:                             defaultVmClusterInitialize,
			machineSpec:                             defaultVmInitialize,
			expCreateVMFound:                        true,
			expLinkPublicIPFound:                    true,
			expCreateInboundSecurityGroupRuleFound:  true,
			expCreateOutboundSecurityGroupRuleFound: true,
			expCreateVMErr:                          nil,
			expCheckVMStateBootErr:                  nil,
			expCheckVolumeStateAvailableErr:         nil,
			expLinkVolumeErr:                        nil,
			expCheckVolumeStateUseErr:               nil,
			expCheckVMStateVolumeErr:                nil,
			expLinkPublicIPErr:                      nil,
			expCheckVMStatePublicIPErr:              nil,
			expLinkLoadBalancerBackendMachineErr:    nil,
			expCreateInboundSecurityGroupRuleErr:    nil,
			expCreateOutboundSecurityGroupRuleErr:   nil,
			expReconcileVmErr:                       nil,
		},
		{
			name:                                    "create two vm (first time reconcile loop)",
			clusterSpec:                             defaultVmClusterInitialize,
			machineSpec:                             defaultMultiVmInitialize,
			expCreateVMFound:                        true,
			expLinkPublicIPFound:                    true,
			expCreateInboundSecurityGroupRuleFound:  true,
			expCreateOutboundSecurityGroupRuleFound: true,
			expCreateVMErr:                          nil,
			expCheckVMStateBootErr:                  nil,
			expCheckVolumeStateAvailableErr:         nil,
			expLinkVolumeErr:                        nil,
			expCheckVolumeStateUseErr:               nil,
			expCheckVMStateVolumeErr:                nil,
			expLinkPublicIPErr:                      nil,
			expCheckVMStatePublicIPErr:              nil,
			expLinkLoadBalancerBackendMachineErr:    nil,
			expCreateInboundSecurityGroupRuleErr:    nil,
			expCreateOutboundSecurityGroupRuleErr:   nil,
			expReconcileVmErr:                       nil,
		},
		{
			name:                                    "user delete vm without cluster-api",
			clusterSpec:                             defaultVmClusterInitialize,
			machineSpec:                             defaultVmInitialize,
			expCreateVMFound:                        true,
			expLinkPublicIPFound:                    true,
			expCreateInboundSecurityGroupRuleFound:  true,
			expCreateOutboundSecurityGroupRuleFound: true,
			expCreateVMErr:                          nil,
			expCheckVMStateBootErr:                  nil,
			expCheckVolumeStateAvailableErr:         nil,
			expLinkVolumeErr:                        nil,
			expCheckVolumeStateUseErr:               nil,
			expCheckVMStateVolumeErr:                nil,
			expLinkPublicIPErr:                      nil,
			expCheckVMStatePublicIPErr:              nil,
			expLinkLoadBalancerBackendMachineErr:    nil,
			expCreateInboundSecurityGroupRuleErr:    nil,
			expCreateOutboundSecurityGroupRuleErr:   nil,
			expReconcileVmErr:                       nil,
		},
		{
			name:                                    "failed to create outbound securityGroupRule",
			clusterSpec:                             defaultVmClusterInitialize,
			machineSpec:                             defaultVmInitialize,
			expCreateVMFound:                        true,
			expLinkPublicIPFound:                    true,
			expCreateInboundSecurityGroupRuleFound:  false,
			expCreateOutboundSecurityGroupRuleFound: true,
			expCreateVMErr:                          nil,
			expCheckVMStateBootErr:                  nil,
			expCheckVolumeStateAvailableErr:         nil,
			expLinkVolumeErr:                        nil,
			expCheckVolumeStateUseErr:               nil,
			expCheckVMStateVolumeErr:                nil,
			expLinkPublicIPErr:                      nil,
			expCheckVMStatePublicIPErr:              nil,
			expLinkLoadBalancerBackendMachineErr:    nil,
			expCreateOutboundSecurityGroupRuleErr:   fmt.Errorf("CreateSecurityGroupRule generic error"),
			expCreateInboundSecurityGroupRuleErr:    nil,
			expReconcileVmErr:                       fmt.Errorf("CreateSecurityGroupRule generic error Can not create outbound securityGroupRule for OscCluster test-system/test-osc"),
		},
		{
			name:                                    "failed to create inbound securityGroupRule",
			clusterSpec:                             defaultVmClusterInitialize,
			machineSpec:                             defaultVmInitialize,
			expCreateVMFound:                        true,
			expLinkPublicIPFound:                    true,
			expCreateInboundSecurityGroupRuleFound:  false,
			expCreateOutboundSecurityGroupRuleFound: false,
			expCreateVMErr:                          nil,
			expCheckVMStateBootErr:                  nil,
			expCheckVolumeStateAvailableErr:         nil,
			expLinkVolumeErr:                        nil,
			expCheckVolumeStateUseErr:               nil,
			expCheckVMStateVolumeErr:                nil,
			expLinkPublicIPErr:                      nil,
			expCheckVMStatePublicIPErr:              nil,
			expLinkLoadBalancerBackendMachineErr:    nil,
			expCreateInboundSecurityGroupRuleErr:    fmt.Errorf("CreateSecurityGroupRule generic error"),
			expCreateOutboundSecurityGroupRuleErr:   nil,
			expReconcileVmErr:                       fmt.Errorf("CreateSecurityGroupRule generic error Can not create inbound securityGroupRule for OscCluster test-system/test-osc"),
		},
		{
			name:                                    "linkPublicIP does not exist",
			clusterSpec:                             defaultVmClusterInitialize,
			machineSpec:                             defaultVmInitialize,
			expCreateVMFound:                        true,
			expLinkPublicIPFound:                    false,
			expCreateInboundSecurityGroupRuleFound:  true,
			expCreateOutboundSecurityGroupRuleFound: true,
			expCreateVMErr:                          nil,
			expCheckVMStateBootErr:                  nil,
			expCheckVolumeStateAvailableErr:         nil,
			expLinkVolumeErr:                        nil,
			expCheckVolumeStateUseErr:               nil,
			expCheckVMStateVolumeErr:                nil,
			expLinkPublicIPErr:                      nil,
			expCheckVMStatePublicIPErr:              nil,
			expLinkLoadBalancerBackendMachineErr:    nil,
			expCreateInboundSecurityGroupRuleErr:    nil,
			expCreateOutboundSecurityGroupRuleErr:   nil,
			expReconcileVmErr:                       nil,
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope, ctx, mockOscVMInterface, mockOscVolumeInterface, mockOscPublicIPInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface := SetupWithVmMock(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			vmName := vtc.machineSpec.Node.VM.Name + "-uid"
			vmId := "i-" + vmName
			vmState := "running"

			volumeName := vtc.machineSpec.Node.VM.VolumeName + "-uid"
			volumeId := "vol-" + volumeName
			volumeRef := machineScope.GetVolumeRef()
			volumeRef.ResourceMap = make(map[string]string)
			volumeRef.ResourceMap[volumeName] = volumeId
			volumeStateAvailable := "available"
			volumeStateUse := "in-use"

			subnetName := vtc.machineSpec.Node.VM.SubnetName + "-uid"
			subnetId := "subnet-" + subnetName
			subnetRef := clusterScope.GetSubnetRef()
			subnetRef.ResourceMap = make(map[string]string)
			subnetRef.ResourceMap[subnetName] = subnetId

			publicIpName := vtc.machineSpec.Node.VM.PublicIPName + "-uid"
			publicIpId := "eipalloc-" + publicIpName
			publicIpRef := clusterScope.GetPublicIPRef()
			publicIpRef.ResourceMap = make(map[string]string)
			publicIpRef.ResourceMap[publicIpName] = publicIpId

			linkPublicIPID := "eipassoc-" + publicIpName
			linkPublicIPRef := machineScope.GetLinkPublicIPRef()
			linkPublicIPRef.ResourceMap = make(map[string]string)
			if vtc.expLinkPublicIPFound {
				linkPublicIPRef.ResourceMap[vmName] = linkPublicIPID
			}

			var privateIPS []string
			VMPrivateIPS := machineScope.GetVMPrivateIPS()
			for _, VMPrivateIP := range *VMPrivateIPS {
				privateIP := VMPrivateIP.PrivateIP
				privateIPS = append(privateIPS, privateIP)
			}

			var securityGroupIds []string
			vmSecurityGroups := machineScope.GetVMSecurityGroups()
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			for _, vmSecurityGroup := range *vmSecurityGroups {
				securityGroupName := vmSecurityGroup.Name + "-uid"
				securityGroupId := "sg-" + securityGroupName
				securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
				securityGroupIds = append(securityGroupIds, securityGroupId)
			}

			deviceName := vtc.machineSpec.Node.VM.DeviceName
			vmSpec := vtc.machineSpec.Node.VM
			var clockInsideLoop time.Duration = 5
			var clockLoop time.Duration = 60
			var firstClockLoop time.Duration = 120
			loadBalancerName := vtc.machineSpec.Node.VM.LoadBalancerName
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
			linkPublicIP := osc.LinkPublicIpResponse{
				LinkPublicIpId: &linkPublicIPID,
			}
			securityGroupRule := osc.CreateSecurityGroupRuleResponse{
				SecurityGroup: &osc.SecurityGroup{
					SecurityGroupId: &loadBalancerSecurityGroupId,
				},
			}
			vm := &createVm[0]
			if vtc.expCreateVMFound {
				mockOscVMInterface.
					EXPECT().
					CreateVM(gomock.Eq(machineScope), gomock.Eq(&vmSpec), gomock.Eq(subnetId), gomock.Eq(securityGroupIds), gomock.Eq(privateIPS), gomock.Eq(vmName)).
					Return(vm, vtc.expCreateVMErr)
			} else {
				mockOscVMInterface.
					EXPECT().
					CreateVM(gomock.Eq(machineScope), gomock.Eq(vmSpec), gomock.Eq(subnetId), gomock.Eq(securityGroupIds), gomock.Eq(privateIPS), gomock.Eq(vmName)).
					Return(nil, vtc.expCreateVMErr)
			}

			mockOscVMInterface.
				EXPECT().
				CheckVMState(gomock.Eq(clockInsideLoop), gomock.Eq(firstClockLoop), gomock.Eq(vmState), gomock.Eq(vmId)).
				Return(vtc.expCheckVMStateBootErr)

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

			mockOscVMInterface.
				EXPECT().
				CheckVMState(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(vmState), gomock.Eq(vmId)).
				Return(vtc.expCheckVMStateVolumeErr)

			if vtc.expLinkPublicIPFound {
				mockOscPublicIPInterface.
					EXPECT().
					LinkPublicIP(gomock.Eq(publicIpId), gomock.Eq(vmId)).
					Return(*linkPublicIP.LinkPublicIpId, vtc.expLinkPublicIPErr)
			} else {
				mockOscPublicIPInterface.
					EXPECT().
					LinkPublicIP(gomock.Eq(publicIpId), gomock.Eq(vmId)).
					Return("", vtc.expLinkPublicIPErr)
			}

			mockOscVMInterface.
				EXPECT().
				CheckVMState(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(vmState), gomock.Eq(vmId)).
				Return(vtc.expCheckVMStatePublicIPErr)

			vmIds := []string{vmId}
			mockOscLoadBalancerInterface.
				EXPECT().
				LinkLoadBalancerBackendMachines(gomock.Eq(vmIds), gomock.Eq(loadBalancerName)).
				Return(vtc.expLinkLoadBalancerBackendMachineErr)

			if vtc.expCreateOutboundSecurityGroupRuleFound {
				mockOscSecurityGroupInterface.
					EXPECT().
					CreateSecurityGroupRule(gomock.Eq(associateSecurityGroupId), gomock.Eq("Outbound"), gomock.Eq(ipProtocol), "", gomock.Eq(securityGroupIds[0]), gomock.Eq(fromPortRange), gomock.Eq(toPortRange)).
					Return(securityGroupRule.SecurityGroup, vtc.expCreateOutboundSecurityGroupRuleErr)
			} else {
				mockOscSecurityGroupInterface.
					EXPECT().
					CreateSecurityGroupRule(gomock.Eq(associateSecurityGroupId), gomock.Eq("Outbound"), gomock.Eq(ipProtocol), "", gomock.Eq(securityGroupIds[0]), gomock.Eq(fromPortRange), gomock.Eq(toPortRange)).
					Return(nil, vtc.expCreateOutboundSecurityGroupRuleErr)
			}

			if vtc.expCreateInboundSecurityGroupRuleFound {
				mockOscSecurityGroupInterface.
					EXPECT().
					CreateSecurityGroupRule(gomock.Eq(associateSecurityGroupId), gomock.Eq("Inbound"), gomock.Eq(ipProtocol), "", gomock.Eq(securityGroupIds[0]), gomock.Eq(fromPortRange), gomock.Eq(toPortRange)).
					Return(securityGroupRule.SecurityGroup, vtc.expCreateInboundSecurityGroupRuleErr)

			} else if !vtc.expCreateInboundSecurityGroupRuleFound && vtc.expCreateOutboundSecurityGroupRuleFound {
			} else {
				mockOscSecurityGroupInterface.
					EXPECT().
					CreateSecurityGroupRule(gomock.Eq(associateSecurityGroupId), gomock.Eq("Inbound"), gomock.Eq(ipProtocol), "", gomock.Eq(securityGroupIds[0]), gomock.Eq(fromPortRange), gomock.Eq(toPortRange)).
					Return(nil, vtc.expCreateInboundSecurityGroupRuleErr)

			}
			reconcileVm, err := reconcileVm(ctx, clusterScope, machineScope, mockOscVMInterface, mockOscVolumeInterface, mockOscPublicIPInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface)
			if err != nil {
				assert.Equal(t, vtc.expReconcileVmErr.Error(), err.Error(), "reconcileVm() should return the same error")
			} else {
				assert.Nil(t, vtc.expReconcileVmErr)
			}
			t.Logf("find reconcileVm %v\n", reconcileVm)
		})
	}
}

// TestReconcileVmCreate has several tests to cover the code of the function reconcileVm
func TestReconcileVmCreate(t *testing.T) {
	vmTestCases := []struct {
		name              string
		clusterSpec       infrastructurev1beta1.OscClusterSpec
		machineSpec       infrastructurev1beta1.OscMachineSpec
		expCreateVMFound  bool
		expCreateVMErr    error
		expReconcileVmErr error
	}{
		{
			name:              "failed to create vm",
			clusterSpec:       defaultVmClusterInitialize,
			machineSpec:       defaultVmInitialize,
			expCreateVMFound:  false,
			expCreateVMErr:    fmt.Errorf("CreateVm generic error"),
			expReconcileVmErr: fmt.Errorf("CreateVm generic error Can not create vm for OscMachine test-system/test-osc"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope, ctx, mockOscVMInterface, mockOscVolumeInterface, mockOscPublicIPInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface := SetupWithVmMock(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			vmName := vtc.machineSpec.Node.VM.Name + "-uid"
			vmId := "i-" + vmName

			volumeName := vtc.machineSpec.Node.VM.VolumeName + "-uid"
			volumeId := "vol-" + volumeName
			volumeRef := machineScope.GetVolumeRef()
			volumeRef.ResourceMap = make(map[string]string)
			volumeRef.ResourceMap[volumeName] = volumeId

			subnetName := vtc.machineSpec.Node.VM.SubnetName + "-uid"
			subnetId := "subnet-" + subnetName
			subnetRef := clusterScope.GetSubnetRef()
			subnetRef.ResourceMap = make(map[string]string)
			subnetRef.ResourceMap[subnetName] = subnetId

			publicIpName := vtc.machineSpec.Node.VM.PublicIPName + "-uid"
			publicIpId := "eipalloc-" + publicIpName
			publicIpRef := clusterScope.GetPublicIPRef()
			publicIpRef.ResourceMap = make(map[string]string)
			publicIpRef.ResourceMap[publicIpName] = publicIpId

			linkPublicIPID := "eipassoc-" + publicIpName
			linkPublicIPRef := machineScope.GetLinkPublicIPRef()
			linkPublicIPRef.ResourceMap = make(map[string]string)
			linkPublicIPRef.ResourceMap[vmName] = linkPublicIPID

			var privateIPS []string
			VMPrivateIPS := machineScope.GetVMPrivateIPS()
			for _, VMPrivateIP := range *VMPrivateIPS {
				privateIP := VMPrivateIP.PrivateIP
				privateIPS = append(privateIPS, privateIP)
			}

			var securityGroupIds []string
			vmSecurityGroups := machineScope.GetVMSecurityGroups()
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			for _, vmSecurityGroup := range *vmSecurityGroups {
				securityGroupName := vmSecurityGroup.Name + "-uid"
				securityGroupId := "sg-" + securityGroupName
				securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
				securityGroupIds = append(securityGroupIds, securityGroupId)
			}
			vmSpec := vtc.machineSpec.Node.VM
			createVms := osc.CreateVmsResponse{
				Vms: &[]osc.Vm{
					{
						VmId: &vmId,
					},
				},
			}

			createVm := *createVms.Vms
			vm := &createVm[0]
			if vtc.expCreateVMFound {
				mockOscVMInterface.
					EXPECT().
					CreateVM(gomock.Eq(machineScope), gomock.Eq(&vmSpec), gomock.Eq(subnetId), gomock.Eq(securityGroupIds), gomock.Eq(privateIPS), gomock.Eq(vmName)).
					Return(vm, vtc.expCreateVMErr)
			} else {
				mockOscVMInterface.
					EXPECT().
					CreateVM(gomock.Eq(machineScope), gomock.Eq(&vmSpec), gomock.Eq(subnetId), gomock.Eq(securityGroupIds), gomock.Eq(privateIPS), gomock.Eq(vmName)).
					Return(nil, vtc.expCreateVMErr)
			}

			reconcileVm, err := reconcileVm(ctx, clusterScope, machineScope, mockOscVMInterface, mockOscVolumeInterface, mockOscPublicIPInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface)
			if err != nil {
				assert.Equal(t, vtc.expReconcileVmErr.Error(), err.Error(), "reconcileVm() should return the same error")
			} else {
				assert.Nil(t, vtc.expReconcileVmErr)
			}
			t.Logf("find reconcileVm %v\n", reconcileVm)
		})
	}
}

// TestReconcileVmLink has several tests to cover the code of the function reconcileVm
func TestReconcileVmLink(t *testing.T) {
	vmTestCases := []struct {
		name                              string
		clusterSpec                       infrastructurev1beta1.OscClusterSpec
		machineSpec                       infrastructurev1beta1.OscMachineSpec
		expCreateVMFound                  bool
		expLinkVolumeFound                bool
		expCheckVMStateBootFound          bool
		expCheckVolumeStateAvailableFound bool
		expCreateVMErr                    error
		expReconcileVmErr                 error
		expCheckVMStateBootErr            error
		expCheckVolumeStateAvailableErr   error
		expLinkVolumeErr                  error
	}{
		{
			name:                              "failed to link volume with vm",
			clusterSpec:                       defaultVmClusterInitialize,
			machineSpec:                       defaultVmInitialize,
			expCreateVMFound:                  true,
			expCreateVMErr:                    nil,
			expCheckVMStateBootFound:          true,
			expCheckVMStateBootErr:            nil,
			expCheckVolumeStateAvailableFound: true,
			expCheckVolumeStateAvailableErr:   nil,
			expLinkVolumeFound:                true,
			expLinkVolumeErr:                  fmt.Errorf("LinkVolume generic error"),
			expReconcileVmErr:                 fmt.Errorf("LinkVolume generic error Can not link volume vol-test-volume-uid with vm i-test-vm-uid for OscMachine test-system/test-osc"),
		},
		{
			name:                              "failed check vm state boot",
			clusterSpec:                       defaultVmClusterInitialize,
			machineSpec:                       defaultVmInitialize,
			expCreateVMFound:                  true,
			expCreateVMErr:                    nil,
			expCheckVMStateBootFound:          true,
			expCheckVMStateBootErr:            fmt.Errorf("checkVMState generic error"),
			expCheckVolumeStateAvailableFound: false,
			expCheckVolumeStateAvailableErr:   nil,
			expLinkVolumeFound:                false,
			expLinkVolumeErr:                  nil,
			expReconcileVmErr:                 fmt.Errorf("checkVMState generic error Can not get vm i-test-vm-uid running for OscMachine test-system/test-osc"),
		},
		{
			name:                              "failed check volume state boot",
			clusterSpec:                       defaultVmClusterInitialize,
			machineSpec:                       defaultVmInitialize,
			expCreateVMFound:                  true,
			expCreateVMErr:                    nil,
			expCheckVMStateBootFound:          true,
			expCheckVMStateBootErr:            nil,
			expCheckVolumeStateAvailableFound: true,
			expCheckVolumeStateAvailableErr:   fmt.Errorf("checkVolumeState generic error"),
			expLinkVolumeFound:                false,
			expLinkVolumeErr:                  nil,
			expReconcileVmErr:                 fmt.Errorf("checkVolumeState generic error Can not get volume vol-test-volume-uid available for OscMachine test-system/test-osc"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope, ctx, mockOscVMInterface, mockOscVolumeInterface, mockOscPublicIPInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface := SetupWithVmMock(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			vmName := vtc.machineSpec.Node.VM.Name + "-uid"
			vmId := "i-" + vmName
			vmState := "running"

			volumeName := vtc.machineSpec.Node.VM.VolumeName + "-uid"
			volumeId := "vol-" + volumeName
			volumeRef := machineScope.GetVolumeRef()
			volumeRef.ResourceMap = make(map[string]string)
			volumeRef.ResourceMap[volumeName] = volumeId
			volumeStateAvailable := "available"

			subnetName := vtc.machineSpec.Node.VM.SubnetName + "-uid"
			subnetId := "subnet-" + subnetName
			subnetRef := clusterScope.GetSubnetRef()
			subnetRef.ResourceMap = make(map[string]string)
			subnetRef.ResourceMap[subnetName] = subnetId

			publicIpName := vtc.machineSpec.Node.VM.PublicIPName + "-uid"
			publicIpId := "eipalloc-" + publicIpName
			publicIpRef := clusterScope.GetPublicIPRef()
			publicIpRef.ResourceMap = make(map[string]string)
			publicIpRef.ResourceMap[publicIpName] = publicIpId

			linkPublicIPID := "eipassoc-" + publicIpName
			linkPublicIPRef := machineScope.GetLinkPublicIPRef()
			linkPublicIPRef.ResourceMap = make(map[string]string)
			linkPublicIPRef.ResourceMap[vmName] = linkPublicIPID

			var privateIPS []string
			VMPrivateIPS := machineScope.GetVMPrivateIPS()
			for _, VMPrivateIP := range *VMPrivateIPS {
				privateIP := VMPrivateIP.PrivateIP
				privateIPS = append(privateIPS, privateIP)
			}

			var securityGroupIds []string
			vmSecurityGroups := machineScope.GetVMSecurityGroups()
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			for _, vmSecurityGroup := range *vmSecurityGroups {
				securityGroupName := vmSecurityGroup.Name + "-uid"
				securityGroupId := "sg-" + securityGroupName
				securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
				securityGroupIds = append(securityGroupIds, securityGroupId)
			}

			vmSpec := vtc.machineSpec.Node.VM
			createVms := osc.CreateVmsResponse{
				Vms: &[]osc.Vm{
					{
						VmId: &vmId,
					},
				},
			}

			createVm := *createVms.Vms
			vm := &createVm[0]
			deviceName := vtc.machineSpec.Node.VM.DeviceName
			var clockInsideLoop time.Duration = 5
			var clockLoop time.Duration = 60
			var firstClockLoop time.Duration = 120
			if vtc.expCreateVMFound {
				mockOscVMInterface.
					EXPECT().
					CreateVM(gomock.Eq(machineScope), gomock.Eq(&vmSpec), gomock.Eq(subnetId), gomock.Eq(securityGroupIds), gomock.Eq(privateIPS), gomock.Eq(vmName)).
					Return(vm, vtc.expCreateVMErr)
			} else {
				mockOscVMInterface.
					EXPECT().
					CreateVM(gomock.Eq(machineScope), gomock.Eq(vmSpec), gomock.Eq(subnetId), gomock.Eq(securityGroupIds), gomock.Eq(privateIPS), gomock.Eq(vmName)).
					Return(nil, vtc.expCreateVMErr)
			}

			if vtc.expCheckVMStateBootFound {
				mockOscVMInterface.
					EXPECT().
					CheckVMState(gomock.Eq(clockInsideLoop), gomock.Eq(firstClockLoop), gomock.Eq(vmState), gomock.Eq(vmId)).
					Return(vtc.expCheckVMStateBootErr)
			}
			if vtc.expCheckVolumeStateAvailableFound {
				mockOscVolumeInterface.
					EXPECT().
					CheckVolumeState(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(volumeStateAvailable), gomock.Eq(volumeId)).
					Return(vtc.expCheckVolumeStateAvailableErr)

			}
			if vtc.expLinkVolumeFound {
				mockOscVolumeInterface.
					EXPECT().
					LinkVolume(gomock.Eq(volumeId), gomock.Eq(vmId), gomock.Eq(deviceName)).
					Return(vtc.expLinkVolumeErr)
			}

			reconcileVm, err := reconcileVm(ctx, clusterScope, machineScope, mockOscVMInterface, mockOscVolumeInterface, mockOscPublicIPInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface)
			if err != nil {
				assert.Equal(t, vtc.expReconcileVmErr.Error(), err.Error(), "reconcileVm() should return the same error")
			} else {
				assert.Nil(t, vtc.expReconcileVmErr)
			}
			t.Logf("find reconcileVm %v\n", reconcileVm)
		})
	}
}

// TestReconcileVmLinkPubicIp has several tests to cover the code of the function reconcileVm
func TestReconcileVmLinkPubicIp(t *testing.T) {
	vmTestCases := []struct {
		name                            string
		clusterSpec                     infrastructurev1beta1.OscClusterSpec
		machineSpec                     infrastructurev1beta1.OscMachineSpec
		expCreateVMFound                bool
		expCheckVolumeStateUseFound     bool
		expCheckVMStateVolumeFound      bool
		expCheckVMStatePublicIPFound    bool
		expLinkPublicIPFound            bool
		expCreateVMErr                  error
		expReconcileVmErr               error
		expCheckVMStateBootErr          error
		expCheckVolumeStateAvailableErr error
		expLinkVolumeErr                error
		expCheckVolumeStateUseErr       error
		expCheckVMStateVolumeErr        error
		expLinkPublicIPErr              error
		expCheckVMStatePublicIPErr      error
	}{
		{
			name:                            "failed linkPublicIP",
			clusterSpec:                     defaultVmClusterInitialize,
			machineSpec:                     defaultVmInitialize,
			expCreateVMFound:                true,
			expLinkPublicIPFound:            true,
			expCheckVolumeStateUseFound:     true,
			expCheckVMStateVolumeFound:      true,
			expCheckVMStatePublicIPFound:    false,
			expCreateVMErr:                  nil,
			expCheckVMStateBootErr:          nil,
			expCheckVolumeStateAvailableErr: nil,
			expLinkVolumeErr:                nil,
			expCheckVolumeStateUseErr:       nil,
			expCheckVMStateVolumeErr:        nil,
			expLinkPublicIPErr:              fmt.Errorf("linkPublicIp generic error"),
			expCheckVMStatePublicIPErr:      nil,
			expReconcileVmErr:               fmt.Errorf("linkPublicIp generic error Can not link publicIp  eipalloc-test-publicip-uid with i-test-vm-uid for OscCluster test-system/test-osc"),
		},
		{
			name:                            "failed VmStatePublicIP",
			clusterSpec:                     defaultVmClusterInitialize,
			machineSpec:                     defaultVmInitialize,
			expCreateVMFound:                true,
			expLinkPublicIPFound:            true,
			expCheckVolumeStateUseFound:     true,
			expCheckVMStateVolumeFound:      true,
			expCheckVMStatePublicIPFound:    true,
			expCreateVMErr:                  nil,
			expCheckVMStateBootErr:          nil,
			expCheckVolumeStateAvailableErr: nil,
			expLinkVolumeErr:                nil,
			expCheckVolumeStateUseErr:       nil,
			expCheckVMStateVolumeErr:        nil,
			expLinkPublicIPErr:              nil,
			expCheckVMStatePublicIPErr:      fmt.Errorf("CheckVmState generic error"),
			expReconcileVmErr:               fmt.Errorf("CheckVmState generic error Can not get vm i-test-vm-uid running for OscMachine test-system/test-osc"),
		},
		{
			name:                            "failed VolumeStateUse",
			clusterSpec:                     defaultVmClusterInitialize,
			machineSpec:                     defaultVmInitialize,
			expCreateVMFound:                true,
			expLinkPublicIPFound:            false,
			expCheckVolumeStateUseFound:     true,
			expCheckVMStateVolumeFound:      false,
			expCheckVMStatePublicIPFound:    false,
			expCreateVMErr:                  nil,
			expCheckVMStateBootErr:          nil,
			expCheckVolumeStateAvailableErr: nil,
			expLinkVolumeErr:                nil,
			expCheckVolumeStateUseErr:       fmt.Errorf("CheckVolumeState generic error"),
			expCheckVMStateVolumeErr:        nil,
			expLinkPublicIPErr:              nil,
			expCheckVMStatePublicIPErr:      nil,
			expReconcileVmErr:               fmt.Errorf("CheckVolumeState generic error Can not get volume vol-test-volume-uid in use for OscMachine test-system/test-osc"),
		},
		{
			name:                            "failed VmStateVolume",
			clusterSpec:                     defaultVmClusterInitialize,
			machineSpec:                     defaultVmInitialize,
			expCreateVMFound:                true,
			expLinkPublicIPFound:            false,
			expCheckVolumeStateUseFound:     true,
			expCheckVMStateVolumeFound:      true,
			expCheckVMStatePublicIPFound:    false,
			expCreateVMErr:                  nil,
			expCheckVMStateBootErr:          nil,
			expCheckVolumeStateAvailableErr: nil,
			expLinkVolumeErr:                nil,
			expCheckVolumeStateUseErr:       nil,
			expCheckVMStateVolumeErr:        fmt.Errorf("CheckVmState generic error"),
			expLinkPublicIPErr:              nil,
			expCheckVMStatePublicIPErr:      nil,
			expReconcileVmErr:               fmt.Errorf("CheckVmState generic error Can not get vm i-test-vm-uid running for OscMachine test-system/test-osc"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope, ctx, mockOscVMInterface, mockOscVolumeInterface, mockOscPublicIPInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface := SetupWithVmMock(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			vmName := vtc.machineSpec.Node.VM.Name + "-uid"
			vmId := "i-" + vmName
			vmState := "running"

			volumeName := vtc.machineSpec.Node.VM.VolumeName + "-uid"
			volumeId := "vol-" + volumeName
			volumeRef := machineScope.GetVolumeRef()
			volumeRef.ResourceMap = make(map[string]string)
			volumeRef.ResourceMap[volumeName] = volumeId
			volumeStateAvailable := "available"
			volumeStateUse := "in-use"

			subnetName := vtc.machineSpec.Node.VM.SubnetName + "-uid"
			subnetId := "subnet-" + subnetName
			subnetRef := clusterScope.GetSubnetRef()
			subnetRef.ResourceMap = make(map[string]string)
			subnetRef.ResourceMap[subnetName] = subnetId

			publicIpName := vtc.machineSpec.Node.VM.PublicIPName + "-uid"
			publicIpId := "eipalloc-" + publicIpName
			publicIpRef := clusterScope.GetPublicIPRef()
			publicIpRef.ResourceMap = make(map[string]string)
			publicIpRef.ResourceMap[publicIpName] = publicIpId

			linkPublicIPID := "eipassoc-" + publicIpName
			linkPublicIPRef := machineScope.GetLinkPublicIPRef()
			linkPublicIPRef.ResourceMap = make(map[string]string)
			linkPublicIPRef.ResourceMap[vmName] = linkPublicIPID

			var privateIPS []string
			VMPrivateIPS := machineScope.GetVMPrivateIPS()
			for _, VMPrivateIP := range *VMPrivateIPS {
				privateIP := VMPrivateIP.PrivateIP
				privateIPS = append(privateIPS, privateIP)
			}

			var securityGroupIds []string
			vmSecurityGroups := machineScope.GetVMSecurityGroups()
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			for _, vmSecurityGroup := range *vmSecurityGroups {
				securityGroupName := vmSecurityGroup.Name + "-uid"
				securityGroupId := "sg-" + securityGroupName
				securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
				securityGroupIds = append(securityGroupIds, securityGroupId)
			}

			deviceName := vtc.machineSpec.Node.VM.DeviceName
			vmSpec := vtc.machineSpec.Node.VM
			var clockInsideLoop time.Duration = 5
			var clockLoop time.Duration = 60
			var firstClockLoop time.Duration = 120
			createVms := osc.CreateVmsResponse{
				Vms: &[]osc.Vm{
					{
						VmId: &vmId,
					},
				},
			}

			createVm := *createVms.Vms
			vm := &createVm[0]
			if vtc.expCreateVMFound {
				mockOscVMInterface.
					EXPECT().
					CreateVM(gomock.Eq(machineScope), gomock.Eq(&vmSpec), gomock.Eq(subnetId), gomock.Eq(securityGroupIds), gomock.Eq(privateIPS), gomock.Eq(vmName)).
					Return(vm, vtc.expCreateVMErr)
			} else {
				mockOscVMInterface.
					EXPECT().
					CreateVM(gomock.Eq(machineScope), gomock.Eq(vmSpec), gomock.Eq(subnetId), gomock.Eq(securityGroupIds), gomock.Eq(privateIPS), gomock.Eq(vmName)).
					Return(nil, vtc.expCreateVMErr)
			}

			mockOscVMInterface.
				EXPECT().
				CheckVMState(gomock.Eq(clockInsideLoop), gomock.Eq(firstClockLoop), gomock.Eq(vmState), gomock.Eq(vmId)).
				Return(vtc.expCheckVMStateBootErr)

			mockOscVolumeInterface.
				EXPECT().
				CheckVolumeState(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(volumeStateAvailable), gomock.Eq(volumeId)).
				Return(vtc.expCheckVolumeStateAvailableErr)

			mockOscVolumeInterface.
				EXPECT().
				LinkVolume(gomock.Eq(volumeId), gomock.Eq(vmId), gomock.Eq(deviceName)).
				Return(vtc.expLinkVolumeErr)
			if vtc.expCheckVolumeStateUseFound {
				mockOscVolumeInterface.
					EXPECT().
					CheckVolumeState(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(volumeStateUse), gomock.Eq(volumeId)).
					Return(vtc.expCheckVolumeStateUseErr)
			}
			if vtc.expCheckVMStateVolumeFound {
				mockOscVMInterface.
					EXPECT().
					CheckVMState(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(vmState), gomock.Eq(vmId)).
					Return(vtc.expCheckVMStateVolumeErr)
			}

			if vtc.expLinkPublicIPFound {
				mockOscPublicIPInterface.
					EXPECT().
					LinkPublicIP(gomock.Eq(publicIpId), gomock.Eq(vmId)).
					Return("", vtc.expLinkPublicIPErr)
			}
			if vtc.expCheckVMStatePublicIPFound {
				mockOscVMInterface.
					EXPECT().
					CheckVMState(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(vmState), gomock.Eq(vmId)).
					Return(vtc.expCheckVMStatePublicIPErr)
			}

			reconcileVm, err := reconcileVm(ctx, clusterScope, machineScope, mockOscVMInterface, mockOscVolumeInterface, mockOscPublicIPInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface)
			if err != nil {
				assert.Equal(t, vtc.expReconcileVmErr.Error(), err.Error(), "reconcileVm() should return the same error")
			} else {
				assert.Nil(t, vtc.expReconcileVmErr)
			}
			t.Logf("find reconcileVm %v\n", reconcileVm)
		})
	}
}

// TestReconcileVmSecurityGroup has several tests to cover the code of the function reconcileVm
func TestReconcileVmSecurityGroup(t *testing.T) {
	vmTestCases := []struct {
		name                                 string
		clusterSpec                          infrastructurev1beta1.OscClusterSpec
		machineSpec                          infrastructurev1beta1.OscMachineSpec
		expCreateVMFound                     bool
		expLinkPublicIPFound                 bool
		expCreateSecurityGroupRuleFound      bool
		expCreateVMErr                       error
		expReconcileVmErr                    error
		expCheckVMStateBootErr               error
		expCheckVolumeStateAvailableErr      error
		expLinkVolumeErr                     error
		expCheckVolumeStateUseErr            error
		expCheckVMStateVolumeErr             error
		expCreateSecurityGroupRuleErr        error
		expLinkPublicIPErr                   error
		expCheckVMStatePublicIPErr           error
		expLinkLoadBalancerBackendMachineErr error
	}{
		{
			name:                                 "failed to link LoadBalancerBackendMachine ",
			clusterSpec:                          defaultVmClusterInitialize,
			machineSpec:                          defaultVmInitialize,
			expCreateVMFound:                     true,
			expLinkPublicIPFound:                 true,
			expCreateSecurityGroupRuleFound:      false,
			expCreateVMErr:                       nil,
			expCheckVMStateBootErr:               nil,
			expCheckVolumeStateAvailableErr:      nil,
			expLinkVolumeErr:                     nil,
			expCheckVolumeStateUseErr:            nil,
			expCheckVMStateVolumeErr:             nil,
			expLinkPublicIPErr:                   nil,
			expCheckVMStatePublicIPErr:           nil,
			expLinkLoadBalancerBackendMachineErr: fmt.Errorf("LinkLoadBalancerBackendMachine generic error"),
			expCreateSecurityGroupRuleErr:        nil,
			expReconcileVmErr:                    fmt.Errorf("LinkLoadBalancerBackendMachine generic error Can not link vm test-loadbalancer with loadBalancerName i-test-vm-uid for OscCluster test-system/test-osc"),
		},
		{
			name:                                 "failed to create SecurityGroupRule",
			clusterSpec:                          defaultVmClusterInitialize,
			machineSpec:                          defaultVmInitialize,
			expCreateVMFound:                     true,
			expLinkPublicIPFound:                 true,
			expCreateSecurityGroupRuleFound:      true,
			expCreateVMErr:                       nil,
			expCheckVMStateBootErr:               nil,
			expCheckVolumeStateAvailableErr:      nil,
			expLinkVolumeErr:                     nil,
			expCheckVolumeStateUseErr:            nil,
			expCheckVMStateVolumeErr:             nil,
			expLinkPublicIPErr:                   nil,
			expCheckVMStatePublicIPErr:           nil,
			expLinkLoadBalancerBackendMachineErr: nil,
			expCreateSecurityGroupRuleErr:        fmt.Errorf("CreateSecurityGroupRule generic error"),
			expReconcileVmErr:                    fmt.Errorf("CreateSecurityGroupRule generic error Can not create outbound securityGroupRule for OscCluster test-system/test-osc"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope, ctx, mockOscVMInterface, mockOscVolumeInterface, mockOscPublicIPInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface := SetupWithVmMock(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			vmName := vtc.machineSpec.Node.VM.Name + "-uid"
			vmId := "i-" + vmName
			vmState := "running"

			volumeName := vtc.machineSpec.Node.VM.VolumeName + "-uid"
			volumeId := "vol-" + volumeName
			volumeRef := machineScope.GetVolumeRef()
			volumeRef.ResourceMap = make(map[string]string)
			volumeRef.ResourceMap[volumeName] = volumeId
			volumeStateAvailable := "available"
			volumeStateUse := "in-use"

			subnetName := vtc.machineSpec.Node.VM.SubnetName + "-uid"
			subnetId := "subnet-" + subnetName
			subnetRef := clusterScope.GetSubnetRef()
			subnetRef.ResourceMap = make(map[string]string)
			subnetRef.ResourceMap[subnetName] = subnetId

			publicIpName := vtc.machineSpec.Node.VM.PublicIPName + "-uid"
			publicIpId := "eipalloc-" + publicIpName
			publicIpRef := clusterScope.GetPublicIPRef()
			publicIpRef.ResourceMap = make(map[string]string)
			publicIpRef.ResourceMap[publicIpName] = publicIpId

			linkPublicIPID := "eipassoc-" + publicIpName
			linkPublicIPRef := machineScope.GetLinkPublicIPRef()
			linkPublicIPRef.ResourceMap = make(map[string]string)
			linkPublicIPRef.ResourceMap[vmName] = linkPublicIPID

			var privateIPS []string
			VMPrivateIPS := machineScope.GetVMPrivateIPS()
			for _, VMPrivateIP := range *VMPrivateIPS {
				privateIP := VMPrivateIP.PrivateIP
				privateIPS = append(privateIPS, privateIP)
			}

			var securityGroupIds []string
			vmSecurityGroups := machineScope.GetVMSecurityGroups()
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			for _, vmSecurityGroup := range *vmSecurityGroups {
				securityGroupName := vmSecurityGroup.Name + "-uid"
				securityGroupId := "sg-" + securityGroupName
				securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
				securityGroupIds = append(securityGroupIds, securityGroupId)
			}

			deviceName := vtc.machineSpec.Node.VM.DeviceName
			vmSpec := vtc.machineSpec.Node.VM
			var clockInsideLoop time.Duration = 5
			var clockLoop time.Duration = 60
			var firstClockLoop time.Duration = 120
			loadBalancerName := vtc.machineSpec.Node.VM.LoadBalancerName
			loadBalancerSpec := clusterScope.GetLoadBalancer()
			loadBalancerSpec.SetDefaultValue()
			loadBalancerSecurityGroupName := loadBalancerSpec.SecurityGroupName
			ipProtocol := strings.ToLower(loadBalancerSpec.Listener.BackendProtocol)
			fromPortRange := loadBalancerSpec.Listener.BackendPort
			toPortRange := loadBalancerSpec.Listener.BackendPort
			flow := "Outbound"
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
			linkPublicIP := osc.LinkPublicIpResponse{
				LinkPublicIpId: &linkPublicIPID,
			}
			vm := &createVm[0]
			if vtc.expCreateVMFound {
				mockOscVMInterface.
					EXPECT().
					CreateVM(gomock.Eq(machineScope), gomock.Eq(&vmSpec), gomock.Eq(subnetId), gomock.Eq(securityGroupIds), gomock.Eq(privateIPS), gomock.Eq(vmName)).
					Return(vm, vtc.expCreateVMErr)
			} else {
				mockOscVMInterface.
					EXPECT().
					CreateVM(gomock.Eq(machineScope), gomock.Eq(vmSpec), gomock.Eq(subnetId), gomock.Eq(securityGroupIds), gomock.Eq(privateIPS), gomock.Eq(vmName)).
					Return(nil, vtc.expCreateVMErr)
			}

			mockOscVMInterface.
				EXPECT().
				CheckVMState(gomock.Eq(clockInsideLoop), gomock.Eq(firstClockLoop), gomock.Eq(vmState), gomock.Eq(vmId)).
				Return(vtc.expCheckVMStateBootErr)

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

			mockOscVMInterface.
				EXPECT().
				CheckVMState(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(vmState), gomock.Eq(vmId)).
				Return(vtc.expCheckVMStateVolumeErr)

			if vtc.expLinkPublicIPFound {
				mockOscPublicIPInterface.
					EXPECT().
					LinkPublicIP(gomock.Eq(publicIpId), gomock.Eq(vmId)).
					Return(*linkPublicIP.LinkPublicIpId, vtc.expLinkPublicIPErr)
			} else {
				mockOscPublicIPInterface.
					EXPECT().
					LinkPublicIP(gomock.Eq(publicIpId), gomock.Eq(vmId)).
					Return("", vtc.expLinkPublicIPErr)
			}

			mockOscVMInterface.
				EXPECT().
				CheckVMState(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(vmState), gomock.Eq(vmId)).
				Return(vtc.expCheckVMStatePublicIPErr)

			vmIds := []string{vmId}
			mockOscLoadBalancerInterface.
				EXPECT().
				LinkLoadBalancerBackendMachines(gomock.Eq(vmIds), gomock.Eq(loadBalancerName)).
				Return(vtc.expLinkLoadBalancerBackendMachineErr)

			if vtc.expCreateSecurityGroupRuleFound {
				mockOscSecurityGroupInterface.
					EXPECT().
					CreateSecurityGroupRule(gomock.Eq(associateSecurityGroupId), gomock.Eq(flow), gomock.Eq(ipProtocol), "", gomock.Eq(securityGroupIds[0]), gomock.Eq(fromPortRange), gomock.Eq(toPortRange)).
					Return(nil, vtc.expCreateSecurityGroupRuleErr)
			}

			reconcileVm, err := reconcileVm(ctx, clusterScope, machineScope, mockOscVMInterface, mockOscVolumeInterface, mockOscPublicIPInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface)
			if err != nil {
				assert.Equal(t, vtc.expReconcileVmErr.Error(), err.Error(), "reconcileVm() should return the same error")
			} else {
				assert.Nil(t, vtc.expReconcileVmErr)
			}
			t.Logf("find reconcileVm %v\n", reconcileVm)
		})
	}
}

// TestReconcileVmGet has several tests to cover the code of the function reconcileVm
func TestReconcileVmGet(t *testing.T) {
	vmTestCases := []struct {
		name               string
		clusterSpec        infrastructurev1beta1.OscClusterSpec
		machineSpec        infrastructurev1beta1.OscMachineSpec
		expGetVmFound      bool
		expGetVMStateFound bool
		expGetVMStateErr   error
		expGetVmErr        error
		expReconcileVmErr  error
	}{
		{
			name:               "get vm",
			clusterSpec:        defaultVmClusterInitialize,
			machineSpec:        defaultVmReconcile,
			expGetVmFound:      true,
			expGetVMStateFound: true,
			expGetVmErr:        nil,
			expGetVMStateErr:   nil,
			expReconcileVmErr:  nil,
		},
		{
			name:               "failed to get vm",
			clusterSpec:        defaultVmClusterInitialize,
			machineSpec:        defaultVmReconcile,
			expGetVmFound:      true,
			expGetVMStateFound: false,
			expGetVmErr:        fmt.Errorf("GetVm generic error"),
			expGetVMStateErr:   nil,
			expReconcileVmErr:  fmt.Errorf("GetVm generic error"),
		},
		{
			name:               "failed to set vmstate",
			clusterSpec:        defaultVmClusterInitialize,
			machineSpec:        defaultVmReconcile,
			expGetVmFound:      true,
			expGetVMStateFound: true,
			expGetVmErr:        nil,
			expGetVMStateErr:   fmt.Errorf("GetVmState generic error"),
			expReconcileVmErr:  fmt.Errorf("GetVmState generic error Can not get vm i-test-vm-uid state for OscMachine test-system/test-osc"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope, ctx, mockOscVMInterface, mockOscVolumeInterface, mockOscPublicIPInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface := SetupWithVmMock(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			vmName := vtc.machineSpec.Node.VM.Name + "-uid"
			vmId := "i-" + vmName
			vmState := "running"

			volumeName := vtc.machineSpec.Node.VM.VolumeName + "-uid"
			volumeId := "vol-" + volumeName
			volumeRef := machineScope.GetVolumeRef()
			volumeRef.ResourceMap = make(map[string]string)
			volumeRef.ResourceMap[volumeName] = volumeId

			subnetName := vtc.machineSpec.Node.VM.SubnetName + "-uid"
			subnetId := "subnet-" + subnetName
			subnetRef := clusterScope.GetSubnetRef()
			subnetRef.ResourceMap = make(map[string]string)
			subnetRef.ResourceMap[subnetName] = subnetId

			publicIpName := vtc.machineSpec.Node.VM.PublicIPName + "-uid"
			publicIpId := "eipalloc-" + publicIpName
			publicIpRef := clusterScope.GetPublicIPRef()
			publicIpRef.ResourceMap = make(map[string]string)
			publicIpRef.ResourceMap[publicIpName] = publicIpId

			linkPublicIPID := "eipassoc-" + publicIpName
			linkPublicIPRef := machineScope.GetLinkPublicIPRef()
			linkPublicIPRef.ResourceMap = make(map[string]string)
			linkPublicIPRef.ResourceMap[vmName] = linkPublicIPID

			vmSecurityGroups := machineScope.GetVMSecurityGroups()
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			for _, vmSecurityGroup := range *vmSecurityGroups {
				securityGroupName := vmSecurityGroup.Name + "-uid"
				securityGroupId := "sg-" + securityGroupName
				securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
			}

			readVms := osc.ReadVmsResponse{
				Vms: &[]osc.Vm{
					{
						VmId: &vmId,
					},
				},
			}

			readVm := *readVms.Vms
			vm := &readVm[0]
			if vtc.expGetVmFound {
				mockOscVMInterface.
					EXPECT().
					GetVM(gomock.Eq(vmId)).
					Return(vm, vtc.expGetVmErr)
			} else {
				mockOscVMInterface.
					EXPECT().
					GetVM(gomock.Eq(vmId)).
					Return(nil, vtc.expGetVmErr)
			}

			if vtc.expGetVMStateFound {
				mockOscVMInterface.
					EXPECT().
					GetVMState(gomock.Eq(vmId)).
					Return(vmState, vtc.expGetVMStateErr)
			}

			reconcileVm, err := reconcileVm(ctx, clusterScope, machineScope, mockOscVMInterface, mockOscVolumeInterface, mockOscPublicIPInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface)
			if err != nil {
				assert.Equal(t, vtc.expReconcileVmErr.Error(), err.Error(), "reconcileVm() should return the same error")
			} else {
				assert.Nil(t, vtc.expReconcileVmErr)
			}
			t.Logf("find reconcileVm %v\n", reconcileVm)

		})
	}
}

// TestReconcileVmResourceID has several tests to cover the code of the function reconcileVm
func TestReconcileVmResourceID(t *testing.T) {
	vmTestCases := []struct {
		name                              string
		clusterSpec                       infrastructurev1beta1.OscClusterSpec
		machineSpec                       infrastructurev1beta1.OscMachineSpec
		expVolumeFound                    bool
		expSubnetFound                    bool
		expPublicIPFound                  bool
		expLinkPublicIPFound              bool
		expSecurityGroupFound             bool
		expLoadBalancerSecurityGroupFound bool
		expReconcileVmErr                 error
	}{
		{
			name:                              "Volume does not exist ",
			clusterSpec:                       defaultVmClusterInitialize,
			machineSpec:                       defaultVmInitialize,
			expVolumeFound:                    false,
			expSubnetFound:                    true,
			expPublicIPFound:                  true,
			expLinkPublicIPFound:              true,
			expSecurityGroupFound:             true,
			expLoadBalancerSecurityGroupFound: true,
			expReconcileVmErr:                 fmt.Errorf("test-volume-uid does not exist"),
		},
		{
			name:                              "Volume does not exist ",
			clusterSpec:                       defaultVmClusterInitialize,
			machineSpec:                       defaultVmInitialize,
			expVolumeFound:                    true,
			expSubnetFound:                    false,
			expPublicIPFound:                  true,
			expLinkPublicIPFound:              true,
			expSecurityGroupFound:             true,
			expLoadBalancerSecurityGroupFound: true,
			expReconcileVmErr:                 fmt.Errorf("test-subnet-uid does not exist"),
		},
		{
			name:                              "PublicIP does not exist ",
			clusterSpec:                       defaultVmClusterInitialize,
			machineSpec:                       defaultVmInitialize,
			expVolumeFound:                    true,
			expSubnetFound:                    true,
			expPublicIPFound:                  false,
			expLinkPublicIPFound:              true,
			expSecurityGroupFound:             true,
			expLoadBalancerSecurityGroupFound: true,
			expReconcileVmErr:                 fmt.Errorf("test-publicip-uid does not exist"),
		},
		{
			name:                              "SecurityGroup does not exist ",
			clusterSpec:                       defaultVmClusterInitialize,
			machineSpec:                       defaultVmInitialize,
			expVolumeFound:                    true,
			expSubnetFound:                    true,
			expPublicIPFound:                  true,
			expLinkPublicIPFound:              true,
			expSecurityGroupFound:             false,
			expLoadBalancerSecurityGroupFound: false,
			expReconcileVmErr:                 fmt.Errorf("test-securitygroup-uid does not exist"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope, ctx, mockOscVMInterface, mockOscVolumeInterface, mockOscPublicIPInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface := SetupWithVmMock(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			vmName := vtc.machineSpec.Node.VM.Name + "-uid"

			volumeName := vtc.machineSpec.Node.VM.VolumeName + "-uid"
			volumeId := "vol-" + volumeName
			volumeRef := machineScope.GetVolumeRef()
			volumeRef.ResourceMap = make(map[string]string)
			if vtc.expVolumeFound {
				volumeRef.ResourceMap[volumeName] = volumeId
			}

			subnetName := vtc.machineSpec.Node.VM.SubnetName + "-uid"
			subnetId := "subnet-" + subnetName
			subnetRef := clusterScope.GetSubnetRef()
			subnetRef.ResourceMap = make(map[string]string)
			if vtc.expSubnetFound {
				subnetRef.ResourceMap[subnetName] = subnetId
			}

			publicIpName := vtc.machineSpec.Node.VM.PublicIPName + "-uid"
			publicIpId := "eipalloc-" + publicIpName
			publicIpRef := clusterScope.GetPublicIPRef()
			publicIpRef.ResourceMap = make(map[string]string)
			if vtc.expPublicIPFound {
				publicIpRef.ResourceMap[publicIpName] = publicIpId
			}

			linkPublicIPID := "eipassoc-" + publicIpName
			linkPublicIPRef := machineScope.GetLinkPublicIPRef()
			linkPublicIPRef.ResourceMap = make(map[string]string)
			if vtc.expLinkPublicIPFound {
				linkPublicIPRef.ResourceMap[vmName] = linkPublicIPID
			}

			vmSecurityGroups := machineScope.GetVMSecurityGroups()
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

			reconcileVm, err := reconcileVm(ctx, clusterScope, machineScope, mockOscVMInterface, mockOscVolumeInterface, mockOscPublicIPInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface)
			if err != nil {
				assert.Equal(t, vtc.expReconcileVmErr, err, "reconcileVm() should return the same error")
			} else {
				assert.Nil(t, vtc.expReconcileVmErr)
			}
			t.Logf("find reconcileVm %v\n", reconcileVm)
		})
	}
}

// TestReconcileDeleteVM has several tests to cover the code of the function reconcileDeleteVm
func TestReconcileDeleteVM(t *testing.T) {
	vmTestCases := []struct {
		name                                    string
		clusterSpec                             infrastructurev1beta1.OscClusterSpec
		machineSpec                             infrastructurev1beta1.OscMachineSpec
		expDeleteInboundSecurityGroupRuleFound  bool
		expDeleteOutboundSecurityGroupRuleFound bool
		expCheckVMStateBootErr                  error
		expUnlinkLoadBalancerBackendMachineErr  error
		expCheckVMStateLoadBalancerErr          error
		expDeleteInboundSecurityGroupRuleErr    error
		expDeleteOutboundSecurityGroupRuleErr   error
		expDeleteVMErr                          error
		expGetVmErr                             error
		expSecurityGroupRuleFound               bool
		expGetVmFound                           bool
		expCheckUnlinkPublicIPErr               error
		expReconcileDeleteVMErr                 error
	}{
		{
			name:                                    "delete vm",
			clusterSpec:                             defaultVmClusterReconcile,
			machineSpec:                             defaultVmReconcile,
			expCheckVMStateBootErr:                  nil,
			expDeleteInboundSecurityGroupRuleFound:  true,
			expDeleteOutboundSecurityGroupRuleFound: true,
			expUnlinkLoadBalancerBackendMachineErr:  nil,
			expCheckVMStateLoadBalancerErr:          nil,
			expDeleteInboundSecurityGroupRuleErr:    nil,
			expDeleteOutboundSecurityGroupRuleErr:   nil,
			expSecurityGroupRuleFound:               true,
			expDeleteVMErr:                          nil,
			expGetVmFound:                           true,
			expGetVmErr:                             nil,
			expCheckUnlinkPublicIPErr:               nil,
			expReconcileDeleteVMErr:                 nil,
		},
		{
			name:                                    "failed to delete vm",
			clusterSpec:                             defaultVmClusterReconcile,
			machineSpec:                             defaultVmReconcile,
			expCheckVMStateBootErr:                  nil,
			expDeleteInboundSecurityGroupRuleFound:  true,
			expDeleteOutboundSecurityGroupRuleFound: true,
			expUnlinkLoadBalancerBackendMachineErr:  nil,
			expCheckVMStateLoadBalancerErr:          nil,
			expSecurityGroupRuleFound:               true,
			expDeleteVMErr:                          fmt.Errorf("DeleteVm generic error"),
			expGetVmFound:                           true,
			expGetVmErr:                             nil,
			expDeleteInboundSecurityGroupRuleErr:    nil,
			expDeleteOutboundSecurityGroupRuleErr:   nil,
			expCheckUnlinkPublicIPErr:               nil,
			expReconcileDeleteVMErr:                 fmt.Errorf("DeleteVm generic error Can not delete vm for OscMachine test-system/test-osc"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope, ctx, mockOscVMInterface, _, mockOscPublicIPInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface := SetupWithVmMock(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			vmName := vtc.machineSpec.Node.VM.Name + "-uid"
			vmId := "i-" + vmName
			vmRef := machineScope.GetVMRef()
			vmRef.ResourceMap = make(map[string]string)
			if vtc.expGetVmFound {
				vmRef.ResourceMap[vmName] = vmId
			}
			vmState := "running"

			var securityGroupIds []string
			vmSecurityGroups := machineScope.GetVMSecurityGroups()
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			for _, vmSecurityGroup := range *vmSecurityGroups {
				securityGroupName := vmSecurityGroup.Name + "-uid"
				securityGroupId := "sg-" + securityGroupName
				securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
				securityGroupIds = append(securityGroupIds, securityGroupId)
			}
			publicIpName := vtc.machineSpec.Node.VM.PublicIPName + "-uid"
			linkPublicIPID := "eipassoc-" + publicIpName
			linkPublicIPRef := machineScope.GetLinkPublicIPRef()
			linkPublicIPRef.ResourceMap = make(map[string]string)
			linkPublicIPRef.ResourceMap[publicIpName] = linkPublicIPID

			var clockInsideLoop time.Duration = 5
			var clockLoop time.Duration = 60
			var firstClockLoop time.Duration = 120
			loadBalancerName := vtc.machineSpec.Node.VM.LoadBalancerName
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
			vm := &createVm[0]
			if vtc.expGetVmFound {
				mockOscVMInterface.
					EXPECT().
					GetVM(gomock.Eq(vmId)).
					Return(vm, vtc.expGetVmErr)
			} else {
				mockOscVMInterface.
					EXPECT().
					GetVM(gomock.Eq(vmId)).
					Return(nil, vtc.expGetVmErr)
			}
			mockOscVMInterface.
				EXPECT().
				CheckVMState(gomock.Eq(clockInsideLoop), gomock.Eq(firstClockLoop), gomock.Eq(vmState), gomock.Eq(vmId)).
				Return(vtc.expCheckVMStateBootErr)

			mockOscVMInterface.
				EXPECT().
				CheckVMState(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(vmState), gomock.Eq(vmId)).
				Return(vtc.expCheckVMStateLoadBalancerErr)

			mockOscPublicIPInterface.
				EXPECT().
				UnlinkPublicIP(gomock.Eq(linkPublicIPID)).
				Return(vtc.expCheckUnlinkPublicIPErr)
			vmIds := []string{vmId}
			mockOscLoadBalancerInterface.
				EXPECT().
				UnlinkLoadBalancerBackendMachines(gomock.Eq(vmIds), gomock.Eq(loadBalancerName)).
				Return(vtc.expUnlinkLoadBalancerBackendMachineErr)

			if vtc.expDeleteOutboundSecurityGroupRuleFound {
				mockOscSecurityGroupInterface.
					EXPECT().
					DeleteSecurityGroupRule(gomock.Eq(associateSecurityGroupId), gomock.Eq("Outbound"), gomock.Eq(ipProtocol), "", gomock.Eq(securityGroupIds[0]), gomock.Eq(fromPortRange), gomock.Eq(toPortRange)).
					Return(vtc.expDeleteOutboundSecurityGroupRuleErr)
			}

			if vtc.expDeleteInboundSecurityGroupRuleFound {
				mockOscSecurityGroupInterface.
					EXPECT().
					DeleteSecurityGroupRule(gomock.Eq(associateSecurityGroupId), gomock.Eq("Inbound"), gomock.Eq(ipProtocol), "", gomock.Eq(securityGroupIds[0]), gomock.Eq(fromPortRange), gomock.Eq(toPortRange)).
					Return(vtc.expDeleteInboundSecurityGroupRuleErr)

			}

			mockOscVMInterface.
				EXPECT().
				DeleteVM(gomock.Eq(vmId)).
				Return(vtc.expDeleteVMErr)
			reconcileDeleteVm, err := reconcileDeleteVm(ctx, clusterScope, machineScope, mockOscVMInterface, mockOscPublicIPInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface)
			if err != nil {
				assert.Equal(t, vtc.expReconcileDeleteVMErr.Error(), err.Error(), "reconccileDeleteVm() hould return the same error")
			} else {
				assert.Nil(t, vtc.expReconcileDeleteVMErr)
			}
			t.Logf("find reconcileDeleteVm %v\n", reconcileDeleteVm)
		})
	}

}

// TestReconcileDeleteVMUnlinkPublicIP has several tests to cover the code of the function reconcileDeleteVm
func TestReconcileDeleteVMUnlinkPublicIP(t *testing.T) {
	vmTestCases := []struct {
		name                        string
		clusterSpec                 infrastructurev1beta1.OscClusterSpec
		machineSpec                 infrastructurev1beta1.OscMachineSpec
		expCheckVMStateBootErr      error
		expCheckVMStateBootFound    bool
		expCheckUnlinkPublicIPFound bool
		expGetVmFound               bool
		expGetVmErr                 error

		expCheckUnlinkPublicIPErr error
		expReconcileDeleteVMErr   error
	}{
		{
			name:                        "failed vmState",
			clusterSpec:                 defaultVmClusterReconcile,
			machineSpec:                 defaultVmReconcile,
			expCheckVMStateBootErr:      fmt.Errorf("CheckVmState generic error"),
			expCheckVMStateBootFound:    true,
			expGetVmFound:               true,
			expGetVmErr:                 nil,
			expCheckUnlinkPublicIPFound: false,
			expCheckUnlinkPublicIPErr:   nil,
			expReconcileDeleteVMErr:     fmt.Errorf("CheckVmState generic error Can not get vm i-test-vm-uid running for OscMachine test-system/test-osc"),
		},
		{
			name:                        "failed unlink volume",
			clusterSpec:                 defaultVmClusterReconcile,
			machineSpec:                 defaultVmReconcile,
			expCheckVMStateBootErr:      nil,
			expCheckVMStateBootFound:    true,
			expGetVmFound:               true,
			expGetVmErr:                 nil,
			expCheckUnlinkPublicIPFound: true,
			expCheckUnlinkPublicIPErr:   fmt.Errorf("CheckUnlinkPublicIp generic error"),
			expReconcileDeleteVMErr:     fmt.Errorf("CheckUnlinkPublicIp generic error Can not unlink publicIp for OscCluster test-system/test-osc"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope, ctx, mockOscVMInterface, _, mockOscPublicIPInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface := SetupWithVmMock(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			vmName := vtc.machineSpec.Node.VM.Name + "-uid"
			vmId := "i-" + vmName
			vmRef := machineScope.GetVMRef()
			vmRef.ResourceMap = make(map[string]string)
			if vtc.expGetVmFound {
				vmRef.ResourceMap[vmName] = vmId
			}
			vmState := "running"

			vmSecurityGroups := machineScope.GetVMSecurityGroups()
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			for _, vmSecurityGroup := range *vmSecurityGroups {
				securityGroupName := vmSecurityGroup.Name + "-uid"
				securityGroupId := "sg-" + securityGroupName
				securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
			}
			publicIpName := vtc.machineSpec.Node.VM.PublicIPName + "-uid"
			linkPublicIPID := "eipassoc-" + publicIpName
			linkPublicIPRef := machineScope.GetLinkPublicIPRef()
			linkPublicIPRef.ResourceMap = make(map[string]string)
			linkPublicIPRef.ResourceMap[publicIpName] = linkPublicIPID

			var clockInsideLoop time.Duration = 5
			var firstClockLoop time.Duration = 120

			createVms := osc.CreateVmsResponse{
				Vms: &[]osc.Vm{
					{
						VmId: &vmId,
					},
				},
			}

			createVm := *createVms.Vms
			vm := &createVm[0]
			if vtc.expGetVmFound {
				mockOscVMInterface.
					EXPECT().
					GetVM(gomock.Eq(vmId)).
					Return(vm, vtc.expGetVmErr)
			} else {
				mockOscVMInterface.
					EXPECT().
					GetVM(gomock.Eq(vmId)).
					Return(nil, vtc.expGetVmErr)
			}
			if vtc.expCheckVMStateBootFound {
				mockOscVMInterface.
					EXPECT().
					CheckVMState(gomock.Eq(clockInsideLoop), gomock.Eq(firstClockLoop), gomock.Eq(vmState), gomock.Eq(vmId)).
					Return(vtc.expCheckVMStateBootErr)
			}

			if vtc.expCheckUnlinkPublicIPFound {
				mockOscPublicIPInterface.
					EXPECT().
					UnlinkPublicIP(gomock.Eq(linkPublicIPID)).
					Return(vtc.expCheckUnlinkPublicIPErr)
			}

			reconcileDeleteVm, err := reconcileDeleteVm(ctx, clusterScope, machineScope, mockOscVMInterface, mockOscPublicIPInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface)
			if err != nil {
				assert.Equal(t, vtc.expReconcileDeleteVMErr.Error(), err.Error(), "reconcileDeleteVm() should return the same error")
			} else {
				assert.Nil(t, vtc.expReconcileDeleteVMErr)
			}
			t.Logf("find reconcileDeleteVm %v\n", reconcileDeleteVm)
		})
	}

}

// TestReconcileDeleteVMLoadBalancer has several tests to cover the code of the function reconcileDeleteVm
func TestReconcileDeleteVMLoadBalancer(t *testing.T) {
	vmTestCases := []struct {
		name                                     string
		clusterSpec                              infrastructurev1beta1.OscClusterSpec
		machineSpec                              infrastructurev1beta1.OscMachineSpec
		expCheckVMStateBootErr                   error
		expUnlinkLoadBalancerBackendMachineErr   error
		expCheckVMStateLoadBalancerErr           error
		expDeleteSecurityGroupRuleErr            error
		expCheckVMStateLoadBalancerFound         bool
		expUnlinkLoadBalancerBackendMachineFound bool
		expDeleteSecurityGroupRuleFound          bool
		expGetVmErr                              error
		expGetVmFound                            bool
		expCheckUnlinkPublicIPErr                error
		expReconcileDeleteVMErr                  error
	}{
		{
			name:                                     "failed checkVMState",
			clusterSpec:                              defaultVmClusterReconcile,
			machineSpec:                              defaultVmReconcile,
			expCheckVMStateBootErr:                   nil,
			expUnlinkLoadBalancerBackendMachineErr:   nil,
			expCheckVMStateLoadBalancerErr:           fmt.Errorf("CheckVmStateLoadBalancer generic error"),
			expDeleteSecurityGroupRuleErr:            nil,
			expDeleteSecurityGroupRuleFound:          false,
			expCheckVMStateLoadBalancerFound:         true,
			expUnlinkLoadBalancerBackendMachineFound: false,
			expGetVmFound:                            true,
			expGetVmErr:                              nil,
			expCheckUnlinkPublicIPErr:                nil,
			expReconcileDeleteVMErr:                  fmt.Errorf("CheckVmStateLoadBalancer generic error Can not get vm i-test-vm-uid running for OscMachine test-system/test-osc"),
		},
		{
			name:                                     "failed UnlinkLoadBalancerBackendMachineFound",
			clusterSpec:                              defaultVmClusterReconcile,
			machineSpec:                              defaultVmReconcile,
			expCheckVMStateBootErr:                   nil,
			expUnlinkLoadBalancerBackendMachineErr:   fmt.Errorf("UnlinkLoadBalancerBackendMachineFound generic error"),
			expCheckVMStateLoadBalancerErr:           nil,
			expDeleteSecurityGroupRuleErr:            nil,
			expDeleteSecurityGroupRuleFound:          false,
			expCheckVMStateLoadBalancerFound:         true,
			expUnlinkLoadBalancerBackendMachineFound: true,
			expGetVmFound:                            true,
			expGetVmErr:                              nil,
			expCheckUnlinkPublicIPErr:                nil,
			expReconcileDeleteVMErr:                  fmt.Errorf("UnlinkLoadBalancerBackendMachineFound generic error Can not unlink vm test-loadbalancer with loadBalancerName i-test-vm-uid for OscCluster test-system/test-osc"),
		},
		{
			name:                                     "failed DeleteSecurityGroupRule",
			clusterSpec:                              defaultVmClusterReconcile,
			machineSpec:                              defaultVmReconcile,
			expCheckVMStateBootErr:                   nil,
			expUnlinkLoadBalancerBackendMachineErr:   nil,
			expCheckVMStateLoadBalancerErr:           nil,
			expDeleteSecurityGroupRuleErr:            fmt.Errorf("DeleteSecurityGroupRule generic error"),
			expDeleteSecurityGroupRuleFound:          true,
			expCheckVMStateLoadBalancerFound:         true,
			expUnlinkLoadBalancerBackendMachineFound: true,
			expGetVmFound:                            true,
			expGetVmErr:                              nil,
			expCheckUnlinkPublicIPErr:                nil,
			expReconcileDeleteVMErr:                  fmt.Errorf("DeleteSecurityGroupRule generic error Can not delete outbound securityGroupRule for OscCluster test-system/test-osc"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope, ctx, mockOscVMInterface, _, mockOscPublicIPInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface := SetupWithVmMock(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			vmName := vtc.machineSpec.Node.VM.Name + "-uid"
			vmId := "i-" + vmName
			vmRef := machineScope.GetVMRef()
			vmRef.ResourceMap = make(map[string]string)
			if vtc.expGetVmFound {
				vmRef.ResourceMap[vmName] = vmId
			}
			vmState := "running"

			var securityGroupIds []string
			vmSecurityGroups := machineScope.GetVMSecurityGroups()
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			for _, vmSecurityGroup := range *vmSecurityGroups {
				securityGroupName := vmSecurityGroup.Name + "-uid"
				securityGroupId := "sg-" + securityGroupName
				securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
				securityGroupIds = append(securityGroupIds, securityGroupId)
			}
			publicIpName := vtc.machineSpec.Node.VM.PublicIPName + "-uid"
			linkPublicIPID := "eipassoc-" + publicIpName
			linkPublicIPRef := machineScope.GetLinkPublicIPRef()
			linkPublicIPRef.ResourceMap = make(map[string]string)
			linkPublicIPRef.ResourceMap[publicIpName] = linkPublicIPID

			var clockInsideLoop time.Duration = 5
			var clockLoop time.Duration = 60
			var firstClockLoop time.Duration = 120
			loadBalancerName := vtc.machineSpec.Node.VM.LoadBalancerName
			loadBalancerSpec := clusterScope.GetLoadBalancer()
			loadBalancerSpec.SetDefaultValue()
			loadBalancerSecurityGroupName := loadBalancerSpec.SecurityGroupName
			ipProtocol := strings.ToLower(loadBalancerSpec.Listener.BackendProtocol)
			fromPortRange := loadBalancerSpec.Listener.BackendPort
			toPortRange := loadBalancerSpec.Listener.BackendPort
			flow := "Outbound"
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
			vm := &createVm[0]
			if vtc.expGetVmFound {
				mockOscVMInterface.
					EXPECT().
					GetVM(gomock.Eq(vmId)).
					Return(vm, vtc.expGetVmErr)
			} else {
				mockOscVMInterface.
					EXPECT().
					GetVM(gomock.Eq(vmId)).
					Return(nil, vtc.expGetVmErr)
			}
			mockOscVMInterface.
				EXPECT().
				CheckVMState(gomock.Eq(clockInsideLoop), gomock.Eq(firstClockLoop), gomock.Eq(vmState), gomock.Eq(vmId)).
				Return(vtc.expCheckVMStateBootErr)

			mockOscPublicIPInterface.
				EXPECT().
				UnlinkPublicIP(gomock.Eq(linkPublicIPID)).
				Return(vtc.expCheckUnlinkPublicIPErr)
			vmIds := []string{vmId}
			if vtc.expCheckVMStateLoadBalancerFound {
				mockOscVMInterface.
					EXPECT().
					CheckVMState(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(vmState), gomock.Eq(vmId)).
					Return(vtc.expCheckVMStateLoadBalancerErr)
			}
			if vtc.expUnlinkLoadBalancerBackendMachineFound {
				mockOscLoadBalancerInterface.
					EXPECT().
					UnlinkLoadBalancerBackendMachines(gomock.Eq(vmIds), gomock.Eq(loadBalancerName)).
					Return(vtc.expUnlinkLoadBalancerBackendMachineErr)
			}

			if vtc.expDeleteSecurityGroupRuleFound {
				mockOscSecurityGroupInterface.
					EXPECT().
					DeleteSecurityGroupRule(gomock.Eq(associateSecurityGroupId), gomock.Eq(flow), gomock.Eq(ipProtocol), "", gomock.Eq(securityGroupIds[0]), gomock.Eq(fromPortRange), gomock.Eq(toPortRange)).
					Return(vtc.expDeleteSecurityGroupRuleErr)
			}

			reconcileDeleteVm, err := reconcileDeleteVm(ctx, clusterScope, machineScope, mockOscVMInterface, mockOscPublicIPInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface)
			if err != nil {
				assert.Equal(t, vtc.expReconcileDeleteVMErr.Error(), err.Error(), "reconccileDeleteVm() hould return the same error")
			} else {
				assert.Nil(t, vtc.expReconcileDeleteVMErr)
			}
			t.Logf("find reconcileDeleteVm %v\n", reconcileDeleteVm)
		})
	}

}

// TestReconcileDeleteVMResourceID has several tests to cover the code of the function reconcileDeleteVm
func TestReconcileDeleteVMResourceID(t *testing.T) {
	vmTestCases := []struct {
		name                    string
		clusterSpec             infrastructurev1beta1.OscClusterSpec
		machineSpec             infrastructurev1beta1.OscMachineSpec
		expGetVmFound           bool
		expGetVmErr             error
		expSecurityGroupFound   bool
		expReconcileDeleteVMErr error
	}{
		{
			name:                    "failed to find vm",
			clusterSpec:             defaultVmClusterReconcile,
			machineSpec:             defaultVmReconcile,
			expGetVmFound:           false,
			expGetVmErr:             nil,
			expSecurityGroupFound:   true,
			expReconcileDeleteVMErr: nil,
		},
		{
			name:                    "failed to find security group",
			clusterSpec:             defaultVmClusterReconcile,
			machineSpec:             defaultVmReconcile,
			expGetVmFound:           true,
			expGetVmErr:             nil,
			expSecurityGroupFound:   false,
			expReconcileDeleteVMErr: fmt.Errorf("test-securitygroup-uid does not exist"),
		},
		{
			name:                    "failed to get vm",
			clusterSpec:             defaultVmClusterReconcile,
			machineSpec:             defaultVmReconcile,
			expGetVmFound:           true,
			expGetVmErr:             fmt.Errorf("GetVm generic error"),
			expSecurityGroupFound:   false,
			expReconcileDeleteVMErr: fmt.Errorf("GetVm generic error"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope, ctx, mockOscVMInterface, _, mockOscPublicIPInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface := SetupWithVmMock(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			vmName := vtc.machineSpec.Node.VM.Name + "-uid"
			vmId := "i-" + vmName
			vmRef := machineScope.GetVMRef()
			vmRef.ResourceMap = make(map[string]string)
			if vtc.expGetVmFound {
				vmRef.ResourceMap[vmName] = vmId
			}

			createVms := osc.CreateVmsResponse{
				Vms: &[]osc.Vm{
					{
						VmId: &vmId,
					},
				},
			}

			createVm := *createVms.Vms
			vm := &createVm[0]
			if vtc.expGetVmFound {
				mockOscVMInterface.
					EXPECT().
					GetVM(gomock.Eq(vmId)).
					Return(vm, vtc.expGetVmErr)
			} else {
				mockOscVMInterface.
					EXPECT().
					GetVM(gomock.Eq(vmId)).
					Return(nil, vtc.expGetVmErr)
			}

			vmSecurityGroups := machineScope.GetVMSecurityGroups()
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			for _, vmSecurityGroup := range *vmSecurityGroups {
				securityGroupName := vmSecurityGroup.Name + "-uid"
				securityGroupId := "sg-" + securityGroupName
				if vtc.expSecurityGroupFound {
					securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
				}
			}
			publicIpName := vtc.machineSpec.Node.VM.PublicIPName + "-uid"
			linkPublicIPID := "eipassoc-" + publicIpName
			linkPublicIPRef := machineScope.GetLinkPublicIPRef()
			linkPublicIPRef.ResourceMap = make(map[string]string)
			linkPublicIPRef.ResourceMap[publicIpName] = linkPublicIPID

			reconcileDeleteVm, err := reconcileDeleteVm(ctx, clusterScope, machineScope, mockOscVMInterface, mockOscPublicIPInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface)
			if err != nil {
				assert.Equal(t, vtc.expReconcileDeleteVMErr, err, "reconccileDeleteVm() hould return the same error")
			} else {
				assert.Nil(t, vtc.expReconcileDeleteVMErr)
			}
			t.Logf("find reconcileDeleteVm %v\n", reconcileDeleteVm)
		})
	}

}

// TestReconcileDeleteVMWithoutSpec has several tests to cover the code of the function reconcileDeleteVm
func TestReconcileDeleteVMWithoutSpec(t *testing.T) {
	vmTestCases := []struct {
		name                                   string
		clusterSpec                            infrastructurev1beta1.OscClusterSpec
		machineSpec                            infrastructurev1beta1.OscMachineSpec
		expCheckVMStateBootErr                 error
		expUnlinkLoadBalancerBackendMachineErr error
		expCheckVMStateLoadBalancerErr         error
		expDeleteSecurityGroupRuleErr          error
		expDeleteVMErr                         error
		expGetVmErr                            error
		expSecurityGroupRuleFound              bool
		expGetVmFound                          bool
		expCheckUnlinkPublicIPErr              error
		expReconcileDeleteVMErr                error
	}{
		{
			name: "delete vm without spec",
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Volumes: []*infrastructurev1beta1.OscVolume{
						{
							ResourceID: "vol-cluster-api-volume-uid",
						},
					},
					VM: infrastructurev1beta1.OscVM{
						ResourceID: "i-cluster-api-vm-uid",
					},
				},
			},
			expCheckVMStateBootErr:                 nil,
			expUnlinkLoadBalancerBackendMachineErr: nil,
			expCheckVMStateLoadBalancerErr:         nil,
			expDeleteSecurityGroupRuleErr:          nil,
			expSecurityGroupRuleFound:              true,
			expDeleteVMErr:                         nil,
			expGetVmFound:                          true,
			expGetVmErr:                            nil,
			expCheckUnlinkPublicIPErr:              nil,
			expReconcileDeleteVMErr:                nil,
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope, ctx, mockOscVMInterface, _, mockOscPublicIPInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface := SetupWithVmMock(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			vmName := "cluster-api-vm-uid"
			vmId := "i-" + vmName
			vmRef := machineScope.GetVMRef()
			vmRef.ResourceMap = make(map[string]string)
			if vtc.expGetVmFound {
				vmRef.ResourceMap[vmName] = vmId
			}

			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			securityGroupName := "cluster-api-securitygroup-kw-uid"
			securityGroupId := "sg-" + securityGroupName
			securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId

			publicIpName := "cluster-api-publicip-uid"
			linkPublicIPID := "eipassoc-" + publicIpName
			linkPublicIPRef := machineScope.GetLinkPublicIPRef()
			linkPublicIPRef.ResourceMap = make(map[string]string)
			linkPublicIPRef.ResourceMap[publicIpName] = linkPublicIPID

			createVms := osc.CreateVmsResponse{
				Vms: &[]osc.Vm{
					{
						VmId: &vmId,
					},
				},
			}

			createVm := *createVms.Vms
			vm := &createVm[0]
			if vtc.expGetVmFound {
				mockOscVMInterface.
					EXPECT().
					GetVM(gomock.Eq(vmId)).
					Return(vm, vtc.expGetVmErr)
			} else {
				mockOscVMInterface.
					EXPECT().
					GetVM(gomock.Eq(vmId)).
					Return(nil, vtc.expGetVmErr)
			}
			mockOscVMInterface.
				EXPECT().
				DeleteVM(gomock.Eq(vmId)).
				Return(vtc.expDeleteVMErr)
			reconcileDeleteVm, err := reconcileDeleteVm(ctx, clusterScope, machineScope, mockOscVMInterface, mockOscPublicIPInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface)
			if err != nil {
				assert.Equal(t, vtc.expReconcileDeleteVMErr, err, "reconcileDeleteVm() should return the same error")
			} else {
				assert.Nil(t, vtc.expReconcileDeleteVMErr)
			}

			t.Logf("find reconcileDeleteVm %v\n", reconcileDeleteVm)
		})
	}

}

// TestReconcileDeleteVMSecurityGroup has several tests to cover the code of the function reconcileDeleteVm
func TestReconcileDeleteVMSecurityGroup(t *testing.T) {
	vmTestCases := []struct {
		name                                    string
		clusterSpec                             infrastructurev1beta1.OscClusterSpec
		machineSpec                             infrastructurev1beta1.OscMachineSpec
		expDeleteInboundSecurityGroupRuleFound  bool
		expDeleteOutboundSecurityGroupRuleFound bool
		expCheckVMStateBootErr                  error
		expUnlinkLoadBalancerBackendMachineErr  error
		expCheckVMStateLoadBalancerErr          error
		expDeleteInboundSecurityGroupRuleErr    error
		expDeleteOutboundSecurityGroupRuleErr   error
		expDeleteVMErr                          error
		expGetVmErr                             error
		expSecurityGroupRuleFound               bool
		expGetVmFound                           bool
		expCheckUnlinkPublicIPErr               error
		expReconcileDeleteVMErr                 error
	}{
		{
			name:                                    "failed to delete inbound securitygroup",
			clusterSpec:                             defaultVmClusterReconcile,
			machineSpec:                             defaultVmReconcile,
			expCheckVMStateBootErr:                  nil,
			expDeleteInboundSecurityGroupRuleFound:  true,
			expDeleteOutboundSecurityGroupRuleFound: true,
			expUnlinkLoadBalancerBackendMachineErr:  nil,
			expCheckVMStateLoadBalancerErr:          nil,
			expDeleteInboundSecurityGroupRuleErr:    fmt.Errorf("CreateSecurityGroupRule generic error"),
			expDeleteOutboundSecurityGroupRuleErr:   nil,
			expSecurityGroupRuleFound:               true,
			expDeleteVMErr:                          nil,
			expGetVmFound:                           true,
			expGetVmErr:                             nil,
			expCheckUnlinkPublicIPErr:               nil,
			expReconcileDeleteVMErr:                 fmt.Errorf("CreateSecurityGroupRule generic error Can not delete inbound securityGroupRule for OscCluster test-system/test-osc"),
		},
		{
			name:                                    "failed to delete outbound securitygroup",
			clusterSpec:                             defaultVmClusterReconcile,
			machineSpec:                             defaultVmReconcile,
			expCheckVMStateBootErr:                  nil,
			expDeleteInboundSecurityGroupRuleFound:  false,
			expDeleteOutboundSecurityGroupRuleFound: true,
			expUnlinkLoadBalancerBackendMachineErr:  nil,
			expCheckVMStateLoadBalancerErr:          nil,
			expSecurityGroupRuleFound:               true,
			expDeleteVMErr:                          nil,
			expGetVmFound:                           true,
			expGetVmErr:                             nil,
			expDeleteInboundSecurityGroupRuleErr:    nil,
			expDeleteOutboundSecurityGroupRuleErr:   fmt.Errorf("CreateSecurityGroupRule generic error"),
			expCheckUnlinkPublicIPErr:               nil,
			expReconcileDeleteVMErr:                 fmt.Errorf("CreateSecurityGroupRule generic error Can not delete outbound securityGroupRule for OscCluster test-system/test-osc"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope, ctx, mockOscVMInterface, _, mockOscPublicIPInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface := SetupWithVmMock(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			vmName := vtc.machineSpec.Node.VM.Name + "-uid"
			vmId := "i-" + vmName
			vmRef := machineScope.GetVMRef()
			vmRef.ResourceMap = make(map[string]string)
			if vtc.expGetVmFound {
				vmRef.ResourceMap[vmName] = vmId
			}
			vmState := "running"

			var securityGroupIds []string
			vmSecurityGroups := machineScope.GetVMSecurityGroups()
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			for _, vmSecurityGroup := range *vmSecurityGroups {
				securityGroupName := vmSecurityGroup.Name + "-uid"
				securityGroupId := "sg-" + securityGroupName
				securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
				securityGroupIds = append(securityGroupIds, securityGroupId)
			}
			publicIpName := vtc.machineSpec.Node.VM.PublicIPName + "-uid"
			linkPublicIPID := "eipassoc-" + publicIpName
			linkPublicIPRef := machineScope.GetLinkPublicIPRef()
			linkPublicIPRef.ResourceMap = make(map[string]string)
			linkPublicIPRef.ResourceMap[publicIpName] = linkPublicIPID

			var clockInsideLoop time.Duration = 5
			var clockLoop time.Duration = 60
			var firstClockLoop time.Duration = 120
			loadBalancerName := vtc.machineSpec.Node.VM.LoadBalancerName
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
			vm := &createVm[0]
			if vtc.expGetVmFound {
				mockOscVMInterface.
					EXPECT().
					GetVM(gomock.Eq(vmId)).
					Return(vm, vtc.expGetVmErr)
			} else {
				mockOscVMInterface.
					EXPECT().
					GetVM(gomock.Eq(vmId)).
					Return(nil, vtc.expGetVmErr)
			}
			mockOscVMInterface.
				EXPECT().
				CheckVMState(gomock.Eq(clockInsideLoop), gomock.Eq(firstClockLoop), gomock.Eq(vmState), gomock.Eq(vmId)).
				Return(vtc.expCheckVMStateBootErr)

			mockOscVMInterface.
				EXPECT().
				CheckVMState(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(vmState), gomock.Eq(vmId)).
				Return(vtc.expCheckVMStateLoadBalancerErr)

			mockOscPublicIPInterface.
				EXPECT().
				UnlinkPublicIP(gomock.Eq(linkPublicIPID)).
				Return(vtc.expCheckUnlinkPublicIPErr)
			vmIds := []string{vmId}
			mockOscLoadBalancerInterface.
				EXPECT().
				UnlinkLoadBalancerBackendMachines(gomock.Eq(vmIds), gomock.Eq(loadBalancerName)).
				Return(vtc.expUnlinkLoadBalancerBackendMachineErr)

			if vtc.expDeleteOutboundSecurityGroupRuleFound {
				mockOscSecurityGroupInterface.
					EXPECT().
					DeleteSecurityGroupRule(gomock.Eq(associateSecurityGroupId), gomock.Eq("Outbound"), gomock.Eq(ipProtocol), "", gomock.Eq(securityGroupIds[0]), gomock.Eq(fromPortRange), gomock.Eq(toPortRange)).
					Return(vtc.expDeleteOutboundSecurityGroupRuleErr)
			}

			if vtc.expDeleteInboundSecurityGroupRuleFound {
				mockOscSecurityGroupInterface.
					EXPECT().
					DeleteSecurityGroupRule(gomock.Eq(associateSecurityGroupId), gomock.Eq("Inbound"), gomock.Eq(ipProtocol), "", gomock.Eq(securityGroupIds[0]), gomock.Eq(fromPortRange), gomock.Eq(toPortRange)).
					Return(vtc.expDeleteInboundSecurityGroupRuleErr)

			}

			reconcileDeleteVm, err := reconcileDeleteVm(ctx, clusterScope, machineScope, mockOscVMInterface, mockOscPublicIPInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface)
			if err != nil {
				assert.Equal(t, vtc.expReconcileDeleteVMErr.Error(), err.Error(), "reconcileDeleteVm() should return the same error")
			} else {
				assert.Nil(t, vtc.expReconcileDeleteVMErr)
			}
			t.Logf("find reconcileDeleteVm %v\n", reconcileDeleteVm)
		})
	}

}
