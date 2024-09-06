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
	"time"

	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/net"
	tag "github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag"
	osc "github.com/outscale/osc-sdk-go/v2"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// GetNatResourceId return the NatId from the resourceMap base on NatName (tag name + cluster object uid)
func getNatResourceId(resourceName string, clusterScope *scope.ClusterScope) (string, error) {
	natServiceRef := clusterScope.GetNatServiceRef()
	if natServiceId, ok := natServiceRef.ResourceMap[resourceName]; ok {
		return natServiceId, nil
	} else {
		return "", fmt.Errorf("%s does not exist", resourceName)
	}
}

// checkNatServiceOscDuplicateName check that there are no identical names already existing for natServices
func checkNatServiceOscDuplicateName(clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	natServicesSpec := clusterScope.GetNatServices()
	for _, natServiceSpec := range natServicesSpec {
		resourceNameList = append(resourceNameList, natServiceSpec.Name)
	}
	clusterScope.V(2).Info("Check unique nat service")
	duplicateResourceErr := alertDuplicate(resourceNameList)
	if duplicateResourceErr != nil {
		return duplicateResourceErr
	} else {
		return nil
	}
}

func checkNatFormatParameters(clusterScope *scope.ClusterScope) (string, error) {
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
	}
	return "", nil
}

// checkNatSubnetOscAssociateResourceName check that Nat Subnet dependencies tag name in both resource configuration are the same.
func checkNatSubnetOscAssociateResourceName(clusterScope *scope.ClusterScope) error {
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

	subnetsSpec := clusterScope.GetSubnet()
	for _, subnetSpec := range subnetsSpec {
		subnetName := subnetSpec.Name + "-" + clusterScope.GetUID()
		resourceNameList = append(resourceNameList, subnetName)
	}

	clusterScope.V(2).Info("Check match subnet with nat services")
	for _, natServiceSpec := range natServicesSpec {
		natSubnetName := natServiceSpec.SubnetName + "-" + clusterScope.GetUID()
		checkOscAssociate := Contains(resourceNameList, natSubnetName)
		if checkOscAssociate {
			return nil
		} else {
			return fmt.Errorf("%s subnet does not exist in natService", natSubnetName)
		}
	}
	return nil
}

// reconcileNatService reconcile the NatService of the cluster.
func reconcileNatService(ctx context.Context, clusterScope *scope.ClusterScope, natServiceSvc net.OscNatServiceInterface, tagSvc tag.OscTagInterface) (reconcile.Result, error) {
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
		tagKey := "Name"
		tagValue := natServiceName
		tag, err := tagSvc.ReadTag(tagKey, tagValue)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not get tag for OscCluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
		}
		if natServiceSpec.ResourceId != "" {
			natServiceRef.ResourceMap[natServiceName] = natServiceSpec.ResourceId
			natServiceId := natServiceSpec.ResourceId
			clusterScope.V(4).Info("Get natService Id", "natService", natServiceId)
			clusterScope.V(2).Info("Check if the desired natService exist")
			natService, err = natServiceSvc.GetNatService(natServiceId)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		if (natService == nil && tag == nil) || (natServiceSpec.ResourceId == "" && tag == nil) {
			clusterScope.V(4).Info("Create the desired natService", "natServiceName", natServiceName)
			networkSpec := clusterScope.GetNetwork()
			clusterName := networkSpec.ClusterName + "-" + clusterScope.GetUID()
			natService, err := natServiceSvc.CreateNatService(publicIpId, subnetId, natServiceName, clusterName)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("%w Can not create natService for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
			}
			natServiceRef.ResourceMap[natServiceName] = natService.GetNatServiceId()
			natServiceSpec.ResourceId = natService.GetNatServiceId()
		}
	}
	return reconcile.Result{}, nil
}

// reconcileDeleteNatService reconcile the destruction of the NatService of the cluster.
func reconcileDeleteNatService(ctx context.Context, clusterScope *scope.ClusterScope, natServiceSvc net.OscNatServiceInterface) (reconcile.Result, error) {
	natServiceRef := clusterScope.GetNatServiceRef()

	for natServiceName, natServiceId := range natServiceRef.ResourceMap {
		natService, err := natServiceSvc.GetNatService(natServiceId)
		if err != nil {
			return reconcile.Result{}, err
		}
		if natService == nil {
			clusterScope.V(2).Info("The desired natService does not exist anymore", "natServiceName", natServiceName)
			delete(natServiceRef.ResourceMap, natServiceName)
			return reconcile.Result{}, nil
		}
		clusterScope.V(2).Info("Delete the desired natService", "natServiceName", natServiceName)
		err = natServiceSvc.DeleteNatService(natServiceId)
		if err != nil {
			return reconcile.Result{RequeueAfter: 30 * time.Second}, fmt.Errorf("%w cannot delete natService for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
		}
		delete(natServiceRef.ResourceMap, natServiceName)
	}
	return reconcile.Result{}, nil
}
