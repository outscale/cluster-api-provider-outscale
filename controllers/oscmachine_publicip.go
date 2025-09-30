/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package controllers

import (
	"context"
	"fmt"

	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// reconcileDeletePublicIp reconcile the destruction of the PublicIp of the cluster.
func (r *OscMachineReconciler) reconcileDeletePublicIp(ctx context.Context, clusterScope *scope.ClusterScope, machineScope *scope.MachineScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	if machineScope.GetVm().PublicIpPool != "" {
		log.V(4).Info("Not deleting publicip from pool")
		return reconcile.Result{}, nil
	}
	svc := r.Cloud.PublicIp(clusterScope.Tenant)
	for k, id := range r.Tracker.getPublicIps(machineScope) {
		ip, err := svc.GetPublicIp(ctx, id)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot get publicIp %s: %w", id, err)
		}
		if ip == nil {
			r.Tracker.untrackIP(machineScope, k)
			continue
		}
		log.V(3).Info("Deleting publicip")
		err = svc.DeletePublicIp(ctx, id)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot delete publicIp %s: %w", id, err)
		}
		log.V(2).Info("Deleted publicip", "publicIpId", id)
		r.Tracker.untrackIP(machineScope, k)
	}
	return reconcile.Result{}, nil
}
