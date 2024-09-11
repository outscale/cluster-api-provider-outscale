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
	"time"

	"github.com/golang/mock/gomock"
	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/compute/mock_compute"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/security/mock_security"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag/mock_tag"
	osc "github.com/outscale/osc-sdk-go/v2"
	"github.com/stretchr/testify/assert"
)

var (
	defaultBastionInitialize = infrastructurev1beta1.OscClusterSpec{
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
			Bastion: infrastructurev1beta1.OscBastion{
				Enable:      true,
				ClusterName: "test-cluster",
				Name:        "test-bastion",
				ImageId:     "ami-00000000",
				DeviceName:  "/dev/xvdb",
				KeypairName: "rke",
				RootDisk: infrastructurev1beta1.OscRootDisk{

					RootDiskSize: 30,
					RootDiskIops: 1500,
					RootDiskType: "io1",
				},
				SubregionName: "eu-west-2a",
				SubnetName:    "test-subnet",
				VmType:        "tinav3.c2r4p2",
				PublicIpName:  "test-publicip",
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

	defaultBastionImageInitialize = infrastructurev1beta1.OscClusterSpec{
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
			Bastion: infrastructurev1beta1.OscBastion{
				Enable:      true,
				ClusterName: "test-cluster",
				Name:        "test-bastion",
				ImageId:     "ami-00000000",
				ImageName:   "ubuntu-2004-2004-kubernetes-v1.22.11-2022-11-23",
				DeviceName:  "/dev/xvdb",
				KeypairName: "rke",
				RootDisk: infrastructurev1beta1.OscRootDisk{

					RootDiskSize: 30,
					RootDiskIops: 1500,
					RootDiskType: "io1",
				},
				SubregionName: "eu-west-2a",
				SubnetName:    "test-subnet",
				VmType:        "tinav3.c2r4p2",
				PublicIpName:  "test-publicip",
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

	defaultPublicIpNameAfterBastionReconcile = infrastructurev1beta1.OscClusterSpec{
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
			Bastion: infrastructurev1beta1.OscBastion{
				Enable:      true,
				ClusterName: "test-cluster",
				Name:        "test-bastion",
				ImageId:     "ami-00000000",
				DeviceName:  "/dev/xvdb",
				KeypairName: "rke",
				RootDisk: infrastructurev1beta1.OscRootDisk{

					RootDiskSize: 30,
					RootDiskIops: 1500,
					RootDiskType: "io1",
				},
				SubregionName:            "eu-west-2a",
				SubnetName:               "test-subnet",
				VmType:                   "tinav3.c2r4p2",
				ResourceId:               "i-test-bastion-uid",
				PublicIpName:             "test-publicip",
				PublicIpNameAfterBastion: true,
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

	defaultBastionReconcile = infrastructurev1beta1.OscClusterSpec{
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
			Bastion: infrastructurev1beta1.OscBastion{
				Enable:      true,
				ClusterName: "test-cluster",
				Name:        "test-bastion",
				ImageId:     "ami-00000000",
				DeviceName:  "/dev/xvdb",
				KeypairName: "rke",
				RootDisk: infrastructurev1beta1.OscRootDisk{

					RootDiskSize: 30,
					RootDiskIops: 1500,
					RootDiskType: "io1",
				},
				SubregionName: "eu-west-2a",
				SubnetName:    "test-subnet",
				VmType:        "tinav3.c2r4p2",
				ResourceId:    "i-test-bastion-uid",
				PublicIpName:  "test-publicip",
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

// SetupWithBastionMock set vmMock with clusterScope, machineScope and oscMachine
func SetupWithBastionMock(t *testing.T, name string, clusterSpec infrastructurev1beta1.OscClusterSpec) (clusterScope *scope.ClusterScope, ctx context.Context, mockOscVmInterface *mock_compute.MockOscVmInterface, mockOscPublicIpInterface *mock_security.MockOscPublicIpInterface, mockOscSecurityGroupInterface *mock_security.MockOscSecurityGroupInterface, mockOscImageInterface *mock_compute.MockOscImageInterface, mockOscTagInterface *mock_tag.MockOscTagInterface) {
	clusterScope = Setup(t, name, clusterSpec)
	mockCtrl := gomock.NewController(t)
	mockOscVmInterface = mock_compute.NewMockOscVmInterface(mockCtrl)
	mockOscPublicIpInterface = mock_security.NewMockOscPublicIpInterface(mockCtrl)
	mockOscSecurityGroupInterface = mock_security.NewMockOscSecurityGroupInterface(mockCtrl)
	mockOscImageInterface = mock_compute.NewMockOscImageInterface(mockCtrl)
	mockOscTagInterface = mock_tag.NewMockOscTagInterface(mockCtrl)
	ctx = context.Background()
	return clusterScope, ctx, mockOscVmInterface, mockOscPublicIpInterface, mockOscSecurityGroupInterface, mockOscImageInterface, mockOscTagInterface
}

// TestGetBastionResourceId has several tests to cover the code of the function getBastionResourceId
func TestGetBastionResourceId(t *testing.T) {
	bastionTestCases := []struct {
		name                       string
		clusterSpec                infrastructurev1beta1.OscClusterSpec
		expBastionFound            bool
		expGetBastionResourceIdErr error
	}{
		{
			name:                       "get bastion",
			clusterSpec:                defaultBastionInitialize,
			expBastionFound:            true,
			expGetBastionResourceIdErr: nil,
		},
		{
			name:                       "can not get bastion",
			clusterSpec:                defaultBastionInitialize,
			expBastionFound:            false,
			expGetBastionResourceIdErr: fmt.Errorf("test-bastion-uid does not exist"),
		},
	}
	for _, btc := range bastionTestCases {
		t.Run(btc.name, func(t *testing.T) {
			clusterScope := Setup(t, btc.name, btc.clusterSpec)
			bastionName := btc.clusterSpec.Network.Bastion.Name + "-uid"
			vmId := "i-" + bastionName
			bastionRef := clusterScope.GetBastionRef()
			bastionRef.ResourceMap = make(map[string]string)
			if btc.expBastionFound {
				bastionRef.ResourceMap[bastionName] = vmId
			}
			bastionResourceId, err := getBastionResourceId(bastionName, clusterScope)
			if err != nil {
				assert.Equal(t, btc.expGetBastionResourceIdErr.Error(), err.Error(), "GetBastionResourceId() should return the same error")
			} else {
				assert.Nil(t, btc.expGetBastionResourceIdErr)
			}
			t.Logf("find bastionResourceId %s", bastionResourceId)
		})
	}
}

// TestCheckBastionSecurityGroupOscAssociateResourceName has several tests to cover the code of the function checkBastionSecurityGroupOscAssociateResourceName
func TestCheckBastionSecurityGroupOscAssociateResourceName(t *testing.T) {
	bastionTestCases := []struct {
		name                                                    string
		clusterSpec                                             infrastructurev1beta1.OscClusterSpec
		expCheckBastionSecurityGroupOscAssociateResourceNameErr error
	}{
		{
			name:        "check securitygrup associate with vm",
			clusterSpec: defaultBastionInitialize,
			expCheckBastionSecurityGroupOscAssociateResourceNameErr: nil,
		},
		{
			name: "check work with bastion spec (with default value)",
			clusterSpec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Bastion: infrastructurev1beta1.OscBastion{
						Enable: true,
					},
				},
			},
			expCheckBastionSecurityGroupOscAssociateResourceNameErr: fmt.Errorf("cluster-api-securitygroup-lb-uid securityGroup does not exist in bastion"),
		},
		{
			name: "check Bad security group name",
			clusterSpec: infrastructurev1beta1.OscClusterSpec{
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
					Bastion: infrastructurev1beta1.OscBastion{
						Enable:      true,
						ClusterName: "test-cluster",
						Name:        "test-bastion",
						ImageId:     "ami-00000000",
						DeviceName:  "/dev/xvdb",
						KeypairName: "rke",
						RootDisk: infrastructurev1beta1.OscRootDisk{

							RootDiskSize: 30,
							RootDiskIops: 1500,
							RootDiskType: "io1",
						},
						SubregionName: "eu-west-2a",
						SubnetName:    "test-subnet",
						VmType:        "tinav3.c2r4p2",
						PublicIpName:  "test-publicip",
						SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
							{
								Name: "test-securitygroup@test",
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
			},
			expCheckBastionSecurityGroupOscAssociateResourceNameErr: fmt.Errorf("test-securitygroup@test-uid securityGroup does not exist in bastion"),
		},
	}
	for _, btc := range bastionTestCases {
		t.Run(btc.name, func(t *testing.T) {
			clusterScope := Setup(t, btc.name, btc.clusterSpec)
			err := checkBastionSecurityGroupOscAssociateResourceName(clusterScope)
			if err != nil {
				assert.Equal(t, btc.expCheckBastionSecurityGroupOscAssociateResourceNameErr, err, "checkBastionSecurityGroupOscAssociateResourceName() should return the same error")
			} else {
				assert.Nil(t, btc.expCheckBastionSecurityGroupOscAssociateResourceNameErr)
			}
		})
	}
}

// TestCheckBastionSubnetAssociateResourceName has several tests to cover the code of the function checkBastionSubnetAssociateResourceName
func TestCheckBastionSubnetAssociateResourceName(t *testing.T) {
	bastionTestCases := []struct {
		name                                          string
		clusterSpec                                   infrastructurev1beta1.OscClusterSpec
		expCheckBastionSubnetAssociateResourceNameErr error
	}{
		{
			name:        "check subnet associate with bastion",
			clusterSpec: defaultBastionInitialize,
			expCheckBastionSubnetAssociateResourceNameErr: nil,
		},
		{
			name: "check work with bastion spec (with default values)",
			clusterSpec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Bastion: infrastructurev1beta1.OscBastion{
						Enable: true,
					},
				},
			},
			expCheckBastionSubnetAssociateResourceNameErr: fmt.Errorf("cluster-api-subnet-public-uid subnet does not exist in bastion"),
		},
		{
			name: "check Bad subnet name",
			clusterSpec: infrastructurev1beta1.OscClusterSpec{
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
					Bastion: infrastructurev1beta1.OscBastion{
						Enable:      true,
						ClusterName: "test-cluster",
						Name:        "test-bastion",
						ImageId:     "ami-00000000",
						DeviceName:  "/dev/xvdb",
						KeypairName: "rke",
						RootDisk: infrastructurev1beta1.OscRootDisk{

							RootDiskSize: 30,
							RootDiskIops: 1500,
							RootDiskType: "io1",
						},
						SubregionName: "eu-west-2a",
						SubnetName:    "test-subnet@test",
						VmType:        "tinav3.c2r4p2",
						PublicIpName:  "test-publicip",
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
			},
			expCheckBastionSubnetAssociateResourceNameErr: fmt.Errorf("test-subnet@test-uid subnet does not exist in bastion"),
		},
	}
	for _, btc := range bastionTestCases {
		t.Run(btc.name, func(t *testing.T) {
			clusterScope := Setup(t, btc.name, btc.clusterSpec)
			err := checkBastionSubnetOscAssociateResourceName(clusterScope)
			if err != nil {
				assert.Equal(t, btc.expCheckBastionSubnetAssociateResourceNameErr, err, "checkBastionSubnetOscAssociateResourceName(() should return the same error")
			} else {
				assert.Nil(t, btc.expCheckBastionSubnetAssociateResourceNameErr)
			}
		})
	}
}

// TestCheckBastionPublicIpOscAssociateResourceName has several tests to cover the code of the function checkVmPublicIpOscAssociateResourceName
func TestCheckBastionPublicIpOscAssociateResourceName(t *testing.T) {
	bastionTestCases := []struct {
		name                                               string
		clusterSpec                                        infrastructurev1beta1.OscClusterSpec
		expCheckBastionPublicIpOscAssociateResourceNameErr error
	}{
		{
			name:        "check publicip association with bastion",
			clusterSpec: defaultBastionInitialize,
			expCheckBastionPublicIpOscAssociateResourceNameErr: nil,
		},
		{
			name: "check work with bastion spec (with default values)",
			clusterSpec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Bastion: infrastructurev1beta1.OscBastion{
						Enable:       true,
						PublicIpName: "cluster-api-publicip",
					},
				},
			},
			expCheckBastionPublicIpOscAssociateResourceNameErr: fmt.Errorf("cluster-api-publicip-uid publicIp does not exist in bastion"),
		},
		{
			name: "check Bad PublicIp  name",
			clusterSpec: infrastructurev1beta1.OscClusterSpec{
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
					Bastion: infrastructurev1beta1.OscBastion{
						Enable:      true,
						ClusterName: "test-cluster",
						Name:        "test-bastion",
						ImageId:     "ami-00000000",
						DeviceName:  "/dev/xvdb",
						KeypairName: "rke",
						RootDisk: infrastructurev1beta1.OscRootDisk{

							RootDiskSize: 30,
							RootDiskIops: 1500,
							RootDiskType: "io1",
						},
						SubregionName: "eu-west-2a",
						SubnetName:    "test-subnet",
						VmType:        "tinav3.c2r4p2",
						PublicIpName:  "test-publicip@test",
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
			},
			expCheckBastionPublicIpOscAssociateResourceNameErr: fmt.Errorf("test-publicip@test-uid publicIp does not exist in bastion"),
		},
	}
	for _, btc := range bastionTestCases {
		t.Run(btc.name, func(t *testing.T) {
			clusterScope := Setup(t, btc.name, btc.clusterSpec)
			err := checkBastionPublicIpOscAssociateResourceName(clusterScope)
			if err != nil {
				assert.Equal(t, btc.expCheckBastionPublicIpOscAssociateResourceNameErr, err, "checkBastionPublicIpOscAssociateResourceName() should return the same error")
			} else {
				assert.Nil(t, btc.expCheckBastionPublicIpOscAssociateResourceNameErr)
			}
		})
	}
}

// TestCheckBastionFormatParameters has several tests to cover the code of the function checkBastionFormatParamter
func TestCheckBastionFormatParameters(t *testing.T) {
	bastionTestCases := []struct {
		name                               string
		clusterSpec                        infrastructurev1beta1.OscClusterSpec
		expCheckBastionFormatParametersErr error
	}{
		{
			name:                               "check bastion format",
			clusterSpec:                        defaultBastionInitialize,
			expCheckBastionFormatParametersErr: nil,
		},
		{
			name: "check Bad name vm",
			clusterSpec: infrastructurev1beta1.OscClusterSpec{
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
					Bastion: infrastructurev1beta1.OscBastion{
						Enable:      true,
						ClusterName: "test-cluster",
						Name:        "test-bastion@test",
						ImageId:     "ami-00000000",
						DeviceName:  "/dev/xvdb",
						KeypairName: "rke",
						RootDisk: infrastructurev1beta1.OscRootDisk{

							RootDiskSize: 30,
							RootDiskIops: 1500,
							RootDiskType: "io1",
						},
						SubregionName: "eu-west-2a",
						SubnetName:    "test-subnet",
						VmType:        "tinav3.c2r4p2",
						PublicIpName:  "test-publicip@test",
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
			},
			expCheckBastionFormatParametersErr: fmt.Errorf("Invalid Tag Name"),
		},

		{
			name: "check Bad imageId",
			clusterSpec: infrastructurev1beta1.OscClusterSpec{
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
					Bastion: infrastructurev1beta1.OscBastion{
						Enable:      true,
						ClusterName: "test-cluster",
						Name:        "test-bastion",
						ImageId:     "omi-00000000",
						DeviceName:  "/dev/xvdb",
						KeypairName: "rke",
						RootDisk: infrastructurev1beta1.OscRootDisk{

							RootDiskSize: 30,
							RootDiskIops: 1500,
							RootDiskType: "io1",
						},
						SubregionName: "eu-west-2a",
						SubnetName:    "test-subnet",
						VmType:        "tinav3.c2r4p2",
						PublicIpName:  "test-publicip@test",
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
			},
			expCheckBastionFormatParametersErr: fmt.Errorf("Invalid imageId"),
		},
		{
			name: "check empty imageId and imagename",
			clusterSpec: infrastructurev1beta1.OscClusterSpec{
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
					Bastion: infrastructurev1beta1.OscBastion{
						Enable:      true,
						ClusterName: "test-cluster",
						Name:        "test-bastion",
						ImageId:     "ami-00000000",
						DeviceName:  "/dev/xvdb",
						KeypairName: "rke",
						RootDisk: infrastructurev1beta1.OscRootDisk{

							RootDiskSize: 30,
							RootDiskIops: 1500,
							RootDiskType: "io1",
						},
						SubregionName: "eu-west-2a",
						SubnetName:    "test-subnet",
						VmType:        "tinav3.c2r4p2",
						PublicIpName:  "test-publicip@test",
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
			},
			expCheckBastionFormatParametersErr: nil,
		},
		{
			name: "check Bad ImageName",
			clusterSpec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Bastion: infrastructurev1beta1.OscBastion{
						Enable:    true,
						ImageName: "!test-image@Name",
					},
				},
			},
			expCheckBastionFormatParametersErr: fmt.Errorf("Invalid Image Name"),
		},
		{
			name: "check Bad keypairname",
			clusterSpec: infrastructurev1beta1.OscClusterSpec{
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
					Bastion: infrastructurev1beta1.OscBastion{
						Enable:      true,
						ClusterName: "test-cluster",
						Name:        "test-bastion",
						ImageId:     "ami-00000000",
						DeviceName:  "/dev/xvdb",
						KeypairName: "rke λ",
						RootDisk: infrastructurev1beta1.OscRootDisk{

							RootDiskSize: 30,
							RootDiskIops: 1500,
							RootDiskType: "io1",
						},
						SubregionName: "eu-west-2a",
						SubnetName:    "test-subnet",
						VmType:        "tinav3.c2r4p2",
						PublicIpName:  "test-publicip@test",
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
			},
			expCheckBastionFormatParametersErr: fmt.Errorf("Invalid KeypairName"),
		},
		{
			name: "check empty imageId",
			clusterSpec: infrastructurev1beta1.OscClusterSpec{
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
					Image: infrastructurev1beta1.OscImage{
						Name: "omi-000",
					},
					Bastion: infrastructurev1beta1.OscBastion{
						Enable:      true,
						ClusterName: "test-cluster",
						Name:        "test-bastion",
						DeviceName:  "/dev/xvdb",
						KeypairName: "rke",
						RootDisk: infrastructurev1beta1.OscRootDisk{

							RootDiskSize: 30,
							RootDiskIops: 1500,
							RootDiskType: "io1",
						},
						SubregionName: "eu-west-2a",
						SubnetName:    "test-subnet",
						VmType:        "tinav3.c2r4p2",
						PublicIpName:  "test-publicip@test",
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
			},
			expCheckBastionFormatParametersErr: nil,
		},
		{
			name: "check Bad device name",
			clusterSpec: infrastructurev1beta1.OscClusterSpec{
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
					Bastion: infrastructurev1beta1.OscBastion{
						Enable:      true,
						ClusterName: "test-cluster",
						Name:        "test-bastion",
						ImageId:     "ami-00000000",
						DeviceName:  "/dev/sdab1",
						KeypairName: "rke",
						RootDisk: infrastructurev1beta1.OscRootDisk{

							RootDiskSize: 30,
							RootDiskIops: 1500,
							RootDiskType: "io1",
						},
						SubregionName: "eu-west-2a",
						SubnetName:    "test-subnet",
						VmType:        "tinav3.c2r4p2",
						PublicIpName:  "test-publicip@test",
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
			},
			expCheckBastionFormatParametersErr: fmt.Errorf("Invalid deviceName"),
		},
		{
			name: "check empty device name",
			clusterSpec: infrastructurev1beta1.OscClusterSpec{
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
					Bastion: infrastructurev1beta1.OscBastion{
						Enable:      true,
						ClusterName: "test-cluster",
						Name:        "test-bastion",
						ImageId:     "ami-00000000",
						KeypairName: "rke",
						RootDisk: infrastructurev1beta1.OscRootDisk{

							RootDiskSize: 30,
							RootDiskIops: 1500,
							RootDiskType: "io1",
						},
						SubregionName: "eu-west-2a",
						SubnetName:    "test-subnet",
						VmType:        "tinav3.c2r4p2",
						PublicIpName:  "test-publicip@test",
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
			},
			expCheckBastionFormatParametersErr: nil,
		},
		{
			name: "Check Bad VmType",
			clusterSpec: infrastructurev1beta1.OscClusterSpec{
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
					Bastion: infrastructurev1beta1.OscBastion{
						Enable:      true,
						ClusterName: "test-cluster",
						Name:        "test-bastion",
						ImageId:     "ami-00000000",
						DeviceName:  "/dev/xvdb",
						KeypairName: "rke",
						RootDisk: infrastructurev1beta1.OscRootDisk{

							RootDiskSize: 30,
							RootDiskIops: 1500,
							RootDiskType: "io1",
						},
						SubregionName: "eu-west-2a",
						SubnetName:    "test-subnet",
						VmType:        "awsv4.c2r4p2",
						PublicIpName:  "test-publicip@test",
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
			},
			expCheckBastionFormatParametersErr: fmt.Errorf("Invalid vmType"),
		},
		{
			name: "Check Bad IpAddr",
			clusterSpec: infrastructurev1beta1.OscClusterSpec{
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
					Bastion: infrastructurev1beta1.OscBastion{
						Enable:      true,
						ClusterName: "test-cluster",
						Name:        "test-bastion",
						ImageId:     "ami-00000000",
						DeviceName:  "/dev/xvdb",
						KeypairName: "rke",
						RootDisk: infrastructurev1beta1.OscRootDisk{

							RootDiskSize: 30,
							RootDiskIops: 1500,
							RootDiskType: "io1",
						},
						SubregionName: "eu-west-2a",
						SubnetName:    "test-subnet",
						VmType:        "tinav3.c2r4p2",
						PublicIpName:  "test-publicip@test",
						SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
							{
								Name: "test-securitygroup",
							},
						},
						PrivateIps: []infrastructurev1beta1.OscPrivateIpElement{
							{
								Name:      "test-privateip",
								PrivateIp: "10.245.0.17",
							},
						},
					},
				},
			},
			expCheckBastionFormatParametersErr: fmt.Errorf("Invalid ip in cidr"),
		},
		{
			name: "Check Bad subregionname",
			clusterSpec: infrastructurev1beta1.OscClusterSpec{
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
					Bastion: infrastructurev1beta1.OscBastion{
						Enable:      true,
						ClusterName: "test-cluster",
						Name:        "test-bastion",
						ImageId:     "ami-00000000",
						DeviceName:  "/dev/xvdb",
						KeypairName: "rke",
						RootDisk: infrastructurev1beta1.OscRootDisk{

							RootDiskSize: 30,
							RootDiskIops: 1500,
							RootDiskType: "io1",
						},
						SubregionName: "eu-west-2c",
						SubnetName:    "test-subnet",
						VmType:        "tinav3.c2r4p2",
						PublicIpName:  "test-publicip@test",
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
			},
			expCheckBastionFormatParametersErr: fmt.Errorf("Invalid subregionName"),
		},
		{
			name: "Check Bad root device size",
			clusterSpec: infrastructurev1beta1.OscClusterSpec{
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
					Bastion: infrastructurev1beta1.OscBastion{
						Enable:      true,
						ClusterName: "test-cluster",
						Name:        "test-bastion",
						ImageId:     "ami-00000000",
						DeviceName:  "/dev/xvdb",
						KeypairName: "rke",
						RootDisk: infrastructurev1beta1.OscRootDisk{

							RootDiskSize: -30,
							RootDiskIops: 1500,
							RootDiskType: "io1",
						},
						SubregionName: "eu-west-2a",
						SubnetName:    "test-subnet",
						VmType:        "tinav3.c2r4p2",
						PublicIpName:  "test-publicip@test",
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
			},
			expCheckBastionFormatParametersErr: fmt.Errorf("Invalid size"),
		},
		{
			name: "Check Bad rootDeviceIops",
			clusterSpec: infrastructurev1beta1.OscClusterSpec{
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
					Bastion: infrastructurev1beta1.OscBastion{
						Enable:      true,
						ClusterName: "test-cluster",
						Name:        "test-bastion",
						ImageId:     "ami-00000000",
						DeviceName:  "/dev/xvdb",
						KeypairName: "rke",
						RootDisk: infrastructurev1beta1.OscRootDisk{

							RootDiskSize: 30,
							RootDiskIops: -15,
							RootDiskType: "io1",
						},
						SubregionName: "eu-west-2a",
						SubnetName:    "test-subnet",
						VmType:        "tinav3.c2r4p2",
						PublicIpName:  "test-publicip@test",
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
			},
			expCheckBastionFormatParametersErr: fmt.Errorf("Invalid iops"),
		},
		{
			name: "Check bad rootDiskType",
			clusterSpec: infrastructurev1beta1.OscClusterSpec{
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
					Bastion: infrastructurev1beta1.OscBastion{
						Enable:      true,
						ClusterName: "test-cluster",
						Name:        "test-bastion",
						ImageId:     "ami-00000000",
						DeviceName:  "/dev/xvdb",
						KeypairName: "rke",
						RootDisk: infrastructurev1beta1.OscRootDisk{

							RootDiskSize: 30,
							RootDiskIops: 1500,
							RootDiskType: "gp3",
						},
						SubregionName: "eu-west-2a",
						SubnetName:    "test-subnet",
						VmType:        "tinav3.c2r4p2",
						PublicIpName:  "test-publicip@test",
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
			},
			expCheckBastionFormatParametersErr: fmt.Errorf("Invalid volumeType"),
		},
	}
	for _, btc := range bastionTestCases {
		t.Run(btc.name, func(t *testing.T) {
			clusterScope := Setup(t, btc.name, btc.clusterSpec)
			subnetName := btc.clusterSpec.Network.Bastion.SubnetName + "-uid"
			subnetId := "subnet-" + subnetName
			subnetRef := clusterScope.GetSubnetRef()
			subnetRef.ResourceMap = make(map[string]string)
			subnetRef.ResourceMap[subnetName] = subnetId
			bastionName, err := checkBastionFormatParameters(clusterScope)
			if err != nil {
				assert.Equal(t, btc.expCheckBastionFormatParametersErr, err, "checkBastionFormatParameters() should return the same error")
			} else {
				assert.Nil(t, btc.expCheckBastionFormatParametersErr)
			}
			t.Logf("find vmName %s\n", bastionName)
		})
	}
}

// TestReconcileBastion has serveral tests to cover the code of function reconcileBastion
func TestReconcileBastion(t *testing.T) {
	bastionTestCases := []struct {
		name                         string
		clusterSpec                  infrastructurev1beta1.OscClusterSpec
		expCreateVmFound             bool
		expLinkPublicIpFound         bool
		expCheckVmStateBootFound     bool
		expCheckVmStatePublicIpFound bool
		expTagFound                  bool
		expCheckVmStateBootErr       error
		expCheckVmStatePublicIpErr   error
		expCreateVmErr               error
		expReadTagErr                error
		expReconcileBastionErr       error
		expLinkPublicIpErr           error
	}{
		{
			name:                         "create bastion (first time reconcile loop)",
			clusterSpec:                  defaultBastionInitialize,
			expCreateVmFound:             true,
			expLinkPublicIpFound:         true,
			expCheckVmStateBootFound:     true,
			expCheckVmStatePublicIpFound: true,
			expTagFound:                  false,
			expCheckVmStateBootErr:       nil,
			expCheckVmStatePublicIpErr:   nil,
			expCreateVmErr:               nil,
			expLinkPublicIpErr:           nil,
			expReadTagErr:                nil,
			expReconcileBastionErr:       nil,
		},
		{
			name:                         "failed checkVmStateBoot",
			clusterSpec:                  defaultBastionInitialize,
			expCreateVmFound:             true,
			expLinkPublicIpFound:         false,
			expCheckVmStateBootFound:     true,
			expCheckVmStatePublicIpFound: false,
			expTagFound:                  false,
			expCheckVmStateBootErr:       fmt.Errorf("CheckVmStateBoot generic error"),
			expCheckVmStatePublicIpErr:   nil,
			expCreateVmErr:               nil,
			expLinkPublicIpErr:           nil,
			expReadTagErr:                nil,
			expReconcileBastionErr:       fmt.Errorf("CheckVmStateBoot generic error Can not get vm i-test-bastion-uid running for OscCluster test-system/test-osc"),
		},
		{
			name:                         "failed checkVmStatePublicIp",
			clusterSpec:                  defaultBastionInitialize,
			expCreateVmFound:             true,
			expLinkPublicIpFound:         true,
			expCheckVmStateBootFound:     true,
			expCheckVmStatePublicIpFound: true,
			expTagFound:                  false,
			expCheckVmStateBootErr:       nil,
			expCheckVmStatePublicIpErr:   fmt.Errorf("CheckVmStatePublicIp generic error"),
			expCreateVmErr:               nil,
			expLinkPublicIpErr:           nil,
			expReadTagErr:                nil,
			expReconcileBastionErr:       fmt.Errorf("CheckVmStatePublicIp generic error Can not get vm i-test-bastion-uid running for OscCluster test-system/test-osc"),
		},
	}
	for _, btc := range bastionTestCases {
		t.Run(btc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscVmInterface, mockOscPublicIpInterface, mockOscSecurityGroupInterface, mockOscImageInterface, mockOscTagInterface := SetupWithBastionMock(t, btc.name, btc.clusterSpec)
			bastionName := btc.clusterSpec.Network.Bastion.Name + "-uid"
			vmId := "i-" + bastionName
			vmState := "running"

			subnetName := btc.clusterSpec.Network.Bastion.SubnetName + "-uid"
			subnetId := "subnet-" + subnetName
			subnetRef := clusterScope.GetSubnetRef()
			subnetRef.ResourceMap = make(map[string]string)
			subnetRef.ResourceMap[subnetName] = subnetId

			publicIpName := btc.clusterSpec.Network.Bastion.PublicIpName + "-uid"
			publicIpId := "eipalloc-" + publicIpName
			publicIpRef := clusterScope.GetPublicIpRef()
			publicIpRef.ResourceMap = make(map[string]string)
			publicIpRef.ResourceMap[publicIpName] = publicIpId

			linkPublicIpId := "eipassoc-" + publicIpName
			linkPublicIpRef := clusterScope.GetLinkPublicIpRef()
			linkPublicIpRef.ResourceMap = make(map[string]string)

			if btc.expLinkPublicIpFound {
				linkPublicIpRef.ResourceMap[publicIpName] = linkPublicIpId
			}

			imageId := btc.clusterSpec.Network.Bastion.ImageId
			var privateIps []string
			bastionPrivateIps := clusterScope.GetBastionPrivateIps()
			for _, bastionPrivateIp := range *bastionPrivateIps {
				privateIp := bastionPrivateIp.PrivateIp
				privateIps = append(privateIps, privateIp)
			}

			var securityGroupIds []string
			bastionSecurityGroups := clusterScope.GetBastionSecurityGroups()
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			for _, bastionSecurityGroup := range *bastionSecurityGroups {
				securityGroupName := bastionSecurityGroup.Name + "-uid"
				securityGroupId := "sg-" + securityGroupName
				securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
				securityGroupIds = append(securityGroupIds, securityGroupId)
			}

			bastionSpec := btc.clusterSpec.Network.Bastion
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
			tag := osc.Tag{
				ResourceId: &vmId,
			}

			if btc.expTagFound {
				mockOscTagInterface.
					EXPECT().
					ReadTag(gomock.Eq("Name"), gomock.Eq(bastionName)).
					Return(&tag, btc.expReadTagErr)
			} else {
				mockOscTagInterface.
					EXPECT().
					ReadTag(gomock.Eq("Name"), gomock.Eq(bastionName)).
					Return(nil, btc.expReadTagErr)
			}
			linkPublicIp := osc.LinkPublicIpResponse{
				LinkPublicIpId: &linkPublicIpId,
			}
			bastion := &createVm[0]
			if btc.expCreateVmFound {
				mockOscVmInterface.
					EXPECT().
					CreateVmUserData(gomock.Eq(""), gomock.Eq(&bastionSpec), gomock.Eq(subnetId), gomock.Eq(securityGroupIds), gomock.Eq(privateIps), gomock.Eq(bastionName), gomock.Eq(imageId)).
					Return(bastion, btc.expCreateVmErr)
			} else {
				mockOscVmInterface.
					EXPECT().
					CreateVmUserData(gomock.Eq(""), gomock.Eq(&bastionSpec), gomock.Eq(subnetId), gomock.Eq(securityGroupIds), gomock.Eq(privateIps), gomock.Eq(bastionName), gomock.Eq(imageId)).
					Return(nil, btc.expCreateVmErr)
			}
			if btc.expCheckVmStateBootFound {
				mockOscVmInterface.
					EXPECT().
					CheckVmState(gomock.Eq(clockInsideLoop), gomock.Eq(firstClockLoop), gomock.Eq(vmState), gomock.Eq(vmId)).
					Return(btc.expCheckVmStateBootErr)
			}

			if btc.expLinkPublicIpFound {
				mockOscPublicIpInterface.
					EXPECT().
					LinkPublicIp(gomock.Eq(publicIpId), gomock.Eq(vmId)).
					Return(*linkPublicIp.LinkPublicIpId, btc.expLinkPublicIpErr)
			}
			if btc.expCheckVmStatePublicIpFound {
				mockOscVmInterface.
					EXPECT().
					CheckVmState(gomock.Eq(clockInsideLoop), gomock.Eq(firstClockLoop), gomock.Eq(vmState), gomock.Eq(vmId)).
					Return(btc.expCheckVmStatePublicIpErr)
			}

			reconcileBastion, err := reconcileBastion(ctx, clusterScope, mockOscVmInterface, mockOscPublicIpInterface, mockOscSecurityGroupInterface, mockOscImageInterface, mockOscTagInterface)
			if err != nil {
				assert.Equal(t, btc.expReconcileBastionErr.Error(), err.Error(), "reconcileBastion() should return the same error")
			} else {
				assert.Nil(t, btc.expReconcileBastionErr)
			}
			t.Logf("find reconcileBastion %v\n", reconcileBastion)
		})
	}
}

// TestReconcileCreateBastion has serveral tests to cover the code of function reconcileBastion
func TestReconcileCreateBastion(t *testing.T) {
	bastionTestCases := []struct {
		name                   string
		clusterSpec            infrastructurev1beta1.OscClusterSpec
		expCreateVmFound       bool
		expLinkPublicIpFound   bool
		expTagFound            bool
		expCreateVmErr         error
		expReadTagErr          error
		expReconcileBastionErr error
	}{
		{
			name:                   "failed to create vm",
			clusterSpec:            defaultBastionInitialize,
			expCreateVmFound:       false,
			expCreateVmErr:         fmt.Errorf("CreateVmUserData generic error"),
			expTagFound:            false,
			expLinkPublicIpFound:   true,
			expReadTagErr:          nil,
			expReconcileBastionErr: fmt.Errorf("CreateVmUserData generic error Can not create bastion for OscCluster test-system/test-osc"),
		},
	}
	for _, btc := range bastionTestCases {
		t.Run(btc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscVmInterface, mockOscPublicIpInterface, mockOscSecurityGroupInterface, mockOscImageInterface, mockOscTagInterface := SetupWithBastionMock(t, btc.name, btc.clusterSpec)
			bastionName := btc.clusterSpec.Network.Bastion.Name + "-uid"
			vmId := "i-" + bastionName

			subnetName := btc.clusterSpec.Network.Bastion.SubnetName + "-uid"
			subnetId := "subnet-" + subnetName
			subnetRef := clusterScope.GetSubnetRef()
			subnetRef.ResourceMap = make(map[string]string)
			subnetRef.ResourceMap[subnetName] = subnetId

			publicIpName := btc.clusterSpec.Network.Bastion.PublicIpName + "-uid"
			publicIpId := "eipalloc-" + publicIpName
			publicIpRef := clusterScope.GetPublicIpRef()
			publicIpRef.ResourceMap = make(map[string]string)
			publicIpRef.ResourceMap[publicIpName] = publicIpId

			linkPublicIpId := "eipassoc-" + publicIpName
			linkPublicIpRef := clusterScope.GetLinkPublicIpRef()
			linkPublicIpRef.ResourceMap = make(map[string]string)

			if btc.expLinkPublicIpFound {
				linkPublicIpRef.ResourceMap[publicIpName] = linkPublicIpId
			}

			imageId := btc.clusterSpec.Network.Bastion.ImageId
			var privateIps []string
			bastionPrivateIps := clusterScope.GetBastionPrivateIps()
			for _, bastionPrivateIp := range *bastionPrivateIps {
				privateIp := bastionPrivateIp.PrivateIp
				privateIps = append(privateIps, privateIp)
			}

			var securityGroupIds []string
			bastionSecurityGroups := clusterScope.GetBastionSecurityGroups()
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			for _, bastionSecurityGroup := range *bastionSecurityGroups {
				securityGroupName := bastionSecurityGroup.Name + "-uid"
				securityGroupId := "sg-" + securityGroupName
				securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
				securityGroupIds = append(securityGroupIds, securityGroupId)
			}

			bastionSpec := btc.clusterSpec.Network.Bastion

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
			if btc.expTagFound {
				mockOscTagInterface.
					EXPECT().
					ReadTag(gomock.Eq("Name"), gomock.Eq(bastionName)).
					Return(&tag, btc.expReadTagErr)
			} else {
				mockOscTagInterface.
					EXPECT().
					ReadTag(gomock.Eq("Name"), gomock.Eq(bastionName)).
					Return(nil, btc.expReadTagErr)
			}
			bastion := &createVm[0]
			if btc.expCreateVmFound {
				mockOscVmInterface.
					EXPECT().
					CreateVmUserData(gomock.Eq(""), gomock.Eq(&bastionSpec), gomock.Eq(subnetId), gomock.Eq(securityGroupIds), gomock.Eq(privateIps), gomock.Eq(bastionName), gomock.Eq(imageId)).
					Return(bastion, btc.expCreateVmErr)
			} else {
				mockOscVmInterface.
					EXPECT().
					CreateVmUserData(gomock.Eq(""), gomock.Eq(&bastionSpec), gomock.Eq(subnetId), gomock.Eq(securityGroupIds), gomock.Eq(privateIps), gomock.Eq(bastionName), gomock.Eq(imageId)).
					Return(nil, btc.expCreateVmErr)
			}

			reconcileBastion, err := reconcileBastion(ctx, clusterScope, mockOscVmInterface, mockOscPublicIpInterface, mockOscSecurityGroupInterface, mockOscImageInterface, mockOscTagInterface)
			if err != nil {
				assert.Equal(t, btc.expReconcileBastionErr.Error(), err.Error(), "reconcileBastion() should return the same error")
			} else {
				assert.Nil(t, btc.expReconcileBastionErr)
			}
			t.Logf("find reconcileBastion %v\n", reconcileBastion)
		})
	}
}

// TestReconcileLinkBastion has serveral tests to cover the code of function reconcileBastion
func TestReconcileLinkBastion(t *testing.T) {
	bastionTestCases := []struct {
		name                         string
		clusterSpec                  infrastructurev1beta1.OscClusterSpec
		expCreateVmFound             bool
		expLinkPublicIpFound         bool
		expCheckVmStateBootFound     bool
		expCheckVmStatePublicIpFound bool
		expTagFound                  bool
		expCreateVmErr               error
		expReconcileBastionErr       error
		expCheckVmStateBootErr       error
		expCheckVmStatePublicIpErr   error
		expLinkPublicIpErr           error
		expReadTagErr                error
	}{
		{
			name:                         "failed to linkPublicIp",
			clusterSpec:                  defaultBastionInitialize,
			expCreateVmFound:             true,
			expLinkPublicIpFound:         true,
			expCheckVmStateBootFound:     true,
			expCheckVmStatePublicIpFound: false,
			expTagFound:                  false,
			expCheckVmStateBootErr:       nil,
			expCheckVmStatePublicIpErr:   nil,
			expCreateVmErr:               nil,
			expLinkPublicIpErr:           fmt.Errorf("linkPublicIp generic error"),
			expReadTagErr:                nil,
			expReconcileBastionErr:       fmt.Errorf("linkPublicIp generic error Can not link publicIp eipalloc-test-publicip-uid with i-test-bastion-uid for OscCluster test-system/test-osc"),
		},
		{
			name:                         "failed to VmStatePublicIp",
			clusterSpec:                  defaultBastionInitialize,
			expCreateVmFound:             true,
			expLinkPublicIpFound:         true,
			expCheckVmStateBootFound:     true,
			expCheckVmStatePublicIpFound: true,
			expTagFound:                  false,
			expCheckVmStateBootErr:       nil,
			expCheckVmStatePublicIpErr:   nil,
			expCreateVmErr:               nil,
			expLinkPublicIpErr:           nil,
			expReadTagErr:                nil,
			expReconcileBastionErr:       nil,
		},
		{
			name:                         "failed to VmState",
			clusterSpec:                  defaultBastionInitialize,
			expCreateVmFound:             true,
			expLinkPublicIpFound:         true,
			expCheckVmStateBootFound:     true,
			expCheckVmStatePublicIpFound: true,
			expTagFound:                  false,
			expCheckVmStateBootErr:       nil,
			expCheckVmStatePublicIpErr:   nil,
			expCreateVmErr:               nil,
			expLinkPublicIpErr:           nil,
			expReadTagErr:                nil,
			expReconcileBastionErr:       nil,
		},
	}
	for _, btc := range bastionTestCases {
		t.Run(btc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscVmInterface, mockOscPublicIpInterface, mockOscSecurityGroupInterface, mockOscImageInterface, mockOscTagInterface := SetupWithBastionMock(t, btc.name, btc.clusterSpec)
			bastionName := btc.clusterSpec.Network.Bastion.Name + "-uid"
			vmId := "i-" + bastionName
			vmState := "running"

			subnetName := btc.clusterSpec.Network.Bastion.SubnetName + "-uid"
			subnetId := "subnet-" + subnetName
			subnetRef := clusterScope.GetSubnetRef()
			subnetRef.ResourceMap = make(map[string]string)
			subnetRef.ResourceMap[subnetName] = subnetId

			publicIpName := btc.clusterSpec.Network.Bastion.PublicIpName + "-uid"
			publicIpId := "eipalloc-" + publicIpName
			publicIpRef := clusterScope.GetPublicIpRef()
			publicIpRef.ResourceMap = make(map[string]string)
			publicIpRef.ResourceMap[publicIpName] = publicIpId

			linkPublicIpId := "eipassoc-" + publicIpName
			linkPublicIpRef := clusterScope.GetLinkPublicIpRef()
			linkPublicIpRef.ResourceMap = make(map[string]string)

			if btc.expLinkPublicIpFound {
				linkPublicIpRef.ResourceMap[publicIpName] = linkPublicIpId
			}

			imageId := btc.clusterSpec.Network.Bastion.ImageId

			var privateIps []string
			bastionPrivateIps := clusterScope.GetBastionPrivateIps()
			for _, bastionPrivateIp := range *bastionPrivateIps {
				privateIp := bastionPrivateIp.PrivateIp
				privateIps = append(privateIps, privateIp)
			}

			var securityGroupIds []string
			bastionSecurityGroups := clusterScope.GetBastionSecurityGroups()
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			for _, bastionSecurityGroup := range *bastionSecurityGroups {
				securityGroupName := bastionSecurityGroup.Name + "-uid"
				securityGroupId := "sg-" + securityGroupName
				securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
				securityGroupIds = append(securityGroupIds, securityGroupId)
			}

			bastionSpec := btc.clusterSpec.Network.Bastion
			var clockInsideLoop time.Duration = 5
			var clockLoop time.Duration = 120
			var firstClockLoop time.Duration = 120
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
			if btc.expTagFound {
				mockOscTagInterface.
					EXPECT().
					ReadTag(gomock.Eq("Name"), gomock.Eq(bastionName)).
					Return(&tag, btc.expReadTagErr)
			} else {
				mockOscTagInterface.
					EXPECT().
					ReadTag(gomock.Eq("Name"), gomock.Eq(bastionName)).
					Return(nil, btc.expReadTagErr)
			}
			linkPublicIp := osc.LinkPublicIpResponse{
				LinkPublicIpId: &linkPublicIpId,
			}
			bastion := &createVm[0]

			if btc.expCheckVmStateBootFound {
				mockOscVmInterface.
					EXPECT().
					CheckVmState(gomock.Eq(clockInsideLoop), gomock.Eq(firstClockLoop), gomock.Eq(vmState), gomock.Eq(vmId)).
					Return(btc.expCheckVmStateBootErr)
			}
			if btc.expCreateVmFound {
				mockOscVmInterface.
					EXPECT().
					CreateVmUserData(gomock.Eq(""), gomock.Eq(&bastionSpec), gomock.Eq(subnetId), gomock.Eq(securityGroupIds), gomock.Eq(privateIps), gomock.Eq(bastionName), gomock.Eq(imageId)).
					Return(bastion, btc.expCreateVmErr)
			} else {
				mockOscVmInterface.
					EXPECT().
					CreateVmUserData(gomock.Eq(""), gomock.Eq(&bastionSpec), gomock.Eq(subnetId), gomock.Eq(securityGroupIds), gomock.Eq(privateIps), gomock.Eq(bastionName), gomock.Eq(imageId)).
					Return(nil, btc.expCreateVmErr)
			}

			if btc.expLinkPublicIpFound {
				mockOscPublicIpInterface.
					EXPECT().
					LinkPublicIp(gomock.Eq(publicIpId), gomock.Eq(vmId)).
					Return(*linkPublicIp.LinkPublicIpId, btc.expLinkPublicIpErr)
			}
			if btc.expCheckVmStatePublicIpFound {
				mockOscVmInterface.
					EXPECT().
					CheckVmState(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(vmState), gomock.Eq(vmId)).
					Return(btc.expCheckVmStatePublicIpErr)
			}

			reconcileBastion, err := reconcileBastion(ctx, clusterScope, mockOscVmInterface, mockOscPublicIpInterface, mockOscSecurityGroupInterface, mockOscImageInterface, mockOscTagInterface)
			if err != nil {
				assert.Equal(t, btc.expReconcileBastionErr.Error(), err.Error(), "reconcileBastion() should return the same error")
			} else {
				assert.Nil(t, btc.expReconcileBastionErr)
			}
			t.Logf("find reconcileBastion %v\n", reconcileBastion)
		})
	}
}

// TestReconcileBastionGet has several tests to cover the code of the function reconcileBastion
func TestReconcileBastionGet(t *testing.T) {
	bastionTestCases := []struct {
		name                         string
		clusterSpec                  infrastructurev1beta1.OscClusterSpec
		expLinkPublicIpFound         bool
		expGetVmFound                bool
		expGetVmStateFound           bool
		expTagFound                  bool
		expCheckVmStatePublicIpFound bool
		expGetVmErr                  error
		expGetVmStateErr             error
		expReadTagErr                error
		expCheckVmStatePublicIpErr   error
		expLinkPublicIpErr           error

		expReconcileBastionErr error
	}{
		{
			name:                         "get bastion",
			clusterSpec:                  defaultBastionReconcile,
			expLinkPublicIpFound:         false,
			expGetVmFound:                true,
			expGetVmStateFound:           true,
			expTagFound:                  false,
			expCheckVmStatePublicIpFound: false,
			expGetVmErr:                  nil,
			expGetVmStateErr:             nil,
			expReadTagErr:                nil,
			expCheckVmStatePublicIpErr:   nil,
			expLinkPublicIpErr:           nil,
			expReconcileBastionErr:       nil,
		},
		{
			name:                         "get bastion with publicIpNameAfterBastion",
			clusterSpec:                  defaultPublicIpNameAfterBastionReconcile,
			expLinkPublicIpFound:         true,
			expGetVmFound:                true,
			expGetVmStateFound:           true,
			expTagFound:                  false,
			expCheckVmStatePublicIpFound: true,
			expGetVmErr:                  nil,
			expGetVmStateErr:             nil,
			expReadTagErr:                nil,
			expCheckVmStatePublicIpErr:   nil,
			expLinkPublicIpErr:           nil,
			expReconcileBastionErr:       nil,
		},
		{
			name:                         "failed to get bastion",
			clusterSpec:                  defaultBastionReconcile,
			expLinkPublicIpFound:         false,
			expGetVmFound:                true,
			expGetVmStateFound:           false,
			expTagFound:                  false,
			expCheckVmStatePublicIpFound: false,
			expGetVmErr:                  fmt.Errorf("GetVm generic error"),
			expGetVmStateErr:             nil,
			expReadTagErr:                nil,
			expCheckVmStatePublicIpErr:   nil,
			expLinkPublicIpErr:           nil,

			expReconcileBastionErr: fmt.Errorf("GetVm generic error"),
		},
		{
			name:                         "failed to get vmstate",
			clusterSpec:                  defaultBastionReconcile,
			expLinkPublicIpFound:         false,
			expGetVmFound:                true,
			expGetVmStateFound:           true,
			expTagFound:                  false,
			expCheckVmStatePublicIpFound: false,

			expGetVmErr:                nil,
			expGetVmStateErr:           fmt.Errorf("GetVmState generic error"),
			expReadTagErr:              nil,
			expCheckVmStatePublicIpErr: nil,
			expLinkPublicIpErr:         nil,

			expReconcileBastionErr: fmt.Errorf("GetVmState generic error Can not get bastion i-test-bastion-uid state for OscCluster test-system/test-osc"),
		},
	}
	for _, btc := range bastionTestCases {
		t.Run(btc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscVmInterface, mockOscPublicIpInterface, mockOscSecurityGroupInterface, mockOscImageInterface, mockOscTagInterface := SetupWithBastionMock(t, btc.name, btc.clusterSpec)
			bastionName := btc.clusterSpec.Network.Bastion.Name + "-uid"
			vmId := "i-" + bastionName
			vmState := "running"

			subnetName := btc.clusterSpec.Network.Bastion.SubnetName + "-uid"
			subnetId := "subnet-" + subnetName
			subnetRef := clusterScope.GetSubnetRef()
			subnetRef.ResourceMap = make(map[string]string)
			subnetRef.ResourceMap[subnetName] = subnetId

			publicIpName := btc.clusterSpec.Network.Bastion.PublicIpName + "-uid"
			publicIpId := "eipalloc-" + publicIpName
			publicIpRef := clusterScope.GetPublicIpRef()
			publicIpRef.ResourceMap = make(map[string]string)
			publicIpRef.ResourceMap[publicIpName] = publicIpId

			linkPublicIpId := "eipassoc-" + publicIpName
			linkPublicIpRef := clusterScope.GetLinkPublicIpRef()
			linkPublicIpRef.ResourceMap = make(map[string]string)
			linkPublicIpRef.ResourceMap[publicIpName] = linkPublicIpId

			bastionSecurityGroups := clusterScope.GetBastionSecurityGroups()
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			for _, bastionSecurityGroup := range *bastionSecurityGroups {
				securityGroupName := bastionSecurityGroup.Name + "-uid"
				securityGroupId := "sg-" + securityGroupName
				securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
			}
			var clockInsideLoop time.Duration = 5
			var clockLoop time.Duration = 120
			readVms := osc.ReadVmsResponse{
				Vms: &[]osc.Vm{
					{
						VmId: &vmId,
					},
				},
			}
			readVm := *readVms.Vms
			tag := osc.Tag{
				ResourceId: &vmId,
			}
			if btc.expTagFound {
				mockOscTagInterface.
					EXPECT().
					ReadTag(gomock.Eq("Name"), gomock.Eq(bastionName)).
					Return(&tag, btc.expReadTagErr)
			} else {
				mockOscTagInterface.
					EXPECT().
					ReadTag(gomock.Eq("Name"), gomock.Eq(bastionName)).
					Return(nil, btc.expReadTagErr)
			}
			vm := &readVm[0]
			if btc.expGetVmFound {
				mockOscVmInterface.
					EXPECT().
					GetVm(gomock.Eq(vmId)).
					Return(vm, btc.expGetVmErr)
			} else {
				mockOscVmInterface.
					EXPECT().
					GetVm(gomock.Eq(vmId)).
					Return(nil, btc.expGetVmErr)
			}
			if btc.expGetVmStateFound {
				mockOscVmInterface.
					EXPECT().
					GetVmState(gomock.Eq(vmId)).
					Return(vmState, btc.expGetVmStateErr)
			}
			linkPublicIp := osc.LinkPublicIpResponse{
				LinkPublicIpId: &linkPublicIpId,
			}
			if btc.expLinkPublicIpFound {
				mockOscPublicIpInterface.
					EXPECT().
					LinkPublicIp(gomock.Eq(publicIpId), gomock.Eq(vmId)).
					Return(*linkPublicIp.LinkPublicIpId, btc.expLinkPublicIpErr)
			}
			if btc.expCheckVmStatePublicIpFound {
				mockOscVmInterface.
					EXPECT().
					CheckVmState(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(vmState), gomock.Eq(vmId)).
					Return(btc.expCheckVmStatePublicIpErr)
			}

			reconcileBastion, err := reconcileBastion(ctx, clusterScope, mockOscVmInterface, mockOscPublicIpInterface, mockOscSecurityGroupInterface, mockOscImageInterface, mockOscTagInterface)
			if err != nil {
				assert.Equal(t, btc.expReconcileBastionErr.Error(), err.Error(), "reconcileBastion() should return the same error")
			} else {
				assert.Nil(t, btc.expReconcileBastionErr)
			}
			t.Logf("find reconcileBastion %v\n", reconcileBastion)
		})
	}
}

// TestReconcileBastionResourceId has serveral tests to cover the code of function reconcileBastion
func TestReconcileBastionResourceId(t *testing.T) {
	bastionTestCases := []struct {
		name                   string
		clusterSpec            infrastructurev1beta1.OscClusterSpec
		expLinkPublicIpFound   bool
		expGetImageNameFound   bool
		expSubnetFound         bool
		expPublicIpFound       bool
		expSecurityGroupFound  bool
		expTagFound            bool
		expGetImageIdErr       error
		expReadTagErr          error
		expReconcileBastionErr error
	}{
		{
			name:                   "PublicIp does not exist",
			clusterSpec:            defaultBastionInitialize,
			expGetImageNameFound:   false,
			expSubnetFound:         true,
			expPublicIpFound:       false,
			expLinkPublicIpFound:   true,
			expSecurityGroupFound:  true,
			expTagFound:            false,
			expGetImageIdErr:       nil,
			expReadTagErr:          nil,
			expReconcileBastionErr: fmt.Errorf("test-publicip-uid does not exist"),
		},
		{
			name:                   "Subnet does not exist",
			clusterSpec:            defaultBastionInitialize,
			expGetImageNameFound:   false,
			expSubnetFound:         false,
			expPublicIpFound:       true,
			expLinkPublicIpFound:   true,
			expSecurityGroupFound:  true,
			expGetImageIdErr:       nil,
			expReadTagErr:          nil,
			expReconcileBastionErr: fmt.Errorf("test-subnet-uid does not exist"),
		},
		{
			name:                   "SecurityGroup does not exist",
			clusterSpec:            defaultBastionInitialize,
			expGetImageNameFound:   false,
			expSubnetFound:         true,
			expPublicIpFound:       true,
			expLinkPublicIpFound:   false,
			expSecurityGroupFound:  false,
			expGetImageIdErr:       nil,
			expReadTagErr:          nil,
			expReconcileBastionErr: fmt.Errorf("test-securitygroup-uid does not exist"),
		},
		{
			name:                   "failed to get ImageId",
			clusterSpec:            defaultBastionImageInitialize,
			expGetImageNameFound:   true,
			expSubnetFound:         true,
			expPublicIpFound:       false,
			expLinkPublicIpFound:   false,
			expSecurityGroupFound:  false,
			expGetImageIdErr:       fmt.Errorf("GetImageId generic error"),
			expReadTagErr:          nil,
			expReconcileBastionErr: fmt.Errorf("GetImageId generic error"),
		},
		{
			name:                   "failed to get tag",
			clusterSpec:            defaultBastionInitialize,
			expGetImageNameFound:   false,
			expSubnetFound:         true,
			expPublicIpFound:       true,
			expLinkPublicIpFound:   true,
			expSecurityGroupFound:  true,
			expGetImageIdErr:       nil,
			expReadTagErr:          fmt.Errorf("ReadTag generic error"),
			expReconcileBastionErr: fmt.Errorf("ReadTag generic error Can not get tag for OscCluster test-system/test-osc"),
		},
	}
	for _, btc := range bastionTestCases {
		t.Run(btc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscVmInterface, mockOscPublicIpInterface, mockOscSecurityGroupInterface, mockOscImageInterface, mockOscTagInterface := SetupWithBastionMock(t, btc.name, btc.clusterSpec)
			bastionName := btc.clusterSpec.Network.Bastion.Name + "-uid"
			vmId := "i-" + bastionName

			subnetName := btc.clusterSpec.Network.Bastion.SubnetName + "-uid"
			subnetId := "subnet-" + subnetName
			subnetRef := clusterScope.GetSubnetRef()
			subnetRef.ResourceMap = make(map[string]string)
			if btc.expSubnetFound {
				subnetRef.ResourceMap[subnetName] = subnetId
			}

			publicIpName := btc.clusterSpec.Network.Bastion.PublicIpName + "-uid"
			publicIpId := "eipalloc-" + publicIpName
			publicIpRef := clusterScope.GetPublicIpRef()
			publicIpRef.ResourceMap = make(map[string]string)
			if btc.expPublicIpFound {
				publicIpRef.ResourceMap[publicIpName] = publicIpId
			}

			linkPublicIpId := "eipassoc-" + publicIpName
			linkPublicIpRef := clusterScope.GetLinkPublicIpRef()
			linkPublicIpRef.ResourceMap = make(map[string]string)

			if btc.expLinkPublicIpFound {
				linkPublicIpRef.ResourceMap[publicIpName] = linkPublicIpId
			}

			imageName := btc.clusterSpec.Network.Bastion.ImageName
			imageId := "ami-00000000"
			bastionSecurityGroups := clusterScope.GetBastionSecurityGroups()
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			for _, bastionSecurityGroup := range *bastionSecurityGroups {
				securityGroupName := bastionSecurityGroup.Name + "-uid"
				securityGroupId := "sg-" + securityGroupName
				if btc.expSecurityGroupFound {
					securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
				}
			}

			if btc.expGetImageNameFound {
				mockOscImageInterface.
					EXPECT().
					GetImageId(gomock.Eq(imageName)).
					Return(imageId, btc.expGetImageIdErr)
			}
			tag := osc.Tag{
				ResourceId: &vmId,
			}
			if btc.expSubnetFound && btc.expPublicIpFound && btc.expLinkPublicIpFound && btc.expSecurityGroupFound {
				if btc.expTagFound {
					mockOscTagInterface.
						EXPECT().
						ReadTag(gomock.Eq("Name"), gomock.Eq(bastionName)).
						Return(&tag, btc.expReadTagErr)
				} else {
					mockOscTagInterface.
						EXPECT().
						ReadTag(gomock.Eq("Name"), gomock.Eq(bastionName)).
						Return(nil, btc.expReadTagErr)
				}
			}
			reconcileBastion, err := reconcileBastion(ctx, clusterScope, mockOscVmInterface, mockOscPublicIpInterface, mockOscSecurityGroupInterface, mockOscImageInterface, mockOscTagInterface)
			if err != nil {
				assert.Equal(t, btc.expReconcileBastionErr.Error(), err.Error(), "reconcileBastion() should return the same error")
			} else {
				assert.Nil(t, btc.expReconcileBastionErr)
			}

			t.Logf("find reconcileBastion %v\n", reconcileBastion)
		})
	}
}

// TestReconcileDeleteBastion has several tests to cover the code of the function reconcileDeleteBastion
func TestReconcileDeleteBastion(t *testing.T) {
	bastionTestCases := []struct {
		name                         string
		clusterSpec                  infrastructurev1beta1.OscClusterSpec
		expDeleteBastionErr          error
		expGetBastionErr             error
		expGetBastionFound           bool
		expCheckUnlinkPublicIpFound  bool
		expCheckUnlinkPublicIpErr    error
		expReconcileDeleteBastionErr error
	}{
		{
			name:                         "delete bastion",
			clusterSpec:                  defaultBastionReconcile,
			expGetBastionFound:           true,
			expCheckUnlinkPublicIpFound:  true,
			expCheckUnlinkPublicIpErr:    nil,
			expDeleteBastionErr:          nil,
			expReconcileDeleteBastionErr: nil,
			expGetBastionErr:             nil,
		},
		{
			name:                         "failed to delete bastion",
			clusterSpec:                  defaultBastionReconcile,
			expGetBastionFound:           true,
			expCheckUnlinkPublicIpFound:  true,
			expCheckUnlinkPublicIpErr:    nil,
			expDeleteBastionErr:          fmt.Errorf("DeleteVm generic error"),
			expReconcileDeleteBastionErr: fmt.Errorf("DeleteVm generic error Can not delete vm for OscCluster test-system/test-osc"),
			expGetBastionErr:             nil,
		},
	}
	for _, btc := range bastionTestCases {
		t.Run(btc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscVmInterface, mockOscPublicIpInterface, mockOscSecurityGroupInterface, _, _ := SetupWithBastionMock(t, btc.name, btc.clusterSpec)
			bastionName := btc.clusterSpec.Network.Bastion.Name + "-uid"
			vmId := "i-" + bastionName
			bastionRef := clusterScope.GetBastionRef()
			bastionRef.ResourceMap = make(map[string]string)
			if btc.expGetBastionFound {
				bastionRef.ResourceMap[bastionName] = vmId
			}

			bastionSecurityGroups := clusterScope.GetBastionSecurityGroups()
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			for _, bastionSecurityGroup := range *bastionSecurityGroups {
				securityGroupName := bastionSecurityGroup.Name + "-uid"
				securityGroupId := "sg-" + securityGroupName
				securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
			}

			publicIpName := btc.clusterSpec.Network.Bastion.PublicIpName + "-uid"
			linkPublicIpId := "eipassoc-" + publicIpName
			linkPublicIpRef := clusterScope.GetLinkPublicIpRef()
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
			bastion := &createVm[0]
			if btc.expGetBastionFound {
				mockOscVmInterface.
					EXPECT().
					GetVm(gomock.Eq(vmId)).
					Return(bastion, btc.expGetBastionErr)
			} else {
				mockOscVmInterface.
					EXPECT().
					GetVm(gomock.Eq(vmId)).
					Return(nil, btc.expGetBastionErr)
			}

			if btc.expCheckUnlinkPublicIpFound {
				mockOscPublicIpInterface.
					EXPECT().
					UnlinkPublicIp(gomock.Eq(linkPublicIpId)).
					Return(btc.expCheckUnlinkPublicIpErr)
			}
			mockOscVmInterface.
				EXPECT().
				DeleteVm(gomock.Eq(vmId)).
				Return(btc.expDeleteBastionErr)

			reconcileDeleteBastion, err := reconcileDeleteBastion(ctx, clusterScope, mockOscVmInterface, mockOscPublicIpInterface, mockOscSecurityGroupInterface)
			if err != nil {
				assert.Equal(t, btc.expReconcileDeleteBastionErr.Error(), err.Error(), "reconcileDeleteBastion() should return the same error")
			} else {
				assert.Nil(t, btc.expReconcileDeleteBastionErr)
			}

			t.Logf("find reconcileDeleteBastion %v\n", reconcileDeleteBastion)
		})
	}
}

// TestReconcileDeleteBastionResourceId has several tests to cover the code of the function reconcileDeleteBastion
func TestReconcileDeleteBastionResourceId(t *testing.T) {
	bastionTestCases := []struct {
		name                         string
		clusterSpec                  infrastructurev1beta1.OscClusterSpec
		expGetBastionFound           bool
		expGetBastionErr             error
		expGetImageIdErr             error
		expSecurityGroupFound        bool
		expReconcileDeleteBastionErr error
	}{
		{
			name:                         "failed to find security group",
			clusterSpec:                  defaultBastionReconcile,
			expGetBastionFound:           true,
			expGetImageIdErr:             nil,
			expGetBastionErr:             nil,
			expSecurityGroupFound:        false,
			expReconcileDeleteBastionErr: fmt.Errorf("test-securitygroup-uid does not exist"),
		},
		{
			name:                         "failed to get bastion",
			clusterSpec:                  defaultBastionReconcile,
			expGetBastionFound:           true,
			expGetImageIdErr:             nil,
			expGetBastionErr:             fmt.Errorf("GetVm generic error"),
			expSecurityGroupFound:        false,
			expReconcileDeleteBastionErr: fmt.Errorf("GetVm generic error"),
		},
		{
			name:                         "bastion is already destroyed",
			clusterSpec:                  defaultBastionInitialize,
			expGetBastionFound:           false,
			expGetImageIdErr:             nil,
			expGetBastionErr:             nil,
			expSecurityGroupFound:        false,
			expReconcileDeleteBastionErr: nil,
		},
		{
			name:                         "bastion does not exist anymore",
			clusterSpec:                  defaultBastionReconcile,
			expGetBastionFound:           true,
			expGetImageIdErr:             nil,
			expGetBastionErr:             nil,
			expSecurityGroupFound:        true,
			expReconcileDeleteBastionErr: nil,
		},
	}
	for _, btc := range bastionTestCases {
		t.Run(btc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscVmInterface, mockOscPublicIpInterface, mockOscSecurityGroupInterface, _, _ := SetupWithBastionMock(t, btc.name, btc.clusterSpec)
			bastionName := btc.clusterSpec.Network.Bastion.Name + "-uid"
			vmId := "i-" + bastionName
			bastionRef := clusterScope.GetBastionRef()
			bastionRef.ResourceMap = make(map[string]string)
			if btc.expGetBastionFound {
				bastionRef.ResourceMap[bastionName] = vmId
			}

			if btc.expGetBastionFound {
				mockOscVmInterface.
					EXPECT().
					GetVm(gomock.Eq(vmId)).
					Return(nil, btc.expGetBastionErr)
			}

			bastionSecurityGroups := clusterScope.GetBastionSecurityGroups()
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			for _, bastionSecurityGroup := range *bastionSecurityGroups {
				securityGroupName := bastionSecurityGroup.Name + "-uid"
				securityGroupId := "sg-" + securityGroupName
				if btc.expSecurityGroupFound {
					securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
				}
			}

			publicIpName := btc.clusterSpec.Network.Bastion.PublicIpName + "-uid"
			linkPublicIpId := "eipassoc-" + publicIpName
			linkPublicIpRef := clusterScope.GetLinkPublicIpRef()
			linkPublicIpRef.ResourceMap = make(map[string]string)
			linkPublicIpRef.ResourceMap[publicIpName] = linkPublicIpId

			reconcileDeleteBastion, err := reconcileDeleteBastion(ctx, clusterScope, mockOscVmInterface, mockOscPublicIpInterface, mockOscSecurityGroupInterface)
			if err != nil {
				assert.Equal(t, btc.expReconcileDeleteBastionErr, err, "reconcileDeleteBastion() should return the same error")
			} else {
				assert.Nil(t, btc.expReconcileDeleteBastionErr)
			}
			t.Logf("find reconcileDeletBastion %v\n", reconcileDeleteBastion)
		})
	}
}

// TestReconcileDeleteBastionWithoutSpec has several tests to cover the code of function reconcileDeleteBastion
func TestReconcileDeleteBastionWithoutSpec(t *testing.T) {
	bastionTestCases := []struct {
		name                         string
		clusterSpec                  infrastructurev1beta1.OscClusterSpec
		expCheckBastionStateBootErr  error
		expDeleteBastionErr          error
		expGetBastionErr             error
		expGetBastionFound           bool
		expSecurityGroupFound        bool
		expCheckUnlinkPublicIpErr    error
		expReconcileDeleteBastionErr error
	}{
		{
			name: "delete bastion without spec",
			clusterSpec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{

						ResourceId: "vpc-test-net-uid",
					},
					Subnets: []*infrastructurev1beta1.OscSubnet{
						{
							ResourceId: "subnet-test-subnet-uid",
						},
					},
					SecurityGroups: []*infrastructurev1beta1.OscSecurityGroup{
						{
							ResourceId: "sg-test-securitygroup-uid",
						},
					},
					PublicIps: []*infrastructurev1beta1.OscPublicIp{
						{
							ResourceId: "test-publicip-uid",
						},
					},
					Bastion: infrastructurev1beta1.OscBastion{
						Enable:      true,
						ClusterName: "test-cluster",
						Name:        "test-bastion",
						ImageId:     "ami-00000000",
						DeviceName:  "/dev/xvdb",
						KeypairName: "rke",
						RootDisk: infrastructurev1beta1.OscRootDisk{

							RootDiskSize: 30,
							RootDiskIops: 1500,
							RootDiskType: "io1",
						},
						SubregionName: "eu-west-2a",
						SubnetName:    "test-subnet",
						VmType:        "tinav3.c2r4p2",
						PublicIpName:  "test-publicip",
						ResourceId:    "i-test-bastion-uid",
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
			},
			expSecurityGroupFound:        true,
			expGetBastionFound:           true,
			expCheckBastionStateBootErr:  nil,
			expDeleteBastionErr:          nil,
			expCheckUnlinkPublicIpErr:    nil,
			expReconcileDeleteBastionErr: nil,
		},
	}
	for _, btc := range bastionTestCases {
		t.Run(btc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscVmInterface, mockOscPublicIpInterface, mockOscSecurityGroupInterface, _, _ := SetupWithBastionMock(t, btc.name, btc.clusterSpec)
			bastionName := btc.clusterSpec.Network.Bastion.Name + "-uid"
			vmId := "i-" + bastionName

			bastionRef := clusterScope.GetBastionRef()
			bastionRef.ResourceMap = make(map[string]string)
			if btc.expGetBastionFound {
				bastionRef.ResourceMap[bastionName] = vmId
			}

			createVms := osc.CreateVmsResponse{
				Vms: &[]osc.Vm{
					{
						VmId: &vmId,
					},
				},
			}
			createVm := *createVms.Vms
			bastion := &createVm[0]
			if btc.expGetBastionFound {
				mockOscVmInterface.
					EXPECT().
					GetVm(gomock.Eq(vmId)).
					Return(bastion, btc.expGetBastionErr)
			} else {
				mockOscVmInterface.
					EXPECT().
					GetVm(gomock.Eq(vmId)).
					Return(nil, btc.expGetBastionErr)
			}

			bastionSecurityGroups := clusterScope.GetBastionSecurityGroups()
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			for _, bastionSecurityGroup := range *bastionSecurityGroups {
				securityGroupName := bastionSecurityGroup.Name + "-uid"
				securityGroupId := "sg-" + securityGroupName
				if btc.expSecurityGroupFound {
					securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
				}
			}

			publicIpName := btc.clusterSpec.Network.Bastion.PublicIpName + "-uid"
			linkPublicIpId := "eipasoc-" + publicIpName
			linkPublicIpRef := clusterScope.GetLinkPublicIpRef()
			linkPublicIpRef.ResourceMap = make(map[string]string)
			linkPublicIpRef.ResourceMap[publicIpName] = linkPublicIpId

			mockOscPublicIpInterface.
				EXPECT().
				UnlinkPublicIp(gomock.Eq(linkPublicIpId)).
				Return(btc.expCheckUnlinkPublicIpErr)
			mockOscVmInterface.
				EXPECT().
				DeleteVm(gomock.Eq(vmId)).
				Return(btc.expDeleteBastionErr)
			reconcileDeleteBastion, err := reconcileDeleteBastion(ctx, clusterScope, mockOscVmInterface, mockOscPublicIpInterface, mockOscSecurityGroupInterface)
			if err != nil {
				assert.Equal(t, btc.expReconcileDeleteBastionErr, err, "reconcileDeleteBastion() should return the same error")
			} else {
				assert.Nil(t, btc.expReconcileDeleteBastionErr)
			}

			t.Logf("find reconcileDeleteBastion %v\n", reconcileDeleteBastion)
		})
	}
}

// TestReconcileDeleteBastionUnlinkPublicIp has several tests to cover the code of the function reconcileDeleteVm
func TestReconcileDeleteBastionUnlinkPublicIp(t *testing.T) {
	bastionTestCases := []struct {
		name                         string
		clusterSpec                  infrastructurev1beta1.OscClusterSpec
		expCheckUnlinkPublicIpFound  bool
		expGetVmFound                bool
		expGetVmErr                  error
		expCheckUnlinkPublicIpErr    error
		expReconcileDeleteBastionErr error
	}{
		{
			name:                         "failed to unlink publicIp",
			clusterSpec:                  defaultBastionReconcile,
			expGetVmFound:                true,
			expGetVmErr:                  nil,
			expCheckUnlinkPublicIpFound:  true,
			expCheckUnlinkPublicIpErr:    fmt.Errorf("CheckUnlinkPublicIp generic error"),
			expReconcileDeleteBastionErr: fmt.Errorf("CheckUnlinkPublicIp generic error Can not unlink publicIp for OscCluster test-system/test-osc"),
		},
	}
	for _, btc := range bastionTestCases {
		t.Run(btc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscVmInterface, mockOscPublicIpInterface, mockOscSecurityGroupInterface, _, _ := SetupWithBastionMock(t, btc.name, btc.clusterSpec)
			bastionName := btc.clusterSpec.Network.Bastion.Name + "-uid"
			vmId := "i-" + bastionName

			publicIpName := btc.clusterSpec.Network.Bastion.PublicIpName + "-uid"
			linkPublicIpId := "eipassoc-" + publicIpName
			linkPublicIpRef := clusterScope.GetLinkPublicIpRef()
			linkPublicIpRef.ResourceMap = make(map[string]string)
			linkPublicIpRef.ResourceMap[publicIpName] = linkPublicIpId

			bastionSecurityGroups := clusterScope.GetBastionSecurityGroups()
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			for _, bastionSecurityGroup := range *bastionSecurityGroups {
				securityGroupName := bastionSecurityGroup.Name + "-uid"
				securityGroupId := "sg-" + securityGroupName
				securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
			}

			createVms := osc.CreateVmsResponse{
				Vms: &[]osc.Vm{
					{
						VmId: &vmId,
					},
				},
			}

			createVm := *createVms.Vms
			bastion := &createVm[0]

			if btc.expGetVmFound {
				mockOscVmInterface.
					EXPECT().
					GetVm(gomock.Eq(vmId)).
					Return(bastion, btc.expGetVmErr)
			} else {
				mockOscVmInterface.
					EXPECT().
					GetVm(gomock.Eq(vmId)).
					Return(nil, btc.expGetVmErr)
			}

			if btc.expCheckUnlinkPublicIpFound {
				mockOscPublicIpInterface.
					EXPECT().
					UnlinkPublicIp(gomock.Eq(linkPublicIpId)).
					Return(btc.expCheckUnlinkPublicIpErr)
			}
			reconcileDeleteBastion, err := reconcileDeleteBastion(ctx, clusterScope, mockOscVmInterface, mockOscPublicIpInterface, mockOscSecurityGroupInterface)
			if err != nil {
				assert.Equal(t, btc.expReconcileDeleteBastionErr.Error(), err.Error(), "reconcileDeleteBastion() should return the same error")
			} else {
				assert.Nil(t, btc.expReconcileDeleteBastionErr)
			}

			t.Logf("find reconcileDeleteBastion %v\n", reconcileDeleteBastion)
		})
	}
}
