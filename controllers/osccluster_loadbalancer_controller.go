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
	"slices"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/security"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/service"
	osc "github.com/outscale/osc-sdk-go/v2"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// checkLoadBalancerSubneOscAssociateResourceName check that LoadBalancer Subnet dependencies tag name in both resource configuration are the same.
func checkLoadBalancerSubnetOscAssociateResourceName(clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	loadBalancerSpec := clusterScope.GetLoadBalancer()
	loadBalancerSubnetName := loadBalancerSpec.SubnetName + "-" + clusterScope.GetUID()
	subnetsSpec := clusterScope.GetSubnets()
	for _, subnetSpec := range subnetsSpec {
		subnetName := subnetSpec.Name + "-" + clusterScope.GetUID()
		resourceNameList = append(resourceNameList, subnetName)
	}
	checkOscAssociate := slices.Contains(resourceNameList, loadBalancerSubnetName)
	if checkOscAssociate {
		return nil
	} else {
		return fmt.Errorf("%s subnet does not exist in loadBalancer", loadBalancerSubnetName)
	}
}

// checkLoadBalancerFormatParameters check LoadBalancer parameters format (Tag format, cidr format, ..)
func checkLoadBalancerFormatParameters(clusterScope *scope.ClusterScope) (string, error) {
	loadBalancerSpec := clusterScope.GetLoadBalancer()
	loadBalancerSpec.SetDefaultValue()
	loadBalancerName := loadBalancerSpec.LoadBalancerName
	err := infrastructurev1beta1.ValidateLoadBalancerName(loadBalancerName)
	if err != nil {
		return loadBalancerName, fmt.Errorf("%s is an invalid loadBalancer name: %w", loadBalancerName, err)
	}

	loadBalancerType := loadBalancerSpec.LoadBalancerType
	err = infrastructurev1beta1.ValidateLoadBalancerType(loadBalancerType)
	if err != nil {
		return loadBalancerName, fmt.Errorf("%s is an invalid loadBalancer type: %w", loadBalancerType, err)
	}

	loadBalancerBackendPort := loadBalancerSpec.Listener.BackendPort
	err = infrastructurev1beta1.ValidatePort(loadBalancerBackendPort)
	if err != nil {
		return loadBalancerName, fmt.Errorf("%d is an %w for loadBalancer backend", loadBalancerBackendPort, err)
	}

	loadBalancerBackendProtocol := loadBalancerSpec.Listener.BackendProtocol
	err = infrastructurev1beta1.ValidateProtocol(loadBalancerBackendProtocol)
	if err != nil {
		return loadBalancerName, fmt.Errorf("%s is an %w for loadBalancer backend", loadBalancerBackendProtocol, err)
	}

	loadBalancerPort := loadBalancerSpec.Listener.LoadBalancerPort
	err = infrastructurev1beta1.ValidatePort(loadBalancerPort)
	if err != nil {
		return loadBalancerName, fmt.Errorf("%d is an %w for loadBalancer", loadBalancerPort, err)
	}

	loadBalancerProtocol := loadBalancerSpec.Listener.LoadBalancerProtocol
	err = infrastructurev1beta1.ValidateProtocol(loadBalancerProtocol)
	if err != nil {
		return loadBalancerName, fmt.Errorf("%s is an %w for loadBalancer", loadBalancerProtocol, err)
	}

	loadBalancerCheckInterval := loadBalancerSpec.HealthCheck.CheckInterval
	err = infrastructurev1beta1.ValidateInterval(loadBalancerCheckInterval)
	if err != nil {
		return loadBalancerName, fmt.Errorf("%d is an %w for loadBalancer", loadBalancerCheckInterval, err)
	}

	loadBalancerHealthyThreshold := loadBalancerSpec.HealthCheck.HealthyThreshold
	err = infrastructurev1beta1.ValidateThreshold(loadBalancerHealthyThreshold)
	if err != nil {
		return loadBalancerName, fmt.Errorf("%d is an %w for loadBalancer", loadBalancerHealthyThreshold, err)
	}

	loadBalancerHealthCheckPort := loadBalancerSpec.HealthCheck.Port
	err = infrastructurev1beta1.ValidatePort(loadBalancerHealthCheckPort)
	if err != nil {
		return loadBalancerName, fmt.Errorf("%d is an %w for loadBalancer", loadBalancerHealthCheckPort, err)
	}

	loadBalancerHealthCheckProtocol := loadBalancerSpec.HealthCheck.Protocol
	err = infrastructurev1beta1.ValidateProtocol(loadBalancerHealthCheckProtocol)
	if err != nil {
		return loadBalancerName, fmt.Errorf("%s is an %w for loadBalancer", loadBalancerHealthCheckProtocol, err)
	}

	loadBalancerTimeout := loadBalancerSpec.HealthCheck.Timeout
	err = infrastructurev1beta1.ValidateTimeout(loadBalancerTimeout)
	if err != nil {
		return loadBalancerName, fmt.Errorf("%d is an %w for loadBalancer", loadBalancerTimeout, err)
	}

	loadBalancerUnhealthyThreshold := loadBalancerSpec.HealthCheck.UnhealthyThreshold
	err = infrastructurev1beta1.ValidateThreshold(loadBalancerUnhealthyThreshold)
	if err != nil {
		return loadBalancerName, fmt.Errorf("%d is an %w for loadBalancer", loadBalancerUnhealthyThreshold, err)
	}

	return "", nil
}

// checkLoadBalancerSecurityOscAssociateResourceName check that LoadBalancer SecurityGroup dependencies tag name in both resource configuration are the same.
func checkLoadBalancerSecurityGroupOscAssociateResourceName(clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	loadBalancerSpec := clusterScope.GetLoadBalancer()
	loadBalancerSecurityGroupName := loadBalancerSpec.SecurityGroupName + "-" + clusterScope.GetUID()
	securityGroupsSpec := clusterScope.GetSecurityGroups()
	for _, securityGroupSpec := range securityGroupsSpec {
		securityGroupName := securityGroupSpec.Name + "-" + clusterScope.GetUID()
		resourceNameList = append(resourceNameList, securityGroupName)
	}
	checkOscAssociate := slices.Contains(resourceNameList, loadBalancerSecurityGroupName)
	if checkOscAssociate {
		return nil
	} else {
		return fmt.Errorf("%s securityGroup does not exist in loadBalancer", loadBalancerSecurityGroupName)
	}
}

// reconcileLoadBalancer reconciles the loadBalancer of the cluster.
func (r *OscClusterReconciler) reconcileLoadBalancer(ctx context.Context, clusterScope *scope.ClusterScope, loadBalancerSvc service.OscLoadBalancerInterface, securityGroupSvc security.OscSecurityGroupInterface) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	loadBalancerSpec := clusterScope.GetLoadBalancer()
	loadBalancerName := loadBalancerSpec.LoadBalancerName
	loadbalancer, err := loadBalancerSvc.GetLoadBalancer(ctx, loadBalancerName)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot get loadbalancer: %w", err)
	}
	// TODO: add a role to loadbalancer definition
	subnetSpec, err := clusterScope.GetSubnet(loadBalancerSpec.SubnetName, infrastructurev1beta1.RolePublic, "")
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("reconcile loadbalancer: %w")
	}
	subnetId, err := r.Tracker.getSubnetId(ctx, subnetSpec, clusterScope)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("reconcile loadbalancer: %w")
	}
	securityGroupName := loadBalancerSpec.SecurityGroupName + "-" + clusterScope.GetUID()
	securityGroupId, err := getSecurityGroupResourceId(securityGroupName, clusterScope)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot get security group: %w", err)
	}
	log.V(4).Info("LoadBalancer info", "subnet", subnetId, "securityGroupId", securityGroupId)
	name := loadBalancerName + "-" + clusterScope.GetUID()
	nameTag := osc.ResourceTag{
		Key:   "Name",
		Value: name,
	}
	if loadbalancer != nil {
		loadBalancerTag, err := loadBalancerSvc.GetLoadBalancerTag(ctx, loadBalancerSpec)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("get loadbalancer tag: %w", err)
		}
		if loadBalancerTag == nil && *loadbalancer.LoadBalancerName == loadBalancerName {
			return reconcile.Result{}, fmt.Errorf("A LoadBalancer with name %s already exists", loadBalancerName)
		}
		if loadBalancerTag != nil && *loadBalancerTag.Key == nameTag.Key && *loadBalancerTag.Value != nameTag.Value {
			return reconcile.Result{}, fmt.Errorf("A LoadBalancer %s already exists that is used by another cluster other than %s", loadBalancerName, clusterScope.GetUID())
		}
	}

	if loadbalancer == nil {
		log.V(2).Info("Creating loadBalancer", "loadBalancerName", loadBalancerName)
		_, err := loadBalancerSvc.CreateLoadBalancer(ctx, loadBalancerSpec, subnetId, securityGroupId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot create loadBalancer: %w", err)
		}
		log.V(2).Info("Deleting default outbound sg rule for loadBalancer", "loadBalancerName", loadBalancerName)
		err = securityGroupSvc.DeleteSecurityGroupRule(ctx, securityGroupId, "Outbound", "-1", "0.0.0.0/0", "", 0, 0)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot delete default sg rules for loadBalancer: %w", err)
		}
		log.V(2).Info("Configuring loadBalancer healthcheck", "loadBalancerName", loadBalancerName)
		loadbalancer, err = loadBalancerSvc.ConfigureHealthCheck(ctx, loadBalancerSpec)
		log.V(4).Info("Get loadbalancer", "loadbalancer", loadbalancer)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot configure healthcheck: %w", err)
		}
		log.V(2).Info("Creating loadBalancer tag name", "loadBalancerName", loadBalancerName)
		err = loadBalancerSvc.CreateLoadBalancerTag(ctx, loadBalancerSpec, nameTag)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot tag loadBalancer: %w", err)
		}
	}
	controlPlaneEndpoint := *loadbalancer.DnsName
	log.V(4).Info("Set controlPlaneEndpoint", "endpoint", controlPlaneEndpoint)

	controlPlanePort := loadBalancerSpec.Listener.LoadBalancerPort

	clusterScope.SetControlPlaneEndpoint(clusterv1.APIEndpoint{
		Host: controlPlaneEndpoint,
		Port: controlPlanePort,
	})
	return reconcile.Result{}, nil
}

// reconcileDeleteLoadBalancer reconcile the destruction of the LoadBalancer of the cluster.

func reconcileDeleteLoadBalancer(ctx context.Context, clusterScope *scope.ClusterScope, loadBalancerSvc service.OscLoadBalancerInterface) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	loadBalancerSpec := clusterScope.GetLoadBalancer()
	loadBalancerSpec.SetDefaultValue()
	loadBalancerName := loadBalancerSpec.LoadBalancerName

	loadbalancer, err := loadBalancerSvc.GetLoadBalancer(ctx, loadBalancerName)
	if err != nil {
		return reconcile.Result{}, err
	}
	if loadbalancer == nil {
		log.V(4).Info("The loadBalancer is already deleted", "loadBalancerName", loadBalancerName)
		return reconcile.Result{}, nil
	}
	name := loadBalancerName + "-" + clusterScope.GetUID()
	nameTag := osc.ResourceTag{
		Key:   "Name",
		Value: name,
	}
	loadBalancerTag, err := loadBalancerSvc.GetLoadBalancerTag(ctx, loadBalancerSpec)
	if err != nil {
		return reconcile.Result{}, err
	}
	if loadBalancerTag != nil && *loadBalancerTag.Key == nameTag.Key && *loadBalancerTag.Value != nameTag.Value {
		log.V(3).Info("Loadbalancer belongs to another cluster, not deleting", "loadBalancer", loadBalancerName)
		return reconcile.Result{}, nil
	}

	err = loadBalancerSvc.CheckLoadBalancerDeregisterVm(ctx, 20, 120, loadBalancerSpec)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("VmBackend is stll registered in loadBalancer: %w", err)
	}

	loadBalancerTagKey := osc.ResourceLoadBalancerTag{
		Key: nameTag.Key,
	}
	log.V(2).Info("Deleting loadBalancer tag", "loadBalancerName", loadBalancerName)
	err = loadBalancerSvc.DeleteLoadBalancerTag(ctx, loadBalancerSpec, loadBalancerTagKey)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot delete loadBalancer tag: %w", err)
	}

	log.V(2).Info("Deleting loadBalancer", "loadBalancerName", loadBalancerName)
	err = loadBalancerSvc.DeleteLoadBalancer(ctx, loadBalancerSpec)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot delete loadBalancer: %w", err)
	}
	return reconcile.Result{}, nil
}
