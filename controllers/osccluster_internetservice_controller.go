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

	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/net"
	tag "github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	osc "github.com/outscale/osc-sdk-go/v2"
	ctrl "sigs.k8s.io/controller-runtime"
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
	internetServiceTagName, err := tag.ValidateTagNameValue(internetServiceName)
	if err != nil {
		return internetServiceTagName, err
	}
	return "", nil
}

// ReconcileInternetService reconcile the InternetService of the cluster.
func reconcileInternetService(ctx context.Context, clusterScope *scope.ClusterScope, internetServiceSvc net.OscInternetServiceInterface, tagSvc tag.OscTagInterface) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	internetServiceSpec := clusterScope.GetInternetService()
	internetServiceRef := clusterScope.GetInternetServiceRef()
	internetServiceName := internetServiceSpec.Name + "-" + clusterScope.GetUID()
	if len(internetServiceRef.ResourceMap) == 0 {
		internetServiceRef.ResourceMap = make(map[string]string)
	}
	if internetServiceSpec.ResourceId != "" && internetServiceSpec.SkipReconcile {
		internetServiceRef.ResourceMap[internetServiceName] = internetServiceSpec.ResourceId
		return reconcile.Result{}, nil
	}
	var internetService *osc.InternetService
	netSpec := clusterScope.GetNet()
	netSpec.SetDefaultValue()
	netName := netSpec.Name + "-" + clusterScope.GetUID()
	netId, err := getNetResourceId(netName, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	tagKey := "Name"
	tagValue := internetServiceName
	tag, err := tagSvc.ReadTag(ctx, tagKey, tagValue)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot get tag: %w", err)
	}
	if internetServiceSpec.ResourceId != "" {
		internetServiceRef.ResourceMap[internetServiceName] = internetServiceSpec.ResourceId
		internetServiceId := internetServiceSpec.ResourceId
		log.V(4).Info("Checking internetservice", "internetserviceName", internetServiceName)
		internetService, err = internetServiceSvc.GetInternetService(ctx, internetServiceId)
		if err != nil {
			return reconcile.Result{}, err
		}
	}
	if (internetService == nil && tag == nil) || (internetServiceSpec.ResourceId == "" && tag == nil) {
		log.V(2).Info("Creating internetservice", "internetServiceName", internetServiceName)
		internetService, err := internetServiceSvc.CreateInternetService(ctx, internetServiceName)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot create internetservice: %w", err)
		}
		log.V(2).Info("Linking internetservice", "internetServiceName", internetServiceName, "netId", netId)
		err = internetServiceSvc.LinkInternetService(ctx, *internetService.InternetServiceId, netId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot link internetService: %w", err)
		}
		internetServiceRef.ResourceMap[internetServiceName] = internetService.GetInternetServiceId()
		internetServiceSpec.ResourceId = internetService.GetInternetServiceId()
	}
	return reconcile.Result{}, nil
}

// reconcileDeleteInternetService reconcile the destruction of the InternetService of the cluster.
func reconcileDeleteInternetService(ctx context.Context, clusterScope *scope.ClusterScope, internetServiceSvc net.OscInternetServiceInterface) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	internetServiceSpec := clusterScope.GetInternetService()
	internetServiceSpec.SetDefaultValue()
	internetServiceId := internetServiceSpec.ResourceId
	internetServiceName := internetServiceSpec.Name
	if internetServiceId != "" && internetServiceSpec.SkipReconcile {
		log.V(2).Info("Not unlinking/deleting the desired internetservice because skip reconcile is true", "internetServiceName", internetServiceName)
		return reconcile.Result{}, nil
	}

	netSpec := clusterScope.GetNet()
	netSpec.SetDefaultValue()
	netName := netSpec.Name + "-" + clusterScope.GetUID()
	netId, err := getNetResourceId(netName, clusterScope)
	if err != nil {
		log.V(3).Info("No net found, skipping internet service deletion")
		return reconcile.Result{}, nil //nolint: nilerr
	}

	internetService, err := internetServiceSvc.GetInternetService(ctx, internetServiceId)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("get internetservice: %w", err)
	}
	if internetService == nil {
		log.V(2).Info("The internetservice is already deleted", "internetServiceName", internetServiceName)
		return reconcile.Result{}, nil
	}
	log.V(2).Info("Unlinking internetservice", "internetServiceName", internetServiceName)
	err = internetServiceSvc.UnlinkInternetService(ctx, internetServiceId, netId)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot unlink internetService and net: %w", err)
	}
	log.V(2).Info("Deleting internetservice", "internetServiceName", internetServiceName)
	err = internetServiceSvc.DeleteInternetService(ctx, internetServiceId)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot delete internetService: %w", err)
	}

	return reconcile.Result{}, nil
}
