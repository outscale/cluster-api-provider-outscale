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
	corev1 "k8s.io/api/core/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
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
	log.V(4).Info("Reconciling subnets")

	netId, err := r.Tracker.getNetId(ctx, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	svc := r.Cloud.Subnet(ctx, *clusterScope)
	for _, subnetSpec := range clusterScope.GetSubnets() {
		subnet, err := r.Tracker.getSubnet(ctx, subnetSpec, clusterScope)
		switch {
		case errors.Is(err, ErrNoResourceFound):
		case err != nil:
			return reconcile.Result{}, fmt.Errorf("get existing: %w", err)
		default:
			log.V(4).Info("Found existing subnet", "roles", subnetSpec.Roles, "subregion", subnetSpec.SubregionName, "subnetId", subnet.GetSubnetId())
			continue
		}
		subnetSpec.SubregionName = clusterScope.GetSubnetSubregion(subnetSpec)
		log.V(3).Info("Creating subnet", "roles", subnetSpec.Roles, "subregion", subnetSpec.SubregionName)
		subnet, err = svc.CreateSubnet(ctx, subnetSpec, netId, clusterScope.GetUID(), clusterScope.GetSubnetName(subnetSpec))
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot create subnet: %w", err)
		}
		log.V(2).Info("Created subnet", "subnetId", subnet.GetSubnetId())
		r.Tracker.setSubnetId(clusterScope, subnetSpec, subnet.GetSubnetId())
		r.Recorder.Eventf(clusterScope.OscCluster, corev1.EventTypeNormal, infrastructurev1beta1.SubnetCreatedReason, "Subnet created %v %s", subnetSpec.Roles, subnetSpec.SubregionName)
	}

	// add failureDomains
	for _, subnetSpec := range clusterScope.GetSubnets() {
		if clusterScope.SubnetHasRole(subnetSpec, infrastructurev1beta1.RoleControlPlane) {
			clusterScope.SetFailureDomain(clusterScope.GetSubnetSubregion(subnetSpec), clusterv1.FailureDomainSpec{
				ControlPlane: true,
			})
		}
	}

	clusterScope.SetReconciliationGeneration(infrastructurev1beta1.ReconcilerSubnet)
	return reconcile.Result{}, nil
}

// reconcileDeleteSubnet reconcile the destruction of the Subnet of the cluster.
func (r *OscClusterReconciler) reconcileDeleteSubnets(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	if clusterScope.GetNetwork().UseExisting.Net {
		log.V(4).Info("Not deleting existing subnets")
		return reconcile.Result{}, nil
	}
	svc := r.Cloud.Subnet(ctx, *clusterScope)
	subnetsSpec := clusterScope.GetSubnets()
	for _, subnetSpec := range subnetsSpec {
		subnet, err := r.Tracker.getSubnet(ctx, subnetSpec, clusterScope)
		switch {
		case errors.Is(err, ErrNoResourceFound) || errors.Is(err, ErrMissingResource):
			continue
		case err != nil:
			return reconcile.Result{}, fmt.Errorf("find existing: %w", err)
		}
		subnetId := subnet.GetSubnetId()
		log.V(2).Info("Deleting subnet", "subnetId", subnetId)
		err = svc.DeleteSubnet(ctx, subnetId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot delete subnet: %w", err)
		}
	}
	return reconcile.Result{}, nil
}
