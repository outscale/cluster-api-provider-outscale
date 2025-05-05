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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// reconcileNatService reconcile the NatService of the cluster.
func (r *OscClusterReconciler) reconcileNatService(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	if !clusterScope.NeedReconciliation(infrastructurev1beta1.ReconcilerNatService) {
		log.V(4).Info("No need for natService reconciliation")
		return reconcile.Result{}, nil
	}
	if clusterScope.GetNetwork().UseExisting.Net {
		log.V(3).Info("Reusing existing natServices")
		return reconcile.Result{}, nil
	}
	log.V(4).Info("Reconciling natServices")

	natServiceSpecs := clusterScope.GetNatServices()
	for _, natServiceSpec := range natServiceSpecs {
		natService, err := r.Tracker.getNatService(ctx, natServiceSpec, clusterScope)
		switch {
		case errors.Is(err, ErrNoResourceFound):
		case err != nil:
			return reconcile.Result{}, fmt.Errorf("find existing: %w", err)
		default:
			log.V(4).Info("Found existing natService", "natServiceId", natService.GetNatServiceId())
			continue
		}

		publicIpId, _, err := r.Tracker.IPAllocator(clusterScope).AllocateIP(ctx,
			clusterScope.GetNatServiceClientToken(natServiceSpec), clusterScope.GetNatServiceName(natServiceSpec), "", clusterScope)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("allocate IP: %w", err)
		}

		subnetSpec, err := clusterScope.GetSubnet(natServiceSpec.SubnetName, infrastructurev1beta1.RoleNat, natServiceSpec.SubregionName)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("find subnet: %w", err)
		}
		subnetId, err := r.Tracker.getSubnetId(ctx, subnetSpec, clusterScope)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("get subnet: %w", err)
		}

		log.V(3).Info("Creating natService")
		natService, err = r.Cloud.NatService(ctx, *clusterScope).CreateNatService(ctx, publicIpId, subnetId,
			clusterScope.GetNatServiceClientToken(natServiceSpec), clusterScope.GetNatServiceName(natServiceSpec), clusterScope.GetUID())
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot create natService: %w", err)
		}
		log.V(2).Info("Created natService", "natServiceId", natService.GetNatServiceId())
		r.Tracker.setNatServiceId(clusterScope, natServiceSpec, natService.GetNatServiceId())
		r.Recorder.Eventf(clusterScope.OscCluster, corev1.EventTypeNormal, infrastructurev1beta1.NatServicesCreatedReason, "NAT created %s", natServiceSpec.SubregionName)
	}
	clusterScope.SetReconciliationGeneration(infrastructurev1beta1.ReconcilerNatService)
	return reconcile.Result{}, nil
}

func (r *OscClusterReconciler) listNATPublicIPs(ctx context.Context, clusterScope *scope.ClusterScope, cidr bool) ([]string, error) {
	netId, err := r.Tracker.getNetId(ctx, clusterScope)
	if err != nil {
		return nil, fmt.Errorf("cannot find net: %w", err)
	}
	natSvc := r.Cloud.NatService(ctx, *clusterScope)
	nats, err := natSvc.ListNatServices(ctx, netId)
	if err != nil {
		return nil, fmt.Errorf("list natServices: %w", err)
	}
	ips := make([]string, 0, len(nats))
	for _, nat := range nats {
		for _, ip := range nat.GetPublicIps() {
			pip := ip.GetPublicIp()
			if cidr {
				pip += "/32"
			}
			ips = append(ips, pip)
		}
	}
	return ips, nil
}

// reconcileDeleteNatService reconcile the destruction of the NatService of the cluster.
func (r *OscClusterReconciler) reconcileDeleteNatService(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	if clusterScope.GetNetwork().UseExisting.Net {
		log.V(3).Info("Not deleting existing nat services")
		return reconcile.Result{}, nil
	}
	netId, err := r.Tracker.getNetId(ctx, clusterScope)
	switch {
	case errors.Is(err, ErrNoResourceFound) || errors.Is(err, ErrMissingResource):
		log.V(4).Info("The net is already deleted, no nat service expected")
		return reconcile.Result{}, nil
	case err != nil:
		return reconcile.Result{}, fmt.Errorf("find existing: %w", err)
	}
	natSvc := r.Cloud.NatService(ctx, *clusterScope)
	nats, err := natSvc.ListNatServices(ctx, netId)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("list natServices: %w", err)
	}

	for _, nat := range nats {
		if nat.GetState() == "deleted" {
			continue
		}
		log.V(2).Info("Deleting natService", "natId", nat.GetNatServiceId())
		err = natSvc.DeleteNatService(ctx, nat.GetNatServiceId())
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot delete natService: %w", err)
		}
	}
	return reconcile.Result{}, nil
}
