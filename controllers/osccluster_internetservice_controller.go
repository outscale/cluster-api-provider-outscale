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

package controllers

import (
	"context"
	"errors"
	"fmt"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
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
	netSpec := clusterScope.GetNet()
	if netSpec.UseExisting {
		log.V(3).Info("Reusing existing internetService")
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
		internetService, err = r.Cloud.InternetService(ctx, *clusterScope).CreateInternetService(ctx, clusterScope.GetInternetServiceName(), clusterScope.GetUID())
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot create internetService: %w", err)
		}
		log.V(2).Info("Created internet service", "internetServiceId", internetService.GetInternetServiceId())
	}
	log.V(2).Info("Linking internet service to net", "internetServiceId", internetService.GetInternetServiceId(), "netId", netId)
	err = r.Cloud.InternetService(ctx, *clusterScope).LinkInternetService(ctx, internetService.GetInternetServiceId(), netId)
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
	netSpec := clusterScope.GetNet()
	if netSpec.UseExisting {
		log.V(4).Info("Not deleting existing internet service")
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
		err = r.Cloud.InternetService(ctx, *clusterScope).UnlinkInternetService(ctx, internetServiceId, *internetService.NetId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot unlink internetService: %w", err)
		}
	}
	log.V(2).Info("Deleting internetservice", "internetServiceId", internetServiceId)
	err = r.Cloud.InternetService(ctx, *clusterScope).DeleteInternetService(ctx, internetServiceId)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot delete internetService: %w", err)
	}
	return reconcile.Result{}, nil
}
