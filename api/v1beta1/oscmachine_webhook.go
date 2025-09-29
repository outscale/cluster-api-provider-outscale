/*
SPDX-FileCopyrightText: 2022 The Kubernetes Authors

SPDX-License-Identifier: Apache-2.0
*/

package v1beta1

import (
	"context"
	"fmt"
	"maps"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// SetupWebhookWithManager sets up the controller with the Manager.
func (r *OscMachine) SetupWebhookWithManager(mgr ctrl.Manager) error {
	h := OscMachineWebhook{}
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).WithValidator(h).WithDefaulter(h).
		Complete()
}

// OscMachineWebhook is the validation/mutation webhook.
type OscMachineWebhook struct{}

//+kubebuilder:webhook:path=/mutate-infrastructure-cluster-x-k8s-io-v1beta1-oscmachine,mutating=true,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=oscmachines,verbs=create;update,versions=v1beta1,name=moscmachine.kb.io,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = OscMachineWebhook{}

// Default implements webhook.CustomDefaulter.
func (OscMachineWebhook) Default(_ context.Context, _ runtime.Object) error {
	return nil
}

//+kubebuilder:webhook:path=/validate-infrastructure-cluster-x-k8s-io-v1beta1-oscmachine,mutating=false,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=oscmachines,verbs=create;update,versions=v1beta1,name=voscmachine.kb.io,admissionReviewVersions=v1

var _ webhook.CustomValidator = OscMachineWebhook{}

// ValidateCreate implements webhook.CustomValidator.
func (OscMachineWebhook) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	r, ok := obj.(*OscMachine)
	if !ok {
		return nil, fmt.Errorf("expected an OscMachine object but got %T", r)
	}
	if allErrs := ValidateOscMachineSpec(r.Spec); len(allErrs) > 0 {
		return nil, apierrors.NewInvalid(GroupVersion.WithKind("OscMachine").GroupKind(), r.Name, allErrs)
	}
	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator.
func (OscMachineWebhook) ValidateUpdate(_ context.Context, obj runtime.Object, oldRaw runtime.Object) (admission.Warnings, error) {
	r, ok := obj.(*OscMachine)
	if !ok {
		return nil, fmt.Errorf("expected an OscMachine object but got %T", r)
	}
	var allErrs field.ErrorList
	old := oldRaw.(*OscMachine)

	if r.Spec.Node.Vm.VmType != old.Spec.Node.Vm.VmType {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("node", "vm", "vmType"),
				r.Spec.Node.Vm.VmType, "field is immutable"),
		)
	}

	if r.Spec.Node.Vm.KeypairName != old.Spec.Node.Vm.KeypairName {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("node", "vm", "keyPairName"),
				r.Spec.Node.Vm.KeypairName, "field is immutable"),
		)
	}

	if old.Spec.Node.Vm.SubregionName != "" && r.Spec.Node.Vm.SubregionName != old.Spec.Node.Vm.SubregionName {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("node", "vm", "subregionName"),
				r.Spec.Node.Vm.SubregionName, "field is immutable"),
		)
	}

	if len(old.Spec.Node.Vm.Tags) > 0 && !maps.Equal(r.Spec.Node.Vm.Tags, old.Spec.Node.Vm.Tags) {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("node", "vm", "tags"),
				r.Spec.Node.Vm.Tags, "field is immutable"),
		)
	}

	if (old.Spec.Node.Vm.SubnetName != "") && r.Spec.Node.Vm.SubnetName != old.Spec.Node.Vm.SubnetName {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("node", "vm", "subnetName"),
				r.Spec.Node.Vm.SubnetName, "field is immutable"),
		)
	}
	if r.Spec.Node.Vm.RootDisk.RootDiskSize != old.Spec.Node.Vm.RootDisk.RootDiskSize {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("node", "vm", "rootDisk", "rootDiskSize"),
				r.Spec.Node.Vm.RootDisk.RootDiskSize, "field is immutable"),
		)
	}

	if r.Spec.Node.Vm.RootDisk.RootDiskIops != old.Spec.Node.Vm.RootDisk.RootDiskIops {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("node", "vm", "rootDisk", "rootDiskIops"),
				r.Spec.Node.Vm.RootDisk.RootDiskIops, "field is immutable"),
		)
	}

	if r.Spec.Node.Vm.RootDisk.RootDiskType != old.Spec.Node.Vm.RootDisk.RootDiskType {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("node", "vm", "rootDisk", "rootDiskType"),
				r.Spec.Node.Vm.RootDisk.RootDiskType, "field is immutable"),
		)
	}

	if len(allErrs) == 0 {
		return nil, nil
	}
	return nil, apierrors.NewInvalid(GroupVersion.WithKind("OscMachine").GroupKind(), r.Name, allErrs)
}

// ValidateDelete implements webhook.CustomValidator.
func (OscMachineWebhook) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}
