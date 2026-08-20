/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package controllers

import (
	"context"
	"fmt"

	infrastructurev1beta2 "github.com/outscale/cluster-api-provider-outscale/api/v1beta2"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// reconcileNet reconcile the Net of the cluster.
func (r *OscClusterReconciler) reconcileNet(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	if !clusterScope.NeedReconciliation(infrastructurev1beta2.ReconcilerNet) {
		log.V(4).Info("No need for net reconciliation")
		return reconcile.Result{}, nil
	}
	log.V(4).Info("Reconciling net")

	net, err := r.Tracker.getNet(ctx, clusterScope)
	switch {
	case IsNotFound(err) && !clusterScope.GetSpec().UseExisting.Net:
	case err != nil:
		return reconcile.Result{}, fmt.Errorf("find existing: %w", err)
	default:
		log.V(4).Info("Found existing net", "netId", net.NetId)
		clusterScope.SetReconciliationGeneration(infrastructurev1beta2.ReconcilerNet)
		return reconcile.Result{}, nil
	}
	log.V(3).Info("Creating net")
	netSpec := clusterScope.GetNet()
	net, err = r.Cloud.Net(clusterScope.Tenant).CreateNet(ctx, netSpec, clusterScope.GetUID(), clusterScope.GetNetName())
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot create net: %w", err)
	}
	log.V(2).Info("Created net", "netId", net.NetId)
	r.Tracker.setNetId(clusterScope, net.NetId)
	clusterScope.SetReconciliationGeneration(infrastructurev1beta2.ReconcilerNet)
	r.Recorder.Event(clusterScope.OscCluster, corev1.EventTypeNormal, infrastructurev1beta2.NetCreatedReason, "Net created")
	return reconcile.Result{}, nil
}

// reconcileDeleteNet reconcile the destruction of the Net of the cluster.
func (r *OscClusterReconciler) reconcileDeleteNet(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	if clusterScope.GetSpec().UseExisting.Net {
		log.V(4).Info("Not deleting existing net")
		return reconcile.Result{}, nil
	}
	net, err := r.Tracker.getNet(ctx, clusterScope)
	switch {
	case IsNotFound(err):
		log.V(4).Info("The net is already deleted")
		return reconcile.Result{}, nil
	case err != nil:
		return reconcile.Result{}, fmt.Errorf("find existing: %w", err)
	}
	log.V(2).Info("Deleting net", "netId", net.NetId)
	err = r.Cloud.Net(clusterScope.Tenant).DeleteNet(ctx, net.NetId)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot delete net: %w", err)
	}
	return reconcile.Result{}, nil
}
