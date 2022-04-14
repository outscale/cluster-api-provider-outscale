package controllers

import (
	"context"
	"fmt"

	infrastructurev1beta1 "github.com/outscale-vbr/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/services/security"
	tag "github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/tag"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// GetResourceId return the resourceId from the resourceMap base on resourceName (tag name + cluster object uid) and resourceType (net, subnet, gateway, route, route-table, public-ip)
func GetPublicIpResourceId(resourceName string, clusterScope *scope.ClusterScope) (string, error) {
	publicIpRef := clusterScope.GetPublicIpRef()
	if publicIpId, ok := publicIpRef.ResourceMap[resourceName]; ok {
		return publicIpId, nil
	} else {
		return "", fmt.Errorf("%s is not exist", resourceName)
	}
}

// CheckFormatParameters check every resource (net, subnet, ...) parameters format (Tag format, cidr format, ..)
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
		publicIpName := publicIpSpec.Name + "-" + clusterScope.UID()
		publicIpTagName, err := tag.ValidateTagNameValue(publicIpName)
		if err != nil {
			return publicIpTagName, err
		}
	}
	return "", nil
}

// CheckOscAssociateResourceName check that resourceType dependancies tag name in both resource configuration are the same.
func CheckPublicIpOscAssociateResourceName(clusterScope *scope.ClusterScope) error {
	clusterScope.Info("check match public ip with nat service")
	var resourceNameList []string
	natServiceSpec := clusterScope.GetNatService()
	natServiceSpec.SetDefaultValue()
	natPublicIpName := natServiceSpec.PublicIpName + "-" + clusterScope.UID()
	var publicIpsSpec []*infrastructurev1beta1.OscPublicIp
	networkSpec := clusterScope.GetNetwork()
	if networkSpec.PublicIps == nil {
		networkSpec.SetPublicIpDefaultValue()
		publicIpsSpec = networkSpec.PublicIps
	} else {
		publicIpsSpec = clusterScope.GetPublicIp()
	}
	for _, publicIpSpec := range publicIpsSpec {
		publicIpName := publicIpSpec.Name + "-" + clusterScope.UID()
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

// CheckOscDuplicateName check that there are not the same name for resource with the same kind of resourceType (route-table, subnet, ..).
func CheckPublicIpOscDuplicateName(clusterScope *scope.ClusterScope) error {
	clusterScope.Info("Check unique name publicIp")
	var publicIpsSpec []*infrastructurev1beta1.OscPublicIp
	var resourceNameList []string
	networkSpec := clusterScope.GetNetwork()
	if networkSpec.PublicIps == nil {
		networkSpec.SetPublicIpDefaultValue()
		publicIpsSpec = networkSpec.PublicIps
	} else {
		publicIpsSpec = clusterScope.GetPublicIp()
	}
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
	osccluster := clusterScope.OscCluster

	clusterScope.Info("Create PublicIp")
	var publicIpsSpec []*infrastructurev1beta1.OscPublicIp
	networkSpec := clusterScope.GetNetwork()
	if networkSpec.PublicIps == nil {
		networkSpec.SetPublicIpDefaultValue()
		publicIpsSpec = networkSpec.PublicIps
	} else {
		publicIpsSpec = clusterScope.GetPublicIp()
	}
        var publicIpId string
	publicIpRef := clusterScope.GetPublicIpRef()
	for _, publicIpSpec := range publicIpsSpec {
		publicIpName := publicIpSpec.Name + "-" + clusterScope.UID()
		publicIpId = publicIpRef.ResourceMap[publicIpName]
	}
	publicIpIds, err := securitysvc.ValidatePublicIpIds(publicIpId)
	if err != nil {
		return reconcile.Result{}, err
	}
	clusterScope.Info("### Check Id  ###", "publicip", publicIpIds)
	for _, publicIpSpec := range publicIpsSpec {
		publicIpName := publicIpSpec.Name + "-" + clusterScope.UID()
		publicIpId := publicIpRef.ResourceMap[publicIpName]
		clusterScope.Info("### Get publicIp Id ###", "publicip", publicIpRef.ResourceMap)
		if publicIpSpec.ResourceId != "" {
			publicIpRef.ResourceMap[publicIpName] = publicIpSpec.ResourceId
		}
		if len(publicIpRef.ResourceMap) == 0 {
			publicIpRef.ResourceMap = make(map[string]string)
		}
		if !contains(publicIpIds, publicIpId) {
			publicIp, err := securitysvc.CreatePublicIp(publicIpName)
			if err != nil {
				return reconcile.Result{}, errors.Wrapf(err, "Can not create publicIp for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
			}
			clusterScope.Info("### Get publicIp  ###", "publicip", publicIp)
			publicIpRef.ResourceMap[publicIpName] = *publicIp.PublicIpId
			publicIpSpec.ResourceId = *publicIp.PublicIpId
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
	var publicIpId string
	publicIpRef := clusterScope.GetPublicIpRef()
	for _, publicIpSpec := range publicIpsSpec {
		publicIpName := publicIpSpec.Name + "-" + clusterScope.UID()
		publicIpId = publicIpRef.ResourceMap[publicIpName]
	}
	publicIpIds, err := securitysvc.ValidatePublicIpIds(publicIpId)
	if err != nil {
		return reconcile.Result{}, err
	}
	clusterScope.Info("### Check Id  ###", "publicip", publicIpIds)
	for _, publicIpSpec := range publicIpsSpec {
		publicIpName := publicIpSpec.Name + "-" + clusterScope.UID()
		publicIpId := publicIpRef.ResourceMap[publicIpName]
		if !contains(publicIpIds, publicIpId) {
			controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
			return reconcile.Result{}, nil
		}
		clusterScope.Info("Remove publicip")
		err = securitysvc.DeletePublicIp(publicIpRef.ResourceMap[publicIpName])
		if err != nil {
			return reconcile.Result{}, errors.Wrapf(err, "Can not delete publicIp for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
		}

	}
	return reconcile.Result{}, nil
}
