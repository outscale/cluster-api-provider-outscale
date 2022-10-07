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

var oscClusterLog = logf.Log.WithName("osccluster-resource")

// SetupWithManager sets up the controller with the Manager.
func (r *OscCluster) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-infrastructure-cluster-x-k8s-io-v1beta1-osccluster,mutating=true,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=oscclusters,verbs=create;update,versions=v1beta1,name=mosccluster.kb.io,admissionReviewVersions=v1
var _ webhook.Defaulter = &OscCluster{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (m *OscCluster) Default() {
	oscClusterLog.Info("default", "name", m.Name)

}

//+kubebuilder:webhook:path=/validate-infrastructure-cluster-x-k8s-io-v1beta1-osccluster,mutating=false,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=oscclusters,verbs=create;update,versions=v1beta1,name=vosccluster.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &OscCluster{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (m *OscCluster) ValidateCreate() error {
	oscMachineLog.Info("validate create", "name", m.Name)
	if allErrs := ValidateOscClusterSpec(m.Spec); len(allErrs) > 0 {
		oscClusterLog.Info("validate error", "error", allErrs)
		return apierrors.NewInvalid(GroupVersion.WithKind("OscCluster").GroupKind(), m.Name, allErrs)
	}
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *OscCluster) ValidateUpdate(oldRaw runtime.Object) error {
	oscClusterLog.Info("validate update", "name", r.Name)
	var allErrs field.ErrorList
	old := oldRaw.(*OscCluster)

	oscClusterLog.Info("validate update old loadBalanceName", "loadBalanceName", old.Spec.Network.LoadBalancer.LoadBalancerName)
	oscClusterLog.Info("validate update old loadBalanceName", "loadBalanceName", r.Spec.Network.LoadBalancer.LoadBalancerName)

	if !reflect.DeepEqual(r.Spec.Network.LoadBalancer.LoadBalancerName, old.Spec.Network.LoadBalancer.LoadBalancerName) {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "loadBalancerName"),
				r.Spec.Network.LoadBalancer.LoadBalancerName, "field is immutable"),
		)
	}
	if len(allErrs) == 0 {
		return nil
	}
	return apierrors.NewInvalid(GroupVersion.WithKind("OscCluster").GroupKind(), r.Name, allErrs)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *OscCluster) ValidateDelete() error {
	oscClusterLog.Info("validate delete", "name", r.Name)
	return nil
}
