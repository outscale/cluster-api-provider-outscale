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

	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/net"
	tag "github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag"
	osc "github.com/outscale/osc-sdk-go/v2"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// getInternetServiceResourceId return the InternetServiceId from the resourceMap base on resourceName (tag name + cluster object uid)
func getInternetServiceResourceId(resourceName string, clusterScope *scope.ClusterScope) (string, error) {
	internetServiceRef := clusterScope.GetInternetServiceRef()
	if internetServiceId, ok := internetServiceRef.ResourceMap[resourceName]; ok {
		return internetServiceId, nil
	} else {
		return "", fmt.Errorf("%s does not exist", resourceName)
	}
}

// checkInternetServiceFormatParameters check InternetService parameters format
func checkInternetServiceFormatParameters(clusterScope *scope.ClusterScope) (string, error) {
	internetServiceSpec := clusterScope.GetInternetService()
	internetServiceSpec.SetDefaultValue()
	internetServiceName := internetServiceSpec.Name + "-" + clusterScope.GetUID()
	clusterScope.V(2).Info("Check Internet Service parameters")
	internetServiceTagName, err := tag.ValidateTagNameValue(internetServiceName)
	if err != nil {
		return internetServiceTagName, err
	}
	return "", nil
}

// ReconcileInternetService reconcile the InternetService of the cluster.
func reconcileInternetService(ctx context.Context, clusterScope *scope.ClusterScope, internetServiceSvc net.OscInternetServiceInterface, tagSvc tag.OscTagInterface) (reconcile.Result, error) {
	internetServiceSpec := clusterScope.GetInternetService()
	internetServiceRef := clusterScope.GetInternetServiceRef()
	internetServiceName := internetServiceSpec.Name + "-" + clusterScope.GetUID()
	var internetService *osc.InternetService
	netSpec := clusterScope.GetNet()
	netSpec.SetDefaultValue()
	netName := netSpec.Name + "-" + clusterScope.GetUID()
	netId, err := getNetResourceId(netName, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	if len(internetServiceRef.ResourceMap) == 0 {
		internetServiceRef.ResourceMap = make(map[string]string)
	}
	tagKey := "Name"
	tagValue := internetServiceName
	tag, err := tagSvc.ReadTag(tagKey, tagValue)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("%w Can not get tag for OscCluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
	}
	if internetServiceSpec.ResourceId != "" {
		internetServiceRef.ResourceMap[internetServiceName] = internetServiceSpec.ResourceId
		internetServiceId := internetServiceSpec.ResourceId
		clusterScope.V(2).Info("Check if the desired internetservice exist", "internetserviceName", internetServiceName)
		clusterScope.V(4).Info("Get internetServiceId", "internetservice", internetServiceRef.ResourceMap)
		internetService, err = internetServiceSvc.GetInternetService(internetServiceId)
		if err != nil {
			return reconcile.Result{}, err
		}
	}
	if (internetService == nil && tag == nil) || (internetServiceSpec.ResourceId == "" && tag == nil) {
		clusterScope.V(2).Info("Create the desired internetservice", "internetServiceName", internetServiceName)
		internetService, err := internetServiceSvc.CreateInternetService(internetServiceName)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not create internetservice for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
		}
		clusterScope.V(2).Info("Link the desired internetservice with a net", "internetServiceName", internetServiceName)
		err = internetServiceSvc.LinkInternetService(*internetService.InternetServiceId, netId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not link internetService with net for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
		}
		internetServiceRef.ResourceMap[internetServiceName] = internetService.GetInternetServiceId()
		internetServiceSpec.ResourceId = internetService.GetInternetServiceId()
	}
	return reconcile.Result{}, nil
}

// reconcileDeleteInternetService reconcile the destruction of the InternetService of the cluster.
func reconcileDeleteInternetService(ctx context.Context, clusterScope *scope.ClusterScope, internetServiceSvc net.OscInternetServiceInterface) (reconcile.Result, error) {
	osccluster := clusterScope.OscCluster
	internetServiceSpec := clusterScope.GetInternetService()
	internetServiceSpec.SetDefaultValue()

	netSpec := clusterScope.GetNet()
	netSpec.SetDefaultValue()
	netName := netSpec.Name + "-" + clusterScope.GetUID()

	netId, err := getNetResourceId(netName, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}

	internetServiceId := internetServiceSpec.ResourceId
	internetServiceName := internetServiceSpec.Name
	internetService, err := internetServiceSvc.GetInternetService(internetServiceId)
	if err != nil {
		return reconcile.Result{}, err
	}
	if internetService == nil {
		clusterScope.V(2).Info("The desired internetservice does not exist anymore", "internetServiceName", internetServiceName)
		controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
		return reconcile.Result{}, nil
	}
	clusterScope.V(2).Info("Unlink the desired internetservice", "internetServiceName", internetServiceName)
	err = internetServiceSvc.UnlinkInternetService(internetServiceId, netId)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("%w Can not unlink internetService and net for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
	}
	clusterScope.V(2).Info("Delete the desired internetservice", "internetServiceName", internetServiceName)
	err = internetServiceSvc.DeleteInternetService(internetServiceId)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("%w Can not delete internetService for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
	}
	return reconcile.Result{}, nil
}
