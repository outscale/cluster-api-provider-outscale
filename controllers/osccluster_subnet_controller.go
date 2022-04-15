package controllers

import (
	"context"
	"fmt"

	infrastructurev1beta1 "github.com/outscale-vbr/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/services/net"
	tag "github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/tag"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// GetResourceId return the resourceId from the resourceMap base on resourceName (tag name + cluster object uid) and resourceType (net, subnet, gateway, route, route-table, public-ip)
func GetSubnetResourceId(resourceName string, clusterScope *scope.ClusterScope) (string, error) {
	subnetRef := clusterScope.GetSubnetRef()
	if subnetId, ok := subnetRef.ResourceMap[resourceName]; ok {
		return subnetId, nil
	} else {
		return "", fmt.Errorf("%s is not exist", resourceName)
	}
}

// CheckFormatParameters check every resource (net, subnet, ...) parameters format (Tag format, cidr format, ..)
func CheckSubnetFormatParameters(clusterScope *scope.ClusterScope) (string, error) {
	clusterScope.Info("Check subnet name parameters")
	var subnetsSpec []*infrastructurev1beta1.OscSubnet
	networkSpec := clusterScope.GetNetwork()
	if networkSpec.Subnets == nil {
		networkSpec.SetSubnetDefaultValue()
		subnetsSpec = networkSpec.Subnets
	} else {
		subnetsSpec = clusterScope.GetSubnet()
	}
	for _, subnetSpec := range subnetsSpec {
		subnetName := subnetSpec.Name + "-" + clusterScope.UID()
		subnetTagName, err := tag.ValidateTagNameValue(subnetName)
		if err != nil {
			return subnetTagName, err
		}
	}
	return "", nil

}

// CheckOscDuplicateName check that there are not the same name for resource with the same kind of resourceType (route-table, subnet, ..).
func CheckSubnetOscDuplicateName(clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	clusterScope.Info("Check unique subnet")
	var subnetsSpec []*infrastructurev1beta1.OscSubnet
	subnetsSpec = clusterScope.GetSubnet()
	for _, subnetSpec := range subnetsSpec {
		resourceNameList = append(resourceNameList, subnetSpec.Name)
	}
	duplicateResourceErr := AlertDuplicate(resourceNameList)
	if duplicateResourceErr != nil {
		return duplicateResourceErr
	} else {
		return nil
	}
}

// ReconcileSubnet reconcile the subnet of the cluster.
func reconcileSubnet(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	netsvc := net.NewService(ctx, clusterScope)
	osccluster := clusterScope.OscCluster

	clusterScope.Info("Create Subnet")

	netSpec := clusterScope.GetNet()
	netSpec.SetDefaultValue()
	netName := netSpec.Name + "-" + clusterScope.UID()
	netId, err := GetNetResourceId(netName, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	var subnetsSpec []*infrastructurev1beta1.OscSubnet
	subnetsSpec = clusterScope.GetSubnet()
	
	subnetRef := clusterScope.GetSubnetRef()
	clusterScope.Info("### Get subnetId ###", "subnet", subnetRef.ResourceMap)
	var subnetIds []string
	subnetIds, err = netsvc.GetSubnetIdsFromNetIds(netId)
	if err != nil {
		return reconcile.Result{}, err
	}
	for _, subnetSpec := range subnetsSpec {
		subnetName := subnetSpec.Name + "-" + clusterScope.UID()
		subnetId := subnetRef.ResourceMap[subnetName]
		if len(subnetRef.ResourceMap) == 0 {
			subnetRef.ResourceMap = make(map[string]string)
		}
		if subnetSpec.ResourceId != "" {
			subnetRef.ResourceMap[subnetName] = subnetSpec.ResourceId
		}
		if !Contains(subnetIds, subnetId) {
			subnet, err := netsvc.CreateSubnet(subnetSpec, netId, subnetName)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("%w: Can not create subnet for Osccluster %s/%s", err, osccluster.Namespace, osccluster.Name)
			}
			clusterScope.Info("### Get subnet ###", "subnet", subnet)
			subnetRef.ResourceMap[subnetName] = *subnet.SubnetId
			subnetSpec.ResourceId = *subnet.SubnetId
		}
	}
	return reconcile.Result{}, nil
}

// ReconcileDeleteSubnet reconcile the destruction of the Subnet of the cluster.
func reconcileDeleteSubnet(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	osccluster := clusterScope.OscCluster
	netsvc := net.NewService(ctx, clusterScope)

	clusterScope.Info("Delete subnet")

	var subnetsSpec []*infrastructurev1beta1.OscSubnet
	subnetsSpec = clusterScope.GetSubnet()
	netSpec := clusterScope.GetNet()
	netSpec.SetDefaultValue()
	netName := netSpec.Name + "-" + clusterScope.UID()

	netId, err := GetNetResourceId(netName, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	subnetRef := clusterScope.GetSubnetRef()
	subnetIds, err := netsvc.GetSubnetIdsFromNetIds(netId)
	if err != nil {
		return reconcile.Result{}, err
	}
	for _, subnetSpec := range subnetsSpec {
		subnetName := subnetSpec.Name + "-" + clusterScope.UID()
		subnetId := subnetRef.ResourceMap[subnetName]
		if !Contains(subnetIds, subnetId) {
			controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
			return reconcile.Result{}, nil
		}
		err = netsvc.DeleteSubnet(subnetRef.ResourceMap[subnetName])
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w: Can not delete subnet for Osccluster %s/%s", err, osccluster.Namespace, osccluster.Name)
		}
	}
	return reconcile.Result{}, nil
}
