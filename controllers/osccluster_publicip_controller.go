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
