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

	"github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/net"
	tag "github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	osc "github.com/outscale/osc-sdk-go/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// getNetResourceId return the netId from the resourceMap base on resourceName (tag name + cluster object uid)
func getNetResourceId(resourceName string, clusterScope *scope.ClusterScope) (string, error) {
	netRef := clusterScope.GetNetRef()
	if netId, ok := netRef.ResourceMap[resourceName]; ok {
		return netId, nil
	} else {
		return "", fmt.Errorf("%s does not exist", resourceName)
	}
}

// checkNetFormatParameters check net parameters format (Tag format, cidr format, ..)
func checkNetFormatParameters(clusterScope *scope.ClusterScope) (string, error) {
	netSpec := clusterScope.GetNet()
	netSpec.SetDefaultValue()
	netName := netSpec.Name + "-" + clusterScope.GetUID()
	netTagName, err := tag.ValidateTagNameValue(netName)
	if err != nil {
		return netTagName, err
	}
	netIpRange := netSpec.IpRange
	_, err = infrastructurev1beta1.ValidateCidr(netIpRange)
	if err != nil {
		return netTagName, err
	}
	return "", nil
}

// reconcileNet reconcile the Net of the cluster.
func reconcileNet(ctx context.Context, clusterScope *scope.ClusterScope, netSvc net.OscNetInterface, tagSvc tag.OscTagInterface) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	netSpec := clusterScope.GetNet()
	netSpec.SetDefaultValue()
	netRef := clusterScope.GetNetRef()
	netName := netSpec.Name + "-" + clusterScope.GetUID()
	clusterName := netSpec.ClusterName + "-" + clusterScope.GetUID()
	if len(netRef.ResourceMap) == 0 {
		netRef.ResourceMap = make(map[string]string)
	}
	if netSpec.ResourceId != "" && netRef.ResourceMap[v1beta1.ManagedByKey(netSpec.ResourceId)] != v1beta1.ManagedByValueCapi {
		netRef.ResourceMap[netName] = netSpec.ResourceId
		return reconcile.Result{}, nil
	}
	var net *osc.Net
	var err error
	tagKey := "Name"
	tagValue := netName
	tag, err := tagSvc.ReadTag(ctx, tagKey, tagValue)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot get tag: %w", err)
	}
	if netSpec.ResourceId != "" {
		netRef.ResourceMap[netName] = netSpec.ResourceId
		netId := netSpec.ResourceId
		log.V(4).Info("Checking net", "netName", netName)
		net, err = netSvc.GetNet(ctx, netId)
		if err != nil {
			return reconcile.Result{}, err
		}
	}
	if (net == nil && tag == nil) || (netSpec.ResourceId == "" && tag == nil) {
		log.V(2).Info("Creating net", "netName", netName)
		net, err := netSvc.CreateNet(ctx, netSpec, clusterName, netName)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot create net: %w", err)
		}
		netRef.ResourceMap[netName] = net.GetNetId()
		netSpec.ResourceId = *net.NetId
		netRef.ResourceMap[netName] = net.GetNetId()
		netSpec.ResourceId = net.GetNetId()
		netRef.ResourceMap[v1beta1.ManagedByKey(net.GetNetId())] = v1beta1.ManagedByValueCapi
	}
	return reconcile.Result{}, nil
}

// reconcileDeleteNet reconcile the destruction of the Net of the cluster.
func reconcileDeleteNet(ctx context.Context, clusterScope *scope.ClusterScope, netSvc net.OscNetInterface) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	netSpec := clusterScope.GetNet()
	netSpec.SetDefaultValue()
	netId := netSpec.ResourceId
	netName := netSpec.Name + "-" + clusterScope.GetUID()
	netRef := clusterScope.GetNetRef()
	if netRef.ResourceMap[v1beta1.ManagedByKey(netId)] != v1beta1.ManagedByValueCapi {
		log.V(2).Info("Not deleting net because it's not managed by capi", "netName", netName)
		return reconcile.Result{}, nil
	}
	net, err := netSvc.GetNet(ctx, netId)
	if err != nil {
		return reconcile.Result{}, err
	}
	if net == nil {
		log.V(4).Info("The net is already deleted", "netName", netName)
		return reconcile.Result{}, nil
	}
	log.V(2).Info("Deleting net", "netName", netName)
	err = netSvc.DeleteNet(ctx, netId)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot delete net: %w", err)
	}
	return reconcile.Result{}, nil
}
