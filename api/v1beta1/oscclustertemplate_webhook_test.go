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

package v1beta1

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestOscClusterTemplate_ValidateCreate check good and bad validation of oscCluster spec
func TestOscClusterTemplate_ValidateCreate(t *testing.T) {
	clusterTestCases := []struct {
		name                 string
		clusterSpec          OscClusterSpec
		expValidateCreateErr error
	}{
		{
			name: "create with bad loadBalancerName",
			clusterSpec: OscClusterSpec{
				Network: OscNetwork{
					LoadBalancer: OscLoadBalancer{
						LoadBalancerName: "test-webhook@test",
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscClusterTemplate.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: loadBalancerName: Invalid value: \"test-webhook@test\": Invalid Description"),
		},
	}
	for _, ctc := range clusterTestCases {
		t.Run(ctc.name, func(t *testing.T) {
			oscInfraClusterTemplate := createOscInfraClusterTemplate(ctc.clusterSpec, "webhook-test", "default")
			_, err := oscInfraClusterTemplate.ValidateCreate()
			if err != nil {
				assert.Equal(t, ctc.expValidateCreateErr.Error(), err.Error(), "ValidateCreate() should return the same error")
			} else {
				assert.Nil(t, ctc.expValidateCreateErr)
			}
		})
	}
}

// TestOscClusterTemplate_ValidateUpdate check good and bad update of oscClusterTemplate
func TestOscClusterTemplate_ValidateUpdate(t *testing.T) {
	clusterTestCases := []struct {
		name                 string
		oldClusterSpec       OscClusterSpec
		newClusterSpec       OscClusterSpec
		expValidateUpdateErr error
	}{
		{
			name: "Update only oscClusterTemplate name",
			oldClusterSpec: OscClusterSpec{
				Network: OscNetwork{
					Net: OscNet{
						Name:    "test-webhook",
						IpRange: "10.0.0.0/24",
					},
					Subnets: []*OscSubnet{
						{
							Name:          "test-webhook",
							IpSubnetRange: "10.0.0.32/28",
						},
					},
					RouteTables: []*OscRouteTable{
						{
							Name: "test-webhook",
							Routes: []OscRoute{
								{
									Name:        "test-webhook",
									Destination: "0.0.0.0/0",
								},
							},
						},
					},
					SecurityGroups: []*OscSecurityGroup{
						{
							Name:        "test-webhook",
							Description: "test webhook",
							SecurityGroupRules: []OscSecurityGroupRule{
								{
									Name:          "test-webhook",
									Flow:          "Inbound",
									IpProtocol:    "tcp",
									IpRange:       "10.0.0.32/28",
									FromPortRange: 10250,
									ToPortRange:   10250,
								},
							},
						},
					},
					LoadBalancer: OscLoadBalancer{
						LoadBalancerName: "test-webhook",
						LoadBalancerType: "internet-facing",
					},
				},
			},
			newClusterSpec: OscClusterSpec{

				Network: OscNetwork{
					Net: OscNet{
						Name:    "test-webhook",
						IpRange: "10.0.0.0/24",
					},
					Subnets: []*OscSubnet{
						{
							Name:          "test-webhook",
							IpSubnetRange: "10.0.0.32/28",
						},
					},
					RouteTables: []*OscRouteTable{
						{
							Name: "test-webhook",
							Routes: []OscRoute{
								{
									Name:        "test-webhook",
									Destination: "0.0.0.0/0",
								},
							},
						},
					},
					SecurityGroups: []*OscSecurityGroup{
						{
							Name:        "test-webhook",
							Description: "test webhook",
							SecurityGroupRules: []OscSecurityGroupRule{
								{
									Name:          "test-webhook",
									Flow:          "Inbound",
									IpProtocol:    "tcp",
									IpRange:       "10.0.0.32/28",
									FromPortRange: 10250,
									ToPortRange:   10250,
								},
							},
						},
					},
					LoadBalancer: OscLoadBalancer{
						LoadBalancerName: "test-webhook",
						LoadBalancerType: "internet-facing",
					},
				},
			},
			expValidateUpdateErr: nil,
		},
	}
	for _, ctc := range clusterTestCases {
		t.Run(ctc.name, func(t *testing.T) {
			oscOldInfraClusterTemplate := createOscInfraClusterTemplate(ctc.oldClusterSpec, "old-webhook-test", "default")
			oscInfraClusterTemplate := createOscInfraClusterTemplate(ctc.newClusterSpec, "webhook-test", "default")
			_, err := oscInfraClusterTemplate.ValidateUpdate(oscOldInfraClusterTemplate)
			if err != nil {
				assert.Equal(t, ctc.expValidateUpdateErr.Error(), err.Error(), "ValidateUpdate should return the same error")
			} else {
				assert.Nil(t, ctc.expValidateUpdateErr)
			}
		})
	}
}

// createOscInfraClusterTemplate create oscInfraClusterTemplate
func createOscInfraClusterTemplate(infraClusterSpec OscClusterSpec, name string, namespace string) *OscClusterTemplate {
	oscInfraClusterTemplate := &OscClusterTemplate{
		Spec: OscClusterTemplateSpec{
			Template: OscClusterTemplateResource{
				Spec: infraClusterSpec,
			},
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	return oscInfraClusterTemplate
}
