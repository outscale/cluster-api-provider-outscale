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
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestOscClusterTemplate_ValidateCreate check good and bad validation of oscCluster spec
func TestOscCluster_ValidateCreate(t *testing.T) {
	clusterTestCases := []struct {
		name                 string
		clusterSpec          OscClusterSpec
		expValidateCreateErr error
	}{
		{
			name: "bad loadBalancerName",
			clusterSpec: OscClusterSpec{
				Network: OscNetwork{
					LoadBalancer: OscLoadBalancer{
						LoadBalancerName: "test-webhook@test",
					},
				},
			},
			expValidateCreateErr: errors.New("OscCluster.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: network.loadBalancer.loadbalancername: Invalid value: \"test-webhook@test\": invalid loadBalancer name"),
		},
		{
			name: "bad loadBalancerType",
			clusterSpec: OscClusterSpec{
				Network: OscNetwork{
					LoadBalancer: OscLoadBalancer{
						LoadBalancerName: "foo",
						LoadBalancerType: "foo",
					},
				},
			},
			expValidateCreateErr: errors.New("OscCluster.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: network.loadBalancer.loadbalancertype: Invalid value: \"foo\": only internet-facing or internal are allowed"),
		},
		{
			name: "bad cidr",
			clusterSpec: OscClusterSpec{
				Network: OscNetwork{
					Net: OscNet{
						IpRange: "1.2.3.4",
					},
					LoadBalancer: OscLoadBalancer{
						LoadBalancerName: "foo",
					},
				},
			},
			expValidateCreateErr: errors.New("OscCluster.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: network.net.ipRange: Invalid value: \"1.2.3.4\": invalid CIDR address"),
		},
		{
			name: "bad subnets",
			clusterSpec: OscClusterSpec{
				Network: OscNetwork{
					Net: OscNet{
						IpRange: "10.0.0.0/24",
					},
					Subnets: []OscSubnet{{IpSubnetRange: "1.2.3.4"}},
					LoadBalancer: OscLoadBalancer{
						LoadBalancerName: "foo",
					},
				},
			},
			expValidateCreateErr: errors.New("OscCluster.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: network.subnets.ipSubnetRange: Invalid value: \"1.2.3.4\": invalid CIDR address"),
		},
		{
			name: "subnet not within net",
			clusterSpec: OscClusterSpec{
				Network: OscNetwork{
					Net: OscNet{
						IpRange: "10.0.0.0/16",
					},
					Subnets: []OscSubnet{{IpSubnetRange: "11.0.0.0/24"}},
					LoadBalancer: OscLoadBalancer{
						LoadBalancerName: "foo",
					},
				},
			},
			expValidateCreateErr: errors.New("OscCluster.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: network.subnets.ipSubnetRange: Invalid value: \"11.0.0.0/24\": subnet must be contained in net"),
		},
		{
			name: "overlapping subnets",
			clusterSpec: OscClusterSpec{
				Network: OscNetwork{
					Net: OscNet{
						IpRange: "10.0.0.0/16",
					},
					Subnets: []OscSubnet{{IpSubnetRange: "10.0.1.0/24"}, {IpSubnetRange: "10.0.1.0/24"}},
					LoadBalancer: OscLoadBalancer{
						LoadBalancerName: "foo",
					},
				},
			},
			expValidateCreateErr: errors.New("OscCluster.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: network.subnets.ipSubnetRange: Invalid value: \"10.0.1.0/24\": subnet overlaps 10.0.1.0/24"),
		},
	}
	for _, ctc := range clusterTestCases {
		t.Run(ctc.name, func(t *testing.T) {
			oscInfraCluster := createOscInfraCluster(ctc.clusterSpec, "webhook-test", "default")
			_, err := oscInfraCluster.ValidateCreate()
			if ctc.expValidateCreateErr != nil {
				require.EqualError(t, err, ctc.expValidateCreateErr.Error(), "ValidateCreate() should return the right error")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// createOscInfraCluster create oscInfraCluster
func createOscInfraCluster(infraClusterSpec OscClusterSpec, name string, namespace string) *OscCluster {
	oscInfraCluster := &OscCluster{
		Spec: infraClusterSpec,
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	return oscInfraCluster
}
