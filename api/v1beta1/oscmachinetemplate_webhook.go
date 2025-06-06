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
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var oscMachineTemplateLog = logf.Log.WithName("oscmachinetemplate-resource")

func (m *OscMachineTemplate) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(m).Complete()
}

//+kubebuilder:webhook:path=/mutate-infrastructure-cluster-x-k8s-io-v1beta1-oscmachinetemplate,mutating=true,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=oscmachinetemplates,verbs=create;update,versions=v1beta1,name=moscmachinetemplate.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &OscMachineTemplate{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (m *OscMachineTemplate) Default() {
	oscMachineTemplateLog.Info("default", "name", m.Name)
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-infrastructure-cluster-x-k8s-io-v1beta1-oscmachinetemplate,mutating=false,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=oscmachinetemplates,verbs=create;update,versions=v1beta1,name=voscmachinetemplate.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &OscMachineTemplate{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (m *OscMachineTemplate) ValidateCreate() (admission.Warnings, error) {
	if allErrs := ValidateOscMachineSpec(m.Spec.Template.Spec); len(allErrs) > 0 {
		return nil, apierrors.NewInvalid(GroupVersion.WithKind("OscMachine").GroupKind(), m.Name, allErrs)
	}
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (m *OscMachineTemplate) ValidateUpdate(oldRaw runtime.Object) (admission.Warnings, error) {
	var allErrs field.ErrorList
	old := oldRaw.(*OscMachineTemplate)
	if !reflect.DeepEqual(m.Spec.Template.Spec, old.Spec.Template.Spec) {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("template", "spec"), m, "spec is immutable"),
		)
	}
	if len(allErrs) == 0 {
		return nil, nil
	}
	return nil, apierrors.NewInvalid(GroupVersion.WithKind("OscMachineTemplate").GroupKind(), m.Name, allErrs)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (m *OscMachineTemplate) ValidateDelete() (admission.Warnings, error) {
	return nil, nil
}
