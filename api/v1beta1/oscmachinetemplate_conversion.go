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
	infrav1beta2 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta2"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

func (src *OscMachineTemplate) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*infrav1beta2.OscMachineTemplate)
	if err := autoConvert_v1beta1_OscMachineTemplate_To_v1beta2_OscMachineTemplate(src, dst, nil); err != nil {
		return err
	}

	restored := &infrav1beta2.OscMachineTemplate{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	return nil
}

func (dst *OscMachineTemplate) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*infrav1beta2.OscMachineTemplate)
	if err := Convert_v1beta2_OscMachineTemplate_To_v1beta1_OscMachineTemplate(src, dst, nil); err != nil {
		return err
	}
	return nil
}

func (src *OscMachineTemplateList) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*infrav1beta2.OscMachineTemplateList)
	return Convert_v1beta1_OscMachineTemplateList_To_v1beta2_OscMachineTemplateList(src, dst, nil)
}

func (dst *OscMachineTemplateList) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*infrav1beta2.OscMachineTemplateList)
	return autoConvert_v1beta2_OscMachineTemplateList_To_v1beta1_OscMachineTemplateList(src, dst, nil)
}
