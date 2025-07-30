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
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/net"
	tag "github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
		clusterScope.V(2).Info("Check subnet name parameters")
		subnetTagName, err := tag.ValidateTagNameValue(subnetName)
		if err != nil {
			return subnetTagName, err
		}
		subnetIpRange := subnetSpec.IpSubnetRange
		clusterScope.V(2).Info("Check subnet ipsubnetrange parameters")
		_, err = infrastructurev1beta1.ValidateCidr(subnetIpRange)
		if err != nil {
			return subnetTagName, err
		}
		subnetSubregionName := subnetSpec.SubregionName
		clusterScope.V(2).Info("Check subnet subregion parameters")
		err = infrastructurev1beta1.ValidateSubregionName(subnetSubregionName)
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
	clusterScope.V(2).Info("Check unique subnet")
	duplicateResourceErr := alertDuplicate(resourceNameList)
	if duplicateResourceErr != nil {
		return duplicateResourceErr
	} else {
		return nil
	}
}

// reconcileSubnet reconcile the subnet of the cluster.
func reconcileSubnet(ctx context.Context, clusterScope *scope.ClusterScope, subnetSvc net.OscSubnetInterface, tagSvc tag.OscTagInterface) (reconcile.Result, error) {
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
	clusterScope.V(2).Info("Check if the desired subnet exist")
	subnetIds, err = subnetSvc.GetSubnetIdsFromNetIds(netId)
	clusterScope.V(4).Info("Get subnetIds", "subnetIds", subnetIds)
	if err != nil {
		return reconcile.Result{}, err
	}
	clusterScope.V(4).Info("Number of subnet", "subnet_length", len(subnetsSpec))
	for _, subnetSpec := range subnetsSpec {
		subnetName := subnetSpec.Name + "-" + clusterScope.GetUID()
		tagKey := "Name"
		tagValue := subnetName
		tag, err := tagSvc.ReadTag(tagKey, tagValue)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not get tag for OscCluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
		}
		subnetId := subnetSpec.ResourceId
		clusterScope.V(4).Info("Get subnetId", "subnetId", subnetId)
		if len(subnetRef.ResourceMap) == 0 {
			subnetRef.ResourceMap = make(map[string]string)
		}
		if subnetSpec.ResourceId != "" {
			subnetRef.ResourceMap[subnetName] = subnetSpec.ResourceId
		}
		_, resourceMapExist := subnetRef.ResourceMap[subnetName]
		if resourceMapExist {
			subnetSpec.ResourceId = subnetRef.ResourceMap[subnetName]
		}
		if !Contains(subnetIds, subnetId) && tag == nil {
			clusterScope.V(2).Info("Create the desired subnet", "subnetName", subnetName)
			subnet, err := subnetSvc.CreateSubnet(subnetSpec, netId, clusterName, subnetName)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("%w Can not create subnet for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
			}
			subnetRef.ResourceMap[subnetName] = subnet.GetSubnetId()
			subnetSpec.ResourceId = subnet.GetSubnetId()
		}
	}
	return reconcile.Result{}, nil
}

// reconcileDeleteSubnet reconcile the destruction of the Subnet of the cluster.
func reconcileDeleteSubnet(ctx context.Context, clusterScope *scope.ClusterScope, subnetSvc net.OscSubnetInterface) (reconcile.Result, error) {
	osccluster := clusterScope.OscCluster
	netSpec := clusterScope.GetNet()
	netSpec.SetDefaultValue()
	netName := netSpec.Name + "-" + clusterScope.GetUID()

	networkSpec := clusterScope.GetNetwork()
	var subnetsSpec []*infrastructurev1beta1.OscSubnet
	if networkSpec.Subnets == nil {
		networkSpec.SetSubnetDefaultValue()
		subnetsSpec = networkSpec.Subnets
	} else {
		subnetsSpec = clusterScope.GetSubnet()
	}
	netId, err := getNetResourceId(netName, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	subnetIds, err := subnetSvc.GetSubnetIdsFromNetIds(netId)
	if err != nil {
		return reconcile.Result{}, err
	}
	clusterScope.V(4).Info("Number of subnet", "subnet_length", len(subnetsSpec))
	for _, subnetSpec := range subnetsSpec {
		subnetId := subnetSpec.ResourceId
		subnetName := subnetSpec.Name + "-" + clusterScope.GetUID()
		if !Contains(subnetIds, subnetId) {
			clusterScope.V(2).Info("the desired subnet does not exist anymore", "subnetName", subnetName)
			controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
			return reconcile.Result{}, nil
		}
		err = subnetSvc.DeleteSubnet(subnetId)
		if err != nil {
			clusterScope.V(2).Info("Delete te desired subnet", "subnetName", subnetName)
			return reconcile.Result{}, fmt.Errorf("%w Can not delete subnet for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
		}
	}
	return reconcile.Result{}, nil
}
