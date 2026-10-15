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

// reconcileRoute reconcile the RouteTable and the Route of the cluster.
func (r *OscClusterReconciler) reconcileRoute(ctx context.Context, clusterScope *scope.ClusterScope, netSpec infrastructurev1beta2.OscNet, routeTableSpec infrastructurev1beta2.OscRouteTable, routeSpec infrastructurev1beta2.OscRoute, routeTable *osc.RouteTable) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	destinationIpRange := routeSpec.Destination
	for _, route := range routeTable.Routes {
		if route.DestinationIpRange == destinationIpRange {
			return reconcile.Result{}, nil
		}
	}
	var resourceId string
	var err error
	switch routeSpec.TargetType {
	case "gateway":
		resourceId, err = r.Tracker.getInternetServiceId(ctx, netSpec.IpRange, clusterScope)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("find internetService for route: %w", err)
		}
	case "nat":
		subnetSpec, err := clusterScope.GetSubnet(netSpec, infrastructurev1beta2.RoleNat, routeTableSpec.Subregion)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("find natService subnet for route: %w", err)
		}
		subnetId, err := r.Tracker.getSubnetId(ctx, subnetSpec, clusterScope)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("find natService subnet for route: %w", err)
		}
		resourceId, err = r.Tracker.getNatServiceId(ctx, subnetId, clusterScope)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("find natService for route: %w", err)
		}
	default:
		log.V(3).Info("Route has no target !", "destinationIpRange", destinationIpRange)
		return reconcile.Result{}, nil
	}
	log.V(2).Info("Creating route", "destination", destinationIpRange, "resourceId", resourceId)
	_, err = r.Cloud.Net(clusterScope.Tenant).CreateRoute(ctx, destinationIpRange, routeTable.RouteTableId, resourceId, routeSpec.TargetType)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot create route: %w", err)
	}

	return reconcile.Result{}, nil
}

// reconcileRouteTable reconcile the RouteTable and the Route of the cluster.
func (r *OscClusterReconciler) reconcileRouteTable(ctx context.Context, clusterScope *scope.ClusterScope, roles ...infrastructurev1beta2.OscRole) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	if !clusterScope.NeedReconciliation(infrastructurev1beta2.ReconcilerRouteTable) {
		log.V(4).Info("No need for routeTable reconciliation")
		return reconcile.Result{}, nil
	}
	if clusterScope.GetSpec().UseExisting.Net {
		log.V(3).Info("Using existing routeTables")
		return reconcile.Result{}, nil
	}
	log.V(4).Info("Reconciling routeTables")

	svc := r.Cloud.Net(clusterScope.Tenant)

	for _, netSpec := range clusterScope.GetNets() {
		netId, err := r.Tracker.getNetId(ctx, netSpec.IpRange, clusterScope)
		if err != nil {
			return reconcile.Result{}, err
		}
		rtbls, err := svc.GetRouteTablesFromNet(ctx, netId)
		if err != nil {
			return reconcile.Result{}, err
		}
		rtblForSubnet := map[string]*osc.RouteTable{}
		for _, rtbl := range rtbls {
			for _, link := range rtbl.LinkRouteTables {
				rtblForSubnet[link.SubnetId] = &rtbl
			}
		}
		routeTablesSpec := clusterScope.GetRouteTables(netSpec)
		for _, routeTableSpec := range routeTablesSpec {
			subnetSpec, err := clusterScope.GetSubnet(netSpec, routeTableSpec.Role, routeTableSpec.Subregion)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("cannot find subnet with role %q: %w", routeTableSpec.Role, err)
			}
			if len(roles) > 0 && !subnetSpec.HasRole(roles[0]) {
				continue
			}
			subnetId, err := r.Tracker.getSubnetId(ctx, subnetSpec, clusterScope)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("cannot get subnet: %w", err)
			}
			var rtbl *osc.RouteTable
			switch {
			case rtblForSubnet[subnetId] != nil:
				log.V(5).Info("Subnet has a route table", "subnetId", subnetId)
				rtbl = rtblForSubnet[subnetId]
			case rtblForSubnet[subnetId] == nil:
				log.V(2).Info("Creating routetable", "subnetId", subnetId)
				rtbl, err = svc.CreateRouteTable(ctx, netId, clusterScope.GetUID(), clusterScope.GetSubnetName(subnetSpec))
				if err != nil {
					return reconcile.Result{}, fmt.Errorf("cannot create routetable: %w", err)
				}
				log.V(2).Info("Created routetable", "routetableId", rtbl.RouteTableId)
				r.Recorder.Eventf(clusterScope.OscCluster, corev1.EventTypeNormal, infrastructurev1beta2.RouteTableCreatedReason, "Route table created %v %s", subnetSpec.Roles, subnetSpec.Subregion)
				log.V(2).Info("Link routetable to subnet", "routeTableId", rtbl.RouteTableId, "subnetId", subnetId)
				_, err := svc.LinkRouteTable(ctx, rtbl.RouteTableId, subnetId)
				if err != nil {
					return reconcile.Result{}, fmt.Errorf("cannot link routetable to subnet: %w", err)
				}
			}
			if rtbl == nil {
				return reconcile.Result{}, fmt.Errorf("no subnet found for route table %q", routeTableSpec.Description)
			}
			for _, routeSpec := range routeTableSpec.Routes {
				_, err = r.reconcileRoute(ctx, clusterScope, netSpec, routeTableSpec, routeSpec, rtbl)
				if err != nil {
					return reconcile.Result{}, err
				}
			}
		}
	}
	if len(roles) == 0 {
		clusterScope.SetReconciliationGeneration(infrastructurev1beta2.ReconcilerRouteTable)
	}
	return reconcile.Result{}, nil
}

// reconcileDeleteRouteTable reconcile the destruction of the RouteTable of the cluster.
func (r *OscClusterReconciler) reconcileDeleteRouteTable(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	if clusterScope.GetSpec().UseExisting.Net {
		log.V(3).Info("Not deleting existing routeTables")
		return reconcile.Result{}, nil
	}

	for _, netSpec := range clusterScope.GetNets() {
		netId, err := r.Tracker.getNetId(ctx, netSpec.IpRange, clusterScope)
		switch {
		case IsNotFound(err):
			log.V(4).Info("The net is already deleted, no route table expected")
			continue
		case err != nil:
			return reconcile.Result{}, err
		}
		svc := r.Cloud.Net(clusterScope.Tenant)
		rtbls, err := svc.GetRouteTablesFromNet(ctx, netId)
		if err != nil {
			return reconcile.Result{}, err
		}

		for _, rtbl := range rtbls {
			for _, link := range rtbl.LinkRouteTables {
				log.V(2).Info("Unlinking route table", "routeTableId", rtbl.RouteTableId, "linkId", link.LinkRouteTableId)
				err := svc.UnlinkRouteTable(ctx, link.LinkRouteTableId)
				if err != nil {
					return reconcile.Result{}, fmt.Errorf("cannot unlink routeTable: %w", err)
				}
			}
			log.V(2).Info("Deleting routeTable", "routeTableId", rtbl.RouteTableId)
			err = svc.DeleteRouteTable(ctx, rtbl.RouteTableId)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("cannot delete routeTable: %w", err)
			}
		}
	}
	return reconcile.Result{}, nil
}
