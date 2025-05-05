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
	"github.com/outscale/osc-sdk-go/v2"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// reconcileRoute reconcile the RouteTable and the Route of the cluster.
func (r *OscClusterReconciler) reconcileRoute(ctx context.Context, clusterScope *scope.ClusterScope, routeTableSpec infrastructurev1beta1.OscRouteTable, routeSpec infrastructurev1beta1.OscRoute, routeTable *osc.RouteTable) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	destinationIpRange := routeSpec.Destination
	for _, route := range routeTable.GetRoutes() {
		if route.GetDestinationIpRange() == destinationIpRange {
			return reconcile.Result{}, nil
		}
	}
	var resourceId string
	var err error
	if routeSpec.TargetType == "gateway" {
		resourceId, err = r.Tracker.getInternetServiceId(ctx, clusterScope)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("find internetService for route: %w", err)
		}
	} else {
		natSpec, err := clusterScope.GetNatService(routeSpec.TargetName, routeTableSpec.SubregionName)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("find natService for route: %w", err)
		}
		resourceId, err = r.Tracker.getNatServiceId(ctx, natSpec, clusterScope)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("find natService for route: %w", err)
		}
	}
	log.V(2).Info("Creating route", "destination", destinationIpRange, "resourceId", resourceId)
	_, err = r.Cloud.RouteTable(ctx, *clusterScope).CreateRoute(ctx, destinationIpRange, routeTable.GetRouteTableId(), resourceId, routeSpec.TargetType)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot create route: %w", err)
	}

	return reconcile.Result{}, nil
}

// reconcileRouteTable reconcile the RouteTable and the Route of the cluster.
func (r *OscClusterReconciler) reconcileRouteTable(ctx context.Context, clusterScope *scope.ClusterScope, roles ...infrastructurev1beta1.OscRole) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	if !clusterScope.NeedReconciliation(infrastructurev1beta1.ReconcilerRouteTable) {
		log.V(4).Info("No need for routeTable reconciliation")
		return reconcile.Result{}, nil
	}
	if clusterScope.GetNetwork().UseExisting.Net {
		log.V(3).Info("Using existing routeTables")
		return reconcile.Result{}, nil
	}
	log.V(4).Info("Reconciling routeTables")

	netId, err := r.Tracker.getNetId(ctx, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	svc := r.Cloud.RouteTable(ctx, *clusterScope)
	rtbls, err := svc.GetRouteTablesFromNet(ctx, netId)
	if err != nil {
		return reconcile.Result{}, err
	}
	rtblForSubnet := map[string]*osc.RouteTable{}
	for _, rtbl := range rtbls {
		for _, link := range rtbl.GetLinkRouteTables() {
			rtblForSubnet[link.GetSubnetId()] = &rtbl
		}
	}
	routeTablesSpec := clusterScope.GetRouteTables()
	for _, routeTableSpec := range routeTablesSpec {
		var rtbl *osc.RouteTable
		names := routeTableSpec.Subnets
		if len(names) == 0 {
			names = []string{""}
		}
		for _, name := range names {
			subnetSpec, err := clusterScope.GetSubnet(name, routeTableSpec.Role, routeTableSpec.SubregionName)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("cannot find subnet with name %q role %q: %w", name, routeTableSpec.Role, err)
			}
			if len(roles) > 0 && !clusterScope.SubnetHasRole(subnetSpec, roles[0]) {
				continue
			}
			subnetId, err := r.Tracker.getSubnetId(ctx, subnetSpec, clusterScope)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("cannot get subnet: %w", err)
			}
			switch {
			case rtbl == nil && rtblForSubnet[subnetId] != nil:
				log.V(5).Info("Subnet has a route table", "subnetId", subnetId)
				rtbl = rtblForSubnet[subnetId]
				continue
			case rtbl == nil && rtblForSubnet[subnetId] == nil:
				log.V(2).Info("Creating routetable", "subnetId", subnetId)
				rtbl, err = svc.CreateRouteTable(ctx, netId, clusterScope.GetUID(), routeTableSpec.Name)
				if err != nil {
					return reconcile.Result{}, fmt.Errorf("cannot create routetable: %w", err)
				}
				log.V(2).Info("Created routetable", "routetableId", rtbl.GetRouteTableId())
				r.Recorder.Eventf(clusterScope.OscCluster, corev1.EventTypeNormal, infrastructurev1beta1.RouteTableCreatedReason, "Route table created %v %s", subnetSpec.Roles, subnetSpec.SubregionName)
				fallthrough
			case rtbl != nil && rtblForSubnet[subnetId] == nil:
				log.V(2).Info("Link routetable with subnet", "routeTableId", rtbl.GetRouteTableId(), "subnetId", subnetId)
				_, err := svc.LinkRouteTable(ctx, rtbl.GetRouteTableId(), subnetId)
				if err != nil {
					return reconcile.Result{}, fmt.Errorf("cannot link routetable with net: %w", err)
				}
			}
		}
		if rtbl == nil {
			continue
		}
		for _, routeSpec := range routeTableSpec.Routes {
			_, err = r.reconcileRoute(ctx, clusterScope, routeTableSpec, routeSpec, rtbl)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
	}
	if len(roles) == 0 {
		clusterScope.SetReconciliationGeneration(infrastructurev1beta1.ReconcilerRouteTable)
	}
	return reconcile.Result{}, nil
}

// reconcileDeleteRouteTable reconcile the destruction of the RouteTable of the cluster.
func (r *OscClusterReconciler) reconcileDeleteRouteTable(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	if clusterScope.GetNetwork().UseExisting.Net {
		log.V(3).Info("Not deleting existing routeTables")
		return reconcile.Result{}, nil
	}

	netId, err := r.Tracker.getNetId(ctx, clusterScope)
	switch {
	case errors.Is(err, ErrNoResourceFound) || errors.Is(err, ErrMissingResource):
		log.V(4).Info("The net is already deleted, no route table expected")
		return reconcile.Result{}, nil
	case err != nil:
		return reconcile.Result{}, err
	}
	svc := r.Cloud.RouteTable(ctx, *clusterScope)
	rtbls, err := svc.GetRouteTablesFromNet(ctx, netId)
	if err != nil {
		return reconcile.Result{}, err
	}

	for _, rtbl := range rtbls {
		for _, link := range *rtbl.LinkRouteTables {
			log.V(2).Info("Unlinking route table", "routeTableId", rtbl.GetRouteTableId(), "linkId", link.GetLinkRouteTableId())
			err := svc.UnlinkRouteTable(ctx, link.GetLinkRouteTableId())
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("cannot unlink routeTable: %w", err)
			}
		}
		log.V(2).Info("Deleting routeTable", "routeTableId", rtbl.GetRouteTableId())
		err = svc.DeleteRouteTable(ctx, rtbl.GetRouteTableId())
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot delete routeTable: %w", err)
		}
	}
	return reconcile.Result{}, nil
}
