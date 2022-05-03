package controllers

import (
	"context"
	"fmt"

	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/net"
	tag "github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag"
	osc "github.com/outscale/osc-sdk-go/v2"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// GetNatResourceId return the NatId from the resourceMap base on NatName (tag name + cluster object uid)
func getNatResourceId(resourceName string, clusterScope *scope.ClusterScope) (string, error) {
	natServiceRef := clusterScope.GetNatServiceRef()
	if natServiceId, ok := natServiceRef.ResourceMap[resourceName]; ok {
		return natServiceId, nil
	} else {
		return "", fmt.Errorf("%s is not exist", resourceName)
	}
}

func checkNatFormatParameters(clusterScope *scope.ClusterScope) (string, error) {
	clusterScope.Info("Check Nat name parameters")
	natServiceSpec := clusterScope.GetNatService()
	natServiceSpec.SetDefaultValue()
	natName := natServiceSpec.Name + "-" + clusterScope.GetUID()
	natSubnetName := natServiceSpec.SubnetName + "-" + clusterScope.GetUID()
	natPublicIpName := natServiceSpec.PublicIpName + "-" + clusterScope.GetUID()
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

// checkNatSubnetOscAssociateResourceName check that Nat Subnet dependancies tag name in both resource configuration are the same.
func checkNatSubnetOscAssociateResourceName(clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	clusterScope.Info("check match subnet with nat service")
	natServiceSpec := clusterScope.GetNatService()
	natServiceSpec.SetDefaultValue()
	natSubnetName := natServiceSpec.SubnetName + "-" + clusterScope.GetUID()
	subnetsSpec := clusterScope.GetSubnet()
	for _, subnetSpec := range subnetsSpec {
		subnetName := subnetSpec.Name + "-" + clusterScope.GetUID()
		resourceNameList = append(resourceNameList, subnetName)
	}
	checkOscAssociate := contains(resourceNameList, natSubnetName)
	if checkOscAssociate {
		return nil
	} else {
		return fmt.Errorf("%s subnet does not exist in natService", natSubnetName)
	}
}

// reconcileNatService reconcile the NatService of the cluster.
func reconcileNatService(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	netsvc := net.NewService(ctx, clusterScope)

	clusterScope.Info("Create NatService")
	natServiceSpec := clusterScope.GetNatService()

	natServiceRef := clusterScope.GetNatServiceRef()
	natServiceName := natServiceSpec.Name + "-" + clusterScope.GetUID()
	var natService *osc.NatService
	publicIpName := natServiceSpec.PublicIpName + "-" + clusterScope.GetUID()
	publicIpId, err := getPublicIpResourceId(publicIpName, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}

	subnetName := natServiceSpec.SubnetName + "-" + clusterScope.GetUID()

	subnetId, err := getSubnetResourceId(subnetName, clusterScope)
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
			return reconcile.Result{}, fmt.Errorf("%w Can not create natservice for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
		}
		clusterScope.Info("### Get natService ###", "natservice", natService)
		natServiceRef.ResourceMap[natServiceName] = natService.GetNatServiceId()
		natServiceSpec.ResourceId = natService.GetNatServiceId()
	}
	return reconcile.Result{}, nil
}

// reconcileDeleteNatService reconcile the destruction of the NatService of the cluster.
func reconcileDeleteNatService(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	osccluster := clusterScope.OscCluster
	netsvc := net.NewService(ctx, clusterScope)

	clusterScope.Info("Delete natService")
	natServiceSpec := clusterScope.GetNatService()
	natServiceSpec.SetDefaultValue()

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
		return reconcile.Result{}, fmt.Errorf("%w Can not delete natService for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
	}
	return reconcile.Result{}, err
}
