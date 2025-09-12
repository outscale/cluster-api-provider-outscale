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
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// reconcileNet reconcile the Net of the cluster.
func (r *OscClusterReconciler) reconcileNet(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	if !clusterScope.NeedReconciliation(infrastructurev1beta1.ReconcilerNet) {
		log.V(4).Info("No need for net reconciliation")
		return reconcile.Result{}, nil
	}
	log.V(4).Info("Reconciling net")

	net, err := r.Tracker.getNet(ctx, clusterScope)
	switch {
	case errors.Is(err, ErrNoResourceFound):
	case err != nil:
		return reconcile.Result{}, fmt.Errorf("find existing: %w", err)
	default:
		log.V(4).Info("Found existing net", "netId", net.GetNetId())
		clusterScope.SetReconciliationGeneration(infrastructurev1beta1.ReconcilerNet)
		return reconcile.Result{}, nil
	}
	log.V(3).Info("Creating net")
	netSpec := clusterScope.GetNet()
	net, err = r.Cloud.Net(clusterScope.Tenant).CreateNet(ctx, netSpec, clusterScope.GetUID(), clusterScope.GetNetName())
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot create net: %w", err)
	}
	log.V(2).Info("Created net", "netId", net.GetNetId())
	r.Tracker.setNetId(clusterScope, net.GetNetId())
	clusterScope.SetReconciliationGeneration(infrastructurev1beta1.ReconcilerNet)
	r.Recorder.Event(clusterScope.OscCluster, corev1.EventTypeNormal, infrastructurev1beta1.NetCreatedReason, "Net created")
	return reconcile.Result{}, nil
}

// reconcileDeleteNet reconcile the destruction of the Net of the cluster.
func (r *OscClusterReconciler) reconcileDeleteNet(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	if clusterScope.GetNetwork().UseExisting.Net {
		log.V(4).Info("Not deleting existing net")
		return reconcile.Result{}, nil
	}
	net, err := r.Tracker.getNet(ctx, clusterScope)
	switch {
	case errors.Is(err, ErrNoResourceFound) || errors.Is(err, ErrMissingResource):
		log.V(4).Info("The net is already deleted")
		return reconcile.Result{}, nil
	case err != nil:
		return reconcile.Result{}, fmt.Errorf("find existing: %w", err)
	}
	log.V(2).Info("Deleting net", "netId", net.GetNetId())
	err = r.Cloud.Net(clusterScope.Tenant).DeleteNet(ctx, net.GetNetId())
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot delete net: %w", err)
	}
	return reconcile.Result{}, nil
}
