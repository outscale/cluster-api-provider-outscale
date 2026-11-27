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
)

// reconcileDeletePublicIp reconcile the destruction of the PublicIp of the cluster.
func (r *OscMachineReconciler) reconcileDeletePublicIp(ctx context.Context, clusterScope *scope.ClusterScope, machineScope *scope.MachineScope) Result {
	log := ctrl.LoggerFrom(ctx)
	if machineScope.GetVm().PublicIpPool != "" {
		log.V(4).Info("Not deleting publicip from pool")
		return Result{Status: ResultStatusOK}
	}
	for k, id := range r.Tracker.getPublicIps(machineScope) {
		err := r.Tracker.IPAllocator(machineScope).DeallocateIP(ctx, k, clusterScope)
		if err != nil {
			return Result{Status: ResultStatusNotOK, Error: fmt.Errorf("cannot delete publicIp %s: %w", id, err)}
		}
		r.Tracker.untrackIP(machineScope, k)
	}
	return Result{Status: ResultStatusOK}
}
