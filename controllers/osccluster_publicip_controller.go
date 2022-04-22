package controllers

import (
	"context"
	"fmt"

	infrastructurev1beta1 "github.com/outscale-vbr/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/services/security"
	tag "github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/tag"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// GetPublicIpResourceId return the resourceId from the resourceMap base on PublicIpName (tag name + cluster object uid)
func GetPublicIpResourceId(resourceName string, clusterScope *scope.ClusterScope) (string, error) {
	publicIpRef := clusterScope.GetPublicIpRef()
	if publicIpId, ok := publicIpRef.ResourceMap[resourceName]; ok {
		return publicIpId, nil
	} else {
		return "", fmt.Errorf("%s is not exist", resourceName)
	}
}

// CheckPublicIpFormatParameters check PublicIp parameters format (Tag format, cidr format, ..)
func CheckPublicIpFormatParameters(clusterScope *scope.ClusterScope) (string, error) {
	clusterScope.Info("Check Public Ip parameters")
	var publicIpsSpec []*infrastructurev1beta1.OscPublicIp
	networkSpec := clusterScope.GetNetwork()
	if networkSpec.PublicIps == nil {
		networkSpec.SetPublicIpDefaultValue()
		publicIpsSpec = networkSpec.PublicIps
	} else {
		publicIpsSpec = clusterScope.GetPublicIp()
	}
	for _, publicIpSpec := range publicIpsSpec {
		publicIpName := publicIpSpec.Name + "-" + clusterScope.GetUID()
		publicIpTagName, err := tag.ValidateTagNameValue(publicIpName)
		if err != nil {
			return publicIpTagName, err
		}
	}
	return "", nil
}

// CheckPublicIpOscAssociateResourceName check that PublicIp dependancies tag name in both resource configuration are the same.
func CheckPublicIpOscAssociateResourceName(clusterScope *scope.ClusterScope) error {
	clusterScope.Info("check match public ip with nat service")
	var resourceNameList []string
	natServiceSpec := clusterScope.GetNatService()
	natServiceSpec.SetDefaultValue()
	natPublicIpName := natServiceSpec.PublicIpName + "-" + clusterScope.GetUID()
	var publicIpsSpec []*infrastructurev1beta1.OscPublicIp
	networkSpec := clusterScope.GetNetwork()
	publicIpsSpec = networkSpec.PublicIps
	for _, publicIpSpec := range publicIpsSpec {
		publicIpName := publicIpSpec.Name + "-" + clusterScope.GetUID()
		resourceNameList = append(resourceNameList, publicIpName)
	}
	checkOscAssociate := CheckAssociate(natPublicIpName, resourceNameList)
	if checkOscAssociate {
		return nil
	} else {
		return fmt.Errorf("publicIp %s does not exist in natService ", natPublicIpName)
	}
	return nil
}

// CheckPublicIpOscDuplicateName check that there are not the same name for PublicIp resource.
func CheckPublicIpOscDuplicateName(clusterScope *scope.ClusterScope) error {
	clusterScope.Info("Check unique name publicIp")
	var publicIpsSpec []*infrastructurev1beta1.OscPublicIp
	var resourceNameList []string
	publicIpsSpec = clusterScope.GetPublicIp()
	for _, publicIpSpec := range publicIpsSpec {
		resourceNameList = append(resourceNameList, publicIpSpec.Name)
	}
	duplicateResourceErr := AlertDuplicate(resourceNameList)
	if duplicateResourceErr != nil {
		return duplicateResourceErr
	} else {
		return nil
	}
	return nil
}

// ReconcilePublicIp reconcile the PublicIp of the cluster.
func reconcilePublicIp(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	securitysvc := security.NewService(ctx, clusterScope)

	clusterScope.Info("Create PublicIp")
	var publicIpsSpec []*infrastructurev1beta1.OscPublicIp
	publicIpsSpec = clusterScope.GetPublicIp()
	var publicIpId string
	publicIpRef := clusterScope.GetPublicIpRef()
	var publicIpIds []string
	for _, publicIpSpec := range publicIpsSpec {
		publicIpId = publicIpSpec.ResourceId
		publicIpIds = append(publicIpIds, publicIpId)
	}
	validPublicIpIds, err := securitysvc.ValidatePublicIpIds(publicIpIds)
	if err != nil {
		return reconcile.Result{}, err
	}
	clusterScope.Info("### Check Id  ###", "publicip", publicIpIds)
	for _, publicIpSpec := range publicIpsSpec {
		publicIpName := publicIpSpec.Name + "-" + clusterScope.GetUID()
		publicIpId := publicIpRef.ResourceMap[publicIpName]
		clusterScope.Info("### Get publicIp Id ###", "publicip", publicIpRef.ResourceMap)
		if len(publicIpRef.ResourceMap) == 0 {
			publicIpRef.ResourceMap = make(map[string]string)
		}
		if publicIpSpec.ResourceId != "" {
			publicIpRef.ResourceMap[publicIpName] = publicIpSpec.ResourceId
		}
		if !Contains(validPublicIpIds, publicIpId) {
			publicIp, err := securitysvc.CreatePublicIp(publicIpName)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("%w Can not create publicIp for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
			}
			clusterScope.Info("### Get publicIp  ###", "publicip", publicIp)
			publicIpRef.ResourceMap[publicIpName] = publicIp.GetPublicIpId()
			publicIpSpec.ResourceId = publicIp.GetPublicIpId()
		}
	}
	return reconcile.Result{}, nil
}

// ReconcileDeletePublicIp reconcile the destruction of the PublicIp of the cluster.
func reconcileDeletePublicIp(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	securitysvc := security.NewService(ctx, clusterScope)
	osccluster := clusterScope.OscCluster

	clusterScope.Info("Delete PublicIp")
	var publicIpsSpec []*infrastructurev1beta1.OscPublicIp
	networkSpec := clusterScope.GetNetwork()
	if networkSpec.PublicIps == nil {
		networkSpec.SetPublicIpDefaultValue()
		publicIpsSpec = networkSpec.PublicIps
	} else {
		publicIpsSpec = clusterScope.GetPublicIp()
	}
	var publicIpIds []string
	var publicIpId string
	for _, publicIpSpec := range publicIpsSpec {
		publicIpId = publicIpSpec.ResourceId
		publicIpIds = append(publicIpIds, publicIpId)
	}
	validPublicIpIds, err := securitysvc.ValidatePublicIpIds(publicIpIds)
	if err != nil {
		return reconcile.Result{}, err
	}
	clusterScope.Info("### Check Id  ###", "publicip", publicIpIds)
	for _, publicIpSpec := range publicIpsSpec {
		//		publicIpName := publicIpSpec.Name + "-" + clusterScope.GetUID()
		publicIpId := publicIpSpec.ResourceId
		if !Contains(validPublicIpIds, publicIpId) {
			controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
			return reconcile.Result{}, nil
		}
		clusterScope.Info("Remove publicip")
		err = securitysvc.DeletePublicIp(publicIpId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not delete publicIp for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
		}

	}
	return reconcile.Result{}, nil
}
