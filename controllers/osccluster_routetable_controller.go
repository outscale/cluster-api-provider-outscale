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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// getRouteTableResourceId return the RouteTableId from the resourceMap base on RouteTableName (tag name + cluster object uid)
func getRouteTableResourceId(resourceName string, clusterScope *scope.ClusterScope) (string, error) {
	routeTableRef := clusterScope.GetRouteTablesRef()
	if routeTableId, ok := routeTableRef.ResourceMap[resourceName]; ok {
		return routeTableId, nil
	} else {
		return "", fmt.Errorf("%s does not exist", resourceName)
	}
}

// getRouteResourceId return the resourceId from the resourceMap base on routeName (tag name + cluster object uid)
func getRouteResourceId(resourceName string, clusterScope *scope.ClusterScope) (string, error) {
	routeRef := clusterScope.GetRouteRef()
	if routeId, ok := routeRef.ResourceMap[resourceName]; ok {
		return routeId, nil
	} else {
		return "", fmt.Errorf("%s does not exist", resourceName)
	}
}

// checkRouteFormatParameters check Route parameters format (Tag format, cidr format, ..)
func checkRouteTableFormatParameters(clusterScope *scope.ClusterScope) (string, error) {
	var routeTablesSpec []*infrastructurev1beta1.OscRouteTable
	networkSpec := clusterScope.GetNetwork()
	if networkSpec.RouteTables == nil {
		networkSpec.SetRouteTableDefaultValue()
		routeTablesSpec = networkSpec.RouteTables
	} else {
		routeTablesSpec = clusterScope.GetRouteTables()
	}
	for _, routeTableSpec := range routeTablesSpec {
		routeTableName := routeTableSpec.Name + "-" + clusterScope.GetUID()
		routeTableTagName, err := tag.ValidateTagNameValue(routeTableName)
		if err != nil {
			return routeTableTagName, err
		}
	}
	return "", nil
}

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
			destinationIpRange := routeSpec.Destination
			err = infrastructurev1beta1.ValidateCidr(destinationIpRange)
			if err != nil {
				return routeTagName, err
			}
		}
	}
	return "", nil
}

// checkRouteTableSubnetOscAssociateResourceName check that RouteTable Subnet dependencies tag name in both resource configuration are the same.
func checkRouteTableSubnetOscAssociateResourceName(clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	routeTablesSpec := clusterScope.GetRouteTables()
	resourceNameList = resourceNameList[:0]
	subnetsSpec := clusterScope.GetSubnet()
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

// checkRouteTableOscDuplicateName check that there are not the same name for RouteTable.
func checkRouteTableOscDuplicateName(clusterScope *scope.ClusterScope) error {
	return utils.CheckDuplicates(clusterScope.GetRouteTables(), func(rt *infrastructurev1beta1.OscRouteTable) string {
		return rt.Name
	})
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
func reconcileRoute(ctx context.Context, clusterScope *scope.ClusterScope, routeSpec infrastructurev1beta1.OscRoute, routeTableName string, routeTableSvc security.OscRouteTableInterface) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	routeRef := clusterScope.GetRouteRef()
	routeTablesRef := clusterScope.GetRouteTablesRef()
	resourceName := routeSpec.TargetName + "-" + clusterScope.GetUID()
	resourceType := routeSpec.TargetType
	routeName := routeSpec.Name + "-" + clusterScope.GetUID()
	if len(routeRef.ResourceMap) == 0 {
		routeRef.ResourceMap = make(map[string]string)
	}
	var resourceId string
	var err error
	if resourceType == "gateway" {
		resourceId, err = getInternetServiceResourceId(resourceName, clusterScope)
		if err != nil {
			return reconcile.Result{}, err
		}
	} else {
		resourceId, err = getNatResourceId(resourceName, clusterScope)
		if err != nil {
			return reconcile.Result{}, err
		}
	}
	destinationIpRange := routeSpec.Destination
	associateRouteTableId := routeTablesRef.ResourceMap[routeTableName]
	log.V(4).Info("Checking route", "routename", routeName)
	routeTableFromRoute, err := routeTableSvc.GetRouteTableFromRoute(ctx, associateRouteTableId, resourceId, resourceType)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot get route table: %w", err)
	}
	if routeTableFromRoute == nil {
		log.V(2).Info("Creating route", "routeName", routeName)
		routeTableFromRoute, err = routeTableSvc.CreateRoute(ctx, destinationIpRange, routeTablesRef.ResourceMap[routeTableName], resourceId, resourceType)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot create route: %w", err)
		}
	}

	routeRef.ResourceMap[routeName] = routeTableFromRoute.GetRouteTableId()
	return reconcile.Result{}, nil
}

// reconcileRoute reconcile the RouteTable and the Route of the cluster.
func reconcileDeleteRoute(ctx context.Context, clusterScope *scope.ClusterScope, routeSpec infrastructurev1beta1.OscRoute, routeTableName string, routeTableSvc security.OscRouteTableInterface) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	routeName := routeSpec.Name + "-" + clusterScope.GetUID()
	routeTableId, err := getRouteResourceId(routeName, clusterScope)
	if err != nil {
		log.V(3).Info("No route table found, skipping route deletion")
		return reconcile.Result{}, nil //nolint: nilerr
	}

	log.V(2).Info("Deleting route", "routeName", routeName)
	err = routeTableSvc.DeleteRoute(ctx, routeSpec.Destination, routeTableId)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot delete route: %w", err)
	}
	return reconcile.Result{}, nil
}

// reconcileRouteTable reconcile the RouteTable and the Route of the cluster.
func reconcileRouteTable(ctx context.Context, clusterScope *scope.ClusterScope, routeTableSvc security.OscRouteTableInterface, tagSvc tag.OscTagInterface) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	routeTablesSpec := clusterScope.GetRouteTables()
	routeTablesRef := clusterScope.GetRouteTablesRef()
	linkRouteTablesRef := clusterScope.GetLinkRouteTablesRef()

	netSpec := clusterScope.GetNet()
	netSpec.SetDefaultValue()
	netName := netSpec.Name + "-" + clusterScope.GetUID()
	netId, err := getNetResourceId(netName, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	networkSpec := clusterScope.GetNetwork()
	clusterName := networkSpec.ClusterName + "-" + clusterScope.GetUID()

	log.V(4).Info("List routetables in net", "netId", netId)
	routeTableIds, err := routeTableSvc.GetRouteTableIdsFromNetIds(ctx, netId)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("list route tables: %w", err)
	}
	log.V(4).Info("Number of routeTables", "routeTableLength", routeTablesSpec)
	for _, routeTableSpec := range routeTablesSpec {
		routeTableName := routeTableSpec.Name + "-" + clusterScope.GetUID()
		log.V(4).Info("Check if routeTable exists in net", "routeTableName", routeTableName)

		tagKey := "Name"
		tagValue := routeTableName
		tag, err := tagSvc.ReadTag(ctx, tagKey, tagValue)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot get tag: %w", err)
		}
		if len(routeTablesRef.ResourceMap) == 0 {
			routeTablesRef.ResourceMap = make(map[string]string)
		}
		if len(linkRouteTablesRef) == 0 {
			linkRouteTablesRef = make(map[string][]string)
		}
		if routeTableSpec.ResourceId != "" {
			routeTablesRef.ResourceMap[routeTableName] = routeTableSpec.ResourceId
		}
		_, resourceMapExist := routeTablesRef.ResourceMap[routeTableName]
		if resourceMapExist {
			routeTableSpec.ResourceId = routeTablesRef.ResourceMap[routeTableName]
		}

		routeTableId := routeTablesRef.ResourceMap[routeTableName]
		natRouteTable := false

		if !slices.Contains(routeTableIds, routeTableId) && tag == nil {
			routesSpec := clusterScope.GetRoute(routeTableSpec.Name)
			log.V(4).Info("Number of routes", "routeLength", len(routesSpec))
			for _, routeSpec := range routesSpec {
				resourceType := routeSpec.TargetType
				log.V(4).Info("Get resourceType", "ResourceType", resourceType)
				if resourceType == "nat" {
					natServiceRef := clusterScope.GetNatServiceRef()
					log.V(4).Info("Get Nat", "Nat", natServiceRef.ResourceMap)
					if len(natServiceRef.ResourceMap) == 0 {
						natRouteTable = true
					}
				}
			}
			if natRouteTable {
				continue
			}
			log.V(2).Info("Creating routetable", "routeTableName", routeTableName)
			routeTable, err := routeTableSvc.CreateRouteTable(ctx, netId, clusterName, routeTableName)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("cannot create routetable: %w", err)
			}

			routeTableId := routeTable.GetRouteTableId()
			routeTablesRef.ResourceMap[routeTableName] = routeTableId
			routeTableSpec.ResourceId = routeTableId
			subnetsSpec := routeTableSpec.Subnets
			linkRouteTableIdArray := make([]string, 0)
			for _, subnet := range subnetsSpec {
				subnetName := subnet + "-" + clusterScope.GetUID()
				subnetId, err := getSubnetResourceId(subnetName, clusterScope)
				if err != nil {
					return reconcile.Result{}, err
				}
				log.V(2).Info("Link routetable with subnet", "routeTableName", routeTableName)
				linkRouteTableId, err := routeTableSvc.LinkRouteTable(ctx, routeTableId, subnetId)
				if err != nil {
					return reconcile.Result{}, fmt.Errorf("cannot link routetable with net: %w", err)
				}
				linkRouteTableIdArray = append(linkRouteTableIdArray, linkRouteTableId)
			}

			linkRouteTablesRef[routeTableName] = linkRouteTableIdArray
			clusterScope.SetLinkRouteTablesRef(linkRouteTablesRef)
			for _, routeSpec := range routesSpec {
				log.V(2).Info("Create route for routetable", "routeTableName", routeTableName)
				_, err = reconcileRoute(ctx, clusterScope, routeSpec, routeTableName, routeTableSvc)
				if err != nil {
					return reconcile.Result{}, err
				}
			}
		}
	}
	return reconcile.Result{}, nil
}

// reconcileDeleteRouteTable reconcile the destruction of the RouteTable of the cluster.
func reconcileDeleteRouteTable(ctx context.Context, clusterScope *scope.ClusterScope, routeTableSvc security.OscRouteTableInterface) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	var routeTablesSpec []*infrastructurev1beta1.OscRouteTable
	networkSpec := clusterScope.GetNetwork()

	if networkSpec.RouteTables == nil {
		networkSpec.SetRouteTableDefaultValue()
		routeTablesSpec = networkSpec.RouteTables
	} else {
		routeTablesSpec = clusterScope.GetRouteTables()
	}
	routeTablesRef := clusterScope.GetRouteTablesRef()
	linkRouteTablesRef := clusterScope.GetLinkRouteTablesRef()

	netSpec := clusterScope.GetNet()
	netSpec.SetDefaultValue()
	netName := netSpec.Name + "-" + clusterScope.GetUID()
	netId, err := getNetResourceId(netName, clusterScope)
	if err != nil {
		log.V(3).Info("No net found, skipping route table deletion")
		return reconcile.Result{}, nil //nolint: nilerr
	}

	routeTableIds, err := routeTableSvc.GetRouteTableIdsFromNetIds(ctx, netId)
	if err != nil {
		return reconcile.Result{}, err
	}

	log.V(4).Info("Number of routeTable", "routeTable", len(routeTablesSpec))
	for _, routeTableSpec := range routeTablesSpec {
		routeTableName := routeTableSpec.Name + "-" + clusterScope.GetUID()
		routeTableId := routeTablesRef.ResourceMap[routeTableName]
		log.V(2).Info("Get routetable", "routeTable", routeTableName)
		if !slices.Contains(routeTableIds, routeTableId) {
			log.V(2).Info("routeTable is already deleted", "routeTableName", routeTableName)
			return reconcile.Result{}, nil
		}
		routesSpec := clusterScope.GetRoute(routeTableSpec.Name)
		log.V(4).Info("Number of route", "routeLength", len(routesSpec))
		for _, routeSpec := range routesSpec {
			_, err = reconcileDeleteRoute(ctx, clusterScope, routeSpec, routeTableName, routeTableSvc)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		log.V(4).Info("Get link", "link", len(linkRouteTablesRef))

		for _, linkRouteTableId := range linkRouteTablesRef[routeTableName] {
			log.V(2).Info("Unlink routeTable", "routeTableName", routeTableName)
			err = routeTableSvc.UnlinkRouteTable(ctx, linkRouteTableId)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("cannot unlink routeTable: %w", err)
			}
		}

		log.V(2).Info("Deleting routeTable", "routeTableName", routeTableName)
		err = routeTableSvc.DeleteRouteTable(ctx, routeTablesRef.ResourceMap[routeTableName])
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot delete routeTable: %w", err)
		}
	}
	return reconcile.Result{}, nil
}
