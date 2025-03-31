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
	"errors"
	"fmt"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/net"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// reconcileSubnet reconcile the subnet of the cluster.
func (r *OscClusterReconciler) reconcileSubnets(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	if !clusterScope.NeedReconciliation(infrastructurev1beta1.ReconcilerSubnet) {
		log.V(4).Info("No need for subnet reconciliation")
		return reconcile.Result{}, nil
	}
	errs := infrastructurev1beta1.ValidateSubnets(clusterScope.GetSubnets(), clusterScope.GetNet())
	if len(errs) > 0 {
		return reconcile.Result{}, errs.ToAggregate()
	}
	netId, err := r.Tracker.getNetId(ctx, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	for _, subnetSpec := range clusterScope.GetSubnets() {
		_, err := r.Tracker.getSubnet(ctx, subnetSpec, clusterScope)
		switch {
		case errors.Is(err, ErrNoResourceFound):
		case err != nil:
			return reconcile.Result{}, fmt.Errorf("reconcile subnet: %w", err)
		default:
			return reconcile.Result{}, nil
		}

		log.V(2).Info("Creating subnet", "role", subnetSpec.GetRole(), "subregion", subnetSpec.SubregionName)
		subnet, err := r.Cloud.Subnet(ctx, *clusterScope).CreateSubnet(ctx, &subnetSpec, netId, clusterScope.GetName(), clusterScope.GetSubnetName(subnetSpec))
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot create subnet: %w", err)
		}
		log.V(2).Info("Created subnet", "subnetId", subnet.GetSubnetId())
	}
	return reconcile.Result{}, nil
}

// reconcileDeleteSubnet reconcile the destruction of the Subnet of the cluster.
func (r *OscClusterReconciler) reconcileDeleteSubnets(ctx context.Context, clusterScope *scope.ClusterScope, subnetSvc net.OscSubnetInterface) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	netSpec := clusterScope.GetNet()
	if netSpec.UseExisting {
		log.V(4).Info("Not deleting existing subnets")
		return reconcile.Result{}, nil
	}
	subnetsSpec := clusterScope.GetSubnets()
	for _, subnetSpec := range subnetsSpec {
		subnet, err := r.Tracker.getSubnet(ctx, subnetSpec, clusterScope)
		switch {
		case errors.Is(err, ErrNoResourceFound):
		case errors.Is(err, ErrMissingResource):
		case err != nil:
			return reconcile.Result{}, fmt.Errorf("reconcile delete subnet: %w", err)
		}
		subnetId := subnet.GetSubnetId()
		log.V(2).Info("Deleting subnet", "subnetId", subnetId)
		err = subnetSvc.DeleteSubnet(ctx, subnetId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot delete subnet: %w", err)
		}
	}
	return reconcile.Result{}, nil
}
