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

package v1beta1_test

import (
	"context"
	"errors"
	"testing"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestOscClusterTemplate_ValidateCreate check good and bad validation of oscCluster spec
func TestOscClusterTemplate_ValidateCreate(t *testing.T) {
	clusterTestCases := []struct {
		name                 string
		clusterSpec          infrastructurev1beta1.OscClusterSpec
		expValidateCreateErr error
	}{
		{
			name: "create with bad loadBalancerName",
			clusterSpec: infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					LoadBalancer: infrastructurev1beta1.OscLoadBalancer{
						LoadBalancerName: "test-webhook@test",
					},
				},
			},
			expValidateCreateErr: errors.New("OscClusterTemplate.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: network.loadBalancer.loadbalancername: Invalid value: \"test-webhook@test\": invalid loadBalancer name"),
		},
	}
	h := infrastructurev1beta1.OscClusterTemplateWebhook{}
	for _, ctc := range clusterTestCases {
		t.Run(ctc.name, func(t *testing.T) {
			oscInfraClusterTemplate := createOscInfraClusterTemplate(ctc.clusterSpec, "webhook-test", "default")
			_, err := h.ValidateCreate(context.TODO(), oscInfraClusterTemplate)
			if ctc.expValidateCreateErr != nil {
				require.EqualError(t, err, ctc.expValidateCreateErr.Error(), "ValidateCreate() should return the same error")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// createOscInfraClusterTemplate create oscInfraClusterTemplate
func createOscInfraClusterTemplate(infraClusterSpec infrastructurev1beta1.OscClusterSpec, name string, namespace string) *infrastructurev1beta1.OscClusterTemplate {
	oscInfraClusterTemplate := &infrastructurev1beta1.OscClusterTemplate{
		Spec: infrastructurev1beta1.OscClusterTemplateSpec{
			Template: infrastructurev1beta1.OscClusterTemplateResource{
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
