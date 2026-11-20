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
	"github.com/outscale/osc-sdk-go/v3/pkg/osc"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *OscClusterReconciler) getMgmtAccountID(clusterScope *scope.ClusterScope) string {
	if clusterScope.GetSpec().NetPeering.ManagementAccountID != "" {
		return clusterScope.GetSpec().NetPeering.ManagementAccountID
	}
	return r.Metadata.AccountID
}

func (r *OscClusterReconciler) getMgmtNetID(clusterScope *scope.ClusterScope) string {
	if clusterScope.GetSpec().NetPeering.ManagementNetID != "" {
		return clusterScope.GetSpec().NetPeering.ManagementNetID
	}
	return r.Metadata.NetID
}

type netPeering struct {
	management bool
	netID      string
	accountID  string
	ipRange    string
}

func (r *OscClusterReconciler) listNetPeerings(clusterScope *scope.ClusterScope) []netPeering {
	var nps []netPeering

	if clusterScope.GetSpec().NetPeering.Enable {
		nps = append(nps, netPeering{
			management: true,
			netID:      r.getMgmtNetID(clusterScope),
			accountID:  r.getMgmtAccountID(clusterScope),
		})
	}
	nets := clusterScope.GetNets()
	if len(nets) <= 1 {
		return nps
	}
	for _, net := range nets[1:] {
		nps = append(nps, netPeering{
			ipRange: net.IpRange,
		})
	}
	return nps
}

// reconcileNetPeering reconcile the NetPeering of the cluster.
func (r *OscClusterReconciler) reconcileNetPeering(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	if clusterScope.GetSpec().UseExisting.Net {
		log.V(4).Info("Not reconciling netPeering for existing net")
		return reconcile.Result{}, nil
	}
	if !clusterScope.NeedReconciliation(infrastructurev1beta2.ReconcilerNetPeering) {
		log.V(4).Info("No need for netPeering reconciliation")
		return reconcile.Result{}, nil
	}
	log.V(4).Info("Reconciling netPeering")

	// hub net for all netPeerings
	netSpec := clusterScope.GetNets()[0]
	hubNetId, err := r.Tracker.getNetId(ctx, netSpec.IpRange, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}

	svc := r.Cloud.Net(clusterScope.Tenant)
	for _, spec := range r.listNetPeerings(clusterScope) {
		if !spec.management {
			netID, err := r.Tracker.getNetId(ctx, spec.ipRange, clusterScope)
			if err != nil {
				return reconcile.Result{}, err
			}
			spec.netID = netID
		}
		np, err := r.Tracker.getNetPeering(ctx, clusterScope, spec.management, spec.ipRange)
		switch {
		case IsNotFound(err):
		case err != nil:
			return reconcile.Result{}, fmt.Errorf("get existing: %w", err)
		case np.State.Name == osc.NetPeeringStateNameActive:
			log.V(4).Info("Found active netPeering", "management", spec.management, "netID", np.AccepterNet.NetId, "account", np.AccepterNet.AccountId)
			return reconcile.Result{}, nil
		}
		if np == nil || np.State.Name != osc.NetPeeringStateNamePendingAcceptance {
			log.V(3).Info("Creating netPeering", "hub", hubNetId, "spoke", spec.netID)
			np, err = svc.CreateNetPeering(ctx, hubNetId, spec.netID, spec.accountID, clusterScope.GetUID())
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("cannot create netPeering: %w", err)
			}
			log.V(2).Info("Created netPeering", "netPeeringId", np.NetPeeringId)
			r.Tracker.setNetPeeringId(clusterScope, spec.management, spec.ipRange, np.NetPeeringId)
			r.Recorder.Eventf(clusterScope.OscCluster, corev1.EventTypeNormal, infrastructurev1beta2.NetPeeringCreatedReason, "NetPeering with %s created", spec.netID)
		}
		if np.State.Name == osc.NetPeeringStateNamePendingAcceptance {
			spokeSvc := svc
			if spec.management {
				mgmt, err := getMgmtTenant(ctx, r.Client, r.Cloud, clusterScope.OscCluster)
				if err != nil {
					return reconcile.Result{}, fmt.Errorf("cannot get mgmt credentials: %w", err)
				}
				spokeSvc = r.Cloud.Net(mgmt)
			}
			log.V(2).Info("Accepting netPeering", "netPeeringId", np.NetPeeringId)
			err = spokeSvc.AcceptNetPeering(ctx, np.NetPeeringId)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("cannot accept netPeering: %w", err)
			}
			r.Recorder.Eventf(clusterScope.OscCluster, corev1.EventTypeNormal, infrastructurev1beta2.NetPeeringCreatedReason, "NetPeering with %s accepted", spec.netID)
		}
		clusterScope.SetReconciliationGeneration(infrastructurev1beta2.ReconcilerNetPeering)
	}
	return reconcile.Result{}, nil
}

// reconcileDeleteNetPeering reconcile the destruction of the NetPeering of the cluster.
func (r *OscClusterReconciler) reconcileDeleteNetPeering(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	if clusterScope.GetSpec().UseExisting.Net {
		log.V(4).Info("Not deleting existing netPeerings")
		return reconcile.Result{}, nil
	}
	netSpec := clusterScope.GetNets()[0]
	netId, err := r.Tracker.getNetId(ctx, netSpec.IpRange, clusterScope)
	switch {
	case IsNotFound(err):
		log.V(4).Info("The net is already deleted, no net peerings expected")
		return reconcile.Result{}, nil
	case err != nil:
		return reconcile.Result{}, fmt.Errorf("find net: %w", err)
	}
	svc := r.Cloud.Net(clusterScope.Tenant)
	nps, err := svc.ListNetPeerings(ctx, netId)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("list netPeerings: %w", err)
	}
	for _, np := range nps {
		if np.State.Name != osc.NetPeeringStateNamePendingAcceptance && np.State.Name != osc.NetPeeringStateNameActive {
			continue
		}
		log.V(2).Info("Deleting netPeering", "netPeeringId", np.NetPeeringId)
		err = svc.DeleteNetPeering(ctx, np.NetPeeringId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot delete netPeering: %w", err)
		}
	}
	return reconcile.Result{}, nil
}
