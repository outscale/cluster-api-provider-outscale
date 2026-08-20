package v1beta1

import (
	infrastructurev1beta2 "github.com/outscale/cluster-api-provider-outscale/api/v1beta2"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

func (src *OscClusterTemplate) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*infrastructurev1beta2.OscClusterTemplate)
	dst.ObjectMeta = src.ObjectMeta
	dst.Spec.Template.ObjectMeta = src.Spec.Template.ObjectMeta
	return src.Spec.Template.Spec.ConvertTo(&dst.Spec.Template.Spec)
}

func (dst *OscClusterTemplate) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*infrastructurev1beta2.OscClusterTemplate)
	dst.ObjectMeta = src.ObjectMeta
	dst.Spec.Template.ObjectMeta = src.Spec.Template.ObjectMeta
	return dst.Spec.Template.Spec.ConvertFrom(&src.Spec.Template.Spec)
}

var _ conversion.Convertible = (*OscCluster)(nil)

func (src *OscClusterTemplateList) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*infrastructurev1beta2.OscClusterTemplateList)
	dst.Items = make([]infrastructurev1beta2.OscClusterTemplate, len(src.Items))
	for i := range src.Items {
		if err := src.Items[i].ConvertTo(&dst.Items[i]); err != nil {
			return err
		}
	}
	return nil
}

func (dst *OscClusterTemplateList) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*infrastructurev1beta2.OscClusterTemplateList)
	dst.Items = make([]OscClusterTemplate, len(src.Items))
	for i := range src.Items {
		if err := dst.Items[i].ConvertFrom(&src.Items[i]); err != nil {
			return err
		}
	}
	return nil
}

var _ conversion.Convertible = (*OscClusterTemplateList)(nil)
