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

// TestOscMachineTemplate_ValidateCreate check good and bad update of oscMachineTemplate.
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
					VM: OscVM{
						ImageID:     "ami-01234567",
						KeypairName: "test-webhook",
						DeviceName:  "/dev/xvdb",
						VMType:      "tinav4.c2r4p2",
					},
				},
			},
			expValidateCreateErr: nil,
		},
		{
			name: "create with bad keypairName",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					VM: OscVM{
						KeypairName: "rke λ",
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: keypairName: Invalid value: \"rke λ\": invalid KeypairName"),
		},
		{
			name: "create with bad imageId",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					VM: OscVM{
						ImageID: "omi-00000000",
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: imageId: Invalid value: \"omi-00000000\": invalid imageId"),
		},
		{
			name: "create with bad deviceName",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					VM: OscVM{
						DeviceName: "/dev/xvaa",
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: deviceName: Invalid value: \"/dev/xvaa\": invalid deviceName"),
		},
		{
			name: "create with bad vmType",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					VM: OscVM{
						VMType: "oscv4.c2r4p2",
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: vmType: Invalid value: \"oscv4.c2r4p2\": invalid vmType"),
		},
		{
			name: "create with bad iops",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Volumes: []*OscVolume{
						{
							Name:          "test-webhook",
							Iops:          -15,
							Size:          30,
							VolumeType:    "io1",
							SubregionName: "eu-weest-2a",
						},
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: iops: Invalid value: -15: invalid iops"),
		},
		{
			name: "create with bad size",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Volumes: []*OscVolume{
						{
							Name:          "test-webhook",
							Iops:          20,
							Size:          -30,
							VolumeType:    "io1",
							SubregionName: "eu-west-2a",
						},
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: size: Invalid value: -30: invalid size"),
		},
		{
			name: "create with bad volumeType",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Volumes: []*OscVolume{
						{
							Name:          "test-webhook",
							Iops:          20,
							Size:          30,
							VolumeType:    "gp3",
							SubregionName: "eu-west-2a",
						},
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: volumeType: Invalid value: \"gp3\": invalid volumeType"),
		},
		{
			name: "create with bad subregionName",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Volumes: []*OscVolume{
						{
							Name:          "test-webhook",
							Iops:          20,
							Size:          30,
							VolumeType:    "io1",
							SubregionName: "eu-west-2c",
						},
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: subregionName: Invalid value: \"eu-west-2c\": invalid subregionName"),
		},
		{
			name: "create with good io1 volumeSpec",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Volumes: []*OscVolume{
						{
							Name:          "test-webhook",
							Iops:          20,
							Size:          20,
							VolumeType:    "io1",
							SubregionName: "eu-west-2a",
						},
					},
				},
			},
			expValidateCreateErr: nil,
		},
		{
			name: "create with good gp2 volumeSpec",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Volumes: []*OscVolume{
						{
							Name:          "test-webhook",
							Size:          20,
							VolumeType:    "gp2",
							SubregionName: "eu-west-2a",
						},
					},
				},
			},
			expValidateCreateErr: nil,
		},
		{
			name: "create with good standard volumeSpec",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Volumes: []*OscVolume{
						{
							Name:          "test-webhook",
							Size:          20,
							VolumeType:    "standard",
							SubregionName: "eu-west-2a",
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
			if err != nil {
				assert.Equal(t, mtc.expValidateCreateErr.Error(), err.Error(), "ValidateCreate should return the same errror")
			} else {
				assert.Nil(t, mtc.expValidateCreateErr)
			}
		})
	}
}

// TestOscMachineTemplate_ValidateUpdate check good and bad update of oscMachineTemplate.
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
					VM: OscVM{
						KeypairName: "test-webhook",
						ImageID:     "ami-00000000",
						DeviceName:  "/dev/xvda",
						VMType:      "tinav4.c2r4p2",
					},
					Volumes: []*OscVolume{
						{
							Name:          "update-webhook",
							Iops:          15,
							Size:          30,
							VolumeType:    "io1",
							SubregionName: "eu-west-2a",
						},
					},
				},
			},
			machineSpec: OscMachineSpec{
				Node: OscNode{
					VM: OscVM{
						KeypairName: "test-webhook",
						ImageID:     "ami-00000000",
						DeviceName:  "/dev/xvda",
						VMType:      "tinav4.c2r4p2",
					},
					Volumes: []*OscVolume{
						{
							Name:          "update-webhook",
							Iops:          15,
							Size:          30,
							VolumeType:    "io1",
							SubregionName: "eu-west-2a",
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
					VM: OscVM{
						KeypairName: "test-webhook-1",
						ImageID:     "ami-00000000",
						DeviceName:  "/dev/xvda",
						VMType:      "tinav4.c2r4p2",
					},
				},
			},
			machineSpec: OscMachineSpec{
				Node: OscNode{
					VM: OscVM{
						KeypairName: "test-webhook-2",
						ImageID:     "ami-00000000",
						DeviceName:  "/dev/xvda",
						VMType:      "tinav4.c2r4p2",
					},
				},
			},
			expValidateUpdateErr: fmt.Errorf("OscMachineTemplate.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: OscMachineTemplate.spec.template.spec: Invalid value: v1beta1.OscMachineTemplate{TypeMeta:v1.TypeMeta{Kind:\"\", APIVersion:\"\"}, ObjectMeta:v1.ObjectMeta{Name:\"webhook-test\", GenerateName:\"\", Namespace:\"default\", SelfLink:\"\", UID:\"\", ResourceVersion:\"\", Generation:0, CreationTimestamp:time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC), DeletionTimestamp:<nil>, DeletionGracePeriodSeconds:(*int64)(nil), Labels:map[string]string(nil), Annotations:map[string]string(nil), OwnerReferences:[]v1.OwnerReference(nil), Finalizers:[]string(nil), ClusterName:\"\", ManagedFields:[]v1.ManagedFieldsEntry(nil)}, Spec:v1beta1.OscMachineTemplateSpec{Template:v1beta1.OscMachineTemplateResource{ObjectMeta:v1beta1.ObjectMeta{Labels:map[string]string(nil), Annotations:map[string]string(nil)}, Spec:v1beta1.OscMachineSpec{ProviderID:(*string)(nil), Node:v1beta1.OscNode{VM:v1beta1.OscVM{Name:\"\", ImageID:\"ami-00000000\", KeypairName:\"test-webhook-2\", VMType:\"tinav4.c2r4p2\", VolumeName:\"\", DeviceName:\"/dev/xvda\", SubnetName:\"\", LoadBalancerName:\"\", PublicIPName:\"\", SubregionName:\"\", PrivateIPS:[]v1beta1.OscPrivateIPElement(nil), SecurityGroupNames:[]v1beta1.OscSecurityGroupElement(nil), ResourceID:\"\", Role:\"\", ClusterName:\"\"}, Image:v1beta1.OscImage{Name:\"\", ResourceID:\"\"}, Volumes:[]*v1beta1.OscVolume(nil), KeyPair:v1beta1.OscKeypair{Name:\"\", PublicKey:\"\", ResourceID:\"\", ClusterName:\"\"}, ClusterName:\"\"}}}}}: OscMachineTemplate spec.template.spec field is immutable."),
		},
	}
	for _, mtc := range machineTestCases {
		t.Run(mtc.name, func(t *testing.T) {
			oscOldInfraMachineTemplate := createOscInfraMachineTemplate(mtc.oldMachineSpec, "old-webhook-test", "default")
			oscInfraMachineTemplate := createOscInfraMachineTemplate(mtc.machineSpec, "webhook-test", "default")
			err := oscInfraMachineTemplate.ValidateUpdate(oscOldInfraMachineTemplate)
			if err != nil {
				assert.Equal(t, mtc.expValidateUpdateErr.Error(), err.Error(), "ValidateUpdate() should return the same error")
			} else {
				assert.Nil(t, mtc.expValidateUpdateErr)
			}
		})
	}
}

// createOscInfraMachineTemplate create oscInfraMachineTemplate.
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
