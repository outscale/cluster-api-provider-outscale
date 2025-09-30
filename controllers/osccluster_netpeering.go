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

func (r *OscClusterReconciler) getMgmtAccountID(clusterScope *scope.ClusterScope) string {
	if clusterScope.GetNetwork().NetPeering.ManagementAccountID != "" {
		return clusterScope.GetNetwork().NetPeering.ManagementAccountID
	}
	return r.Metadata.AccountID
}

func (r *OscClusterReconciler) getMgmtNetID(clusterScope *scope.ClusterScope) string {
	if clusterScope.GetNetwork().NetPeering.ManagementNetID != "" {
		return clusterScope.GetNetwork().NetPeering.ManagementNetID
	}
	return r.Metadata.NetID
}

// reconcileNetPeering reconcile the NetPeering of the cluster.
func (r *OscClusterReconciler) reconcileNetPeering(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	if clusterScope.GetNetwork().UseExisting.Net {
		log.V(4).Info("Not reconciling netPeering for existing net")
		return reconcile.Result{}, nil
	}
	if !clusterScope.NeedReconciliation(infrastructurev1beta1.ReconcilerNetPeering) {
		log.V(4).Info("No need for netPeering reconciliation")
		return reconcile.Result{}, nil
	}
	log.V(4).Info("Reconciling netPeering")

	svc := r.Cloud.NetPeering(clusterScope.Tenant)
	np, err := r.Tracker.getNetPeering(ctx, clusterScope)
	switch {
	case errors.Is(err, ErrNoResourceFound):
	case err != nil:
		return reconcile.Result{}, fmt.Errorf("get existing: %w", err)
	case np.State.GetName() == "active":
		log.V(4).Info("Found active netPeering", "mgmtNetID", np.AccepterNet.GetNetId(), "mgmtAccount", np.AccepterNet.GetAccountId())
		return reconcile.Result{}, nil
	}
	netId, err := r.Tracker.getNetId(ctx, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	if np == nil || np.State.GetName() != "pending-acceptance" {
		log.V(3).Info("Creating netPeering")
		np, err = svc.CreateNetPeering(ctx, netId, r.getMgmtNetID(clusterScope), r.getMgmtAccountID(clusterScope), clusterScope.GetUID())
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot create netPeering: %w", err)
		}
		log.V(2).Info("Created netPeering", "netPeeringId", np.GetNetPeeringId())
		r.Tracker.setNetPeeringId(clusterScope, np.GetNetPeeringId())
		r.Recorder.Event(clusterScope.OscCluster, corev1.EventTypeNormal, infrastructurev1beta1.NetPeeringCreatedReason, "NetPeering created")
	}
	if np.State.GetName() == "pending-acceptance" {
		mgmt, err := getMgmtTenant(ctx, r.Client, r.Cloud, clusterScope.OscCluster)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot get mgmt credentials: %w", err)
		}
		mgmtSvc := r.Cloud.NetPeering(mgmt)
		log.V(2).Info("Accepting netPeering")
		err = mgmtSvc.AcceptNetPeering(ctx, np.GetNetPeeringId())
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot accept netPeering: %w", err)
		}
		r.Recorder.Event(clusterScope.OscCluster, corev1.EventTypeNormal, infrastructurev1beta1.NetPeeringCreatedReason, "NetPeering accepted")
	}
	clusterScope.SetReconciliationGeneration(infrastructurev1beta1.ReconcilerNetPeering)
	return reconcile.Result{}, nil
}

// reconcileDeleteNetPeering reconcile the destruction of the NetPeering of the cluster.
func (r *OscClusterReconciler) reconcileDeleteNetPeering(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	if clusterScope.GetNetwork().UseExisting.Net {
		log.V(4).Info("Not deleting existing netPeerings")
		return reconcile.Result{}, nil
	}
	netId, err := r.Tracker.getNetId(ctx, clusterScope)
	switch {
	case errors.Is(err, ErrNoResourceFound) || errors.Is(err, ErrMissingResource):
		log.V(4).Info("The net is already deleted, no net peerings expected")
		return reconcile.Result{}, nil
	case err != nil:
		return reconcile.Result{}, fmt.Errorf("find net: %w", err)
	}
	svc := r.Cloud.NetPeering(clusterScope.Tenant)
	nps, err := svc.ListNetPeerings(ctx, netId)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("list netPeerings: %w", err)
	}
	for _, np := range nps {
		if np.State.GetName() != "pending-acceptance" && np.State.GetName() != "active" {
			continue
		}
		log.V(2).Info("Deleting netPeering", "subnetId", np.GetNetPeeringId())
		err = svc.DeleteNetPeering(ctx, np.GetNetPeeringId())
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot delete netPeering: %w", err)
		}
	}
	return reconcile.Result{}, nil
}
