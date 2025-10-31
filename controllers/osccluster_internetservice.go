/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package controllers

import (
	"context"
	"errors"
	"fmt"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// reconcileInternetService reconcile the InternetService of the cluster.
func (r *OscClusterReconciler) reconcileInternetService(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	if !clusterScope.NeedReconciliation(infrastructurev1beta1.ReconcilerInternetService) {
		log.V(4).Info("No need for internetService reconciliation")
		return reconcile.Result{}, nil
	}
	if clusterScope.GetNetwork().UseExisting.Net {
		log.V(3).Info("Reusing existing internetService")
		return reconcile.Result{}, nil
	}
	if clusterScope.IsInternetDisabled() {
		log.V(3).Info("No internet service, internet is disabled")
		return reconcile.Result{}, nil
	}

	log.V(4).Info("Reconciling internetService")

	internetService, err := r.Tracker.getInternetService(ctx, clusterScope)
	switch {
	case errors.Is(err, ErrNoResourceFound):
	case err != nil:
		return reconcile.Result{}, fmt.Errorf("get existing: %w", err)
	case internetService.NetId != nil:
		log.V(4).Info("Found existing internetService", "internetServiceId", internetService.GetInternetServiceId())
		clusterScope.SetReconciliationGeneration(infrastructurev1beta1.ReconcilerInternetService)
		return reconcile.Result{}, nil
	}
	netId, err := r.Tracker.getNetId(ctx, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	if internetService == nil {
		log.V(3).Info("Creating internet service")
		internetService, err = r.Cloud.InternetService(clusterScope.Tenant).CreateInternetService(ctx, clusterScope.GetInternetServiceName(), clusterScope.GetUID())
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot create internetService: %w", err)
		}
		log.V(2).Info("Created internet service", "internetServiceId", internetService.GetInternetServiceId())
		r.Recorder.Event(clusterScope.OscCluster, corev1.EventTypeNormal, infrastructurev1beta1.InternetServicesCreatedReason, "Internet service created")
	}
	log.V(2).Info("Linking internet service to net", "internetServiceId", internetService.GetInternetServiceId(), "netId", netId)
	err = r.Cloud.InternetService(clusterScope.Tenant).LinkInternetService(ctx, internetService.GetInternetServiceId(), netId)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot link internetService: %w", err)
	}
	r.Tracker.setInternetServiceId(clusterScope, internetService.GetInternetServiceId())
	clusterScope.SetReconciliationGeneration(infrastructurev1beta1.ReconcilerInternetService)
	return reconcile.Result{}, nil
}

// reconcileDeleteInternetService reconcile the destruction of the InternetService of the cluster.
func (r *OscClusterReconciler) reconcileDeleteInternetService(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	if clusterScope.GetNetwork().UseExisting.Net {
		log.V(4).Info("Not deleting existing internet service")
		return reconcile.Result{}, nil
	}
	if clusterScope.IsInternetDisabled() {
		log.V(3).Info("No internet service, internet is disabled")
		return reconcile.Result{}, nil
	}
	internetService, err := r.Tracker.getInternetService(ctx, clusterScope)
	switch {
	case errors.Is(err, ErrNoResourceFound) || errors.Is(err, ErrMissingResource):
		log.V(4).Info("The internet service is already deleted")
		return reconcile.Result{}, nil
	case err != nil:
		return reconcile.Result{}, fmt.Errorf("get existing: %w", err)
	}

	internetServiceId := internetService.GetInternetServiceId()
	if internetService.NetId != nil {
		log.V(2).Info("Unlinking internetservice", "internetServiceId", internetServiceId)
		err = r.Cloud.InternetService(clusterScope.Tenant).UnlinkInternetService(ctx, internetServiceId, *internetService.NetId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot unlink internetService: %w", err)
		}
	}
	log.V(2).Info("Deleting internetservice", "internetServiceId", internetServiceId)
	err = r.Cloud.InternetService(clusterScope.Tenant).DeleteInternetService(ctx, internetServiceId)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot delete internetService: %w", err)
	}
	return reconcile.Result{}, nil
}
