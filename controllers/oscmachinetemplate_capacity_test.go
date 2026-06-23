/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package controllers

import (
	"context"
	"testing"

	infrastructurev1beta2 "github.com/outscale/cluster-api-provider-outscale/api/v1beta2"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	defaultVmMachineTemplateInitialize = infrastructurev1beta2.OscMachineTemplateSpec{
		Template: infrastructurev1beta2.OscMachineTemplateResource{
			Spec: infrastructurev1beta2.OscMachineSpec{
				Node: infrastructurev1beta2.OscNode{
					Volumes: []infrastructurev1beta2.OscVolume{
						{
							Name:       "test-volume",
							Iops:       1000,
							Size:       50,
							VolumeType: "io1",
						},
					},
					Vm: infrastructurev1beta2.OscVm{
						Name:    "test-vm",
						ImageId: "ami-00000000",
						Role:    "controlplane",
						RootDisk: infrastructurev1beta2.OscRootDisk{
							RootDiskSize: 30,
							RootDiskIops: 1500,
							RootDiskType: "io1",
						},
						KeypairName:   "rke",
						SubregionName: "eu-west-2a",
						SubnetName:    "test-subnet",
						VmType:        "tinav3.c2r4p2",
						SecurityGroupNames: []infrastructurev1beta2.OscSecurityGroupElement{
							{
								Name: "test-securitygroup",
							},
						},
						PrivateIps: []infrastructurev1beta2.OscPrivateIpElement{
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
	awsTypeVmMachineTemplate = infrastructurev1beta2.OscMachineTemplateSpec{
		Template: infrastructurev1beta2.OscMachineTemplateResource{
			Spec: infrastructurev1beta2.OscMachineSpec{
				Node: infrastructurev1beta2.OscNode{
					Vm: infrastructurev1beta2.OscVm{
						VmType: "m4.2xlarge",
					},
				},
			},
		},
	}
)

// Setup set osccluster, oscmachine, machineScope and clusterScope
func SetupMachineTemplate(t *testing.T, name string, machineSpec infrastructurev1beta2.OscMachineTemplateSpec) (machineTemplateScope *scope.MachineTemplateScope) {
	t.Helper()
	t.Logf("Validate to %s", name)

	oscMachineTemplate := infrastructurev1beta2.OscMachineTemplate{
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
		machineTemplateSpec infrastructurev1beta2.OscMachineTemplateSpec
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
