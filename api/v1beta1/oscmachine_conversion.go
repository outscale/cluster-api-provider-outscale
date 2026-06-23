package v1beta1

import (
	infrastructurev1beta2 "github.com/outscale/cluster-api-provider-outscale/api/v1beta2"
	"github.com/samber/lo"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

func (src *OscMachineSpec) ConvertTo(dst *infrastructurev1beta2.OscMachineSpec) error {
	srcNode := src.Node
	*dst = infrastructurev1beta2.OscMachineSpec{
		ProviderID: src.ProviderID,
		Node: infrastructurev1beta2.OscNode{
			Vm: infrastructurev1beta2.OscVm{
				Name:           srcNode.Vm.Name,
				ImageId:        srcNode.Vm.ImageId,
				KeypairName:    srcNode.Vm.KeypairName,
				VmType:         srcNode.Vm.VmType,
				SubnetName:     srcNode.Vm.SubnetName,
				PublicIp:       srcNode.Vm.PublicIp,
				RootDisk:       infrastructurev1beta2.OscRootDisk(srcNode.Vm.RootDisk),
				SubregionName:  srcNode.Vm.SubregionName,
				SubregionMode:  infrastructurev1beta2.SubregionMode(srcNode.Vm.SubregionMode),
				SubregionNames: srcNode.Vm.SubregionNames,
				SecurityGroupNames: lo.Map(srcNode.Vm.SecurityGroupNames, func(src OscSecurityGroupElement, _ int) infrastructurev1beta2.OscSecurityGroupElement {
					return infrastructurev1beta2.OscSecurityGroupElement(src)
				}),
				Role:      infrastructurev1beta2.OscRole(srcNode.Vm.Role),
				Tags:      srcNode.Vm.Tags,
				Placement: infrastructurev1beta2.OscPlacement(srcNode.Vm.Placement),
			},
			Volumes: lo.Map(srcNode.Volumes, func(src OscVolume, _ int) infrastructurev1beta2.OscVolume {
				return infrastructurev1beta2.OscVolume{
					Name:         src.Name,
					Device:       src.Device,
					Iops:         src.Iops,
					Size:         src.Size,
					VolumeType:   src.VolumeType,
					FromSnapshot: src.FromSnapshot,
				}
			}),
			Image: infrastructurev1beta2.OscImage{
				Name:               srcNode.Image.Name,
				AccountId:          srcNode.Image.AccountId,
				OutscaleOpenSource: srcNode.Image.OutscaleOpenSource,
			},
		},
	}
	if src.Node.Vm.FGPU != nil {
		dst.Node.Vm.FGPU = new(infrastructurev1beta2.OscFGPU(*src.Node.Vm.FGPU))
	}
	if src.Node.ReconciliationRule != nil {
		dst.Node.ReconciliationRule = &infrastructurev1beta2.OscReconciliationRule{
			AppliesTo: lo.Map(src.Node.ReconciliationRule.AppliesTo, func(src Reconciler, _ int) infrastructurev1beta2.Reconciler {
				return infrastructurev1beta2.Reconciler(src)
			}),
			Mode:                 infrastructurev1beta2.ReconciliationMode(src.Node.ReconciliationRule.Mode),
			ReconciliationChance: src.Node.ReconciliationRule.ReconciliationChance,
		}
	}
	return nil
}

func (dst *OscMachineSpec) ConvertFrom(src *infrastructurev1beta2.OscMachineSpec) error {
	srcNode := src.Node
	*dst = OscMachineSpec{
		ProviderID: src.ProviderID,
		Node: OscNode{
			Vm: OscVm{
				Name:           srcNode.Vm.Name,
				ImageId:        srcNode.Vm.ImageId,
				KeypairName:    srcNode.Vm.KeypairName,
				VmType:         srcNode.Vm.VmType,
				SubnetName:     srcNode.Vm.SubnetName,
				PublicIp:       srcNode.Vm.PublicIp,
				RootDisk:       OscRootDisk(srcNode.Vm.RootDisk),
				SubregionName:  srcNode.Vm.SubregionName,
				SubregionMode:  SubregionMode(srcNode.Vm.SubregionMode),
				SubregionNames: srcNode.Vm.SubregionNames,
				SecurityGroupNames: lo.Map(srcNode.Vm.SecurityGroupNames, func(src infrastructurev1beta2.OscSecurityGroupElement, _ int) OscSecurityGroupElement {
					return OscSecurityGroupElement(src)
				}),
				Role:      OscRole(srcNode.Vm.Role),
				Tags:      srcNode.Vm.Tags,
				Placement: OscPlacement(srcNode.Vm.Placement),
			},
			Volumes: lo.Map(srcNode.Volumes, func(src infrastructurev1beta2.OscVolume, _ int) OscVolume {
				return OscVolume{
					Name:         src.Name,
					Device:       src.Device,
					Iops:         src.Iops,
					Size:         src.Size,
					VolumeType:   src.VolumeType,
					FromSnapshot: src.FromSnapshot,
				}
			}),
			Image: OscImage{
				Name:               srcNode.Image.Name,
				AccountId:          srcNode.Image.AccountId,
				OutscaleOpenSource: srcNode.Image.OutscaleOpenSource,
			},
		},
	}
	if src.Node.Vm.FGPU != nil {
		dst.Node.Vm.FGPU = new(OscFGPU(*src.Node.Vm.FGPU))
	}
	if src.Node.ReconciliationRule != nil {
		dst.Node.ReconciliationRule = &OscReconciliationRule{
			AppliesTo: lo.Map(src.Node.ReconciliationRule.AppliesTo, func(src infrastructurev1beta2.Reconciler, _ int) Reconciler {
				return Reconciler(src)
			}),
			Mode:                 ReconciliationMode(src.Node.ReconciliationRule.Mode),
			ReconciliationChance: src.Node.ReconciliationRule.ReconciliationChance,
		}
	}
	return nil
}

func (src *OscMachine) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*infrastructurev1beta2.OscMachine)
	dst.ObjectMeta = src.ObjectMeta
	dst.Status = infrastructurev1beta2.OscMachineStatus{
		Ready:          src.Status.Ready,
		Addresses:      src.Status.Addresses,
		FailureDomain:  src.Status.FailureDomain,
		FailureReason:  src.Status.FailureReason,
		FailureMessage: src.Status.FailureMessage,
		VmState:        src.Status.VmState,
		Resources:      infrastructurev1beta2.OscMachineResources(src.Status.Resources),
		ReconcilerGeneration: lo.MapEntries(src.Status.ReconcilerGeneration, func(k Reconciler, v int64) (infrastructurev1beta2.Reconciler, int64) {
			return infrastructurev1beta2.Reconciler(k), v
		}),
		Conditions: src.Status.Conditions,
	}
	return src.Spec.ConvertTo(&dst.Spec)
}

func (dst *OscMachine) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*infrastructurev1beta2.OscMachine)
	dst.ObjectMeta = src.ObjectMeta
	dst.Status = OscMachineStatus{
		Ready:          src.Status.Ready,
		Addresses:      src.Status.Addresses,
		FailureDomain:  src.Status.FailureDomain,
		FailureReason:  src.Status.FailureReason,
		FailureMessage: src.Status.FailureMessage,
		VmState:        src.Status.VmState,
		Resources:      OscMachineResources(src.Status.Resources),
		ReconcilerGeneration: lo.MapEntries(src.Status.ReconcilerGeneration, func(k infrastructurev1beta2.Reconciler, v int64) (Reconciler, int64) {
			return Reconciler(k), v
		}),
		Conditions: src.Status.Conditions,
	}
	return dst.Spec.ConvertFrom(&src.Spec)
}

var _ conversion.Convertible = (*OscCluster)(nil)
