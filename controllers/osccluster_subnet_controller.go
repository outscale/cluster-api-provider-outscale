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

// getSubnetResourceId return the subnetId from the resourceMap base on subnetName (tag name + cluster object uid)
func getSubnetResourceId(resourceName string, clusterScope *scope.ClusterScope) (string, error) {
	subnetRef := clusterScope.GetSubnetRef()
	if subnetId, ok := subnetRef.ResourceMap[resourceName]; ok {
		return subnetId, nil
	} else {
		return "", fmt.Errorf("%s is not exist", resourceName)
	}
}

// checkSubnetFormatParameters check Subnet parameters format (Tag format, cidr format, ..)
func checkSubnetFormatParameters(clusterScope *scope.ClusterScope) (string, error) {
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
		subnetName := subnetSpec.Name + "-" + clusterScope.GetUID()
		subnetTagName, err := tag.ValidateTagNameValue(subnetName)
		if err != nil {
			return subnetTagName, err
		}
	}
	return "", nil

}

// checkSubnetOscDuplicateName check that there are not the same name for subnet
func checkSubnetOscDuplicateName(clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	clusterScope.Info("Check unique subnet")
	var subnetsSpec []*infrastructurev1beta1.OscSubnet
	subnetsSpec = clusterScope.GetSubnet()
	for _, subnetSpec := range subnetsSpec {
		resourceNameList = append(resourceNameList, subnetSpec.Name)
	}
	duplicateResourceErr := alertDuplicate(resourceNameList)
	if duplicateResourceErr != nil {
		return duplicateResourceErr
	} else {
		return nil
	}
}

// reconcileSubnet reconcile the subnet of the cluster.
func reconcileSubnet(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	netsvc := net.NewService(ctx, clusterScope)

	clusterScope.Info("Create Subnet")

	netSpec := clusterScope.GetNet()
	netSpec.SetDefaultValue()
	netName := netSpec.Name + "-" + clusterScope.GetUID()
	netId, err := getNetResourceId(netName, clusterScope)
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
		subnetName := subnetSpec.Name + "-" + clusterScope.GetUID()
		subnetId := subnetSpec.ResourceId
		if len(subnetRef.ResourceMap) == 0 {
			subnetRef.ResourceMap = make(map[string]string)
		}
		if subnetSpec.ResourceId != "" {
			subnetRef.ResourceMap[subnetName] = subnetSpec.ResourceId
		}
		if !contains(subnetIds, subnetId) {
			subnet, err := netsvc.CreateSubnet(subnetSpec, netId, subnetName)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("%w Can not create subnet for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
			}
			clusterScope.Info("### Get subnet ###", "subnet", subnet)
			subnetRef.ResourceMap[subnetName] = subnet.GetSubnetId()
			subnetSpec.ResourceId = subnet.GetSubnetId()
		}
	}
	return reconcile.Result{}, nil
}

// reconcileDeleteSubnet reconcile the destruction of the Subnet of the cluster.
func reconcileDeleteSubnet(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	osccluster := clusterScope.OscCluster
	netsvc := net.NewService(ctx, clusterScope)

	clusterScope.Info("Delete subnet")

	var subnetsSpec []*infrastructurev1beta1.OscSubnet
	subnetsSpec = clusterScope.GetSubnet()
	netSpec := clusterScope.GetNet()
	netSpec.SetDefaultValue()
	netName := netSpec.Name + "-" + clusterScope.GetUID()

	netId, err := getNetResourceId(netName, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	//	subnetRef := clusterScope.GetSubnetRef()
	subnetIds, err := netsvc.GetSubnetIdsFromNetIds(netId)
	if err != nil {
		return reconcile.Result{}, err
	}
	for _, subnetSpec := range subnetsSpec {
		//		subnetName := subnetSpec.Name + "-" + clusterScope.GetUID()
		subnetId := subnetSpec.ResourceId
		if !contains(subnetIds, subnetId) {
			controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
			return reconcile.Result{}, nil
		}
		err = netsvc.DeleteSubnet(subnetId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not delete subnet for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
		}
	}
	return reconcile.Result{}, nil
}
