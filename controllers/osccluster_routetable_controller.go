package controllers

import (
	"context"
	"fmt"

	infrastructurev1beta1 "github.com/outscale-vbr/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/services/net"
	"github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/services/security"
	tag "github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/tag"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// GetRouteTableResourceId return the RouteTableId from the resourceMap base on RouteTableName (tag name + cluster object uid)
func GetRouteTableResourceId(resourceName string, clusterScope *scope.ClusterScope) (string, error) {
	routeTableRef := clusterScope.GetRouteTablesRef()
	if routeTableId, ok := routeTableRef.ResourceMap[resourceName]; ok {
		return routeTableId, nil
	} else {
		return "", fmt.Errorf("%s is not exist", resourceName)
	}
}

// GetRouteResourceId return the resourceId from the resourceMap base on routeName (tag name + cluster object uid)
func GetRouteResourceId(resourceName string, clusterScope *scope.ClusterScope) (string, error) {
	routeRef := clusterScope.GetRouteRef()
	if routeId, ok := routeRef.ResourceMap[resourceName]; ok {
		return routeId, nil
	} else {
		return "", fmt.Errorf("%s is not exist", resourceName)
	}
}

// CheckRouteFormatParameters check Route parameters format (Tag format, cidr format, ..)
func CheckRouteTableFormatParameters(clusterScope *scope.ClusterScope) (string, error) {
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

// CheckRouteFormatParameters check Route parameters format (Tag format, cidr format, ..)
func CheckRouteFormatParameters(clusterScope *scope.ClusterScope) (string, error) {
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

// CheckRouteTableSubnetOscAssociateResourceName check that RouteTable Subnet dependancies tag name in both resource configuration are the same.
func CheckRouteTableSubnetOscAssociateResourceName(clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	clusterScope.Info("check match subnet with route table service")
	var routeTablesSpec []*infrastructurev1beta1.OscRouteTable
	routeTablesSpec = clusterScope.GetRouteTables()
	resourceNameList = resourceNameList[:0]
	var subnetsSpec []*infrastructurev1beta1.OscSubnet
	networkSpec := clusterScope.GetNetwork()
	if networkSpec.Subnets == nil {
		networkSpec.SetSubnetDefaultValue()
		subnetsSpec = networkSpec.Subnets
	} else {
		subnetsSpec = clusterScope.GetSubnet()
	}
	for _, subnetSpec := range subnetsSpec {
		subnetName := subnetSpec.Name + "-" + clusterScope.GetUID()
		resourceNameList = append(resourceNameList, subnetName)
	}
	for _, routeTableSpec := range routeTablesSpec {
		routeTableSubnetName := routeTableSpec.SubnetName + "-" + clusterScope.GetUID()
		checkOscAssociate := CheckAssociate(routeTableSubnetName, resourceNameList)
		if checkOscAssociate {
			return nil
		} else {
			return fmt.Errorf("%s subnet dooes not exist in routeTable", routeTableSubnetName)
		}
	}
	return nil
}

// CheckRouteTableOscDuplicateName check that there are not the same name for RouteTable.
func CheckRouteTableOscDuplicateName(clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	clusterScope.Info("check unique routetable")
	var routeTablesSpec []*infrastructurev1beta1.OscRouteTable
	routeTablesSpec = clusterScope.GetRouteTables()
	for _, routeTableSpec := range routeTablesSpec {
		resourceNameList = append(resourceNameList, routeTableSpec.Name)
	}
	duplicateResourceErr := AlertDuplicate(resourceNameList)
	if duplicateResourceErr != nil {
		return duplicateResourceErr
	} else {
		return nil
	}
	return nil
}

// CheckRouteOscDuplicateName check that there are not the same name for route.
func CheckRouteOscDuplicateName(clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	clusterScope.Info("check unique route")
	routeTablesSpec := clusterScope.GetRouteTables()
	for _, routeTableSpec := range routeTablesSpec {
		routesSpec := clusterScope.GetRoute(routeTableSpec.Name)
		for _, routeSpec := range *routesSpec {
			resourceNameList = append(resourceNameList, routeSpec.Name)
		}
		duplicateResourceErr := AlertDuplicate(resourceNameList)
		if duplicateResourceErr != nil {
			return duplicateResourceErr
		} else {
			return nil
		}
	}
	return nil
}

// ReconcileRoute reconcile the RouteTable and the Route of the cluster.
func reconcileRoute(ctx context.Context, clusterScope *scope.ClusterScope, routeSpec infrastructurev1beta1.OscRoute, routeTableName string) (reconcile.Result, error) {
	securitysvc := security.NewService(ctx, clusterScope)
	osccluster := clusterScope.OscCluster

	routeRef := clusterScope.GetRouteRef()
	routeTablesRef := clusterScope.GetRouteTablesRef()

	resourceName := routeSpec.TargetName + "-" + clusterScope.GetUID()
	resourceType := routeSpec.TargetType
	routeName := routeSpec.Name + "-" + clusterScope.GetUID()
	if len(routeRef.ResourceMap) == 0 {
		routeRef.ResourceMap = make(map[string]string)
	}
	if routeSpec.ResourceId != "" {
		routeRef.ResourceMap[routeName] = routeSpec.ResourceId
	}
	var resourceId string
	var err error
	if resourceType == "gateway" {
		resourceId, err = GetInternetServiceResourceId(resourceName, clusterScope)
		if err != nil {
			return reconcile.Result{}, err
		}
	} else {
		resourceId, err = GetNatResourceId(resourceName, clusterScope)
		if err != nil {
			return reconcile.Result{}, err
		}
	}
	destinationIpRange := routeSpec.Destination
	associateRouteTableId := routeTablesRef.ResourceMap[routeTableName]
	routeTableFromRoute, err := securitysvc.GetRouteTableFromRoute(associateRouteTableId, resourceId, resourceType)
	if err != nil {
		return reconcile.Result{}, err
	}
	if routeTableFromRoute == nil {
		clusterScope.Info("### Create Route ###", "Route", resourceId)
		routeTableFromRoute, err = securitysvc.CreateRoute(destinationIpRange, routeTablesRef.ResourceMap[routeTableName], resourceId, resourceType)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not create route for Osccluster %s/%s", err, osccluster.GetNamespace, osccluster.GetName)
		}
	}
	routeRef.ResourceMap[routeName] = *routeTableFromRoute.RouteTableId
	routeSpec.ResourceId = *routeTableFromRoute.RouteTableId
	return reconcile.Result{}, nil

}

// ReconcileRoute reconcile the RouteTable and the Route of the cluster.
func reconcileDeleteRoute(ctx context.Context, clusterScope *scope.ClusterScope, routeSpec infrastructurev1beta1.OscRoute, routeTableName string) (reconcile.Result, error) {
	securitysvc := security.NewService(ctx, clusterScope)
	osccluster := clusterScope.OscCluster

	routeTablesRef := clusterScope.GetRouteTablesRef()

	resourceName := routeSpec.TargetName + "-" + clusterScope.GetUID()
	resourceType := routeSpec.TargetType
	routeName := routeSpec.Name + "-" + clusterScope.GetUID()
	var resourceId string
	var err error
	if resourceType == "gateway" {
		resourceId, err = GetInternetServiceResourceId(resourceName, clusterScope)
		if err != nil {
			return reconcile.Result{}, err
		}
	} else {

		resourceId, err = GetNatResourceId(resourceName, clusterScope)
		if err != nil {
			return reconcile.Result{}, err
		}
	}
	routeTableId, err := GetRouteResourceId(routeName, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	destinationIpRange := routeSpec.Destination
	associateRouteTableId := routeTablesRef.ResourceMap[routeTableName]
	routeTableFromRoute, err := securitysvc.GetRouteTableFromRoute(associateRouteTableId, resourceId, resourceType)
	if err != nil {
		return reconcile.Result{}, err
	}
	if routeTableFromRoute == nil {
		controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
		return reconcile.Result{}, nil
	}
	clusterScope.Info("Delete Route")
	clusterScope.Info("### delete destinationIpRange###", "routeTable", destinationIpRange)

	err = securitysvc.DeleteRoute(destinationIpRange, routeTableId)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("%w Can not delete route for Osccluster %s/%s", err, osccluster.GetNamespace, osccluster.GetName)
	}
	return reconcile.Result{}, nil

}

// ReconcileRouteTable reconcile the RouteTable and the Route of the cluster.
func reconcileRouteTable(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	securitysvc := security.NewService(ctx, clusterScope)
	osccluster := clusterScope.OscCluster

	clusterScope.Info("Create RouteTable")
	var routeTablesSpec []*infrastructurev1beta1.OscRouteTable
	routeTablesSpec = clusterScope.GetRouteTables()
	routeTablesRef := clusterScope.GetRouteTablesRef()
	linkRouteTablesRef := clusterScope.GetLinkRouteTablesRef()

	netSpec := clusterScope.GetNet()
	netSpec.SetDefaultValue()
	netName := netSpec.Name + "-" + clusterScope.GetUID()
	netId, err := GetNetResourceId(netName, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}

	routeTableIds, err := securitysvc.GetRouteTableIdsFromNetIds(netId)
	if err != nil {
		return reconcile.Result{}, err
	}
	for _, routeTableSpec := range routeTablesSpec {
		routeTableName := routeTableSpec.Name + "-" + clusterScope.GetUID()
		routeTableId := routeTablesRef.ResourceMap[routeTableName]
		clusterScope.Info("### Get routeTable Id ###", "routeTable", routeTablesRef.ResourceMap)
		subnetName := routeTableSpec.SubnetName + "-" + clusterScope.GetUID()
		subnetId, err := GetSubnetResourceId(subnetName, clusterScope)
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

			routeTable, err := securitysvc.CreateRouteTable(netId, routeTableName)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("%w Can not create routetable for Osccluster %s/%s", err, osccluster.GetNamespace, osccluster.GetName)
			}
			linkRouteTableId, err := securitysvc.LinkRouteTable(*routeTable.RouteTableId, subnetId)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("%w Can not link routetable with net for Osccluster %s/%s", err, osccluster.GetNamespace, osccluster.GetName)
			}
			clusterScope.Info("### Get routeTable ###", "routeTable", routeTable)
			routeTablesRef.ResourceMap[routeTableName] = *routeTable.RouteTableId
			routeTableSpec.ResourceId = *routeTable.RouteTableId
			linkRouteTablesRef.ResourceMap[routeTableName] = linkRouteTableId

			clusterScope.Info("check route")
			for _, routeSpec := range *routesSpec {
				_, err = reconcileRoute(ctx, clusterScope, routeSpec, routeTableName)
				if err != nil {
					return reconcile.Result{}, err
				}
			}
		}
	}
	return reconcile.Result{}, nil
}

// ReconcileDeleteRouteTable reconcile the destruction of the RouteTable of the cluster.
func reconcileDeleteRouteTable(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	osccluster := clusterScope.OscCluster
	securitysvc := security.NewService(ctx, clusterScope)

	clusterScope.Info("Delete RouteTable")
	var routeTablesSpec []*infrastructurev1beta1.OscRouteTable
	routeTablesSpec = clusterScope.GetRouteTables()
	routeTablesRef := clusterScope.GetRouteTablesRef()
	linkRouteTablesRef := clusterScope.GetLinkRouteTablesRef()

	netSpec := clusterScope.GetNet()
	netSpec.SetDefaultValue()
	netName := netSpec.Name + "-" + clusterScope.GetUID()
	netId, err := GetNetResourceId(netName, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}

	routeTableIds, err := securitysvc.GetRouteTableIdsFromNetIds(netId)
	if err != nil {
		return reconcile.Result{}, err
	}
	for _, routeTableSpec := range routeTablesSpec {
		routeTableSpec.SetDefaultValue()
		routeTableName := routeTableSpec.Name + "-" + clusterScope.GetUID()
		routeTableId := routeTablesRef.ResourceMap[routeTableName]
		clusterScope.Info("### delete routeTable Id ###", "routeTable", routeTableId)

		if !Contains(routeTableIds, routeTableId) {
			controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
			return reconcile.Result{}, nil
		}
		clusterScope.Info("Remove Route")
		routesSpec := clusterScope.GetRoute(routeTableSpec.Name)
		for _, routeSpec := range *routesSpec {
			_, err = reconcileDeleteRoute(ctx, clusterScope, routeSpec, routeTableName)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		clusterScope.Info("Unlink Routetable")

		err = securitysvc.UnlinkRouteTable(linkRouteTablesRef.ResourceMap[routeTableName])
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not delete routeTable for Osccluster %s/%s", err, osccluster.GetNamespace, osccluster.GetName)
		}
		clusterScope.Info("Delete RouteTable")

		err = securitysvc.DeleteRouteTable(routeTablesRef.ResourceMap[routeTableName])
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not delete internetService for Osccluster %s/%s", err, osccluster.GetNamespace, osccluster.GetName)
		}
	}
	return reconcile.Result{}, nil
}
