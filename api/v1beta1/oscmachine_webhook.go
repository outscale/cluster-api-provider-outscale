/*
Copyright 2022.

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
	"reflect"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var oscmachinelog = logf.Log.WithName("oscmachine-resource")

func (r *OscMachine) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-infrastructure-cluster-x-k8s-io-v1beta1-oscmachine,mutating=true,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=oscmachines,verbs=create;update,versions=v1beta1,name=moscmachine.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &OscMachine{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (m *OscMachine) Default() {
	oscmachinelog.Info("default", "name", m.Name)

}

//+kubebuilder:webhook:path=/validate-infrastructure-cluster-x-k8s-io-v1beta1-oscmachine,mutating=false,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=oscmachines,verbs=create;update,versions=v1beta1,name=voscmachine.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &OscMachine{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (m *OscMachine) ValidateCreate() error {
	oscmachinelog.Info("validate create", "name", m.Name)
	if allErrs := ValidateOscMachineSpec(m.Spec); len(allErrs) > 0 {
                oscmachinelog.Info("validate error", "error", allErrs)
                return apierrors.NewInvalid(GroupVersion.WithKind("OscMachine").GroupKind(), m.Name, allErrs)
	}
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (m *OscMachine) ValidateUpdate(oldRaw runtime.Object) error {
	oscmachinelog.Info("validate update", "name", m.Name)
	var allErrs field.ErrorList
	old := oldRaw.(*OscMachine)

	oscmachinelog.Info("validate update old imageId", "old imageId", old.Spec.Node.Vm.ImageId)
	oscmachinelog.Info("validate update imageId", "imageId", m.Spec.Node.Vm.ImageId)
	if !reflect.DeepEqual(m.Spec.Node.Vm.ImageId, old.Spec.Node.Vm.ImageId) {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "imageId"),
				m.Spec.Node.Vm.ImageId, "field is immutable"),
		)
	}

	oscmachinelog.Info("validate update old vmType", "old vmType", old.Spec.Node.Vm.VmType)
	oscmachinelog.Info("validate update vmType", "vmType", m.Spec.Node.Vm.VmType)

	if !reflect.DeepEqual(m.Spec.Node.Vm.VmType, old.Spec.Node.Vm.VmType) {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "vmType"),
				m.Spec.Node.Vm.VmType, "field is immutable"),
		)
	}

	oscmachinelog.Info("validate update old deviceName", "old deviceName", old.Spec.Node.Vm.DeviceName)
	oscmachinelog.Info("validate update deviceName", "deviceName", m.Spec.Node.Vm.DeviceName)

	if !reflect.DeepEqual(m.Spec.Node.Vm.DeviceName, old.Spec.Node.Vm.DeviceName) {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "deviceName"),
				m.Spec.Node.Vm.DeviceName, "field is immutable"),
		)
	}

	oscmachinelog.Info("validate update old volumeName", "old volumeName", old.Spec.Node.Vm.VolumeName)
	oscmachinelog.Info("validate update volumeName", "volumeName", m.Spec.Node.Vm.VolumeName)

	if !reflect.DeepEqual(m.Spec.Node.Vm.VolumeName, old.Spec.Node.Vm.VolumeName) {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "volumeName"),
				m.Spec.Node.Vm.VolumeName, "field is immutable"),
		)
	}

	oscmachinelog.Info("validate update old keypairName", "old keypairName", old.Spec.Node.Vm.KeypairName)
	oscmachinelog.Info("validate update keyPairName", "keypairName", m.Spec.Node.Vm.KeypairName)

	if !reflect.DeepEqual(m.Spec.Node.Vm.KeypairName, old.Spec.Node.Vm.KeypairName) {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "keyPairName"),
				m.Spec.Node.Vm.KeypairName, "field is immutable"),
		)
	}

	oscmachinelog.Info("validate update old subnetName", "old subnetName", old.Spec.Node.Vm.SubnetName)
	oscmachinelog.Info("validate update subnetName", "subnetName", m.Spec.Node.Vm.SubnetName)

	if !reflect.DeepEqual(m.Spec.Node.Vm.SubnetName, old.Spec.Node.Vm.SubnetName) {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "subnetName"),
				m.Spec.Node.Vm.SubnetName, "field is immutable"),
		)
	}


	oscmachinelog.Info("validate update old loadBalancerName", "old loadBalancerName", old.Spec.Node.Vm.LoadBalancerName)
	oscmachinelog.Info("validate update loadBalancerName", "loadBalancerName", m.Spec.Node.Vm.LoadBalancerName)
 
	if !reflect.DeepEqual(m.Spec.Node.Vm.LoadBalancerName, old.Spec.Node.Vm.LoadBalancerName) {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "loadBalancerName"),
				m.Spec.Node.Vm.LoadBalancerName, "field is immutable"),
		)
	}

	oscmachinelog.Info("validate update old subregionName", "old subregionName", old.Spec.Node.Vm.SubregionName)
	oscmachinelog.Info("validate update subregionName", "subregionName", m.Spec.Node.Vm.SubregionName)

	if !reflect.DeepEqual(m.Spec.Node.Vm.SubregionName, old.Spec.Node.Vm.SubregionName) {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "subregionName"),
				m.Spec.Node.Vm.SubregionName, "field is immutable"),
		)
	}

	oscmachinelog.Info("validate update old privateIps", "old vmPrivateIps", old.Spec.Node.Vm.PrivateIps)
	oscmachinelog.Info("validate update privateIps", "vmPrivateIps", m.Spec.Node.Vm.PrivateIps)

	if !reflect.DeepEqual(m.Spec.Node.Vm.PrivateIps, old.Spec.Node.Vm.PrivateIps) {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "privateIps"),
				m.Spec.Node.Vm.PrivateIps, "field is immutable"),
		)
	}

	oscmachinelog.Info("validate update old securityGroupNames", "old securityGroupNames", old.Spec.Node.Vm.SecurityGroupNames)
	oscmachinelog.Info("validate update securityGroupNames", "securityGroupNames", m.Spec.Node.Vm.SecurityGroupNames)

	if !reflect.DeepEqual(m.Spec.Node.Vm.SecurityGroupNames, old.Spec.Node.Vm.SecurityGroupNames) {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "securityGroupNames"),
				m.Spec.Node.Vm.SecurityGroupNames, "field is immutable"),
		)
	}
	volumesSpec := m.Spec.Node.Volumes
	oldVolumesSpec := old.Spec.Node.Volumes
	for _, volumeSpec := range volumesSpec {
		for _, oldVolumeSpec := range oldVolumesSpec {

			oscmachinelog.Info("validate update old volumeIops", "old volumeIops", oldVolumeSpec.Iops)
			oscmachinelog.Info("validate update volumeIops", "volumeIops", volumeSpec.Iops)

			if !reflect.DeepEqual(volumeSpec.Iops, oldVolumeSpec.Iops) {
				allErrs = append(allErrs,
					field.Invalid(field.NewPath("spec", "iops"),
						volumeSpec.Iops, "field is immutable"),
				)
			}
			
			oscmachinelog.Info("validate update old volumeSize", "old volumeSize", oldVolumeSpec.Size)
			oscmachinelog.Info("validate update volumeSize", "volumeSize", volumeSpec.Size)
			
			if !reflect.DeepEqual(volumeSpec.Size, oldVolumeSpec.Size) {
				allErrs = append(allErrs,
					field.Invalid(field.NewPath("spec", "size"),
						volumeSpec.Size, "field is immutable"),
				)
			}

			oscmachinelog.Info("validate update old subregionName", "old subregionName", oldVolumeSpec.SubregionName)
			oscmachinelog.Info("validate update subregionName", "subregionName", volumeSpec.SubregionName)

			if !reflect.DeepEqual(volumeSpec.SubregionName, oldVolumeSpec.SubregionName) {
				allErrs = append(allErrs,
					field.Invalid(field.NewPath("spec", "subregionName"),
						volumeSpec.SubregionName, "field is immutable"),
				)
			}

			oscmachinelog.Info("validate update old volumeType", "old volumeType", oldVolumeSpec.VolumeType)
			oscmachinelog.Info("validate update volumeType", "volumeType", volumeSpec.VolumeType)

			if !reflect.DeepEqual(volumeSpec.VolumeType, oldVolumeSpec.VolumeType) {
				allErrs = append(allErrs,
					field.Invalid(field.NewPath("spec", "volumeType"),
						volumeSpec.VolumeType, "field is immutable"),
				)
			}
		}
	}
	if len(allErrs) == 0 {
		return nil
	}
	return apierrors.NewInvalid(GroupVersion.WithKind("OscMachine").GroupKind(), m.Name, allErrs)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (m *OscMachine) ValidateDelete() error {
	oscmachinelog.Info("validate delete", "name", m.Name)

	return nil
}
