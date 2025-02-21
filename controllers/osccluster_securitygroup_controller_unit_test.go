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
	"sync"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/golang/mock/gomock"
	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/security"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/security/mock_security"
	"github.com/outscale/cluster-api-provider-outscale/cloud/tag/mock_tag"
	"github.com/stretchr/testify/require"

	osc "github.com/outscale/osc-sdk-go/v2"
)

var (
	defaultSecurityGroupInitialize = infrastructurev1beta1.OscClusterSpec{
		Network: infrastructurev1beta1.OscNetwork{
			ClusterName: "test-cluster",
			Net: infrastructurev1beta1.OscNet{
				Name:    "test-net",
				IpRange: "10.0.0.0/16",
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
		},
	}

	defaultSecurityGroupTagInitialize = infrastructurev1beta1.OscClusterSpec{
		Network: infrastructurev1beta1.OscNetwork{
			ClusterName: "test-cluster",
			Net: infrastructurev1beta1.OscNet{
				Name:    "test-net",
				IpRange: "10.0.0.0/16",
			},
			SecurityGroups: []*infrastructurev1beta1.OscSecurityGroup{
				{
					Name:        "test-securitygroup",
					Description: "test securitygroup",
					Tag:         "OscK8sMainSG",
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
		},
	}

	defaultSecurityGroupReconcile = infrastructurev1beta1.OscClusterSpec{
		Network: infrastructurev1beta1.OscNetwork{
			ClusterName: "test-cluster",
			Net: infrastructurev1beta1.OscNet{
				Name:       "test-net",
				IpRange:    "10.0.0.0/16",
				ResourceId: "vpc-test-net-uid",
			},
			SecurityGroups: []*infrastructurev1beta1.OscSecurityGroup{
				{Name: "test-securitygroup",
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
		},
	}

	defaultSecurityGroupReconcileExtraSecurityGroupRule = infrastructurev1beta1.OscClusterSpec{
		Network: infrastructurev1beta1.OscNetwork{
			ClusterName:            "test-cluster",
			ExtraSecurityGroupRule: true,
			Net: infrastructurev1beta1.OscNet{
				Name:       "test-net",
				IpRange:    "10.0.0.0/16",
				ResourceId: "vpc-test-net-uid",
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
		},
	}
)

// SetupWithSecurityGroupMock set securityGroupMock with clusterScope and osccluster
func SetupWithSecurityGroupMock(t *testing.T, name string, spec infrastructurev1beta1.OscClusterSpec) (clusterScope *scope.ClusterScope, ctx context.Context, mockOscSecurityGroupInterface *mock_security.MockOscSecurityGroupInterface, mockOscTagInterface *mock_tag.MockOscTagInterface) {
	clusterScope = Setup(t, name, spec)
	mockCtrl := gomock.NewController(t)
	mockOscSecurityGroupInterface = mock_security.NewMockOscSecurityGroupInterface(mockCtrl)
	mockOscTagInterface = mock_tag.NewMockOscTagInterface(mockCtrl)
	ctx = context.Background()
	return clusterScope, ctx, mockOscSecurityGroupInterface, mockOscTagInterface
}

// TestGetSecurityGroupResourceId has several tests to cover the code of the function getSecurityGrouptResourceId

func TestGetSecurityGroupResourceId(t *testing.T) {
	securityGroupTestCases := []struct {
		name                             string
		spec                             infrastructurev1beta1.OscClusterSpec
		expSecurityGroupsFound           bool
		expGetSecurityGroupResourceIdErr error
	}{
		{
			name:                             "get securityGroupId",
			spec:                             defaultSecurityGroupInitialize,
			expSecurityGroupsFound:           true,
			expGetSecurityGroupResourceIdErr: nil,
		},
		{
			name:                             "can not get securityGroupId",
			spec:                             defaultSecurityGroupInitialize,
			expSecurityGroupsFound:           false,
			expGetSecurityGroupResourceIdErr: errors.New("test-securitygroup-uid does not exist"),
		},
	}
	for _, sgtc := range securityGroupTestCases {
		t.Run(sgtc.name, func(t *testing.T) {
			clusterScope := Setup(t, sgtc.name, sgtc.spec)
			securityGroupsSpec := sgtc.spec.Network.SecurityGroups
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)

			for _, securityGroupSpec := range securityGroupsSpec {
				securityGroupName := securityGroupSpec.Name + "-uid"
				securityGroupId := "sg-" + securityGroupName
				if sgtc.expSecurityGroupsFound {
					securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
				}
				securityGroupResourceId, err := getSecurityGroupResourceId(securityGroupName, clusterScope)
				if sgtc.expGetSecurityGroupResourceIdErr != nil {
					require.EqualError(t, err, sgtc.expGetSecurityGroupResourceIdErr.Error(), "getSecurityGroupResourceId() should return the same error")
				} else {
					require.NoError(t, err)
				}
				t.Logf("Find securityGroupResourceId %s\n", securityGroupResourceId)
			}
		})
	}
}

// TestGetSecurityGroupRuleResourceId has several tests to cover the code of the function getSecurityGroupRuleResourceId
func TestGetSecurityGroupRuleResourceId(t *testing.T) {
	securityGroupRuleTestCases := []struct {
		name                                 string
		spec                                 infrastructurev1beta1.OscClusterSpec
		expSecurityGroupRuleFound            bool
		expGetSecurityGroupRuleResourceIdErr error
	}{
		{
			name:                                 "get securityGroupRuleId",
			spec:                                 defaultSecurityGroupInitialize,
			expSecurityGroupRuleFound:            true,
			expGetSecurityGroupRuleResourceIdErr: nil,
		},
		{
			name:                                 "can not get securityGroupRuleId",
			spec:                                 defaultSecurityGroupInitialize,
			expSecurityGroupRuleFound:            false,
			expGetSecurityGroupRuleResourceIdErr: errors.New("test-securitygrouprule-uid does not exist"),
		},
	}
	for _, sgrtc := range securityGroupRuleTestCases {
		t.Run(sgrtc.name, func(t *testing.T) {
			clusterScope := Setup(t, sgrtc.name, sgrtc.spec)
			securityGroupsSpec := sgrtc.spec.Network.SecurityGroups
			securityGroupRuleRef := clusterScope.GetSecurityGroupRuleRef()
			securityGroupRuleRef.ResourceMap = make(map[string]string)
			for _, securityGroupSpec := range securityGroupsSpec {
				securityGroupRulesSpec := securityGroupSpec.SecurityGroupRules
				securityGroupName := securityGroupSpec.Name + "-uid"
				securityGroupId := "sg-" + securityGroupName
				for _, securityGroupRuleSpec := range securityGroupRulesSpec {
					securityGroupRuleName := securityGroupRuleSpec.Name + "-uid"
					if sgrtc.expSecurityGroupRuleFound {
						securityGroupRuleRef.ResourceMap[securityGroupRuleName] = securityGroupId
					}
					securityGroupRuleResourceId, err := getSecurityGroupRulesResourceId(securityGroupRuleName, clusterScope)
					if sgrtc.expGetSecurityGroupRuleResourceIdErr != nil {
						require.EqualError(t, err, sgrtc.expGetSecurityGroupRuleResourceIdErr.Error(), "getSecurityGroupRuleResourceId() should return the same error")
					} else {
						require.NoError(t, err)
					}
					t.Logf("Find securityGroupRuleResourceId %s\n", securityGroupRuleResourceId)
				}
			}
		})
	}
}

// TestCheckSecurityGroupOscDuplicateName has several tests to cover the code of the function checkSecurityGroupOscDuplicateName
func TestCheckSecurityGroupOscDuplicateName(t *testing.T) {
	securityGroupTestCases := []struct {
		name                                     string
		spec                                     infrastructurev1beta1.OscClusterSpec
		expCheckSecurityGroupOscDuplicateNameErr error
	}{
		{
			name:                                     "get no duplicate securityGroup name",
			spec:                                     defaultSecurityGroupInitialize,
			expCheckSecurityGroupOscDuplicateNameErr: nil,
		},
		{
			name: "get duplicate securitygroup Name",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IpRange: "10.0.0.0/16",
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
				},
			},
			expCheckSecurityGroupOscDuplicateNameErr: errors.New("test-securitygroup already exist"),
		},
	}
	for _, sgtc := range securityGroupTestCases {
		t.Run(sgtc.name, func(t *testing.T) {
			clusterScope := Setup(t, sgtc.name, sgtc.spec)
			err := checkSecurityGroupOscDuplicateName(clusterScope)
			if sgtc.expCheckSecurityGroupOscDuplicateNameErr != nil {
				require.EqualError(t, err, sgtc.expCheckSecurityGroupOscDuplicateNameErr.Error(), "checkSecurityGroupOscDuplicateName() should return the same error")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestCheckSecurityGroupRuleOscDuplicateName has several tests to cover the code of the function checkSecurityGroupRuleOscDuplicateName
func TestCheckSecurityGroupRuleOscDuplicateName(t *testing.T) {
	securityGroupRuleTestCases := []struct {
		name                                         string
		spec                                         infrastructurev1beta1.OscClusterSpec
		expCheckSecurityGroupRuleOscDuplicateNameErr error
	}{
		{
			name: " get no securityGroup duplicate Name",
			spec: defaultSecurityGroupInitialize,
			expCheckSecurityGroupRuleOscDuplicateNameErr: nil,
		},
		{
			name: "get no securityGroup name",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{},
			},
			expCheckSecurityGroupRuleOscDuplicateNameErr: nil,
		},
		{
			name: " get securityGroup duplicate Name",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IpRange: "10.0.0.0/16",
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
				},
			},
			expCheckSecurityGroupRuleOscDuplicateNameErr: errors.New("test-securitygrouprule already exist"),
		},
	}
	for _, sgrtc := range securityGroupRuleTestCases {
		t.Run(sgrtc.name, func(t *testing.T) {
			clusterScope := Setup(t, sgrtc.name, sgrtc.spec)
			err := checkSecurityGroupRuleOscDuplicateName(clusterScope)
			if sgrtc.expCheckSecurityGroupRuleOscDuplicateNameErr != nil {
				require.EqualError(t, err, sgrtc.expCheckSecurityGroupRuleOscDuplicateNameErr.Error(), "checkSecurityGroupRuleOscDuplicateName() should return the same error")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestCheckSecurityGroupFormatParameters has several tests to cover the code of the function checkSecurityGroupFormatParameters
func TestCheckSecurityGroupFormatParameters(t *testing.T) {
	securityGroupTestCases := []struct {
		name                                     string
		spec                                     infrastructurev1beta1.OscClusterSpec
		expCheckSecurityGroupFormatParametersErr error
	}{
		{
			name: "check success without net and securityGroup spec (with default values)",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{},
			},
			expCheckSecurityGroupFormatParametersErr: nil,
		},
		{
			name:                                     "check securityGroup format",
			spec:                                     defaultSecurityGroupInitialize,
			expCheckSecurityGroupFormatParametersErr: nil,
		},
		{
			name: "check securityGroup bad name format",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IpRange: "10.0.0.0/16",
					},
					SecurityGroups: []*infrastructurev1beta1.OscSecurityGroup{
						{
							Name:        "test-securitygroup@test",
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
				},
			},
			expCheckSecurityGroupFormatParametersErr: errors.New("Invalid Tag Name"),
		},
		{
			name: "check securityGroup bad description format",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IpRange: "10.0.0.0/16",
					},
					SecurityGroups: []*infrastructurev1beta1.OscSecurityGroup{
						{
							Name:        "test-securitygroup",
							Description: "test securitygroup λ",
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
				},
			},
			expCheckSecurityGroupFormatParametersErr: errors.New("Invalid Description"),
		},
	}
	for _, sgtc := range securityGroupTestCases {
		t.Run(sgtc.name, func(t *testing.T) {
			clusterScope := Setup(t, sgtc.name, sgtc.spec)
			_, err := checkSecurityGroupFormatParameters(clusterScope)
			if sgtc.expCheckSecurityGroupFormatParametersErr != nil {
				require.EqualError(t, err, sgtc.expCheckSecurityGroupFormatParametersErr.Error(), "checkSecurityGroupFormatParameters() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("Find all securityGroupName")
		})
	}
}

// TestCheckSecurityGroupRuleFormatParameters has several tests to cover the code of the function checkSecurityGroupRuleFormatParameters
func TestCheckSecurityGroupRuleFormatParameters(t *testing.T) {
	securityGroupRuleTestCases := []struct {
		name                                         string
		spec                                         infrastructurev1beta1.OscClusterSpec
		expCheckSecurityGroupRuleFormatParametersErr error
	}{
		{
			name: "check work without net  and routetable (with default values)",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{},
			},
			expCheckSecurityGroupRuleFormatParametersErr: nil,
		},
		{
			name: "check securityGroupRule format",
			spec: defaultSecurityGroupInitialize,
			expCheckSecurityGroupRuleFormatParametersErr: nil,
		},
		{
			name: "check Bad Name SecurityGroupRule",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IpRange: "10.0.0.0/16",
					},
					SecurityGroups: []*infrastructurev1beta1.OscSecurityGroup{
						{
							Name:        "test-securitygroup",
							Description: "test securitygroup",
							SecurityGroupRules: []infrastructurev1beta1.OscSecurityGroupRule{
								{
									Name:          "test-securitygrouprule@test",
									Flow:          "Inbound",
									IpProtocol:    "tcp",
									IpRange:       "0.0.0.0/0",
									FromPortRange: 6443,
									ToPortRange:   6443,
								},
							},
						},
					},
				},
			},
			expCheckSecurityGroupRuleFormatParametersErr: errors.New("Invalid Tag Name"),
		},
		{
			name: "check Bad Flow SecurityGroupRule",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IpRange: "10.0.0.0/16",
					},
					SecurityGroups: []*infrastructurev1beta1.OscSecurityGroup{
						{
							Name:        "test-securitygroup",
							Description: "test securitygroup",
							SecurityGroupRules: []infrastructurev1beta1.OscSecurityGroupRule{
								{
									Name:          "test-securitygrouprule",
									Flow:          "Nobound",
									IpProtocol:    "tcp",
									IpRange:       "0.0.0.0/0",
									FromPortRange: 6443,
									ToPortRange:   6443,
								},
							},
						},
					},
				},
			},
			expCheckSecurityGroupRuleFormatParametersErr: errors.New("Invalid flow"),
		},
		{
			name: "check Bad IpProtocol SecurityGroupRule",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IpRange: "10.0.0.0/16",
					},
					SecurityGroups: []*infrastructurev1beta1.OscSecurityGroup{
						{
							Name:        "test-securitygroup",
							Description: "test securitygroup",
							SecurityGroupRules: []infrastructurev1beta1.OscSecurityGroupRule{
								{
									Name:          "test-securitygrouprule",
									Flow:          "Inbound",
									IpProtocol:    "sctp",
									IpRange:       "0.0.0.0/0",
									FromPortRange: 6443,
									ToPortRange:   6443,
								},
							},
						},
					},
				},
			},
			expCheckSecurityGroupRuleFormatParametersErr: errors.New("Invalid protocol"),
		},
		{
			name: "check Bad Ip Range Prefix securityGroupRule",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IpRange: "10.0.0.0/16",
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
									IpRange:       "10.0.0.0/36",
									FromPortRange: 6443,
									ToPortRange:   6443,
								},
							},
						},
					},
				},
			},
			expCheckSecurityGroupRuleFormatParametersErr: errors.New("invalid CIDR address: 10.0.0.0/36"),
		},
		{
			name: "check Bad Ip Range Ip securityGroupRule",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IpRange: "10.0.0.0/16",
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
									IpRange:       "10.0.0.256/16",
									FromPortRange: 6443,
									ToPortRange:   6443,
								},
							},
						},
					},
				},
			},
			expCheckSecurityGroupRuleFormatParametersErr: errors.New("invalid CIDR address: 10.0.0.256/16"),
		},
		{
			name: "check bad FromPortRange securityGroupRule",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IpRange: "10.0.0.0/16",
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
									FromPortRange: 65537,
									ToPortRange:   6443,
								},
							},
						},
					},
				},
			},
			expCheckSecurityGroupRuleFormatParametersErr: errors.New("Invalid Port"),
		},
		{
			name: "check bad ToPortRange securityGroupRule",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IpRange: "10.0.0.0/16",
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
									ToPortRange:   65537,
								},
							},
						},
					},
				},
			},
			expCheckSecurityGroupRuleFormatParametersErr: errors.New("Invalid Port"),
		},
	}
	for _, sgrtc := range securityGroupRuleTestCases {
		t.Run(sgrtc.name, func(t *testing.T) {
			clusterScope := Setup(t, sgrtc.name, sgrtc.spec)
			_, err := checkSecurityGroupRuleFormatParameters(clusterScope)
			if sgrtc.expCheckSecurityGroupRuleFormatParametersErr != nil {
				require.EqualError(t, err, sgrtc.expCheckSecurityGroupRuleFormatParametersErr.Error(), "checkSecurityGroupRuleFormatParameters() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("find all securityGroupRule")
		})
	}
}

// TestReconcileSecurityGroupRuleCreate has several tests to cover the code of the function reconcileSecurityGroupRule

func TestReconcileSecurityGroupRuleCreate(t *testing.T) {
	securityGroupRuleTestCases := []struct {
		name                             string
		spec                             infrastructurev1beta1.OscClusterSpec
		expCreateSecurityGroupRuleFound  bool
		expTagFound                      bool
		expSecurityGroupHasRuleErr       error
		expCreateSecurityGroupRuleErr    error
		expReadTagErr                    error
		expReconcileSecurityGroupRuleErr error
	}{
		{
			name:                             "create securityGroupRule  (first time reconcileloop)",
			spec:                             defaultSecurityGroupInitialize,
			expCreateSecurityGroupRuleFound:  true,
			expTagFound:                      false,
			expSecurityGroupHasRuleErr:       nil,
			expCreateSecurityGroupRuleErr:    nil,
			expReadTagErr:                    nil,
			expReconcileSecurityGroupRuleErr: nil,
		},
		{
			name:                             "failed to create securityGroupRule",
			spec:                             defaultSecurityGroupInitialize,
			expCreateSecurityGroupRuleFound:  false,
			expTagFound:                      false,
			expSecurityGroupHasRuleErr:       nil,
			expCreateSecurityGroupRuleErr:    errors.New("CreateSecurityGroupRule generic errors"),
			expReadTagErr:                    nil,
			expReconcileSecurityGroupRuleErr: errors.New("cannot create securityGroupRule: CreateSecurityGroupRule generic errors"),
		},
	}
	for _, sgrtc := range securityGroupRuleTestCases {
		t.Run(sgrtc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscSecurityGroupInterface, mockOscTagInterface := SetupWithSecurityGroupMock(t, sgrtc.name, sgrtc.spec)
			securityGroupsSpec := sgrtc.spec.Network.SecurityGroups
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			for _, securityGroupSpec := range securityGroupsSpec {
				securityGroupName := securityGroupSpec.Name + "-uid"
				securityGroupId := "sg-" + securityGroupName
				tag := osc.Tag{
					ResourceId: &securityGroupId,
				}
				if sgrtc.expTagFound {
					mockOscTagInterface.
						EXPECT().
						ReadTag(gomock.Any(), gomock.Eq("Name"), gomock.Eq(securityGroupName)).
						Return(&tag, sgrtc.expReadTagErr)
				}

				securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
				securityGroupRulesSpec := securityGroupSpec.SecurityGroupRules
				for _, securityGroupRuleSpec := range securityGroupRulesSpec {
					securityGroupRuleFlow := securityGroupRuleSpec.Flow
					securityGroupRuleIpProtocol := securityGroupRuleSpec.IpProtocol
					securityGroupRuleIpRange := securityGroupRuleSpec.IpRange
					securityGroupRuleFromPortRange := securityGroupRuleSpec.FromPortRange
					securityGroupRuleToPortRange := securityGroupRuleSpec.ToPortRange
					securityGroupMemberId := ""
					securityGroupRule := osc.CreateSecurityGroupRuleResponse{
						SecurityGroup: &osc.SecurityGroup{
							SecurityGroupId: &securityGroupId,
						},
					}

					mockOscSecurityGroupInterface.
						EXPECT().
						SecurityGroupHasRule(gomock.Any(), gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIpProtocol), gomock.Eq(securityGroupRuleIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
						Return(false, sgrtc.expSecurityGroupHasRuleErr)

					if sgrtc.expCreateSecurityGroupRuleFound {
						mockOscSecurityGroupInterface.
							EXPECT().
							CreateSecurityGroupRule(gomock.Any(), gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIpProtocol), gomock.Eq(securityGroupRuleIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
							Return(securityGroupRule.SecurityGroup, sgrtc.expCreateSecurityGroupRuleErr)
					} else {
						mockOscSecurityGroupInterface.
							EXPECT().
							CreateSecurityGroupRule(gomock.Any(), gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIpProtocol), gomock.Eq(securityGroupRuleIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
							Return(nil, sgrtc.expCreateSecurityGroupRuleErr)
					}
					reconcileSecurityGroupRule, err := reconcileSecurityGroupRule(ctx, clusterScope, securityGroupRuleSpec, securityGroupName, mockOscSecurityGroupInterface)
					if sgrtc.expReconcileSecurityGroupRuleErr != nil {
						require.EqualError(t, err, sgrtc.expReconcileSecurityGroupRuleErr.Error(), "reconcileSecurityGroupRules() should return the same error")
					} else {
						require.NoError(t, err)
					}
					t.Logf("find reconcileSecurityGroupRule %v\n", reconcileSecurityGroupRule)
				}
			}
		})
	}
}

// TestReconcileSecurityGroupRuleGet has several tests to cover the code of the function reconcileSecurityGroupRule
func TestReconcileSecurityGroupRuleGet(t *testing.T) {
	securityGroupRuleTestCases := []struct {
		name                             string
		spec                             infrastructurev1beta1.OscClusterSpec
		expSecurityGroupRuleFound        bool
		expTagFound                      bool
		expSecurityGroupHasRuleErr       error
		expReadTagErr                    error
		expReconcileSecurityGroupRuleErr error
	}{
		{
			name:                             "get securityGroupRule ((second time reconcile loop)",
			spec:                             defaultSecurityGroupReconcile,
			expSecurityGroupRuleFound:        true,
			expTagFound:                      false,
			expSecurityGroupHasRuleErr:       nil,
			expReadTagErr:                    nil,
			expReconcileSecurityGroupRuleErr: nil,
		},
		{
			name:                             "get securityGroupRule ((second time reconcile loop)) with extraSecurityGroupRule",
			spec:                             defaultSecurityGroupReconcileExtraSecurityGroupRule,
			expSecurityGroupRuleFound:        true,
			expTagFound:                      false,
			expSecurityGroupHasRuleErr:       nil,
			expReadTagErr:                    nil,
			expReconcileSecurityGroupRuleErr: nil,
		},
		{
			name:                             "failed to get securityGroup",
			spec:                             defaultSecurityGroupReconcile,
			expSecurityGroupRuleFound:        true,
			expTagFound:                      false,
			expReadTagErr:                    nil,
			expSecurityGroupHasRuleErr:       errors.New("SecurityGroupHasRule generic errors"),
			expReconcileSecurityGroupRuleErr: errors.New("SecurityGroupHasRule generic errors"),
		},
	}
	for _, sgrtc := range securityGroupRuleTestCases {
		t.Run(sgrtc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscSecurityGroupInterface, mockOscTagInterface := SetupWithSecurityGroupMock(t, sgrtc.name, sgrtc.spec)
			securityGroupsSpec := sgrtc.spec.Network.SecurityGroups
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			for _, securityGroupSpec := range securityGroupsSpec {
				securityGroupName := securityGroupSpec.Name + "-uid"
				securityGroupId := "sg-" + securityGroupName
				tag := osc.Tag{
					ResourceId: &securityGroupId,
				}
				if sgrtc.expTagFound {
					mockOscTagInterface.
						EXPECT().
						ReadTag(gomock.Any(), gomock.Eq("Name"), gomock.Eq(securityGroupName)).
						Return(&tag, sgrtc.expReadTagErr)
				}

				securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
				securityGroupRulesSpec := securityGroupSpec.SecurityGroupRules

				for _, securityGroupRuleSpec := range securityGroupRulesSpec {
					securityGroupRuleFlow := securityGroupRuleSpec.Flow
					securityGroupRuleIpProtocol := securityGroupRuleSpec.IpProtocol
					securityGroupRuleIpRange := securityGroupRuleSpec.IpRange
					securityGroupRuleFromPortRange := securityGroupRuleSpec.FromPortRange
					securityGroupRuleToPortRange := securityGroupRuleSpec.ToPortRange
					securityGroupMemberId := ""

					if sgrtc.expSecurityGroupRuleFound {
						mockOscSecurityGroupInterface.
							EXPECT().
							SecurityGroupHasRule(gomock.Any(), gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIpProtocol), gomock.Eq(securityGroupRuleIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
							Return(true, sgrtc.expSecurityGroupHasRuleErr)
					}
					reconcileSecurityGroupRule, err := reconcileSecurityGroupRule(ctx, clusterScope, securityGroupRuleSpec, securityGroupName, mockOscSecurityGroupInterface)
					if sgrtc.expReconcileSecurityGroupRuleErr != nil {
						require.EqualError(t, err, sgrtc.expReconcileSecurityGroupRuleErr.Error(), "reconcileSecurityGroupRules() should return the same error")
					} else {
						require.NoError(t, err)
					}
					t.Logf("find reconcileSecurityGroupRule %v\n", reconcileSecurityGroupRule)
				}
			}
		})
	}
}

// TestReconcileDeleteSecurityGroupRuleDelete has several tests to cover the code of the function reconcileDeleteSecurityGroupRule
func TestReconcileDeleteSecurityGroupRuleDelete(t *testing.T) {
	securityGroupRuleTestCases := []struct {
		name                                   string
		spec                                   infrastructurev1beta1.OscClusterSpec
		expSecurityGroupHasRuleErr             error
		expDeleteSecurityGroupRuleErr          error
		expReconcileDeleteSecurityGroupRuleErr error
	}{
		{
			name:                                   "failed to delete securityGroupRule",
			spec:                                   defaultSecurityGroupReconcile,
			expSecurityGroupHasRuleErr:             nil,
			expDeleteSecurityGroupRuleErr:          errors.New("DeleteSecurityGroupRule generic error"),
			expReconcileDeleteSecurityGroupRuleErr: errors.New("cannot delete securityGroupRule: DeleteSecurityGroupRule generic error"),
		},
		{
			name:                                   "delete securityGroupRule",
			spec:                                   defaultSecurityGroupReconcile,
			expSecurityGroupHasRuleErr:             nil,
			expDeleteSecurityGroupRuleErr:          nil,
			expReconcileDeleteSecurityGroupRuleErr: nil,
		},
	}
	for _, sgrtc := range securityGroupRuleTestCases {
		t.Run(sgrtc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscSecurityGroupInterface, _ := SetupWithSecurityGroupMock(t, sgrtc.name, sgrtc.spec)
			securityGroupsSpec := sgrtc.spec.Network.SecurityGroups
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			for _, securityGroupSpec := range securityGroupsSpec {
				securityGroupName := securityGroupSpec.Name + "-uid"
				securityGroupId := "sg-" + securityGroupName
				securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
				securityGroupRulesSpec := securityGroupSpec.SecurityGroupRules
				for _, securityGroupRuleSpec := range securityGroupRulesSpec {
					securityGroupRuleFlow := securityGroupRuleSpec.Flow
					securityGroupRuleIpProtocol := securityGroupRuleSpec.IpProtocol
					securityGroupRuleIpRange := securityGroupRuleSpec.IpRange
					securityGroupRuleFromPortRange := securityGroupRuleSpec.FromPortRange
					securityGroupRuleToPortRange := securityGroupRuleSpec.ToPortRange
					securityGroupMemberId := ""

					mockOscSecurityGroupInterface.
						EXPECT().
						SecurityGroupHasRule(gomock.Any(), gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIpProtocol), gomock.Eq(securityGroupRuleIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
						Return(true, sgrtc.expSecurityGroupHasRuleErr)
					mockOscSecurityGroupInterface.
						EXPECT().
						DeleteSecurityGroupRule(gomock.Any(), gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIpProtocol), gomock.Eq(securityGroupRuleIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
						Return(sgrtc.expDeleteSecurityGroupRuleErr)
					reconcileDeleteSecurityGroupRule, err := reconcileDeleteSecurityGroupRule(ctx, clusterScope, securityGroupRuleSpec, securityGroupName, mockOscSecurityGroupInterface)
					if sgrtc.expReconcileDeleteSecurityGroupRuleErr != nil {
						require.EqualError(t, err, sgrtc.expReconcileDeleteSecurityGroupRuleErr.Error(), "reconcileDeleteSecuritygroupRules() should return the same error")
					} else {
						require.NoError(t, err)
					}
					t.Logf("find reconcileDeleteSecurityGroupRule %v\n", reconcileDeleteSecurityGroupRule)
				}
			}
		})
	}
}

// TestReconcileDeleteSecurityGroupRuleGet has several tests to cover the code of the function reconcileDeleteSecurityGroupRule
func TestReconcileDeleteSecurityGroupRuleGet(t *testing.T) {
	securityGroupRuleTestCases := []struct {
		name                                   string
		spec                                   infrastructurev1beta1.OscClusterSpec
		expSecurityGroupHasRuleErr             error
		expReconcileDeleteSecurityGroupRuleErr error
	}{
		{
			name:                       "failed to get securityGroupRule",
			spec:                       defaultSecurityGroupReconcile,
			expSecurityGroupHasRuleErr: errors.New("SecurityGroupHasRule generic errors"),

			expReconcileDeleteSecurityGroupRuleErr: errors.New("SecurityGroupHasRule generic errors"),
		},
		{
			name:                                   "remove finalizer (user delete securityGroup without cluster-api)",
			spec:                                   defaultSecurityGroupReconcile,
			expSecurityGroupHasRuleErr:             nil,
			expReconcileDeleteSecurityGroupRuleErr: nil,
		},
	}
	for _, sgrtc := range securityGroupRuleTestCases {
		t.Run(sgrtc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscSecurityGroupInterface, _ := SetupWithSecurityGroupMock(t, sgrtc.name, sgrtc.spec)
			securityGroupsSpec := sgrtc.spec.Network.SecurityGroups
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			for _, securityGroupSpec := range securityGroupsSpec {
				securityGroupName := securityGroupSpec.Name + "-uid"
				securityGroupId := "sg-" + securityGroupName
				securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
				securityGroupRulesSpec := securityGroupSpec.SecurityGroupRules
				for _, securityGroupRuleSpec := range securityGroupRulesSpec {
					securityGroupRuleFlow := securityGroupRuleSpec.Flow
					securityGroupRuleIpProtocol := securityGroupRuleSpec.IpProtocol
					securityGroupRuleIpRange := securityGroupRuleSpec.IpRange
					securityGroupRuleFromPortRange := securityGroupRuleSpec.FromPortRange
					securityGroupRuleToPortRange := securityGroupRuleSpec.ToPortRange
					securityGroupMemberId := ""
					mockOscSecurityGroupInterface.
						EXPECT().
						SecurityGroupHasRule(gomock.Any(), gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIpProtocol), gomock.Eq(securityGroupRuleIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
						Return(false, sgrtc.expSecurityGroupHasRuleErr)
					reconcileDeleteSecurityGroupRule, err := reconcileDeleteSecurityGroupRule(ctx, clusterScope, securityGroupRuleSpec, securityGroupName, mockOscSecurityGroupInterface)
					if sgrtc.expReconcileDeleteSecurityGroupRuleErr != nil {
						require.EqualError(t, err, sgrtc.expReconcileDeleteSecurityGroupRuleErr.Error(), "reconcileDeleteSecuritygroupRules() should return the same error")
					} else {
						require.NoError(t, err)
					}
					t.Logf("find reconcileDeleteSecurityGroupRule %v\n", reconcileDeleteSecurityGroupRule)
				}
			}
		})
	}
}

// TestReconcileCreateSecurityGroupCreate has several tests to cover the code of the function reconcileCreateSecurityGroup
func TestReconcileCreateSecurityGroupCreate(t *testing.T) {
	securityGroupTestCases := []struct {
		name                             string
		spec                             infrastructurev1beta1.OscClusterSpec
		expSecurityGroupRuleFound        bool
		expCreateSecurityGroupRuleFound  bool
		expTagFound                      bool
		expGetSecurityGroupFromNetIdsErr error
		expCreateSecurityGroupErr        error
		expGetSecurityGroupRuleErr       error
		expCreateSecurityGroupRuleErr    error
		expReadTagErr                    error
		expReconcileSecurityGroupErr     error
	}{
		{
			name: "create securityGroup",

			spec:                             defaultSecurityGroupInitialize,
			expTagFound:                      false,
			expSecurityGroupRuleFound:        false,
			expCreateSecurityGroupRuleFound:  true,
			expGetSecurityGroupFromNetIdsErr: nil,
			expCreateSecurityGroupErr:        nil,
			expGetSecurityGroupRuleErr:       nil,
			expCreateSecurityGroupRuleErr:    nil,
			expReadTagErr:                    nil,
			expReconcileSecurityGroupErr:     nil,
		},
		{
			name:                             "create securityGroupTag",
			spec:                             defaultSecurityGroupTagInitialize,
			expSecurityGroupRuleFound:        false,
			expCreateSecurityGroupRuleFound:  true,
			expTagFound:                      false,
			expGetSecurityGroupFromNetIdsErr: nil,
			expCreateSecurityGroupErr:        nil,
			expGetSecurityGroupRuleErr:       nil,
			expCreateSecurityGroupRuleErr:    nil,
			expReadTagErr:                    nil,
			expReconcileSecurityGroupErr:     nil,
		},
		{
			name: "failed to create securityGroupRule",

			spec:                             defaultSecurityGroupInitialize,
			expSecurityGroupRuleFound:        false,
			expCreateSecurityGroupRuleFound:  false,
			expTagFound:                      false,
			expGetSecurityGroupFromNetIdsErr: nil,
			expCreateSecurityGroupErr:        nil,
			expGetSecurityGroupRuleErr:       nil,
			expCreateSecurityGroupRuleErr:    errors.New("CreateSecurityGroupRule generic errors"),
			expReadTagErr:                    nil,
			expReconcileSecurityGroupErr:     errors.New("cannot create securityGroupRule: CreateSecurityGroupRule generic errors"),
		},
	}
	for _, sgtc := range securityGroupTestCases {
		t.Run(sgtc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscSecurityGroupInterface, mockOscTagInterface := SetupWithSecurityGroupMock(t, sgtc.name, sgtc.spec)

			clusterName := sgtc.spec.Network.ClusterName + "-uid"
			netName := sgtc.spec.Network.Net.Name + "-uid"
			netId := "vpc-" + netName
			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			netRef.ResourceMap[netName] = netId

			securityGroupsSpec := sgtc.spec.Network.SecurityGroups
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)

			var securityGroupIds []string
			for _, securityGroupSpec := range securityGroupsSpec {
				securityGroupName := securityGroupSpec.Name + "-uid"
				securityGroupId := "sg-" + securityGroupName
				tag := osc.Tag{
					ResourceId: &securityGroupId,
				}
				if sgtc.expTagFound {
					mockOscTagInterface.
						EXPECT().
						ReadTag(gomock.Any(), gomock.Eq("Name"), gomock.Eq(securityGroupName)).
						Return(&tag, sgtc.expReadTagErr)
				} else {
					mockOscTagInterface.
						EXPECT().
						ReadTag(gomock.Any(), gomock.Eq("Name"), gomock.Eq(securityGroupName)).
						Return(nil, sgtc.expReadTagErr)
				}
				securityGroupIds = append(securityGroupIds, securityGroupId)
				securityGroupDescription := securityGroupSpec.Description
				securityGroupTag := securityGroupSpec.Tag
				securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
				securityGroup := osc.CreateSecurityGroupResponse{
					SecurityGroup: &osc.SecurityGroup{
						SecurityGroupId: &securityGroupId,
					},
				}
				mockOscSecurityGroupInterface.
					EXPECT().
					GetSecurityGroupIdsFromNetIds(gomock.Any(), gomock.Eq(netId)).
					Return(nil, sgtc.expGetSecurityGroupFromNetIdsErr)
				mockOscSecurityGroupInterface.
					EXPECT().
					CreateSecurityGroup(gomock.Any(), gomock.Eq(netId), gomock.Eq(clusterName), gomock.Eq(securityGroupName), gomock.Eq(securityGroupDescription), gomock.Eq(securityGroupTag)).
					Return(securityGroup.SecurityGroup, sgtc.expCreateSecurityGroupErr)
				for _, securityGroupSpec := range securityGroupsSpec {
					securityGroupName := securityGroupSpec.Name + "-uid"
					securityGroupId := "sg-" + securityGroupName
					securityGroupIds = append(securityGroupIds, securityGroupId)
					securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
					securityGroupRulesSpec := securityGroupSpec.SecurityGroupRules
					for _, securityGroupRuleSpec := range securityGroupRulesSpec {
						securityGroupRuleFlow := securityGroupRuleSpec.Flow
						securityGroupRuleIpProtocol := securityGroupRuleSpec.IpProtocol
						securityGroupRuleIpRange := securityGroupRuleSpec.IpRange
						securityGroupRuleFromPortRange := securityGroupRuleSpec.FromPortRange
						securityGroupRuleToPortRange := securityGroupRuleSpec.ToPortRange
						securityGroupMemberId := ""
						securityGroupRule := osc.CreateSecurityGroupRuleResponse{
							SecurityGroup: &osc.SecurityGroup{
								SecurityGroupId: &securityGroupId,
							},
						}

						readSecurityGroups := osc.ReadSecurityGroupsResponse{
							SecurityGroups: &[]osc.SecurityGroup{
								*securityGroupRule.SecurityGroup,
							},
						}
						readSecurityGroup := *readSecurityGroups.SecurityGroups

						if sgtc.expSecurityGroupRuleFound {
							mockOscSecurityGroupInterface.
								EXPECT().
								SecurityGroupHasRule(gomock.Any(), gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIpProtocol), gomock.Eq(securityGroupRuleIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
								Return(&readSecurityGroup[0], sgtc.expGetSecurityGroupRuleErr)
						} else {
							mockOscSecurityGroupInterface.
								EXPECT().
								SecurityGroupHasRule(gomock.Any(), gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIpProtocol), gomock.Eq(securityGroupRuleIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
								Return(false, sgtc.expGetSecurityGroupRuleErr)
							if sgtc.expCreateSecurityGroupRuleFound {
								mockOscSecurityGroupInterface.
									EXPECT().
									CreateSecurityGroupRule(gomock.Any(), gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIpProtocol), gomock.Eq(securityGroupRuleIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
									Return(securityGroupRule.SecurityGroup, sgtc.expCreateSecurityGroupRuleErr)
							} else {
								mockOscSecurityGroupInterface.
									EXPECT().
									CreateSecurityGroupRule(gomock.Any(), gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIpProtocol), gomock.Eq(securityGroupRuleIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
									Return(nil, sgtc.expCreateSecurityGroupRuleErr)
							}
						}
					}
					reconcileSecurityGroup, err := reconcileSecurityGroup(ctx, clusterScope, mockOscSecurityGroupInterface, mockOscTagInterface)
					if sgtc.expReconcileSecurityGroupErr != nil {
						require.EqualError(t, err, sgtc.expReconcileSecurityGroupErr.Error(), "reconcileSecurityGroup() should return the same error")
					} else {
						require.NoError(t, err)
					}

					t.Logf("find reconcileSecurityGroup %v\n", reconcileSecurityGroup)
				}
			}
		})
	}
}

// TestReconcileCreateSecurityGroupGet has several tests to cover the code of the function reconcileCreateSecurityGroup
func TestReconcileCreateSecurityGroupGet(t *testing.T) {
	securityGroupTestCases := []struct {
		name                             string
		spec                             infrastructurev1beta1.OscClusterSpec
		expSecurityGroupFound            bool
		expTagFound                      bool
		expNetFound                      bool
		expGetSecurityGroupFromNetIdsErr error
		expReadTagErr                    error
		expReconcileSecurityGroupErr     error
	}{
		{
			name: "failed to get securityGroup",

			spec:                             defaultSecurityGroupReconcile,
			expSecurityGroupFound:            false,
			expTagFound:                      false,
			expNetFound:                      true,
			expGetSecurityGroupFromNetIdsErr: errors.New("GetSecurityGroup generic error"),
			expReadTagErr:                    nil,
			expReconcileSecurityGroupErr:     errors.New("GetSecurityGroup generic error"),
		},
		{
			name: "get securityGroup (second time reconcile loop)",

			spec:                             defaultSecurityGroupReconcile,
			expSecurityGroupFound:            true,
			expTagFound:                      false,
			expNetFound:                      true,
			expGetSecurityGroupFromNetIdsErr: nil,
			expReadTagErr:                    nil,
			expReconcileSecurityGroupErr:     nil,
		},
	}
	for _, sgtc := range securityGroupTestCases {
		t.Run(sgtc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscSecurityGroupInterface, mockOscTagInterface := SetupWithSecurityGroupMock(t, sgtc.name, sgtc.spec)

			netName := sgtc.spec.Network.Net.Name + "-uid"
			netId := "vpc-" + netName
			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			netRef.ResourceMap[netName] = netId
			if sgtc.expNetFound {
				netRef.ResourceMap[netName] = netId
			}
			securityGroupsSpec := sgtc.spec.Network.SecurityGroups
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)

			var securityGroupIds []string
			for _, securityGroupSpec := range securityGroupsSpec {
				securityGroupName := securityGroupSpec.Name + "-uid"
				securityGroupId := "sg-" + securityGroupName
				tag := osc.Tag{
					ResourceId: &securityGroupId,
				}
				if sgtc.expSecurityGroupFound {
					if sgtc.expSecurityGroupFound {
						mockOscTagInterface.
							EXPECT().
							ReadTag(gomock.Any(), gomock.Eq("Name"), gomock.Eq(securityGroupName)).
							Return(&tag, sgtc.expReadTagErr)
					} else {
						mockOscTagInterface.
							EXPECT().
							ReadTag(gomock.Any(), gomock.Eq("Name"), gomock.Eq(securityGroupName)).
							Return(nil, sgtc.expReadTagErr)
					}
				}
				securityGroupIds = append(securityGroupIds, securityGroupId)
				securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
				if sgtc.expSecurityGroupFound {
					mockOscSecurityGroupInterface.
						EXPECT().
						GetSecurityGroupIdsFromNetIds(gomock.Any(), gomock.Eq(netId)).
						Return(securityGroupIds, sgtc.expGetSecurityGroupFromNetIdsErr)
				} else {
					mockOscSecurityGroupInterface.
						EXPECT().
						GetSecurityGroupIdsFromNetIds(gomock.Any(), gomock.Eq(netId)).
						Return(nil, sgtc.expGetSecurityGroupFromNetIdsErr)
				}
				reconcileSecurityGroup, err := reconcileSecurityGroup(ctx, clusterScope, mockOscSecurityGroupInterface, mockOscTagInterface)
				if sgtc.expReconcileSecurityGroupErr != nil {
					require.EqualError(t, err, sgtc.expReconcileSecurityGroupErr.Error(), "reconcileSecurityGroup() should return the same error")
				} else {
					require.NoError(t, err)
				}

				t.Logf("find reconcileSecurityGroup %v\n", reconcileSecurityGroup)
			}
		})
	}
}

// TestReconcileCreateSecurityGroupFailedCreate has several tests to cover the code of the function reconcileCreateSecurityGroup
func TestReconcileCreateSecurityGroupFailedCreate(t *testing.T) {
	securityGroupTestCases := []struct {
		name                             string
		spec                             infrastructurev1beta1.OscClusterSpec
		expTagFound                      bool
		expGetSecurityGroupFromNetIdsErr error
		expCreateSecurityGroupErr        error
		expReadTagErr                    error
		expReconcileSecurityGroupErr     error
	}{
		{
			name: "failed to create securityGroup",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IpRange: "10.0.0.0/16",
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
				},
			},
			expGetSecurityGroupFromNetIdsErr: nil,
			expTagFound:                      false,
			expCreateSecurityGroupErr:        errors.New("CreateSecurityGroup generic error"),
			expReadTagErr:                    nil,
			expReconcileSecurityGroupErr:     errors.New("cannot create securityGroup: CreateSecurityGroup generic error"),
		},
	}
	for _, sgtc := range securityGroupTestCases {
		t.Run(sgtc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscSecurityGroupInterface, mockOscTagInterface := SetupWithSecurityGroupMock(t, sgtc.name, sgtc.spec)

			netName := sgtc.spec.Network.Net.Name + "-uid"
			netId := "vpc-" + netName
			clusterName := sgtc.spec.Network.ClusterName + "-uid"
			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			netRef.ResourceMap[netName] = netId

			securityGroupsSpec := sgtc.spec.Network.SecurityGroups
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)

			var securityGroupIds []string
			for _, securityGroupSpec := range securityGroupsSpec {
				securityGroupName := securityGroupSpec.Name + "-uid"
				securityGroupId := "sg-" + securityGroupName
				tag := osc.Tag{
					ResourceId: &securityGroupId,
				}
				if sgtc.expTagFound {
					mockOscTagInterface.
						EXPECT().
						ReadTag(gomock.Any(), gomock.Eq("Name"), gomock.Eq(securityGroupName)).
						Return(&tag, sgtc.expReadTagErr)
				} else {
					mockOscTagInterface.
						EXPECT().
						ReadTag(gomock.Any(), gomock.Eq("Name"), gomock.Eq(securityGroupName)).
						Return(nil, sgtc.expReadTagErr)
				}
				securityGroupIds = append(securityGroupIds, securityGroupId)
				securityGroupTag := securityGroupSpec.Tag
				securityGroupDescription := securityGroupSpec.Description
				securityGroup := osc.CreateSecurityGroupResponse{
					SecurityGroup: &osc.SecurityGroup{
						SecurityGroupId: &securityGroupId,
					},
				}
				mockOscSecurityGroupInterface.
					EXPECT().
					GetSecurityGroupIdsFromNetIds(gomock.Any(), gomock.Eq(netId)).
					Return(nil, sgtc.expGetSecurityGroupFromNetIdsErr)
				mockOscSecurityGroupInterface.
					EXPECT().
					CreateSecurityGroup(gomock.Any(), gomock.Eq(netId), gomock.Eq(clusterName), gomock.Eq(securityGroupName), gomock.Eq(securityGroupDescription), gomock.Eq(securityGroupTag)).
					Return(securityGroup.SecurityGroup, sgtc.expCreateSecurityGroupErr)
			}
			reconcileSecurityGroup, err := reconcileSecurityGroup(ctx, clusterScope, mockOscSecurityGroupInterface, mockOscTagInterface)
			if sgtc.expReconcileSecurityGroupErr != nil {
				require.EqualError(t, err, sgtc.expReconcileSecurityGroupErr.Error(), "reconcileSecurityGroup() should return the same error")
			} else {
				require.NoError(t, err)
			}

			t.Logf("find reconcileSecurityGroup %v\n", reconcileSecurityGroup)
		})
	}
}

// TestReconcileCreateSecurityGroupResourceId has several tests to cover the code of the function reconcileCreateSecurityGroup
func TestReconcileCreateSecurityGroupResourceId(t *testing.T) {
	securityGroupTestCases := []struct {
		name                             string
		spec                             infrastructurev1beta1.OscClusterSpec
		expTagFound                      bool
		expNetFound                      bool
		expReadTagErr                    error
		expGetSecurityGroupIdsFromNetIds error
		expReconcileSecurityGroupErr     error
	}{
		{
			name: "net does not exist",

			spec:                             defaultSecurityGroupReconcile,
			expTagFound:                      false,
			expNetFound:                      false,
			expReadTagErr:                    nil,
			expGetSecurityGroupIdsFromNetIds: nil,
			expReconcileSecurityGroupErr:     errors.New("test-net-uid does not exist"),
		},
		{
			name:                             "failed to get tag",
			spec:                             defaultSecurityGroupReconcile,
			expTagFound:                      true,
			expNetFound:                      true,
			expGetSecurityGroupIdsFromNetIds: nil,
			expReadTagErr:                    errors.New("ReadTag generic error"),
			expReconcileSecurityGroupErr:     errors.New("cannot get tag: ReadTag generic error"),
		},
	}
	for _, sgtc := range securityGroupTestCases {
		t.Run(sgtc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscSecurityGroupInterface, mockOscTagInterface := SetupWithSecurityGroupMock(t, sgtc.name, sgtc.spec)
			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			netName := sgtc.spec.Network.Net.Name + "-uid"
			netId := "vpc-" + netName
			if sgtc.expNetFound {
				netRef.ResourceMap[netName] = netId
			}
			securityGroupsSpec := sgtc.spec.Network.SecurityGroups
			var securityGroupIds []string

			if sgtc.expTagFound {
				for _, securityGroupSpec := range securityGroupsSpec {
					securityGroupName := securityGroupSpec.Name + "-uid"
					securityGroupId := "sg-" + securityGroupName
					securityGroupIds = append(securityGroupIds, securityGroupId)
					mockOscTagInterface.
						EXPECT().
						ReadTag(gomock.Any(), gomock.Eq("Name"), gomock.Eq(securityGroupName)).
						Return(nil, sgtc.expReadTagErr)
				}

				mockOscSecurityGroupInterface.
					EXPECT().
					GetSecurityGroupIdsFromNetIds(gomock.Any(), gomock.Eq(netId)).
					Return(securityGroupIds, sgtc.expGetSecurityGroupIdsFromNetIds)
			}

			reconcileSecurityGroup, err := reconcileSecurityGroup(ctx, clusterScope, mockOscSecurityGroupInterface, mockOscTagInterface)
			if sgtc.expReconcileSecurityGroupErr != nil {
				require.EqualError(t, err, sgtc.expReconcileSecurityGroupErr.Error(), "reconcileSecurityGroup() should return the same error")
			} else {
				require.NoError(t, err)
			}

			t.Logf("find reconcileSecurityGroup %v\n", reconcileSecurityGroup)
		})
	}
}

// TestDeleteSecurityGroup has several tests to cover the code of the function deleteSecurityGroup
func TestDeleteSecurityGroup(t *testing.T) {
	securityGroupTestCases := []struct {
		name                               string
		spec                               infrastructurev1beta1.OscClusterSpec
		expSecurityGroupFound              bool
		expDeleteSecurityGroupFirstMockErr error
		expDeleteSecurityGroupError        error
	}{
		{
			name:                               "delete securityGroup",
			spec:                               defaultSecurityGroupReconcile,
			expSecurityGroupFound:              true,
			expDeleteSecurityGroupFirstMockErr: nil,
			expDeleteSecurityGroupError:        nil,
		},
		{
			name:                               "delete securityGroup unmatch to catch",
			spec:                               defaultSecurityGroupReconcile,
			expSecurityGroupFound:              true,
			expDeleteSecurityGroupFirstMockErr: errors.New("DeleteSecurityGroup first generic error"),
			expDeleteSecurityGroupError:        errors.New("cannot delete securityGroup: DeleteSecurityGroup first generic error"),
		},
		{
			name:                               "waiting loadbalancer to timeout",
			spec:                               defaultSecurityGroupReconcile,
			expSecurityGroupFound:              true,
			expDeleteSecurityGroupFirstMockErr: security.ErrResourceConflict,
			expDeleteSecurityGroupError:        errors.New("timeout trying to delete securityGroup: " + security.ErrResourceConflict.Error()),
		},
	}
	for _, sgtc := range securityGroupTestCases {
		t.Run(sgtc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscSecurityGroupInterface, _ := SetupWithSecurityGroupMock(t, sgtc.name, sgtc.spec)
			var err error
			securityGroupsSpec := sgtc.spec.Network.SecurityGroups
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			for _, securityGroupSpec := range securityGroupsSpec {
				securityGroupName := securityGroupSpec.Name + "-uid"
				securityGroupId := "sg-" + securityGroupName
				securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
				mockOscSecurityGroupInterface.
					EXPECT().
					DeleteSecurityGroup(gomock.Any(), gomock.Eq(securityGroupId)).
					Return(sgtc.expDeleteSecurityGroupFirstMockErr).MinTimes(1)
				clock_mock := clock.NewMock()
				wg := sync.WaitGroup{}
				wg.Add(1)
				go func() {
					clock_mock.Add(630 * time.Second)
					wg.Done()
				}()
				clock_mock.Sleep(time.Second)
				_, err = deleteSecurityGroup(ctx, clusterScope, securityGroupId, mockOscSecurityGroupInterface, clock_mock)
				if sgtc.expDeleteSecurityGroupError != nil {
					require.EqualError(t, err, sgtc.expDeleteSecurityGroupError.Error(), "deleteSecurityGroup() should return the right error")
				} else {
					require.NoError(t, err)
				}
				wg.Wait()
			}
		})
	}
}

// TestReconcileDeleteSecurityGroup has several tests to cover the code of the function reconcileDeleteSecurityGroup
func TestReconcileDeleteSecurityGroup(t *testing.T) {
	securityGroupTestCases := []struct {
		name                               string
		spec                               infrastructurev1beta1.OscClusterSpec
		expNetFound                        bool
		expSecurityGroupFound              bool
		expSecurityGroupRuleFound          bool
		expDeleteSecurityGroupFound        bool
		expDeleteSecurityGroupRuleFound    bool
		expSecurityGroupHasRuleErr         error
		expGetSecurityGroupFromNetIdsErr   error
		expDeleteSecurityGroupRuleErr      error
		expReconcileDeleteSecurityGroupErr error
	}{
		{
			name:                               "failed to delete securityGroupRule",
			spec:                               defaultSecurityGroupReconcile,
			expNetFound:                        true,
			expSecurityGroupFound:              true,
			expSecurityGroupRuleFound:          true,
			expDeleteSecurityGroupFound:        false,
			expDeleteSecurityGroupRuleFound:    false,
			expGetSecurityGroupFromNetIdsErr:   nil,
			expSecurityGroupHasRuleErr:         nil,
			expDeleteSecurityGroupRuleErr:      errors.New("DeleteSecurityGroupRule generic error"),
			expReconcileDeleteSecurityGroupErr: errors.New("cannot delete securityGroupRule: DeleteSecurityGroupRule generic error"),
		},
	}
	for _, sgtc := range securityGroupTestCases {
		t.Run(sgtc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscSecurityGroupInterface, _ := SetupWithSecurityGroupMock(t, sgtc.name, sgtc.spec)

			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			netName := sgtc.spec.Network.Net.Name + "-uid"
			netId := "vpc-" + netName
			if sgtc.expNetFound {
				netRef.ResourceMap[netName] = netId
			}

			securityGroupsSpec := sgtc.spec.Network.SecurityGroups
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			var securityGroupIds []string

			if sgtc.expNetFound {
				for _, securityGroupSpec := range securityGroupsSpec {
					securityGroupName := securityGroupSpec.Name + "-uid"
					securityGroupId := "sg-" + securityGroupName
					securityGroupIds = append(securityGroupIds, securityGroupId)
					securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
					if sgtc.expSecurityGroupFound {
						mockOscSecurityGroupInterface.
							EXPECT().
							GetSecurityGroupIdsFromNetIds(gomock.Any(), gomock.Eq(netId)).
							Return(securityGroupIds, sgtc.expGetSecurityGroupFromNetIdsErr)
					} else {
						mockOscSecurityGroupInterface.
							EXPECT().
							GetSecurityGroupIdsFromNetIds(gomock.Any(), gomock.Eq(netId)).
							Return(nil, sgtc.expGetSecurityGroupFromNetIdsErr)
					}

					if sgtc.expSecurityGroupFound {
						securityGroupRulesSpec := securityGroupSpec.SecurityGroupRules
						for _, securityGroupRuleSpec := range securityGroupRulesSpec {
							securityGroupRuleFlow := securityGroupRuleSpec.Flow
							securityGroupRuleIpProtocol := securityGroupRuleSpec.IpProtocol
							securityGroupRuleIpRange := securityGroupRuleSpec.IpRange
							securityGroupRuleFromPortRange := securityGroupRuleSpec.FromPortRange
							securityGroupRuleToPortRange := securityGroupRuleSpec.ToPortRange
							securityGroupMemberId := ""

							if sgtc.expSecurityGroupRuleFound {
								mockOscSecurityGroupInterface.
									EXPECT().
									SecurityGroupHasRule(gomock.Any(), gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIpProtocol), gomock.Eq(securityGroupRuleIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
									Return(true, sgtc.expSecurityGroupHasRuleErr)
							} else {
								mockOscSecurityGroupInterface.
									EXPECT().
									SecurityGroupHasRule(gomock.Any(), gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIpProtocol), gomock.Eq(securityGroupRuleIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
									Return(false, sgtc.expSecurityGroupHasRuleErr)
							}
							mockOscSecurityGroupInterface.
								EXPECT().
								DeleteSecurityGroupRule(gomock.Any(), gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIpProtocol), gomock.Eq(securityGroupRuleIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
								Return(sgtc.expDeleteSecurityGroupRuleErr)
						}
					}
				}
			}
			reconcileDeleteSecurityGroup, err := reconcileDeleteSecurityGroup(ctx, clusterScope, mockOscSecurityGroupInterface)
			if sgtc.expReconcileDeleteSecurityGroupErr != nil {
				require.EqualError(t, err, sgtc.expReconcileDeleteSecurityGroupErr.Error(), "reconcileDeleteSecurityGroup() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("find reconcileDeleteSecurityGroup %v\n", reconcileDeleteSecurityGroup)
		})
	}
}

// TestReconcileDeleteSecurityGroupDelete has several tests to cover the code of the function reconcileDeleteSecurityGroup
func TestReconcileDeleteSecurityGroupDelete(t *testing.T) {
	securityGroupTestCases := []struct {
		name                               string
		spec                               infrastructurev1beta1.OscClusterSpec
		expSecurityGroupFound              bool
		expGetSecurityGroupFromNetIdsErr   error
		expSecurityGroupHasRuleErr         error
		expDeleteSecurityGroupRuleErr      error
		expDeleteSecurityGroupErr          error
		expReconcileDeleteSecurityGroupErr error
	}{
		{
			name:                               "delete securityGroup",
			spec:                               defaultSecurityGroupReconcile,
			expSecurityGroupFound:              true,
			expGetSecurityGroupFromNetIdsErr:   nil,
			expSecurityGroupHasRuleErr:         nil,
			expDeleteSecurityGroupRuleErr:      nil,
			expDeleteSecurityGroupErr:          nil,
			expReconcileDeleteSecurityGroupErr: nil,
		},
		{
			name:                               "failed to delete securityGroup",
			spec:                               defaultSecurityGroupReconcile,
			expSecurityGroupFound:              true,
			expGetSecurityGroupFromNetIdsErr:   nil,
			expSecurityGroupHasRuleErr:         nil,
			expDeleteSecurityGroupRuleErr:      nil,
			expDeleteSecurityGroupErr:          errors.New("DeleteSecurityGroup error"),
			expReconcileDeleteSecurityGroupErr: errors.New("cannot delete securityGroup: DeleteSecurityGroup error"),
		},
	}
	for _, sgtc := range securityGroupTestCases {
		t.Run(sgtc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscSecurityGroupInterface, _ := SetupWithSecurityGroupMock(t, sgtc.name, sgtc.spec)

			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			netName := sgtc.spec.Network.Net.Name + "-uid"
			netId := "vpc-" + netName
			netRef.ResourceMap[netName] = netId

			securityGroupsSpec := sgtc.spec.Network.SecurityGroups
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)

			var securityGroupIds []string
			for _, securityGroupSpec := range securityGroupsSpec {
				securityGroupName := securityGroupSpec.Name + "-uid"
				securityGroupId := "sg-" + securityGroupName
				securityGroupIds = append(securityGroupIds, securityGroupId)
				securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
				mockOscSecurityGroupInterface.
					EXPECT().
					GetSecurityGroupIdsFromNetIds(gomock.Any(), gomock.Eq(netId)).
					Return(securityGroupIds, sgtc.expGetSecurityGroupFromNetIdsErr)
				securityGroupRulesSpec := securityGroupSpec.SecurityGroupRules
				for _, securityGroupRuleSpec := range securityGroupRulesSpec {
					securityGroupRuleFlow := securityGroupRuleSpec.Flow
					securityGroupRuleIpProtocol := securityGroupRuleSpec.IpProtocol
					securityGroupRuleIpRange := securityGroupRuleSpec.IpRange
					securityGroupRuleFromPortRange := securityGroupRuleSpec.FromPortRange
					securityGroupRuleToPortRange := securityGroupRuleSpec.ToPortRange
					securityGroupMemberId := ""

					mockOscSecurityGroupInterface.
						EXPECT().
						SecurityGroupHasRule(gomock.Any(), gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIpProtocol), gomock.Eq(securityGroupRuleIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
						Return(true, sgtc.expSecurityGroupHasRuleErr)
					mockOscSecurityGroupInterface.
						EXPECT().
						DeleteSecurityGroupRule(gomock.Any(), gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIpProtocol), gomock.Eq(securityGroupRuleIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
						Return(sgtc.expDeleteSecurityGroupRuleErr)
				}
				mockOscSecurityGroupInterface.
					EXPECT().
					DeleteSecurityGroup(gomock.Any(), gomock.Eq(securityGroupId)).
					Return(sgtc.expDeleteSecurityGroupErr)
			}
			reconcileDeleteSecurityGroup, err := reconcileDeleteSecurityGroup(ctx, clusterScope, mockOscSecurityGroupInterface)
			if sgtc.expReconcileDeleteSecurityGroupErr != nil {
				require.EqualError(t, err, sgtc.expReconcileDeleteSecurityGroupErr.Error(), "reconcileDeleteSecurityGroup() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("find reconcileDeleteSecurityGroup %v\n", reconcileDeleteSecurityGroup)
		})
	}
}

// TestReconcileDeleteSecurityGroupDeleteWithoutSpec has several tests to cover the code of the function reconcileDeleteSecurityGroup
func TestReconcileDeleteSecurityGroupDeleteWithoutSpec(t *testing.T) {
	securityGroupTestCases := []struct {
		name                               string
		spec                               infrastructurev1beta1.OscClusterSpec
		expSecurityGroupHasRuleErr         error
		expGetSecurityGroupFromNetIdsErr   error
		expDeleteSecurityGroupRuleErr      error
		expDeleteSecurityGroupErr          error
		expReconcileDeleteSecurityGroupErr error
	}{
		{
			name:                               "delete securityGroup without spec (with default values)",
			expSecurityGroupHasRuleErr:         nil,
			expGetSecurityGroupFromNetIdsErr:   nil,
			expDeleteSecurityGroupRuleErr:      nil,
			expDeleteSecurityGroupErr:          nil,
			expReconcileDeleteSecurityGroupErr: nil,
		},
	}
	for _, sgtc := range securityGroupTestCases {
		t.Run(sgtc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscSecurityGroupInterface, _ := SetupWithSecurityGroupMock(t, sgtc.name, sgtc.spec)

			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			netName := "cluster-api-net-uid"
			netId := "vpc-" + netName
			netRef.ResourceMap[netName] = netId

			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)

			var securityGroupIds []string
			securityGroupName := "cluster-api-securitygroup-kw-uid"
			securityGroupId := "sg-" + securityGroupName
			securityGroupIds = append(securityGroupIds, securityGroupId)
			securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
			securityGroupRuleKubeletKwFlow := "Inbound"
			securityGroupRuleKubeletKwIpProtocol := "tcp"
			securityGroupRuleKubeletKwIpRange := "10.0.3.0/24"
			securityGroupMemberId := ""
			var securityGroupRuleKubeletKwFromPortRange int32 = 10250
			var securityGroupRuleKubeletKwToPortRange int32 = 10250

			securityGroupRuleNodeIpKwFlow := "Inbound"
			securityGroupRuleNodeIpKwIpProtocol := "tcp"
			securityGroupRuleNodeIpKwIpRange := "10.0.3.0/24"
			var securityGroupRuleNodeIpKwFromPortRange int32 = 30000
			var securityGroupRuleNodeIpKwToPortRange int32 = 32767

			securityGroupRuleNodeIpKcpFlow := "Inbound"
			securityGroupRuleNodeIpKcpIpProtocol := "tcp"
			securityGroupRuleNodeIpKcpIpRange := "10.0.4.0/24"
			var securityGroupRuleNodeIpKcpFromPortRange int32 = 30000
			var securityGroupRuleNodeIpKcpToPortRange int32 = 32767

			securityGroupRuleKubeletKcpFlow := "Inbound"
			securityGroupRuleKubeletKcpIpProtocol := "tcp"
			securityGroupRuleKubeletKcpIpRange := "10.0.4.0/24"
			var securityGroupRuleKubeletKcpFromPortRange int32 = 10250
			var securityGroupRuleKubeletKcpToPortRange int32 = 10250

			securityGroupRuleKcpBgpFlow := "Inbound"
			securityGroupRuleKcpBgpIpProtocol := "tcp"
			securityGroupRuleKcpBgpIpRange := "10.0.0.0/16"
			var securityGroupRuleKcpBgpFromPortRange int32 = 179
			var securityGroupRuleKcpBgpToPortRange int32 = 179

			mockOscSecurityGroupInterface.
				EXPECT().
				GetSecurityGroupIdsFromNetIds(gomock.Any(), gomock.Eq(netId)).
				Return(securityGroupIds, sgtc.expGetSecurityGroupFromNetIdsErr)

			mockOscSecurityGroupInterface.
				EXPECT().
				SecurityGroupHasRule(gomock.Any(), gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleKubeletKwFlow), gomock.Eq(securityGroupRuleKubeletKwIpProtocol), gomock.Eq(securityGroupRuleKubeletKwIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleKubeletKwFromPortRange), gomock.Eq(securityGroupRuleKubeletKwToPortRange)).
				Return(true, sgtc.expSecurityGroupHasRuleErr)
			mockOscSecurityGroupInterface.
				EXPECT().
				DeleteSecurityGroupRule(gomock.Any(), gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleKubeletKwFlow), gomock.Eq(securityGroupRuleKubeletKwIpProtocol), gomock.Eq(securityGroupRuleKubeletKwIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleKubeletKwFromPortRange), gomock.Eq(securityGroupRuleKubeletKwToPortRange)).
				Return(sgtc.expDeleteSecurityGroupRuleErr)

			mockOscSecurityGroupInterface.
				EXPECT().
				SecurityGroupHasRule(gomock.Any(), gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleKubeletKcpFlow), gomock.Eq(securityGroupRuleKubeletKcpIpProtocol), gomock.Eq(securityGroupRuleKubeletKcpIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleKubeletKcpFromPortRange), gomock.Eq(securityGroupRuleKubeletKcpToPortRange)).
				Return(true, sgtc.expSecurityGroupHasRuleErr)

			mockOscSecurityGroupInterface.
				EXPECT().
				DeleteSecurityGroupRule(gomock.Any(), gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleKubeletKcpFlow), gomock.Eq(securityGroupRuleKubeletKcpIpProtocol), gomock.Eq(securityGroupRuleKubeletKcpIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleKubeletKcpFromPortRange), gomock.Eq(securityGroupRuleKubeletKcpToPortRange)).
				Return(sgtc.expDeleteSecurityGroupRuleErr)

			mockOscSecurityGroupInterface.
				EXPECT().
				SecurityGroupHasRule(gomock.Any(), gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleNodeIpKwFlow), gomock.Eq(securityGroupRuleNodeIpKwIpProtocol), gomock.Eq(securityGroupRuleNodeIpKwIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleNodeIpKwFromPortRange), gomock.Eq(securityGroupRuleNodeIpKwToPortRange)).
				Return(true, sgtc.expSecurityGroupHasRuleErr)
			mockOscSecurityGroupInterface.
				EXPECT().
				DeleteSecurityGroupRule(gomock.Any(), gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleNodeIpKwFlow), gomock.Eq(securityGroupRuleNodeIpKwIpProtocol), gomock.Eq(securityGroupRuleNodeIpKwIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleNodeIpKwFromPortRange), gomock.Eq(securityGroupRuleNodeIpKwToPortRange)).
				Return(sgtc.expDeleteSecurityGroupRuleErr)
			mockOscSecurityGroupInterface.
				EXPECT().
				SecurityGroupHasRule(gomock.Any(), gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleNodeIpKcpFlow), gomock.Eq(securityGroupRuleNodeIpKcpIpProtocol), gomock.Eq(securityGroupRuleNodeIpKcpIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleNodeIpKcpFromPortRange), gomock.Eq(securityGroupRuleNodeIpKcpToPortRange)).
				Return(true, sgtc.expSecurityGroupHasRuleErr)
			mockOscSecurityGroupInterface.
				EXPECT().
				DeleteSecurityGroupRule(gomock.Any(), gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleNodeIpKcpFlow), gomock.Eq(securityGroupRuleNodeIpKcpIpProtocol), gomock.Eq(securityGroupRuleNodeIpKcpIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleNodeIpKcpFromPortRange), gomock.Eq(securityGroupRuleNodeIpKcpToPortRange)).
				Return(sgtc.expDeleteSecurityGroupRuleErr)

			mockOscSecurityGroupInterface.
				EXPECT().
				SecurityGroupHasRule(gomock.Any(), gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleKcpBgpFlow), gomock.Eq(securityGroupRuleKcpBgpIpProtocol), gomock.Eq(securityGroupRuleKcpBgpIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleKcpBgpFromPortRange), gomock.Eq(securityGroupRuleKcpBgpToPortRange)).
				Return(true, sgtc.expSecurityGroupHasRuleErr)
			mockOscSecurityGroupInterface.
				EXPECT().
				DeleteSecurityGroupRule(gomock.Any(), gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleKcpBgpFlow), gomock.Eq(securityGroupRuleKcpBgpIpProtocol), gomock.Eq(securityGroupRuleKcpBgpIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleKcpBgpFromPortRange), gomock.Eq(securityGroupRuleKcpBgpToPortRange)).
				Return(sgtc.expDeleteSecurityGroupRuleErr)

			mockOscSecurityGroupInterface.
				EXPECT().
				DeleteSecurityGroup(gomock.Any(), gomock.Eq(securityGroupId)).
				Return(sgtc.expDeleteSecurityGroupErr)
			reconcileDeleteSecurityGroup, err := reconcileDeleteSecurityGroup(ctx, clusterScope, mockOscSecurityGroupInterface)
			if sgtc.expReconcileDeleteSecurityGroupErr != nil {
				require.EqualError(t, err, sgtc.expReconcileDeleteSecurityGroupErr.Error(), "reconcileDeleteSecurityGroup() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("find reconcileDeleteSecurityGroup %v\n", reconcileDeleteSecurityGroup)
		})
	}
}

// TestReconcileDeleteSecurityGroupGet has several tests to cover the code of the function reconcileDeleteSecurityGroup
func TestReconcileDeleteSecurityGroupGet(t *testing.T) {
	securityGroupTestCases := []struct {
		name                               string
		spec                               infrastructurev1beta1.OscClusterSpec
		expNetFound                        bool
		expSecurityGroupFound              bool
		expGetSecurityGroupFromNetIdsErr   error
		expReconcileDeleteSecurityGroupErr error
	}{
		{
			name:                               "failed to get securityGroup",
			spec:                               defaultSecurityGroupReconcile,
			expNetFound:                        true,
			expSecurityGroupFound:              false,
			expGetSecurityGroupFromNetIdsErr:   errors.New("GetSecurityGroup generic error"),
			expReconcileDeleteSecurityGroupErr: errors.New("GetSecurityGroup generic error"),
		},
		{
			name:                               "remove finalizer (user delete securityGroup without cluster-api)",
			spec:                               defaultSecurityGroupReconcile,
			expNetFound:                        true,
			expSecurityGroupFound:              false,
			expGetSecurityGroupFromNetIdsErr:   nil,
			expReconcileDeleteSecurityGroupErr: nil,
		},
	}
	for _, sgtc := range securityGroupTestCases {
		t.Run(sgtc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscSecurityGroupInterface, _ := SetupWithSecurityGroupMock(t, sgtc.name, sgtc.spec)

			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			netName := sgtc.spec.Network.Net.Name + "-uid"
			netId := "vpc-" + netName
			if sgtc.expNetFound {
				netRef.ResourceMap[netName] = netId
			}

			securityGroupsSpec := sgtc.spec.Network.SecurityGroups
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)

			var securityGroupIds []string
			if sgtc.expNetFound {
				for _, securityGroupSpec := range securityGroupsSpec {
					securityGroupName := securityGroupSpec.Name + "-uid"
					securityGroupId := "sg-" + securityGroupName
					securityGroupIds = append(securityGroupIds, securityGroupId)
					securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
					if sgtc.expSecurityGroupFound {
						mockOscSecurityGroupInterface.
							EXPECT().
							GetSecurityGroupIdsFromNetIds(gomock.Any(), gomock.Eq(netId)).
							Return(securityGroupIds, sgtc.expGetSecurityGroupFromNetIdsErr)
					} else {
						mockOscSecurityGroupInterface.
							EXPECT().
							GetSecurityGroupIdsFromNetIds(gomock.Any(), gomock.Eq(netId)).
							Return(nil, sgtc.expGetSecurityGroupFromNetIdsErr)
					}
				}
			}
			reconcileDeleteSecurityGroup, err := reconcileDeleteSecurityGroup(ctx, clusterScope, mockOscSecurityGroupInterface)
			if sgtc.expReconcileDeleteSecurityGroupErr != nil {
				require.EqualError(t, err, sgtc.expReconcileDeleteSecurityGroupErr.Error(), "reconcileDeleteSecurityGroup() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("find reconcileDeleteSecurityGroup %v\n", reconcileDeleteSecurityGroup)
		})
	}
}

// TestReconcileDeleteSecurityGroup_NoNetKnown tests that reconciliation succeeds if no net is known
func TestReconcileDeleteSecurityGroup_NoNetKnown(t *testing.T) {
	securityGroupTestCases := []struct {
		name                               string
		spec                               infrastructurev1beta1.OscClusterSpec
		expReconcileDeleteSecurityGroupErr error
	}{
		{
			name: "net does not exist",
			spec: defaultSecurityGroupReconcile,
		},
		{
			name: "check failed without net and securityGroup spec (retrieve default values cluster-api)",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{},
			},
		},
	}
	for _, sgtc := range securityGroupTestCases {
		t.Run(sgtc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscSecurityGroupInterface, _ := SetupWithSecurityGroupMock(t, sgtc.name, sgtc.spec)
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			reconcileDeleteSecurityGroup, err := reconcileDeleteSecurityGroup(ctx, clusterScope, mockOscSecurityGroupInterface)
			if sgtc.expReconcileDeleteSecurityGroupErr != nil {
				require.EqualError(t, err, sgtc.expReconcileDeleteSecurityGroupErr.Error(), "reconcileDeleteSecurityGroup() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("find reconcileDeleteSecurityGroup %v\n", reconcileDeleteSecurityGroup)
		})
	}
}
