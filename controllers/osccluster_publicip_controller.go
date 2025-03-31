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

// reconcilePublicIp reconcile the PublicIp of the cluster.
func (r *OscClusterReconciler) reconcilePublicIp(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	return reconcile.Result{}, nil
}

// reconcileDeletePublicIp reconcile the destruction of the PublicIp of the cluster.
func (r *OscClusterReconciler) reconcileDeletePublicIp(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	svc := r.Cloud.PublicIp(ctx, *clusterScope)
	for k, v := range r.Tracker.getPublicIps(clusterScope) {
		log.V(2).Info("Deleting publicip", "publicIpId", v)
		err := svc.DeletePublicIp(ctx, v)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot delete publicIp %s: %w", v, err)
		}
		r.Tracker.untrackIP(clusterScope, k)
	}
	return reconcile.Result{}, nil
}
