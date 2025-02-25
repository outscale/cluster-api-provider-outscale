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
	"testing"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	defaultVmMachineTemplateInitialize = infrastructurev1beta1.OscMachineTemplateSpec{
		Template: infrastructurev1beta1.OscMachineTemplateResource{
			Spec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Volumes: []*infrastructurev1beta1.OscVolume{
						{
							Name:       "test-volume",
							Iops:       1000,
							Size:       50,
							VolumeType: "io1",
						},
					},
					Vm: infrastructurev1beta1.OscVm{
						ClusterName: "test-cluster",
						Name:        "test-vm",
						ImageId:     "ami-00000000",
						Role:        "controlplane",
						DeviceName:  "/dev/sda1",
						RootDisk: infrastructurev1beta1.OscRootDisk{
							RootDiskSize: 30,
							RootDiskIops: 1500,
							RootDiskType: "io1",
						},
						KeypairName:      "rke",
						SubregionName:    "eu-west-2a",
						SubnetName:       "test-subnet",
						LoadBalancerName: "test-loadbalancer",
						PublicIpName:     "test-publicip",
						VmType:           "tinav3.c2r4p2",
						Replica:          1,
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
		},
	}
	awsTypeVmMachineTemplate = infrastructurev1beta1.OscMachineTemplateSpec{
		Template: infrastructurev1beta1.OscMachineTemplateResource{
			Spec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						VmType: "m4.2xlarge",
					},
				},
			},
		},
	}
)

// Setup set osccluster, oscmachine, machineScope and clusterScope
func SetupMachineTemplate(t *testing.T, name string, machineSpec infrastructurev1beta1.OscMachineTemplateSpec) (machineTemplateScope *scope.MachineTemplateScope) {
	t.Logf("Validate to %s", name)

	oscMachineTemplate := infrastructurev1beta1.OscMachineTemplate{
		Spec: machineSpec,
		ObjectMeta: metav1.ObjectMeta{
			UID:       "uid",
			Name:      "test-osc",
			Namespace: "test-system",
		},
	}

	machineTemplateScope = &scope.MachineTemplateScope{
		OscMachineTemplate: &oscMachineTemplate,
	}
	return machineTemplateScope
}

// TestReconcileCapacity tests that reconcileCapacity correctly sets Status.Capacity.
func TestReconcileCapacity(t *testing.T) {
	capacityTestCases := []struct {
		name                string
		machineTemplateSpec infrastructurev1beta1.OscMachineTemplateSpec
		expGetCapacityFound bool
	}{
		{
			name:                "with tina vm type",
			machineTemplateSpec: defaultVmMachineTemplateInitialize,
			expGetCapacityFound: true,
		},
		{
			name:                "with aws vm type, no capacity is found",
			machineTemplateSpec: awsTypeVmMachineTemplate,
		},
	}
	for _, ctc := range capacityTestCases {
		t.Run(ctc.name, func(t *testing.T) {
			machineTemplateScope := SetupMachineTemplate(t, ctc.name, ctc.machineTemplateSpec)
			var capacity corev1.ResourceList
			if ctc.expGetCapacityFound {
				capacity = make(corev1.ResourceList)
				memory, err := resource.ParseQuantity("4Gi")
				require.NoError(t, err)
				cpu, err := resource.ParseQuantity("2")
				require.NoError(t, err)
				capacity[corev1.ResourceMemory] = memory
				capacity[corev1.ResourceCPU] = cpu
			}
			reconcileCapacity, err := reconcileCapacity(context.TODO(), machineTemplateScope)
			require.NoError(t, err)
			assert.Zero(t, reconcileCapacity)
			assert.Equal(t, capacity, machineTemplateScope.GetCapacity())
		})
	}
}
