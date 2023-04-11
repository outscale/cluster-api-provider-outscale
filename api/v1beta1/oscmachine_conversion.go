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
	v1beta2 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta2"
	apiconversion "k8s.io/apimachinery/pkg/conversion"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

func (src *OscMachine) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1beta2.OscMachine)
	dst.ObjectMeta = src.ObjectMeta
	if err := Convert_v1beta1_OscMachine_To_v1beta2_OscMachine(src, dst, nil); err != nil {
		return err
	}

	restored := &v1beta2.OscMachine{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	return nil
}

func (dst *OscMachine) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1beta2.OscMachine)
	dst.ObjectMeta = src.ObjectMeta
	if err := Convert_v1beta2_OscMachine_To_v1beta1_OscMachine(src, dst, nil); err != nil {
		return err
	}

	if err := utilconversion.MarshalData(src, dst); err != nil {
		return err
	}
	return nil
}

func (src *OscMachineList) ConverTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1beta2.OscMachineList)
	return Convert_v1beta1_OscMachineList_To_v1beta2_OscMachineList(src, dst, nil)
}

func (dst *OscMachineList) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1beta2.OscMachineList)
	return Convert_v1beta2_OscMachineList_To_v1beta1_OscMachineList(src, dst, nil)
}

func Convert_v1beta2_OscMachineSpec_to_v1beta1_OscMachineSpec(in *v1beta2.OscMachineSpec, out *OscMachineSpec, s apiconversion.Scope) error {
	return autoConvert_v1beta2_OscMachineSpec_To_v1beta1_OscMachineSpec(in, out, s)
}
