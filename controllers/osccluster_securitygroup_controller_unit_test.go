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
	"io/ioutil"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/benbjohnson/clock"
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
			expGetSecurityGroupResourceIdErr: fmt.Errorf("test-securitygroup-uid does not exist"),
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

// TestReconcileSecurityGroupRuleCreate has several tests to cover the code of the function reconcileSecurityGroupRule

func TestReconcileSecurityGroupRuleCreate(t *testing.T) {
	securityGroupRuleTestCases := []struct {
		name                                        string
		spec                                        infrastructurev1beta1.OscClusterSpec
		expCreateSecurityGroupRuleFound             bool
		expTagFound                                 bool
		expGetSecurityGroupFromSecurityGroupRuleErr error
		expCreateSecurityGroupRuleErr               error
		expReadTagErr                               error
		expReconcileSecurityGroupRuleErr            error
	}{
		{
			name:                            "create securityGroupRule  (first time reconcileloop)",
			spec:                            defaultSecurityGroupInitialize,
			expCreateSecurityGroupRuleFound: true,
			expTagFound:                     false,
			expGetSecurityGroupFromSecurityGroupRuleErr: nil,
			expCreateSecurityGroupRuleErr:               nil,
			expReadTagErr:                               nil,
			expReconcileSecurityGroupRuleErr:            nil,
		},
		{
			name:                            "failed to create securityGroupRule",
			spec:                            defaultSecurityGroupInitialize,
			expCreateSecurityGroupRuleFound: false,
			expTagFound:                     false,
			expGetSecurityGroupFromSecurityGroupRuleErr: nil,
			expCreateSecurityGroupRuleErr:               fmt.Errorf("CreateSecurityGroupRule generic errors"),
			expReadTagErr:                               nil,
			expReconcileSecurityGroupRuleErr:            fmt.Errorf("CreateSecurityGroupRule generic errors Can not create securityGroupRule for OscCluster test-system/test-osc"),
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
						ReadTag(gomock.Eq("Name"), gomock.Eq(securityGroupName)).
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
						GetSecurityGroupFromSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIpProtocol), gomock.Eq(securityGroupRuleIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
						Return(nil, sgrtc.expGetSecurityGroupFromSecurityGroupRuleErr)

					if sgrtc.expCreateSecurityGroupRuleFound {
						mockOscSecurityGroupInterface.
							EXPECT().
							CreateSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIpProtocol), gomock.Eq(securityGroupRuleIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
							Return(securityGroupRule.SecurityGroup, sgrtc.expCreateSecurityGroupRuleErr)
					} else {
						mockOscSecurityGroupInterface.
							EXPECT().
							CreateSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIpProtocol), gomock.Eq(securityGroupRuleIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
							Return(nil, sgrtc.expCreateSecurityGroupRuleErr)
					}
					// TO FIX
					// reconcileSecurityGroupRule, err := reconcileSecurityGroupRule(ctx, clusterScope, securityGroupRuleSpec, securityGroupName, mockOscSecurityGroupInterface)
					// if err != nil {
					// 	assert.Equal(t, sgrtc.expReconcileSecurityGroupRuleErr.Error(), err.Error(), "reconcileSecurityGroupRules() should return the same error")
					// } else {
					// 	assert.Nil(t, sgrtc.expReconcileSecurityGroupRuleErr)
					// }
					// t.Logf("find reconcileSecurityGroupRule %v\n", reconcileSecurityGroupRule)
				}
			}
		})
	}
}

// TestReconcileSecurityGroupRuleGet has several tests to cover the code of the function reconcileSecurityGroupRule
func TestReconcileSecurityGroupRuleGet(t *testing.T) {
	securityGroupRuleTestCases := []struct {
		name                                        string
		spec                                        infrastructurev1beta1.OscClusterSpec
		expSecurityGroupRuleFound                   bool
		expTagFound                                 bool
		expGetSecurityGroupFromSecurityGroupRuleErr error
		expReadTagErr                               error
		expReconcileSecurityGroupRuleErr            error
	}{
		{
			name:                      "get securityGroupRule ((second time reconcile loop)",
			spec:                      defaultSecurityGroupReconcile,
			expSecurityGroupRuleFound: true,
			expTagFound:               false,
			expGetSecurityGroupFromSecurityGroupRuleErr: nil,
			expReadTagErr:                    nil,
			expReconcileSecurityGroupRuleErr: nil,
		},
		{
			name:                      "failed to get securityGroup",
			spec:                      defaultSecurityGroupReconcile,
			expSecurityGroupRuleFound: true,
			expTagFound:               false,
			expReadTagErr:             nil,
			expGetSecurityGroupFromSecurityGroupRuleErr: fmt.Errorf("GetSecurityGroupFromSecurityGroupRule generic errors"),
			expReconcileSecurityGroupRuleErr:            fmt.Errorf("GetSecurityGroupFromSecurityGroupRule generic errors"),
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
						ReadTag(gomock.Eq("Name"), gomock.Eq(securityGroupName)).
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

					readSecurityGroups := osc.ReadSecurityGroupsResponse{
						SecurityGroups: &[]osc.SecurityGroup{
							*securityGroupRule.SecurityGroup,
						},
					}
					readSecurityGroup := *readSecurityGroups.SecurityGroups
					if sgrtc.expSecurityGroupRuleFound {
						mockOscSecurityGroupInterface.
							EXPECT().
							GetSecurityGroupFromSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIpProtocol), gomock.Eq(securityGroupRuleIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
							Return(&readSecurityGroup[0], sgrtc.expGetSecurityGroupFromSecurityGroupRuleErr)
					}
					// TO FIX
					// reconcileSecurityGroupRule, err := reconcileSecurityGroupRule(ctx, clusterScope, securityGroupRuleSpec, securityGroupName, mockOscSecurityGroupInterface)
					// if err != nil {
					// 	assert.Equal(t, sgrtc.expReconcileSecurityGroupRuleErr, err, "reconcileSecurityGroupRules() should return the same error")
					// } else {
					// 	assert.Nil(t, sgrtc.expReconcileSecurityGroupRuleErr)
					// }
					// t.Logf("find reconcileSecurityGroupRule %v\n", reconcileSecurityGroupRule)
				}
			}
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
			expReconcileDeleteSecurityGroupRuleErr:      fmt.Errorf("DeleteSecurityGroupRule generic error Can not delete securityGroupRule for OscCluster test-system/test-osc"),
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
			expCreateSecurityGroupRuleErr:    fmt.Errorf("CreateSecurityGroupRule generic errors"),
			expReadTagErr:                    nil,
			expReconcileSecurityGroupErr:     fmt.Errorf("CreateSecurityGroupRule generic errors Can not create securityGroupRule for OscCluster test-system/test-osc"),
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
						ReadTag(gomock.Eq("Name"), gomock.Eq(securityGroupName)).
						Return(&tag, sgtc.expReadTagErr)
				} else {
					mockOscTagInterface.
						EXPECT().
						ReadTag(gomock.Eq("Name"), gomock.Eq(securityGroupName)).
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
					GetSecurityGroupIdsFromNetIds(gomock.Eq(netId)).
					Return(nil, sgtc.expGetSecurityGroupFromNetIdsErr)
				mockOscSecurityGroupInterface.
					EXPECT().
					CreateSecurityGroup(gomock.Eq(netId), gomock.Eq(clusterName), gomock.Eq(securityGroupName), gomock.Eq(securityGroupDescription), gomock.Eq(securityGroupTag)).
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
								GetSecurityGroupFromSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIpProtocol), gomock.Eq(securityGroupRuleIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
								Return(&readSecurityGroup[0], sgtc.expGetSecurityGroupRuleErr)
						} else {
							mockOscSecurityGroupInterface.
								EXPECT().
								GetSecurityGroupFromSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIpProtocol), gomock.Eq(securityGroupRuleIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
								Return(nil, sgtc.expGetSecurityGroupRuleErr)
							if sgtc.expCreateSecurityGroupRuleFound {
								mockOscSecurityGroupInterface.
									EXPECT().
									CreateSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIpProtocol), gomock.Eq(securityGroupRuleIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
									Return(securityGroupRule.SecurityGroup, sgtc.expCreateSecurityGroupRuleErr)
							} else {
								mockOscSecurityGroupInterface.
									EXPECT().
									CreateSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIpProtocol), gomock.Eq(securityGroupRuleIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
									Return(nil, sgtc.expCreateSecurityGroupRuleErr)
							}
						}
					}
					reconcileSecurityGroup, err := reconcileSecurityGroup(ctx, clusterScope, mockOscSecurityGroupInterface, mockOscTagInterface)
					if err != nil {
						assert.Equal(t, sgtc.expReconcileSecurityGroupErr.Error(), err.Error(), "reconcileSecurityGroup() should return the same error")
					} else {
						assert.Nil(t, sgtc.expReconcileSecurityGroupErr)
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
			expReconcileSecurityGroupErr:     fmt.Errorf("CreateSecurityGroup generic error Can not create securityGroup for Osccluster test-system/test-osc"),
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
			expReconcileSecurityGroupErr:     fmt.Errorf("ReadTag generic error Can not get tag for OscCluster test-system/test-osc"),
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

// TestDeleteSecurityGroup has several tests to cover the code of the function deleteSecurityGroup
func TestDeleteSecurityGroup(t *testing.T) {
	securityGroupTestCases := []struct {
		name                                      string
		spec                                      infrastructurev1beta1.OscClusterSpec
		expSecurityGroupFound                     bool
		expLoadBalancerResourceConflict           bool
		expInvalidDeleteSecurityGroupJsonResponse bool
		expDeleteSecurityGroupFirstMockErr        error
		expDeleteSecurityGroupError               error
	}{
		{
			name:                            "delete securityGroup",
			spec:                            defaultSecurityGroupReconcile,
			expSecurityGroupFound:           true,
			expLoadBalancerResourceConflict: false,
			expInvalidDeleteSecurityGroupJsonResponse: false,
			expDeleteSecurityGroupFirstMockErr:        nil,
			expDeleteSecurityGroupError:               nil,
		},
		{
			name:                            "delete securityGroup unmatch to catch",
			spec:                            defaultSecurityGroupReconcile,
			expSecurityGroupFound:           true,
			expLoadBalancerResourceConflict: false,
			expInvalidDeleteSecurityGroupJsonResponse: false,
			expDeleteSecurityGroupFirstMockErr:        fmt.Errorf("DeleteSecurityGroup first generic error"),
			expDeleteSecurityGroupError:               fmt.Errorf(" Can not delete securityGroup because of the uncatch error for Osccluster test-system/test-osc"),
		},
		{
			name:                            "invalid json response",
			spec:                            defaultSecurityGroupReconcile,
			expSecurityGroupFound:           true,
			expLoadBalancerResourceConflict: false,
			expInvalidDeleteSecurityGroupJsonResponse: true,
			expDeleteSecurityGroupFirstMockErr:        fmt.Errorf("DeleteSecurityGroup first generic error"),
			expDeleteSecurityGroupError:               fmt.Errorf("invalid character 'B' looking for beginning of value Can not delete securityGroup for Osccluster test-system/test-osc"),
		},
		{
			name:                            "waiting loadbalancer to timeout",
			spec:                            defaultSecurityGroupReconcile,
			expSecurityGroupFound:           true,
			expLoadBalancerResourceConflict: true,
			expInvalidDeleteSecurityGroupJsonResponse: false,
			expDeleteSecurityGroupFirstMockErr:        fmt.Errorf("DeleteSecurityGroup first generic error"),
			expDeleteSecurityGroupError:               fmt.Errorf("DeleteSecurityGroup first generic error Can not delete securityGroup because to waiting loadbalancer to be delete timeout  for Osccluster test-system/test-osc"),
		},
	}
	for _, sgtc := range securityGroupTestCases {
		t.Run(sgtc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscSecurityGroupInterface, _ := SetupWithSecurityGroupMock(t, sgtc.name, sgtc.spec)
			var err error
			var deleteSg reconcile.Result
			var wg sync.WaitGroup
			securityGroupsSpec := sgtc.spec.Network.SecurityGroups
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			clock_mock := clock.NewMock()
			clock_mock.Now().UTC()
			for _, securityGroupSpec := range securityGroupsSpec {
				securityGroupName := securityGroupSpec.Name + "-uid"
				securityGroupId := "sg-" + securityGroupName
				securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
				var httpResponse *http.Response
				if sgtc.expDeleteSecurityGroupFirstMockErr != nil {
					if sgtc.expLoadBalancerResourceConflict {
						httpResponse = &http.Response{
							StatusCode: 9085,
							Body: ioutil.NopCloser(strings.NewReader(`{
							"Errors": [
                                                	        {
                                                	       	    "Type": "ResourceConflict",
        	                                                    "Details": "",
	                                                            "Code": "9085"
                                 	                       }
                                        	        ],
                                               	 	"ResponseContext": {
                                                        	"RequestId": "aaaaa-bbbbb-ccccc"
                                                	}
						}`)),
						}
					} else {
						httpResponse = &http.Response{
							StatusCode: 10014,
							Body: ioutil.NopCloser(strings.NewReader(`{
                                                        "Errors": [
                                                                {
                                                                    "Type": "TooManyResources (QuotaExceded)",
                                                                    "Details": "",
                                                                    "Code": "10014"
                                                               }
                                                        ],
                                                        "ResponseContext": {
                                                                "RequestId": "aaaaa-bbbbb-ccccc"
                                                        }
                                                }`)),
						}
					}
					if sgtc.expInvalidDeleteSecurityGroupJsonResponse {
						httpResponse = &http.Response{
							StatusCode: 0,
							Body: ioutil.NopCloser(strings.NewReader(`{
                                                        "Errors": [
                                                                {
                                                                    "Type":Bad,
                                                               }
                                                        ],
                                                        "ResponseContext": {
                                                                "RequestId": "aaaaa-bbbbb-ccccc"
                                                        }
                                                }`)),
						}

					}
					mockOscSecurityGroupInterface.
						EXPECT().
						DeleteSecurityGroup(gomock.Eq(securityGroupId)).
						Return(sgtc.expDeleteSecurityGroupFirstMockErr, httpResponse)

				} else {
					httpResponse = &http.Response{
						StatusCode: 200,
						Body: ioutil.NopCloser(strings.NewReader(`{
                                                        "ResponseContext": {
                                                                "RequestId": "aaaaa-bbbbb-ccccc"
                                                        }
                                                }`)),
					}
					mockOscSecurityGroupInterface.
						EXPECT().
						DeleteSecurityGroup(gomock.Eq(securityGroupId)).
						Return(nil, httpResponse)
				}

				wg.Add(1)
				go func() {
					deleteSg, err = deleteSecurityGroup(ctx, clusterScope, securityGroupId, mockOscSecurityGroupInterface)
					wg.Done()
				}()
				runtime.Gosched()
				wg.Wait()
				if err != nil {
					assert.Equal(t, sgtc.expDeleteSecurityGroupError.Error(), err.Error(), "deleteSecurityGroup() should return the same error")
				} else {
					assert.Nil(t, sgtc.expDeleteSecurityGroupError)
				}
				t.Logf("Find  deleteSecurityGroup %v\n", deleteSg)
			}
		})
	}
}

// TestReconcileDeleteSecurityGroup has several tests to cover the code of the function reconcileDeleteSecurityGroups
func TestReconcileDeleteSecurityGroup(t *testing.T) {
	securityGroupTestCases := []struct {
		name                                        string
		spec                                        infrastructurev1beta1.OscClusterSpec
		expNetFound                                 bool
		expSecurityGroupFound                       bool
		expSecurityGroupRuleFound                   bool
		expDeleteSecurityGroupFound                 bool
		expDeleteSecurityGroupRuleFound             bool
		expGetSecurityGroupfromSecurityGroupRuleErr error
		expGetSecurityGroupFromNetIdsErr            error
		expDeleteSecurityGroupRuleErr               error
		expReconcileDeleteSecurityGroupErr          error
	}{
		{
			name:                             "failed to delete securityGroupRule",
			spec:                             defaultSecurityGroupReconcile,
			expNetFound:                      true,
			expSecurityGroupFound:            true,
			expSecurityGroupRuleFound:        true,
			expDeleteSecurityGroupFound:      false,
			expDeleteSecurityGroupRuleFound:  false,
			expGetSecurityGroupFromNetIdsErr: nil,
			expGetSecurityGroupfromSecurityGroupRuleErr: nil,
			expDeleteSecurityGroupRuleErr:               fmt.Errorf("DeleteSecurityGroupRule generic error"),
			expReconcileDeleteSecurityGroupErr:          fmt.Errorf("DeleteSecurityGroupRule generic error Can not delete securityGroupRule for OscCluster test-system/test-osc"),
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

					if sgtc.expSecurityGroupFound {
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
									GetSecurityGroupFromSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIpProtocol), gomock.Eq(securityGroupRuleIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
									Return(&readSecurityGroup[0], sgtc.expGetSecurityGroupfromSecurityGroupRuleErr)
							} else {
								mockOscSecurityGroupInterface.
									EXPECT().
									GetSecurityGroupFromSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIpProtocol), gomock.Eq(securityGroupRuleIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
									Return(nil, sgtc.expGetSecurityGroupfromSecurityGroupRuleErr)
							}
							mockOscSecurityGroupInterface.
								EXPECT().
								DeleteSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIpProtocol), gomock.Eq(securityGroupRuleIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
								Return(sgtc.expDeleteSecurityGroupRuleErr)
						}
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

// TestReconcileDeleteSecurityGroupDelete has several tests to cover the code of the function reconcileDeleteSecurityGroups
func TestReconcileDeleteSecurityGroupDelete(t *testing.T) {
	securityGroupTestCases := []struct {
		name                                        string
		spec                                        infrastructurev1beta1.OscClusterSpec
		expSecurityGroupFound                       bool
		expGetSecurityGroupFromNetIdsErr            error
		expGetSecurityGroupfromSecurityGroupRuleErr error
		expDeleteSecurityGroupRuleErr               error
		expDeleteSecurityGroupErr                   error
		expReconcileDeleteSecurityGroupErr          error
	}{
		{
			name:                             "delete securityGroup",
			spec:                             defaultSecurityGroupReconcile,
			expSecurityGroupFound:            true,
			expGetSecurityGroupFromNetIdsErr: nil,
			expGetSecurityGroupfromSecurityGroupRuleErr: nil,
			expDeleteSecurityGroupRuleErr:               nil,
			expDeleteSecurityGroupErr:                   nil,
			expReconcileDeleteSecurityGroupErr:          nil,
		},
		{
			name:                             "failed to delete securityGroup",
			spec:                             defaultSecurityGroupReconcile,
			expSecurityGroupFound:            true,
			expGetSecurityGroupFromNetIdsErr: nil,
			expGetSecurityGroupfromSecurityGroupRuleErr: nil,
			expDeleteSecurityGroupRuleErr:               nil,
			expDeleteSecurityGroupErr:                   fmt.Errorf(" Can not delete securityGroup because of the uncatch error for Osccluster test-system/test-osc"),
			expReconcileDeleteSecurityGroupErr:          fmt.Errorf(" Can not delete securityGroup because of the uncatch error for Osccluster test-system/test-osc Can not delete securityGroup  for Osccluster test-system/test-osc"),
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
				var httpResponse *http.Response
				mockOscSecurityGroupInterface.
					EXPECT().
					GetSecurityGroupIdsFromNetIds(gomock.Eq(netId)).
					Return(securityGroupIds, sgtc.expGetSecurityGroupFromNetIdsErr)
				if sgtc.expDeleteSecurityGroupErr == nil {
					httpResponse = &http.Response{
						StatusCode: 200,
						Body: ioutil.NopCloser(strings.NewReader(`{
	                                                "ResponseContext": {
        	                                                "RequestId": "aaaaa-bbbbb-ccccc"
                	                                }
                        	                }`)),
					}

				} else {
					httpResponse = &http.Response{
						StatusCode: 10014,
						Body: ioutil.NopCloser(strings.NewReader(`{
                                                        "Errors": [
                                                                {
                                                                    "Type": "TooManyResources (QuotaExceded)",
                                                                    "Details": "",
                                                                    "Code": "10014"
                                                               }
                                                        ],
                                                        "ResponseContext": {
                                                                "RequestId": "aaaaa-bbbbb-ccccc"
                                                        }
                                                }`)),
					}

				}
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
						Return(&readSecurityGroup[0], sgtc.expGetSecurityGroupfromSecurityGroupRuleErr)
					mockOscSecurityGroupInterface.
						EXPECT().
						DeleteSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIpProtocol), gomock.Eq(securityGroupRuleIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
						Return(sgtc.expDeleteSecurityGroupRuleErr)

				}
				mockOscSecurityGroupInterface.
					EXPECT().
					DeleteSecurityGroup(gomock.Eq(securityGroupId)).
					Return(sgtc.expDeleteSecurityGroupErr, httpResponse)
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

// TestReconcileDeleteSecurityGroupDeleteWithoutSpec has several tests to cover the code of the function reconcileDeleteSecurityGroups
func TestReconcileDeleteSecurityGroupDeleteWithoutSpec(t *testing.T) {
	securityGroupTestCases := []struct {
		name                                        string
		spec                                        infrastructurev1beta1.OscClusterSpec
		expGetSecurityGroupfromSecurityGroupRuleErr error
		expGetSecurityGroupFromNetIdsErr            error
		expDeleteSecurityGroupRuleErr               error
		expDeleteSecurityGroupErr                   error
		expReconcileDeleteSecurityGroupErr          error
	}{
		{
			name: "delete securityGroup without spec (with default values)",
			expGetSecurityGroupfromSecurityGroupRuleErr: nil,
			expGetSecurityGroupFromNetIdsErr:            nil,
			expDeleteSecurityGroupRuleErr:               nil,
			expDeleteSecurityGroupErr:                   nil,
			expReconcileDeleteSecurityGroupErr:          nil,
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
			var httpResponse *http.Response
			httpResponse = &http.Response{
				StatusCode: 200,
				Body: ioutil.NopCloser(strings.NewReader(`{
	                                                "ResponseContext": {
        	                                                "RequestId": "aaaaa-bbbbb-ccccc"
                	                                }
                        	                }`)),
			}
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
				GetSecurityGroupIdsFromNetIds(gomock.Eq(netId)).
				Return(securityGroupIds, sgtc.expGetSecurityGroupFromNetIdsErr)

			mockOscSecurityGroupInterface.
				EXPECT().
				GetSecurityGroupFromSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleKubeletKwFlow), gomock.Eq(securityGroupRuleKubeletKwIpProtocol), gomock.Eq(securityGroupRuleKubeletKwIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleKubeletKwFromPortRange), gomock.Eq(securityGroupRuleKubeletKwToPortRange)).
				Return(&readSecurityGroup[0], sgtc.expGetSecurityGroupfromSecurityGroupRuleErr)
			mockOscSecurityGroupInterface.
				EXPECT().
				DeleteSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleKubeletKwFlow), gomock.Eq(securityGroupRuleKubeletKwIpProtocol), gomock.Eq(securityGroupRuleKubeletKwIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleKubeletKwFromPortRange), gomock.Eq(securityGroupRuleKubeletKwToPortRange)).
				Return(sgtc.expDeleteSecurityGroupRuleErr)

			mockOscSecurityGroupInterface.
				EXPECT().
				GetSecurityGroupFromSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleKubeletKcpFlow), gomock.Eq(securityGroupRuleKubeletKcpIpProtocol), gomock.Eq(securityGroupRuleKubeletKcpIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleKubeletKcpFromPortRange), gomock.Eq(securityGroupRuleKubeletKcpToPortRange)).
				Return(&readSecurityGroup[0], sgtc.expGetSecurityGroupfromSecurityGroupRuleErr)

			mockOscSecurityGroupInterface.
				EXPECT().
				DeleteSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleKubeletKcpFlow), gomock.Eq(securityGroupRuleKubeletKcpIpProtocol), gomock.Eq(securityGroupRuleKubeletKcpIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleKubeletKcpFromPortRange), gomock.Eq(securityGroupRuleKubeletKcpToPortRange)).
				Return(sgtc.expDeleteSecurityGroupRuleErr)

			mockOscSecurityGroupInterface.
				EXPECT().
				GetSecurityGroupFromSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleNodeIpKwFlow), gomock.Eq(securityGroupRuleNodeIpKwIpProtocol), gomock.Eq(securityGroupRuleNodeIpKwIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleNodeIpKwFromPortRange), gomock.Eq(securityGroupRuleNodeIpKwToPortRange)).
				Return(&readSecurityGroup[0], sgtc.expGetSecurityGroupfromSecurityGroupRuleErr)
			mockOscSecurityGroupInterface.
				EXPECT().
				DeleteSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleNodeIpKwFlow), gomock.Eq(securityGroupRuleNodeIpKwIpProtocol), gomock.Eq(securityGroupRuleNodeIpKwIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleNodeIpKwFromPortRange), gomock.Eq(securityGroupRuleNodeIpKwToPortRange)).
				Return(sgtc.expDeleteSecurityGroupRuleErr)
			mockOscSecurityGroupInterface.
				EXPECT().
				GetSecurityGroupFromSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleNodeIpKcpFlow), gomock.Eq(securityGroupRuleNodeIpKcpIpProtocol), gomock.Eq(securityGroupRuleNodeIpKcpIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleNodeIpKcpFromPortRange), gomock.Eq(securityGroupRuleNodeIpKcpToPortRange)).
				Return(&readSecurityGroup[0], sgtc.expGetSecurityGroupfromSecurityGroupRuleErr)
			mockOscSecurityGroupInterface.
				EXPECT().
				DeleteSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleNodeIpKcpFlow), gomock.Eq(securityGroupRuleNodeIpKcpIpProtocol), gomock.Eq(securityGroupRuleNodeIpKcpIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleNodeIpKcpFromPortRange), gomock.Eq(securityGroupRuleNodeIpKcpToPortRange)).
				Return(sgtc.expDeleteSecurityGroupRuleErr)

			mockOscSecurityGroupInterface.
				EXPECT().
				GetSecurityGroupFromSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleKcpBgpFlow), gomock.Eq(securityGroupRuleKcpBgpIpProtocol), gomock.Eq(securityGroupRuleKcpBgpIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleKcpBgpFromPortRange), gomock.Eq(securityGroupRuleKcpBgpToPortRange)).
				Return(&readSecurityGroup[0], sgtc.expGetSecurityGroupfromSecurityGroupRuleErr)
			mockOscSecurityGroupInterface.
				EXPECT().
				DeleteSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleKcpBgpFlow), gomock.Eq(securityGroupRuleKcpBgpIpProtocol), gomock.Eq(securityGroupRuleKcpBgpIpRange), gomock.Eq(securityGroupMemberId), gomock.Eq(securityGroupRuleKcpBgpFromPortRange), gomock.Eq(securityGroupRuleKcpBgpToPortRange)).
				Return(sgtc.expDeleteSecurityGroupRuleErr)

			mockOscSecurityGroupInterface.
				EXPECT().
				DeleteSecurityGroup(gomock.Eq(securityGroupId)).
				Return(sgtc.expDeleteSecurityGroupErr, httpResponse)
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
