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
	"testing"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestOscMachine_ValidateCreate check good and bad validation of oscMachine spec
func TestOscMachine_ValidateCreate(t *testing.T) {
	machineTestCases := []struct {
		name        string
		machineSpec infrastructurev1beta1.OscMachineSpec
		errorCount  int
	}{
		{
			name: "create Valid Vm Spec",
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						ImageId:     "ami-01234567",
						KeypairName: "test-webhook",
						VmType:      "tinav4.c2r4p2",
					},
				},
			},
		},
		{
			name: "create with bad vmType",
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						KeypairName: "test-webhook",
						VmType:      "oscv4.c2r4p2",
					},
				},
			},
			errorCount: 1,
		},
		{
			name: "create with bad iops",
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						KeypairName: "test-webhook",
						VmType:      "tinav4.c2r4p2",
					},
					Volumes: []infrastructurev1beta1.OscVolume{
						{
							Name:       "test-webhook",
							Device:     "/dev/sdb",
							Iops:       -15,
							Size:       30,
							VolumeType: "io1",
						},
					},
				},
			},
			errorCount: 1,
		},
		{
			name: "create rootdisk with bad iops",
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						KeypairName: "test-webhook",
						VmType:      "tinav4.c2r4p2",
						RootDisk: infrastructurev1beta1.OscRootDisk{
							RootDiskIops: -15,
						},
					},
				},
			},
			errorCount: 1,
		},
		{
			name: "create rootdisk with bad size",
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						KeypairName: "test-webhook",
						VmType:      "tinav4.c2r4p2",
						RootDisk: infrastructurev1beta1.OscRootDisk{
							RootDiskSize: -15,
						},
					},
				},
			},
			errorCount: 1,
		},
		{
			name: "create rootdisk with bad volumeType",
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						KeypairName: "test-webhook",
						VmType:      "tinav4.c2r4p2",
						RootDisk: infrastructurev1beta1.OscRootDisk{
							RootDiskType: "ssd1",
						},
					},
				},
			},
			errorCount: 1,
		},
		{
			name: "create with bad volumeType",
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						KeypairName: "test-webhook",
						VmType:      "tinav4.c2r4p2",
					},
					Volumes: []infrastructurev1beta1.OscVolume{
						{
							Name:       "test-webhook",
							Device:     "/dev/sdb",
							Iops:       20,
							Size:       30,
							VolumeType: "ssd1",
						},
					},
				},
			},
			errorCount: 1,
		},
		{
			name: "create with good io1 volumeSpec",
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						KeypairName: "test-webhook",
						VmType:      "tinav4.c2r4p2",
					},
					Volumes: []infrastructurev1beta1.OscVolume{
						{
							Name:       "test-webhook",
							Device:     "/dev/sdb",
							Iops:       20,
							Size:       20,
							VolumeType: "io1",
						},
					},
				},
			},
			errorCount: 0,
		},
		{
			name: "create with bad io1 ratio size iops",
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						KeypairName: "test-webhook",
						VmType:      "tinav4.c2r4p2",
					},
					Volumes: []infrastructurev1beta1.OscVolume{
						{
							Name:       "test-webhook",
							Device:     "/dev/sdb",
							Iops:       2000,
							Size:       20,
							VolumeType: "io1",
						},
					},
				},
			},
			errorCount: 0,
		},
		{
			name: "create with good gp2 volumeSpec",
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						KeypairName: "test-webhook",
						VmType:      "tinav4.c2r4p2",
					},
					Volumes: []infrastructurev1beta1.OscVolume{
						{
							Name:       "test-webhook",
							Device:     "/dev/sdb",
							Size:       20,
							VolumeType: "gp2",
						},
					},
				},
			},
			errorCount: 0,
		},
		{
			name: "create with good standard volumeSpec",
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						KeypairName: "test-webhook",
						VmType:      "tinav4.c2r4p2",
					},
					Volumes: []infrastructurev1beta1.OscVolume{
						{
							Name:       "test-webhook",
							Device:     "/dev/sdb",
							Size:       20,
							VolumeType: "standard",
						},
					},
				},
			},
			errorCount: 0,
		},
	}
	for _, mtc := range machineTestCases {
		t.Run(mtc.name, func(t *testing.T) {
			oscInfraMachine := createOscInfraMachine(mtc.machineSpec, "webhook-test", "default")
			_, err := oscInfraMachine.ValidateCreate()
			if mtc.errorCount > 0 {
				require.Error(t, err, "ValidateCreate should return the same errror")
				require.Len(t, err.(*apierrors.StatusError).Status().Details.Causes, mtc.errorCount)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestOscMachine_ValidateUpdate check good and bad update of oscMachine
func TestOscMachine_ValidateUpdate(t *testing.T) {
	machineTestCases := []struct {
		name           string
		oldMachineSpec infrastructurev1beta1.OscMachineSpec
		machineSpec    infrastructurev1beta1.OscMachineSpec
		errorCount     int
	}{
		{
			name: "update only oscMachine name",
			oldMachineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						KeypairName: "test-webhook",
						ImageId:     "ami-00000000",
						VmType:      "tinav4.c2r4p2",
					},
					Volumes: []infrastructurev1beta1.OscVolume{
						{
							Name:       "update-webhook",
							Iops:       15,
							Size:       30,
							VolumeType: "io1",
						},
					},
				},
			},
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						KeypairName: "test-webhook",
						ImageId:     "ami-00000000",
						VmType:      "tinav4.c2r4p2",
					},
					Volumes: []infrastructurev1beta1.OscVolume{
						{
							Name:       "update-webhook",
							Iops:       15,
							Size:       30,
							VolumeType: "io1",
						},
					},
				},
			},
			errorCount: 0,
		},
		{
			name: "update keypairName",
			oldMachineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						KeypairName: "test-webhook-1",
					},
				},
			},
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						KeypairName: "test-webhook-2",
					},
				},
			},
			errorCount: 1,
		},
		{
			name: "update vmType",
			oldMachineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						VmType: "tinav4.c2r4p2",
					},
				},
			},
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						VmType: "tinav4.c2r4p1",
					},
				},
			},
			errorCount: 1,
		},
	}
	for _, mtc := range machineTestCases {
		t.Run(mtc.name, func(t *testing.T) {
			oscOldInfraMachine := createOscInfraMachine(mtc.oldMachineSpec, "old-webhook-test", "default")
			oscInfraMachine := createOscInfraMachine(mtc.machineSpec, "webhook-test", "default")
			_, err := oscInfraMachine.ValidateUpdate(oscOldInfraMachine)
			if mtc.errorCount > 0 {
				require.Error(t, err, "ValidateCreate should return the same errror")
				require.Len(t, err.(*apierrors.StatusError).Status().Details.Causes, mtc.errorCount)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// createOscInfraMachine create oscInfraMachine
func createOscInfraMachine(infraMachineSpec infrastructurev1beta1.OscMachineSpec, name string, namespace string) *infrastructurev1beta1.OscMachine {
	oscInfraMachine := &infrastructurev1beta1.OscMachine{
		Spec: infraMachineSpec,
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	return oscInfraMachine
}
