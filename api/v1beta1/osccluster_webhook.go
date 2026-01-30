/*
SPDX-FileCopyrightText: 2022 The Kubernetes Authors

SPDX-License-Identifier: Apache-2.0
*/

package v1beta1

import (
	"context"
	"fmt"
	"slices"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// SetupWebhookWithManager sets up the controller with the Manager.
func (r *OscCluster) SetupWebhookWithManager(mgr ctrl.Manager) error {
	h := OscClusterWebhook{}
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).WithValidator(h).WithDefaulter(h).
		Complete()
}

// OscClusterWebhook is the validation/mutation webhook.
type OscClusterWebhook struct{}

// +kubebuilder:webhook:path=/mutate-infrastructure-cluster-x-k8s-io-v1beta1-osccluster,mutating=true,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=oscclusters,verbs=create;update,versions=v1beta1,name=mosccluster.kb.io,admissionReviewVersions=v1
var _ webhook.CustomDefaulter = OscClusterWebhook{}

// Default implements webhook.CustomDefaulter.
func (OscClusterWebhook) Default(_ context.Context, _ runtime.Object) error {
	return nil
}

//+kubebuilder:webhook:path=/validate-infrastructure-cluster-x-k8s-io-v1beta1-osccluster,mutating=false,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=oscclusters,verbs=create;update,versions=v1beta1,name=vosccluster.kb.io,admissionReviewVersions=v1

var _ webhook.CustomValidator = OscClusterWebhook{}

// ValidateCreate implements webhook.CustomValidator.
func (OscClusterWebhook) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	r, ok := obj.(*OscCluster)
	if !ok {
		return nil, fmt.Errorf("expected an OscCluster object but got %T", r)
	}
	if allErrs := ValidateOscClusterSpec(r.Spec); len(allErrs) > 0 {
		return nil, apierrors.NewInvalid(GroupVersion.WithKind("OscCluster").GroupKind(), r.Name, allErrs)
	}
	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator.
func (OscClusterWebhook) ValidateUpdate(_ context.Context, obj runtime.Object, oldRaw runtime.Object) (admission.Warnings, error) {
	r, ok := obj.(*OscCluster)
	if !ok {
		return nil, fmt.Errorf("expected an OscCluster object but got %T", r)
	}
	var allErrs field.ErrorList
	old := oldRaw.(*OscCluster)
	if !slices.Contains(r.Spec.Network.Disable, DisableLB) {
		if r.Spec.Network.LoadBalancer.LoadBalancerName != old.Spec.Network.LoadBalancer.LoadBalancerName {
			allErrs = append(allErrs,
				field.Invalid(field.NewPath("network", "loadBalancer", "loadbalancername"),
					r.Spec.Network.LoadBalancer.LoadBalancerName, "field is immutable"),
			)
		}
		if r.Spec.Network.LoadBalancer.LoadBalancerType != old.Spec.Network.LoadBalancer.LoadBalancerType {
			allErrs = append(allErrs,
				field.Invalid(field.NewPath("network", "loadBalancer", "loadbalancertype"),
					r.Spec.Network.LoadBalancer.LoadBalancerType, "field is immutable"),
			)
		}
	}
	if len(allErrs) == 0 {
		return nil, nil
	}
	return nil, apierrors.NewInvalid(GroupVersion.WithKind("OscCluster").GroupKind(), r.Name, allErrs)
}

// ValidateDelete implements webhook.CustomValidator.
func (OscClusterWebhook) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}
