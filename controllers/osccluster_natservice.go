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

// reconcileNatService reconcile the NatService of the cluster.
func (r *OscClusterReconciler) reconcileNatService(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	if !clusterScope.NeedReconciliation(infrastructurev1beta2.ReconcilerNatService) {
		log.V(4).Info("No need for natService reconciliation")
		return reconcile.Result{}, nil
	}
	if clusterScope.GetSpec().UseExisting.Net {
		log.V(3).Info("Reusing existing natServices")
		return reconcile.Result{}, nil
	}
	if clusterScope.GetSpec().Disable.Internet {
		log.V(3).Info("No nat services, internet is disabled")
		return reconcile.Result{}, nil
	}
	log.V(4).Info("Reconciling natServices")

	for _, n := range clusterScope.GetNets() {
		for _, subnetSpec := range n.Subnets {
			if !subnetSpec.HasRole(infrastructurev1beta2.RoleNat) {
				continue
			}
			subnetId, err := r.Tracker.getSubnetId(ctx, subnetSpec, clusterScope)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("get subnet: %w", err)
			}

			natService, err := r.Tracker.getNatService(ctx, subnetId, clusterScope)
			switch {
			case IsNotFound(err):
			case err != nil:
				return reconcile.Result{}, fmt.Errorf("find existing: %w", err)
			default:
				log.V(4).Info("Found existing natService", "subnetId", subnetId, "natServiceId", natService.NatServiceId)
				continue
			}

			publicIpId, _, err := r.Tracker.IPAllocator(clusterScope).AllocateIP(ctx,
				clusterScope.GetNatServiceIPTrackingID(subnetId), clusterScope.GetNatServiceName(subnetSpec), clusterScope.GetSpec().NatPublicIpPool, clusterScope)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("allocate IP: %w", err)
			}

			log.V(3).Info("Creating natService", "subnetId", subnetId)
			natService, err = r.Cloud.Net(clusterScope.Tenant).CreateNatService(ctx, publicIpId, subnetId,
				clusterScope.GetNatServiceName(subnetSpec), clusterScope.GetUID())
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("cannot create natService: %w", err)
			}
			log.V(2).Info("Created natService", "subnetId", subnetId, "natServiceId", natService.NatServiceId)
			r.Tracker.setNatServiceId(clusterScope, subnetId, natService.NatServiceId)
			r.Recorder.Eventf(clusterScope.OscCluster, corev1.EventTypeNormal, infrastructurev1beta2.NatServicesCreatedReason, "NAT created in %s", subnetId)
		}
	}
	clusterScope.SetReconciliationGeneration(infrastructurev1beta2.ReconcilerNatService)
	return reconcile.Result{}, nil
}

func (r *OscClusterReconciler) listNATPublicIPs(ctx context.Context, clusterScope *scope.ClusterScope, cidr bool) ([]string, error) {
	var ips []string
	for _, n := range clusterScope.GetNets() {
		netId, err := r.Tracker.getNetId(ctx, n.IpRange, clusterScope)
		if err != nil {
			return nil, fmt.Errorf("cannot find net: %w", err)
		}
		natSvc := r.Cloud.Net(clusterScope.Tenant)
		nats, err := natSvc.ListNatServices(ctx, netId)
		if err != nil {
			return nil, fmt.Errorf("list natServices: %w", err)
		}
		for _, nat := range nats {
			for _, ip := range nat.PublicIps {
				pip := ip.PublicIp
				if cidr {
					pip += "/32"
				}
				ips = append(ips, pip)
			}
		}
	}
	return ips, nil
}

// reconcileDeleteNatService reconcile the destruction of the NatService of the cluster.
func (r *OscClusterReconciler) reconcileDeleteNatService(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	if clusterScope.GetSpec().UseExisting.Net {
		log.V(3).Info("Not deleting existing nat services")
		return reconcile.Result{}, nil
	}
	for _, n := range clusterScope.GetNets() {
		netId, err := r.Tracker.getNetId(ctx, n.IpRange, clusterScope)
		switch {
		case IsNotFound(err):
			log.V(4).Info("The net is already deleted, no nat service expected")
			continue
		case err != nil:
			return reconcile.Result{}, fmt.Errorf("find existing: %w", err)
		}
		natSvc := r.Cloud.Net(clusterScope.Tenant)
		nats, err := natSvc.ListNatServices(ctx, netId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("list natServices: %w", err)
		}

		for _, nat := range nats {
			if nat.State == osc.NatServiceStateDeleted {
				continue
			}
			log.V(2).Info("Deleting natService", "natId", nat.NatServiceId)
			err = natSvc.DeleteNatService(ctx, nat.NatServiceId)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("cannot delete natService: %w", err)
			}
			if len(nat.PublicIps) > 0 {
				// retrack publick IP in case it is missing from status
				r.Tracker.trackIP(clusterScope, clusterScope.GetNatServiceIPTrackingID(nat.SubnetId), nat.PublicIps[0].PublicIpId)
			}
		}
	}
	return reconcile.Result{}, nil
}
