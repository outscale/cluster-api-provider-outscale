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

// TestOscMachineTemplate_ValidateCreate check good and bad update of oscMachineTemplate
func TestOscMachineTemplate_ValidateCreate(t *testing.T) {
	machineTestCases := []struct {
		name                 string
		machineSpec          OscMachineSpec
		expValidateCreateErr error
	}{
		{
			name: "create Valid Vm Spec",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Vm: OscVm{
						ImageId:     "ami-01234567",
						KeypairName: "test-webhook",
						DeviceName:  "/dev/xvdb",
						VmType:      "tinav3.c2r4p2",
						Tags:        map[string]string{"key1": "value1"},
					},
				},
			},
			expValidateCreateErr: nil,
		},
		{
			name: "create with bad keypairName",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Vm: OscVm{
						KeypairName: "rke λ",
					},
				},
			},
			expValidateCreateErr: errors.New("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: node.vm.keypairName: Invalid value: \"rke λ\": invalid keypair name"),
		},
		{
			name: "create with bad vmType",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Vm: OscVm{
						VmType: "oscv4.c2r4p2",
					},
				},
			},
			expValidateCreateErr: errors.New("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: node.vm.vmType: Invalid value: \"oscv4.c2r4p2\": invalid vm type"),
		},
		{
			name: "create rootdisk with bad iops",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Vm: OscVm{
						RootDisk: OscRootDisk{
							RootDiskIops: -15,
						},
					},
				},
			},
			expValidateCreateErr: errors.New("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: node.vm.rootDisk.rootDiskIops: Invalid value: -15: invalid iops"),
		},
		{
			name: "create rootdisk with bad size",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Vm: OscVm{
						RootDisk: OscRootDisk{
							RootDiskSize: -15,
						},
					},
				},
			},
			expValidateCreateErr: errors.New("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: node.vm.rootDisk.rootDiskSize: Invalid value: -15: invalid size"),
		},
		{
			name: "create rootdisk with bad volumeType",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Vm: OscVm{
						RootDisk: OscRootDisk{
							RootDiskType: "gp3",
						},
					},
				},
			},
			expValidateCreateErr: errors.New("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: node.vm.rootDisk.rootDiskType: Invalid value: \"gp3\": invalid volume type (allowed: standard, gp2, io1)"),
		},
		{
			name: "create with bad volume size",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Volumes: []*OscVolume{
						{
							Name:       "test-webhook",
							Device:     "/dev/sdb",
							Iops:       20,
							Size:       -30,
							VolumeType: "io1",
						},
					},
				},
			},
			expValidateCreateErr: errors.New("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: node.volumes.size: Invalid value: -30: invalid size"),
		},
		{
			name: "create with bad volume iops",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Volumes: []*OscVolume{
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
			expValidateCreateErr: errors.New(`OscMachine.infrastructure.cluster.x-k8s.io "webhook-test" is invalid: node.volumes.iops: Invalid value: -15: invalid iops`),
		},
		{
			name: "create with missing volume device",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Volumes: []*OscVolume{
						{
							Name:       "test-webhook",
							Iops:       20,
							Size:       30,
							VolumeType: "io1",
						},
					},
				},
			},
			expValidateCreateErr: errors.New(`OscMachine.infrastructure.cluster.x-k8s.io "webhook-test" is invalid: node.volumes.device: Invalid value: "": device name is required`),
		},
		{
			name: "create with invalid volume device",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Volumes: []*OscVolume{
						{
							Name:       "test-webhook",
							Device:     "foo",
							Iops:       20,
							Size:       30,
							VolumeType: "io1",
						},
					},
				},
			},
			expValidateCreateErr: errors.New(`OscMachine.infrastructure.cluster.x-k8s.io "webhook-test" is invalid: node.volumes.device: Invalid value: "foo": invalid device name`),
		},
		{
			name: "create with bad volumeType",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Volumes: []*OscVolume{
						{
							Name:       "test-webhook",
							Device:     "/dev/sdb",
							Iops:       20,
							Size:       30,
							VolumeType: "gp3",
						},
					},
				},
			},
			expValidateCreateErr: errors.New("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: node.volumes.volumeType: Invalid value: \"gp3\": invalid volume type (allowed: standard, gp2, io1)"),
		},
		{
			name: "create with valid io1 volumeSpec",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Volumes: []*OscVolume{
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
			expValidateCreateErr: nil,
		},
		{
			name: "create with valid gp2 volumeSpec",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Volumes: []*OscVolume{
						{
							Name:       "test-webhook",
							Device:     "/dev/sdb",
							Size:       20,
							VolumeType: "gp2",
						},
					},
				},
			},
			expValidateCreateErr: nil,
		},
		{
			name: "create with valid standard volumeSpec",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Volumes: []*OscVolume{
						{
							Name:       "test-webhook",
							Device:     "/dev/sdb",
							Size:       20,
							VolumeType: "standard",
						},
					},
				},
			},
			expValidateCreateErr: nil,
		},
	}
	for _, mtc := range machineTestCases {
		t.Run(mtc.name, func(t *testing.T) {
			oscInfraMachine := createOscInfraMachineTemplate(mtc.machineSpec, "webhook-test", "default")
			err := oscInfraMachine.ValidateCreate()
			if mtc.expValidateCreateErr != nil {
				require.EqualError(t, err, mtc.expValidateCreateErr.Error(), "ValidateCreate should return the same errror")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestOscMachineTemplate_ValidateUpdate check good and bad update of oscMachineTemplate
func TestOscMachineTemplate_ValidateUpdate(t *testing.T) {
	machineTestCases := []struct {
		name                 string
		oldMachineSpec       OscMachineSpec
		machineSpec          OscMachineSpec
		expValidateUpdateErr error
	}{
		{
			name: "Update only oscMachineTemplate name",
			oldMachineSpec: OscMachineSpec{
				Node: OscNode{
					Vm: OscVm{
						KeypairName: "test-webhook",
						ImageId:     "ami-00000000",
						DeviceName:  "/dev/xvda",
						VmType:      "tinav3.c2r4p2",
						PublicIp:    false,
					},
					Volumes: []*OscVolume{
						{
							Name:       "update-webhook",
							Iops:       15,
							Size:       30,
							VolumeType: "io1",
						},
					},
				},
			},
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Vm: OscVm{
						KeypairName: "test-webhook",
						ImageId:     "ami-00000000",
						DeviceName:  "/dev/xvda",
						VmType:      "tinav3.c2r4p2",
					},
					Volumes: []*OscVolume{
						{
							Name:       "update-webhook",
							Iops:       15,
							Size:       30,
							VolumeType: "io1",
						},
					},
				},
			},
			expValidateUpdateErr: nil,
		},
		{
			name: "update one element (keypair)",
			oldMachineSpec: OscMachineSpec{
				Node: OscNode{
					Vm: OscVm{
						KeypairName: "test-webhook-1",
						ImageId:     "ami-00000000",
						DeviceName:  "/dev/xvda",
						VmType:      "tinav3.c2r4p2",
					},
				},
			},
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Vm: OscVm{
						KeypairName: "test-webhook-2",
						ImageId:     "ami-00000000",
						DeviceName:  "/dev/xvda",
						VmType:      "tinav3.c2r4p2",
					},
				},
			},
			expValidateUpdateErr: errors.New("OscMachineTemplate.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: OscMachineTemplate.spec.template.spec: Invalid value: v1beta1.OscMachineTemplate{TypeMeta:v1.TypeMeta{Kind:\"\", APIVersion:\"\"}, ObjectMeta:v1.ObjectMeta{Name:\"webhook-test\", GenerateName:\"\", Namespace:\"default\", SelfLink:\"\", UID:\"\", ResourceVersion:\"\", Generation:0, CreationTimestamp:time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC), DeletionTimestamp:<nil>, DeletionGracePeriodSeconds:(*int64)(nil), Labels:map[string]string(nil), Annotations:map[string]string(nil), OwnerReferences:[]v1.OwnerReference(nil), Finalizers:[]string(nil), ManagedFields:[]v1.ManagedFieldsEntry(nil)}, Spec:v1beta1.OscMachineTemplateSpec{Template:v1beta1.OscMachineTemplateResource{ObjectMeta:v1beta1.ObjectMeta{Labels:map[string]string(nil), Annotations:map[string]string(nil)}, Spec:v1beta1.OscMachineSpec{ProviderID:(*string)(nil), Node:v1beta1.OscNode{Vm:v1beta1.OscVm{Name:\"\", ImageId:\"ami-00000000\", KeypairName:\"test-webhook-2\", VmType:\"tinav3.c2r4p2\", VolumeName:\"\", VolumeDeviceName:\"\", DeviceName:\"/dev/xvda\", SubnetName:\"\", RootDisk:v1beta1.OscRootDisk{RootDiskIops:0, RootDiskSize:0, RootDiskType:\"\"}, LoadBalancerName:\"\", PublicIpName:\"\", PublicIp:false, SubregionName:\"\", PrivateIps:[]v1beta1.OscPrivateIpElement(nil), SecurityGroupNames:[]v1beta1.OscSecurityGroupElement(nil), ResourceId:\"\", Role:\"\", ClusterName:\"\", Replica:0, Tags:map[string]string(nil)}, Image:v1beta1.OscImage{Name:\"\", ResourceId:\"\"}, Volumes:[]*v1beta1.OscVolume(nil), KeyPair:v1beta1.OscKeypair{Name:\"\", PublicKey:\"\", ResourceId:\"\", ClusterName:\"\", DeleteKeypair:false}, ClusterName:\"\"}}}}, Status:v1beta1.OscMachineTemplateStatus{Capacity:v1.ResourceList(nil), Conditions:v1beta1.Conditions(nil)}}: OscMachineTemplate spec.template.spec field is immutable."),
		},
	}
	for _, mtc := range machineTestCases {
		t.Run(mtc.name, func(t *testing.T) {
			oscOldInfraMachineTemplate := createOscInfraMachineTemplate(mtc.oldMachineSpec, "old-webhook-test", "default")
			oscInfraMachineTemplate := createOscInfraMachineTemplate(mtc.machineSpec, "webhook-test", "default")
			err := oscInfraMachineTemplate.ValidateUpdate(oscOldInfraMachineTemplate)
			if mtc.expValidateUpdateErr != nil {
				require.EqualError(t, err, mtc.expValidateUpdateErr.Error(), "ValidateUpdate() should return the same error")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// createOscInfraMachineTemplate create oscInfraMachineTemplate
func createOscInfraMachineTemplate(infraMachineSpec OscMachineSpec, name string, namespace string) *OscMachineTemplate {
	oscInfraMachineTemplate := &OscMachineTemplate{
		Spec: OscMachineTemplateSpec{
			Template: OscMachineTemplateResource{
				Spec: infraMachineSpec,
			},
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	return oscInfraMachineTemplate
}
