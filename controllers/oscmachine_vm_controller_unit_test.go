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
				Name:        "test-net",
				IpRange:     "10.0.0.0/16",
				ClusterName: "test-cluster",
			},
			Subnets: []*infrastructurev1beta1.OscSubnet{
				{
					Name:          "test-subnet",
					IpSubnetRange: "10.0.0.0/24",
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
					RootDiskSize: 30,
					RootDiskIops: 1500,
					RootDiskType: "io1",
				},
				KeypairName:      "rke",
				SubregionName:    "eu-west-2a",
				SubnetName:       "test-subnet",
				LoadBalancerName: "test-loadbalancer",
				PublicIpName:     "test-publicip",
				VmType:           "tinav4.c2r4p2",
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
					RootDiskType: "io1",
				},
				KeypairName:      "rke",
				SubregionName:    "eu-west-2a",
				SubnetName:       "test-subnet",
				LoadBalancerName: "test-loadbalancer",
				PublicIpName:     "test-publicip",
				VmType:           "tinav4.c2r4p2",
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
				VmType:           "tinav4.c2r4p2",
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
				VmType:           "tinav4.c2r4p2",
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
)

// SetupWithVmMock set vmMock with clusterScope, machineScope and oscmachine
func SetupWithVmMock(t *testing.T, name string, clusterSpec infrastructurev1beta1.OscClusterSpec, machineSpec infrastructurev1beta1.OscMachineSpec) (clusterScope *scope.ClusterScope, machineScope *scope.MachineScope, ctx context.Context, mockOscVmInterface *mock_compute.MockOscVmInterface, mockOscVolumeInterface *mock_storage.MockOscVolumeInterface, mockOscPublicIpInterface *mock_security.MockOscPublicIpInterface, mockOscLoadBalancerInterface *mock_service.MockOscLoadBalancerInterface, mockOscSecurityGroupInterface *mock_security.MockOscSecurityGroupInterface) {
	clusterScope, machineScope = SetupMachine(t, name, clusterSpec, machineSpec)
	mockCtrl := gomock.NewController(t)
	mockOscVmInterface = mock_compute.NewMockOscVmInterface(mockCtrl)
	mockOscVolumeInterface = mock_storage.NewMockOscVolumeInterface(mockCtrl)
	mockOscPublicIpInterface = mock_security.NewMockOscPublicIpInterface(mockCtrl)
	mockOscLoadBalancerInterface = mock_service.NewMockOscLoadBalancerInterface(mockCtrl)
	mockOscSecurityGroupInterface = mock_security.NewMockOscSecurityGroupInterface(mockCtrl)
	ctx = context.Background()
	return clusterScope, machineScope, ctx, mockOscVmInterface, mockOscVolumeInterface, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface
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
			expGetVmResourceIdErr: fmt.Errorf("test-vm-uid does not exist"),
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
			if err != nil {
				assert.Equal(t, vtc.expGetVmResourceIdErr.Error(), err.Error(), "GetVmResourceId() should return the same error")
			} else {
				assert.Nil(t, vtc.expGetVmResourceIdErr)
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
			expCheckVmVolumeOscAssociateResourceNameErr: fmt.Errorf("test-volume-uid volume does not exist in vm"),
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
						VmType:           "tinav4.c2r4p2",
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
			expCheckVmVolumeOscAssociateResourceNameErr: fmt.Errorf("test-volume@test-uid volume does not exist in vm"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			_, machineScope := SetupMachine(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			err := checkVmVolumeOscAssociateResourceName(machineScope)
			if err != nil {
				assert.Equal(t, vtc.expCheckVmVolumeOscAssociateResourceNameErr, err, "checkVmVolumeOscAssociateResourceName() should return the same eror")
			} else {
				assert.Nil(t, vtc.expCheckVmVolumeOscAssociateResourceNameErr)
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
						VmType:       "tinav4.c2r4p2",
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
			expCheckVmLoadBalancerOscAssociateResourceNameErr: fmt.Errorf("test-loadbalancer@test-uid loadBalancer does not exist in vm"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope := SetupMachine(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			err := checkVmLoadBalancerOscAssociateResourceName(machineScope, clusterScope)
			if err != nil {
				assert.Equal(t, vtc.expCheckVmLoadBalancerOscAssociateResourceNameErr, err, "checkVmLoadBalancerOscAssociateResourceName() should return the same erroor")
			} else {
				assert.Nil(t, vtc.expCheckVmLoadBalancerOscAssociateResourceNameErr)
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
						VmType:           "tinav4.c2r4p2",
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
			expCheckVmSecurityGroupOscAssociateResourceNameErr: fmt.Errorf("test-securitygroup@test-uid securityGroup does not exist in vm"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope := SetupMachine(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			err := checkVmSecurityGroupOscAssociateResourceName(machineScope, clusterScope)
			if err != nil {
				assert.Equal(t, vtc.expCheckVmSecurityGroupOscAssociateResourceNameErr, err, "checkVmSecurityGroupOscAssociateResourceName() should return the same error")
			} else {
				assert.Nil(t, vtc.expCheckVmSecurityGroupOscAssociateResourceNameErr)
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
			expCheckVmPublicIpOscAssociateResourceNameErr: fmt.Errorf("cluster-api-publicip-uid publicIp does not exist in vm"),
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
						VmType:           "tinav4.c2r4p2",
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
			expCheckVmPublicIpOscAssociateResourceNameErr: fmt.Errorf("test-publicip@test-uid publicIp does not exist in vm"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope := SetupMachine(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			err := checkVmPublicIpOscAssociateResourceName(machineScope, clusterScope)
			if err != nil {
				assert.Equal(t, vtc.expCheckVmPublicIpOscAssociateResourceNameErr, err, "checkVmPublicIpOscAssociateResourceName() should return the same error")
			} else {
				assert.Nil(t, vtc.expCheckVmPublicIpOscAssociateResourceNameErr)
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
						VmType:           "tinav4.c2r4p2",
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
			expCheckVmFormatParametersErr: fmt.Errorf("Invalid Tag Name"),
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
						Name:             "test-vm@test",
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
						VmType:           "tinav4.c2r4p2",
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
			expCheckVmFormatParametersErr: fmt.Errorf("Invalid Tag Name"),
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
						VmType:           "tinav4.c2r4p2",
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
			expCheckVmFormatParametersErr: fmt.Errorf("Invalid imageId"),
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
						VmType:           "tinav4.c2r4p2",
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
			expCheckVmFormatParametersErr: fmt.Errorf("Invalid KeypairName"),
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
							RootDiskType: "io1",
						},
						KeypairName:      "rke",
						SubregionName:    "eu-west-2a",
						SubnetName:       "test-subnet",
						LoadBalancerName: "test-loadbalancer",
						VmType:           "tinav4.c2r4p2",
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
			expCheckVmFormatParametersErr: fmt.Errorf("Invalid deviceName"),
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
			expCheckVmFormatParametersErr: fmt.Errorf("Invalid vmType"),
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
						VmType:           "tinav4.c2r4p2",
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
			expCheckVmFormatParametersErr: fmt.Errorf("Invalid ip in cidr"),
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
						SubregionName:    "eu-west-2c",
						SubnetName:       "test-subnet",
						LoadBalancerName: "test-loadbalancer",
						VmType:           "tinav4.c2r4p2",
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
			expCheckVmFormatParametersErr: fmt.Errorf("Invalid subregionName"),
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
						VmType:           "tinav4.c2r4p2",
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
			expCheckVmFormatParametersErr: fmt.Errorf("Invalid size"),
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
						VmType:           "tinav4.c2r4p2",
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
			expCheckVmFormatParametersErr: fmt.Errorf("Invalid iops"),
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
						VmType:           "tinav4.c2r4p2",
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
			expCheckVmFormatParametersErr: fmt.Errorf("Invalid volumeType"),
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
			if err != nil {
				assert.Equal(t, vtc.expCheckVmFormatParametersErr, err, "checkVmFormatParameters() should return the same error")
			} else {
				assert.Nil(t, vtc.expCheckVmFormatParametersErr)
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
						VmType:           "tinav4.c2r4p2",
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
			expCheckVmSubnetAssociateResourceNameErr: fmt.Errorf("test-subnet@test-uid subnet does not exist in vm"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope := SetupMachine(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			err := checkVmSubnetOscAssociateResourceName(machineScope, clusterScope)
			if err != nil {
				assert.Equal(t, vtc.expCheckVmSubnetAssociateResourceNameErr, err, "checkVmSubnetOscAssociateResourceName() should return the same error")
			} else {
				assert.Nil(t, vtc.expCheckVmSubnetAssociateResourceNameErr)
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
						VmType:           "tinav4.c2r4p2",
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
			expCheckVmPrivateIpsOscDuplicateNameErr: fmt.Errorf("test-privateip-first already exist"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			_, machineScope := SetupMachine(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			duplicateResourceVmPrivateIpsErr := checkVmPrivateIpOscDuplicateName(machineScope)
			if duplicateResourceVmPrivateIpsErr != nil {
				assert.Equal(t, vtc.expCheckVmPrivateIpsOscDuplicateNameErr, duplicateResourceVmPrivateIpsErr, "checkVmPrivateIpsOscDuplicateName() should return the same error")
			} else {
				assert.Nil(t, vtc.expCheckVmPrivateIpsOscDuplicateNameErr)
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
						VmType:           "tinav4.c2r4p2",
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
						VmType:           "tinav4.c2r4p2",
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
			expCheckVmVolumeSubregionNameErr: fmt.Errorf("volume test-volume and vm test-vm are not in the same subregion eu-west-2b"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			_, machineScope := SetupMachine(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			err := checkVmVolumeSubregionName(machineScope)
			if err != nil {
				assert.Equal(t, vtc.expCheckVmVolumeSubregionNameErr, err, "checkVmVolumeSubregionName() should return the same error")
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
		expCreateVmFound                        bool
		expLinkPublicIpFound                    bool
		expCreateInboundSecurityGroupRuleFound  bool
		expCreateOutboundSecurityGroupRuleFound bool
		expGetOutboundSecurityGroupRuleFound    bool
		expGetInboundSecurityGroupRuleFound     bool
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
			expCreateVmErr:                          nil,
			expCheckVmStateBootErr:                  nil,
			expCheckVolumeStateAvailableErr:         nil,
			expLinkVolumeErr:                        nil,
			expCheckVolumeStateUseErr:               nil,
			expCheckVmStateVolumeErr:                nil,
			expLinkPublicIpErr:                      nil,
			expCheckVmStatePublicIpErr:              nil,
			expLinkLoadBalancerBackendMachineErr:    nil,
			expCreateOutboundSecurityGroupRuleErr:   fmt.Errorf("CreateSecurityGroupRule generic error"),
			expGetOutboundSecurityGroupRuleErr:      nil,
			expCreateInboundSecurityGroupRuleErr:    nil,
			expGetInboundSecurityGroupRuleErr:       nil,
			expReconcileVmErr:                       fmt.Errorf("CreateSecurityGroupRule generic error Can not create outbound securityGroupRule for OscCluster test-system/test-osc"),
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
			expGetOutboundSecurityGroupRuleErr:      fmt.Errorf("GetSecurityGroupRule generic error"),
			expCreateInboundSecurityGroupRuleErr:    nil,
			expGetInboundSecurityGroupRuleErr:       nil,
			expReconcileVmErr:                       fmt.Errorf("GetSecurityGroupRule generic error Can not get outbound securityGroupRule for OscCluster test-system/test-osc"),
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
			expCreateVmErr:                          nil,
			expCheckVmStateBootErr:                  nil,
			expCheckVolumeStateAvailableErr:         nil,
			expLinkVolumeErr:                        nil,
			expCheckVolumeStateUseErr:               nil,
			expCheckVmStateVolumeErr:                nil,
			expLinkPublicIpErr:                      nil,
			expCheckVmStatePublicIpErr:              nil,
			expLinkLoadBalancerBackendMachineErr:    nil,
			expCreateInboundSecurityGroupRuleErr:    fmt.Errorf("CreateSecurityGroupRule generic error"),
			expGetInboundSecurityGroupRuleErr:       nil,
			expCreateOutboundSecurityGroupRuleErr:   nil,
			expGetOutboundSecurityGroupRuleErr:      nil,
			expReconcileVmErr:                       fmt.Errorf("CreateSecurityGroupRule generic error Can not create inbound securityGroupRule for OscCluster test-system/test-osc"),
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
			expGetInboundSecurityGroupRuleErr:       fmt.Errorf("GetSecurityGroupRule generic error"),
			expCreateOutboundSecurityGroupRuleErr:   nil,
			expGetOutboundSecurityGroupRuleErr:      nil,
			expReconcileVmErr:                       fmt.Errorf("GetSecurityGroupRule generic error Can not get inbound securityGroupRule for OscCluster test-system/test-osc"),
		},
		{
			name:             "linkPublicIp does not exist",
			clusterSpec:      defaultVmClusterInitialize,
			machineSpec:      defaultVmInitialize,
			expCreateVmFound: true,

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
			expReconcileVmErr:                       nil,
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope, ctx, mockOscVmInterface, mockOscVolumeInterface, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface := SetupWithVmMock(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			vmName := vtc.machineSpec.Node.Vm.Name + "-uid"
			vmId := "i-" + vmName
			vmState := "running"

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
			vmSpec := vtc.machineSpec.Node.Vm
			var clockInsideLoop time.Duration = 5
			var clockLoop time.Duration = 60
			var firstClockLoop time.Duration = 120
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
					CreateVm(gomock.Eq(machineScope), gomock.Eq(&vmSpec), gomock.Eq(subnetId), gomock.Eq(securityGroupIds), gomock.Eq(privateIps), gomock.Eq(vmName)).
					Return(vm, vtc.expCreateVmErr)
			} else {
				mockOscVmInterface.
					EXPECT().
					CreateVm(gomock.Eq(machineScope), gomock.Eq(vmSpec), gomock.Eq(subnetId), gomock.Eq(securityGroupIds), gomock.Eq(privateIps), gomock.Eq(vmName)).
					Return(nil, vtc.expCreateVmErr)
			}

			mockOscVmInterface.
				EXPECT().
				CheckVmState(gomock.Eq(clockInsideLoop), gomock.Eq(firstClockLoop), gomock.Eq(vmState), gomock.Eq(vmId)).
				Return(vtc.expCheckVmStateBootErr)

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

			mockOscVmInterface.
				EXPECT().
				CheckVmState(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(vmState), gomock.Eq(vmId)).
				Return(vtc.expCheckVmStateVolumeErr)

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

			mockOscVmInterface.
				EXPECT().
				CheckVmState(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(vmState), gomock.Eq(vmId)).
				Return(vtc.expCheckVmStatePublicIpErr)

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
			reconcileVm, err := reconcileVm(ctx, clusterScope, machineScope, mockOscVmInterface, mockOscVolumeInterface, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface)
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
		expCreateVmFound  bool
		expCreateVmErr    error
		expReconcileVmErr error
	}{
		{
			name:              "failed to create vm",
			clusterSpec:       defaultVmClusterInitialize,
			machineSpec:       defaultVmInitialize,
			expCreateVmFound:  false,
			expCreateVmErr:    fmt.Errorf("CreateVm generic error"),
			expReconcileVmErr: fmt.Errorf("CreateVm generic error Can not create vm for OscMachine test-system/test-osc"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope, ctx, mockOscVmInterface, mockOscVolumeInterface, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface := SetupWithVmMock(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
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
					CreateVm(gomock.Eq(machineScope), gomock.Eq(&vmSpec), gomock.Eq(subnetId), gomock.Eq(securityGroupIds), gomock.Eq(privateIps), gomock.Eq(vmName)).
					Return(vm, vtc.expCreateVmErr)
			} else {
				mockOscVmInterface.
					EXPECT().
					CreateVm(gomock.Eq(machineScope), gomock.Eq(&vmSpec), gomock.Eq(subnetId), gomock.Eq(securityGroupIds), gomock.Eq(privateIps), gomock.Eq(vmName)).
					Return(nil, vtc.expCreateVmErr)
			}

			reconcileVm, err := reconcileVm(ctx, clusterScope, machineScope, mockOscVmInterface, mockOscVolumeInterface, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface)
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
		expCreateVmFound                  bool
		expLinkVolumeFound                bool
		expCheckVmStateBootFound          bool
		expCheckVolumeStateAvailableFound bool
		expCreateVmErr                    error
		expReconcileVmErr                 error
		expCheckVmStateBootErr            error
		expCheckVolumeStateAvailableErr   error
		expLinkVolumeErr                  error
	}{
		{
			name:                              "failed to link volume with vm",
			clusterSpec:                       defaultVmClusterInitialize,
			machineSpec:                       defaultVmVolumeInitialize,
			expCreateVmFound:                  true,
			expCreateVmErr:                    nil,
			expCheckVmStateBootFound:          true,
			expCheckVmStateBootErr:            nil,
			expCheckVolumeStateAvailableFound: true,
			expCheckVolumeStateAvailableErr:   nil,
			expLinkVolumeFound:                true,
			expLinkVolumeErr:                  fmt.Errorf("LinkVolume generic error"),
			expReconcileVmErr:                 fmt.Errorf("LinkVolume generic error Can not link volume vol-test-volume-uid with vm i-test-vm-uid for OscMachine test-system/test-osc"),
		},
		{
			name:                              "failed check vm state boot",
			clusterSpec:                       defaultVmClusterInitialize,
			machineSpec:                       defaultVmVolumeInitialize,
			expCreateVmFound:                  true,
			expCreateVmErr:                    nil,
			expCheckVmStateBootFound:          true,
			expCheckVmStateBootErr:            fmt.Errorf("checkVmState generic error"),
			expCheckVolumeStateAvailableFound: false,
			expCheckVolumeStateAvailableErr:   nil,
			expLinkVolumeFound:                false,
			expLinkVolumeErr:                  nil,
			expReconcileVmErr:                 fmt.Errorf("checkVmState generic error Can not get vm i-test-vm-uid running for OscMachine test-system/test-osc"),
		},
		{
			name:                              "failed check volume state boot",
			clusterSpec:                       defaultVmClusterInitialize,
			machineSpec:                       defaultVmVolumeInitialize,
			expCreateVmFound:                  true,
			expCreateVmErr:                    nil,
			expCheckVmStateBootFound:          true,
			expCheckVmStateBootErr:            nil,
			expCheckVolumeStateAvailableFound: true,
			expCheckVolumeStateAvailableErr:   fmt.Errorf("checkVolumeState generic error"),
			expLinkVolumeFound:                false,
			expLinkVolumeErr:                  nil,
			expReconcileVmErr:                 fmt.Errorf("checkVolumeState generic error Can not get volume vol-test-volume-uid available for OscMachine test-system/test-osc"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope, ctx, mockOscVmInterface, mockOscVolumeInterface, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface := SetupWithVmMock(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			vmName := vtc.machineSpec.Node.Vm.Name + "-uid"
			vmId := "i-" + vmName
			vmState := "running"

			volumeName := vtc.machineSpec.Node.Vm.VolumeName + "-uid"
			volumeId := "vol-" + volumeName
			volumeRef := machineScope.GetVolumeRef()
			volumeRef.ResourceMap = make(map[string]string)
			volumeRef.ResourceMap[volumeName] = volumeId
			volumeStateAvailable := "available"

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
			volumeDeviceName := vtc.machineSpec.Node.Vm.VolumeDeviceName

			var clockInsideLoop time.Duration = 5
			var clockLoop time.Duration = 60
			var firstClockLoop time.Duration = 120
			if vtc.expCreateVmFound {
				mockOscVmInterface.
					EXPECT().
					CreateVm(gomock.Eq(machineScope), gomock.Eq(&vmSpec), gomock.Eq(subnetId), gomock.Eq(securityGroupIds), gomock.Eq(privateIps), gomock.Eq(vmName)).
					Return(vm, vtc.expCreateVmErr)
			} else {
				mockOscVmInterface.
					EXPECT().
					CreateVm(gomock.Eq(machineScope), gomock.Eq(vmSpec), gomock.Eq(subnetId), gomock.Eq(securityGroupIds), gomock.Eq(privateIps), gomock.Eq(vmName)).
					Return(nil, vtc.expCreateVmErr)
			}

			if vtc.expCheckVmStateBootFound {
				mockOscVmInterface.
					EXPECT().
					CheckVmState(gomock.Eq(clockInsideLoop), gomock.Eq(firstClockLoop), gomock.Eq(vmState), gomock.Eq(vmId)).
					Return(vtc.expCheckVmStateBootErr)
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
					LinkVolume(gomock.Eq(volumeId), gomock.Eq(vmId), gomock.Eq(volumeDeviceName)).
					Return(vtc.expLinkVolumeErr)
			}

			reconcileVm, err := reconcileVm(ctx, clusterScope, machineScope, mockOscVmInterface, mockOscVolumeInterface, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface)
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
		expCreateVmFound                bool
		expCheckVolumeStateUseFound     bool
		expCheckVmStateVolumeFound      bool
		expCheckVmStatePublicIpFound    bool
		expLinkPublicIpFound            bool
		expCreateVmErr                  error
		expReconcileVmErr               error
		expCheckVmStateBootErr          error
		expCheckVolumeStateAvailableErr error
		expLinkVolumeErr                error
		expCheckVolumeStateUseErr       error
		expCheckVmStateVolumeErr        error
		expLinkPublicIpErr              error
		expCheckVmStatePublicIpErr      error
	}{
		{
			name:                            "failed linkPublicIp",
			clusterSpec:                     defaultVmClusterInitialize,
			machineSpec:                     defaultVmInitialize,
			expCreateVmFound:                true,
			expLinkPublicIpFound:            true,
			expCheckVolumeStateUseFound:     false,
			expCheckVmStateVolumeFound:      true,
			expCheckVmStatePublicIpFound:    false,
			expCreateVmErr:                  nil,
			expCheckVmStateBootErr:          nil,
			expCheckVolumeStateAvailableErr: nil,
			expLinkVolumeErr:                nil,
			expCheckVolumeStateUseErr:       nil,
			expCheckVmStateVolumeErr:        nil,
			expLinkPublicIpErr:              fmt.Errorf("linkPublicIp generic error"),
			expCheckVmStatePublicIpErr:      nil,
			expReconcileVmErr:               fmt.Errorf("linkPublicIp generic error Can not link publicIp  eipalloc-test-publicip-uid with i-test-vm-uid for OscCluster test-system/test-osc"),
		},
		{
			name:                            "failed VmStatePublicIp",
			clusterSpec:                     defaultVmClusterInitialize,
			machineSpec:                     defaultVmInitialize,
			expCreateVmFound:                true,
			expLinkPublicIpFound:            true,
			expCheckVolumeStateUseFound:     false,
			expCheckVmStateVolumeFound:      true,
			expCheckVmStatePublicIpFound:    true,
			expCreateVmErr:                  nil,
			expCheckVmStateBootErr:          nil,
			expCheckVolumeStateAvailableErr: nil,
			expLinkVolumeErr:                nil,
			expCheckVolumeStateUseErr:       nil,
			expCheckVmStateVolumeErr:        nil,
			expLinkPublicIpErr:              nil,
			expCheckVmStatePublicIpErr:      fmt.Errorf("CheckVmState generic error"),
			expReconcileVmErr:               fmt.Errorf("CheckVmState generic error Can not get vm i-test-vm-uid running for OscMachine test-system/test-osc"),
		},
		{
			name:                            "failed VolumeStateUse",
			clusterSpec:                     defaultVmClusterInitialize,
			machineSpec:                     defaultVmVolumeInitialize,
			expCreateVmFound:                true,
			expLinkPublicIpFound:            false,
			expCheckVolumeStateUseFound:     true,
			expCheckVmStateVolumeFound:      false,
			expCheckVmStatePublicIpFound:    false,
			expCreateVmErr:                  nil,
			expCheckVmStateBootErr:          nil,
			expCheckVolumeStateAvailableErr: nil,
			expLinkVolumeErr:                nil,
			expCheckVolumeStateUseErr:       fmt.Errorf("CheckVolumeState generic error"),
			expCheckVmStateVolumeErr:        nil,
			expLinkPublicIpErr:              nil,
			expCheckVmStatePublicIpErr:      nil,
			expReconcileVmErr:               fmt.Errorf("CheckVolumeState generic error Can not get volume vol-test-volume-uid in use for OscMachine test-system/test-osc"),
		},
		{
			name:                            "failed VmStateVolume",
			clusterSpec:                     defaultVmClusterInitialize,
			machineSpec:                     defaultVmInitialize,
			expCreateVmFound:                true,
			expLinkPublicIpFound:            false,
			expCheckVolumeStateUseFound:     false,
			expCheckVmStateVolumeFound:      true,
			expCheckVmStatePublicIpFound:    false,
			expCreateVmErr:                  nil,
			expCheckVmStateBootErr:          nil,
			expCheckVolumeStateAvailableErr: nil,
			expLinkVolumeErr:                nil,
			expCheckVolumeStateUseErr:       nil,
			expCheckVmStateVolumeErr:        fmt.Errorf("CheckVmState generic error"),
			expLinkPublicIpErr:              nil,
			expCheckVmStatePublicIpErr:      nil,
			expReconcileVmErr:               fmt.Errorf("CheckVmState generic error Can not get vm i-test-vm-uid running for OscMachine test-system/test-osc"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope, ctx, mockOscVmInterface, mockOscVolumeInterface, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface := SetupWithVmMock(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			vmName := vtc.machineSpec.Node.Vm.Name + "-uid"
			vmId := "i-" + vmName
			vmState := "running"

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

			vmSpec := vtc.machineSpec.Node.Vm
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
			volumeDeviceName := vtc.machineSpec.Node.Vm.VolumeDeviceName

			if vtc.expCreateVmFound {
				mockOscVmInterface.
					EXPECT().
					CreateVm(gomock.Eq(machineScope), gomock.Eq(&vmSpec), gomock.Eq(subnetId), gomock.Eq(securityGroupIds), gomock.Eq(privateIps), gomock.Eq(vmName)).
					Return(vm, vtc.expCreateVmErr)
			} else {
				mockOscVmInterface.
					EXPECT().
					CreateVm(gomock.Eq(machineScope), gomock.Eq(vmSpec), gomock.Eq(subnetId), gomock.Eq(securityGroupIds), gomock.Eq(privateIps), gomock.Eq(vmName)).
					Return(nil, vtc.expCreateVmErr)
			}

			mockOscVmInterface.
				EXPECT().
				CheckVmState(gomock.Eq(clockInsideLoop), gomock.Eq(firstClockLoop), gomock.Eq(vmState), gomock.Eq(vmId)).
				Return(vtc.expCheckVmStateBootErr)

			if vtc.machineSpec.Node.Vm.VolumeName != "" {
				mockOscVolumeInterface.
					EXPECT().
					CheckVolumeState(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(volumeStateAvailable), gomock.Eq(volumeId)).
					Return(vtc.expCheckVolumeStateAvailableErr)

				mockOscVolumeInterface.
					EXPECT().
					LinkVolume(gomock.Eq(volumeId), gomock.Eq(vmId), gomock.Eq(volumeDeviceName)).
					Return(vtc.expLinkVolumeErr)

			}
			if vtc.expCheckVolumeStateUseFound {
				mockOscVolumeInterface.
					EXPECT().
					CheckVolumeState(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(volumeStateUse), gomock.Eq(volumeId)).
					Return(vtc.expCheckVolumeStateUseErr)
			}
			if vtc.expCheckVmStateVolumeFound {
				mockOscVmInterface.
					EXPECT().
					CheckVmState(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(vmState), gomock.Eq(vmId)).
					Return(vtc.expCheckVmStateVolumeErr)
			}

			if vtc.expLinkPublicIpFound {
				mockOscPublicIpInterface.
					EXPECT().
					LinkPublicIp(gomock.Eq(publicIpId), gomock.Eq(vmId)).
					Return("", vtc.expLinkPublicIpErr)
			}
			if vtc.expCheckVmStatePublicIpFound {
				mockOscVmInterface.
					EXPECT().
					CheckVmState(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(vmState), gomock.Eq(vmId)).
					Return(vtc.expCheckVmStatePublicIpErr)
			}

			reconcileVm, err := reconcileVm(ctx, clusterScope, machineScope, mockOscVmInterface, mockOscVolumeInterface, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface)
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
		expCreateVmFound                     bool
		expLinkPublicIpFound                 bool
		expCreateSecurityGroupRuleFound      bool
		expCreateVmErr                       error
		expReconcileVmErr                    error
		expCheckVmStateBootErr               error
		expCheckVolumeStateAvailableErr      error
		expLinkVolumeErr                     error
		expCheckVolumeStateUseErr            error
		expCheckVmStateVolumeErr             error
		expCreateSecurityGroupRuleErr        error
		expLinkPublicIpErr                   error
		expCheckVmStatePublicIpErr           error
		expLinkLoadBalancerBackendMachineErr error
	}{
		{
			name:                                 "failed to link LoadBalancerBackendMachine ",
			clusterSpec:                          defaultVmClusterInitialize,
			machineSpec:                          defaultVmInitialize,
			expCreateVmFound:                     true,
			expLinkPublicIpFound:                 true,
			expCreateSecurityGroupRuleFound:      false,
			expCreateVmErr:                       nil,
			expCheckVmStateBootErr:               nil,
			expCheckVolumeStateAvailableErr:      nil,
			expLinkVolumeErr:                     nil,
			expCheckVolumeStateUseErr:            nil,
			expCheckVmStateVolumeErr:             nil,
			expLinkPublicIpErr:                   nil,
			expCheckVmStatePublicIpErr:           nil,
			expLinkLoadBalancerBackendMachineErr: fmt.Errorf("LinkLoadBalancerBackendMachine generic error"),
			expCreateSecurityGroupRuleErr:        nil,
			expReconcileVmErr:                    fmt.Errorf("LinkLoadBalancerBackendMachine generic error Can not link vm test-loadbalancer with loadBalancerName i-test-vm-uid for OscCluster test-system/test-osc"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope, ctx, mockOscVmInterface, mockOscVolumeInterface, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface := SetupWithVmMock(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			vmName := vtc.machineSpec.Node.Vm.Name + "-uid"
			vmId := "i-" + vmName
			vmState := "running"

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

			deviceName := vtc.machineSpec.Node.Vm.DeviceName
			vmSpec := vtc.machineSpec.Node.Vm
			var clockInsideLoop time.Duration = 5
			var clockLoop time.Duration = 60
			var firstClockLoop time.Duration = 120
			loadBalancerName := vtc.machineSpec.Node.Vm.LoadBalancerName

			createVms := osc.CreateVmsResponse{
				Vms: &[]osc.Vm{
					{
						VmId: &vmId,
					},
				},
			}

			createVm := *createVms.Vms
			linkPublicIp := osc.LinkPublicIpResponse{
				LinkPublicIpId: &linkPublicIpId,
			}
			vm := &createVm[0]
			if vtc.expCreateVmFound {
				mockOscVmInterface.
					EXPECT().
					CreateVm(gomock.Eq(machineScope), gomock.Eq(&vmSpec), gomock.Eq(subnetId), gomock.Eq(securityGroupIds), gomock.Eq(privateIps), gomock.Eq(vmName)).
					Return(vm, vtc.expCreateVmErr)
			} else {
				mockOscVmInterface.
					EXPECT().
					CreateVm(gomock.Eq(machineScope), gomock.Eq(vmSpec), gomock.Eq(subnetId), gomock.Eq(securityGroupIds), gomock.Eq(privateIps), gomock.Eq(vmName)).
					Return(nil, vtc.expCreateVmErr)
			}

			mockOscVmInterface.
				EXPECT().
				CheckVmState(gomock.Eq(clockInsideLoop), gomock.Eq(firstClockLoop), gomock.Eq(vmState), gomock.Eq(vmId)).
				Return(vtc.expCheckVmStateBootErr)
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
			mockOscVmInterface.
				EXPECT().
				CheckVmState(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(vmState), gomock.Eq(vmId)).
				Return(vtc.expCheckVmStateVolumeErr)

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

			mockOscVmInterface.
				EXPECT().
				CheckVmState(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(vmState), gomock.Eq(vmId)).
				Return(vtc.expCheckVmStatePublicIpErr)

			vmIds := []string{vmId}
			mockOscLoadBalancerInterface.
				EXPECT().
				LinkLoadBalancerBackendMachines(gomock.Eq(vmIds), gomock.Eq(loadBalancerName)).
				Return(vtc.expLinkLoadBalancerBackendMachineErr)

			reconcileVm, err := reconcileVm(ctx, clusterScope, machineScope, mockOscVmInterface, mockOscVolumeInterface, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface)
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
		name                   string
		clusterSpec            infrastructurev1beta1.OscClusterSpec
		machineSpec            infrastructurev1beta1.OscMachineSpec
		expGetVmFound          bool
		expGetVmStateFound     bool
		expAddCcmTagFound      bool
		expPrivateDnsNameFound bool
		expGetVmStateErr       error
		expGetVmErr            error
		expAddCcmTagErr        error
		expPrivateDnsNameErr   error
		expReconcileVmErr      error
	}{
		{
			name:                   "get vm",
			clusterSpec:            defaultVmClusterInitialize,
			machineSpec:            defaultVmReconcile,
			expGetVmFound:          true,
			expGetVmStateFound:     true,
			expAddCcmTagFound:      true,
			expPrivateDnsNameFound: true,
			expGetVmErr:            nil,
			expGetVmStateErr:       nil,
			expAddCcmTagErr:        nil,
			expPrivateDnsNameErr:   nil,
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
			expGetVmErr:            fmt.Errorf("GetVm generic error"),
			expGetVmStateErr:       nil,
			expAddCcmTagErr:        nil,
			expPrivateDnsNameErr:   nil,
			expReconcileVmErr:      fmt.Errorf("GetVm generic error"),
		},
		{
			name:                   "failed to set vmstate",
			clusterSpec:            defaultVmClusterInitialize,
			machineSpec:            defaultVmReconcile,
			expGetVmFound:          true,
			expGetVmStateFound:     true,
			expAddCcmTagFound:      true,
			expPrivateDnsNameFound: true,
			expGetVmErr:            nil,
			expAddCcmTagErr:        nil,
			expGetVmStateErr:       fmt.Errorf("GetVmState generic error"),
			expPrivateDnsNameErr:   nil,
			expReconcileVmErr:      fmt.Errorf("GetVmState generic error Can not get vm i-test-vm-uid state for OscMachine test-system/test-osc"),
		},
		{
			name:                   "failed to add tag",
			clusterSpec:            defaultVmClusterReconcile,
			machineSpec:            defaultVmReconcile,
			expGetVmFound:          true,
			expGetVmStateFound:     false,
			expAddCcmTagFound:      true,
			expPrivateDnsNameFound: true,
			expGetVmErr:            nil,
			expGetVmStateErr:       nil,
			expAddCcmTagErr:        fmt.Errorf("AddCcmTag generic error"),
			expPrivateDnsNameErr:   nil,
			expReconcileVmErr:      fmt.Errorf("AddCcmTag generic error can not add ccm tag test-system/test-osc"),
		},
		{
			name:                   "Failed to retrieve privateDnsName",
			clusterSpec:            defaultVmClusterReconcile,
			machineSpec:            defaultVmReconcile,
			expGetVmFound:          true,
			expGetVmStateFound:     false,
			expAddCcmTagFound:      false,
			expPrivateDnsNameFound: false,
			expGetVmErr:            nil,
			expGetVmStateErr:       nil,
			expAddCcmTagErr:        nil,
			expPrivateDnsNameErr:   fmt.Errorf("GetPrivateDnsNameok generic error"),
			expReconcileVmErr:      fmt.Errorf("Can not found privateDnsName test-system/test-osc"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope, ctx, mockOscVmInterface, mockOscVolumeInterface, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface := SetupWithVmMock(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			vmName := vtc.machineSpec.Node.Vm.Name + "-uid"
			vmId := "i-" + vmName
			vmState := "running"

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
			if vtc.expGetVmFound {
				mockOscVmInterface.
					EXPECT().
					GetVm(gomock.Eq(vmId)).
					Return(vm, vtc.expGetVmErr)
			} else {
				mockOscVmInterface.
					EXPECT().
					GetVm(gomock.Eq(vmId)).
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
			reconcileVm, err := reconcileVm(ctx, clusterScope, machineScope, mockOscVmInterface, mockOscVolumeInterface, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface)
			if err != nil {
				assert.Equal(t, vtc.expReconcileVmErr.Error(), err.Error(), "reconcileVm() should return the same error")
			} else {
				assert.Nil(t, vtc.expReconcileVmErr)
			}
			t.Logf("find reconcileVm %v\n", reconcileVm)

		})
	}
}

// TestReconcileVmResourceId has several tests to cover the code of the function reconcileVm
func TestReconcileVmResourceId(t *testing.T) {
	vmTestCases := []struct {
		name                              string
		clusterSpec                       infrastructurev1beta1.OscClusterSpec
		machineSpec                       infrastructurev1beta1.OscMachineSpec
		expVolumeFound                    bool
		expSubnetFound                    bool
		expPublicIpFound                  bool
		expLinkPublicIpFound              bool
		expSecurityGroupFound             bool
		expLoadBalancerSecurityGroupFound bool
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
			expReconcileVmErr:                 fmt.Errorf("test-subnet-uid does not exist"),
		},
		{
			name:                              "PublicIp does not exist ",
			clusterSpec:                       defaultVmClusterInitialize,
			machineSpec:                       defaultVmInitialize,
			expVolumeFound:                    true,
			expSubnetFound:                    true,
			expPublicIpFound:                  false,
			expLinkPublicIpFound:              true,
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
			expPublicIpFound:                  true,
			expLinkPublicIpFound:              true,
			expSecurityGroupFound:             false,
			expLoadBalancerSecurityGroupFound: false,
			expReconcileVmErr:                 fmt.Errorf("test-securitygroup-uid does not exist"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope, ctx, mockOscVmInterface, mockOscVolumeInterface, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface := SetupWithVmMock(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			vmName := vtc.machineSpec.Node.Vm.Name + "-uid"

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

			reconcileVm, err := reconcileVm(ctx, clusterScope, machineScope, mockOscVmInterface, mockOscVolumeInterface, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface)
			if err != nil {
				assert.Equal(t, vtc.expReconcileVmErr, err, "reconcileVm() should return the same error")
			} else {
				assert.Nil(t, vtc.expReconcileVmErr)
			}
			t.Logf("find reconcileVm %v\n", reconcileVm)
		})
	}
}

// TestReconcileDeleteVm has several tests to cover the code of the function reconcileDeleteVm
func TestReconcileDeleteVm(t *testing.T) {
	vmTestCases := []struct {
		name                                    string
		clusterSpec                             infrastructurev1beta1.OscClusterSpec
		machineSpec                             infrastructurev1beta1.OscMachineSpec
		expDeleteInboundSecurityGroupRuleFound  bool
		expDeleteOutboundSecurityGroupRuleFound bool
		expCheckVmStateBootErr                  error
		expUnlinkLoadBalancerBackendMachineErr  error
		expCheckVmStateLoadBalancerErr          error
		expDeleteInboundSecurityGroupRuleErr    error
		expDeleteOutboundSecurityGroupRuleErr   error
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
			expCheckVmStateBootErr:                  nil,
			expDeleteInboundSecurityGroupRuleFound:  true,
			expDeleteOutboundSecurityGroupRuleFound: true,
			expUnlinkLoadBalancerBackendMachineErr:  nil,
			expCheckVmStateLoadBalancerErr:          nil,
			expDeleteInboundSecurityGroupRuleErr:    nil,
			expDeleteOutboundSecurityGroupRuleErr:   nil,
			expSecurityGroupRuleFound:               true,
			expDeleteVmErr:                          nil,
			expGetVmFound:                           true,
			expGetVmErr:                             nil,
			expCheckUnlinkPublicIpErr:               nil,
			expReconcileDeleteVmErr:                 nil,
		},
		{
			name:                                    "failed to delete vm",
			clusterSpec:                             defaultVmClusterReconcile,
			machineSpec:                             defaultVmReconcile,
			expCheckVmStateBootErr:                  nil,
			expDeleteInboundSecurityGroupRuleFound:  true,
			expDeleteOutboundSecurityGroupRuleFound: true,
			expUnlinkLoadBalancerBackendMachineErr:  nil,
			expCheckVmStateLoadBalancerErr:          nil,
			expSecurityGroupRuleFound:               true,
			expDeleteVmErr:                          fmt.Errorf("DeleteVm generic error"),
			expGetVmFound:                           true,
			expGetVmErr:                             nil,
			expDeleteInboundSecurityGroupRuleErr:    nil,
			expDeleteOutboundSecurityGroupRuleErr:   nil,
			expCheckUnlinkPublicIpErr:               nil,
			expReconcileDeleteVmErr:                 fmt.Errorf("DeleteVm generic error Can not delete vm for OscMachine test-system/test-osc"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope, ctx, mockOscVmInterface, _, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface := SetupWithVmMock(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			vmName := vtc.machineSpec.Node.Vm.Name + "-uid"
			vmId := "i-" + vmName
			vmRef := machineScope.GetVmRef()
			vmRef.ResourceMap = make(map[string]string)
			if vtc.expGetVmFound {
				vmRef.ResourceMap[vmName] = vmId
			}
			vmState := "running"

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
			publicIpName := vtc.machineSpec.Node.Vm.PublicIpName + "-uid"
			linkPublicIpId := "eipassoc-" + publicIpName
			linkPublicIpRef := machineScope.GetLinkPublicIpRef()
			linkPublicIpRef.ResourceMap = make(map[string]string)
			linkPublicIpRef.ResourceMap[publicIpName] = linkPublicIpId

			var clockInsideLoop time.Duration = 5
			var clockLoop time.Duration = 60
			var firstClockLoop time.Duration = 120
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
			vm := &createVm[0]
			if vtc.expGetVmFound {
				mockOscVmInterface.
					EXPECT().
					GetVm(gomock.Eq(vmId)).
					Return(vm, vtc.expGetVmErr)
			} else {
				mockOscVmInterface.
					EXPECT().
					GetVm(gomock.Eq(vmId)).
					Return(nil, vtc.expGetVmErr)
			}
			mockOscVmInterface.
				EXPECT().
				CheckVmState(gomock.Eq(clockInsideLoop), gomock.Eq(firstClockLoop), gomock.Eq(vmState), gomock.Eq(vmId)).
				Return(vtc.expCheckVmStateBootErr)

			mockOscVmInterface.
				EXPECT().
				CheckVmState(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(vmState), gomock.Eq(vmId)).
				Return(vtc.expCheckVmStateLoadBalancerErr)

			mockOscPublicIpInterface.
				EXPECT().
				UnlinkPublicIp(gomock.Eq(linkPublicIpId)).
				Return(vtc.expCheckUnlinkPublicIpErr)
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

			mockOscVmInterface.
				EXPECT().
				DeleteVm(gomock.Eq(vmId)).
				Return(vtc.expDeleteVmErr)
			reconcileDeleteVm, err := reconcileDeleteVm(ctx, clusterScope, machineScope, mockOscVmInterface, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface)
			if err != nil {
				assert.Equal(t, vtc.expReconcileDeleteVmErr.Error(), err.Error(), "reconccileDeleteVm() hould return the same error")
			} else {
				assert.Nil(t, vtc.expReconcileDeleteVmErr)
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
		expCheckVmStateBootErr      error
		expCheckVmStateBootFound    bool
		expCheckUnlinkPublicIpFound bool
		expGetVmFound               bool
		expGetVmErr                 error

		expCheckUnlinkPublicIpErr error
		expReconcileDeleteVmErr   error
	}{
		{
			name:                        "failed vmState",
			clusterSpec:                 defaultVmClusterReconcile,
			machineSpec:                 defaultVmReconcile,
			expCheckVmStateBootErr:      fmt.Errorf("CheckVmState generic error"),
			expCheckVmStateBootFound:    true,
			expGetVmFound:               true,
			expGetVmErr:                 nil,
			expCheckUnlinkPublicIpFound: false,
			expCheckUnlinkPublicIpErr:   nil,
			expReconcileDeleteVmErr:     fmt.Errorf("CheckVmState generic error Can not get vm i-test-vm-uid running for OscMachine test-system/test-osc"),
		},
		{
			name:                        "failed unlink volume",
			clusterSpec:                 defaultVmClusterReconcile,
			machineSpec:                 defaultVmReconcile,
			expCheckVmStateBootErr:      nil,
			expCheckVmStateBootFound:    true,
			expGetVmFound:               true,
			expGetVmErr:                 nil,
			expCheckUnlinkPublicIpFound: true,
			expCheckUnlinkPublicIpErr:   fmt.Errorf("CheckUnlinkPublicIp generic error"),
			expReconcileDeleteVmErr:     fmt.Errorf("CheckUnlinkPublicIp generic error Can not unlink publicIp for OscCluster test-system/test-osc"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope, ctx, mockOscVmInterface, _, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface := SetupWithVmMock(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			vmName := vtc.machineSpec.Node.Vm.Name + "-uid"
			vmId := "i-" + vmName
			vmRef := machineScope.GetVmRef()
			vmRef.ResourceMap = make(map[string]string)
			if vtc.expGetVmFound {
				vmRef.ResourceMap[vmName] = vmId
			}
			vmState := "running"

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
			publicIpName := vtc.machineSpec.Node.Vm.PublicIpName + "-uid"
			linkPublicIpId := "eipassoc-" + publicIpName
			linkPublicIpRef := machineScope.GetLinkPublicIpRef()
			linkPublicIpRef.ResourceMap = make(map[string]string)
			linkPublicIpRef.ResourceMap[publicIpName] = linkPublicIpId

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
				mockOscVmInterface.
					EXPECT().
					GetVm(gomock.Eq(vmId)).
					Return(vm, vtc.expGetVmErr)
			} else {
				mockOscVmInterface.
					EXPECT().
					GetVm(gomock.Eq(vmId)).
					Return(nil, vtc.expGetVmErr)
			}
			if vtc.expCheckVmStateBootFound {
				mockOscVmInterface.
					EXPECT().
					CheckVmState(gomock.Eq(clockInsideLoop), gomock.Eq(firstClockLoop), gomock.Eq(vmState), gomock.Eq(vmId)).
					Return(vtc.expCheckVmStateBootErr)
			}

			if vtc.expCheckUnlinkPublicIpFound {
				mockOscPublicIpInterface.
					EXPECT().
					UnlinkPublicIp(gomock.Eq(linkPublicIpId)).
					Return(vtc.expCheckUnlinkPublicIpErr)
			}

			reconcileDeleteVm, err := reconcileDeleteVm(ctx, clusterScope, machineScope, mockOscVmInterface, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface)
			if err != nil {
				assert.Equal(t, vtc.expReconcileDeleteVmErr.Error(), err.Error(), "reconcileDeleteVm() should return the same error")
			} else {
				assert.Nil(t, vtc.expReconcileDeleteVmErr)
			}
			t.Logf("find reconcileDeleteVm %v\n", reconcileDeleteVm)
		})
	}

}

// TestReconcileDeleteVmLoadBalancer has several tests to cover the code of the function reconcileDeleteVm
func TestReconcileDeleteVmLoadBalancer(t *testing.T) {
	vmTestCases := []struct {
		name                                     string
		clusterSpec                              infrastructurev1beta1.OscClusterSpec
		machineSpec                              infrastructurev1beta1.OscMachineSpec
		expCheckVmStateBootErr                   error
		expUnlinkLoadBalancerBackendMachineErr   error
		expCheckVmStateLoadBalancerErr           error
		expDeleteSecurityGroupRuleErr            error
		expCheckVmStateLoadBalancerFound         bool
		expUnlinkLoadBalancerBackendMachineFound bool
		expDeleteSecurityGroupRuleFound          bool
		expGetVmErr                              error
		expGetVmFound                            bool
		expCheckUnlinkPublicIpErr                error
		expReconcileDeleteVmErr                  error
	}{
		{
			name:                                     "failed checkVmState",
			clusterSpec:                              defaultVmClusterReconcile,
			machineSpec:                              defaultVmReconcile,
			expCheckVmStateBootErr:                   nil,
			expUnlinkLoadBalancerBackendMachineErr:   nil,
			expCheckVmStateLoadBalancerErr:           fmt.Errorf("CheckVmStateLoadBalancer generic error"),
			expDeleteSecurityGroupRuleErr:            nil,
			expDeleteSecurityGroupRuleFound:          false,
			expCheckVmStateLoadBalancerFound:         true,
			expUnlinkLoadBalancerBackendMachineFound: false,
			expGetVmFound:                            true,
			expGetVmErr:                              nil,
			expCheckUnlinkPublicIpErr:                nil,
			expReconcileDeleteVmErr:                  fmt.Errorf("CheckVmStateLoadBalancer generic error Can not get vm i-test-vm-uid running for OscMachine test-system/test-osc"),
		},
		{
			name:                                     "failed UnlinkLoadBalancerBackendMachineFound",
			clusterSpec:                              defaultVmClusterReconcile,
			machineSpec:                              defaultVmReconcile,
			expCheckVmStateBootErr:                   nil,
			expUnlinkLoadBalancerBackendMachineErr:   fmt.Errorf("UnlinkLoadBalancerBackendMachineFound generic error"),
			expCheckVmStateLoadBalancerErr:           nil,
			expDeleteSecurityGroupRuleErr:            nil,
			expDeleteSecurityGroupRuleFound:          false,
			expCheckVmStateLoadBalancerFound:         true,
			expUnlinkLoadBalancerBackendMachineFound: true,
			expGetVmFound:                            true,
			expGetVmErr:                              nil,
			expCheckUnlinkPublicIpErr:                nil,
			expReconcileDeleteVmErr:                  fmt.Errorf("UnlinkLoadBalancerBackendMachineFound generic error Can not unlink vm test-loadbalancer with loadBalancerName i-test-vm-uid for OscCluster test-system/test-osc"),
		},
		{
			name:                                     "failed DeleteSecurityGroupRule",
			clusterSpec:                              defaultVmClusterReconcile,
			machineSpec:                              defaultVmReconcile,
			expCheckVmStateBootErr:                   nil,
			expUnlinkLoadBalancerBackendMachineErr:   nil,
			expCheckVmStateLoadBalancerErr:           nil,
			expDeleteSecurityGroupRuleErr:            fmt.Errorf("DeleteSecurityGroupRule generic error"),
			expDeleteSecurityGroupRuleFound:          true,
			expCheckVmStateLoadBalancerFound:         true,
			expUnlinkLoadBalancerBackendMachineFound: true,
			expGetVmFound:                            true,
			expGetVmErr:                              nil,
			expCheckUnlinkPublicIpErr:                nil,
			expReconcileDeleteVmErr:                  fmt.Errorf("DeleteSecurityGroupRule generic error Can not delete outbound securityGroupRule for OscCluster test-system/test-osc"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope, ctx, mockOscVmInterface, _, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface := SetupWithVmMock(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			vmName := vtc.machineSpec.Node.Vm.Name + "-uid"
			vmId := "i-" + vmName
			vmRef := machineScope.GetVmRef()
			vmRef.ResourceMap = make(map[string]string)
			if vtc.expGetVmFound {
				vmRef.ResourceMap[vmName] = vmId
			}
			vmState := "running"

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
			publicIpName := vtc.machineSpec.Node.Vm.PublicIpName + "-uid"
			linkPublicIpId := "eipassoc-" + publicIpName
			linkPublicIpRef := machineScope.GetLinkPublicIpRef()
			linkPublicIpRef.ResourceMap = make(map[string]string)
			linkPublicIpRef.ResourceMap[publicIpName] = linkPublicIpId

			var clockInsideLoop time.Duration = 5
			var clockLoop time.Duration = 60
			var firstClockLoop time.Duration = 120
			loadBalancerName := vtc.machineSpec.Node.Vm.LoadBalancerName
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
				mockOscVmInterface.
					EXPECT().
					GetVm(gomock.Eq(vmId)).
					Return(vm, vtc.expGetVmErr)
			} else {
				mockOscVmInterface.
					EXPECT().
					GetVm(gomock.Eq(vmId)).
					Return(nil, vtc.expGetVmErr)
			}
			mockOscVmInterface.
				EXPECT().
				CheckVmState(gomock.Eq(clockInsideLoop), gomock.Eq(firstClockLoop), gomock.Eq(vmState), gomock.Eq(vmId)).
				Return(vtc.expCheckVmStateBootErr)

			mockOscPublicIpInterface.
				EXPECT().
				UnlinkPublicIp(gomock.Eq(linkPublicIpId)).
				Return(vtc.expCheckUnlinkPublicIpErr)
			vmIds := []string{vmId}
			if vtc.expCheckVmStateLoadBalancerFound {
				mockOscVmInterface.
					EXPECT().
					CheckVmState(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(vmState), gomock.Eq(vmId)).
					Return(vtc.expCheckVmStateLoadBalancerErr)
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

			reconcileDeleteVm, err := reconcileDeleteVm(ctx, clusterScope, machineScope, mockOscVmInterface, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface)
			if err != nil {
				assert.Equal(t, vtc.expReconcileDeleteVmErr.Error(), err.Error(), "reconccileDeleteVm() hould return the same error")
			} else {
				assert.Nil(t, vtc.expReconcileDeleteVmErr)
			}
			t.Logf("find reconcileDeleteVm %v\n", reconcileDeleteVm)
		})
	}

}

// TestReconcileDeleteVmResourceId has several tests to cover the code of the function reconcileDeleteVm
func TestReconcileDeleteVmResourceId(t *testing.T) {
	vmTestCases := []struct {
		name                    string
		clusterSpec             infrastructurev1beta1.OscClusterSpec
		machineSpec             infrastructurev1beta1.OscMachineSpec
		expGetVmFound           bool
		expGetVmErr             error
		expSecurityGroupFound   bool
		expReconcileDeleteVmErr error
	}{
		{
			name:                    "failed to find vm",
			clusterSpec:             defaultVmClusterReconcile,
			machineSpec:             defaultVmReconcile,
			expGetVmFound:           false,
			expGetVmErr:             nil,
			expSecurityGroupFound:   true,
			expReconcileDeleteVmErr: nil,
		},
		{
			name:                    "failed to find security group",
			clusterSpec:             defaultVmClusterReconcile,
			machineSpec:             defaultVmReconcile,
			expGetVmFound:           true,
			expGetVmErr:             nil,
			expSecurityGroupFound:   false,
			expReconcileDeleteVmErr: fmt.Errorf("test-securitygroup-uid does not exist"),
		},
		{
			name:                    "failed to get vm",
			clusterSpec:             defaultVmClusterReconcile,
			machineSpec:             defaultVmReconcile,
			expGetVmFound:           true,
			expGetVmErr:             fmt.Errorf("GetVm generic error"),
			expSecurityGroupFound:   false,
			expReconcileDeleteVmErr: fmt.Errorf("GetVm generic error"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope, ctx, mockOscVmInterface, _, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface := SetupWithVmMock(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			vmName := vtc.machineSpec.Node.Vm.Name + "-uid"
			vmId := "i-" + vmName
			vmRef := machineScope.GetVmRef()
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
				mockOscVmInterface.
					EXPECT().
					GetVm(gomock.Eq(vmId)).
					Return(vm, vtc.expGetVmErr)
			} else {
				mockOscVmInterface.
					EXPECT().
					GetVm(gomock.Eq(vmId)).
					Return(nil, vtc.expGetVmErr)
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
			publicIpName := vtc.machineSpec.Node.Vm.PublicIpName + "-uid"
			linkPublicIpId := "eipassoc-" + publicIpName
			linkPublicIpRef := machineScope.GetLinkPublicIpRef()
			linkPublicIpRef.ResourceMap = make(map[string]string)
			linkPublicIpRef.ResourceMap[publicIpName] = linkPublicIpId

			reconcileDeleteVm, err := reconcileDeleteVm(ctx, clusterScope, machineScope, mockOscVmInterface, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface)
			if err != nil {
				assert.Equal(t, vtc.expReconcileDeleteVmErr, err, "reconccileDeleteVm() hould return the same error")
			} else {
				assert.Nil(t, vtc.expReconcileDeleteVmErr)
			}
			t.Logf("find reconcileDeleteVm %v\n", reconcileDeleteVm)
		})
	}

}

// TestReconcileDeleteVmWithoutSpec has several tests to cover the code of the function reconcileDeleteVm
func TestReconcileDeleteVmWithoutSpec(t *testing.T) {
	vmTestCases := []struct {
		name                                   string
		clusterSpec                            infrastructurev1beta1.OscClusterSpec
		machineSpec                            infrastructurev1beta1.OscMachineSpec
		expCheckVmStateBootErr                 error
		expUnlinkLoadBalancerBackendMachineErr error
		expCheckVmStateLoadBalancerErr         error
		expDeleteSecurityGroupRuleErr          error
		expDeleteVmErr                         error
		expGetVmErr                            error
		expSecurityGroupRuleFound              bool
		expGetVmFound                          bool
		expCheckUnlinkPublicIpErr              error
		expReconcileDeleteVmErr                error
	}{
		{
			name: "delete vm without spec",
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Volumes: []*infrastructurev1beta1.OscVolume{
						{
							ResourceId: "vol-cluster-api-volume-uid",
						},
					},
					Vm: infrastructurev1beta1.OscVm{
						ResourceId: "i-cluster-api-vm-uid",
					},
				},
			},
			expCheckVmStateBootErr:                 nil,
			expUnlinkLoadBalancerBackendMachineErr: nil,
			expCheckVmStateLoadBalancerErr:         nil,
			expDeleteSecurityGroupRuleErr:          nil,
			expSecurityGroupRuleFound:              true,
			expDeleteVmErr:                         nil,
			expGetVmFound:                          true,
			expGetVmErr:                            nil,
			expCheckUnlinkPublicIpErr:              nil,
			expReconcileDeleteVmErr:                nil,
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope, ctx, mockOscVmInterface, _, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface := SetupWithVmMock(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			vmName := "cluster-api-vm-uid"
			vmId := "i-" + vmName
			vmRef := machineScope.GetVmRef()
			vmRef.ResourceMap = make(map[string]string)
			if vtc.expGetVmFound {
				vmRef.ResourceMap[vmName] = vmId
			}

			var securityGroupIds []string
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			securityGroupKwName := "cluster-api-securitygroup-kw-uid"
			securityGroupKwId := "sg-" + securityGroupKwName
			securityGroupsRef.ResourceMap[securityGroupKwName] = securityGroupKwId
			securityGroupNodeName := "cluster-api-securitygroup-node-uid"
			securityGroupNodeId := "sg-" + securityGroupNodeName
			securityGroupsRef.ResourceMap[securityGroupNodeName] = securityGroupNodeId
			securityGroupIds = append(securityGroupIds, securityGroupKwId)
			securityGroupIds = append(securityGroupIds, securityGroupNodeId)
			publicIpName := "cluster-api-publicip-uid"
			linkPublicIpId := "eipassoc-" + publicIpName
			linkPublicIpRef := machineScope.GetLinkPublicIpRef()
			linkPublicIpRef.ResourceMap = make(map[string]string)
			linkPublicIpRef.ResourceMap[publicIpName] = linkPublicIpId

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
				mockOscVmInterface.
					EXPECT().
					GetVm(gomock.Eq(vmId)).
					Return(vm, vtc.expGetVmErr)
			} else {
				mockOscVmInterface.
					EXPECT().
					GetVm(gomock.Eq(vmId)).
					Return(nil, vtc.expGetVmErr)
			}
			mockOscVmInterface.
				EXPECT().
				DeleteVm(gomock.Eq(vmId)).
				Return(vtc.expDeleteVmErr)
			reconcileDeleteVm, err := reconcileDeleteVm(ctx, clusterScope, machineScope, mockOscVmInterface, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface)
			if err != nil {
				assert.Equal(t, vtc.expReconcileDeleteVmErr, err, "reconcileDeleteVm() should return the same error")
			} else {
				assert.Nil(t, vtc.expReconcileDeleteVmErr)
			}

			t.Logf("find reconcileDeleteVm %v\n", reconcileDeleteVm)
		})
	}

}

// TestReconcileDeleteVmSecurityGroup has several tests to cover the code of the function reconcileDeleteVm
func TestReconcileDeleteVmSecurityGroup(t *testing.T) {
	vmTestCases := []struct {
		name                                    string
		clusterSpec                             infrastructurev1beta1.OscClusterSpec
		machineSpec                             infrastructurev1beta1.OscMachineSpec
		expDeleteInboundSecurityGroupRuleFound  bool
		expDeleteOutboundSecurityGroupRuleFound bool
		expCheckVmStateBootErr                  error
		expUnlinkLoadBalancerBackendMachineErr  error
		expCheckVmStateLoadBalancerErr          error
		expDeleteInboundSecurityGroupRuleErr    error
		expDeleteOutboundSecurityGroupRuleErr   error
		expDeleteVmErr                          error
		expGetVmErr                             error
		expSecurityGroupRuleFound               bool
		expGetVmFound                           bool
		expCheckUnlinkPublicIpErr               error
		expReconcileDeleteVmErr                 error
	}{
		{
			name:                                    "failed to delete inbound securitygroup",
			clusterSpec:                             defaultVmClusterReconcile,
			machineSpec:                             defaultVmReconcile,
			expCheckVmStateBootErr:                  nil,
			expDeleteInboundSecurityGroupRuleFound:  true,
			expDeleteOutboundSecurityGroupRuleFound: true,
			expUnlinkLoadBalancerBackendMachineErr:  nil,
			expCheckVmStateLoadBalancerErr:          nil,
			expDeleteInboundSecurityGroupRuleErr:    fmt.Errorf("CreateSecurityGroupRule generic error"),
			expDeleteOutboundSecurityGroupRuleErr:   nil,
			expSecurityGroupRuleFound:               true,
			expDeleteVmErr:                          nil,
			expGetVmFound:                           true,
			expGetVmErr:                             nil,
			expCheckUnlinkPublicIpErr:               nil,
			expReconcileDeleteVmErr:                 fmt.Errorf("CreateSecurityGroupRule generic error Can not delete inbound securityGroupRule for OscCluster test-system/test-osc"),
		},
		{
			name:                                    "failed to delete outbound securitygroup",
			clusterSpec:                             defaultVmClusterReconcile,
			machineSpec:                             defaultVmReconcile,
			expCheckVmStateBootErr:                  nil,
			expDeleteInboundSecurityGroupRuleFound:  false,
			expDeleteOutboundSecurityGroupRuleFound: true,
			expUnlinkLoadBalancerBackendMachineErr:  nil,
			expCheckVmStateLoadBalancerErr:          nil,
			expSecurityGroupRuleFound:               true,
			expDeleteVmErr:                          nil,
			expGetVmFound:                           true,
			expGetVmErr:                             nil,
			expDeleteInboundSecurityGroupRuleErr:    nil,
			expDeleteOutboundSecurityGroupRuleErr:   fmt.Errorf("CreateSecurityGroupRule generic error"),
			expCheckUnlinkPublicIpErr:               nil,
			expReconcileDeleteVmErr:                 fmt.Errorf("CreateSecurityGroupRule generic error Can not delete outbound securityGroupRule for OscCluster test-system/test-osc"),
		},
	}
	for _, vtc := range vmTestCases {
		t.Run(vtc.name, func(t *testing.T) {
			clusterScope, machineScope, ctx, mockOscVmInterface, _, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface := SetupWithVmMock(t, vtc.name, vtc.clusterSpec, vtc.machineSpec)
			vmName := vtc.machineSpec.Node.Vm.Name + "-uid"
			vmId := "i-" + vmName
			vmRef := machineScope.GetVmRef()
			vmRef.ResourceMap = make(map[string]string)
			if vtc.expGetVmFound {
				vmRef.ResourceMap[vmName] = vmId
			}
			vmState := "running"

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
			publicIpName := vtc.machineSpec.Node.Vm.PublicIpName + "-uid"
			linkPublicIpId := "eipassoc-" + publicIpName
			linkPublicIpRef := machineScope.GetLinkPublicIpRef()
			linkPublicIpRef.ResourceMap = make(map[string]string)
			linkPublicIpRef.ResourceMap[publicIpName] = linkPublicIpId

			var clockInsideLoop time.Duration = 5
			var clockLoop time.Duration = 60
			var firstClockLoop time.Duration = 120
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
			vm := &createVm[0]
			if vtc.expGetVmFound {
				mockOscVmInterface.
					EXPECT().
					GetVm(gomock.Eq(vmId)).
					Return(vm, vtc.expGetVmErr)
			} else {
				mockOscVmInterface.
					EXPECT().
					GetVm(gomock.Eq(vmId)).
					Return(nil, vtc.expGetVmErr)
			}
			mockOscVmInterface.
				EXPECT().
				CheckVmState(gomock.Eq(clockInsideLoop), gomock.Eq(firstClockLoop), gomock.Eq(vmState), gomock.Eq(vmId)).
				Return(vtc.expCheckVmStateBootErr)

			mockOscVmInterface.
				EXPECT().
				CheckVmState(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(vmState), gomock.Eq(vmId)).
				Return(vtc.expCheckVmStateLoadBalancerErr)

			mockOscPublicIpInterface.
				EXPECT().
				UnlinkPublicIp(gomock.Eq(linkPublicIpId)).
				Return(vtc.expCheckUnlinkPublicIpErr)
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

			reconcileDeleteVm, err := reconcileDeleteVm(ctx, clusterScope, machineScope, mockOscVmInterface, mockOscPublicIpInterface, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface)
			if err != nil {
				assert.Equal(t, vtc.expReconcileDeleteVmErr.Error(), err.Error(), "reconcileDeleteVm() should return the same error")
			} else {
				assert.Nil(t, vtc.expReconcileDeleteVmErr)
			}
			t.Logf("find reconcileDeleteVm %v\n", reconcileDeleteVm)
		})
	}

}
