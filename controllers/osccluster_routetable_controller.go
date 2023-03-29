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

	infrastructurev1beta2 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta2"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/security"
	tag "github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
	var routeTablesSpec []*infrastructurev1beta2.OscRouteTable
	networkSpec := clusterScope.GetNetwork()
	if networkSpec.RouteTables == nil {
		networkSpec.SetRouteTableDefaultValue()
		routeTablesSpec = networkSpec.RouteTables
	} else {
		routeTablesSpec = clusterScope.GetRouteTables()
	}
	for _, routeTableSpec := range routeTablesSpec {
		routeTableName := routeTableSpec.Name + "-" + clusterScope.GetUID()
		clusterScope.V(2).Info("Check Route table parameters")
		routeTableTagName, err := tag.ValidateTagNameValue(routeTableName)
		if err != nil {
			return routeTableTagName, err
		}
	}
	return "", nil
}

// checkRouteFormatParameters check Route parameters format (Tag format, cidr format, ..)
func checkRouteFormatParameters(clusterScope *scope.ClusterScope) (string, error) {
	var routeTablesSpec []*infrastructurev1beta2.OscRouteTable
	routeTablesSpec = clusterScope.GetRouteTables()
	for _, routeTableSpec := range routeTablesSpec {
		routesSpec := clusterScope.GetRoute(routeTableSpec.Name)
		for _, routeSpec := range *routesSpec {
			routeName := routeSpec.Name + "-" + clusterScope.GetUID()
			routeTagName, err := tag.ValidateTagNameValue(routeName)
			if err != nil {
				return routeTagName, err
			}
			clusterScope.V(2).Info("Check route destination IpRange parameters")
			destinationIpRange := routeSpec.Destination
			_, err = infrastructurev1beta2.ValidateCidr(destinationIpRange)
			if err != nil {
				return routeTagName, err
			}
		}
	}
	return "", nil
}

// checkRouteTableSubnetOscAssociateResourceName check that RouteTable Subnet dependancies tag name in both resource configuration are the same.
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
			clusterScope.V(2).Info("Check the desired subnet", "routeTableSubnet", routeTableSubnet)
			checkOscAssociate := Contains(resourceNameList, routeTableSubnetName)
			if checkOscAssociate {
				return nil
			} else {
				return fmt.Errorf("%s subnet does not exist in routeTable", routeTableSubnetName)
			}
		}
	}
	return nil
}

// checkRouteTableOscDuplicateName check that there are not the same name for RouteTable.
func checkRouteTableOscDuplicateName(clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	routeTablesSpec := clusterScope.GetRouteTables()
	for _, routeTableSpec := range routeTablesSpec {
		resourceNameList = append(resourceNameList, routeTableSpec.Name)
	}
	clusterScope.V(2).Info("Check unique routetable")
	duplicateResourceErr := alertDuplicate(resourceNameList)
	if duplicateResourceErr != nil {
		return duplicateResourceErr
	} else {
		return nil
	}
}

// checkRouteOscDuplicateName check that there are not the same name for route.
func checkRouteOscDuplicateName(clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	routeTablesSpec := clusterScope.GetRouteTables()
	for _, routeTableSpec := range routeTablesSpec {
		routesSpec := clusterScope.GetRoute(routeTableSpec.Name)
		for _, routeSpec := range *routesSpec {
			resourceNameList = append(resourceNameList, routeSpec.Name)
		}
		clusterScope.V(2).Info("Check unique route")
		duplicateResourceErr := alertDuplicate(resourceNameList)
		if duplicateResourceErr != nil {
			return duplicateResourceErr
		} else {
			return nil
		}
	}
	return nil
}

// reconcileRoute reconcile the RouteTable and the Route of the cluster.
func reconcileRoute(ctx context.Context, clusterScope *scope.ClusterScope, routeSpec infrastructurev1beta2.OscRoute, routeTableName string, routeTableSvc security.OscRouteTableInterface) (reconcile.Result, error) {
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
	clusterScope.V(2).Info("Check if the desired route exist", "routename", routeName)
	routeTableFromRoute, err := routeTableSvc.GetRouteTableFromRoute(associateRouteTableId, resourceId, resourceType)
	if err != nil {
		return reconcile.Result{}, err
	}
	if routeTableFromRoute == nil {
		clusterScope.V(4).Info("Create Route", "Route", resourceId)
		clusterScope.V(2).Info("Create the desired route", "routeName", routeName)
		routeTableFromRoute, err = routeTableSvc.CreateRoute(destinationIpRange, routeTablesRef.ResourceMap[routeTableName], resourceId, resourceType)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not create route for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
		}
	}

	routeRef.ResourceMap[routeName] = routeTableFromRoute.GetRouteTableId()
	return reconcile.Result{}, nil

}

// reconcileRoute reconcile the RouteTable and the Route of the cluster.
func reconcileDeleteRoute(ctx context.Context, clusterScope *scope.ClusterScope, routeSpec infrastructurev1beta2.OscRoute, routeTableName string, routeTableSvc security.OscRouteTableInterface) (reconcile.Result, error) {
	osccluster := clusterScope.OscCluster

	routeTablesRef := clusterScope.GetRouteTablesRef()

	resourceName := routeSpec.TargetName + "-" + clusterScope.GetUID()
	resourceType := routeSpec.TargetType
	routeName := routeSpec.Name + "-" + clusterScope.GetUID()
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
	routeTableId, err := getRouteResourceId(routeName, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	destinationIpRange := routeSpec.Destination
	associateRouteTableId := routeTablesRef.ResourceMap[routeTableName]

	clusterScope.V(2).Info("Check if the desired route still exist", "routeName", routeName)
	routeTableFromRoute, err := routeTableSvc.GetRouteTableFromRoute(associateRouteTableId, resourceId, resourceType)
	if err != nil {
		return reconcile.Result{}, err
	}
	if routeTableFromRoute == nil {
		clusterScope.V(2).Info("The desired route does not exist anymore", "routeName", routeName)
		controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
		return reconcile.Result{}, nil
	}
	clusterScope.V(4).Info("Delete destinationIpRange", "routeTable", destinationIpRange)
	clusterScope.V(4).Info("Delete the desired route", "routeName", routeName)
	err = routeTableSvc.DeleteRoute(destinationIpRange, routeTableId)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("%w Can not delete route for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
	}
	return reconcile.Result{}, nil

}

// reconcileRouteTable reconcile the RouteTable and the Route of the cluster.
func reconcileRouteTable(ctx context.Context, clusterScope *scope.ClusterScope, routeTableSvc security.OscRouteTableInterface, tagSvc tag.OscTagInterface) (reconcile.Result, error) {

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

	clusterScope.V(4).Info("Get list of all desired routetable in net", "netId", netId)
	routeTableIds, err := routeTableSvc.GetRouteTableIdsFromNetIds(netId)
	if err != nil {
		return reconcile.Result{}, err
	}
	clusterScope.V(4).Info("Number of routeTable", "routeTableLength", routeTablesSpec)
	for _, routeTableSpec := range routeTablesSpec {
		routeTableName := routeTableSpec.Name + "-" + clusterScope.GetUID()
		clusterScope.V(2).Info("Check if the desired routeTable exist in net", "routeTableName", routeTableName)
		clusterScope.V(4).Info("Get routeTable Id", "routeTable", routeTablesRef.ResourceMap)

		tagKey := "Name"
		tagValue := routeTableName
		tag, err := tagSvc.ReadTag(tagKey, tagValue)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not get tag for OscCluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
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
		var natRouteTable bool = false

		if !Contains(routeTableIds, routeTableId) && tag == nil {
			clusterScope.V(2).Info("Check Nat RouteTable")
			routesSpec := clusterScope.GetRoute(routeTableSpec.Name)
			clusterScope.V(4).Info("Number of route", "routeLength", len(*routesSpec))
			for _, routeSpec := range *routesSpec {
				resourceType := routeSpec.TargetType
				clusterScope.V(4).Info("Get resourceType", "ResourceType", resourceType)
				if resourceType == "nat" {
					natServiceRef := clusterScope.GetNatServiceRef()
					clusterScope.V(4).Info("Get Nat", "Nat", natServiceRef.ResourceMap)
					if len(natServiceRef.ResourceMap) == 0 {
						natRouteTable = true
					}
				}
			}
			if natRouteTable {
				continue
			}
			clusterScope.V(4).Info("Create the desired routetable", "routeTableName", routeTableName)
			routeTable, err := routeTableSvc.CreateRouteTable(netId, clusterName, routeTableName)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("%w Can not create routetable for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
			}

			routeTableId := routeTable.GetRouteTableId()
			clusterScope.V(4).Info("Get routeTable", "routeTable", routeTable)
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
				clusterScope.V(2).Info("Link the desired routetable with subnet", "routeTableName", routeTableName)

				linkRouteTableId, err := routeTableSvc.LinkRouteTable(routeTableId, subnetId)
				if err != nil {
					return reconcile.Result{}, fmt.Errorf("%w Can not link routetable with net for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
				}
				linkRouteTableIdArray = append(linkRouteTableIdArray, linkRouteTableId)
			}

			linkRouteTablesRef[routeTableName] = linkRouteTableIdArray
			clusterScope.SetLinkRouteTablesRef(linkRouteTablesRef)
			for _, routeSpec := range *routesSpec {
				clusterScope.V(2).Info("Create route for the desired routetable", "routeTableName", routeTableName)
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
	var routeTablesSpec []*infrastructurev1beta2.OscRouteTable
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
		return reconcile.Result{}, err
	}

	routeTableIds, err := routeTableSvc.GetRouteTableIdsFromNetIds(netId)
	if err != nil {
		return reconcile.Result{}, err
	}

	osccluster := clusterScope.OscCluster
	clusterScope.V(4).Info("Number of routeTable", "routeTable", len(routeTablesSpec))
	for _, routeTableSpec := range routeTablesSpec {
		routeTableName := routeTableSpec.Name + "-" + clusterScope.GetUID()
		routeTableId := routeTablesRef.ResourceMap[routeTableName]
		clusterScope.V(2).Info("Get routetable", "routeTable", routeTableName)
		if !Contains(routeTableIds, routeTableId) {
			clusterScope.V(2).Info("The desired routeTable does no exist anymore", "routeTableName", routeTableName)
			controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
			return reconcile.Result{}, nil
		}
		routesSpec := clusterScope.GetRoute(routeTableSpec.Name)
		clusterScope.V(4).Info("Number of route", "routeLength", len(*routesSpec))
		for _, routeSpec := range *routesSpec {
			_, err = reconcileDeleteRoute(ctx, clusterScope, routeSpec, routeTableName, routeTableSvc)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		clusterScope.V(4).Info("Get link", "link", len(linkRouteTablesRef))

		for _, linkRouteTableId := range linkRouteTablesRef[routeTableName] {
			clusterScope.V(2).Info("Unlink the desired routeTable", "routeTableName", routeTableName)
			err = routeTableSvc.UnlinkRouteTable(linkRouteTableId)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("%w Can not unlink routeTable for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
			}
		}

		clusterScope.V(2).Info("Delete the desired routeTable", "routeTableName", routeTableName)
		err = routeTableSvc.DeleteRouteTable(routeTablesRef.ResourceMap[routeTableName])
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not delete routeTable for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
		}
	}
	return reconcile.Result{}, nil
}
