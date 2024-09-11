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

// getPublicIpResourceId return the resourceId from the resourceMap base on PublicIpName (tag name + cluster object uid)
func getPublicIpResourceId(resourceName string, clusterScope *scope.ClusterScope) (string, error) {
	publicIpRef := clusterScope.GetPublicIpRef()
	if publicIpId, ok := publicIpRef.ResourceMap[resourceName]; ok {
		return publicIpId, nil
	} else {
		return "", fmt.Errorf("%s does not exist", resourceName)
	}
}

func getLinkPublicIpResourceId(resourceName string, machineScope *scope.MachineScope) (string, error) {
	linkPublicIpRef := machineScope.GetLinkPublicIpRef()
	if linkPublicIpId, ok := linkPublicIpRef.ResourceMap[resourceName]; ok {
		return linkPublicIpId, nil
	} else {
		return "", fmt.Errorf("%s does not exist", resourceName)
	}
}

// checkPublicIpFormatParameters check PublicIp parameters format (Tag format, cidr format, ..)
func checkPublicIpFormatParameters(clusterScope *scope.ClusterScope) (string, error) {
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
		clusterScope.V(2).Info("Check Public Ip parameters")
		publicIpTagName, err := tag.ValidateTagNameValue(publicIpName)
		if err != nil {
			return publicIpTagName, err
		}
	}
	return "", nil
}

// checkPublicIpOscAssociateResourceName check that PublicIp dependancies tag name in both resource configuration are the same.
func checkPublicIpOscAssociateResourceName(clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	var natServicesSpec []*infrastructurev1beta1.OscNatService
	networkSpec := clusterScope.GetNetwork()
	if networkSpec.NatServices == nil {
		// Add backwards compatibility with NatService parameter that used single NatService
		natServiceSpec := clusterScope.GetNatService()
		natServiceSpec.SetDefaultValue()
		natServicesSpec = append(natServicesSpec, natServiceSpec)
	} else {
		natServicesSpec = clusterScope.GetNatServices()
	}

	for _, natServiceSpec := range natServicesSpec {
		natPublicIpName := natServiceSpec.PublicIpName + "-" + clusterScope.GetUID()

		var publicIpsSpec []*infrastructurev1beta1.OscPublicIp
		networkSpec := clusterScope.GetNetwork()
		publicIpsSpec = networkSpec.PublicIps
		for _, publicIpSpec := range publicIpsSpec {
			publicIpName := publicIpSpec.Name + "-" + clusterScope.GetUID()
			resourceNameList = append(resourceNameList, publicIpName)
		}
		clusterScope.V(2).Info("Check match public ip with nat service")

		checkOscAssociate := Contains(resourceNameList, natPublicIpName)
		if checkOscAssociate {
			return nil
		} else {
			return fmt.Errorf("publicIp %s does not exist in natService ", natPublicIpName)
		}
	}
	return nil
}

// checkPublicIpOscDuplicateName check that there are not the same name for PublicIp resource.
func checkPublicIpOscDuplicateName(clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	publicIpsSpec := clusterScope.GetPublicIp()
	for _, publicIpSpec := range publicIpsSpec {
		resourceNameList = append(resourceNameList, publicIpSpec.Name)
	}
	clusterScope.V(2).Info("Check unique name publicIp")
	duplicateResourceErr := alertDuplicate(resourceNameList)
	if duplicateResourceErr != nil {
		return duplicateResourceErr
	} else {
		return nil
	}
}

// reconcilePublicIp reconcile the PublicIp of the cluster.
func reconcilePublicIp(ctx context.Context, clusterScope *scope.ClusterScope, publicIpSvc security.OscPublicIpInterface, tagSvc tag.OscTagInterface) (reconcile.Result, error) {
	publicIpsSpec := clusterScope.GetPublicIp()
	var publicIpId string
	publicIpRef := clusterScope.GetPublicIpRef()
	var publicIpIds []string

	for _, publicIpSpec := range publicIpsSpec {
		publicIpId = publicIpSpec.ResourceId
		publicIpIds = append(publicIpIds, publicIpId)
	}
	clusterScope.V(2).Info("Check if the desired publicip exist")
	validPublicIpIds, err := publicIpSvc.ValidatePublicIpIds(publicIpIds)
	clusterScope.V(4).Info("Check public Ip Ids")
	if err != nil {
		return reconcile.Result{}, err
	}

	clusterScope.V(4).Info("Number of publicIp", "publicIpLength", len(publicIpsSpec))
	for _, publicIpSpec := range publicIpsSpec {
		publicIpName := publicIpSpec.Name + "-" + clusterScope.GetUID()
		tagKey := "Name"
		tagValue := publicIpName
		tag, err := tagSvc.ReadTag(tagKey, tagValue)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not get tag for OscCluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
		}
		if len(publicIpRef.ResourceMap) == 0 {
			publicIpRef.ResourceMap = make(map[string]string)
		}
		if publicIpSpec.ResourceId != "" {
			publicIpRef.ResourceMap[publicIpName] = publicIpSpec.ResourceId
		}
		_, resourceMapExist := publicIpRef.ResourceMap[publicIpName]
		if resourceMapExist {
			publicIpSpec.ResourceId = publicIpRef.ResourceMap[publicIpName]
		}

		publicIpId := publicIpRef.ResourceMap[publicIpName]
		if !Contains(validPublicIpIds, publicIpId) && tag == nil {
			publicIp, err := publicIpSvc.CreatePublicIp(publicIpName)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("%w Can not create publicIp for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
			}
			clusterScope.V(4).Info("Get publicIp", "publicip", publicIp)
			publicIpRef.ResourceMap[publicIpName] = publicIp.GetPublicIpId()
			publicIpSpec.ResourceId = publicIp.GetPublicIpId()
		}
	}
	return reconcile.Result{}, nil
}

// reconcileDeletePublicIp reconcile the destruction of the PublicIp of the cluster.
func reconcileDeletePublicIp(ctx context.Context, clusterScope *scope.ClusterScope, publicIpSvc security.OscPublicIpInterface) (reconcile.Result, error) {
	osccluster := clusterScope.OscCluster
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
	validPublicIpIds, err := publicIpSvc.ValidatePublicIpIds(publicIpIds)
	clusterScope.V(4).Info("Check validPublicIpIds", "validPublicIpIds", validPublicIpIds)
	if err != nil {
		return reconcile.Result{}, err
	}
	clusterScope.V(4).Info("Check publicIp Ids", "publicip", publicIpIds)
	clusterScope.V(4).Info("Number of publicIp", "publicIpLength", len(publicIpsSpec))
	for _, publicIpSpec := range publicIpsSpec {
		publicIpId := publicIpSpec.ResourceId
		clusterScope.V(4).Info("Check publicIp Id", "publicipid", publicIpId)
		publicIpName := publicIpSpec.Name + "-" + clusterScope.GetUID()
		if !Contains(validPublicIpIds, publicIpId) {
			controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
			return reconcile.Result{}, nil
		}
		err = publicIpSvc.CheckPublicIpUnlink(5, 120, publicIpId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not delete publicIp %s for Osccluster %s/%s", err, publicIpId, clusterScope.GetNamespace(), clusterScope.GetName())
		}
		clusterScope.V(2).Info("Delete the desired publicip", "publicIpName", publicIpName)
		err = publicIpSvc.DeletePublicIp(publicIpId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not delete publicIp for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
		}

	}
	return reconcile.Result{}, nil
}
