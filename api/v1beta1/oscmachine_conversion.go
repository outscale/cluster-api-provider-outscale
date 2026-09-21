package v1beta1

import (
	"github.com/google/go-cmp/cmp"
	infrastructurev1beta2 "github.com/outscale/cluster-api-provider-outscale/api/v1beta2"
	"github.com/outscale/osc-sdk-go/v3/pkg/osc"
	"github.com/samber/lo"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

func (src *OscMachineSpec) ConvertTo(dst *infrastructurev1beta2.OscMachineSpec) error {
	srcNode := src.Node
	*dst = infrastructurev1beta2.OscMachineSpec{
		ProviderID: src.ProviderID,

		ProviderIDScheme: infrastructurev1beta2.SchemeAWS,

		Vm: infrastructurev1beta2.OscVm{
			Name:          srcNode.Vm.Name,
			Keypair:       srcNode.Vm.KeypairName,
			Type:          srcNode.Vm.VmType,
			SubnetName:    srcNode.Vm.SubnetName,
			PublicIp:      srcNode.Vm.PublicIp,
			PublicIpPool:  srcNode.Vm.PublicIpPool,
			Subregions:    lo.Map(srcNode.Vm.SubregionNames, func(s string, _ int) infrastructurev1beta2.OscSubRegion { return infrastructurev1beta2.OscSubRegion(s) }),
			SubregionMode: infrastructurev1beta2.SubregionMode(srcNode.Vm.SubregionMode),
			SecurityGroupNames: lo.Map(srcNode.Vm.SecurityGroupNames, func(src OscSecurityGroupElement, _ int) infrastructurev1beta2.OscSecurityGroupElement {
				return infrastructurev1beta2.OscSecurityGroupElement(src)
			}),
			Role:      infrastructurev1beta2.OscRole(srcNode.Vm.Role),
			Tags:      srcNode.Vm.Tags,
			Placement: infrastructurev1beta2.OscPlacement(srcNode.Vm.Placement),
		},
		Image: infrastructurev1beta2.OscImage{
			ID:                 srcNode.Vm.ImageId,
			Name:               srcNode.Image.Name,
			AccountID:          srcNode.Image.AccountId,
			OutscaleOpenSource: srcNode.Image.OutscaleOpenSource,
		},
	}
	if len(srcNode.Vm.SubregionNames) == 0 && srcNode.Vm.SubregionName != "" {
		dst.Vm.Subregions = []infrastructurev1beta2.OscSubRegion{infrastructurev1beta2.OscSubRegion(srcNode.Vm.SubregionName)}
	}
	vols := make([]infrastructurev1beta2.OscVolume, 0, len(srcNode.Volumes)+1)
	if !cmp.Equal(srcNode.Vm.RootDisk, OscRootDisk{}) {
		vols = append(vols, infrastructurev1beta2.OscVolume{
			Root: true,
			Iops: srcNode.Vm.RootDisk.RootDiskIops,
			Size: srcNode.Vm.RootDisk.RootDiskSize,
			Type: infrastructurev1beta2.OscVolumeType(srcNode.Vm.RootDisk.RootDiskType),
		})
	}
	vols = append(vols, lo.Map(src.Node.Volumes, func(src OscVolume, _ int) infrastructurev1beta2.OscVolume {
		return infrastructurev1beta2.OscVolume{
			Name:         src.Name,
			Device:       src.Device,
			Iops:         src.Iops,
			Size:         src.Size,
			Type:         infrastructurev1beta2.OscVolumeType(src.VolumeType),
			FromSnapshot: src.FromSnapshot,
		}
	})...)
	dst.Volumes = vols
	if src.Node.Vm.FGPU != nil {
		dst.Vm.FGPU = new(infrastructurev1beta2.OscFGPU(*src.Node.Vm.FGPU))
	}
	if src.Node.ReconciliationRule != nil {
		dst.ReconciliationRule = &infrastructurev1beta2.OscReconciliationRule{
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
	*dst = OscMachineSpec{
		ProviderID: src.ProviderID,
		Node: OscNode{
			Vm: OscVm{
				Name:          src.Vm.Name,
				ImageId:       src.Image.ID,
				KeypairName:   src.Vm.Keypair,
				VmType:        src.Vm.Type,
				SubnetName:    src.Vm.SubnetName,
				PublicIp:      src.Vm.PublicIp,
				PublicIpPool:  src.Vm.PublicIpPool,
				SubregionMode: SubregionMode(src.Vm.SubregionMode),
				SecurityGroupNames: lo.Map(src.Vm.SecurityGroupNames, func(src infrastructurev1beta2.OscSecurityGroupElement, _ int) OscSecurityGroupElement {
					return OscSecurityGroupElement(src)
				}),
				Role:      OscRole(src.Vm.Role),
				Tags:      src.Vm.Tags,
				Placement: OscPlacement(src.Vm.Placement),
			},
			Volumes: lo.FilterMap(src.Volumes, func(src infrastructurev1beta2.OscVolume, _ int) (OscVolume, bool) {
				if infrastructurev1beta2.IsRootVolume(src) {
					return OscVolume{}, false
				}
				return OscVolume{
					Name:         src.Name,
					Device:       src.Device,
					Iops:         src.Iops,
					Size:         src.Size,
					VolumeType:   osc.VolumeType(src.Type),
					FromSnapshot: src.FromSnapshot,
				}, true
			}),
			Image: OscImage{
				Name:               src.Image.Name,
				AccountId:          src.Image.AccountID,
				OutscaleOpenSource: src.Image.OutscaleOpenSource,
			},
		},
	}
	if len(src.Vm.Subregions) == 1 {
		dst.Node.Vm.SubregionName = string(src.Vm.Subregions[0])
	} else {
		dst.Node.Vm.SubregionNames = lo.Map(src.Vm.Subregions, func(s infrastructurev1beta2.OscSubRegion, _ int) string { return string(s) })
	}
	if root, found := lo.Find(src.Volumes, infrastructurev1beta2.IsRootVolume); found {
		dst.Node.Vm.RootDisk = OscRootDisk{
			RootDiskIops: root.Iops,
			RootDiskSize: root.Size,
			RootDiskType: osc.VolumeType(root.Type),
		}
	}
	if src.Vm.FGPU != nil {
		dst.Node.Vm.FGPU = new(OscFGPU(*src.Vm.FGPU))
	}
	if src.ReconciliationRule != nil {
		dst.Node.ReconciliationRule = &OscReconciliationRule{
			AppliesTo: lo.Map(src.ReconciliationRule.AppliesTo, func(src infrastructurev1beta2.Reconciler, _ int) Reconciler {
				return Reconciler(src)
			}),
			Mode:                 ReconciliationMode(src.ReconciliationRule.Mode),
			ReconciliationChance: src.ReconciliationRule.ReconciliationChance,
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
	if err := src.Spec.ConvertTo(&dst.Spec); err != nil {
		return err
	}
	restored := &infrastructurev1beta2.OscMachine{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// restore new fields
	dst.Spec.ProviderIDScheme = restored.Spec.ProviderIDScheme
	return nil
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
	if err := dst.Spec.ConvertFrom(&src.Spec); err != nil {
		return err
	}
	return utilconversion.MarshalData(src, dst)
}

var _ conversion.Convertible = (*OscMachine)(nil)

func (src *OscMachineList) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*infrastructurev1beta2.OscMachineList)
	dst.Items = make([]infrastructurev1beta2.OscMachine, len(src.Items))
	for i := range src.Items {
		if err := src.Items[i].ConvertTo(&dst.Items[i]); err != nil {
			return err
		}
	}
	return nil
}

func (dst *OscMachineList) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*infrastructurev1beta2.OscMachineList)
	dst.Items = make([]OscMachine, len(src.Items))
	for i := range src.Items {
		if err := dst.Items[i].ConvertFrom(&src.Items[i]); err != nil {
			return err
		}
	}
	return nil
}

var _ conversion.Convertible = (*OscMachineList)(nil)
