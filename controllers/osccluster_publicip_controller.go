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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// reconcileDeletePublicIp reconcile the destruction of the PublicIp of the cluster.
func (r *OscClusterReconciler) reconcileDeletePublicIp(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	svc := r.Cloud.PublicIp(ctx, *clusterScope)
	for k, id := range r.Tracker.getPublicIps(clusterScope) {
		ip, err := svc.GetPublicIp(ctx, id)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot get publicIp %s: %w", id, err)
		}
		if ip == nil {
			r.Tracker.untrackIP(clusterScope, k)
			continue
		}
		log.V(2).Info("Deleting publicip", "publicIpId", id)
		err = svc.DeletePublicIp(ctx, id)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot delete publicIp %s: %w", id, err)
		}
		r.Tracker.untrackIP(clusterScope, k)
	}
	return reconcile.Result{}, nil
}
