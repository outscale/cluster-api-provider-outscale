/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package controllers

import (
	"context"
	"fmt"

	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// reconcileDeletePublicIp reconcile the destruction of the PublicIp of the cluster.
func (r *OscClusterReconciler) reconcileDeletePublicIp(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	for k, id := range r.Tracker.getPublicIps(clusterScope) {
		err := r.Tracker.IPAllocator(clusterScope).DeallocateIP(ctx, k, clusterScope)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot delete publicIp %s: %w", id, err)
		}
		r.Tracker.untrackIP(clusterScope, k)
	}
	return reconcile.Result{}, nil
}
