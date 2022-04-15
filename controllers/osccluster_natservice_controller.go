package controllers

import (
	"context"
	"fmt"

	infrastructurev1beta1 "github.com/outscale-vbr/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/services/net"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	tag "github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/tag"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
        osc "github.com/outscale/osc-sdk-go/v2"

)

// GetResourceId return the resourceId from the resourceMap base on resourceName (tag name + cluster object uid) and resourceType (net, subnet, gateway, route, route-table, public-ip)
func GetNatResourceId(resourceName string, clusterScope *scope.ClusterScope) (string, error) {
	natServiceRef := clusterScope.GetNatServiceRef()
	if natServiceId, ok := natServiceRef.ResourceMap[resourceName]; ok {
		return natServiceId, nil
	} else {
		return "", fmt.Errorf("%s is not exist", resourceName)
	}
}

func CheckNatFormatParameters(clusterScope *scope.ClusterScope) (string, error) {
	clusterScope.Info("Check Nat name parameters")
	natServiceSpec := clusterScope.GetNatService()
	natServiceSpec.SetDefaultValue()
        natName := natServiceSpec.Name + "-" + clusterScope.UID()
	natSubnetName := natServiceSpec.SubnetName + "-" + clusterScope.UID()
        natPublicIpName := natServiceSpec.PublicIpName + "-" + clusterScope.UID()
	natTagName, err := tag.ValidateTagNameValue(natName)
	if err != nil {
		return natTagName, err
	}
	natSubnetTagName, err := tag.ValidateTagNameValue(natSubnetName)
	if err != nil {
		return natSubnetTagName, err
	}
	natPublicIpTagName, err := tag.ValidateTagNameValue(natPublicIpName)
	if err != nil {
		return natPublicIpTagName, err
	}
	return "", nil
}

	 			
// CheckOscAssociateResourceName check that resourceType dependancies tag name in both resource configuration are the same.
func CheckNatSubnetOscAssociateResourceName(clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	clusterScope.Info("check match subnet with nat service")
	natServiceSpec := clusterScope.GetNatService()
	natServiceSpec.SetDefaultValue()
	natSubnetName := natServiceSpec.SubnetName + "-" + clusterScope.UID()
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
	natServiceSpec := clusterScope.GetNatService()
	natServiceRef := clusterScope.GetNatServiceRef()
	natServiceName := natServiceSpec.Name + "-" + clusterScope.UID()
	var natService *osc.NatService
	publicIpName := natServiceSpec.PublicIpName + "-" + clusterScope.UID()
	publicIpId, err := GetPublicIpResourceId(publicIpName, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}

	subnetName := natServiceSpec.SubnetName + "-" + clusterScope.UID()

	subnetId, err := GetSubnetResourceId(subnetName, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	if len(natServiceRef.ResourceMap) == 0 {
		natServiceRef.ResourceMap = make(map[string]string)
	}
	if natServiceSpec.ResourceId != "" {
		natServiceRef.ResourceMap[natServiceName] = natServiceSpec.ResourceId
		natServiceId := natServiceSpec.ResourceId
		clusterScope.Info("### Get natService Id ###", "natservice", natServiceRef.ResourceMap)
		natService, err = netsvc.GetNatService(natServiceId)
		if err != nil {
			return reconcile.Result{}, err
		}
	}
	
	if natService == nil || natServiceSpec.ResourceId == "" {

		natService, err := netsvc.CreateNatService(publicIpId, subnetId, natServiceName)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w: Can not create natservice for Osccluster %s/%s", err, osccluster.Namespace, osccluster.Name)
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
	natServiceSpec := clusterScope.GetNatService()
//	natServiceRef := clusterScope.GetNatServiceRef()
//	natServiceName := natServiceSpec.Name + "-" + clusterScope.UID()
	natServiceId := natServiceSpec.ResourceId
	natservice, err := netsvc.GetNatService(natServiceId)
	if err != nil {
		return reconcile.Result{}, err
	}
	if natservice == nil {
		controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
		return reconcile.Result{}, nil
	}
	err = netsvc.DeleteNatService(natServiceId)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("%w: Can not delete natService for Osccluster %s/%s", err, osccluster.Namespace, osccluster.Name)
	}
	return reconcile.Result{}, err
}
