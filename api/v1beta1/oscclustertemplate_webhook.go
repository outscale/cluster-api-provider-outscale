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
	"k8s.io/apimachinery/pkg/util/validation/field"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	OscClusterTemplateImmutableMsg = "OscClusterTemplate spec.template.spec field is immutable."
)

// log is for logging in this package.
var oscClusterTemplateLog = logf.Log.WithName("oscclustertemplate-resource")

func (r *OscClusterTemplate) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(r).Complete()
}

//+kubebuilder:webhook:path=/mutate-infrastructure-cluster-x-k8s-io-v1beta1-oscclustertemplate,mutating=true,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=oscclustertemplates,verbs=create;update,versions=v1beta1,name=moscclustertemplate.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &OscClusterTemplate{}

func (r *OscClusterTemplate) Default() {
	oscClusterTemplateLog.Info("default", "name", r.Name)

}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-infrastructure-cluster-x-k8s-io-v1beta1-oscclustertemplate,mutating=false,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=oscclustertemplates,verbs=create;update,versions=v1beta1,name=voscclustertemplate.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &OscClusterTemplate{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *OscClusterTemplate) ValidateCreate() (admission.Warnings, error) {
	oscClusterTemplateLog.Info("validate create", "name", r.Name)
	if allErrs := ValidateOscClusterSpec(r.Spec.Template.Spec); len(allErrs) > 0 {
		oscClusterTemplateLog.Info("validate error", "error", allErrs)
		return nil, apierrors.NewInvalid(GroupVersion.WithKind("OscClusterTemplate").GroupKind(), r.Name, allErrs)
	}
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *OscClusterTemplate) ValidateUpdate(oldRaw runtime.Object) (admission.Warnings, error) {
	oscClusterTemplateLog.Info("validate update", "name", r.Name)
	var allErrs field.ErrorList
	old := oldRaw.(*OscClusterTemplate)
	if !reflect.DeepEqual(r.Spec.Template.Spec, old.Spec.Template.Spec) {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("OscClusterTemplate", "spec", "template", "spec"), r, OscClusterTemplateImmutableMsg),
		)
	}
	if len(allErrs) == 0 {
		return nil, nil
	}
	return nil, apierrors.NewInvalid(GroupVersion.WithKind("OscClusterTemmplate").GroupKind(), r.Name, allErrs)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *OscClusterTemplate) ValidateDelete() (admission.Warnings, error) {
	oscClusterTemplateLog.Info("validate delete", "name", r.Name)
	return nil, nil
}
