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
func (r *OscMachineReconciler) reconcileDeletePublicIp(ctx context.Context, clusterScope *scope.ClusterScope, machineScope *scope.MachineScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	if machineScope.GetVm().PublicIpPool != "" {
		log.V(4).Info("Not deleting publicip from pool")
		return reconcile.Result{}, nil
	}
	svc := r.Cloud.PublicIp(ctx, *clusterScope)
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
