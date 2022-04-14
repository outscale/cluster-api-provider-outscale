package controllers

import (
	"context"
	"fmt"

	infrastructurev1beta1 "github.com/outscale-vbr/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/services/net"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// GetResourceId return the resourceId from the resourceMap base on resourceName (tag name + cluster object uid) and resourceType (net, subnet, gateway, route, route-table, public-ip)
func GetNatResourceId(resourceName string, clusterScope *scope.ClusterScope) (string, error) {
	natServiceRef := clusterScope.NatServiceRef()
	if natServiceId, ok := natServiceRef.ResourceMap[resourceName]; ok {
		return natServiceId, nil
	} else {
		return "", fmt.Errorf("%s is not exist", resourceName)
	}
}

// CheckOscAssociateResourceName check that resourceType dependancies tag name in both resource configuration are the same.
func CheckNatSubnetOscAssociateResourceName(clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	clusterScope.Info("check match subnet with nat service")
	natServiceSpec := clusterScope.NatService()
	natServiceSpec.SetDefaultValue()
	natSubnetName := natServiceSpec.SubnetName + "-" + clusterScope.UID()
	var subnetsSpec []*infrastructurev1beta1.OscSubnet
	networkSpec := clusterScope.Network()
	if networkSpec.Subnets == nil {
		networkSpec.SetSubnetDefaultValue()
		subnetsSpec = networkSpec.Subnets
	} else {
		subnetsSpec = clusterScope.Subnet()
	}
	for _, subnetSpec := range subnetsSpec {
		subnetName := subnetSpec.Name + "-" + clusterScope.UID()
		resourceNameList = append(resourceNameList, subnetName)
	}
	checkOscAssociate := CheckAssociate(natSubnetName, resourceNameList)
	if checkOscAssociate {
		return nil
	} else {
		return fmt.Errorf("%s subnet does not exist in natService", natSubnetName)
	}
	return nil
}

// ReconcileNatService reconcile the NatService of the cluster.
func reconcileNatService(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	netsvc := net.NewService(ctx, clusterScope)
	osccluster := clusterScope.OscCluster

	clusterScope.Info("Create NatService")
	natServiceSpec := clusterScope.NatService()
	natServiceRef := clusterScope.NatServiceRef()
	natServiceSpec.SetDefaultValue()
	natServiceName := natServiceSpec.Name + "-" + clusterScope.UID()

	publicIpName := natServiceSpec.PublicIpName + "-" + clusterScope.UID()
	publicIpId, err := GetResourceId(publicIpName, "public-ip", clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}

	subnetName := natServiceSpec.SubnetName + "-" + clusterScope.UID()

	subnetId, err := GetResourceId(subnetName, "subnet", clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	if len(natServiceRef.ResourceMap) == 0 {
		natServiceRef.ResourceMap = make(map[string]string)
	}
	if natServiceSpec.ResourceId != "" {
		natServiceRef.ResourceMap[natServiceName] = natServiceSpec.ResourceId
	}
	var natServiceIds = []string{natServiceRef.ResourceMap[natServiceName]}
	clusterScope.Info("### Get natService Id ###", "natservice", natServiceRef.ResourceMap)
	natService, err := netsvc.GetNatService(natServiceIds)
	if err != nil {
		return reconcile.Result{}, err
	}

	if natService == nil {

		natService, err = netsvc.CreateNatService(publicIpId, subnetId, natServiceName)
		if err != nil {
			return reconcile.Result{}, errors.Wrapf(err, "Can not create natservice for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
		}
		clusterScope.Info("### Get natService ###", "natservice", natService)
		natServiceRef.ResourceMap[natServiceName] = *natService.NatServiceId
		natServiceSpec.ResourceId = *natService.NatServiceId
	}
	return reconcile.Result{}, nil
}

// ReconcileDeleteNatService reconcile the destruction of the NatService of the cluster.
func reconcileDeleteNatService(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	osccluster := clusterScope.OscCluster
	netsvc := net.NewService(ctx, clusterScope)

	clusterScope.Info("Delete natService")
	natServiceSpec := clusterScope.NatService()
	natServiceSpec.SetDefaultValue()
	natServiceRef := clusterScope.NatServiceRef()
	natServiceName := natServiceSpec.Name + "-" + clusterScope.UID()
	var natServiceIds = []string{natServiceRef.ResourceMap[natServiceName]}
	natservice, err := netsvc.GetNatService(natServiceIds)
	if err != nil {
		return reconcile.Result{}, err
	}
	if natservice == nil {
		controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
		return reconcile.Result{}, nil
	}
	err = netsvc.DeleteNatService(natServiceRef.ResourceMap[natServiceName])
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "Can not delete natService for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
	}
	return reconcile.Result{}, err
}
