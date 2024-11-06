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

	//"github.com/benbjohnson/clock"
	"github.com/golang/mock/gomock"
	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/security/mock_security"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag/mock_tag"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

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
)

// Helper function for initializing a default security group spec
func defaultSecurityGroupSpec() infrastructurev1beta1.OscClusterSpec {
	return infrastructurev1beta1.OscClusterSpec{
		Network: infrastructurev1beta1.OscNetwork{
			SecurityGroups: []*infrastructurev1beta1.OscSecurityGroup{
				{
					Name: "test-securitygroup",
					SecurityGroupRules: []infrastructurev1beta1.OscSecurityGroupRule{ // Use value instead of pointer
						{
							Name:          "test-rule",
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
}

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
			expGetSecurityGroupResourceIdErr: fmt.Errorf("test-securitygroup-uid does not exist (yet)"),
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
				if err != nil {
					assert.Equal(t, sgtc.expGetSecurityGroupResourceIdErr, err, "getSecurityGroupResourceId() should return the same error")
				} else {
					assert.Nil(t, sgtc.expGetSecurityGroupResourceIdErr)
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
			expGetSecurityGroupRuleResourceIdErr: fmt.Errorf("test-securitygrouprule-uid does not exist"),
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
					if err != nil {
						assert.Equal(t, sgrtc.expGetSecurityGroupRuleResourceIdErr, err, "getSecurityGroupRuleResourceId() should return the same error")
					} else {
						assert.Nil(t, sgrtc.expGetSecurityGroupRuleResourceIdErr)
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
			expCheckSecurityGroupOscDuplicateNameErr: fmt.Errorf("test-securitygroup already exist"),
		},
	}
	for _, sgtc := range securityGroupTestCases {
		t.Run(sgtc.name, func(t *testing.T) {
			clusterScope := Setup(t, sgtc.name, sgtc.spec)
			duplicateResourceSecurityGroupNameErr := checkSecurityGroupOscDuplicateName(clusterScope)
			if duplicateResourceSecurityGroupNameErr != nil {
				assert.Equal(t, sgtc.expCheckSecurityGroupOscDuplicateNameErr, duplicateResourceSecurityGroupNameErr, "checkSecurityGroupOscDuplicateName() should return the same error")
			} else {
				assert.Nil(t, sgtc.expCheckSecurityGroupOscDuplicateNameErr)
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
			expCheckSecurityGroupRuleOscDuplicateNameErr: fmt.Errorf("test-securitygrouprule already exist"),
		},
	}
	for _, sgrtc := range securityGroupRuleTestCases {
		t.Run(sgrtc.name, func(t *testing.T) {
			clusterScope := Setup(t, sgrtc.name, sgrtc.spec)
			duplicateResourceSecurityGroupRuleNameErr := checkSecurityGroupRuleOscDuplicateName(clusterScope)
			if duplicateResourceSecurityGroupRuleNameErr != nil {
				assert.Equal(t, sgrtc.expCheckSecurityGroupRuleOscDuplicateNameErr, duplicateResourceSecurityGroupRuleNameErr, "checkSecurityGroupRuleOscDuplicateName() should return the same error")
			} else {
				assert.Nil(t, sgrtc.expCheckSecurityGroupRuleOscDuplicateNameErr)
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
			expCheckSecurityGroupFormatParametersErr: fmt.Errorf("Invalid Tag Name"),
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
			expCheckSecurityGroupFormatParametersErr: fmt.Errorf("Invalid Description"),
		},
	}
	for _, sgtc := range securityGroupTestCases {
		t.Run(sgtc.name, func(t *testing.T) {
			clusterScope := Setup(t, sgtc.name, sgtc.spec)
			_, err := checkSecurityGroupFormatParameters(clusterScope)
			if err != nil {
				assert.Equal(t, sgtc.expCheckSecurityGroupFormatParametersErr, err, "checkSecurityGroupFormatParameters() should return the same error")
			} else {
				assert.Nil(t, sgtc.expCheckSecurityGroupFormatParametersErr)
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
			expCheckSecurityGroupRuleFormatParametersErr: fmt.Errorf("Invalid Tag Name"),
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
			expCheckSecurityGroupRuleFormatParametersErr: fmt.Errorf("Invalid flow"),
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
			expCheckSecurityGroupRuleFormatParametersErr: fmt.Errorf("Invalid protocol"),
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
			expCheckSecurityGroupRuleFormatParametersErr: fmt.Errorf("invalid CIDR address: 10.0.0.0/36"),
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
			expCheckSecurityGroupRuleFormatParametersErr: fmt.Errorf("invalid CIDR address: 10.0.0.256/16"),
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
			expCheckSecurityGroupRuleFormatParametersErr: fmt.Errorf("Invalid Port"),
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
			expCheckSecurityGroupRuleFormatParametersErr: fmt.Errorf("Invalid Port"),
		},
	}
	for _, sgrtc := range securityGroupRuleTestCases {
		t.Run(sgrtc.name, func(t *testing.T) {
			clusterScope := Setup(t, sgrtc.name, sgrtc.spec)
			_, err := checkSecurityGroupRuleFormatParameters(clusterScope)
			if err != nil {
				assert.Equal(t, sgrtc.expCheckSecurityGroupRuleFormatParametersErr.Error(), err.Error(), "checkSecurityGroupRuleFormatParameters() should return the same error")
			} else {
				assert.Nil(t, sgrtc.expCheckSecurityGroupRuleFormatParametersErr)
			}
			t.Logf("find all securityGroupRule")
		})
	}
}

func TestReconcileSecurityGroupRuleGet(t *testing.T) {
	testCases := []struct {
		name                     string
		spec                     infrastructurev1beta1.OscClusterSpec
		expectCreateSecurityRule bool
		expectedError            error
	}{
		{
			name:                     "create_securityGroupRule_first_time",
			spec:                     defaultSecurityGroupSpec(),
			expectCreateSecurityRule: true,
			expectedError:            nil,
		},
		{
			name:                     "create_securityGroupRule_already_exists",
			spec:                     defaultSecurityGroupSpec(),
			expectCreateSecurityRule: true,
			expectedError:            nil, // Expect no error even if rule exists (409 conflict)
		},
		{
			name:                     "create_securityGroupRule_error",
			spec:                     defaultSecurityGroupSpec(),
			expectCreateSecurityRule: true,
			expectedError:            fmt.Errorf("CreateSecurityGroupRule error"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up mocks and context
			clusterScope, ctx, mockOscSecurityGroupInterface, mockTagSvc := SetupWithSecurityGroupMock(t, tc.name, tc.spec)
			securityGroupID := "sg-test-securitygroup-uid"

			// Explicitly populate ResourceMap with a mock security group ID to prevent the empty error
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = map[string]string{
				"test-securitygroup-uid": securityGroupID,
			}

			// Mock `CreateSecurityGroupRule`
			if tc.expectCreateSecurityRule {
				if tc.expectedError == nil {
					// Simulate 409 Conflict for already existing rule, should be ignored by the function
					mockOscSecurityGroupInterface.EXPECT().
						CreateSecurityGroupRule(securityGroupID, "Inbound", "tcp", "0.0.0.0/0", "", int32(6443), int32(6443)).
						Return(nil, fmt.Errorf("409 Conflict: Rule already exists"))
				} else {
					// Simulate an error other than 409 Conflict
					mockOscSecurityGroupInterface.EXPECT().
						CreateSecurityGroupRule(securityGroupID, "Inbound", "tcp", "0.0.0.0/0", "", int32(6443), int32(6443)).
						Return(nil, tc.expectedError)
				}
			}

			// Run the reconcile function
			result, err := reconcileSecurityGroupRule(ctx, clusterScope, mockOscSecurityGroupInterface, mockTagSvc)

			// Validate results
			if tc.expectedError != nil && tc.expectedError.Error() != "409 Conflict: Rule already exists" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError.Error())
			} else {
				assert.NoError(t, err) // Should ignore 409 Conflict error
			}
			assert.Equal(t, reconcile.Result{}, result)
		})
	}
}

// TestReconcileDeleteSecurityGroupRuleDelete has several tests to cover the code of the function reconcileDeleteSecurityGroupsRule
func TestReconcileDeleteSecurityGroupRuleDelete(t *testing.T) {
	securityGroupRuleTestCases := []struct {
		name                                        string
		spec                                        infrastructurev1beta1.OscClusterSpec
		expGetSecurityGroupfromSecurityGroupRuleErr error
		expDeleteSecurityGroupRuleErr               error
		expReconcileDeleteSecurityGroupRuleErr      error
	}{
		{
			name: "failed to delete securityGroupRule",
			spec: defaultSecurityGroupReconcile,
			expGetSecurityGroupfromSecurityGroupRuleErr: nil,
			expDeleteSecurityGroupRuleErr:               fmt.Errorf("DeleteSecurityGroupRule generic error"),
			expReconcileDeleteSecurityGroupRuleErr:      fmt.Errorf("DeleteSecurityGroupRule generic error cannot delete securityGroupRule for OscCluster test-system/test-osc"),
		},
		{
			name: "delete securityGroupRule",
			spec: defaultSecurityGroupReconcile,
			expGetSecurityGroupfromSecurityGroupRuleErr: nil,
			expDeleteSecurityGroupRuleErr:               nil,
			expReconcileDeleteSecurityGroupRuleErr:      nil,
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
					mockOscSecurityGroupInterface.
						EXPECT().
						GetSecurityGroupFromSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIpProtocol), gomock.Eq(securityGroupRuleIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
						Return(&readSecurityGroup[0], sgrtc.expGetSecurityGroupfromSecurityGroupRuleErr)
					mockOscSecurityGroupInterface.
						EXPECT().
						DeleteSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIpProtocol), gomock.Eq(securityGroupRuleIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
						Return(sgrtc.expDeleteSecurityGroupRuleErr)
					reconcileDeleteSecurityGroupsRule, err := reconcileDeleteSecurityGroupsRule(ctx, clusterScope, securityGroupRuleSpec, securityGroupName, mockOscSecurityGroupInterface)
					if err != nil {
						assert.Equal(t, sgrtc.expReconcileDeleteSecurityGroupRuleErr.Error(), err.Error(), "reconcileDeleteSecuritygroupRules() should return the same error")
					} else {
						assert.Nil(t, sgrtc.expReconcileDeleteSecurityGroupRuleErr)
					}
					t.Logf("find reconcileDeleteSecurityGroupsRule %v\n", reconcileDeleteSecurityGroupsRule)
				}
			}
		})
	}
}

// / TestReconcileSecurityGroupRuleCreate has several tests to cover the code of the function reconcileSecurityGroupRule
func TestReconcileSecurityGroupRuleCreate(t *testing.T) {
	testCases := []struct {
		name                     string
		spec                     infrastructurev1beta1.OscClusterSpec
		expectedError            error
		expectGetSecurityGroup   bool
		expectCreateSecurityRule bool
	}{
		{
			name:                     "create_securityGroupRule_first_time",
			spec:                     defaultSecurityGroupSpec(),
			expectGetSecurityGroup:   true,
			expectCreateSecurityRule: true,
			expectedError:            nil,
		},
		{
			name:                     "failed_to_create_securityGroupRule",
			spec:                     defaultSecurityGroupSpec(),
			expectGetSecurityGroup:   true,
			expectCreateSecurityRule: false,
			expectedError:            fmt.Errorf("securityGroupsRef.ResourceMap is empty, security groups should be reconciled first"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up the cluster scope and initialize the ResourceMap
			clusterScope, ctx, mockSecurityGroupInterface, mockTagInterface := SetupWithSecurityGroupMock(t, tc.name, tc.spec)
			securityGroupID := "sg-test-securitygroup-uid"

			if tc.name != "failed_to_create_securityGroupRule" {
				clusterScope.GetSecurityGroupsRef().ResourceMap = map[string]string{
					"test-securitygroup-uid": securityGroupID,
				}
			}

			// Mock GetSecurityGroupFromSecurityGroupRule to simulate rule retrieval
			if tc.expectGetSecurityGroup {
				mockSecurityGroupInterface.EXPECT().
					GetSecurityGroupFromSecurityGroupRule(
						gomock.Eq(securityGroupID),
						gomock.Eq("Inbound"),
						gomock.Eq("tcp"),
						gomock.Eq("0.0.0.0/0"),
						gomock.Eq(""),
						gomock.Eq(int32(6443)),
						gomock.Eq(int32(6443)),
					).
					Return(nil, nil).AnyTimes() // Simulate that the rule doesn't exist yet
			}

			// Mock CreateSecurityGroupRule if the rule should be created
			if tc.expectCreateSecurityRule {
				mockSecurityGroupInterface.EXPECT().
					CreateSecurityGroupRule(
						gomock.Eq(securityGroupID),
						gomock.Eq("Inbound"),
						gomock.Eq("tcp"),
						gomock.Eq("0.0.0.0/0"),
						gomock.Eq(""),
						gomock.Eq(int32(6443)),
						gomock.Eq(int32(6443)),
					).
					Return(nil, tc.expectedError)
			}

			// Execute the reconciliation function
			result, err := reconcileSecurityGroupRule(ctx, clusterScope, mockSecurityGroupInterface, mockTagInterface)

			// Validate the expected results
			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "securityGroupsRef.ResourceMap is empty, security groups should be reconciled first")
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, reconcile.Result{}, result)

			t.Logf("Test %s result: %v, error: %v", tc.name, result, err)
		})
	}
}

// TestReconcileDeleteSecurityGroupRuleGet has several tests to cover the code of the function reconcileDeleteSecurityGroupsRule
func TestReconcileDeleteSecurityGroupRuleGet(t *testing.T) {
	securityGroupRuleTestCases := []struct {
		name                                        string
		spec                                        infrastructurev1beta1.OscClusterSpec
		expGetSecurityGroupfromSecurityGroupRuleErr error
		expReconcileDeleteSecurityGroupRuleErr      error
	}{
		{
			name: "failed to get securityGroupRule",
			spec: defaultSecurityGroupReconcile,
			expGetSecurityGroupfromSecurityGroupRuleErr: fmt.Errorf("GetSecurityGroupFromSecurityGroupRule generic errors"),

			expReconcileDeleteSecurityGroupRuleErr: fmt.Errorf("GetSecurityGroupFromSecurityGroupRule generic errors"),
		},
		{
			name: "remove finalizer (user delete securityGroup without cluster-api)",
			spec: defaultSecurityGroupReconcile,
			expGetSecurityGroupfromSecurityGroupRuleErr: nil,
			expReconcileDeleteSecurityGroupRuleErr:      nil,
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
						GetSecurityGroupFromSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIpProtocol), gomock.Eq(securityGroupRuleIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
						Return(nil, sgrtc.expGetSecurityGroupfromSecurityGroupRuleErr)
					reconcileDeleteSecurityGroupsRule, err := reconcileDeleteSecurityGroupsRule(ctx, clusterScope, securityGroupRuleSpec, securityGroupName, mockOscSecurityGroupInterface)
					if err != nil {
						assert.Equal(t, sgrtc.expReconcileDeleteSecurityGroupRuleErr, err, "reconcileDeleteSecuritygroupRules() should return the same error")
					} else {
						assert.Nil(t, sgrtc.expReconcileDeleteSecurityGroupRuleErr)
					}
					t.Logf("find reconcileDeleteSecurityGroupsRule %v\n", reconcileDeleteSecurityGroupsRule)
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
			expGetSecurityGroupFromNetIdsErr: fmt.Errorf("GetSecurityGroup generic error"),
			expReadTagErr:                    nil,
			expReconcileSecurityGroupErr:     fmt.Errorf("GetSecurityGroup generic error"),
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
			if netRef.ResourceMap == nil {
				netRef.ResourceMap = make(map[string]string)
			}
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
							ReadTag(gomock.Eq("Name"), gomock.Eq(securityGroupName)).
							Return(&tag, sgtc.expReadTagErr)
					} else {
						mockOscTagInterface.
							EXPECT().
							ReadTag(gomock.Eq("Name"), gomock.Eq(securityGroupName)).
							Return(nil, sgtc.expReadTagErr)
					}
				}
				securityGroupIds = append(securityGroupIds, securityGroupId)
				securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
				if sgtc.expSecurityGroupFound {
					mockOscSecurityGroupInterface.
						EXPECT().
						GetSecurityGroupIdsFromNetIds(gomock.Eq(netId)).
						Return(securityGroupIds, sgtc.expGetSecurityGroupFromNetIdsErr)
				} else {
					mockOscSecurityGroupInterface.
						EXPECT().
						GetSecurityGroupIdsFromNetIds(gomock.Eq(netId)).
						Return(nil, sgtc.expGetSecurityGroupFromNetIdsErr)
				}
				reconcileSecurityGroup, err := reconcileSecurityGroup(ctx, clusterScope, mockOscSecurityGroupInterface, mockOscTagInterface)
				if err != nil {
					assert.Equal(t, sgtc.expReconcileSecurityGroupErr, err, "reconcileSecurityGroup() should return the same error")
				} else {
					assert.Nil(t, sgtc.expReconcileSecurityGroupErr)
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
			expCreateSecurityGroupErr:        fmt.Errorf("CreateSecurityGroup generic error"),
			expReadTagErr:                    nil,
			expReconcileSecurityGroupErr:     fmt.Errorf("CreateSecurityGroup generic error cannot create securityGroup for Osccluster test-system/test-osc"),
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
						ReadTag(gomock.Eq("Name"), gomock.Eq(securityGroupName)).
						Return(&tag, sgtc.expReadTagErr)

				} else {
					mockOscTagInterface.
						EXPECT().
						ReadTag(gomock.Eq("Name"), gomock.Eq(securityGroupName)).
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
					GetSecurityGroupIdsFromNetIds(gomock.Eq(netId)).
					Return(nil, sgtc.expGetSecurityGroupFromNetIdsErr)
				mockOscSecurityGroupInterface.
					EXPECT().
					CreateSecurityGroup(gomock.Eq(netId), gomock.Eq(clusterName), gomock.Eq(securityGroupName), gomock.Eq(securityGroupDescription), gomock.Eq(securityGroupTag)).
					Return(securityGroup.SecurityGroup, sgtc.expCreateSecurityGroupErr)
			}
			reconcileSecurityGroup, err := reconcileSecurityGroup(ctx, clusterScope, mockOscSecurityGroupInterface, mockOscTagInterface)
			if err != nil {
				assert.Equal(t, sgtc.expReconcileSecurityGroupErr.Error(), err.Error(), "reconcileSecurityGroup() should return the same error")
			} else {
				assert.Nil(t, sgtc.expReconcileSecurityGroupErr)
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
			expReconcileSecurityGroupErr:     fmt.Errorf("test-net-uid does not exist"),
		},
		{
			name:                             "failed to get tag",
			spec:                             defaultSecurityGroupReconcile,
			expTagFound:                      true,
			expNetFound:                      true,
			expGetSecurityGroupIdsFromNetIds: nil,
			expReadTagErr:                    fmt.Errorf("ReadTag generic error"),
			expReconcileSecurityGroupErr:     fmt.Errorf("ReadTag generic error cannot get tag for OscCluster test-system/test-osc"),
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
						ReadTag(gomock.Eq("Name"), gomock.Eq(securityGroupName)).
						Return(nil, sgtc.expReadTagErr)
				}

				mockOscSecurityGroupInterface.
					EXPECT().
					GetSecurityGroupIdsFromNetIds(gomock.Eq(netId)).
					Return(securityGroupIds, sgtc.expGetSecurityGroupIdsFromNetIds)

			}

			reconcileSecurityGroup, err := reconcileSecurityGroup(ctx, clusterScope, mockOscSecurityGroupInterface, mockOscTagInterface)
			if err != nil {
				assert.Equal(t, sgtc.expReconcileSecurityGroupErr.Error(), err.Error(), "reconcileSecurityGroup() should return the same error")
			} else {
				assert.Nil(t, sgtc.expReconcileSecurityGroupErr)
			}

			t.Logf("find reconcileSecurityGroup %v\n", reconcileSecurityGroup)
		})
	}
}

func TestDeleteSecurityGroup(t *testing.T) {
	securityGroupTestCases := []struct {
		name                            string
		spec                            infrastructurev1beta1.OscClusterSpec
		expSecurityGroupFound           bool
		expLoadBalancerResourceConflict bool
		expDeleteSecurityGroupErr       error
		expGetSecurityGroupErr          error
		expExpectedError                string
		expRequeueAfter                 time.Duration
	}{
		{
			name:                            "delete securityGroup successfully",
			spec:                            defaultSecurityGroupReconcile,
			expSecurityGroupFound:           true,
			expLoadBalancerResourceConflict: false,
			expDeleteSecurityGroupErr:       nil,
			expGetSecurityGroupErr:          nil,
			expExpectedError:                "",
			expRequeueAfter:                 0,
		},
		{
			name:                            "delete securityGroup with uncatch error",
			spec:                            defaultSecurityGroupReconcile,
			expSecurityGroupFound:           true,
			expLoadBalancerResourceConflict: false,
			expDeleteSecurityGroupErr:       fmt.Errorf("cannot delete securityGroup"),
			expGetSecurityGroupErr:          nil,
			expExpectedError:                "cannot delete securityGroup for Osccluster test-system/test-osc",
			expRequeueAfter:                 30 * time.Second,
		},
		{
			name:                            "waiting load balancer to timeout",
			spec:                            defaultSecurityGroupReconcile,
			expSecurityGroupFound:           true,
			expLoadBalancerResourceConflict: true,
			expDeleteSecurityGroupErr:       fmt.Errorf("DeleteSecurityGroup error"),
			expGetSecurityGroupErr:          nil,
			// Expect only the main error message, without the detailed load balancer conflict message
			expExpectedError: "cannot delete securityGroup for Osccluster test-system/test-osc",
			expRequeueAfter:  30 * time.Second,
		},
	}

	for _, tc := range securityGroupTestCases {
		t.Run(tc.name, func(t *testing.T) {
			clusterScope, ctx, mockSecurityGroupInterface, _ := SetupWithSecurityGroupMock(t, tc.name, tc.spec)

			// Setup SecurityGroup reference
			securityGroupID := "sg-test-securitygroup-uid"
			clusterScope.GetSecurityGroupsRef().ResourceMap = map[string]string{
				"test-securitygroup-uid": securityGroupID,
			}

			// Mock GetSecurityGroup to simulate existence check
			if tc.expSecurityGroupFound {
				mockSecurityGroupInterface.EXPECT().
					GetSecurityGroup(gomock.Eq(securityGroupID)).
					Return(&osc.SecurityGroup{SecurityGroupId: &securityGroupID}, tc.expGetSecurityGroupErr).
					AnyTimes()
			} else {
				mockSecurityGroupInterface.EXPECT().
					GetSecurityGroup(gomock.Eq(securityGroupID)).
					Return(nil, tc.expGetSecurityGroupErr).
					AnyTimes()
			}

			// Mock DeleteSecurityGroup with specified error if any
			mockSecurityGroupInterface.EXPECT().
				DeleteSecurityGroup(gomock.Eq(securityGroupID)).
				Return(tc.expDeleteSecurityGroupErr)

			// Call deleteSecurityGroup and capture the result
			result, err := deleteSecurityGroup(ctx, clusterScope, securityGroupID, mockSecurityGroupInterface)

			// Assertions based on the expected outcome
			if tc.expExpectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expExpectedError)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tc.expRequeueAfter, result.RequeueAfter)

			t.Logf("Test %s result: %v, error: %v", tc.name, result, err)
		})
	}
}

// TestReconcileDeleteSecurityGroupDelete tests the delete functionality of security groups.
func TestReconcileDeleteSecurityGroupDelete(t *testing.T) {
	securityGroupTestCases := []struct {
		name                       string
		spec                       infrastructurev1beta1.OscClusterSpec
		expDeleteSecurityGroup     bool
		expDeleteSecurityGroupRule bool
		expGetSecurityGroupErr     error
		expDeleteSecurityGroupErr  error
		expectedError              string
		expectedRequeueAfter       time.Duration
	}{
		{
			name:                       "successfully_delete_all_security_groups",
			spec:                       defaultSecurityGroupReconcile,
			expDeleteSecurityGroup:     true,
			expDeleteSecurityGroupRule: true,
			expGetSecurityGroupErr:     nil,
			expDeleteSecurityGroupErr:  nil,
			expectedError:              "",
			expectedRequeueAfter:       0,
		},
		{
			name:                       "security_group_rule_deletion_error",
			spec:                       defaultSecurityGroupReconcile,
			expDeleteSecurityGroup:     true,
			expDeleteSecurityGroupRule: true,
			expGetSecurityGroupErr:     nil,
			expDeleteSecurityGroupErr:  fmt.Errorf("DeleteSecurityGroupRule error"),
			expectedError:              "cannot delete securityGroup for Osccluster test-system/test-osc",
			expectedRequeueAfter:       30 * time.Second,
		},
		// Additional cases can be added as needed
	}

	for _, tc := range securityGroupTestCases {
		t.Run(tc.name, func(t *testing.T) {
			clusterScope, ctx, mockSecurityGroupInterface, _ := SetupWithSecurityGroupMock(t, tc.name, tc.spec)
			securityGroupID := "sg-test-securitygroup-uid"

			// Mock GetSecurityGroup calls
			if tc.expDeleteSecurityGroupRule {
				mockSecurityGroupInterface.EXPECT().
					GetSecurityGroup(gomock.Eq(securityGroupID)).
					Return(&osc.SecurityGroup{SecurityGroupId: &securityGroupID}, tc.expGetSecurityGroupErr).
					AnyTimes()
			}

			// Mock DeleteSecurityGroupRule calls
			if tc.expDeleteSecurityGroupRule {
				mockSecurityGroupInterface.EXPECT().
					DeleteSecurityGroupRule(
						gomock.Eq(securityGroupID),
						gomock.Eq("Inbound"),
						gomock.Eq("tcp"),
						gomock.Eq("0.0.0.0/0"),
						gomock.Any(),
						gomock.Eq(int32(6443)),
						gomock.Eq(int32(6443)),
					).Return(tc.expDeleteSecurityGroupErr).AnyTimes()
			}

			// Mock DeleteSecurityGroup calls
			if tc.expDeleteSecurityGroup {
				mockSecurityGroupInterface.EXPECT().
					DeleteSecurityGroup(gomock.Eq(securityGroupID)).
					Return(tc.expDeleteSecurityGroupErr)
			}

			// Execute deleteSecurityGroup
			result, err := deleteSecurityGroup(ctx, clusterScope, securityGroupID, mockSecurityGroupInterface)

			// Validate the expected result
			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			} else {
				assert.NoError(t, err)
			}

			// Check RequeueAfter matches the expected duration
			assert.Equal(t, tc.expectedRequeueAfter, result.RequeueAfter, "Unexpected RequeueAfter duration")
		})
	}
}

// TestReconcileDeleteSecurityGroupDeleteWithoutSpec has several tests to cover the code of the function reconcileDeleteSecurityGroups
func TestReconcileDeleteSecurityGroupDeleteWithoutSpec(t *testing.T) {
	securityGroupTestCases := []struct {
		name                      string
		spec                      infrastructurev1beta1.OscClusterSpec
		expDeleteSecurityGroupErr error
		expectedError             error
	}{
		{
			name:                      "delete securityGroup without existing spec (default values)",
			spec:                      defaultSecurityGroupInitialize, // Ensuring default values
			expDeleteSecurityGroupErr: nil,
			expectedError:             nil,
		},
		{
			name:                      "failed to delete securityGroup",
			spec:                      defaultSecurityGroupInitialize,
			expDeleteSecurityGroupErr: fmt.Errorf("delete securityGroup error"),
			expectedError:             fmt.Errorf("cannot delete securityGroup for Osccluster test-system/test-osc"),
		},
	}

	for _, tc := range securityGroupTestCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up mocks and context
			clusterScope, ctx, mockOscSecurityGroupInterface, _ := SetupWithSecurityGroupMock(t, tc.name, tc.spec)
			securityGroupName := tc.spec.Network.SecurityGroups[0].Name + "-uid"
			securityGroupId := "sg-" + securityGroupName
			clusterScope.GetSecurityGroupsRef().ResourceMap = map[string]string{securityGroupName: securityGroupId}

			// Mock the `DeleteSecurityGroup` call based on test case expectation
			mockOscSecurityGroupInterface.EXPECT().
				DeleteSecurityGroup(gomock.Eq(securityGroupId)).
				Return(tc.expDeleteSecurityGroupErr)

			// Call deleteSecurityGroup and validate result
			result, err := deleteSecurityGroup(ctx, clusterScope, securityGroupId, mockOscSecurityGroupInterface)

			// Validate expected error or success
			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, reconcile.Result{}, result)
			}

			t.Logf("Test %s result: %v, error: %v", tc.name, result, err)
		})
	}
}

// TestReconcileDeleteSecurityGroupGet has several tests to cover the code of the function reconcileDeleteSecurityGroups
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
			expGetSecurityGroupFromNetIdsErr:   fmt.Errorf("GetSecurityGroup generic error"),
			expReconcileDeleteSecurityGroupErr: fmt.Errorf("GetSecurityGroup generic error"),
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
							GetSecurityGroupIdsFromNetIds(gomock.Eq(netId)).
							Return(securityGroupIds, sgtc.expGetSecurityGroupFromNetIdsErr)
					} else {
						mockOscSecurityGroupInterface.
							EXPECT().
							GetSecurityGroupIdsFromNetIds(gomock.Eq(netId)).
							Return(nil, sgtc.expGetSecurityGroupFromNetIdsErr)
					}
				}
			}
			reconcileDeleteSecurityGroups, err := reconcileDeleteSecurityGroups(ctx, clusterScope, mockOscSecurityGroupInterface)
			if err != nil {
				assert.Equal(t, sgtc.expReconcileDeleteSecurityGroupErr.Error(), err.Error(), "reconcileDeleteSecurityGroups() should return the same error")
			} else {
				assert.Nil(t, sgtc.expReconcileDeleteSecurityGroupErr)
			}
			t.Logf("find reconcileDeleteSecurityGroups %v\n", reconcileDeleteSecurityGroups)
		})
	}
}

// TestReconcileDeleteSecurityGroupResourceId has several tests to cover the code of the function reconcileDeleteSecurityGroups
func TestReconcileDeleteSecurityGroupResourceId(t *testing.T) {
	securityGroupTestCases := []struct {
		name                               string
		spec                               infrastructurev1beta1.OscClusterSpec
		expNetFound                        bool
		expReconcileDeleteSecurityGroupErr error
	}{
		{
			name:                               "net does not exist",
			spec:                               defaultSecurityGroupReconcile,
			expNetFound:                        false,
			expReconcileDeleteSecurityGroupErr: fmt.Errorf("test-net-uid does not exist"),
		},
		{
			name: "check failed without net and securityGroup spec (retrieve default values cluster-api)",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{},
			},
			expNetFound:                        false,
			expReconcileDeleteSecurityGroupErr: fmt.Errorf("cluster-api-net-uid does not exist"),
		},
	}
	for _, sgtc := range securityGroupTestCases {
		t.Run(sgtc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscSecurityGroupInterface, _ := SetupWithSecurityGroupMock(t, sgtc.name, sgtc.spec)
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			reconcileDeleteSecurityGroups, err := reconcileDeleteSecurityGroups(ctx, clusterScope, mockOscSecurityGroupInterface)
			if err != nil {
				assert.Equal(t, sgtc.expReconcileDeleteSecurityGroupErr.Error(), err.Error(), "reconcileDeleteSecurityGroups() should return the same error")
			} else {
				assert.Nil(t, sgtc.expReconcileDeleteSecurityGroupErr)
			}
			t.Logf("find reconcileDeleteSecurityGroups %v\n", reconcileDeleteSecurityGroups)
		})
	}
}
