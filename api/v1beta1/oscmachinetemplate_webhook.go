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
	"context"
	"fmt"
	"reflect"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// SetupWebhookWithManager sets up the controller with the Manager.
func (m *OscMachineTemplate) SetupWebhookWithManager(mgr ctrl.Manager) error {
	h := OscMachineTemplateWebhook{}
	return ctrl.NewWebhookManagedBy(mgr).
		For(m).WithValidator(h).WithDefaulter(h).
		Complete()
}

// OscMachineTemplateWebhook is the validation/mutation webhook.
type OscMachineTemplateWebhook struct{}

//+kubebuilder:webhook:path=/mutate-infrastructure-cluster-x-k8s-io-v1beta1-oscmachinetemplate,mutating=true,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=oscmachinetemplates,verbs=create;update,versions=v1beta1,name=moscmachinetemplate.kb.io,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = OscMachineTemplateWebhook{}

// Default implements webhook.CustomDefaulter.
func (OscMachineTemplateWebhook) Default(_ context.Context, _ runtime.Object) error {
	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-infrastructure-cluster-x-k8s-io-v1beta1-oscmachinetemplate,mutating=false,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=oscmachinetemplates,verbs=create;update,versions=v1beta1,name=voscmachinetemplate.kb.io,admissionReviewVersions=v1

var _ webhook.CustomValidator = OscMachineTemplateWebhook{}

// ValidateCreate implements webhook.CustomValidator.
func (OscMachineTemplateWebhook) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	r, ok := obj.(*OscMachineTemplate)
	if !ok {
		return nil, fmt.Errorf("expected an OscMachineTemplate object but got %T", r)
	}
	if allErrs := ValidateOscMachineSpec(r.Spec.Template.Spec); len(allErrs) > 0 {
		return nil, apierrors.NewInvalid(GroupVersion.WithKind("OscMachineTemplate").GroupKind(), r.Name, allErrs)
	}
	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator.
func (OscMachineTemplateWebhook) ValidateUpdate(_ context.Context, obj runtime.Object, oldRaw runtime.Object) (admission.Warnings, error) {
	r, ok := obj.(*OscMachineTemplate)
	if !ok {
		return nil, fmt.Errorf("expected an OscMachineTemplate object but got %T", r)
	}
	var allErrs field.ErrorList
	old := oldRaw.(*OscMachineTemplate)
	if !reflect.DeepEqual(r.Spec.Template.Spec, old.Spec.Template.Spec) {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("template", "spec"), r, "spec is immutable"),
		)
	}
	if len(allErrs) == 0 {
		return nil, nil
	}
	return nil, apierrors.NewInvalid(GroupVersion.WithKind("OscMachineTemplate").GroupKind(), r.Name, allErrs)
}

// ValidateDelete implements webhook.CustomValidator.
func (OscMachineTemplateWebhook) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}
