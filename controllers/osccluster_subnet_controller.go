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
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/net"
	tag "github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// getSubnetResourceId return the subnetId from the resourceMap base on subnetName (tag name + cluster object uid)
func getSubnetResourceId(resourceName string, clusterScope *scope.ClusterScope) (string, error) {
	subnetRef := clusterScope.GetSubnetRef()
	if subnetId, ok := subnetRef.ResourceMap[resourceName]; ok {
		return subnetId, nil
	} else {
		return "", fmt.Errorf("%s does not exist", resourceName)
	}
}

// checkSubnetFormatParameters check Subnet parameters format (Tag format, cidr format, ..)
func checkSubnetFormatParameters(clusterScope *scope.ClusterScope) (string, error) {
	var subnetsSpec []*infrastructurev1beta1.OscSubnet
	networkSpec := clusterScope.GetNetwork()
	if networkSpec.Subnets == nil {
		networkSpec.SetSubnetDefaultValue()
		subnetsSpec = networkSpec.Subnets
	} else {
		subnetsSpec = clusterScope.GetSubnet()
	}
	networkSpec.SetSubnetSubregionNameDefaultValue()
	for _, subnetSpec := range subnetsSpec {
		subnetName := subnetSpec.Name + "-" + clusterScope.GetUID()
		subnetTagName, err := tag.ValidateTagNameValue(subnetName)
		if err != nil {
			return subnetTagName, err
		}
		subnetIpRange := subnetSpec.IpSubnetRange
		_, err = infrastructurev1beta1.ValidateCidr(subnetIpRange)
		if err != nil {
			return subnetTagName, err
		}
		subnetSubregionName := subnetSpec.SubregionName
		_, err = infrastructurev1beta1.ValidateSubregionName(subnetSubregionName)
		if err != nil {
			return subnetTagName, err
		}
	}
	return "", nil
}

// checkSubnetOscDuplicateName check that there are not the same name for subnet
func checkSubnetOscDuplicateName(clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	subnetsSpec := clusterScope.GetSubnet()
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
func reconcileSubnet(ctx context.Context, clusterScope *scope.ClusterScope, subnetSvc net.OscSubnetInterface, tagSvc tag.OscTagInterface) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	netSpec := clusterScope.GetNet()
	netSpec.SetDefaultValue()
	netName := netSpec.Name + "-" + clusterScope.GetUID()
	netId, err := getNetResourceId(netName, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	subnetsSpec := clusterScope.GetSubnet()

	subnetRef := clusterScope.GetSubnetRef()
	networkSpec := clusterScope.GetNetwork()
	clusterName := networkSpec.ClusterName + "-" + clusterScope.GetUID()
	var subnetIds []string
	log.V(4).Info("Checking subnet")
	subnetIds, err = subnetSvc.GetSubnetIdsFromNetIds(ctx, netId)
	log.V(4).Info("Get subnetIds", "subnetIds", subnetIds)
	if err != nil {
		return reconcile.Result{}, err
	}
	log.V(4).Info("Number of subnet", "subnet_length", len(subnetsSpec))
	if len(subnetRef.ResourceMap) == 0 {
		subnetRef.ResourceMap = make(map[string]string)
	}
	for _, subnetSpec := range subnetsSpec {
		subnetName := subnetSpec.Name + "-" + clusterScope.GetUID()
		if subnetSpec.ResourceId != "" && subnetSpec.SkipReconcile {
			subnetRef.ResourceMap[subnetName] = subnetSpec.ResourceId
			continue
		}
		tagKey := "Name"
		tagValue := subnetName
		tag, err := tagSvc.ReadTag(ctx, tagKey, tagValue)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot get tag: %w", err)
		}
		subnetId := subnetSpec.ResourceId
		log.V(4).Info("Get subnetId", "subnetId", subnetId)
		if subnetSpec.ResourceId != "" {
			subnetRef.ResourceMap[subnetName] = subnetSpec.ResourceId
		}
		_, resourceMapExist := subnetRef.ResourceMap[subnetName]
		if resourceMapExist {
			subnetSpec.ResourceId = subnetRef.ResourceMap[subnetName]
		}
		if !slices.Contains(subnetIds, subnetId) && tag == nil {
			log.V(2).Info("Creating subnet", "subnetName", subnetName)
			subnet, err := subnetSvc.CreateSubnet(ctx, subnetSpec, netId, clusterName, subnetName)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("cannot create subnet: %w", err)
			}
			subnetRef.ResourceMap[subnetName] = subnet.GetSubnetId()
			subnetSpec.ResourceId = subnet.GetSubnetId()
		}
	}
	return reconcile.Result{}, nil
}

// reconcileDeleteSubnet reconcile the destruction of the Subnet of the cluster.
func reconcileDeleteSubnet(ctx context.Context, clusterScope *scope.ClusterScope, subnetSvc net.OscSubnetInterface) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	subnetsSpec := clusterScope.GetSubnet()
	netSpec := clusterScope.GetNet()
	netSpec.SetDefaultValue()
	netName := netSpec.Name + "-" + clusterScope.GetUID()

	networkSpec := clusterScope.GetNetwork()
	if networkSpec.Subnets == nil {
		networkSpec.SetSubnetDefaultValue()
		subnetsSpec = networkSpec.Subnets
	}
	netId, err := getNetResourceId(netName, clusterScope)
	if err != nil {
		log.V(3).Info("No net found, skipping subnet deletion")
		return reconcile.Result{}, nil //nolint: nilerr
	}
	subnetIds, err := subnetSvc.GetSubnetIdsFromNetIds(ctx, netId)
	if err != nil {
		return reconcile.Result{}, err
	}
	log.V(4).Info("Number of subnet", "subnet_length", len(subnetsSpec))
	for _, subnetSpec := range subnetsSpec {
		subnetId := subnetSpec.ResourceId
		subnetName := subnetSpec.Name + "-" + clusterScope.GetUID()
		if subnetSpec.SkipReconcile {
			log.V(2).Info("Not deleting the desired subnet because skip reconcile true", "subnetName", subnetName)
			continue
		}
		if !slices.Contains(subnetIds, subnetId) {
			log.V(2).Info("subnet does not exist anymore", "subnetName", subnetName)
			return reconcile.Result{}, nil
		}
		log.V(2).Info("Deleting subnet", "subnetName", subnetName)
		err = subnetSvc.DeleteSubnet(ctx, subnetId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot delete subnet: %w", err)
		}
	}
	return reconcile.Result{}, nil
}
