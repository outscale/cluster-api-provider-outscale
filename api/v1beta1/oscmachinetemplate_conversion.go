package v1beta1

import (
	infrastructurev1beta2 "github.com/outscale/cluster-api-provider-outscale/api/v1beta2"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

func (src *OscMachineTemplate) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*infrastructurev1beta2.OscMachineTemplate)
	dst.ObjectMeta = src.ObjectMeta
	dst.Status = infrastructurev1beta2.OscMachineTemplateStatus(src.Status)
	dst.Spec.Template.ObjectMeta = src.Spec.Template.ObjectMeta
	if err := src.Spec.Template.Spec.ConvertTo(&dst.Spec.Template.Spec); err != nil {
		return err
	}
	restored := &infrastructurev1beta2.OscMachineTemplate{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	// restore new fields
	if restored.Spec.Template.Spec.ProviderIDScheme != "" {
		dst.Spec.Template.Spec.ProviderIDScheme = restored.Spec.Template.Spec.ProviderIDScheme
	}
	return nil
}

func (dst *OscMachineTemplate) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*infrastructurev1beta2.OscMachineTemplate)
	dst.ObjectMeta = src.ObjectMeta
	dst.Status = OscMachineTemplateStatus(src.Status)
	dst.Spec.Template.ObjectMeta = src.Spec.Template.ObjectMeta
	if err := dst.Spec.Template.Spec.ConvertFrom(&src.Spec.Template.Spec); err != nil {
		return err
	}
	return utilconversion.MarshalData(src, dst)
}

var _ conversion.Convertible = (*OscMachineTemplate)(nil)

func (src *OscMachineTemplateList) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*infrastructurev1beta2.OscMachineTemplateList)
	dst.Items = make([]infrastructurev1beta2.OscMachineTemplate, len(src.Items))
	for i := range src.Items {
		if err := src.Items[i].ConvertTo(&dst.Items[i]); err != nil {
			return err
		}
	}
	return nil
}

func (dst *OscMachineTemplateList) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*infrastructurev1beta2.OscMachineTemplateList)
	dst.Items = make([]OscMachineTemplate, len(src.Items))
	for i := range src.Items {
		if err := dst.Items[i].ConvertFrom(&src.Items[i]); err != nil {
			return err
		}
	}
	return nil
}

var _ conversion.Convertible = (*OscMachineTemplateList)(nil)
