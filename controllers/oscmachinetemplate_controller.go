/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package controllers

import (
	"context"
	"time"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type OscMachineTemplateReconciler struct {
	client.Client
	Recorder         record.EventRecorder
	ReconcileTimeout time.Duration
	WatchFilterValue string
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=oscmachinetemplates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=oscmachinetemplates/status,verbs=get;update;patch

// Reconcile manages the lifecycle of an OscMachineTemplate object.
func (r *OscMachineTemplateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	log := ctrl.LoggerFrom(ctx)

	machineTemplate := &infrastructurev1beta1.OscMachineTemplate{}
	if err := r.Get(ctx, req.NamespacedName, machineTemplate); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	machineTemplateScope, err := scope.NewMachineTemplateScope(scope.MachineTemplateScopeParams{
		Client:             r.Client,
		OscMachineTemplate: machineTemplate,
	})
	if err != nil {
		log.V(3).Error(err, "failed to get machineTemplate, requeing.")
		return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
	}
	defer func() {
		if err := machineTemplateScope.Close(ctx); err != nil && reterr == nil {
			reterr = err
		}
	}()

	if !machineTemplate.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, machineTemplateScope)
	}

	return r.reconcile(ctx, machineTemplateScope)
}

// reconcile reconcile the creation of the machine
func (r *OscMachineTemplateReconciler) reconcile(ctx context.Context, machineTemplateScope *scope.MachineTemplateScope) (reconcile.Result, error) {
	return reconcileCapacity(ctx, machineTemplateScope)
}

// reconcileDelete reconcile the deletion of the machine
func (r *OscMachineTemplateReconciler) reconcileDelete(ctx context.Context, machineTemplateScope *scope.MachineTemplateScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.V(2).Info("Deleting OscMachineTemplate")
	// Previous versions set a oscmachine finalizer, remove it.
	controllerutil.RemoveFinalizer(machineTemplateScope.OscMachineTemplate, "oscmachine.infrastructure.cluster.x-k8s.io")
	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OscMachineTemplateReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		For(&infrastructurev1beta1.OscMachineTemplate{}).
		WithEventFilter(predicates.ResourceNotPausedAndHasFilterLabel(mgr.GetScheme(), ctrl.LoggerFrom(ctx), r.WatchFilterValue)).
		Complete(r)
}
