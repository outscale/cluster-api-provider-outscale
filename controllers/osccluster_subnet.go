/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package controllers

import (
	"context"
	"fmt"

	infrastructurev1beta2 "github.com/outscale/cluster-api-provider-outscale/api/v1beta2"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// reconcileSubnet reconcile the subnet of the cluster.
func (r *OscClusterReconciler) reconcileSubnets(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	if !clusterScope.NeedReconciliation(infrastructurev1beta2.ReconcilerSubnet) {
		log.V(4).Info("No need for subnet reconciliation")
		return reconcile.Result{}, nil
	}
	log.V(4).Info("Reconciling subnets")

	netId, err := r.Tracker.getNetId(ctx, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	svc := r.Cloud.Net(clusterScope.Tenant)
	for _, subnetSpec := range clusterScope.GetSubnets() {
		subnet, err := r.Tracker.getSubnet(ctx, subnetSpec, clusterScope)
		switch {
		case IsNotFound(err) && !clusterScope.GetSpec().UseExisting.Net:
		case err != nil:
			return reconcile.Result{}, fmt.Errorf("get existing: %w", err)
		default:
			log.V(4).Info("Found existing subnet", "roles", subnetSpec.Roles, "subregion", subnetSpec.Subregion, "subnetId", subnet.SubnetId)
			continue
		}
		subnetSpec.Subregion = clusterScope.GetSubnetSubregion(subnetSpec)
		log.V(3).Info("Creating subnet", "roles", subnetSpec.Roles, "subregion", subnetSpec.Subregion)
		subnet, err = svc.CreateSubnet(ctx, subnetSpec, netId, clusterScope.GetUID(), clusterScope.GetSubnetName(subnetSpec))
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot create subnet: %w", err)
		}
		log.V(2).Info("Created subnet", "subnetId", subnet.SubnetId)
		r.Tracker.setSubnetId(clusterScope, subnetSpec, subnet.SubnetId)
		r.Recorder.Eventf(clusterScope.OscCluster, corev1.EventTypeNormal, infrastructurev1beta2.SubnetCreatedReason, "Subnet created %v %s", subnetSpec.Roles, subnetSpec.Subregion)
	}

	// add failureDomains
	for _, subnetSpec := range clusterScope.GetSubnets() {
		if clusterScope.SubnetHasRole(subnetSpec, infrastructurev1beta2.RoleControlPlane) {
			clusterScope.SetFailureDomain(clusterScope.GetSubnetSubregion(subnetSpec), true)
		}
	}

	clusterScope.SetReconciliationGeneration(infrastructurev1beta2.ReconcilerSubnet)
	return reconcile.Result{}, nil
}

// reconcileDeleteSubnet reconcile the destruction of the Subnet of the cluster.
func (r *OscClusterReconciler) reconcileDeleteSubnets(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	if clusterScope.GetSpec().UseExisting.Net {
		log.V(4).Info("Not deleting existing subnets")
		return reconcile.Result{}, nil
	}
	svc := r.Cloud.Net(clusterScope.Tenant)
	subnetsSpec := clusterScope.GetSubnets()
	for _, subnetSpec := range subnetsSpec {
		subnet, err := r.Tracker.getSubnet(ctx, subnetSpec, clusterScope)
		switch {
		case IsNotFound(err):
			continue
		case err != nil:
			return reconcile.Result{}, fmt.Errorf("find existing: %w", err)
		}
		subnetId := subnet.SubnetId
		log.V(2).Info("Deleting subnet", "subnetId", subnetId)
		err = svc.DeleteSubnet(ctx, subnetId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot delete subnet: %w", err)
		}
	}
	return reconcile.Result{}, nil
}
