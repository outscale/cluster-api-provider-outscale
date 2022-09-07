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
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/service/mock_service"
	osc "github.com/outscale/osc-sdk-go/v2"
	"github.com/stretchr/testify/assert"
)

var (
	defaultLoadBalancerInitialize = infrastructurev1beta1.OscClusterSpec{
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
		},
	}

	defaultLoadBalancerReconcile = infrastructurev1beta1.OscClusterSpec{
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
		},
	}
)

// SetupWithLoadBalancerMock set loadBalancerMock with clusterScope and osccluster
func SetupWithLoadBalancerMock(t *testing.T, name string, spec infrastructurev1beta1.OscClusterSpec) (clusterScope *scope.ClusterScope, ctx context.Context, mockOscLoadBalancerInterface *mock_service.MockOscLoadBalancerInterface) {
	clusterScope = Setup(t, name, spec)
	mockCtrl := gomock.NewController(t)
	mockOscLoadBalancerInterface = mock_service.NewMockOscLoadBalancerInterface(mockCtrl)
	ctx = context.Background()
	return clusterScope, ctx, mockOscLoadBalancerInterface
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
						SubnetName:        "test-subnet-test",
						SecurityGroupName: "test-securitygroup",
					},
				},
			},
			expCheckLoadBalancerSubnetOscAssociateResourceNameErr: fmt.Errorf("test-subnet-test-uid subnet does not exist in loadBalancer"),
		},
	}
	for _, lbtc := range loadBalancerTestCases {
		t.Run(lbtc.name, func(t *testing.T) {
			clusterScope := Setup(t, lbtc.name, lbtc.spec)
			err := checkLoadBalancerSubnetOscAssociateResourceName(clusterScope)
			if err != nil {
				assert.Equal(t, lbtc.expCheckLoadBalancerSubnetOscAssociateResourceNameErr, err, "checkLoadBalancerSubnetOscAssociateResourceName() should return the same error")
			} else {
				assert.Nil(t, lbtc.expCheckLoadBalancerSubnetOscAssociateResourceNameErr)
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
			expCheckLoadBalancerFormatParametersErr: fmt.Errorf("test-loadbalancer@test is an invalid loadBalancer name: invalid Description"),
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
			expCheckLoadBalancerFormatParametersErr: fmt.Errorf("internet is an invalid loadBalancer type: invalid LoadBalancerType"),
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
			expCheckLoadBalancerFormatParametersErr: fmt.Errorf("65537 is an invalid Port for loadBalancer backend"),
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
			expCheckLoadBalancerFormatParametersErr: fmt.Errorf("SCTP is an invalid protocol for loadBalancer backend"),
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
			expCheckLoadBalancerFormatParametersErr: fmt.Errorf("65537 is an invalid Port for loadBalancer"),
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
			expCheckLoadBalancerFormatParametersErr: fmt.Errorf("SCTP is an invalid protocol for loadBalancer"),
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
			expCheckLoadBalancerFormatParametersErr: fmt.Errorf("602 is an invalid Interval for loadBalancer"),
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
			expCheckLoadBalancerFormatParametersErr: fmt.Errorf("12 is an invalid threshold for loadBalancer"),
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
			expCheckLoadBalancerFormatParametersErr: fmt.Errorf("65537 is an invalid Port for loadBalancer"),
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
			expCheckLoadBalancerFormatParametersErr: fmt.Errorf("SCTP is an invalid protocol for loadBalancer"),
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
			expCheckLoadBalancerFormatParametersErr: fmt.Errorf("62 is an invalid Timeout for loadBalancer"),
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
			expCheckLoadBalancerFormatParametersErr: fmt.Errorf("12 is an invalid threshold for loadBalancer"),
		},
	}
	for _, lbtc := range loadBalancerTestCases {
		t.Run(lbtc.name, func(t *testing.T) {
			clusterScope := Setup(t, lbtc.name, lbtc.spec)
			loadBalancerName, err := checkLoadBalancerFormatParameters(clusterScope)
			if err != nil {
				assert.Equal(t, lbtc.expCheckLoadBalancerFormatParametersErr.Error(), err.Error(), "checkLoadBalancerFormatParameters should return the same error")
			} else {
				assert.Nil(t, lbtc.expCheckLoadBalancerFormatParametersErr)
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
						SecurityGroupName: "test-securitygroup-test",
					},
				},
			},
			expCheckLoadBalancerSecuriyGroupOscAssociateResourceNameErr: fmt.Errorf("test-securitygroup-test-uid securityGroup does not exist in loadBalancer"),
		},
	}
	for _, lbtc := range loadBalancerTestCases {
		t.Run(lbtc.name, func(t *testing.T) {
			clusterScope := Setup(t, lbtc.name, lbtc.spec)
			err := checkLoadBalancerSecurityGroupOscAssociateResourceName(clusterScope)
			if err != nil {
				assert.Equal(t, lbtc.expCheckLoadBalancerSecuriyGroupOscAssociateResourceNameErr, err, "checkLoadBalancerSecurityGroupOscAssociateResourceName() should return the same error")
			} else {
				assert.Nil(t, lbtc.expCheckLoadBalancerSecuriyGroupOscAssociateResourceNameErr)
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
		expDescribeLoadBalancerErr    error
		expCreateLoadBalancerErr      error
		expConfigureLoadBalancerErr   error
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
			expDescribeLoadBalancerErr:    nil,
			expCreateLoadBalancerErr:      nil,
			expConfigureLoadBalancerErr:   nil,
			expReconcileLoadBalancerErr:   nil,
		},
		{
			name:                          "failed to configure loadBalancer",
			spec:                          defaultLoadBalancerInitialize,
			expLoadBalancerFound:          false,
			expSubnetFound:                true,
			expSecurityGroupFound:         true,
			expCreateLoadBalancerFound:    true,
			expConfigureLoadBalancerFound: false,
			expDescribeLoadBalancerErr:    nil,
			expCreateLoadBalancerErr:      nil,
			expConfigureLoadBalancerErr:   fmt.Errorf("ConfigureLoadBalancer generic error"),
			expReconcileLoadBalancerErr:   fmt.Errorf("ConfigureLoadBalancer generic error Can not configure healthcheck for Osccluster test-system/test-osc"),
		},
	}
	for _, lbtc := range loadBalancerTestCases {
		t.Run(lbtc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscLoadBalancerInterface := SetupWithLoadBalancerMock(t, lbtc.name, lbtc.spec)
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
			}
			reconcileLoadBalancer, err := reconcileLoadBalancer(ctx, clusterScope, mockOscLoadBalancerInterface)
			if err != nil {
				assert.Equal(t, lbtc.expReconcileLoadBalancerErr.Error(), err.Error(), "reconcileLoadBalancer() should return the same error")
			} else {
				assert.Nil(t, lbtc.expReconcileLoadBalancerErr)
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
		expCreateLoadBalancerFound    bool
		expCreateLoadBalancerErr      error
		expDescribeLoadBalancerErr    error
		expConfigureLoadBalancerFound bool
		expConfigureLoadBalancerErr   error
		expReconcileLoadBalancerErr   error
	}{
		{
			name:                          "check loadBalancer exist (second time reconcile loop)",
			spec:                          defaultLoadBalancerReconcile,
			expLoadBalancerFound:          true,
			expSubnetFound:                true,
			expSecurityGroupFound:         true,
			expCreateLoadBalancerFound:    false,
			expConfigureLoadBalancerFound: false,
			expDescribeLoadBalancerErr:    nil,
			expCreateLoadBalancerErr:      nil,
			expConfigureLoadBalancerErr:   nil,
			expReconcileLoadBalancerErr:   nil,
		},
		{
			name:                          "failed to get loadBalancer",
			spec:                          defaultLoadBalancerInitialize,
			expLoadBalancerFound:          false,
			expSubnetFound:                false,
			expSecurityGroupFound:         false,
			expCreateLoadBalancerFound:    false,
			expConfigureLoadBalancerFound: false,
			expDescribeLoadBalancerErr:    fmt.Errorf("GetLoadBalancer generic error"),
			expCreateLoadBalancerErr:      nil,
			expConfigureLoadBalancerErr:   nil,
			expReconcileLoadBalancerErr:   fmt.Errorf("GetLoadBalancer generic error"),
		},
	}
	for _, lbtc := range loadBalancerTestCases {
		t.Run(lbtc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscLoadBalancerInterface := SetupWithLoadBalancerMock(t, lbtc.name, lbtc.spec)
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
			reconcileLoadBalancer, err := reconcileLoadBalancer(ctx, clusterScope, mockOscLoadBalancerInterface)
			if err != nil {
				assert.Equal(t, lbtc.expReconcileLoadBalancerErr.Error(), err.Error(), "reconcileLoadBalancer() should return the same error")
			} else {
				assert.Nil(t, lbtc.expReconcileLoadBalancerErr)
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
			expCreateLoadBalancerErr:    fmt.Errorf("CreateLoadBalancer generic error"),
			expConfigureLoadBalancerErr: nil,
			expDescribeLoadBalancerErr:  nil,
			expReconcileLoadBalancerErr: fmt.Errorf("CreateLoadBalancer generic error Can not create loadBalancer for Osccluster test-system/test-osc"),
		},
	}
	for _, lbtc := range loadBalancerTestCases {
		t.Run(lbtc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscLoadBalancerInterface := SetupWithLoadBalancerMock(t, lbtc.name, lbtc.spec)

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
			reconcileLoadBalancer, err := reconcileLoadBalancer(ctx, clusterScope, mockOscLoadBalancerInterface)
			if err != nil {
				assert.Equal(t, lbtc.expReconcileLoadBalancerErr.Error(), err.Error(), "reconcileLoadBalancer() should return the same error")
			} else {
				assert.Nil(t, lbtc.expReconcileLoadBalancerErr)
			}
			t.Logf("find reconcileLoadBalancer %v\n", reconcileLoadBalancer)
		})
	}
}

// TestReconcileLoadBalancerResourceID has several tests to cover the code of the function reconcileLoadBalancer
func TestReconcileLoadBalancerResourceID(t *testing.T) {
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
			expReconcileLoadBalancerErr: fmt.Errorf("test-subnet-uid does not exist"),
		},
		{
			name:                        "securitygroup does not exist",
			spec:                        defaultLoadBalancerInitialize,
			expSubnetFound:              true,
			expSecurityGroupFound:       false,
			expDescribeLoadBalancerErr:  nil,
			expReconcileLoadBalancerErr: fmt.Errorf("test-securitygroup-uid does not exist"),
		},
	}
	for _, lbtc := range loadBalancerTestCases {
		t.Run(lbtc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscLoadBalancerInterface := SetupWithLoadBalancerMock(t, lbtc.name, lbtc.spec)

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
			reconcileLoadBalancer, err := reconcileLoadBalancer(ctx, clusterScope, mockOscLoadBalancerInterface)
			if err != nil {
				assert.Equal(t, lbtc.expReconcileLoadBalancerErr.Error(), err.Error(), "reconcileLoadBalancer() should return the same error")
			} else {
				assert.Nil(t, lbtc.expReconcileLoadBalancerErr)
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
		expDescribeLoadBalancerErr          error
		expDeleteLoadBalancerErr            error
		expCheckLoadBalancerDeregisterVMErr error
		expReconcileDeleteLoadBalancerErr   error
	}{
		{
			name:                                "delete loadBalancer (first time reconcile loop)",
			spec:                                defaultLoadBalancerReconcile,
			expLoadBalancerFound:                true,
			expDeleteLoadBalancerErr:            nil,
			expDescribeLoadBalancerErr:          nil,
			expCheckLoadBalancerDeregisterVMErr: nil,
			expReconcileDeleteLoadBalancerErr:   nil,
		},
		{
			name:                                "failed to delete loadBalancer",
			spec:                                defaultLoadBalancerReconcile,
			expLoadBalancerFound:                true,
			expDeleteLoadBalancerErr:            fmt.Errorf("DeleteLoadBalancer generic error"),
			expDescribeLoadBalancerErr:          nil,
			expCheckLoadBalancerDeregisterVMErr: nil,
			expReconcileDeleteLoadBalancerErr:   fmt.Errorf("DeleteLoadBalancer generic error Can not delete loadBalancer for Osccluster test-system/test-osc"),
		},
	}
	for _, lbtc := range loadBalancerTestCases {
		t.Run(lbtc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscLoadBalancerInterface := SetupWithLoadBalancerMock(t, lbtc.name, lbtc.spec)
			loadBalancerName := lbtc.spec.Network.LoadBalancer.LoadBalancerName + "-uid"
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
			readLoadBalancer := *readLoadBalancers.LoadBalancers
			var clockInsideLoop time.Duration = 5
			var clockLoop time.Duration = 120
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
				CheckLoadBalancerDeregisterVM(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(&loadBalancerSpec)).
				Return(lbtc.expCheckLoadBalancerDeregisterVMErr)

			reconcileDeleteLoadBalancer, err := reconcileDeleteLoadBalancer(ctx, clusterScope, mockOscLoadBalancerInterface)
			if err != nil {
				assert.Equal(t, lbtc.expReconcileDeleteLoadBalancerErr.Error(), err.Error(), "reconcileDeleteLoadBalancer() should return the same error")
			} else {
				assert.Nil(t, lbtc.expReconcileDeleteLoadBalancerErr)
			}
			t.Logf("find reconcileDeleteLoadBalancer %v\n", reconcileDeleteLoadBalancer)
		})
	}
}

// TestReconcileDeleteLoadBalancerDeleteWithoutSpec  has several tests to cover the code of the function reconcileDeleteLoadBalancer
func TestReconcileDeleteLoadBalancerDeleteWithoutSpec(t *testing.T) {
	loadBalancerTestCases := []struct {
		name                                string
		spec                                infrastructurev1beta1.OscClusterSpec
		expLoadBalancerFound                bool
		expDescribeLoadBalancerErr          error
		expDeleteLoadBalancerErr            error
		expCheckLoadBalancerDeregisterVMErr error
		expReconcileDeleteLoadBalancerErr   error
	}{
		{
			name:                                "delete loadBalancer without spec (with default values)",
			spec:                                defaultLoadBalancerReconcile,
			expLoadBalancerFound:                true,
			expDescribeLoadBalancerErr:          nil,
			expDeleteLoadBalancerErr:            nil,
			expCheckLoadBalancerDeregisterVMErr: nil,
			expReconcileDeleteLoadBalancerErr:   nil,
		},
	}
	for _, lbtc := range loadBalancerTestCases {
		t.Run(lbtc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscLoadBalancerInterface := SetupWithLoadBalancerMock(t, lbtc.name, lbtc.spec)
			loadBalancerName := "OscClusterApi-1-uid"
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
			readLoadBalancer := *readLoadBalancers.LoadBalancers
			var clockInsideLoop time.Duration = 5
			var clockLoop time.Duration = 120
			mockOscLoadBalancerInterface.
				EXPECT().
				GetLoadBalancer(gomock.Eq(loadBalancerSpec)).
				Return(&readLoadBalancer[0], lbtc.expDescribeLoadBalancerErr)
			mockOscLoadBalancerInterface.
				EXPECT().
				CheckLoadBalancerDeregisterVM(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(loadBalancerSpec)).
				Return(lbtc.expCheckLoadBalancerDeregisterVMErr)
			mockOscLoadBalancerInterface.
				EXPECT().
				DeleteLoadBalancer(gomock.Eq(loadBalancerSpec)).
				Return(lbtc.expDeleteLoadBalancerErr)
			reconcileDeleteLoadBalancer, err := reconcileDeleteLoadBalancer(ctx, clusterScope, mockOscLoadBalancerInterface)
			if err != nil {
				assert.Equal(t, lbtc.expReconcileDeleteLoadBalancerErr.Error(), err.Error(), "reconcileDeleteLoadBalancer() should return the same error")
			} else {
				assert.Nil(t, lbtc.expReconcileDeleteLoadBalancerErr)
			}
			t.Logf("find reconcileDeleteLoadBalancer %v\n", reconcileDeleteLoadBalancer)
		})
	}
}

// TestReconcileDeleteLoadBalancerCheck  has one tests to cover the code of the function ReconcileDeleteLoadBalancer
func TestReconcileDeleteLoadBalancerCheck(t *testing.T) {
	loadBalancerTestCases := []struct {
		name                                string
		spec                                infrastructurev1beta1.OscClusterSpec
		expLoadBalancerFound                bool
		expDescribeLoadBalancerErr          error
		expCheckLoadBalancerDeregisterVMErr error
		expReconcileDeleteLoadBalancerErr   error
	}{
		{
			name:                                "failed to delete loadBalancer",
			spec:                                defaultLoadBalancerReconcile,
			expLoadBalancerFound:                true,
			expDescribeLoadBalancerErr:          nil,
			expCheckLoadBalancerDeregisterVMErr: fmt.Errorf("CheckLoadBalancerDeregisterVM generic error"),
			expReconcileDeleteLoadBalancerErr:   fmt.Errorf("CheckLoadBalancerDeregisterVM generic error VmBackend is not deregister in loadBalancer test-loadbalancer for OscCluster test-system/test-osc"),
		},
	}
	for _, lbtc := range loadBalancerTestCases {
		t.Run(lbtc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscLoadBalancerInterface := SetupWithLoadBalancerMock(t, lbtc.name, lbtc.spec)
			loadBalancerName := lbtc.spec.Network.LoadBalancer.LoadBalancerName + "-uid"
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
			readLoadBalancer := *readLoadBalancers.LoadBalancers
			var clockInsideLoop time.Duration = 5
			var clockLoop time.Duration = 120
			mockOscLoadBalancerInterface.
				EXPECT().
				GetLoadBalancer(gomock.Eq(&loadBalancerSpec)).
				Return(&readLoadBalancer[0], lbtc.expDescribeLoadBalancerErr)
			mockOscLoadBalancerInterface.
				EXPECT().
				CheckLoadBalancerDeregisterVM(gomock.Eq(clockInsideLoop), gomock.Eq(clockLoop), gomock.Eq(&loadBalancerSpec)).
				Return(lbtc.expCheckLoadBalancerDeregisterVMErr)

			reconcileDeleteLoadBalancer, err := reconcileDeleteLoadBalancer(ctx, clusterScope, mockOscLoadBalancerInterface)
			if err != nil {
				assert.Equal(t, lbtc.expReconcileDeleteLoadBalancerErr.Error(), err.Error(), "reconcileDeleteLoadBalancer() should return the same error")
			} else {
				assert.Nil(t, lbtc.expReconcileDeleteLoadBalancerErr)
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
			expDescribeLoadBalancerErr:        fmt.Errorf("GetLoadBalancer generic error"),
			expReconcileDeleteLoadBalancerErr: fmt.Errorf("GetLoadBalancer generic error"),
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
			clusterScope, ctx, mockOscLoadBalancerInterface := SetupWithLoadBalancerMock(t, lbtc.name, lbtc.spec)
			loadBalancerSpec := lbtc.spec.Network.LoadBalancer
			loadBalancerSpec.SetDefaultValue()
			mockOscLoadBalancerInterface.
				EXPECT().
				GetLoadBalancer(gomock.Eq(&loadBalancerSpec)).
				Return(nil, lbtc.expDescribeLoadBalancerErr)
			reconcileDeleteLoadBalancer, err := reconcileDeleteLoadBalancer(ctx, clusterScope, mockOscLoadBalancerInterface)
			if err != nil {
				assert.Equal(t, lbtc.expReconcileDeleteLoadBalancerErr.Error(), err.Error(), "reconcileDeleteLoadBalancer() should return the same error")
			} else {
				assert.Nil(t, lbtc.expReconcileDeleteLoadBalancerErr)
			}
			t.Logf("find reconcileDeleteLoadBalancer %v\n", reconcileDeleteLoadBalancer)
		})
	}
}
