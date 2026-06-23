package v1beta1

import (
	infrastructurev1beta2 "github.com/outscale/cluster-api-provider-outscale/api/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

func (src *OscMachineTemplate) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*infrastructurev1beta2.OscMachineTemplate)
	dst.ObjectMeta = src.ObjectMeta
	dst.Status = infrastructurev1beta2.OscMachineTemplateStatus(src.Status)
	return src.Spec.Template.Spec.ConvertTo(&dst.Spec.Template.Spec)
}

func (dst *OscMachineTemplate) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*infrastructurev1beta2.OscMachineTemplate)
	dst.ObjectMeta = src.ObjectMeta
	dst.Status = OscMachineTemplateStatus(src.Status)
	return dst.Spec.Template.Spec.ConvertFrom(&src.Spec.Template.Spec)
}

var _ conversion.Convertible = (*OscCluster)(nil)
