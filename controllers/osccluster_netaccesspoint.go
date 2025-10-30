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

// reconcileNetAccessPoints reconcile the NetAccessPoints of the cluster.
func (r *OscClusterReconciler) reconcileNetAccessPoints(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	if !clusterScope.NeedReconciliation(infrastructurev1beta1.ReconcilerNetAccessPoint) {
		log.V(4).Info("No need for netAccessPoint reconciliation")
		return reconcile.Result{}, nil
	}
	if clusterScope.GetNetwork().UseExisting.Net {
		log.V(3).Info("Reusing existing netAccessPoints")
		return reconcile.Result{}, nil
	}
	log.V(4).Info("Reconciling netAccessPoints")
	netId, err := r.Tracker.getNetId(ctx, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	region := clusterScope.Tenant.Region()

	// Find routetables to link to
	svc := r.Cloud.RouteTable(clusterScope.Tenant)
	rtbls, err := svc.GetRouteTablesFromNet(ctx, netId)
	if err != nil {
		return reconcile.Result{}, err
	}
	rtblForSubnet := map[string]string{}
	for _, rtbl := range rtbls {
		for _, link := range rtbl.GetLinkRouteTables() {
			rtblForSubnet[link.GetSubnetId()] = rtbl.GetRouteTableId()
		}
	}
	rtblIds := make([]string, 0, len(rtbls))
	for _, subnetSpec := range clusterScope.GetSubnets() {
		if !clusterScope.SubnetHasRole(subnetSpec, infrastructurev1beta1.RoleWorker) && !clusterScope.SubnetHasRole(subnetSpec, infrastructurev1beta1.RoleControlPlane) {
			continue
		}
		subnetId, err := r.Tracker.getSubnetId(ctx, subnetSpec, clusterScope)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot get subnet: %w", err)
		}
		rtblIds = append(rtblIds, rtblForSubnet[subnetId])
	}

	for _, service := range clusterScope.GetNetwork().NetAccessPoints {
		netAccessPoint, err := r.Tracker.getNetAccessPoint(ctx, service, clusterScope)
		switch {
		case errors.Is(err, ErrNoResourceFound):
		case err != nil:
			return reconcile.Result{}, fmt.Errorf("get existing: %w", err)
		default:
			log.V(4).Info("Found existing netAccessPoint", "netAccessPointId", netAccessPoint.GetNetAccessPointId())
			continue
		}
		log.V(3).Info("Creating net access point", "service", service)
		netAccessPoint, err = r.Cloud.NetAccessPoint(clusterScope.Tenant).CreateNetAccessPoint(ctx, netId, region, string(service), rtblIds, clusterScope.GetUID())
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot create netAccessPoint: %w", err)
		}
		log.V(2).Info("Created net access point", "netAccessPointId", netAccessPoint.GetNetAccessPointId())
		r.Recorder.Eventf(clusterScope.OscCluster, corev1.EventTypeNormal, infrastructurev1beta1.NetAccessPointCreatedReason, "Net Access Point created %s", service)
		r.Tracker.setNetAccessPointId(clusterScope, service, netAccessPoint.GetNetAccessPointId())
	}
	clusterScope.SetReconciliationGeneration(infrastructurev1beta1.ReconcilerNetAccessPoint)
	return reconcile.Result{}, nil
}

// reconcileDeleteNetAccessPoints reconcile the destruction of the NetAccessPoint of the cluster.
func (r *OscClusterReconciler) reconcileDeleteNetAccessPoints(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	if clusterScope.GetNetwork().UseExisting.Net {
		log.V(4).Info("Not deleting existing netAccessPoints")
		return reconcile.Result{}, nil
	}
	netId, err := r.Tracker.getNetId(ctx, clusterScope)
	switch {
	case errors.Is(err, ErrNoResourceFound) || errors.Is(err, ErrMissingResource):
		log.V(4).Info("The net is already deleted, no net access point expected")
		return reconcile.Result{}, nil
	case err != nil:
		return reconcile.Result{}, err
	}
	svc := r.Cloud.NetAccessPoint(clusterScope.Tenant)
	naps, err := svc.ListNetAccessPoints(ctx, netId)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("list net access points: %w", err)
	}
	for _, nap := range naps {
		log.V(2).Info("Deleting netAccessPoint", "netAccessPointId", nap.GetNetAccessPointId())
		err = svc.DeleteNetAccessPoint(ctx, nap.GetNetAccessPointId())
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("delete netAccessPoint: %w", err)
		}
	}
	return reconcile.Result{}, nil
}
