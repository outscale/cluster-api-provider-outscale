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
var oscmachinelog = logf.Log.WithName("oscmachine-resource")

func (r *OscMachine) SetupWebhookWithManager(mgr ctrl.Manager) error {
	h := OscMachineWebhook{}
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).WithValidator(h).WithDefaulter(h).
		Complete()
}

type OscMachineWebhook struct{}

//+kubebuilder:webhook:path=/mutate-infrastructure-cluster-x-k8s-io-v1beta2-oscmachine,mutating=true,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=oscmachines,verbs=create;update,versions=v1beta2,name=moscmachine.kb.io,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = OscMachineWebhook{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (OscMachineWebhook) Default(ctx context.Context, obj runtime.Object) error {
	r, ok := obj.(*OscMachine)
	if !ok {
		return fmt.Errorf("expected an OscMachine object but got %T", r)
	}
	oscmachinelog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-infrastructure-cluster-x-k8s-io-v1beta2-oscmachine,mutating=false,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=oscmachines,verbs=create;update,versions=v1beta2,name=voscmachine.kb.io,admissionReviewVersions=v1

var _ webhook.CustomValidator = OscMachineWebhook{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (OscMachineWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	r, ok := obj.(*OscMachine)
	if !ok {
		return nil, fmt.Errorf("expected an OscMachine object but got %T", r)
	}
	oscmachinelog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (OscMachineWebhook) ValidateUpdate(ctx context.Context, obj runtime.Object, old runtime.Object) (admission.Warnings, error) {
	r, ok := obj.(*OscMachine)
	if !ok {
		return nil, fmt.Errorf("expected an OscMachine object but got %T", r)
	}
	oscmachinelog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (OscMachineWebhook) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	r, ok := obj.(*OscMachine)
	if !ok {
		return nil, fmt.Errorf("expected an OscMachine object but got %T", r)
	}
	oscmachinelog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}
