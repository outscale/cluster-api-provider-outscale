package v1beta1

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
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
						VmType:      "tinav4.c2r4p2",
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
			expValidateCreateErr: fmt.Errorf("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: keypairName: Invalid value: \"rke λ\": Invalid KeypairName"),
		},
		{
			name: "create with bad imageId",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Vm: OscVm{
						ImageId: "omi-00000000",
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: imageId: Invalid value: \"omi-00000000\": Invalid imageId"),
		},
		{
			name: "create with bad deviceName",
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Vm: OscVm{
						DeviceName: "/dev/xvaa",
					},
				},
			},
			expValidateCreateErr: fmt.Errorf("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: deviceName: Invalid value: \"/dev/xvaa\": Invalid deviceName"),
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
			expValidateCreateErr: fmt.Errorf("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: vmType: Invalid value: \"oscv4.c2r4p2\": Invalid vmType"),
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
			expValidateCreateErr: fmt.Errorf("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: iops: Invalid value: -15: Invalid iops"),
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
			expValidateCreateErr: fmt.Errorf("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: iops: Invalid value: -15: Invalid iops"),
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
			expValidateCreateErr: fmt.Errorf("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: size: Invalid value: -15: Invalid size"),
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
			expValidateCreateErr: fmt.Errorf("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: diskType: Invalid value: \"gp3\": Invalid volumeType"),
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
			expValidateCreateErr: fmt.Errorf("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: size: Invalid value: -30: Invalid size"),
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
			expValidateCreateErr: fmt.Errorf("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: volumeType: Invalid value: \"gp3\": Invalid volumeType"),
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
			expValidateCreateErr: fmt.Errorf("OscMachine.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: subregionName: Invalid value: \"eu-west-2c\": Invalid subregionName"),
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
						VmType:      "tinav4.c2r4p2",
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
					Vm: OscVm{
						KeypairName: "test-webhook",
						ImageId:     "ami-00000000",
						DeviceName:  "/dev/xvda",
						VmType:      "tinav4.c2r4p2",
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
					Vm: OscVm{
						KeypairName: "test-webhook-1",
						ImageId:     "ami-00000000",
						DeviceName:  "/dev/xvda",
						VmType:      "tinav4.c2r4p2",
					},
				},
			},
			machineSpec: OscMachineSpec{
				Node: OscNode{
					Vm: OscVm{
						KeypairName: "test-webhook-2",
						ImageId:     "ami-00000000",
						DeviceName:  "/dev/xvda",
						VmType:      "tinav4.c2r4p2",
					},
				},
			},
			expValidateUpdateErr: fmt.Errorf("OscMachineTemplate.infrastructure.cluster.x-k8s.io \"webhook-test\" is invalid: OscMachineTemplate.spec.template.spec: Invalid value: v1beta1.OscMachineTemplate{TypeMeta:v1.TypeMeta{Kind:\"\", APIVersion:\"\"}, ObjectMeta:v1.ObjectMeta{Name:\"webhook-test\", GenerateName:\"\", Namespace:\"default\", SelfLink:\"\", UID:\"\", ResourceVersion:\"\", Generation:0, CreationTimestamp:time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC), DeletionTimestamp:<nil>, DeletionGracePeriodSeconds:(*int64)(nil), Labels:map[string]string(nil), Annotations:map[string]string(nil), OwnerReferences:[]v1.OwnerReference(nil), Finalizers:[]string(nil), ClusterName:\"\", ManagedFields:[]v1.ManagedFieldsEntry(nil)}, Spec:v1beta1.OscMachineTemplateSpec{Template:v1beta1.OscMachineTemplateResource{ObjectMeta:v1beta1.ObjectMeta{Labels:map[string]string(nil), Annotations:map[string]string(nil)}, Spec:v1beta1.OscMachineSpec{ProviderID:(*string)(nil), Node:v1beta1.OscNode{Vm:v1beta1.OscVm{Name:\"\", ImageId:\"ami-00000000\", KeypairName:\"test-webhook-2\", VmType:\"tinav4.c2r4p2\", VolumeName:\"\", VolumeDeviceName:\"\", DeviceName:\"/dev/xvda\", SubnetName:\"\", RootDisk:v1beta1.OscRootDisk{RootDiskIops:0, RootDiskSize:0, RootDiskType:\"\"}, LoadBalancerName:\"\", PublicIpName:\"\", SubregionName:\"\", PrivateIps:[]v1beta1.OscPrivateIpElement(nil), SecurityGroupNames:[]v1beta1.OscSecurityGroupElement(nil), ResourceId:\"\", Role:\"\", ClusterName:\"\"}, Image:v1beta1.OscImage{Name:\"\", ResourceId:\"\"}, Volumes:[]*v1beta1.OscVolume(nil), KeyPair:v1beta1.OscKeypair{Name:\"\", PublicKey:\"\", ResourceId:\"\", ClusterName:\"\"}, ClusterName:\"\"}}}}}: OscMachineTemplate spec.template.spec field is immutable."),
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
