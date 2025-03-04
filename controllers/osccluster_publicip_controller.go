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
	"slices"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/security"
	tag "github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	"github.com/outscale/cluster-api-provider-outscale/cloud/utils"
	ctrl "sigs.k8s.io/controller-runtime"
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
		publicIpTagName, err := tag.ValidateTagNameValue(publicIpName)
		if err != nil {
			return publicIpTagName, err
		}
	}
	return "", nil
}

// checkPublicIpOscAssociateResourceName check that PublicIp dependencies tag name in both resource configuration are the same.
func checkPublicIpOscAssociateResourceName(clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	natServicesSpec := clusterScope.GetNatServices()

	for _, natServiceSpec := range natServicesSpec {
		natPublicIpName := natServiceSpec.PublicIpName + "-" + clusterScope.GetUID()

		var publicIpsSpec []*infrastructurev1beta1.OscPublicIp
		networkSpec := clusterScope.GetNetwork()
		publicIpsSpec = networkSpec.PublicIps
		for _, publicIpSpec := range publicIpsSpec {
			publicIpName := publicIpSpec.Name + "-" + clusterScope.GetUID()
			resourceNameList = append(resourceNameList, publicIpName)
		}

		checkOscAssociate := slices.Contains(resourceNameList, natPublicIpName)
		if checkOscAssociate {
			return nil
		} else {
			return fmt.Errorf("publicIp %s does not exist in natService", natPublicIpName)
		}
	}
	return nil
}

// checkPublicIpOscDuplicateName check that there are not the same name for PublicIp resource.
func checkPublicIpOscDuplicateName(clusterScope *scope.ClusterScope) error {
	return utils.CheckDuplicates(clusterScope.GetPublicIp(), func(ip *infrastructurev1beta1.OscPublicIp) string {
		return ip.Name
	})
}

// reconcilePublicIp reconcile the PublicIp of the cluster.
func reconcilePublicIp(ctx context.Context, clusterScope *scope.ClusterScope, publicIpSvc security.OscPublicIpInterface, tagSvc tag.OscTagInterface) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	publicIpsSpec := clusterScope.GetPublicIp()

	publicIpIds := make([]string, 0, len(publicIpsSpec))
	for _, publicIpSpec := range publicIpsSpec {
		if publicIpSpec.ResourceId != "" {
			publicIpIds = append(publicIpIds, publicIpSpec.ResourceId)
		}
	}
	var validPublicIpIds []string
	if len(publicIpIds) > 0 {
		var err error
		validPublicIpIds, err = publicIpSvc.ValidatePublicIpIds(ctx, publicIpIds)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	publicIpRef := clusterScope.GetPublicIpRef()
	if publicIpRef.ResourceMap == nil {
		publicIpRef.ResourceMap = make(map[string]string)
	}

	for _, publicIpSpec := range publicIpsSpec {
		publicIpId := publicIpSpec.ResourceId
		publicIpName := publicIpSpec.Name + "-" + clusterScope.GetUID()

		if publicIpId != "" && slices.Contains(validPublicIpIds, publicIpId) {
			publicIpRef.ResourceMap[publicIpName] = publicIpId
			continue
		}

		tag, err := tagSvc.ReadTag(ctx, "Name", publicIpName)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot get tag: %w", err)
		}
		if tag.GetResourceId() != "" {
			publicIpSpec.ResourceId = tag.GetResourceId()
			publicIpRef.ResourceMap[publicIpName] = tag.GetResourceId()
			continue
		}

		log.V(3).Info("Creating publicIp")
		publicIp, err := publicIpSvc.CreatePublicIp(ctx, publicIpName)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot create publicIp: %w", err)
		}
		log.V(2).Info("Created publicIp", "publicIpId", publicIp.GetPublicIpId())
		publicIpRef.ResourceMap[publicIpName] = publicIp.GetPublicIpId()
		publicIpSpec.ResourceId = publicIp.GetPublicIpId()
	}
	return reconcile.Result{}, nil
}

// reconcileDeletePublicIp reconcile the destruction of the PublicIp of the cluster.
func reconcileDeletePublicIp(ctx context.Context, clusterScope *scope.ClusterScope, publicIpSvc security.OscPublicIpInterface) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
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
	validPublicIpIds, err := publicIpSvc.ValidatePublicIpIds(ctx, publicIpIds)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot validate publicips: %w", err)
	}
	log.V(4).Info("Check publicIp Ids", "publicip", publicIpIds)
	log.V(4).Info("Number of publicIp", "publicIpLength", len(publicIpsSpec))
	for _, publicIpSpec := range publicIpsSpec {
		publicIpId := publicIpSpec.ResourceId
		log.V(4).Info("Check publicIp Id", "publicipid", publicIpId)
		publicIpName := publicIpSpec.Name + "-" + clusterScope.GetUID()
		if !slices.Contains(validPublicIpIds, publicIpId) {
			return reconcile.Result{}, nil
		}
		err = publicIpSvc.CheckPublicIpUnlink(ctx, 5, 120, publicIpId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot check publicIp %s: %w", publicIpId, err)
		}
		log.V(2).Info("Deleting publicip", "publicIpName", publicIpName)
		err = publicIpSvc.DeletePublicIp(ctx, publicIpId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot delete publicIp %s: %w", publicIpId, err)
		}
	}
	return reconcile.Result{}, nil
}
