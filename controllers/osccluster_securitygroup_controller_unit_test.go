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
	"time"

	"github.com/benbjohnson/clock"
	"github.com/golang/mock/gomock"
	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/security/mock_security"
	osc "github.com/outscale/osc-sdk-go/v2"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	defaultSecurityGroupInitialize = infrastructurev1beta1.OscClusterSpec{
		Network: infrastructurev1beta1.OscNetwork{
			Net: infrastructurev1beta1.OscNet{
				Name:    "test-net",
				IPRange: "10.0.0.0/16",
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
		},
	}
	defaultSecurityGroupReconcile = infrastructurev1beta1.OscClusterSpec{
		Network: infrastructurev1beta1.OscNetwork{
			Net: infrastructurev1beta1.OscNet{Name: "test-net",
				IPRange:    "10.0.0.0/16",
				ResourceID: "vpc-test-net-uid",
			},
			SecurityGroups: []*infrastructurev1beta1.OscSecurityGroup{
				{Name: "test-securitygroup",
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
		},
	}
)

// SetupWithSecurityGroupMock set securityGroupMock with clusterScope and osccluster
func SetupWithSecurityGroupMock(t *testing.T, name string, spec infrastructurev1beta1.OscClusterSpec) (clusterScope *scope.ClusterScope, ctx context.Context, mockOscSecurityGroupInterface *mock_security.MockOscSecurityGroupInterface) {
	clusterScope = Setup(t, name, spec)
	mockCtrl := gomock.NewController(t)
	mockOscSecurityGroupInterface = mock_security.NewMockOscSecurityGroupInterface(mockCtrl)
	ctx = context.Background()
	return clusterScope, ctx, mockOscSecurityGroupInterface
}

// TestGetSecurityGroupResourceID has several tests to cover the code of the function getSecurityGrouptResourceId

func TestGetSecurityGroupResourceID(t *testing.T) {
	securityGroupTestCases := []struct {
		name                             string
		spec                             infrastructurev1beta1.OscClusterSpec
		expSecurityGroupsFound           bool
		expGetSecurityGroupResourceIDErr error
	}{
		{
			name:                             "get securityGroupId",
			spec:                             defaultSecurityGroupInitialize,
			expSecurityGroupsFound:           true,
			expGetSecurityGroupResourceIDErr: nil,
		},
		{
			name:                             "can not get securityGroupId",
			spec:                             defaultSecurityGroupInitialize,
			expSecurityGroupsFound:           false,
			expGetSecurityGroupResourceIDErr: fmt.Errorf("test-securitygroup-uid does not exist"),
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
				securityGroupResourceID, err := getSecurityGroupResourceID(securityGroupName, clusterScope)
				if err != nil {
					assert.Equal(t, sgtc.expGetSecurityGroupResourceIDErr, err, "getSecurityGroupResourceId() should return the same error")
				} else {
					assert.Nil(t, sgtc.expGetSecurityGroupResourceIDErr)
				}
				t.Logf("Find securityGroupResourceID %s\n", securityGroupResourceID)
			}
		})
	}
}

// TestGetSecurityGroupRuleResourceID has several tests to cover the code of the function getSecurityGroupRuleResourceId
func TestGetSecurityGroupRuleResourceID(t *testing.T) {
	securityGroupRuleTestCases := []struct {
		name                                 string
		spec                                 infrastructurev1beta1.OscClusterSpec
		expSecurityGroupRuleFound            bool
		expGetSecurityGroupRuleResourceIDErr error
	}{
		{
			name:                                 "get securityGroupRuleId",
			spec:                                 defaultSecurityGroupInitialize,
			expSecurityGroupRuleFound:            true,
			expGetSecurityGroupRuleResourceIDErr: nil,
		},
		{
			name:                                 "can not get securityGroupRuleId",
			spec:                                 defaultSecurityGroupInitialize,
			expSecurityGroupRuleFound:            false,
			expGetSecurityGroupRuleResourceIDErr: fmt.Errorf("test-securitygrouprule-uid does not exist"),
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
					securityGroupRuleResourceID, err := getSecurityGroupRulesResourceID(securityGroupRuleName, clusterScope)
					if err != nil {
						assert.Equal(t, sgrtc.expGetSecurityGroupRuleResourceIDErr, err, "getSecurityGroupRuleResourceId() should return the same error")
					} else {
						assert.Nil(t, sgrtc.expGetSecurityGroupRuleResourceIDErr)
					}
					t.Logf("Find securityGroupRuleResourceID %s\n", securityGroupRuleResourceID)
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
						IPRange: "10.0.0.0/16",
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
						IPRange: "10.0.0.0/16",
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
						IPRange: "10.0.0.0/16",
					},
					SecurityGroups: []*infrastructurev1beta1.OscSecurityGroup{
						{
							Name:        "test-securitygroup@test",
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
				},
			},
			expCheckSecurityGroupFormatParametersErr: fmt.Errorf("invalid Tag Name"),
		},
		{
			name: "check securityGroup bad description format",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IPRange: "10.0.0.0/16",
					},
					SecurityGroups: []*infrastructurev1beta1.OscSecurityGroup{
						{
							Name:        "test-securitygroup",
							Description: "test securitygroup λ",
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
				},
			},
			expCheckSecurityGroupFormatParametersErr: fmt.Errorf("invalid Description"),
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
						IPRange: "10.0.0.0/16",
					},
					SecurityGroups: []*infrastructurev1beta1.OscSecurityGroup{
						{
							Name:        "test-securitygroup",
							Description: "test securitygroup",
							SecurityGroupRules: []infrastructurev1beta1.OscSecurityGroupRule{
								{
									Name:          "test-securitygrouprule@test",
									Flow:          "Inbound",
									IPProtocol:    "tcp",
									IPRange:       "0.0.0.0/0",
									FromPortRange: 6443,
									ToPortRange:   6443,
								},
							},
						},
					},
				},
			},
			expCheckSecurityGroupRuleFormatParametersErr: fmt.Errorf("invalid Tag Name"),
		},
		{
			name: "check Bad Flow SecurityGroupRule",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IPRange: "10.0.0.0/16",
					},
					SecurityGroups: []*infrastructurev1beta1.OscSecurityGroup{
						{
							Name:        "test-securitygroup",
							Description: "test securitygroup",
							SecurityGroupRules: []infrastructurev1beta1.OscSecurityGroupRule{
								{
									Name:          "test-securitygrouprule",
									Flow:          "Nobound",
									IPProtocol:    "tcp",
									IPRange:       "0.0.0.0/0",
									FromPortRange: 6443,
									ToPortRange:   6443,
								},
							},
						},
					},
				},
			},
			expCheckSecurityGroupRuleFormatParametersErr: fmt.Errorf("invalid flow"),
		},
		{
			name: "check Bad IPProtocol SecurityGroupRule",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IPRange: "10.0.0.0/16",
					},
					SecurityGroups: []*infrastructurev1beta1.OscSecurityGroup{
						{
							Name:        "test-securitygroup",
							Description: "test securitygroup",
							SecurityGroupRules: []infrastructurev1beta1.OscSecurityGroupRule{
								{
									Name:          "test-securitygrouprule",
									Flow:          "Inbound",
									IPProtocol:    "sctp",
									IPRange:       "0.0.0.0/0",
									FromPortRange: 6443,
									ToPortRange:   6443,
								},
							},
						},
					},
				},
			},
			expCheckSecurityGroupRuleFormatParametersErr: fmt.Errorf("invalid protocol"),
		},
		{
			name: "check Bad Ip Range Prefix securityGroupRule",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IPRange: "10.0.0.0/16",
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
									IPRange:       "10.0.0.0/36",
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
						IPRange: "10.0.0.0/16",
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
									IPRange:       "10.0.0.256/16",
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
						IPRange: "10.0.0.0/16",
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
									FromPortRange: 65537,
									ToPortRange:   6443,
								},
							},
						},
					},
				},
			},
			expCheckSecurityGroupRuleFormatParametersErr: fmt.Errorf("invalid Port"),
		},
		{
			name: "check bad ToPortRange securityGroupRule",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IPRange: "10.0.0.0/16",
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
									ToPortRange:   65537,
								},
							},
						},
					},
				},
			},
			expCheckSecurityGroupRuleFormatParametersErr: fmt.Errorf("invalid Port"),
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
		expGetSecurityGroupFromSecurityGroupRuleErr error
		expCreateSecurityGroupRuleErr               error
		expReconcileSecurityGroupRuleErr            error
	}{
		{
			name:                            "create securityGroupRule  (first time reconcileloop)",
			spec:                            defaultSecurityGroupInitialize,
			expCreateSecurityGroupRuleFound: true,
			expGetSecurityGroupFromSecurityGroupRuleErr: nil,
			expCreateSecurityGroupRuleErr:               nil,
			expReconcileSecurityGroupRuleErr:            nil,
		},
		{
			name:                            "failed to create securityGroupRule",
			spec:                            defaultSecurityGroupInitialize,
			expCreateSecurityGroupRuleFound: false,
			expGetSecurityGroupFromSecurityGroupRuleErr: nil,
			expCreateSecurityGroupRuleErr:               fmt.Errorf("CreateSecurityGroupRule generic errors"),
			expReconcileSecurityGroupRuleErr:            fmt.Errorf("CreateSecurityGroupRule generic errors Can not create securityGroupRule for OscCluster test-system/test-osc"),
		},
	}
	for _, sgrtc := range securityGroupRuleTestCases {
		t.Run(sgrtc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscSecurityGroupInterface := SetupWithSecurityGroupMock(t, sgrtc.name, sgrtc.spec)
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
					securityGroupRuleIPProtocol := securityGroupRuleSpec.IPProtocol
					securityGroupRuleIPRange := securityGroupRuleSpec.IPRange
					securityGroupRuleFromPortRange := securityGroupRuleSpec.FromPortRange
					securityGroupRuleToPortRange := securityGroupRuleSpec.ToPortRange
					securityGroupMemberID := ""
					securityGroupRule := osc.CreateSecurityGroupRuleResponse{
						SecurityGroup: &osc.SecurityGroup{
							SecurityGroupId: &securityGroupId,
						},
					}

					mockOscSecurityGroupInterface.
						EXPECT().
						GetSecurityGroupFromSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIPProtocol), gomock.Eq(securityGroupRuleIPRange), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
						Return(nil, sgrtc.expGetSecurityGroupFromSecurityGroupRuleErr)

					if sgrtc.expCreateSecurityGroupRuleFound {
						mockOscSecurityGroupInterface.
							EXPECT().
							CreateSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIPProtocol), gomock.Eq(securityGroupRuleIPRange), gomock.Eq(securityGroupMemberID), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
							Return(securityGroupRule.SecurityGroup, sgrtc.expCreateSecurityGroupRuleErr)
					} else {
						mockOscSecurityGroupInterface.
							EXPECT().
							CreateSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIPProtocol), gomock.Eq(securityGroupRuleIPRange), gomock.Eq(securityGroupMemberID), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
							Return(nil, sgrtc.expCreateSecurityGroupRuleErr)
					}
					reconcileSecurityGroupRule, err := reconcileSecurityGroupRule(ctx, clusterScope, securityGroupRuleSpec, securityGroupName, mockOscSecurityGroupInterface)
					if err != nil {
						assert.Equal(t, sgrtc.expReconcileSecurityGroupRuleErr.Error(), err.Error(), "reconcileSecurityGroupRules() should return the same error")
					} else {
						assert.Nil(t, sgrtc.expReconcileSecurityGroupRuleErr)
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
		name                                        string
		spec                                        infrastructurev1beta1.OscClusterSpec
		expSecurityGroupRuleFound                   bool
		expGetSecurityGroupFromSecurityGroupRuleErr error
		expReconcileSecurityGroupRuleErr            error
	}{
		{
			name:                      "get securityGroupRule ((second time reconcile loop)",
			spec:                      defaultSecurityGroupReconcile,
			expSecurityGroupRuleFound: true,
			expGetSecurityGroupFromSecurityGroupRuleErr: nil,
			expReconcileSecurityGroupRuleErr:            nil,
		},
		{
			name:                      "failed to get securityGroup",
			spec:                      defaultSecurityGroupReconcile,
			expSecurityGroupRuleFound: true,
			expGetSecurityGroupFromSecurityGroupRuleErr: fmt.Errorf("GetSecurityGroupFromSecurityGroupRule generic errors"),
			expReconcileSecurityGroupRuleErr:            fmt.Errorf("GetSecurityGroupFromSecurityGroupRule generic errors"),
		},
	}
	for _, sgrtc := range securityGroupRuleTestCases {
		t.Run(sgrtc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscSecurityGroupInterface := SetupWithSecurityGroupMock(t, sgrtc.name, sgrtc.spec)
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
					securityGroupRuleIPProtocol := securityGroupRuleSpec.IPProtocol
					securityGroupRuleIPRange := securityGroupRuleSpec.IPRange
					securityGroupRuleFromPortRange := securityGroupRuleSpec.FromPortRange
					securityGroupRuleToPortRange := securityGroupRuleSpec.ToPortRange
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
							GetSecurityGroupFromSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIPProtocol), gomock.Eq(securityGroupRuleIPRange), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
							Return(&readSecurityGroup[0], sgrtc.expGetSecurityGroupFromSecurityGroupRuleErr)
					}
					reconcileSecurityGroupRule, err := reconcileSecurityGroupRule(ctx, clusterScope, securityGroupRuleSpec, securityGroupName, mockOscSecurityGroupInterface)
					if err != nil {
						assert.Equal(t, sgrtc.expReconcileSecurityGroupRuleErr, err, "reconcileSecurityGroupRules() should return the same error")
					} else {
						assert.Nil(t, sgrtc.expReconcileSecurityGroupRuleErr)
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
			clusterScope, ctx, mockOscSecurityGroupInterface := SetupWithSecurityGroupMock(t, sgrtc.name, sgrtc.spec)
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
					securityGroupRuleIPProtocol := securityGroupRuleSpec.IPProtocol
					securityGroupRuleIPRange := securityGroupRuleSpec.IPRange
					securityGroupRuleFromPortRange := securityGroupRuleSpec.FromPortRange
					securityGroupRuleToPortRange := securityGroupRuleSpec.ToPortRange
					securityGroupMemberID := ""
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
						GetSecurityGroupFromSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIPProtocol), gomock.Eq(securityGroupRuleIPRange), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
						Return(&readSecurityGroup[0], sgrtc.expGetSecurityGroupfromSecurityGroupRuleErr)

					mockOscSecurityGroupInterface.
						EXPECT().
						DeleteSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIPProtocol), gomock.Eq(securityGroupRuleIPRange), gomock.Eq(securityGroupMemberID), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
						Return(sgrtc.expDeleteSecurityGroupRuleErr)
					reconcileDeleteSecurityGroupRule, err := reconcileDeleteSecurityGroupRule(ctx, clusterScope, securityGroupRuleSpec, securityGroupName, mockOscSecurityGroupInterface)
					if err != nil {
						assert.Equal(t, sgrtc.expReconcileDeleteSecurityGroupRuleErr.Error(), err.Error(), "reconcileDeleteSecuritygroupRules() should return the same error")
					} else {
						assert.Nil(t, sgrtc.expReconcileDeleteSecurityGroupRuleErr)
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
		name                                        string
		spec                                        infrastructurev1beta1.OscClusterSpec
		expGetSecurityGroupfromSecurityGroupRuleErr error
		expReconcileDeleteSecurityGroupRuleErr      error
	}{
		{
			name: "failed to get securityGroupRule",
			spec: defaultSecurityGroupReconcile,
			expGetSecurityGroupfromSecurityGroupRuleErr: fmt.Errorf("GetSecurityGroupFromSecurityGroupRule generic errors"),
			expReconcileDeleteSecurityGroupRuleErr:      fmt.Errorf("GetSecurityGroupFromSecurityGroupRule generic errors"),
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
			clusterScope, ctx, mockOscSecurityGroupInterface := SetupWithSecurityGroupMock(t, sgrtc.name, sgrtc.spec)
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
					securityGroupRuleIPProtocol := securityGroupRuleSpec.IPProtocol
					securityGroupRuleIPRange := securityGroupRuleSpec.IPRange
					securityGroupRuleFromPortRange := securityGroupRuleSpec.FromPortRange
					securityGroupRuleToPortRange := securityGroupRuleSpec.ToPortRange

					mockOscSecurityGroupInterface.
						EXPECT().
						GetSecurityGroupFromSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIPProtocol), gomock.Eq(securityGroupRuleIPRange), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
						Return(nil, sgrtc.expGetSecurityGroupfromSecurityGroupRuleErr)
					reconcileDeleteSecurityGroupRule, err := reconcileDeleteSecurityGroupRule(ctx, clusterScope, securityGroupRuleSpec, securityGroupName, mockOscSecurityGroupInterface)
					if err != nil {
						assert.Equal(t, sgrtc.expReconcileDeleteSecurityGroupRuleErr, err, "reconcileDeleteSecuritygroupRules() should return the same error")
					} else {
						assert.Nil(t, sgrtc.expReconcileDeleteSecurityGroupRuleErr)
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
		expGetSecurityGroupFromNetIdsErr error
		expCreateSecurityGroupErr        error
		expGetSecurityGroupRuleErr       error
		expCreateSecurityGroupRuleErr    error

		expReconcileSecurityGroupErr error
	}{
		{
			name: "create securityGroup",

			spec:                             defaultSecurityGroupInitialize,
			expSecurityGroupRuleFound:        false,
			expCreateSecurityGroupRuleFound:  true,
			expGetSecurityGroupFromNetIdsErr: nil,
			expCreateSecurityGroupErr:        nil,
			expGetSecurityGroupRuleErr:       nil,
			expCreateSecurityGroupRuleErr:    nil,
			expReconcileSecurityGroupErr:     nil,
		},
		{
			name: "failed to create securityGroupRule",

			spec:                             defaultSecurityGroupInitialize,
			expSecurityGroupRuleFound:        false,
			expCreateSecurityGroupRuleFound:  false,
			expGetSecurityGroupFromNetIdsErr: nil,
			expCreateSecurityGroupErr:        nil,
			expGetSecurityGroupRuleErr:       nil,
			expCreateSecurityGroupRuleErr:    fmt.Errorf("CreateSecurityGroupRule generic errors"),
			expReconcileSecurityGroupErr:     fmt.Errorf("CreateSecurityGroupRule generic errors Can not create securityGroupRule for OscCluster test-system/test-osc"),
		},
	}
	for _, sgtc := range securityGroupTestCases {
		t.Run(sgtc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscSecurityGroupInterface := SetupWithSecurityGroupMock(t, sgtc.name, sgtc.spec)

			netName := sgtc.spec.Network.Net.Name + "-uid"
			netId := "vpc-" + netName
			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			netRef.ResourceMap[netName] = netId

			securityGroupsSpec := sgtc.spec.Network.SecurityGroups
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)

			for _, securityGroupSpec := range securityGroupsSpec {
				securityGroupName := securityGroupSpec.Name + "-uid"
				securityGroupId := "sg-" + securityGroupName
				securityGroupDescription := securityGroupSpec.Description
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
					CreateSecurityGroup(gomock.Eq(netId), gomock.Eq(securityGroupName), gomock.Eq(securityGroupDescription)).
					Return(securityGroup.SecurityGroup, sgtc.expCreateSecurityGroupErr)
				for _, securityGroupSpec := range securityGroupsSpec {
					securityGroupName := securityGroupSpec.Name + "-uid"
					securityGroupId := "sg-" + securityGroupName
					securityGroupsRef.ResourceMap[securityGroupName] = securityGroupId
					securityGroupRulesSpec := securityGroupSpec.SecurityGroupRules
					for _, securityGroupRuleSpec := range securityGroupRulesSpec {
						securityGroupRuleFlow := securityGroupRuleSpec.Flow
						securityGroupRuleIPProtocol := securityGroupRuleSpec.IPProtocol
						securityGroupRuleIPRange := securityGroupRuleSpec.IPRange
						securityGroupRuleFromPortRange := securityGroupRuleSpec.FromPortRange
						securityGroupRuleToPortRange := securityGroupRuleSpec.ToPortRange
						securityGroupMemberID := ""
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
								GetSecurityGroupFromSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIPProtocol), gomock.Eq(securityGroupRuleIPRange), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
								Return(&readSecurityGroup[0], sgtc.expGetSecurityGroupRuleErr)
						} else {
							mockOscSecurityGroupInterface.
								EXPECT().
								GetSecurityGroupFromSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIPProtocol), gomock.Eq(securityGroupRuleIPRange), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
								Return(nil, sgtc.expGetSecurityGroupRuleErr)
							if sgtc.expCreateSecurityGroupRuleFound {
								mockOscSecurityGroupInterface.
									EXPECT().
									CreateSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIPProtocol), gomock.Eq(securityGroupRuleIPRange), gomock.Eq(securityGroupMemberID), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
									Return(securityGroupRule.SecurityGroup, sgtc.expCreateSecurityGroupRuleErr)
							} else {
								mockOscSecurityGroupInterface.
									EXPECT().
									CreateSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIPProtocol), gomock.Eq(securityGroupRuleIPRange), gomock.Eq(securityGroupMemberID), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
									Return(nil, sgtc.expCreateSecurityGroupRuleErr)
							}
						}
					}
					reconcileSecurityGroup, err := reconcileSecurityGroup(ctx, clusterScope, mockOscSecurityGroupInterface)
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
		expGetSecurityGroupFromNetIdsErr error

		expReconcileSecurityGroupErr error
	}{
		{
			name: "failed to get securityGroup",

			spec:                             defaultSecurityGroupReconcile,
			expSecurityGroupFound:            false,
			expGetSecurityGroupFromNetIdsErr: fmt.Errorf("GetSecurityGroup generic error"),
			expReconcileSecurityGroupErr:     fmt.Errorf("GetSecurityGroup generic error"),
		},
		{
			name: "get securityGroup (second time reconcile loop)",

			spec:                             defaultSecurityGroupReconcile,
			expSecurityGroupFound:            true,
			expGetSecurityGroupFromNetIdsErr: nil,
			expReconcileSecurityGroupErr:     nil,
		},
	}
	for _, sgtc := range securityGroupTestCases {
		t.Run(sgtc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscSecurityGroupInterface := SetupWithSecurityGroupMock(t, sgtc.name, sgtc.spec)

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
				reconcileSecurityGroup, err := reconcileSecurityGroup(ctx, clusterScope, mockOscSecurityGroupInterface)
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
		expGetSecurityGroupFromNetIdsErr error
		expCreateSecurityGroupErr        error
		expReconcileSecurityGroupErr     error
	}{
		{
			name: "failed to create securityGroup",
			spec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "test-net",
						IPRange: "10.0.0.0/16",
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
				},
			},
			expGetSecurityGroupFromNetIdsErr: nil,
			expCreateSecurityGroupErr:        fmt.Errorf("CreateSecurityGroup generic error"),
			expReconcileSecurityGroupErr:     fmt.Errorf("CreateSecurityGroup generic error Can not create securityGroup for Osccluster test-system/test-osc"),
		},
	}
	for _, sgtc := range securityGroupTestCases {
		t.Run(sgtc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscSecurityGroupInterface := SetupWithSecurityGroupMock(t, sgtc.name, sgtc.spec)

			netName := sgtc.spec.Network.Net.Name + "-uid"
			netId := "vpc-" + netName
			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			netRef.ResourceMap[netName] = netId

			securityGroupsSpec := sgtc.spec.Network.SecurityGroups
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)

			for _, securityGroupSpec := range securityGroupsSpec {
				securityGroupName := securityGroupSpec.Name + "-uid"
				securityGroupId := "sg-" + securityGroupName
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
					CreateSecurityGroup(gomock.Eq(netId), gomock.Eq(securityGroupName), gomock.Eq(securityGroupDescription)).
					Return(securityGroup.SecurityGroup, sgtc.expCreateSecurityGroupErr)
			}
			reconcileSecurityGroup, err := reconcileSecurityGroup(ctx, clusterScope, mockOscSecurityGroupInterface)
			if err != nil {
				assert.Equal(t, sgtc.expReconcileSecurityGroupErr.Error(), err.Error(), "reconcileSecurityGroup() should return the same error")
			} else {
				assert.Nil(t, sgtc.expReconcileSecurityGroupErr)
			}

			t.Logf("find reconcileSecurityGroup %v\n", reconcileSecurityGroup)
		})
	}
}

// TestReconcileCreateSecurityGroupResourceID has several tests to cover the code of the function reconcileCreateSecurityGroup
func TestReconcileCreateSecurityGroupResourceID(t *testing.T) {
	securityGroupTestCases := []struct {
		name                         string
		spec                         infrastructurev1beta1.OscClusterSpec
		expReconcileSecurityGroupErr error
	}{
		{
			name: "net does not exist",

			spec:                         defaultSecurityGroupReconcile,
			expReconcileSecurityGroupErr: fmt.Errorf("test-net-uid does not exist"),
		},
	}
	for _, sgtc := range securityGroupTestCases {
		t.Run(sgtc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscSecurityGroupInterface := SetupWithSecurityGroupMock(t, sgtc.name, sgtc.spec)
			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			reconcileSecurityGroup, err := reconcileSecurityGroup(ctx, clusterScope, mockOscSecurityGroupInterface)
			if err != nil {
				assert.Equal(t, sgtc.expReconcileSecurityGroupErr, err, "reconcileSecurityGroup() should return the same error")
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
		expinvalidDeleteSecurityGroupJsonResponse bool
		expDeleteSecurityGroupFirstMockErr        error
		expDeleteSecurityGroupError               error
	}{
		{
			name:                            "delete securityGroup",
			spec:                            defaultSecurityGroupReconcile,
			expSecurityGroupFound:           true,
			expLoadBalancerResourceConflict: false,
			expinvalidDeleteSecurityGroupJsonResponse: false,
			expDeleteSecurityGroupFirstMockErr:        nil,
			expDeleteSecurityGroupError:               nil,
		},
		{
			name:                            "delete securityGroup unmatch to catch",
			spec:                            defaultSecurityGroupReconcile,
			expSecurityGroupFound:           true,
			expLoadBalancerResourceConflict: false,
			expinvalidDeleteSecurityGroupJsonResponse: false,
			expDeleteSecurityGroupFirstMockErr:        fmt.Errorf("DeleteSecurityGroup first generic error"),
			expDeleteSecurityGroupError:               fmt.Errorf(" Can not delete securityGroup because of the uncatch error for Osccluster test-system/test-osc"),
		},
		{
			name:                            "invalid json response",
			spec:                            defaultSecurityGroupReconcile,
			expSecurityGroupFound:           true,
			expLoadBalancerResourceConflict: false,
			expinvalidDeleteSecurityGroupJsonResponse: true,
			expDeleteSecurityGroupFirstMockErr:        fmt.Errorf("DeleteSecurityGroup first generic error"),
			expDeleteSecurityGroupError:               fmt.Errorf("invalid character 'B' looking for beginning of value Can not delete securityGroup for Osccluster test-system/test-osc"),
		},
		{
			name:                            "waiting loadbalancer to timeout",
			spec:                            defaultSecurityGroupReconcile,
			expSecurityGroupFound:           true,
			expLoadBalancerResourceConflict: true,
			expinvalidDeleteSecurityGroupJsonResponse: false,
			expDeleteSecurityGroupFirstMockErr:        fmt.Errorf("DeleteSecurityGroup first generic error"),
			expDeleteSecurityGroupError:               fmt.Errorf("DeleteSecurityGroup first generic error Can not delete securityGroup because to waiting loadbalancer to be delete timeout  for Osccluster test-system/test-osc"),
		},
	}
	for _, sgtc := range securityGroupTestCases {
		t.Run(sgtc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscSecurityGroupInterface := SetupWithSecurityGroupMock(t, sgtc.name, sgtc.spec)
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
					if sgtc.expinvalidDeleteSecurityGroupJsonResponse {
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
						Return(httpResponse, sgtc.expDeleteSecurityGroupFirstMockErr)

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
						Return(httpResponse, nil)
				}

				wg.Add(1)
				go func() {
					clock_mock.Sleep(5 * time.Second)
					deleteSg, err = deleteSecurityGroup(ctx, clusterScope, securityGroupId, mockOscSecurityGroupInterface, clock_mock)
					wg.Done()
				}()
				runtime.Gosched()
				clock_mock.Add(120 * time.Second)
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

// TestReconcileDeleteSecurityGroup has several tests to cover the code of the function reconcileDeleteSecurityGroup
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
			clusterScope, ctx, mockOscSecurityGroupInterface := SetupWithSecurityGroupMock(t, sgtc.name, sgtc.spec)

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
							securityGroupRuleIPProtocol := securityGroupRuleSpec.IPProtocol
							securityGroupRuleIPRange := securityGroupRuleSpec.IPRange
							securityGroupRuleFromPortRange := securityGroupRuleSpec.FromPortRange
							securityGroupRuleToPortRange := securityGroupRuleSpec.ToPortRange
							securityGroupMemberID := ""
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
									GetSecurityGroupFromSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIPProtocol), gomock.Eq(securityGroupRuleIPRange), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
									Return(&readSecurityGroup[0], sgtc.expGetSecurityGroupfromSecurityGroupRuleErr)
							} else {
								mockOscSecurityGroupInterface.
									EXPECT().
									GetSecurityGroupFromSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIPProtocol), gomock.Eq(securityGroupRuleIPRange), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
									Return(nil, sgtc.expGetSecurityGroupfromSecurityGroupRuleErr)
							}
							mockOscSecurityGroupInterface.
								EXPECT().
								DeleteSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIPProtocol), gomock.Eq(securityGroupRuleIPRange), gomock.Eq(securityGroupMemberID), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
								Return(sgtc.expDeleteSecurityGroupRuleErr)
						}
					}
				}
			}
			reconcileDeleteSecurityGroup, err := reconcileDeleteSecurityGroup(ctx, clusterScope, mockOscSecurityGroupInterface)
			if err != nil {
				assert.Equal(t, sgtc.expReconcileDeleteSecurityGroupErr.Error(), err.Error(), "reconcileDeleteSecurityGroup() should return the same error")
			} else {
				assert.Nil(t, sgtc.expReconcileDeleteSecurityGroupErr)
			}
			t.Logf("find reconcileDeleteSecurityGroup %v\n", reconcileDeleteSecurityGroup)
		})
	}
}

// TestReconcileDeleteSecurityGroupDelete has several tests to cover the code of the function reconcileDeleteSecurityGroup
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
			clusterScope, ctx, mockOscSecurityGroupInterface := SetupWithSecurityGroupMock(t, sgtc.name, sgtc.spec)

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
					securityGroupRuleIPProtocol := securityGroupRuleSpec.IPProtocol
					securityGroupRuleIPRange := securityGroupRuleSpec.IPRange
					securityGroupRuleFromPortRange := securityGroupRuleSpec.FromPortRange
					securityGroupRuleToPortRange := securityGroupRuleSpec.ToPortRange
					securityGroupMemberID := ""
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
						GetSecurityGroupFromSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIPProtocol), gomock.Eq(securityGroupRuleIPRange), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
						Return(&readSecurityGroup[0], sgtc.expGetSecurityGroupfromSecurityGroupRuleErr)
					mockOscSecurityGroupInterface.
						EXPECT().
						DeleteSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleFlow), gomock.Eq(securityGroupRuleIPProtocol), gomock.Eq(securityGroupRuleIPRange), gomock.Eq(securityGroupMemberID), gomock.Eq(securityGroupRuleFromPortRange), gomock.Eq(securityGroupRuleToPortRange)).
						Return(sgtc.expDeleteSecurityGroupRuleErr)

				}
				mockOscSecurityGroupInterface.
					EXPECT().
					DeleteSecurityGroup(gomock.Eq(securityGroupId)).
					Return(httpResponse, sgtc.expDeleteSecurityGroupErr)
			}
			reconcileDeleteSecurityGroup, err := reconcileDeleteSecurityGroup(ctx, clusterScope, mockOscSecurityGroupInterface)
			if err != nil {
				assert.Equal(t, sgtc.expReconcileDeleteSecurityGroupErr.Error(), err.Error(), "reconcileDeleteSecurityGroup() should return the same error")
			} else {
				assert.Nil(t, sgtc.expReconcileDeleteSecurityGroupErr)
			}
			t.Logf("find reconcileDeleteSecurityGroup %v\n", reconcileDeleteSecurityGroup)
		})
	}
}

// TestReconcileDeleteSecurityGroupDeleteWithoutSpec has several tests to cover the code of the function reconcileDeleteSecurityGroup
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
			clusterScope, ctx, mockOscSecurityGroupInterface := SetupWithSecurityGroupMock(t, sgtc.name, sgtc.spec)

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
			var httpResponse = &http.Response{
				StatusCode: 200,
				Body: ioutil.NopCloser(strings.NewReader(`{
	                                                "ResponseContext": {
        	                                                "RequestId": "aaaaa-bbbbb-ccccc"
                	                                }
                        	                }`)),
			}
			securityGroupRuleKubeletKwFlow := "Inbound"
			securityGroupRuleKubeletKwIPProtocol := "tcp"
			securityGroupRuleKubeletKwIPRange := "10.0.0.128/26"
			securityGroupMemberID := ""
			var securityGroupRuleKubeletKwFromPortRange int32 = 10250
			var securityGroupRuleKubeletKwToPortRange int32 = 10250

			securityGroupRuleNodeIpKwFlow := "Inbound"
			securityGroupRuleNodeIpKwIPProtocol := "tcp"
			securityGroupRuleNodeIpKwIPRange := "10.0.0.128/26"
			var securityGroupRuleNodeIpKwFromPortRange int32 = 30000
			var securityGroupRuleNodeIpKwToPortRange int32 = 32767

			securityGroupRuleNodeIpKcpFlow := "Inbound"
			securityGroupRuleNodeIpKcpIPProtocol := "tcp"
			securityGroupRuleNodeIpKcpIPRange := "10.0.0.32/28"
			var securityGroupRuleNodeIpKcpFromPortRange int32 = 30000
			var securityGroupRuleNodeIpKcpToPortRange int32 = 32767

			securityGroupRuleKubeletKcpFlow := "Inbound"
			securityGroupRuleKubeletKcpIPProtocol := "tcp"
			securityGroupRuleKubeletKcpIPRange := "10.0.0.32/28"
			var securityGroupRuleKubeletKcpFromPortRange int32 = 10250
			var securityGroupRuleKubeletKcpToPortRange int32 = 10250

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
				GetSecurityGroupFromSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleKubeletKwFlow), gomock.Eq(securityGroupRuleKubeletKwIPProtocol), gomock.Eq(securityGroupRuleKubeletKwIPRange), gomock.Eq(securityGroupRuleKubeletKwFromPortRange), gomock.Eq(securityGroupRuleKubeletKwToPortRange)).
				Return(&readSecurityGroup[0], sgtc.expGetSecurityGroupfromSecurityGroupRuleErr)
			mockOscSecurityGroupInterface.
				EXPECT().
				DeleteSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleKubeletKwFlow), gomock.Eq(securityGroupRuleKubeletKwIPProtocol), gomock.Eq(securityGroupRuleKubeletKwIPRange), gomock.Eq(securityGroupMemberID), gomock.Eq(securityGroupRuleKubeletKwFromPortRange), gomock.Eq(securityGroupRuleKubeletKwToPortRange)).
				Return(sgtc.expDeleteSecurityGroupRuleErr)

			mockOscSecurityGroupInterface.
				EXPECT().
				GetSecurityGroupFromSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleKubeletKcpFlow), gomock.Eq(securityGroupRuleKubeletKcpIPProtocol), gomock.Eq(securityGroupRuleKubeletKcpIPRange), gomock.Eq(securityGroupRuleKubeletKcpFromPortRange), gomock.Eq(securityGroupRuleKubeletKcpToPortRange)).
				Return(&readSecurityGroup[0], sgtc.expGetSecurityGroupfromSecurityGroupRuleErr)
			mockOscSecurityGroupInterface.
				EXPECT().
				DeleteSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleKubeletKcpFlow), gomock.Eq(securityGroupRuleKubeletKcpIPProtocol), gomock.Eq(securityGroupRuleKubeletKcpIPRange), gomock.Eq(securityGroupMemberID), gomock.Eq(securityGroupRuleKubeletKcpFromPortRange), gomock.Eq(securityGroupRuleKubeletKcpToPortRange)).
				Return(sgtc.expDeleteSecurityGroupRuleErr)

			mockOscSecurityGroupInterface.
				EXPECT().
				GetSecurityGroupFromSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleNodeIpKwFlow), gomock.Eq(securityGroupRuleNodeIpKwIPProtocol), gomock.Eq(securityGroupRuleNodeIpKwIPRange), gomock.Eq(securityGroupRuleNodeIpKwFromPortRange), gomock.Eq(securityGroupRuleNodeIpKwToPortRange)).
				Return(&readSecurityGroup[0], sgtc.expGetSecurityGroupfromSecurityGroupRuleErr)
			mockOscSecurityGroupInterface.
				EXPECT().
				DeleteSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleNodeIpKwFlow), gomock.Eq(securityGroupRuleNodeIpKwIPProtocol), gomock.Eq(securityGroupRuleNodeIpKwIPRange), gomock.Eq(securityGroupMemberID), gomock.Eq(securityGroupRuleNodeIpKwFromPortRange), gomock.Eq(securityGroupRuleNodeIpKwToPortRange)).
				Return(sgtc.expDeleteSecurityGroupRuleErr)

			mockOscSecurityGroupInterface.
				EXPECT().
				GetSecurityGroupFromSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleNodeIpKcpFlow), gomock.Eq(securityGroupRuleNodeIpKcpIPProtocol), gomock.Eq(securityGroupRuleNodeIpKcpIPRange), gomock.Eq(securityGroupRuleNodeIpKcpFromPortRange), gomock.Eq(securityGroupRuleNodeIpKcpToPortRange)).
				Return(&readSecurityGroup[0], sgtc.expGetSecurityGroupfromSecurityGroupRuleErr)
			mockOscSecurityGroupInterface.
				EXPECT().
				DeleteSecurityGroupRule(gomock.Eq(securityGroupId), gomock.Eq(securityGroupRuleNodeIpKcpFlow), gomock.Eq(securityGroupRuleNodeIpKcpIPProtocol), gomock.Eq(securityGroupRuleNodeIpKcpIPRange), gomock.Eq(securityGroupMemberID), gomock.Eq(securityGroupRuleNodeIpKcpFromPortRange), gomock.Eq(securityGroupRuleNodeIpKcpToPortRange)).
				Return(sgtc.expDeleteSecurityGroupRuleErr)

			mockOscSecurityGroupInterface.
				EXPECT().
				DeleteSecurityGroup(gomock.Eq(securityGroupId)).
				Return(httpResponse, sgtc.expDeleteSecurityGroupErr)
			reconcileDeleteSecurityGroup, err := reconcileDeleteSecurityGroup(ctx, clusterScope, mockOscSecurityGroupInterface)
			if err != nil {
				assert.Equal(t, sgtc.expReconcileDeleteSecurityGroupErr.Error(), err.Error(), "reconcileDeleteSecurityGroup() should return the same error")
			} else {
				assert.Nil(t, sgtc.expReconcileDeleteSecurityGroupErr)
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
			clusterScope, ctx, mockOscSecurityGroupInterface := SetupWithSecurityGroupMock(t, sgtc.name, sgtc.spec)

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
			reconcileDeleteSecurityGroup, err := reconcileDeleteSecurityGroup(ctx, clusterScope, mockOscSecurityGroupInterface)
			if err != nil {
				assert.Equal(t, sgtc.expReconcileDeleteSecurityGroupErr.Error(), err.Error(), "reconcileDeleteSecurityGroup() should return the same error")
			} else {
				assert.Nil(t, sgtc.expReconcileDeleteSecurityGroupErr)
			}
			t.Logf("find reconcileDeleteSecurityGroup %v\n", reconcileDeleteSecurityGroup)
		})
	}
}

// TestReconcileDeleteSecurityGroupResourceID has several tests to cover the code of the function reconcileDeleteSecurityGroup
func TestReconcileDeleteSecurityGroupResourceID(t *testing.T) {
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
			clusterScope, ctx, mockOscSecurityGroupInterface := SetupWithSecurityGroupMock(t, sgtc.name, sgtc.spec)
			securityGroupsRef := clusterScope.GetSecurityGroupsRef()
			securityGroupsRef.ResourceMap = make(map[string]string)
			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			reconcileDeleteSecurityGroup, err := reconcileDeleteSecurityGroup(ctx, clusterScope, mockOscSecurityGroupInterface)
			if err != nil {
				assert.Equal(t, sgtc.expReconcileDeleteSecurityGroupErr.Error(), err.Error(), "reconcileDeleteSecurityGroup() should return the same error")
			} else {
				assert.Nil(t, sgtc.expReconcileDeleteSecurityGroupErr)
			}
			t.Logf("find reconcileDeleteSecurityGroup %v\n", reconcileDeleteSecurityGroup)
		})
	}
}
