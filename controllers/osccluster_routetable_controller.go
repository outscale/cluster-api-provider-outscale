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
	"fmt"
	"slices"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/security"
	tag "github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	"github.com/outscale/cluster-api-provider-outscale/cloud/utils"
	"github.com/outscale/osc-sdk-go/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// checkRouteFormatParameters check Route parameters format (Tag format, cidr format, ..)
func checkRouteFormatParameters(clusterScope *scope.ClusterScope) (string, error) {
	routeTablesSpec := clusterScope.GetRouteTables()
	for _, routeTableSpec := range routeTablesSpec {
		routesSpec := clusterScope.GetRoute(routeTableSpec.Name)
		for _, routeSpec := range routesSpec {
			routeName := routeSpec.Name + "-" + clusterScope.GetUID()
			routeTagName, err := tag.ValidateTagNameValue(routeName)
			if err != nil {
				return routeTagName, err
			}
			// FIXME
			// destinationIpRange := routeSpec.Destination
			// err = infrastructurev1beta1.ValidateCidr(destinationIpRange)
			// if err != nil {
			// 	return routeTagName, err
			// }
		}
	}
	return "", nil
}

// checkRouteTableSubnetOscAssociateResourceName check that RouteTable Subnet dependencies tag name in both resource configuration are the same.
func checkRouteTableSubnetOscAssociateResourceName(clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	routeTablesSpec := clusterScope.GetRouteTables()
	resourceNameList = resourceNameList[:0]
	subnetsSpec := clusterScope.GetSubnets()
	for _, subnetSpec := range subnetsSpec {
		subnetName := subnetSpec.Name + "-" + clusterScope.GetUID()
		resourceNameList = append(resourceNameList, subnetName)
	}
	for _, routeTableSpec := range routeTablesSpec {
		routeTableSubnetsSpec := routeTableSpec.Subnets
		for _, routeTableSubnet := range routeTableSubnetsSpec {
			routeTableSubnetName := routeTableSubnet + "-" + clusterScope.GetUID()
			checkOscAssociate := slices.Contains(resourceNameList, routeTableSubnetName)
			if checkOscAssociate {
				return nil
			} else {
				return fmt.Errorf("subnet %s does not exist in routeTable", routeTableSubnetName)
			}
		}
	}
	return nil
}

// checkRouteOscDuplicateName check that there are not the same name for route.
func checkRouteOscDuplicateName(clusterScope *scope.ClusterScope) error {
	routeTablesSpec := clusterScope.GetRouteTables()
	for _, routeTableSpec := range routeTablesSpec {
		err := utils.CheckDuplicates(clusterScope.GetRoute(routeTableSpec.Name), func(r infrastructurev1beta1.OscRoute) string {
			return r.Name
		})
		if err != nil {
			return err
		}
	}
	return nil
}

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
func (r *OscClusterReconciler) reconcileRouteTable(ctx context.Context, clusterScope *scope.ClusterScope, routeTableSvc security.OscRouteTableInterface, tagSvc tag.OscTagInterface) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	if !clusterScope.NeedReconciliation(infrastructurev1beta1.ReconcilerRouteTable) {
		log.V(4).Info("No need for routeTable reconciliation")
		return reconcile.Result{}, nil
	}
	netSpec := clusterScope.GetNet()
	if netSpec.UseExisting {
		log.V(3).Info("Using existing routeTables")
		return reconcile.Result{}, nil
	}

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
				return reconcile.Result{}, fmt.Errorf("cannot find subnet for routeTable: %w", err)
			}
			subnetId, err := r.Tracker.getSubnetId(ctx, subnetSpec, clusterScope)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("cannot find subnet for routeTable: %w", err)
			}

			switch {
			case rtbl == nil && rtblForSubnet[subnetId] != nil:
				log.V(5).Info("routetable exists", "subnetId", subnetId)
				rtbl = rtblForSubnet[subnetId]
				continue
			case rtbl == nil && rtblForSubnet[subnetId] == nil:
				log.V(2).Info("Creating routetable")
				rtbl, err = svc.CreateRouteTable(ctx, netId, clusterScope.GetName(), routeTableSpec.Name)
				if err != nil {
					return reconcile.Result{}, fmt.Errorf("cannot create routetable: %w", err)
				}
				log.V(2).Info("Created routetable", "routetableId", rtbl.GetRouteTableId())
				fallthrough
			case rtbl != nil && rtblForSubnet[subnetId] == nil:
				log.V(2).Info("Link routetable with subnet", "routeTableId", rtbl.GetRouteTableId(), "subnetId", subnetId)
				_, err := svc.LinkRouteTable(ctx, rtbl.GetRouteTableId(), subnetId)
				if err != nil {
					return reconcile.Result{}, fmt.Errorf("cannot link routetable with net: %w", err)
				}
			}
		}
		for _, routeSpec := range routeTableSpec.Routes {
			_, err = r.reconcileRoute(ctx, clusterScope, routeTableSpec, routeSpec, rtbl)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
	}
	clusterScope.SetReconciliationGeneration(infrastructurev1beta1.ReconcilerRouteTable)
	return reconcile.Result{}, nil
}

// reconcileDeleteRouteTable reconcile the destruction of the RouteTable of the cluster.
func (r *OscClusterReconciler) reconcileDeleteRouteTable(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	netSpec := clusterScope.GetNet()
	if netSpec.UseExisting {
		log.V(3).Info("Not deleting existing routeTables")
		return reconcile.Result{}, nil
	}

	netId, err := r.Tracker.getNetId(ctx, clusterScope)
	if err != nil {
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
