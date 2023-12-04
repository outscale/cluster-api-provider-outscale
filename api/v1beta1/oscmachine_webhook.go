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
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var oscMachineLog = logf.Log.WithName("oscmachine-resource")

func (r *OscMachine) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(r).Complete()
}

//+kubebuilder:webhook:path=/mutate-infrastructure-cluster-x-k8s-io-v1beta1-oscmachine,mutating=true,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=oscmachines,verbs=create;update,versions=v1beta1,name=moscmachine.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &OscMachine{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (m *OscMachine) Default() {
	oscMachineLog.Info("default", "name", m.Name)

}

//+kubebuilder:webhook:path=/validate-infrastructure-cluster-x-k8s-io-v1beta1-oscmachine,mutating=false,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=oscmachines,verbs=create;update,versions=v1beta1,name=voscmachine.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &OscMachine{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (m *OscMachine) ValidateCreate() error {
	oscMachineLog.Info("validate create", "name", m.Name)
	if allErrs := ValidateOscMachineSpec(m.Spec); len(allErrs) > 0 {
		oscMachineLog.Info("validate error", "error", allErrs)
		return apierrors.NewInvalid(GroupVersion.WithKind("OscMachine").GroupKind(), m.Name, allErrs)
	}
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (m *OscMachine) ValidateUpdate(oldRaw runtime.Object) error {
	oscMachineLog.Info("validate update", "name", m.Name)
	var allErrs field.ErrorList
	old := oldRaw.(*OscMachine)

	oscMachineLog.Info("validate update old vmType", "old vmType", old.Spec.Node.Vm.VmType)
	oscMachineLog.Info("validate update vmType", "vmType", m.Spec.Node.Vm.VmType)

	if !reflect.DeepEqual(m.Spec.Node.Vm.VmType, old.Spec.Node.Vm.VmType) {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "vmType"),
				m.Spec.Node.Vm.VmType, "field is immutable"),
		)
	}

	oscMachineLog.Info("validate update old keypairName", "old keypairName", old.Spec.Node.Vm.KeypairName)
	oscMachineLog.Info("validate update keyPairName", "keypairName", m.Spec.Node.Vm.KeypairName)

	if !reflect.DeepEqual(m.Spec.Node.Vm.KeypairName, old.Spec.Node.Vm.KeypairName) {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "keyPairName"),
				m.Spec.Node.Vm.KeypairName, "field is immutable"),
		)
	}

	oscMachineLog.Info("validate update old loadBalancerName", "old loadBalancerName", old.Spec.Node.Vm.LoadBalancerName)
	oscMachineLog.Info("validate update loadBalancerName", "loadBalancerName", m.Spec.Node.Vm.LoadBalancerName)

	if !reflect.DeepEqual(m.Spec.Node.Vm.LoadBalancerName, old.Spec.Node.Vm.LoadBalancerName) {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "loadBalancerName"),
				m.Spec.Node.Vm.LoadBalancerName, "field is immutable"),
		)
	}

	oscMachineLog.Info("validate update old subregionName", "old subregionName", old.Spec.Node.Vm.SubregionName)
	oscMachineLog.Info("validate update subregionName", "subregionName", m.Spec.Node.Vm.SubregionName)

	if old.Spec.Node.Vm.SubregionName != "" && !reflect.DeepEqual(m.Spec.Node.Vm.SubregionName, old.Spec.Node.Vm.SubregionName) {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "subregionName"),
				m.Spec.Node.Vm.SubregionName, "field is immutable"),
		)
	}

	oscMachineLog.Info("validate update old subnetName", "old subnetName", old.Spec.Node.Vm.SubnetName)
	oscMachineLog.Info("validate update subnetName", "subnetName", m.Spec.Node.Vm.SubnetName)

	if (old.Spec.Node.Vm.SubnetName != "") && !reflect.DeepEqual(m.Spec.Node.Vm.SubnetName, old.Spec.Node.Vm.SubnetName) {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "subnetName"),
				m.Spec.Node.Vm.SubnetName, "field is immutable"),
		)
	}
	oscMachineLog.Info("validate update old rootDiskSize", "old rootDiskSize", old.Spec.Node.Vm.RootDisk.RootDiskSize)
	oscMachineLog.Info("validate update rootDiskSize", "rootDiskSize", m.Spec.Node.Vm.RootDisk.RootDiskSize)

	if !reflect.DeepEqual(m.Spec.Node.Vm.RootDisk.RootDiskSize, old.Spec.Node.Vm.RootDisk.RootDiskSize) {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "rootDiskSize"),
				m.Spec.Node.Vm.RootDisk.RootDiskSize, "field is immutable"),
		)
	}

	oscMachineLog.Info("validate update of old rootDiskIops", "old rootDiskIops", old.Spec.Node.Vm.RootDisk.RootDiskIops)
	oscMachineLog.Info("validate update rootDiskIops", "old rootDiskIops", m.Spec.Node.Vm.RootDisk.RootDiskIops)
	if !reflect.DeepEqual(m.Spec.Node.Vm.RootDisk.RootDiskIops, old.Spec.Node.Vm.RootDisk.RootDiskIops) {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "rootDiskIops"),
				m.Spec.Node.Vm.RootDisk.RootDiskIops, "field is immutable"),
		)
	}

	oscMachineLog.Info("validate update of old rootDiskTyp", "old rootDisktype", old.Spec.Node.Vm.RootDisk.RootDiskType)
	oscMachineLog.Info("validate update rootDiskType", "old rootDiskType", m.Spec.Node.Vm.RootDisk.RootDiskType)
	if !reflect.DeepEqual(m.Spec.Node.Vm.RootDisk.RootDiskType, old.Spec.Node.Vm.RootDisk.RootDiskType) {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "rootDiskTyp"),
				m.Spec.Node.Vm.RootDisk.RootDiskType, "field is immutable"),
		)
	}

	if len(allErrs) == 0 {
		return nil
	}
	return apierrors.NewInvalid(GroupVersion.WithKind("OscMachine").GroupKind(), m.Name, allErrs)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (m *OscMachine) ValidateDelete() error {
	oscMachineLog.Info("validate delete", "name", m.Name)

	return nil
}
