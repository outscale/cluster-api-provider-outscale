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

	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/security"
	tag "github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// getPublicIPResourceID return the resourceId from the resourceMap base on PublicIPName  (tag name + cluster object uid)
func getPublicIPResourceID(resourceName string, clusterScope *scope.ClusterScope) (string, error) {
	publicIpRef := clusterScope.GetPublicIPRef()
	if publicIpId, ok := publicIpRef.ResourceMap[resourceName]; ok {
		return publicIpId, nil
	} else {
		return "", fmt.Errorf("%s does not exist", resourceName)
	}
}

// checkPublicIPFormatParameters check PublicIp parameters format (Tag format, cidr format, ..)
func checkPublicIPFormatParameters(clusterScope *scope.ClusterScope) (string, error) {
	clusterScope.Info("Check Public Ip parameters")
	var publicIpsSpec []*infrastructurev1beta1.OscPublicIP
	networkSpec := clusterScope.GetNetwork()
	if networkSpec.PublicIPS == nil {
		networkSpec.SetPublicIPDefaultValue()
		publicIpsSpec = networkSpec.PublicIPS
	} else {
		publicIpsSpec = clusterScope.GetPublicIP()
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

// checkPublicIPOscAssociateResourceName check that PublicIp dependancies tag name in both resource configuration are the same.
func checkPublicIPOscAssociateResourceName(clusterScope *scope.ClusterScope) error {
	clusterScope.Info("check match public ip with nat service")
	var resourceNameList []string
	natServiceSpec := clusterScope.GetNatService()
	natServiceSpec.SetDefaultValue()
	natPublicIPName := natServiceSpec.PublicIPName + "-" + clusterScope.GetUID()
	var publicIpsSpec []*infrastructurev1beta1.OscPublicIP
	networkSpec := clusterScope.GetNetwork()
	publicIpsSpec = networkSpec.PublicIPS
	for _, publicIpSpec := range publicIpsSpec {
		publicIpName := publicIpSpec.Name + "-" + clusterScope.GetUID()
		resourceNameList = append(resourceNameList, publicIpName)
	}
	checkOscAssociate := Contains(resourceNameList, natPublicIPName)
	if checkOscAssociate {
		return nil
	} else {
		return fmt.Errorf("publicIp %s does not exist in natService ", natPublicIPName)
	}
}

// checkPublicIpOscDuplicateName check that there are not the same name for PublicIp resource.
func checkPublicIPOscDuplicateName(clusterScope *scope.ClusterScope) error {
	clusterScope.Info("Check unique name publicIp")
	var resourceNameList []string
	publicIpsSpec := clusterScope.GetPublicIP()
	for _, publicIpSpec := range publicIpsSpec {
		resourceNameList = append(resourceNameList, publicIpSpec.Name)
	}
	duplicateResourceErr := alertDuplicate(resourceNameList)
	if duplicateResourceErr != nil {
		return duplicateResourceErr
	} else {
		return nil
	}
}

// reconcilePublicIP reconcile the PublicIp of the cluster.
func reconcilePublicIP(ctx context.Context, clusterScope *scope.ClusterScope, publicIpSvc security.OscPublicIPInterface) (reconcile.Result, error) {

	clusterScope.Info("Create PublicIp")
	var publicIpsSpec []*infrastructurev1beta1.OscPublicIP = clusterScope.GetPublicIP()
	var publicIpId string
	publicIpRef := clusterScope.GetPublicIPRef()
	var publicIpIds []string
	for _, publicIpSpec := range publicIpsSpec {
		publicIpId = publicIpSpec.ResourceID
		publicIpIds = append(publicIpIds, publicIpId)
	}
	clusterScope.Info("Check if the desired publicip exist")
	validPublicIpIds, err := publicIpSvc.ValidatePublicIPIds(publicIpIds)
	if err != nil {
		return reconcile.Result{}, err
	}
	clusterScope.Info("### Check Id  ###", "publicip", publicIpIds)
	for _, publicIpSpec := range publicIpsSpec {
		publicIpName := publicIpSpec.Name + "-" + clusterScope.GetUID()
		clusterScope.Info("### Get publicIp Id ###", "publicip", publicIpRef.ResourceMap)
		if len(publicIpRef.ResourceMap) == 0 {
			publicIpRef.ResourceMap = make(map[string]string)
		}
		if publicIpSpec.ResourceID != "" {
			publicIpRef.ResourceMap[publicIpName] = publicIpSpec.ResourceID
		}
		_, resourceMapExist := publicIpRef.ResourceMap[publicIpName]
		if resourceMapExist {
			publicIpSpec.ResourceID = publicIpRef.ResourceMap[publicIpName]
		}

		publicIpId := publicIpRef.ResourceMap[publicIpName]
		if !Contains(validPublicIpIds, publicIpId) {
			publicIp, err := publicIpSvc.CreatePublicIP(publicIpName)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("%w Can not create publicIp for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
			}
			clusterScope.Info("### Get publicIp  ###", "publicip", publicIp)
			publicIpRef.ResourceMap[publicIpName] = publicIp.GetPublicIpId()
			publicIpSpec.ResourceID = publicIp.GetPublicIpId()
		}
	}
	return reconcile.Result{}, nil
}

// reconcileDeletePublicIP reconcile the destruction of the PublicIp of the cluster.
func reconcileDeletePublicIP(ctx context.Context, clusterScope *scope.ClusterScope, publicIpSvc security.OscPublicIPInterface) (reconcile.Result, error) {
	osccluster := clusterScope.OscCluster

	clusterScope.Info("Delete PublicIp")
	var publicIpsSpec []*infrastructurev1beta1.OscPublicIP
	networkSpec := clusterScope.GetNetwork()
	if networkSpec.PublicIPS == nil {
		networkSpec.SetPublicIPDefaultValue()
		publicIpsSpec = networkSpec.PublicIPS
	} else {
		publicIpsSpec = clusterScope.GetPublicIP()
	}
	var publicIpIds []string
	var publicIpId string
	for _, publicIpSpec := range publicIpsSpec {
		publicIpId = publicIpSpec.ResourceID
		publicIpIds = append(publicIpIds, publicIpId)
	}
	validPublicIpIds, err := publicIpSvc.ValidatePublicIPIds(publicIpIds)
	if err != nil {
		return reconcile.Result{}, err
	}
	clusterScope.Info("### Check Id  ###", "publicip", publicIpIds)
	for _, publicIpSpec := range publicIpsSpec {
		publicIpId := publicIpSpec.ResourceID
		clusterScope.Info("### check PublicIp Id ###", "publicipid", publicIpId)
		publicIpName := publicIpSpec.Name + "-" + clusterScope.GetUID()
		clusterScope.Info("### Check validPublicIpIds###", "validPublicIpIds", validPublicIpIds)
		if !Contains(validPublicIpIds, publicIpId) {
			controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
			return reconcile.Result{}, nil
		}
		err = publicIpSvc.CheckPublicIPUnlink(5, 120, publicIpId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not delete publicIp %s for Osccluster %s/%s", err, publicIpId, clusterScope.GetNamespace(), clusterScope.GetName())
		}

		clusterScope.Info("Remove publicip")
		clusterScope.Info("Delete the desired publicip", "publicIpName", publicIpName)
		err = publicIpSvc.DeletePublicIP(publicIpId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not delete publicIp for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
		}

	}
	return reconcile.Result{}, nil
}
