/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package controllers

import (
	"context"
	"errors"
	"fmt"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	tag "github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	osc "github.com/outscale/osc-sdk-go/v2"
	corev1 "k8s.io/api/core/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func getLoadBalancerNameTag(lb *osc.LoadBalancer) string {
	for _, tg := range lb.GetTags() {
		if tg.Key == tag.NameKey {
			return tg.Value
		}
	}
	return ""
}

// reconcileLoadBalancer reconciles the loadBalancer of the cluster.
func (r *OscClusterReconciler) reconcileLoadBalancer(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	if !clusterScope.NeedReconciliation(infrastructurev1beta1.ReconcilerLoadbalancer) {
		log.V(4).Info("No need for loadbalancer reconciliation")
		return reconcile.Result{}, nil
	}
	log.V(4).Info("Reconciling loadBalancer")

	loadBalancerSpec := clusterScope.GetLoadBalancer()
	loadBalancerName := loadBalancerSpec.LoadBalancerName
	svc := r.Cloud.LoadBalancer(clusterScope.Tenant)
	loadbalancer, err := svc.GetLoadBalancer(ctx, loadBalancerName)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot get loadbalancer: %w", err)
	}

	nameTag := loadBalancerName + "-" + clusterScope.GetUID()
	if loadbalancer != nil {
		lbName := getLoadBalancerNameTag(loadbalancer)
		if lbName == "" && *loadbalancer.LoadBalancerName == loadBalancerName {
			return reconcile.Result{}, fmt.Errorf("a LoadBalancer with name %s already exists", loadBalancerName)
		}
		if lbName != "" && lbName != nameTag {
			return reconcile.Result{}, fmt.Errorf("a LoadBalancer %s already exists for another cluster", loadBalancerName)
		}
	}

	if loadbalancer == nil {
		subnetSpec, err := clusterScope.GetSubnet(loadBalancerSpec.SubnetName, infrastructurev1beta1.RoleLoadBalancer, "")
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("find subnet: %w", err)
		}
		subnetId, err := r.Tracker.getSubnetId(ctx, subnetSpec, clusterScope)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("get subnet: %w", err)
		}
		var sgSpecs []infrastructurev1beta1.OscSecurityGroup
		if loadBalancerSpec.SecurityGroupName != "" {
			sgSpecs, err = clusterScope.GetSecurityGroupsFor([]infrastructurev1beta1.OscSecurityGroupElement{{Name: loadBalancerSpec.SecurityGroupName}}, infrastructurev1beta1.RoleLoadBalancer)
		} else {
			sgSpecs, err = clusterScope.GetSecurityGroupsFor(nil, infrastructurev1beta1.RoleLoadBalancer)
		}
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("find securityGroup: %w", err)
		}
		if len(sgSpecs) == 0 {
			return reconcile.Result{}, errors.New("no security group found")
		}
		securityGroupId, err := r.Tracker.getSecurityGroupId(ctx, sgSpecs[0], clusterScope)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot get security group: %w", err)
		}
		log.V(2).Info("Creating loadBalancer", "loadBalancerName", loadBalancerName, "subnet", subnetId, "securityGroupId", securityGroupId)
		loadbalancer, err = svc.CreateLoadBalancer(ctx, &loadBalancerSpec, subnetId, securityGroupId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot create loadBalancer: %w", err)
		}
		r.Recorder.Event(clusterScope.OscCluster, corev1.EventTypeNormal, infrastructurev1beta1.LoadBalancerCreatedReason, "Loadbalancer created")
		log.V(2).Info("Configuring loadBalancer healthcheck", "loadBalancerName", loadBalancerName)
		_, err = svc.ConfigureHealthCheck(ctx, &loadBalancerSpec)
		log.V(4).Info("Get loadbalancer", "loadbalancer", loadbalancer)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot configure healthcheck: %w", err)
		}
	}
	if len(loadbalancer.GetTags()) == 0 {
		log.V(2).Info("Creating loadBalancer name tag", "loadBalancerName", loadBalancerName)
		err = svc.CreateLoadBalancerTag(ctx, &loadBalancerSpec, osc.NewResourceTag(tag.NameKey, nameTag))
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot tag loadBalancer: %w", err)
		}
	}
	controlPlaneEndpoint := loadbalancer.GetDnsName()
	log.V(4).Info("Set controlPlaneEndpoint", "endpoint", controlPlaneEndpoint)

	controlPlanePort := loadBalancerSpec.Listener.LoadBalancerPort

	clusterScope.SetControlPlaneEndpoint(clusterv1.APIEndpoint{
		Host: controlPlaneEndpoint,
		Port: controlPlanePort,
	})
	clusterScope.SetReconciliationGeneration(infrastructurev1beta1.ReconcilerLoadbalancer)
	return reconcile.Result{}, nil
}

// reconcileDeleteLoadBalancer reconcile the destruction of the LoadBalancer of the cluster.

func (r *OscClusterReconciler) reconcileDeleteLoadBalancer(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	loadBalancerSpec := clusterScope.GetLoadBalancer()
	loadBalancerName := loadBalancerSpec.LoadBalancerName

	svc := r.Cloud.LoadBalancer(clusterScope.Tenant)
	loadbalancer, err := svc.GetLoadBalancer(ctx, loadBalancerName)
	if err != nil {
		return reconcile.Result{}, err
	}
	if loadbalancer == nil {
		log.V(4).Info("The loadBalancer is already deleted", "loadBalancerName", loadBalancerName)
		return reconcile.Result{}, nil
	}
	name := loadBalancerName + "-" + clusterScope.GetUID()
	if name != getLoadBalancerNameTag(loadbalancer) {
		log.V(3).Info("Loadbalancer belongs to another cluster, not deleting", "loadBalancer", loadBalancerName)
		return reconcile.Result{}, nil
	}

	// err = svc.UnlinkLoadBalancerBackendMachines(ctx, loadbalancer.GetBackendIps(), loadBalancerName)
	// if err != nil {
	// 	return reconcile.Result{}, fmt.Errorf("cannot unlink loadBalancer backends: %w", err)
	// }

	log.V(2).Info("Deleting loadBalancer", "loadBalancerName", loadBalancerName)
	err = svc.DeleteLoadBalancer(ctx, &loadBalancerSpec)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot delete loadBalancer: %w", err)
	}
	return reconcile.Result{}, nil
}
