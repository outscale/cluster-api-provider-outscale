package controllers

import (
	"context"
	"fmt"

	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/net"
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
	clusterScope.Info("Check Route table parameters")
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
	clusterScope.Info("Check Route parameters")
	var routeTablesSpec []*infrastructurev1beta1.OscRouteTable
	routeTablesSpec = clusterScope.GetRouteTables()
	for _, routeTableSpec := range routeTablesSpec {
		routesSpec := clusterScope.GetRoute(routeTableSpec.Name)
		for _, routeSpec := range *routesSpec {
			routeName := routeSpec.Name + "-" + clusterScope.GetUID()
			routeTagName, err := tag.ValidateTagNameValue(routeName)
			if err != nil {
				return routeTagName, err
			}
			clusterScope.Info("Check route destination IpRange parameters")
			destinationIpRange := routeSpec.Destination
			_, err = net.ValidateCidr(destinationIpRange)
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
	clusterScope.Info("check match subnet with route table service")
	routeTablesSpec := clusterScope.GetRouteTables()
	resourceNameList = resourceNameList[:0]
	subnetsSpec := clusterScope.GetSubnet()
	for _, subnetSpec := range subnetsSpec {
		subnetName := subnetSpec.Name + "-" + clusterScope.GetUID()
		resourceNameList = append(resourceNameList, subnetName)
	}
	for _, routeTableSpec := range routeTablesSpec {
		routeTableSubnetName := routeTableSpec.SubnetName + "-" + clusterScope.GetUID()
		checkOscAssociate := Contains(resourceNameList, routeTableSubnetName)
		if checkOscAssociate {
			return nil
		} else {
			return fmt.Errorf("%s subnet dooes not exist in routeTable", routeTableSubnetName)
		}
	}
	return nil
}

// checkRouteTableOscDuplicateName check that there are not the same name for RouteTable.
func checkRouteTableOscDuplicateName(clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	clusterScope.Info("check unique routetable")
	routeTablesSpec := clusterScope.GetRouteTables()
	for _, routeTableSpec := range routeTablesSpec {
		resourceNameList = append(resourceNameList, routeTableSpec.Name)
	}
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
	clusterScope.Info("check unique route")
	routeTablesSpec := clusterScope.GetRouteTables()
	for _, routeTableSpec := range routeTablesSpec {
		routesSpec := clusterScope.GetRoute(routeTableSpec.Name)
		for _, routeSpec := range *routesSpec {
			resourceNameList = append(resourceNameList, routeSpec.Name)
		}
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
func reconcileRoute(ctx context.Context, clusterScope *scope.ClusterScope, routeSpec infrastructurev1beta1.OscRoute, routeTableName string, routeTableSvc security.OscRouteTableInterface) (reconcile.Result, error) {
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
	clusterScope.Info("check if the desired route exist", "routename", routeName)
	routeTableFromRoute, err := routeTableSvc.GetRouteTableFromRoute(associateRouteTableId, resourceId, resourceType)
	if err != nil {
		return reconcile.Result{}, err
	}
	if routeTableFromRoute == nil {
		clusterScope.Info("### Create Route ###", "Route", resourceId)
		clusterScope.Info("Create the desired route", "routeName", routeName)
		routeTableFromRoute, err = routeTableSvc.CreateRoute(destinationIpRange, routeTablesRef.ResourceMap[routeTableName], resourceId, resourceType)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not create route for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
		}
	}

	routeRef.ResourceMap[routeName] = routeTableFromRoute.GetRouteTableId()
	return reconcile.Result{}, nil

}

// reconcileRoute reconcile the RouteTable and the Route of the cluster.
func reconcileDeleteRoute(ctx context.Context, clusterScope *scope.ClusterScope, routeSpec infrastructurev1beta1.OscRoute, routeTableName string, routeTableSvc security.OscRouteTableInterface) (reconcile.Result, error) {
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

	clusterScope.Info("Check if the desired route still exist", "routeName", routeName)
	routeTableFromRoute, err := routeTableSvc.GetRouteTableFromRoute(associateRouteTableId, resourceId, resourceType)
	if err != nil {
		return reconcile.Result{}, err
	}
	if routeTableFromRoute == nil {
		clusterScope.Info("the desired route does not exist anymore", "routeName", routeName)
		controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
		return reconcile.Result{}, nil
	}
	clusterScope.Info("Delete Route")
	clusterScope.Info("### delete destinationIpRange###", "routeTable", destinationIpRange)

	clusterScope.Info("Delete the desired route", "routeName", routeName)
	err = routeTableSvc.DeleteRoute(destinationIpRange, routeTableId)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("%w Can not delete route for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
	}
	return reconcile.Result{}, nil

}

// reconcileRouteTable reconcile the RouteTable and the Route of the cluster.
func reconcileRouteTable(ctx context.Context, clusterScope *scope.ClusterScope, routeTableSvc security.OscRouteTableInterface) (reconcile.Result, error) {

	clusterScope.Info("Create RouteTable")
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

	clusterScope.Info("Get list of all desired routetable in net", "netId", netId)
	routeTableIds, err := routeTableSvc.GetRouteTableIdsFromNetIds(netId)
	if err != nil {
		return reconcile.Result{}, err
	}
	for _, routeTableSpec := range routeTablesSpec {
		routeTableName := routeTableSpec.Name + "-" + clusterScope.GetUID()
		clusterScope.Info("Check if the desired routeTable existin net", "routeTableName", routeTableName)
		clusterScope.Info("### Get routeTable Id ###", "routeTable", routeTablesRef.ResourceMap)
		subnetName := routeTableSpec.SubnetName + "-" + clusterScope.GetUID()
		subnetId, err := getSubnetResourceId(subnetName, clusterScope)
		if err != nil {
			return reconcile.Result{}, err
		}

		if len(routeTablesRef.ResourceMap) == 0 {
			routeTablesRef.ResourceMap = make(map[string]string)
		}
		if len(linkRouteTablesRef.ResourceMap) == 0 {
			linkRouteTablesRef.ResourceMap = make(map[string]string)
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

		if !Contains(routeTableIds, routeTableId) {
			clusterScope.Info("check Nat RouteTable")
			routesSpec := clusterScope.GetRoute(routeTableSpec.Name)

			for _, routeSpec := range *routesSpec {
				resourceType := routeSpec.TargetType
				if resourceType == "nat" {
					natServiceRef := clusterScope.GetNatServiceRef()
					clusterScope.Info("### Get Nat ###", "Nat", natServiceRef.ResourceMap)
					if len(natServiceRef.ResourceMap) == 0 {
						natRouteTable = true
					}
				}
			}
			if natRouteTable {
				continue
			}
			clusterScope.Info("Create the desired routetable", "routeTableName", routeTableName)
			routeTable, err := routeTableSvc.CreateRouteTable(netId, routeTableName)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("%w Can not create routetable for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
			}

			routeTableId := routeTable.GetRouteTableId()
			clusterScope.Info("Link the desired routetable with subnet", "routeTableName", routeTableName)
			linkRouteTableId, err := routeTableSvc.LinkRouteTable(routeTableId, subnetId)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("%w Can not link routetable with net for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
			}
			clusterScope.Info("### Get routeTable ###", "routeTable", routeTable)
			routeTablesRef.ResourceMap[routeTableName] = routeTableId
			routeTableSpec.ResourceId = routeTableId
			linkRouteTablesRef.ResourceMap[routeTableName] = linkRouteTableId

			for _, routeSpec := range *routesSpec {
				clusterScope.Info("Set route")
				clusterScope.Info("Create route for the desired routetable", "routeTableName", routeTableName)
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
	clusterScope.Info("Delete RouteTable")
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
		return reconcile.Result{}, err
	}

	routeTableIds, err := routeTableSvc.GetRouteTableIdsFromNetIds(netId)
	if err != nil {
		return reconcile.Result{}, err
	}

	osccluster := clusterScope.OscCluster
	for _, routeTableSpec := range routeTablesSpec {
		routeTableName := routeTableSpec.Name + "-" + clusterScope.GetUID()
		routeTableId := routeTablesRef.ResourceMap[routeTableName]
		clusterScope.Info("### delete routeTable Id ###", "routeTable", routeTableId)

		if !Contains(routeTableIds, routeTableId) {
			clusterScope.Info("the desired routeTable does no exist anymore", "routeTableName", routeTableName)
			controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
			return reconcile.Result{}, nil
		}
		clusterScope.Info("Remove Route")
		routesSpec := clusterScope.GetRoute(routeTableSpec.Name)
		for _, routeSpec := range *routesSpec {
			_, err = reconcileDeleteRoute(ctx, clusterScope, routeSpec, routeTableName, routeTableSvc)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		clusterScope.Info("Unlink the desired routeTable", "routeTableName", routeTableName)
		err = routeTableSvc.UnlinkRouteTable(linkRouteTablesRef.ResourceMap[routeTableName])
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not unlink routeTable for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
		}

		clusterScope.Info("delete the desired routeTable", "routeTableName", routeTableName)
		err = routeTableSvc.DeleteRouteTable(routeTablesRef.ResourceMap[routeTableName])
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not delete routeTable for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
		}
	}
	return reconcile.Result{}, nil
}
