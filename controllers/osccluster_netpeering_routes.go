/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package controllers

import (
	"context"
	"errors"
	"fmt"
	"slices"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/cluster-api-provider-outscale/cloud/tenant"
	osc "github.com/outscale/osc-sdk-go/v2"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// getManagementRouteTablesAndIPRange computes the list of route tables that need to be updated, and the associated IPRange.
func (r *OscClusterReconciler) getManagementRouteTablesAndIPRange(ctx context.Context, mgmt tenant.Tenant, clusterScope *scope.ClusterScope) ([]osc.RouteTable, string, error) {
	mgmtSvc := r.Cloud.RouteTable(mgmt)
	netId := r.getMgmtNetID(clusterScope)
	rtbls, err := mgmtSvc.GetRouteTablesFromNet(ctx, netId)
	if err != nil {
		return nil, "", fmt.Errorf("find mgmt route tables: %w", err)
	}
	subnetId := clusterScope.GetNetwork().NetPeering.ManagementSubnetID
	if subnetId == "" {
		n, err := r.Cloud.Net(mgmt).GetNet(ctx, netId)
		if err != nil {
			return nil, "", fmt.Errorf("get mgmt net: %w", err)
		}
		return rtbls, n.GetIpRange(), nil
	}
	sn, err := r.Cloud.Subnet(mgmt).GetSubnet(ctx, subnetId)
	if err != nil {
		return nil, "", fmt.Errorf("get mgmt subnet: %w", err)
	}
	for _, rtbl := range rtbls {
		if slices.ContainsFunc(rtbl.GetLinkRouteTables(), func(l osc.LinkRouteTable) bool {
			return l.GetSubnetId() == subnetId
		}) {
			return []osc.RouteTable{rtbl}, sn.GetIpRange(), nil
		}
	}
	return nil, "", errors.New("no routing table are linked to managementSubnetId")
}

// reconcileNetPeeringRoutes reconcile the NetPeering routes.
func (r *OscClusterReconciler) reconcileNetPeeringRoutes(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	if clusterScope.GetNetwork().UseExisting.Net {
		log.V(4).Info("Not reconciling netPeering routes for existing net")
		return reconcile.Result{}, nil
	}
	if !clusterScope.NeedReconciliation(infrastructurev1beta1.ReconcilerNetPeeringRoutes) {
		log.V(4).Info("No need for netPeering routes reconciliation")
		return reconcile.Result{}, nil
	}
	log.V(4).Info("Reconciling netPeering routes")

	np, err := r.Tracker.getNetPeering(ctx, clusterScope)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("get netpeering: %w", err)
	}
	netId, err := r.Tracker.getNetId(ctx, clusterScope)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("get net: %w", err)
	}

	// Add routes to management route tables
	mgmt, err := getMgmtTenant(ctx, r.Client, r.Cloud, clusterScope.OscCluster)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("get mgmt credentials: %w", err)
	}
	mgmtSvc := r.Cloud.RouteTable(mgmt)
	mgmtRtbls, mgmtIPRange, err := r.getManagementRouteTablesAndIPRange(ctx, mgmt, clusterScope)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("list mgmt route tables: %w", err)
	}
	for _, mgmtRtbl := range mgmtRtbls {
		if !slices.ContainsFunc(mgmtRtbl.GetRoutes(), func(r osc.Route) bool {
			return r.GetNetPeeringId() == np.GetNetPeeringId()
		}) {
			wrkIPRange := clusterScope.GetNet().IpRange
			log.V(3).Info("Creating management route to netPeering", "routeTableId", mgmtRtbl.GetRouteTableId(), "IPRange", wrkIPRange)
			_, err := mgmtSvc.CreateRoute(ctx, wrkIPRange, mgmtRtbl.GetRouteTableId(), np.GetNetPeeringId(), "netPeering")
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("create mgmt route: %w", err)
			}
			r.Recorder.Event(clusterScope.OscCluster, corev1.EventTypeNormal, infrastructurev1beta1.NetPeeringCreatedReason, "NetPeering management route created")
		}
	}

	// Add routes to workload route tables
	svc := r.Cloud.RouteTable(clusterScope.Tenant)
	rtbls, err := svc.GetRouteTablesFromNet(ctx, netId)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("list workload route tables: %w", err)
	}
	for _, rtbl := range rtbls {
		if slices.ContainsFunc(rtbl.GetRoutes(), func(r osc.Route) bool {
			return r.GetNetPeeringId() == np.GetNetPeeringId()
		}) {
			log.V(5).Info(fmt.Sprintf("Found route from %q to %q", rtbl.GetRouteTableId(), np.GetNetPeeringId()))
			continue
		}
		log.V(3).Info("Creating workload route to netPeering", "routeTableId", rtbl.GetRouteTableId(), "IPRange", mgmtIPRange)
		_, err := svc.CreateRoute(ctx, mgmtIPRange, rtbl.GetRouteTableId(), np.GetNetPeeringId(), "netPeering")
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("create workload route: %w", err)
		}
		r.Recorder.Event(clusterScope.OscCluster, corev1.EventTypeNormal, infrastructurev1beta1.NetPeeringCreatedReason, "NetPeering workload route created")
	}

	clusterScope.SetReconciliationGeneration(infrastructurev1beta1.ReconcilerNetPeeringRoutes)
	return reconcile.Result{}, nil
}

// reconcileDeleteNetPeering reconcile the destruction of the NetPeering of the cluster.
func (r *OscClusterReconciler) reconcileDeleteNetPeeringRoutes(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	if clusterScope.GetNetwork().UseExisting.Net {
		log.V(4).Info("Not deleting existing netPeering routes")
		return reconcile.Result{}, nil
	}
	np, err := r.Tracker.getNetPeering(ctx, clusterScope)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("get netpeering: %w", err)
	}
	netId, err := r.Tracker.getNetId(ctx, clusterScope)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("get net: %w", err)
	}

	// remove routes from management route tables
	mgmt, err := getMgmtTenant(ctx, r.Client, r.Cloud, clusterScope.OscCluster)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("get mgmt credentials: %w", err)
	}
	mgmtSvc := r.Cloud.RouteTable(mgmt)
	mgmtRtbls, _, err := r.getManagementRouteTablesAndIPRange(ctx, mgmt, clusterScope)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("list mgmt route tables: %w", err)
	}
	for _, mgmtRtbl := range mgmtRtbls {
		for _, r := range mgmtRtbl.GetRoutes() {
			if r.GetNetPeeringId() != np.GetNetPeeringId() {
				continue
			}
			log.V(3).Info("Deleting management route to netPeering", "routeTableId", mgmtRtbl.GetRouteTableId(), "IPRange", r.GetDestinationIpRange())
			err = mgmtSvc.DeleteRoute(ctx, r.GetDestinationIpRange(), mgmtRtbl.GetRouteTableId())
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("delete mgmt route: %w", err)
			}
		}
	}

	// remove routes from workload route tables
	svc := r.Cloud.RouteTable(clusterScope.Tenant)
	rtbls, err := svc.GetRouteTablesFromNet(ctx, netId)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("list workload route tables: %w", err)
	}
	for _, rtbl := range rtbls {
		for _, r := range rtbl.GetRoutes() {
			if r.GetNetPeeringId() != np.GetNetPeeringId() {
				continue
			}
			log.V(3).Info("Deleting workload route to netPeering", "routeTableId", rtbl.GetRouteTableId(), "IPRange", r.GetDestinationIpRange())
			err := svc.DeleteRoute(ctx, r.GetDestinationIpRange(), rtbl.GetRouteTableId())
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("delete workload route: %w", err)
			}
		}
	}
	return reconcile.Result{}, nil
}
