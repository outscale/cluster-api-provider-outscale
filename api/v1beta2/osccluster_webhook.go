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
var oscclusterlog = logf.Log.WithName("osccluster-resource")

func (r *OscCluster) SetupWebhookWithManager(mgr ctrl.Manager) error {
	h := OscClusterWebhook{}
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).WithValidator(h).WithDefaulter(h).
		Complete()
}

type OscClusterWebhook struct{}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-infrastructure-cluster-x-k8s-io-v1beta2-osccluster,mutating=true,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=oscclusters,verbs=create;update,versions=v1beta2,name=mosccluster.kb.io,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = OscClusterWebhook{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (OscClusterWebhook) Default(ctx context.Context, obj runtime.Object) error {
	r, ok := obj.(*OscCluster)
	if !ok {
		return fmt.Errorf("expected an OscCluster object but got %T", r)
	}
	oscclusterlog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-infrastructure-cluster-x-k8s-io-v1beta2-osccluster,mutating=false,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=oscclusters,verbs=create;update,versions=v1beta2,name=vosccluster.kb.io,admissionReviewVersions=v1

var _ webhook.CustomValidator = OscClusterWebhook{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (OscClusterWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	r, ok := obj.(*OscCluster)
	if !ok {
		return nil, fmt.Errorf("expected an OscCluster object but got %T", r)
	}
	oscclusterlog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (OscClusterWebhook) ValidateUpdate(ctx context.Context, obj runtime.Object, old runtime.Object) (admission.Warnings, error) {
	r, ok := obj.(*OscCluster)
	if !ok {
		return nil, fmt.Errorf("expected an OscCluster object but got %T", r)
	}
	oscclusterlog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (OscClusterWebhook) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	r, ok := obj.(*OscCluster)
	if !ok {
		return nil, fmt.Errorf("expected an OscCluster object but got %T", r)
	}
	oscclusterlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}
