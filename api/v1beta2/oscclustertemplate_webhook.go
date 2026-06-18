/*
SPDX-FileCopyrightText: 2022 The Kubernetes Authors

SPDX-License-Identifier: Apache-2.0
*/

package v1beta2

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var oscclustertemplatelog = logf.Log.WithName("oscclustertemplate-resource")

func (r *OscClusterTemplate) SetupWebhookWithManager(mgr ctrl.Manager) error {
	h := OscClusterTemplateWebhook{}
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).WithValidator(h).WithDefaulter(h).
		Complete()
}

type OscClusterTemplateWebhook struct{}

//+kubebuilder:webhook:path=/mutate-infrastructure-cluster-x-k8s-io-v1beta2-oscclustertemplate,mutating=true,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=oscclustertemplates,verbs=create;update,versions=v1beta2,name=moscclustertemplate.kb.io,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = OscClusterTemplateWebhook{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (OscClusterTemplateWebhook) Default(ctx context.Context, obj runtime.Object) error {
	r, ok := obj.(*OscClusterTemplate)
	if !ok {
		return fmt.Errorf("expected an OscClusterTemplate object but got %T", r)
	}
	oscclustertemplatelog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-infrastructure-cluster-x-k8s-io-v1beta2-oscclustertemplate,mutating=false,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=oscclustertemplates,verbs=create;update,versions=v1beta2,name=voscclustertemplate.kb.io,admissionReviewVersions=v1

var _ webhook.CustomValidator = OscClusterTemplateWebhook{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (OscClusterTemplateWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	r, ok := obj.(*OscClusterTemplate)
	if !ok {
		return nil, fmt.Errorf("expected an OscClusterTemplate object but got %T", r)
	}
	oscclustertemplatelog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (OscClusterTemplateWebhook) ValidateUpdate(ctx context.Context, obj runtime.Object, old runtime.Object) (admission.Warnings, error) {
	r, ok := obj.(*OscClusterTemplate)
	if !ok {
		return nil, fmt.Errorf("expected an OscClusterTemplate object but got %T", r)
	}
	oscclustertemplatelog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (OscClusterTemplateWebhook) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	r, ok := obj.(*OscClusterTemplate)
	if !ok {
		return nil, fmt.Errorf("expected an OscClusterTemplate object but got %T", r)
	}
	oscclustertemplatelog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}
