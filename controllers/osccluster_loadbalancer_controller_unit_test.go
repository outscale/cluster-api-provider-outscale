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
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/security/mock_security"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/service/mock_service"
	osc "github.com/outscale/osc-sdk-go/v2"
	"github.com/stretchr/testify/require"
)

var (
	defaultLoadBalancerInitialize = infrastructurev1beta1.OscClusterSpec{
		Network: infrastructurev1beta1.OscNetwork{
			Net: infrastructurev1beta1.OscNet{
				Name:    "test-net",
				IpRange: "10.0.0.0/16",
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
		},
	}

	defaultLoadBalancerReconcile = infrastructurev1beta1.OscClusterSpec{
		Network: infrastructurev1beta1.OscNetwork{
			Net: infrastructurev1beta1.OscNet{
				Name:       "test-net",
				IpRange:    "10.0.0.0/16",
				ResourceId: "vpc-test-net-uid",
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
		},
	}
)

// SetupWithLoadBalancerMock set loadBalancerMock with clusterScope and osccluster
func SetupWithLoadBalancerMock(t *testing.T, name string, spec infrastructurev1beta1.OscClusterSpec) (clusterScope *scope.ClusterScope, ctx context.Context, mockOscLoadBalancerInterface *mock_service.MockOscLoadBalancerInterface, mockOscSecurityGroupInterface *mock_security.MockOscSecurityGroupInterface) {
	clusterScope = Setup(t, name, spec)
	mockCtrl := gomock.NewController(t)
	mockOscLoadBalancerInterface = mock_service.NewMockOscLoadBalancerInterface(mockCtrl)
	mockOscSecurityGroupInterface = mock_security.NewMockOscSecurityGroupInterface(mockCtrl)
	ctx = context.Background()
	return clusterScope, ctx, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface
}

// TestCheckLoadBalancerSubnetOscAssociateResourceName has several tests to cover the code of the function checkLoadBalancerSubnetOscAssociateResourceName
func TestCheckLoadBalancerSubnetOscAssociateResourceName(t *testing.T) {
	loadBalancerTestCases := []struct {
		name                                                  string
		spec                                                  infrastructurev1beta1.OscClusterSpec
		expCheckLoadBalancerSubnetOscAssociateResourceNameErr error
	}{
		{
			name: "check loadBalancer association with subnet",
			spec: defaultLoadBalancerInitialize,
			expCheckLoadBalancerSubnetOscAssociateResourceNameErr: nil,
		},
		{
			name: "check loadBalancer association with bad subnet",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IpRange: "10.0.0.0/16",
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
						SubnetName:        "test-subnet-test",
						SecurityGroupName: "test-securitygroup",
					},
				},
			},
			expCheckLoadBalancerSubnetOscAssociateResourceNameErr: errors.New("test-subnet-test-uid subnet does not exist in loadBalancer"),
		},
	}
	for _, lbtc := range loadBalancerTestCases {
		t.Run(lbtc.name, func(t *testing.T) {
			clusterScope := Setup(t, lbtc.name, lbtc.spec)
			err := checkLoadBalancerSubnetOscAssociateResourceName(clusterScope)
			if lbtc.expCheckLoadBalancerSubnetOscAssociateResourceNameErr != nil {
				require.EqualError(t, err, lbtc.expCheckLoadBalancerSubnetOscAssociateResourceNameErr.Error(), "checkLoadBalancerSubnetOscAssociateResourceName() should return the same error")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestCheckLoadBalancerFormatParameters has several tests to cover the code of the function checkLoadBalancerFormatParameters
func TestCheckLoadBalancerFormatParameters(t *testing.T) {
	loadBalancerTestCases := []struct {
		name                                    string
		spec                                    infrastructurev1beta1.OscClusterSpec
		expCheckLoadBalancerFormatParametersErr error
	}{
		{
			name: "check success without loadBalancer spec (with default values)",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{},
			},
			expCheckLoadBalancerFormatParametersErr: nil,
		},
		{
			name:                                    "check loadBalancer format",
			spec:                                    defaultLoadBalancerInitialize,
			expCheckLoadBalancerFormatParametersErr: nil,
		},
		{
			name: "check invalid name loadBalancer",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					LoadBalancer: infrastructurev1beta1.OscLoadBalancer{
						LoadBalancerName:  "test-loadbalancer@test",
						LoadBalancerType:  "internet-facing",
						SubnetName:        "test-subnet",
						SecurityGroupName: "test-securitygroup",
					},
				},
			},
			expCheckLoadBalancerFormatParametersErr: errors.New("test-loadbalancer@test is an invalid loadBalancer name: Invalid Description"),
		},
		{
			name: "check invalid type loadBalancer",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					LoadBalancer: infrastructurev1beta1.OscLoadBalancer{
						LoadBalancerName:  "test-loadbalancer",
						LoadBalancerType:  "internet",
						SubnetName:        "test-subnet",
						SecurityGroupName: "test-securitygroup",
					},
				},
			},
			expCheckLoadBalancerFormatParametersErr: errors.New("internet is an invalid loadBalancer type: Invalid LoadBalancerType"),
		},
		{
			name: "check invalid backend port loadBalancer",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					LoadBalancer: infrastructurev1beta1.OscLoadBalancer{
						Listener: infrastructurev1beta1.OscLoadBalancerListener{
							BackendPort:          65537,
							BackendProtocol:      "TCP",
							LoadBalancerPort:     6443,
							LoadBalancerProtocol: "TCP",
						},
					},
				},
			},
			expCheckLoadBalancerFormatParametersErr: errors.New("65537 is an Invalid Port for loadBalancer backend"),
		},
		{
			name: "check invalid backend protocol loadBalancer",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					LoadBalancer: infrastructurev1beta1.OscLoadBalancer{
						Listener: infrastructurev1beta1.OscLoadBalancerListener{
							BackendPort:          6443,
							BackendProtocol:      "SCTP",
							LoadBalancerPort:     6443,
							LoadBalancerProtocol: "TCP",
						},
					},
				},
			},
			expCheckLoadBalancerFormatParametersErr: errors.New("SCTP is an Invalid protocol for loadBalancer backend"),
		},
		{
			name: "check invalid loadBalancer port",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					LoadBalancer: infrastructurev1beta1.OscLoadBalancer{
						Listener: infrastructurev1beta1.OscLoadBalancerListener{
							BackendPort:          6443,
							BackendProtocol:      "TCP",
							LoadBalancerPort:     65537,
							LoadBalancerProtocol: "TCP",
						},
					},
				},
			},
			expCheckLoadBalancerFormatParametersErr: errors.New("65537 is an Invalid Port for loadBalancer"),
		},
		{
			name: "check invalid loadBalancer protocol",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					LoadBalancer: infrastructurev1beta1.OscLoadBalancer{
						Listener: infrastructurev1beta1.OscLoadBalancerListener{
							BackendPort:          6443,
							BackendProtocol:      "TCP",
							LoadBalancerPort:     6443,
							LoadBalancerProtocol: "SCTP",
						},
					},
				},
			},
			expCheckLoadBalancerFormatParametersErr: errors.New("SCTP is an Invalid protocol for loadBalancer"),
		},
		{
			name: "check invalid loadBalancer health check interval",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					LoadBalancer: infrastructurev1beta1.OscLoadBalancer{
						HealthCheck: infrastructurev1beta1.OscLoadBalancerHealthCheck{
							CheckInterval:      602,
							HealthyThreshold:   10,
							Port:               6443,
							Protocol:           "TCP",
							Timeout:            5,
							UnhealthyThreshold: 2,
						},
					},
				},
			},
			expCheckLoadBalancerFormatParametersErr: errors.New("602 is an Invalid Interval for loadBalancer"),
		},
		{
			name: "check invalid loadBalancer healthcheck healthy threshold",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					LoadBalancer: infrastructurev1beta1.OscLoadBalancer{
						HealthCheck: infrastructurev1beta1.OscLoadBalancerHealthCheck{
							CheckInterval:      30,
							HealthyThreshold:   12,
							Port:               6443,
							Protocol:           "TCP",
							Timeout:            5,
							UnhealthyThreshold: 2,
						},
					},
				},
			},
			expCheckLoadBalancerFormatParametersErr: errors.New("12 is an Invalid threshold for loadBalancer"),
		},
		{
			name: "check invalid loadBalancer healthcheck port",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					LoadBalancer: infrastructurev1beta1.OscLoadBalancer{
						HealthCheck: infrastructurev1beta1.OscLoadBalancerHealthCheck{
							CheckInterval:      30,
							HealthyThreshold:   10,
							Port:               65537,
							Protocol:           "TCP",
							Timeout:            5,
							UnhealthyThreshold: 2,
						},
					},
				},
			},
			expCheckLoadBalancerFormatParametersErr: errors.New("65537 is an Invalid Port for loadBalancer"),
		},
		{
			name: "check invalid loadBalancer healthcheck protocol",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					LoadBalancer: infrastructurev1beta1.OscLoadBalancer{
						HealthCheck: infrastructurev1beta1.OscLoadBalancerHealthCheck{
							CheckInterval:      30,
							HealthyThreshold:   10,
							Port:               6443,
							Protocol:           "SCTP",
							Timeout:            5,
							UnhealthyThreshold: 2,
						},
					},
				},
			},
			expCheckLoadBalancerFormatParametersErr: errors.New("SCTP is an Invalid protocol for loadBalancer"),
		},
		{
			name: "check invalid loadBalancer healthcheck timeout",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					LoadBalancer: infrastructurev1beta1.OscLoadBalancer{
						HealthCheck: infrastructurev1beta1.OscLoadBalancerHealthCheck{
							CheckInterval:      30,
							HealthyThreshold:   10,
							Port:               6443,
							Protocol:           "TCP",
							Timeout:            62,
							UnhealthyThreshold: 2,
						},
					},
				},
			},
			expCheckLoadBalancerFormatParametersErr: errors.New("62 is an Invalid Timeout for loadBalancer"),
		},
		{
			name: "check invalid loadBalancer healthcheck unhealthy threshold",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					LoadBalancer: infrastructurev1beta1.OscLoadBalancer{
						HealthCheck: infrastructurev1beta1.OscLoadBalancerHealthCheck{
							CheckInterval:      30,
							HealthyThreshold:   10,
							Port:               6443,
							Protocol:           "TCP",
							Timeout:            5,
							UnhealthyThreshold: 12,
						},
					},
				},
			},
			expCheckLoadBalancerFormatParametersErr: errors.New("12 is an Invalid threshold for loadBalancer"),
		},
	}
	for _, lbtc := range loadBalancerTestCases {
		t.Run(lbtc.name, func(t *testing.T) {
			clusterScope := Setup(t, lbtc.name, lbtc.spec)
			loadBalancerName, err := checkLoadBalancerFormatParameters(clusterScope)
			if lbtc.expCheckLoadBalancerFormatParametersErr != nil {
				require.EqualError(t, err, lbtc.expCheckLoadBalancerFormatParametersErr.Error(), "checkLoadBalancerFormatParameters should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("Find loadBalancer %s\n", loadBalancerName)
		})
	}
}

// TestCheckLoadBalancerSecurityGroupOscAssociateResourceName has several tests to cover the code of the function checkLoadBalancerSecurityGroupOscAssociateResourceName
func TestCheckLoadBalancerSecurityGroupOscAssociateResourceName(t *testing.T) {
	loadBalancerTestCases := []struct {
		name                                                        string
		spec                                                        infrastructurev1beta1.OscClusterSpec
		expCheckLoadBalancerSecuriyGroupOscAssociateResourceNameErr error
	}{
		{
			name: "check loadBalancer association with securityGroup",
			spec: defaultLoadBalancerInitialize,
			expCheckLoadBalancerSecuriyGroupOscAssociateResourceNameErr: nil,
		},
		{
			name: "check loadBalancer association with bad securitygroup",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IpRange: "10.0.0.0/16",
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
						SecurityGroupName: "test-securitygroup-test",
					},
				},
			},
			expCheckLoadBalancerSecuriyGroupOscAssociateResourceNameErr: errors.New("test-securitygroup-test-uid securityGroup does not exist in loadBalancer"),
		},
	}
	for _, lbtc := range loadBalancerTestCases {
		t.Run(lbtc.name, func(t *testing.T) {
			clusterScope := Setup(t, lbtc.name, lbtc.spec)
			err := checkLoadBalancerSecurityGroupOscAssociateResourceName(clusterScope)
			if lbtc.expCheckLoadBalancerSecuriyGroupOscAssociateResourceNameErr != nil {
				require.EqualError(t, err, lbtc.expCheckLoadBalancerSecuriyGroupOscAssociateResourceNameErr.Error(), "checkLoadBalancerSecurityGroupOscAssociateResourceName() should return the same error")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestReconcileLoadBalancer has several tests to cover the code of the function reconcileLoadBalancer
func TestReconcileLoadBalancer(t *testing.T) {
	loadBalancerTestCases := []struct {
		name                          string
		spec                          infrastructurev1beta1.OscClusterSpec
		expLoadBalancerFound          bool
		expSubnetFound                bool
		expSecurityGroupFound         bool
		expCreateLoadBalancerFound    bool
		expConfigureLoadBalancerFound bool
		expDeleteOutboundSgRule       bool
		expCreateLoadBalancerTag      bool
		expDeleteOutboundSgRuleErr    error
		expDescribeLoadBalancerErr    error
		expCreateLoadBalancerErr      error
		expConfigureLoadBalancerErr   error
		expCreateLoadbalancerTagErr   error
		expReconcileLoadBalancerErr   error
	}{
		{
			name:                          "create loadBalancer (first time reconcile loop)",
			spec:                          defaultLoadBalancerInitialize,
			expLoadBalancerFound:          false,
			expSubnetFound:                true,
			expSecurityGroupFound:         true,
			expCreateLoadBalancerFound:    true,
			expConfigureLoadBalancerFound: true,
			expDeleteOutboundSgRule:       true,
			expCreateLoadBalancerTag:      true,
			expDeleteOutboundSgRuleErr:    nil,
			expDescribeLoadBalancerErr:    nil,
			expCreateLoadBalancerErr:      nil,
			expConfigureLoadBalancerErr:   nil,
			expCreateLoadbalancerTagErr:   nil,
			expReconcileLoadBalancerErr:   nil,
		},
		{
			name:                          "create loadBalancer (first time reconcile loop)",
			spec:                          defaultLoadBalancerInitialize,
			expLoadBalancerFound:          false,
			expSubnetFound:                true,
			expSecurityGroupFound:         true,
			expCreateLoadBalancerFound:    true,
			expConfigureLoadBalancerFound: true,
			expDeleteOutboundSgRule:       true,
			expCreateLoadBalancerTag:      true,
			expDeleteOutboundSgRuleErr:    nil,
			expDescribeLoadBalancerErr:    nil,
			expCreateLoadBalancerErr:      nil,
			expConfigureLoadBalancerErr:   nil,
			expCreateLoadbalancerTagErr:   errors.New("CreateLoadbalancerTag generic error"),
			expReconcileLoadBalancerErr:   errors.New("CreateLoadbalancerTag generic error Can not tag loadBalancer for OscCluster test-system/test-osc"),
		},
		{
			name:                          "failed to delete outbound Sg for loadBalancer",
			spec:                          defaultLoadBalancerInitialize,
			expLoadBalancerFound:          false,
			expSubnetFound:                true,
			expSecurityGroupFound:         true,
			expCreateLoadBalancerFound:    true,
			expConfigureLoadBalancerFound: false,
			expDeleteOutboundSgRule:       false,
			expCreateLoadBalancerTag:      false,
			expDeleteOutboundSgRuleErr:    errors.New("DeleteSecurityGroupsRules generic error"),
			expDescribeLoadBalancerErr:    nil,
			expCreateLoadBalancerErr:      nil,
			expConfigureLoadBalancerErr:   nil,
			expCreateLoadbalancerTagErr:   nil,
			expReconcileLoadBalancerErr:   errors.New("DeleteSecurityGroupsRules generic error can not empty Outbound sg rules for loadBalancer for Osccluster test-system/test-osc"),
		},
		{
			name:                          "failed to configure loadBalancer",
			spec:                          defaultLoadBalancerInitialize,
			expLoadBalancerFound:          false,
			expSubnetFound:                true,
			expSecurityGroupFound:         true,
			expCreateLoadBalancerFound:    true,
			expConfigureLoadBalancerFound: false,
			expDeleteOutboundSgRule:       true,
			expCreateLoadBalancerTag:      false,
			expDeleteOutboundSgRuleErr:    nil,
			expDescribeLoadBalancerErr:    nil,
			expCreateLoadBalancerErr:      nil,
			expCreateLoadbalancerTagErr:   nil,
			expConfigureLoadBalancerErr:   errors.New("ConfigureLoadBalancer generic error"),
			expReconcileLoadBalancerErr:   errors.New("ConfigureLoadBalancer generic error Can not configure healthcheck for Osccluster test-system/test-osc"),
		},
	}
	for _, lbtc := range loadBalancerTestCases {
		t.Run(lbtc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface := SetupWithLoadBalancerMock(t, lbtc.name, lbtc.spec)
			loadBalancerName := lbtc.spec.Network.LoadBalancer.LoadBalancerName + "-uid"
			loadBalancerDnsName := loadBalancerName + "." + "eu-west-2" + "." + ".lbu.outscale.com"
			loadBalancerSpec := lbtc.spec.Network.LoadBalancer
			subnetRef := clusterScope.GetSubnetRef()
			subnetRef.ResourceMap = make(map[string]string)
			subnetName := lbtc.spec.Network.LoadBalancer.SubnetName + "-uid"
			subnetId := "subnet-" + subnetName
			if lbtc.expSubnetFound {
				subnetRef.ResourceMap[subnetName] = subnetId
			}

			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			securityGroupName := lbtc.spec.Network.LoadBalancer.SecurityGroupName + "-uid"
			securityGroupId := "sg-" + securityGroupName

			if lbtc.expSecurityGroupFound {
				securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
			}

			loadBalancer := osc.CreateLoadBalancerResponse{
				LoadBalancer: &osc.LoadBalancer{
					LoadBalancerName: &loadBalancerName,
					DnsName:          &loadBalancerDnsName,
				},
			}
			readLoadBalancers := osc.ReadLoadBalancersResponse{
				LoadBalancers: &[]osc.LoadBalancer{
					*loadBalancer.LoadBalancer,
				},
			}

			readLoadBalancer := *readLoadBalancers.LoadBalancers
			if lbtc.expLoadBalancerFound {
				mockOscLoadBalancerInterface.
					EXPECT().
					GetLoadBalancer(gomock.Eq(&loadBalancerSpec)).
					Return(&readLoadBalancer[0], lbtc.expDescribeLoadBalancerErr)
			} else {
				mockOscLoadBalancerInterface.
					EXPECT().
					GetLoadBalancer(gomock.Eq(&loadBalancerSpec)).
					Return(nil, lbtc.expDescribeLoadBalancerErr)
			}
			if lbtc.expCreateLoadBalancerFound {
				mockOscLoadBalancerInterface.
					EXPECT().
					CreateLoadBalancer(gomock.Eq(&loadBalancerSpec), gomock.Eq(subnetId), gomock.Eq(securityGroupId)).
					Return(loadBalancer.LoadBalancer, lbtc.expCreateLoadBalancerErr)
			} else {
				mockOscLoadBalancerInterface.
					EXPECT().
					CreateLoadBalancer(gomock.Eq(&loadBalancerSpec), gomock.Eq(subnetId), gomock.Eq(securityGroupId)).
					Return(nil, lbtc.expCreateLoadBalancerErr)
			}

			if lbtc.expCreateLoadBalancerErr == nil {
				if lbtc.expDeleteOutboundSgRuleErr == nil {
					mockOscSecurityGroupInterface.
						EXPECT().
						DeleteSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq("Outbound"), gomock.Eq("-1"), gomock.Eq("0.0.0.0/0"), gomock.Eq(""), gomock.Eq(int32(0)), gomock.Eq(int32(0))).
						Return(lbtc.expDeleteOutboundSgRuleErr)
					if lbtc.expConfigureLoadBalancerFound {
						mockOscLoadBalancerInterface.
							EXPECT().
							ConfigureHealthCheck(gomock.Eq(&loadBalancerSpec)).
							Return(loadBalancer.LoadBalancer, lbtc.expConfigureLoadBalancerErr)
					} else {
						mockOscLoadBalancerInterface.
							EXPECT().
							ConfigureHealthCheck(gomock.Eq(&loadBalancerSpec)).
							Return(nil, lbtc.expConfigureLoadBalancerErr)
					}
				} else {
					mockOscSecurityGroupInterface.
						EXPECT().
						DeleteSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq("Outbound"), gomock.Eq("-1"), gomock.Eq("0.0.0.0/0"), gomock.Eq(""), gomock.Eq(int32(0)), gomock.Eq(int32(0))).
						Return(lbtc.expDeleteOutboundSgRuleErr)
				}
				if lbtc.expCreateLoadBalancerTag {
					name := "test-loadbalancer-uid"
					nameTag := osc.ResourceTag{
						Key:   "Name",
						Value: name,
					}

					mockOscLoadBalancerInterface.
						EXPECT().
						CreateLoadBalancerTag(gomock.Eq(&loadBalancerSpec), gomock.Eq(nameTag)).
						Return(lbtc.expCreateLoadbalancerTagErr)
				}
			}

			reconcileLoadBalancer, err := reconcileLoadBalancer(ctx, clusterScope, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface)
			if lbtc.expReconcileLoadBalancerErr != nil {
				require.EqualError(t, err, lbtc.expReconcileLoadBalancerErr.Error(), "reconcileLoadBalancer() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("find reconcileLoadBalancer %v\n", reconcileLoadBalancer)
		})
	}
}

// TestReconcileLoadBalancerGet has several tests to cover the code of the function reconcileLoadBalancer
func TestReconcileLoadBalancerGet(t *testing.T) {
	loadBalancerTestCases := []struct {
		name                          string
		spec                          infrastructurev1beta1.OscClusterSpec
		expLoadBalancerFound          bool
		expSubnetFound                bool
		expSecurityGroupFound         bool
		expConfigureLoadBalancerFound bool
		expGetLoadBalancerTagFound    bool
		expCreateLoadBalancerTagFound bool
		expGetBadLoadBalancerTagFound bool
		expCreateLoadBalancerErr      error
		expDescribeLoadBalancerErr    error
		expConfigureLoadBalancerErr   error
		expGetLoadBalancerTagErr      error
		expReconcileLoadBalancerErr   error
	}{
		{
			name:                          "check loadBalancer exist (second time reconcile loop)",
			spec:                          defaultLoadBalancerReconcile,
			expLoadBalancerFound:          true,
			expSubnetFound:                true,
			expSecurityGroupFound:         true,
			expConfigureLoadBalancerFound: false,
			expCreateLoadBalancerTagFound: false,
			expGetLoadBalancerTagFound:    true,
			expGetBadLoadBalancerTagFound: false,
			expDescribeLoadBalancerErr:    nil,
			expCreateLoadBalancerErr:      nil,
			expConfigureLoadBalancerErr:   nil,
			expGetLoadBalancerTagErr:      nil,
			expReconcileLoadBalancerErr:   nil,
		},
		{
			name:                          "failed to get loadBalancer",
			spec:                          defaultLoadBalancerInitialize,
			expLoadBalancerFound:          false,
			expSubnetFound:                false,
			expSecurityGroupFound:         false,
			expConfigureLoadBalancerFound: false,
			expCreateLoadBalancerTagFound: false,
			expGetLoadBalancerTagFound:    false,
			expGetBadLoadBalancerTagFound: false,
			expDescribeLoadBalancerErr:    errors.New("GetLoadBalancer generic error"),
			expCreateLoadBalancerErr:      nil,
			expConfigureLoadBalancerErr:   nil,
			expGetLoadBalancerTagErr:      nil,
			expReconcileLoadBalancerErr:   errors.New("GetLoadBalancer generic error"),
		},
	}

	for _, lbtc := range loadBalancerTestCases {
		t.Run(lbtc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface := SetupWithLoadBalancerMock(t, lbtc.name, lbtc.spec)
			loadBalancerName := lbtc.spec.Network.LoadBalancer.LoadBalancerName
			loadBalancerDnsName := loadBalancerName + "." + "eu-west-2" + "." + ".lbu.outscale.com"
			loadBalancerSpec := lbtc.spec.Network.LoadBalancer

			subnetRef := clusterScope.GetSubnetRef()
			subnetRef.ResourceMap = make(map[string]string)
			subnetName := lbtc.spec.Network.LoadBalancer.SubnetName + "-uid"
			subnetId := "subnet-" + subnetName
			if lbtc.expSubnetFound {
				subnetRef.ResourceMap[subnetName] = subnetId
			}

			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			securityGroupName := lbtc.spec.Network.LoadBalancer.SecurityGroupName + "-uid"
			securityGroupId := "sg-" + securityGroupName

			if lbtc.expSecurityGroupFound {
				securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
			}

			loadBalancer := osc.CreateLoadBalancerResponse{
				LoadBalancer: &osc.LoadBalancer{
					LoadBalancerName: &loadBalancerName,
					DnsName:          &loadBalancerDnsName,
				},
			}
			readLoadBalancers := osc.ReadLoadBalancersResponse{
				LoadBalancers: &[]osc.LoadBalancer{
					*loadBalancer.LoadBalancer,
				},
			}
			loadBalancerKey := "Name"
			loadBalancerValue := loadBalancerName + "-uid"
			tag := osc.LoadBalancerTag{
				Key:              &loadBalancerKey,
				LoadBalancerName: &loadBalancerName,
				Value:            &loadBalancerValue,
			}
			readLoadBalancerTags := osc.ReadLoadBalancerTagsResponse{
				Tags: &[]osc.LoadBalancerTag{
					tag,
				},
			}
			readLoadBalancer := *readLoadBalancers.LoadBalancers
			readLoadBalancerTag := *readLoadBalancerTags.Tags
			if lbtc.expLoadBalancerFound {
				mockOscLoadBalancerInterface.
					EXPECT().
					GetLoadBalancer(gomock.Eq(&loadBalancerSpec)).
					Return(&readLoadBalancer[0], lbtc.expDescribeLoadBalancerErr)
				if lbtc.expGetLoadBalancerTagFound {
					mockOscLoadBalancerInterface.
						EXPECT().
						GetLoadBalancerTag(gomock.Eq(&loadBalancerSpec)).
						Return(&readLoadBalancerTag[0], lbtc.expGetLoadBalancerTagErr)
				}
			} else {
				mockOscLoadBalancerInterface.
					EXPECT().
					GetLoadBalancer(gomock.Eq(&loadBalancerSpec)).
					Return(nil, lbtc.expDescribeLoadBalancerErr)
			}
			reconcileLoadBalancer, err := reconcileLoadBalancer(ctx, clusterScope, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface)
			if lbtc.expReconcileLoadBalancerErr != nil {
				require.EqualError(t, err, lbtc.expReconcileLoadBalancerErr.Error(), "reconcileLoadBalancer() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("find reconcileLoadBalancer %v\n", reconcileLoadBalancer)
		})
	}
}

// TestReconcileLoadBalancerGetTag has several tests to cover the code of the function reconcileLoadBalancer
func TestReconcileLoadBalancerGetTag(t *testing.T) {
	loadBalancerTestCases := []struct {
		name                          string
		spec                          infrastructurev1beta1.OscClusterSpec
		expLoadBalancerFound          bool
		expSubnetFound                bool
		expSecurityGroupFound         bool
		expCreateLoadBalancerFound    bool
		expConfigureLoadBalancerFound bool
		expGetLoadBalancerTagFound    bool
		expCreateLoadBalancerTagFound bool
		expGetBadLoadBalancerTagFound bool
		expCreateLoadBalancerErr      error
		expDescribeLoadBalancerErr    error
		expConfigureLoadBalancerErr   error
		expGetLoadBalancerTagErr      error
		expCreateLoadBalancerTagErr   error
		expReconcileLoadBalancerErr   error
	}{
		{
			name:                          "failed to get loadBalancer Tag",
			spec:                          defaultLoadBalancerReconcile,
			expLoadBalancerFound:          true,
			expSubnetFound:                true,
			expSecurityGroupFound:         true,
			expCreateLoadBalancerFound:    false,
			expConfigureLoadBalancerFound: false,
			expCreateLoadBalancerTagFound: false,
			expGetLoadBalancerTagFound:    true,
			expGetBadLoadBalancerTagFound: false,
			expDescribeLoadBalancerErr:    nil,
			expCreateLoadBalancerErr:      nil,
			expConfigureLoadBalancerErr:   nil,
			expGetLoadBalancerTagErr:      errors.New("GetLoadBalancerTag generic error"),
			expCreateLoadBalancerTagErr:   nil,
			expReconcileLoadBalancerErr:   errors.New("GetLoadBalancerTag generic error"),
		},
		{
			name:                          "a loadBalancer with the same name already exists without tag",
			spec:                          defaultLoadBalancerReconcile,
			expLoadBalancerFound:          true,
			expSubnetFound:                true,
			expSecurityGroupFound:         true,
			expCreateLoadBalancerFound:    false,
			expConfigureLoadBalancerFound: false,
			expCreateLoadBalancerTagFound: false,
			expGetLoadBalancerTagFound:    false,
			expGetBadLoadBalancerTagFound: false,
			expDescribeLoadBalancerErr:    nil,
			expCreateLoadBalancerErr:      nil,
			expConfigureLoadBalancerErr:   nil,
			expGetLoadBalancerTagErr:      nil,
			expCreateLoadBalancerTagErr:   nil,
			expReconcileLoadBalancerErr:   errors.New("A LoadBalancer test-loadbalancer already exists"),
		},
		{
			name:                          "a loadBalancer with the same name in other cluster already exists",
			spec:                          defaultLoadBalancerReconcile,
			expLoadBalancerFound:          true,
			expSubnetFound:                true,
			expSecurityGroupFound:         true,
			expCreateLoadBalancerFound:    false,
			expConfigureLoadBalancerFound: false,
			expCreateLoadBalancerTagFound: false,
			expGetLoadBalancerTagFound:    true,
			expGetBadLoadBalancerTagFound: true,
			expDescribeLoadBalancerErr:    nil,
			expCreateLoadBalancerErr:      nil,
			expConfigureLoadBalancerErr:   nil,
			expGetLoadBalancerTagErr:      nil,
			expCreateLoadBalancerTagErr:   nil,
			expReconcileLoadBalancerErr:   errors.New("A LoadBalancer test-loadbalancer already exists that is used by another cluster other than uid"),
		},
	}

	for _, lbtc := range loadBalancerTestCases {
		t.Run(lbtc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface := SetupWithLoadBalancerMock(t, lbtc.name, lbtc.spec)
			loadBalancerName := lbtc.spec.Network.LoadBalancer.LoadBalancerName
			loadBalancerDnsName := loadBalancerName + "." + "eu-west-2" + "." + ".lbu.outscale.com"
			loadBalancerSpec := lbtc.spec.Network.LoadBalancer

			subnetRef := clusterScope.GetSubnetRef()
			subnetRef.ResourceMap = make(map[string]string)
			subnetName := lbtc.spec.Network.LoadBalancer.SubnetName + "-uid"
			subnetId := "subnet-" + subnetName
			if lbtc.expSubnetFound {
				subnetRef.ResourceMap[subnetName] = subnetId
			}

			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			securityGroupName := lbtc.spec.Network.LoadBalancer.SecurityGroupName + "-uid"
			securityGroupId := "sg-" + securityGroupName

			if lbtc.expSecurityGroupFound {
				securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
			}

			loadBalancer := osc.CreateLoadBalancerResponse{
				LoadBalancer: &osc.LoadBalancer{
					LoadBalancerName: &loadBalancerName,
					DnsName:          &loadBalancerDnsName,
				},
			}
			readLoadBalancers := osc.ReadLoadBalancersResponse{
				LoadBalancers: &[]osc.LoadBalancer{
					*loadBalancer.LoadBalancer,
				},
			}
			loadBalancerKey := "Name"
			var loadBalancerValue string
			if lbtc.expGetBadLoadBalancerTagFound {
				loadBalancerValue = loadBalancerName + "-other-uid"
			} else {
				loadBalancerValue = loadBalancerName + "-uid"
			}
			tag := osc.LoadBalancerTag{
				Key:              &loadBalancerKey,
				LoadBalancerName: &loadBalancerName,
				Value:            &loadBalancerValue,
			}
			readLoadBalancerTags := osc.ReadLoadBalancerTagsResponse{
				Tags: &[]osc.LoadBalancerTag{
					tag,
				},
			}
			readLoadBalancer := *readLoadBalancers.LoadBalancers
			readLoadBalancerTag := *readLoadBalancerTags.Tags
			if lbtc.expLoadBalancerFound {
				mockOscLoadBalancerInterface.
					EXPECT().
					GetLoadBalancer(gomock.Eq(&loadBalancerSpec)).
					Return(&readLoadBalancer[0], lbtc.expDescribeLoadBalancerErr)
				if lbtc.expGetLoadBalancerTagFound {
					mockOscLoadBalancerInterface.
						EXPECT().
						GetLoadBalancerTag(gomock.Eq(&loadBalancerSpec)).
						Return(&readLoadBalancerTag[0], lbtc.expGetLoadBalancerTagErr)
				} else {
					mockOscLoadBalancerInterface.
						EXPECT().
						GetLoadBalancerTag(gomock.Eq(&loadBalancerSpec)).
						Return(nil, lbtc.expGetLoadBalancerTagErr)
				}
			} else {
				mockOscLoadBalancerInterface.
					EXPECT().
					GetLoadBalancer(gomock.Eq(&loadBalancerSpec)).
					Return(nil, lbtc.expDescribeLoadBalancerErr)
			}
			reconcileLoadBalancer, err := reconcileLoadBalancer(ctx, clusterScope, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface)
			if lbtc.expReconcileLoadBalancerErr != nil {
				require.EqualError(t, err, lbtc.expReconcileLoadBalancerErr.Error(), "reconcileLoadBalancer() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("find reconcileLoadBalancer %v\n", reconcileLoadBalancer)
		})
	}
}

// TestReconcileLoadBalancerCreate has several tests to cover the code of the function reconcileLoadBalancer

func TestReconcileLoadBalancerCreate(t *testing.T) {
	loadBalancerTestCases := []struct {
		name                        string
		spec                        infrastructurev1beta1.OscClusterSpec
		expCreateLoadBalancerErr    error
		expConfigureLoadBalancerErr error
		expDescribeLoadBalancerErr  error
		expReconcileLoadBalancerErr error
	}{
		{
			name:                        "failed to create loadBalancer",
			spec:                        defaultLoadBalancerInitialize,
			expCreateLoadBalancerErr:    errors.New("CreateLoadBalancer generic error"),
			expConfigureLoadBalancerErr: nil,
			expDescribeLoadBalancerErr:  nil,
			expReconcileLoadBalancerErr: errors.New("CreateLoadBalancer generic error Can not create loadBalancer for Osccluster test-system/test-osc"),
		},
	}
	for _, lbtc := range loadBalancerTestCases {
		t.Run(lbtc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface := SetupWithLoadBalancerMock(t, lbtc.name, lbtc.spec)

			subnetRef := clusterScope.GetSubnetRef()
			subnetRef.ResourceMap = make(map[string]string)
			subnetName := lbtc.spec.Network.LoadBalancer.SubnetName + "-uid"
			subnetId := "subnet-" + subnetName
			subnetRef.ResourceMap[subnetName] = subnetId

			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			securityGroupName := lbtc.spec.Network.LoadBalancer.SecurityGroupName + "-uid"
			securityGroupId := "sg-" + securityGroupName
			securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId

			loadBalancerSpec := lbtc.spec.Network.LoadBalancer
			mockOscLoadBalancerInterface.
				EXPECT().
				GetLoadBalancer(gomock.Eq(&loadBalancerSpec)).
				Return(nil, lbtc.expDescribeLoadBalancerErr)
			mockOscLoadBalancerInterface.
				EXPECT().
				CreateLoadBalancer(gomock.Eq(&loadBalancerSpec), gomock.Eq(subnetId), gomock.Eq(securityGroupId)).
				Return(nil, lbtc.expCreateLoadBalancerErr)
			reconcileLoadBalancer, err := reconcileLoadBalancer(ctx, clusterScope, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface)
			if lbtc.expReconcileLoadBalancerErr != nil {
				require.EqualError(t, err, lbtc.expReconcileLoadBalancerErr.Error(), "reconcileLoadBalancer() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("find reconcileLoadBalancer %v\n", reconcileLoadBalancer)
		})
	}
}

// TestReconcileLoadBalancerResourceId has several tests to cover the code of the function reconcileLoadBalancer
func TestReconcileLoadBalancerResourceId(t *testing.T) {
	loadBalancerTestCases := []struct {
		name                        string
		spec                        infrastructurev1beta1.OscClusterSpec
		expSubnetFound              bool
		expSecurityGroupFound       bool
		expDescribeLoadBalancerErr  error
		expReconcileLoadBalancerErr error
	}{
		{
			name:                        "subnet does not exist",
			spec:                        defaultLoadBalancerInitialize,
			expSubnetFound:              false,
			expSecurityGroupFound:       false,
			expDescribeLoadBalancerErr:  nil,
			expReconcileLoadBalancerErr: errors.New("test-subnet-uid does not exist"),
		},
		{
			name:                        "securitygroup does not exist",
			spec:                        defaultLoadBalancerInitialize,
			expSubnetFound:              true,
			expSecurityGroupFound:       false,
			expDescribeLoadBalancerErr:  nil,
			expReconcileLoadBalancerErr: errors.New("test-securitygroup-uid does not exist"),
		},
	}
	for _, lbtc := range loadBalancerTestCases {
		t.Run(lbtc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface := SetupWithLoadBalancerMock(t, lbtc.name, lbtc.spec)

			subnetRef := clusterScope.GetSubnetRef()
			subnetRef.ResourceMap = make(map[string]string)
			subnetName := lbtc.spec.Network.LoadBalancer.SubnetName + "-uid"
			subnetId := "subnet-" + subnetName
			if lbtc.expSubnetFound {
				subnetRef.ResourceMap[subnetName] = subnetId
			}

			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			securityGroupName := lbtc.spec.Network.LoadBalancer.SecurityGroupName + "-uid"
			securityGroupId := "sg-" + securityGroupName

			if lbtc.expSecurityGroupFound {
				securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
			}

			loadBalancerSpec := lbtc.spec.Network.LoadBalancer
			mockOscLoadBalancerInterface.
				EXPECT().
				GetLoadBalancer(gomock.Eq(&loadBalancerSpec)).
				Return(nil, lbtc.expDescribeLoadBalancerErr)
			reconcileLoadBalancer, err := reconcileLoadBalancer(ctx, clusterScope, mockOscLoadBalancerInterface, mockOscSecurityGroupInterface)
			if lbtc.expReconcileLoadBalancerErr != nil {
				require.EqualError(t, err, lbtc.expReconcileLoadBalancerErr.Error(), "reconcileLoadBalancer() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("find reconcileLoadBalancer %v\n", reconcileLoadBalancer)
		})
	}
}

// TestReconcileDeleteLoadBalancerDelete  has several tests to cover the code of the function ReconcileDeleteLoadBalancer
func TestReconcileDeleteLoadBalancerDelete(t *testing.T) {
	loadBalancerTestCases := []struct {
		name                                string
		spec                                infrastructurev1beta1.OscClusterSpec
		expLoadBalancerFound                bool
		expGetLoadBalancerTagFound          bool
		expDeleteLoadBalancerTagFound       bool
		expDescribeLoadBalancerErr          error
		expDeleteLoadBalancerErr            error
		expCheckLoadBalancerDeregisterVmErr error
		expGetLoadBalancerTagErr            error
		expDeleteLoadBalancerTagErr         error
		expReconcileDeleteLoadBalancerErr   error
	}{
		{
			name:                                "delete loadBalancer (first time reconcile loop)",
			spec:                                defaultLoadBalancerReconcile,
			expLoadBalancerFound:                true,
			expDeleteLoadBalancerTagFound:       true,
			expGetLoadBalancerTagFound:          true,
			expDeleteLoadBalancerErr:            nil,
			expDescribeLoadBalancerErr:          nil,
			expCheckLoadBalancerDeregisterVmErr: nil,
			expGetLoadBalancerTagErr:            nil,
			expDeleteLoadBalancerTagErr:         nil,
			expReconcileDeleteLoadBalancerErr:   nil,
		},
		{
			name:                                "failed to delete loadBalancer",
			spec:                                defaultLoadBalancerReconcile,
			expLoadBalancerFound:                true,
			expGetLoadBalancerTagFound:          true,
			expDeleteLoadBalancerTagFound:       true,
			expDeleteLoadBalancerErr:            errors.New("DeleteLoadBalancer generic error"),
			expDescribeLoadBalancerErr:          nil,
			expCheckLoadBalancerDeregisterVmErr: nil,
			expGetLoadBalancerTagErr:            nil,
			expDeleteLoadBalancerTagErr:         nil,
			expReconcileDeleteLoadBalancerErr:   errors.New("DeleteLoadBalancer generic error Can not delete loadBalancer for Osccluster test-system/test-osc"),
		},
	}
	for _, lbtc := range loadBalancerTestCases {
		t.Run(lbtc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscLoadBalancerInterface, _ := SetupWithLoadBalancerMock(t, lbtc.name, lbtc.spec)
			loadBalancerName := lbtc.spec.Network.LoadBalancer.LoadBalancerName
			loadBalancerDnsName := loadBalancerName + ".eu-west-2.lbu.outscale.com"
			loadBalancerSpec := lbtc.spec.Network.LoadBalancer
			loadBalancerSpec.SetDefaultValue()
			loadBalancer := osc.CreateLoadBalancerResponse{
				LoadBalancer: &osc.LoadBalancer{
					LoadBalancerName: &loadBalancerName,
					DnsName:          &loadBalancerDnsName,
				},
			}
			readLoadBalancers := osc.ReadLoadBalancersResponse{
				LoadBalancers: &[]osc.LoadBalancer{
					*loadBalancer.LoadBalancer,
				},
			}
			loadBalancerKey := "Name"
			loadBalancerValue := loadBalancerName + "-uid"
			tag := osc.LoadBalancerTag{
				Key:              &loadBalancerKey,
				LoadBalancerName: &loadBalancerName,
				Value:            &loadBalancerValue,
			}
			readLoadBalancerTags := osc.ReadLoadBalancerTagsResponse{
				Tags: &[]osc.LoadBalancerTag{
					tag,
				},
			}
			readLoadBalancer := *readLoadBalancers.LoadBalancers
			readLoadBalancerTag := *readLoadBalancerTags.Tags
			var clockInsideLoop time.Duration = 20
			var clockLoop time.Duration = 120
			if lbtc.expGetLoadBalancerTagFound {
				mockOscLoadBalancerInterface.
					EXPECT().
					GetLoadBalancerTag(gomock.Eq(&loadBalancerSpec)).
					Return(&readLoadBalancerTag[0], lbtc.expGetLoadBalancerTagErr)
			}
			if lbtc.expDeleteLoadBalancerTagFound {
				loadBalancerTagKey := osc.ResourceLoadBalancerTag{
					Key: *tag.Key,
				}
				mockOscLoadBalancerInterface.
					EXPECT().
					DeleteLoadBalancerTag(gomock.Eq(&loadBalancerSpec), gomock.Eq(loadBalancerTagKey)).
					Return(lbtc.expDeleteLoadBalancerTagErr)
			}
			mockOscLoadBalancerInterface.
				EXPECT().
				GetLoadBalancer(gomock.Eq(&loadBalancerSpec)).
				Return(&readLoadBalancer[0], lbtc.expDescribeLoadBalancerErr)
			mockOscLoadBalancerInterface.
				EXPECT().
				DeleteLoadBalancer(gomock.Eq(&loadBalancerSpec)).
				Return(lbtc.expDeleteLoadBalancerErr)

			mockOscLoadBalancerInterface.
				EXPECT().
				CheckLoadBalancerDeregisterVm(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(&loadBalancerSpec)).
				Return(lbtc.expCheckLoadBalancerDeregisterVmErr)

			reconcileDeleteLoadBalancer, err := reconcileDeleteLoadBalancer(ctx, clusterScope, mockOscLoadBalancerInterface)
			if lbtc.expReconcileDeleteLoadBalancerErr != nil {
				require.EqualError(t, err, lbtc.expReconcileDeleteLoadBalancerErr.Error(), "reconcileDeleteLoadBalancer() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("find reconcileDeleteLoadBalancer %v\n", reconcileDeleteLoadBalancer)
		})
	}
}

// TestReconcileDeleteLoadBalancerDeleteTag  has several tests to cover the code of the function ReconcileDeleteLoadBalancer
func TestReconcileDeleteLoadBalancerDeleteTag(t *testing.T) {
	loadBalancerTestCases := []struct {
		name                                  string
		spec                                  infrastructurev1beta1.OscClusterSpec
		expLoadBalancerFound                  bool
		expGetLoadBalancerTagFound            bool
		expDeleteLoadBalancerTagFound         bool
		expGetBadLoadBalancerTagFound         bool
		expCheckLoadBalancerDeregisterVmFound bool
		expDescribeLoadBalancerErr            error
		expDeleteLoadBalancerErr              error
		expCheckLoadBalancerDeregisterVmErr   error
		expGetLoadBalancerTagErr              error
		expDeleteLoadBalancerTagErr           error
		expReconcileDeleteLoadBalancerErr     error
	}{

		{
			name:                                  "failed to get loadBalancer Tag",
			spec:                                  defaultLoadBalancerReconcile,
			expLoadBalancerFound:                  true,
			expDeleteLoadBalancerTagFound:         false,
			expGetBadLoadBalancerTagFound:         false,
			expGetLoadBalancerTagFound:            false,
			expCheckLoadBalancerDeregisterVmFound: false,
			expDeleteLoadBalancerErr:              nil,
			expDescribeLoadBalancerErr:            nil,
			expCheckLoadBalancerDeregisterVmErr:   nil,
			expGetLoadBalancerTagErr:              errors.New("GetLoadBalancerTag generic error"),
			expDeleteLoadBalancerTagErr:           nil,
			expReconcileDeleteLoadBalancerErr:     errors.New("GetLoadBalancerTag generic error"),
		},
		{
			name:                                  "a loadBalancer with the same name in other cluster already exists",
			spec:                                  defaultLoadBalancerReconcile,
			expLoadBalancerFound:                  true,
			expDeleteLoadBalancerTagFound:         false,
			expGetBadLoadBalancerTagFound:         true,
			expGetLoadBalancerTagFound:            true,
			expCheckLoadBalancerDeregisterVmFound: false,
			expDeleteLoadBalancerErr:              nil,
			expDescribeLoadBalancerErr:            nil,
			expCheckLoadBalancerDeregisterVmErr:   nil,
			expGetLoadBalancerTagErr:              nil,
			expDeleteLoadBalancerTagErr:           nil,
			expReconcileDeleteLoadBalancerErr:     nil,
		},
		{
			name:                                  "failed to delete loadBalancer tag",
			spec:                                  defaultLoadBalancerReconcile,
			expLoadBalancerFound:                  true,
			expDeleteLoadBalancerTagFound:         true,
			expGetBadLoadBalancerTagFound:         false,
			expCheckLoadBalancerDeregisterVmFound: true,
			expGetLoadBalancerTagFound:            true,
			expDeleteLoadBalancerErr:              nil,
			expDescribeLoadBalancerErr:            nil,
			expCheckLoadBalancerDeregisterVmErr:   nil,
			expGetLoadBalancerTagErr:              nil,
			expDeleteLoadBalancerTagErr:           errors.New("DeleteLoadBalancerTag generic error"),
			expReconcileDeleteLoadBalancerErr:     errors.New("DeleteLoadBalancerTag generic error Can not delete loadBalancer Tag for OscCluster test-system/test-osc"),
		},
	}
	for _, lbtc := range loadBalancerTestCases {
		t.Run(lbtc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscLoadBalancerInterface, _ := SetupWithLoadBalancerMock(t, lbtc.name, lbtc.spec)
			loadBalancerName := lbtc.spec.Network.LoadBalancer.LoadBalancerName
			loadBalancerDnsName := loadBalancerName + ".eu-west-2.lbu.outscale.com"
			loadBalancerSpec := lbtc.spec.Network.LoadBalancer
			loadBalancerSpec.SetDefaultValue()
			loadBalancer := osc.CreateLoadBalancerResponse{
				LoadBalancer: &osc.LoadBalancer{
					LoadBalancerName: &loadBalancerName,
					DnsName:          &loadBalancerDnsName,
				},
			}
			readLoadBalancers := osc.ReadLoadBalancersResponse{
				LoadBalancers: &[]osc.LoadBalancer{
					*loadBalancer.LoadBalancer,
				},
			}
			loadBalancerKey := "Name"
			var loadBalancerValue string
			if lbtc.expGetBadLoadBalancerTagFound {
				loadBalancerValue = loadBalancerName + "-other-uid"
			} else {
				loadBalancerValue = loadBalancerName + "-uid"
			}
			tag := osc.LoadBalancerTag{
				Key:              &loadBalancerKey,
				LoadBalancerName: &loadBalancerName,
				Value:            &loadBalancerValue,
			}
			readLoadBalancerTags := osc.ReadLoadBalancerTagsResponse{
				Tags: &[]osc.LoadBalancerTag{
					tag,
				},
			}
			readLoadBalancer := *readLoadBalancers.LoadBalancers
			readLoadBalancerTag := *readLoadBalancerTags.Tags
			var clockInsideLoop time.Duration = 20
			var clockLoop time.Duration = 120
			if lbtc.expGetLoadBalancerTagFound {
				mockOscLoadBalancerInterface.
					EXPECT().
					GetLoadBalancerTag(gomock.Eq(&loadBalancerSpec)).
					Return(&readLoadBalancerTag[0], lbtc.expGetLoadBalancerTagErr)
			} else {
				mockOscLoadBalancerInterface.
					EXPECT().
					GetLoadBalancerTag(gomock.Eq(&loadBalancerSpec)).
					Return(nil, lbtc.expGetLoadBalancerTagErr)
			}
			if lbtc.expDeleteLoadBalancerTagFound {
				loadBalancerTagKey := osc.ResourceLoadBalancerTag{
					Key: *tag.Key,
				}
				mockOscLoadBalancerInterface.
					EXPECT().
					DeleteLoadBalancerTag(gomock.Eq(&loadBalancerSpec), gomock.Eq(loadBalancerTagKey)).
					Return(lbtc.expDeleteLoadBalancerTagErr)
			}
			mockOscLoadBalancerInterface.
				EXPECT().
				GetLoadBalancer(gomock.Eq(&loadBalancerSpec)).
				Return(&readLoadBalancer[0], lbtc.expDescribeLoadBalancerErr)
			if lbtc.expCheckLoadBalancerDeregisterVmFound {
				mockOscLoadBalancerInterface.
					EXPECT().
					CheckLoadBalancerDeregisterVm(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(&loadBalancerSpec)).
					Return(lbtc.expCheckLoadBalancerDeregisterVmErr)
			}
			reconcileDeleteLoadBalancer, err := reconcileDeleteLoadBalancer(ctx, clusterScope, mockOscLoadBalancerInterface)
			if lbtc.expReconcileDeleteLoadBalancerErr != nil {
				require.EqualError(t, err, lbtc.expReconcileDeleteLoadBalancerErr.Error(), "reconcileDeleteLoadBalancer() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("find reconcileDeleteLoadBalancer %v\n", reconcileDeleteLoadBalancer)
		})
	}
}

// TestReconcileDeleteLoadBalancerDeleteWithoutSpec  has several tests to cover the code of the function reconcileDeleteLoadBalancer
func TestReconcileDeleteLoadBalancerDeleteWithoutSpec(t *testing.T) {
	loadBalancerTestCases := []struct {
		name                                  string
		spec                                  infrastructurev1beta1.OscClusterSpec
		expLoadBalancerFound                  bool
		expGetLoadBalancerTagFound            bool
		expDeleteLoadBalancerTagFound         bool
		expGetBadLoadBalancerTagFound         bool
		expCheckLoadBalancerDeregisterVmFound bool
		expDescribeLoadBalancerErr            error
		expDeleteLoadBalancerErr              error
		expCheckLoadBalancerDeregisterVmErr   error
		expGetLoadBalancerTagErr              error
		expDeleteLoadBalancerTagErr           error
		expReconcileDeleteLoadBalancerErr     error
	}{
		{
			name:                                  "delete loadBalancer without spec (with default values)",
			expLoadBalancerFound:                  true,
			expGetLoadBalancerTagFound:            true,
			expGetBadLoadBalancerTagFound:         false,
			expCheckLoadBalancerDeregisterVmFound: true,
			expDeleteLoadBalancerTagFound:         true,
			expDescribeLoadBalancerErr:            nil,
			expDeleteLoadBalancerErr:              nil,
			expGetLoadBalancerTagErr:              nil,
			expDeleteLoadBalancerTagErr:           nil,
			expCheckLoadBalancerDeregisterVmErr:   nil,
			expReconcileDeleteLoadBalancerErr:     nil,
		},
	}
	for _, lbtc := range loadBalancerTestCases {
		t.Run(lbtc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscLoadBalancerInterface, _ := SetupWithLoadBalancerMock(t, lbtc.name, lbtc.spec)
			loadBalancerName := "OscClusterApi-1"
			loadBalancerDnsName := loadBalancerName + ".eu-west-2.lbu.outscale.com"
			loadBalancerSpec := clusterScope.GetLoadBalancer()
			loadBalancerSpec.SetDefaultValue()
			loadBalancer := osc.CreateLoadBalancerResponse{
				LoadBalancer: &osc.LoadBalancer{
					LoadBalancerName: &loadBalancerName,
					DnsName:          &loadBalancerDnsName,
				},
			}
			readLoadBalancers := osc.ReadLoadBalancersResponse{
				LoadBalancers: &[]osc.LoadBalancer{
					*loadBalancer.LoadBalancer,
				},
			}
			loadBalancerKey := "Name"
			var loadBalancerValue string
			if lbtc.expGetBadLoadBalancerTagFound {
				loadBalancerValue = loadBalancerName + "-other-uid"
			} else {
				loadBalancerValue = loadBalancerName + "-uid"
			}
			tag := osc.LoadBalancerTag{
				Key:              &loadBalancerKey,
				LoadBalancerName: &loadBalancerName,
				Value:            &loadBalancerValue,
			}
			readLoadBalancerTags := osc.ReadLoadBalancerTagsResponse{
				Tags: &[]osc.LoadBalancerTag{
					tag,
				},
			}
			readLoadBalancer := *readLoadBalancers.LoadBalancers
			readLoadBalancerTag := *readLoadBalancerTags.Tags
			var clockInsideLoop time.Duration = 20
			var clockLoop time.Duration = 120
			if lbtc.expGetLoadBalancerTagFound {
				mockOscLoadBalancerInterface.
					EXPECT().
					GetLoadBalancerTag(gomock.Eq(loadBalancerSpec)).
					Return(&readLoadBalancerTag[0], lbtc.expGetLoadBalancerTagErr)
			} else {
				mockOscLoadBalancerInterface.
					EXPECT().
					GetLoadBalancerTag(gomock.Eq(loadBalancerSpec)).
					Return(nil, lbtc.expGetLoadBalancerTagErr)
			}
			if lbtc.expDeleteLoadBalancerTagFound {
				loadBalancerTagKey := osc.ResourceLoadBalancerTag{
					Key: *tag.Key,
				}
				mockOscLoadBalancerInterface.
					EXPECT().
					DeleteLoadBalancerTag(gomock.Eq(loadBalancerSpec), gomock.Eq(loadBalancerTagKey)).
					Return(lbtc.expDeleteLoadBalancerTagErr)
			}
			mockOscLoadBalancerInterface.
				EXPECT().
				GetLoadBalancer(gomock.Eq(loadBalancerSpec)).
				Return(&readLoadBalancer[0], lbtc.expDescribeLoadBalancerErr)
			if lbtc.expCheckLoadBalancerDeregisterVmFound {
				mockOscLoadBalancerInterface.
					EXPECT().
					CheckLoadBalancerDeregisterVm(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(loadBalancerSpec)).
					Return(lbtc.expCheckLoadBalancerDeregisterVmErr)
			}
			mockOscLoadBalancerInterface.
				EXPECT().
				DeleteLoadBalancer(gomock.Eq(loadBalancerSpec)).
				Return(lbtc.expDeleteLoadBalancerErr)
			reconcileDeleteLoadBalancer, err := reconcileDeleteLoadBalancer(ctx, clusterScope, mockOscLoadBalancerInterface)
			if lbtc.expReconcileDeleteLoadBalancerErr != nil {
				require.EqualError(t, err, lbtc.expReconcileDeleteLoadBalancerErr.Error(), "reconcileDeleteLoadBalancer() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("find reconcileDeleteLoadBalancer %v\n", reconcileDeleteLoadBalancer)
		})
	}
}

// TestReconcileDeleteLoadBalancerCheck  has one tests to cover the code of the function ReconcileDeleteLoadBalancer
func TestReconcileDeleteLoadBalancerCheck(t *testing.T) {
	loadBalancerTestCases := []struct {
		name                                  string
		spec                                  infrastructurev1beta1.OscClusterSpec
		expLoadBalancerFound                  bool
		expGetLoadBalancerTagFound            bool
		expCheckLoadBalancerDeregisterVmFound bool
		expGetLoadBalancerTagErr              error
		expDescribeLoadBalancerErr            error
		expCheckLoadBalancerDeregisterVmErr   error
		expReconcileDeleteLoadBalancerErr     error
	}{
		{
			name:                                  "failed to delete loadBalancer",
			spec:                                  defaultLoadBalancerReconcile,
			expLoadBalancerFound:                  true,
			expGetLoadBalancerTagFound:            true,
			expCheckLoadBalancerDeregisterVmFound: true,
			expDescribeLoadBalancerErr:            nil,
			expGetLoadBalancerTagErr:              nil,
			expCheckLoadBalancerDeregisterVmErr:   errors.New("CheckLoadBalancerDeregisterVm generic error"),
			expReconcileDeleteLoadBalancerErr:     errors.New("CheckLoadBalancerDeregisterVm generic error VmBackend is not deregister in loadBalancer test-loadbalancer for OscCluster test-system/test-osc"),
		},
	}
	for _, lbtc := range loadBalancerTestCases {
		t.Run(lbtc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscLoadBalancerInterface, _ := SetupWithLoadBalancerMock(t, lbtc.name, lbtc.spec)
			loadBalancerName := lbtc.spec.Network.LoadBalancer.LoadBalancerName
			loadBalancerDnsName := loadBalancerName + ".eu-west-2.lbu.outscale.com"
			loadBalancerSpec := lbtc.spec.Network.LoadBalancer
			loadBalancerSpec.SetDefaultValue()
			loadBalancer := osc.CreateLoadBalancerResponse{
				LoadBalancer: &osc.LoadBalancer{
					LoadBalancerName: &loadBalancerName,
					DnsName:          &loadBalancerDnsName,
				},
			}
			readLoadBalancers := osc.ReadLoadBalancersResponse{
				LoadBalancers: &[]osc.LoadBalancer{
					*loadBalancer.LoadBalancer,
				},
			}
			loadBalancerKey := "Name"
			loadBalancerValue := loadBalancerName + "-uid"
			tag := osc.LoadBalancerTag{
				Key:              &loadBalancerKey,
				LoadBalancerName: &loadBalancerName,
				Value:            &loadBalancerValue,
			}
			readLoadBalancerTags := osc.ReadLoadBalancerTagsResponse{
				Tags: &[]osc.LoadBalancerTag{
					tag,
				},
			}
			readLoadBalancer := *readLoadBalancers.LoadBalancers
			readLoadBalancerTag := *readLoadBalancerTags.Tags
			var clockInsideLoop time.Duration = 20
			var clockLoop time.Duration = 120
			if lbtc.expGetLoadBalancerTagFound {
				mockOscLoadBalancerInterface.
					EXPECT().
					GetLoadBalancerTag(gomock.Eq(&loadBalancerSpec)).
					Return(&readLoadBalancerTag[0], lbtc.expGetLoadBalancerTagErr)
			}
			mockOscLoadBalancerInterface.
				EXPECT().
				GetLoadBalancer(gomock.Eq(&loadBalancerSpec)).
				Return(&readLoadBalancer[0], lbtc.expDescribeLoadBalancerErr)

			if lbtc.expCheckLoadBalancerDeregisterVmFound {
				mockOscLoadBalancerInterface.
					EXPECT().
					CheckLoadBalancerDeregisterVm(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(&loadBalancerSpec)).
					Return(lbtc.expCheckLoadBalancerDeregisterVmErr)
			}

			reconcileDeleteLoadBalancer, err := reconcileDeleteLoadBalancer(ctx, clusterScope, mockOscLoadBalancerInterface)
			if lbtc.expReconcileDeleteLoadBalancerErr != nil {
				require.EqualError(t, err, lbtc.expReconcileDeleteLoadBalancerErr.Error(), "reconcileDeleteLoadBalancer() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("find reconcileDeleteLoadBalancer %v\n", reconcileDeleteLoadBalancer)
		})
	}
}

// TestReconcileDeleteLoadBalancerGet  has several tests to cover the code of the function reconcileDeleteLoadBalancer
func TestReconcileDeleteLoadBalancerGet(t *testing.T) {
	loadBalancerTestCases := []struct {
		name                              string
		spec                              infrastructurev1beta1.OscClusterSpec
		expLoadBalancerFound              bool
		expDescribeLoadBalancerErr        error
		expReconcileDeleteLoadBalancerErr error
	}{
		{
			name:                              "failed to get loadBalancer",
			spec:                              defaultLoadBalancerReconcile,
			expLoadBalancerFound:              false,
			expDescribeLoadBalancerErr:        errors.New("GetLoadBalancer generic error"),
			expReconcileDeleteLoadBalancerErr: errors.New("GetLoadBalancer generic error"),
		},
		{
			name:                              "remove finalizer (user delete loadBalancer without cluster-api)",
			spec:                              defaultLoadBalancerReconcile,
			expLoadBalancerFound:              false,
			expDescribeLoadBalancerErr:        nil,
			expReconcileDeleteLoadBalancerErr: nil,
		},
	}
	for _, lbtc := range loadBalancerTestCases {
		t.Run(lbtc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscLoadBalancerInterface, _ := SetupWithLoadBalancerMock(t, lbtc.name, lbtc.spec)
			loadBalancerSpec := lbtc.spec.Network.LoadBalancer
			loadBalancerSpec.SetDefaultValue()
			mockOscLoadBalancerInterface.
				EXPECT().
				GetLoadBalancer(gomock.Eq(&loadBalancerSpec)).
				Return(nil, lbtc.expDescribeLoadBalancerErr)
			reconcileDeleteLoadBalancer, err := reconcileDeleteLoadBalancer(ctx, clusterScope, mockOscLoadBalancerInterface)
			if lbtc.expReconcileDeleteLoadBalancerErr != nil {
				require.EqualError(t, err, lbtc.expReconcileDeleteLoadBalancerErr.Error(), "reconcileDeleteLoadBalancer() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("find reconcileDeleteLoadBalancer %v\n", reconcileDeleteLoadBalancer)
		})
	}
}
