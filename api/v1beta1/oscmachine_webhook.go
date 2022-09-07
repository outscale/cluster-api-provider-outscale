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
	"reflect"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var oscMachineLog = logf.Log.WithName("oscmachine-resource")

func (m *OscMachine) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(m).Complete()
}

//+kubebuilder:webhook:path=/mutate-infrastructure-cluster-x-k8s-io-v1beta1-oscmachine,mutating=true,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=oscmachines,verbs=create;update,versions=v1beta1,name=moscmachine.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &OscMachine{}

// Default implements webhook.Defaulter so a webhook will be registered for the type.
func (m *OscMachine) Default() {
	oscMachineLog.Info("default", "name", m.Name)
}

//+kubebuilder:webhook:path=/validate-infrastructure-cluster-x-k8s-io-v1beta1-oscmachine,mutating=false,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=oscmachines,verbs=create;update,versions=v1beta1,name=voscmachine.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &OscMachine{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (m *OscMachine) ValidateCreate() error {
	oscMachineLog.Info("validate create", "name", m.Name)
	if allErrs := ValidateOscMachineSpec(m.Spec); len(allErrs) > 0 {
		oscMachineLog.Info("validate error", "error", allErrs)
		return apierrors.NewInvalid(GroupVersion.WithKind("OscMachine").GroupKind(), m.Name, allErrs)
	}
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (m *OscMachine) ValidateUpdate(oldRaw runtime.Object) error {
	oscMachineLog.Info("validate update", "name", m.Name)
	var allErrs field.ErrorList
	old := oldRaw.(*OscMachine)

	oscMachineLog.Info("validate update old imageId", "old imageId", old.Spec.Node.VM.ImageID)
	oscMachineLog.Info("validate update imageId", "imageId", m.Spec.Node.VM.ImageID)
	if !reflect.DeepEqual(m.Spec.Node.VM.ImageID, old.Spec.Node.VM.ImageID) {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "imageId"),
				m.Spec.Node.VM.ImageID, "field is immutable"),
		)
	}

	oscMachineLog.Info("validate update old vmType", "old vmType", old.Spec.Node.VM.VMType)
	oscMachineLog.Info("validate update vmType", "vmType", m.Spec.Node.VM.VMType)

	if !reflect.DeepEqual(m.Spec.Node.VM.VMType, old.Spec.Node.VM.VMType) {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "vmType"),
				m.Spec.Node.VM.VMType, "field is immutable"),
		)
	}

	oscMachineLog.Info("validate update old deviceName", "old deviceName", old.Spec.Node.VM.DeviceName)
	oscMachineLog.Info("validate update deviceName", "deviceName", m.Spec.Node.VM.DeviceName)

	if !reflect.DeepEqual(m.Spec.Node.VM.DeviceName, old.Spec.Node.VM.DeviceName) {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "deviceName"),
				m.Spec.Node.VM.DeviceName, "field is immutable"),
		)
	}

	oscMachineLog.Info("validate update old volumeName", "old volumeName", old.Spec.Node.VM.VolumeName)
	oscMachineLog.Info("validate update volumeName", "volumeName", m.Spec.Node.VM.VolumeName)

	if !reflect.DeepEqual(m.Spec.Node.VM.VolumeName, old.Spec.Node.VM.VolumeName) {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "volumeName"),
				m.Spec.Node.VM.VolumeName, "field is immutable"),
		)
	}

	oscMachineLog.Info("validate update old keypairName", "old keypairName", old.Spec.Node.VM.KeypairName)
	oscMachineLog.Info("validate update keyPairName", "keypairName", m.Spec.Node.VM.KeypairName)

	if !reflect.DeepEqual(m.Spec.Node.VM.KeypairName, old.Spec.Node.VM.KeypairName) {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "keyPairName"),
				m.Spec.Node.VM.KeypairName, "field is immutable"),
		)
	}

	oscMachineLog.Info("validate update old subnetName", "old subnetName", old.Spec.Node.VM.SubnetName)
	oscMachineLog.Info("validate update subnetName", "subnetName", m.Spec.Node.VM.SubnetName)

	if !reflect.DeepEqual(m.Spec.Node.VM.SubnetName, old.Spec.Node.VM.SubnetName) {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "subnetName"),
				m.Spec.Node.VM.SubnetName, "field is immutable"),
		)
	}

	oscMachineLog.Info("validate update old loadBalancerName", "old loadBalancerName", old.Spec.Node.VM.LoadBalancerName)
	oscMachineLog.Info("validate update loadBalancerName", "loadBalancerName", m.Spec.Node.VM.LoadBalancerName)

	if !reflect.DeepEqual(m.Spec.Node.VM.LoadBalancerName, old.Spec.Node.VM.LoadBalancerName) {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "loadBalancerName"),
				m.Spec.Node.VM.LoadBalancerName, "field is immutable"),
		)
	}

	oscMachineLog.Info("validate update old subregionName", "old subregionName", old.Spec.Node.VM.SubregionName)
	oscMachineLog.Info("validate update subregionName", "subregionName", m.Spec.Node.VM.SubregionName)

	if !reflect.DeepEqual(m.Spec.Node.VM.SubregionName, old.Spec.Node.VM.SubregionName) {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "subregionName"),
				m.Spec.Node.VM.SubregionName, "field is immutable"),
		)
	}

	oscMachineLog.Info("validate update old privateIps", "old VMPrivateIPS", old.Spec.Node.VM.PrivateIPS)
	oscMachineLog.Info("validate update privateIps", "VMPrivateIPS", m.Spec.Node.VM.PrivateIPS)

	if !reflect.DeepEqual(m.Spec.Node.VM.PrivateIPS, old.Spec.Node.VM.PrivateIPS) {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "privateIps"),
				m.Spec.Node.VM.PrivateIPS, "field is immutable"),
		)
	}

	oscMachineLog.Info("validate update old securityGroupNames", "old securityGroupNames", old.Spec.Node.VM.SecurityGroupNames)
	oscMachineLog.Info("validate update securityGroupNames", "securityGroupNames", m.Spec.Node.VM.SecurityGroupNames)

	if !reflect.DeepEqual(m.Spec.Node.VM.SecurityGroupNames, old.Spec.Node.VM.SecurityGroupNames) {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "securityGroupNames"),
				m.Spec.Node.VM.SecurityGroupNames, "field is immutable"),
		)
	}
	volumesSpec := m.Spec.Node.Volumes
	oldVolumesSpec := old.Spec.Node.Volumes
	for _, volumeSpec := range volumesSpec {
		for _, oldVolumeSpec := range oldVolumesSpec {
			oscMachineLog.Info("validate update old volumeIops", "old volumeIops", oldVolumeSpec.Iops)
			oscMachineLog.Info("validate update volumeIops", "volumeIops", volumeSpec.Iops)

			if !reflect.DeepEqual(volumeSpec.Iops, oldVolumeSpec.Iops) {
				allErrs = append(allErrs,
					field.Invalid(field.NewPath("spec", "iops"),
						volumeSpec.Iops, "field is immutable"),
				)
			}

			oscMachineLog.Info("validate update old volumeSize", "old volumeSize", oldVolumeSpec.Size)
			oscMachineLog.Info("validate update volumeSize", "volumeSize", volumeSpec.Size)

			if !reflect.DeepEqual(volumeSpec.Size, oldVolumeSpec.Size) {
				allErrs = append(allErrs,
					field.Invalid(field.NewPath("spec", "size"),
						volumeSpec.Size, "field is immutable"),
				)
			}

			oscMachineLog.Info("validate update old subregionName", "old subregionName", oldVolumeSpec.SubregionName)
			oscMachineLog.Info("validate update subregionName", "subregionName", volumeSpec.SubregionName)

			if !reflect.DeepEqual(volumeSpec.SubregionName, oldVolumeSpec.SubregionName) {
				allErrs = append(allErrs,
					field.Invalid(field.NewPath("spec", "subregionName"),
						volumeSpec.SubregionName, "field is immutable"),
				)
			}

			oscMachineLog.Info("validate update old volumeType", "old volumeType", oldVolumeSpec.VolumeType)
			oscMachineLog.Info("validate update volumeType", "volumeType", volumeSpec.VolumeType)

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

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (m *OscMachine) ValidateDelete() error {
	oscMachineLog.Info("validate delete", "name", m.Name)

	return nil
}
