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

	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	osc "github.com/outscale/osc-sdk-go/v2"

	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/security"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/service"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// checkLoadBalancerSubneOscAssociateResourceName check that LoadBalancer Subnet dependancies tag name in both resource configuration are the same.
func checkLoadBalancerSubnetOscAssociateResourceName(clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	loadBalancerSpec := clusterScope.GetLoadBalancer()
	loadBalancerSubnetName := loadBalancerSpec.SubnetName + "-" + clusterScope.GetUID()
	subnetsSpec := clusterScope.GetSubnet()
	for _, subnetSpec := range subnetsSpec {
		subnetName := subnetSpec.Name + "-" + clusterScope.GetUID()
		resourceNameList = append(resourceNameList, subnetName)
	}
	clusterScope.V(2).Info("Check match subnet with loadBalancer")
	checkOscAssociate := Contains(resourceNameList, loadBalancerSubnetName)
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
	clusterScope.V(2).Info("Check LoadBalancer name parameters")
	_, err := infrastructurev1beta1.ValidateLoadBalancerName(loadBalancerName)
	if err != nil {
		return loadBalancerName, fmt.Errorf("%s is an invalid loadBalancer name: %w", loadBalancerName, err)
	}

	loadBalancerType := loadBalancerSpec.LoadBalancerType
	_, err = infrastructurev1beta1.ValidateLoadBalancerType(loadBalancerType)
	if err != nil {
		return loadBalancerName, fmt.Errorf("%s is an invalid loadBalancer type: %w", loadBalancerType, err)
	}

	loadBalancerBackendPort := loadBalancerSpec.Listener.BackendPort
	_, err = infrastructurev1beta1.ValidatePort(loadBalancerBackendPort)
	if err != nil {
		return loadBalancerName, fmt.Errorf("%d is an %w for loadBalancer backend", loadBalancerBackendPort, err)
	}

	loadBalancerBackendProtocol := loadBalancerSpec.Listener.BackendProtocol
	_, err = infrastructurev1beta1.ValidateProtocol(loadBalancerBackendProtocol)
	if err != nil {
		return loadBalancerName, fmt.Errorf("%s is an %w for loadBalancer backend", loadBalancerBackendProtocol, err)
	}

	loadBalancerPort := loadBalancerSpec.Listener.LoadBalancerPort
	_, err = infrastructurev1beta1.ValidatePort(loadBalancerPort)
	if err != nil {
		return loadBalancerName, fmt.Errorf("%d is an %w for loadBalancer", loadBalancerPort, err)
	}

	loadBalancerProtocol := loadBalancerSpec.Listener.LoadBalancerProtocol
	_, err = infrastructurev1beta1.ValidateProtocol(loadBalancerProtocol)
	if err != nil {
		return loadBalancerName, fmt.Errorf("%s is an %w for loadBalancer", loadBalancerProtocol, err)
	}

	loadBalancerCheckInterval := loadBalancerSpec.HealthCheck.CheckInterval
	_, err = infrastructurev1beta1.ValidateInterval(loadBalancerCheckInterval)
	if err != nil {
		return loadBalancerName, fmt.Errorf("%d is an %w for loadBalancer", loadBalancerCheckInterval, err)
	}

	loadBalancerHealthyThreshold := loadBalancerSpec.HealthCheck.HealthyThreshold
	_, err = infrastructurev1beta1.ValidateThreshold(loadBalancerHealthyThreshold)
	if err != nil {
		return loadBalancerName, fmt.Errorf("%d is an %w for loadBalancer", loadBalancerHealthyThreshold, err)
	}

	loadBalancerHealthCheckPort := loadBalancerSpec.HealthCheck.Port
	_, err = infrastructurev1beta1.ValidatePort(loadBalancerHealthCheckPort)
	if err != nil {
		return loadBalancerName, fmt.Errorf("%d is an %w for loadBalancer", loadBalancerHealthCheckPort, err)
	}

	loadBalancerHealthCheckProtocol := loadBalancerSpec.HealthCheck.Protocol
	_, err = infrastructurev1beta1.ValidateProtocol(loadBalancerHealthCheckProtocol)
	if err != nil {
		return loadBalancerName, fmt.Errorf("%s is an %w for loadBalancer", loadBalancerHealthCheckProtocol, err)
	}

	loadBalancerTimeout := loadBalancerSpec.HealthCheck.Timeout
	_, err = infrastructurev1beta1.ValidateTimeout(loadBalancerTimeout)
	if err != nil {
		return loadBalancerName, fmt.Errorf("%d is an %w for loadBalancer", loadBalancerTimeout, err)
	}

	loadBalancerUnhealthyThreshold := loadBalancerSpec.HealthCheck.UnhealthyThreshold
	_, err = infrastructurev1beta1.ValidateThreshold(loadBalancerUnhealthyThreshold)
	if err != nil {
		return loadBalancerName, fmt.Errorf("%d is an %w for loadBalancer", loadBalancerUnhealthyThreshold, err)
	}

	return "", nil
}

// checkLoadBalancerSecurityOscAssociateResourceName check that LoadBalancer SecurityGroup dependancies tag name in both resource configuration are the same.
func checkLoadBalancerSecurityGroupOscAssociateResourceName(clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	loadBalancerSpec := clusterScope.GetLoadBalancer()
	loadBalancerSecurityGroupName := loadBalancerSpec.SecurityGroupName + "-" + clusterScope.GetUID()
	securityGroupsSpec := clusterScope.GetSecurityGroups()
	for _, securityGroupSpec := range securityGroupsSpec {
		securityGroupName := securityGroupSpec.Name + "-" + clusterScope.GetUID()
		resourceNameList = append(resourceNameList, securityGroupName)
	}
	clusterScope.V(2).Info("Check match securityGroup with loadBalancer")
	checkOscAssociate := Contains(resourceNameList, loadBalancerSecurityGroupName)
	if checkOscAssociate {
		return nil
	} else {
		return fmt.Errorf("%s securityGroup does not exist in loadBalancer", loadBalancerSecurityGroupName)
	}
}

// reconcileLoadBalancer reconciles the loadBalancer of the cluster.
func reconcileLoadBalancer(ctx context.Context, clusterScope *scope.ClusterScope, loadBalancerSvc service.OscLoadBalancerInterface, securityGroupSvc security.OscSecurityGroupInterface) (reconcile.Result, error) {

	loadBalancerSpec := clusterScope.GetLoadBalancer()
	loadBalancerName := loadBalancerSpec.LoadBalancerName
	clusterScope.V(2).Info("Check if the desired loadbalancer exist", "loadBalancerName", loadBalancerName)
	loadbalancer, err := loadBalancerSvc.GetLoadBalancer(loadBalancerSpec)
	if err != nil {
		return reconcile.Result{}, err
	}
	subnetName := loadBalancerSpec.SubnetName + "-" + clusterScope.GetUID()
	subnetId, err := getSubnetResourceId(subnetName, clusterScope)
	clusterScope.V(4).Info("Get loadBalancer subnetId", "subnet", subnetId)
	if err != nil {
		return reconcile.Result{}, err
	}
	securityGroupName := loadBalancerSpec.SecurityGroupName + "-" + clusterScope.GetUID()
	securityGroupId, err := getSecurityGroupResourceId(securityGroupName, clusterScope)
	clusterScope.V(4).Info("Get loadBalancer subnetId", "sg", securityGroupId)
	if err != nil {
		return reconcile.Result{}, err
	}
	name := loadBalancerName + "-" + clusterScope.GetUID()
	nameTag := osc.ResourceTag{
		Key:   "Name",
		Value: name,
	}
	if loadbalancer != nil {
		loadBalancerTag, err := loadBalancerSvc.GetLoadBalancerTag(loadBalancerSpec)
		if err != nil {
			return reconcile.Result{}, err
		}
		if loadBalancerTag == nil && *loadbalancer.LoadBalancerName == loadBalancerName {
			clusterScope.V(4).Info("LoadBalancer already exists", "loadBalancer", loadBalancerName)
			return reconcile.Result{}, fmt.Errorf("A LoadBalancer %s already exists", loadBalancerName)
		}
		if loadBalancerTag != nil && *loadBalancerTag.Key == nameTag.Key && *loadBalancerTag.Value != nameTag.Value {
			clusterScope.V(4).Info("LoadBalancer already exists by other cluster", "loadBalancer", loadBalancerName)

			return reconcile.Result{}, fmt.Errorf("A LoadBalancer %s already exists that is used by another cluster other than %s", loadBalancerName, clusterScope.GetUID())
		}
	}

	if loadbalancer == nil {
		clusterScope.V(2).Info("Create the desired loadBalancer", "loadBalancerName", loadBalancerName)
		_, err := loadBalancerSvc.CreateLoadBalancer(loadBalancerSpec, subnetId, securityGroupId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w cannot create loadBalancer for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
		}
		clusterScope.V(2).Info("Delete default outbound rule for loadBalancer", "loadBalancerName", loadBalancerName)
		err = securityGroupSvc.DeleteSecurityGroupRule(securityGroupId, "Outbound", "-1", "0.0.0.0/0", "", 0, 0)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w cannot empty Outbound sg rules for loadBalancer for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
		}
		clusterScope.V(2).Info("Configure the desired loadBalancer", "loadBalancerName", loadBalancerName)
		loadbalancer, err = loadBalancerSvc.ConfigureHealthCheck(loadBalancerSpec)
		clusterScope.V(4).Info("Get loadbalancer", "loadbalancer", loadbalancer)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w cannot configure healthcheck for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
		}
		err = loadBalancerSvc.CreateLoadBalancerTag(loadBalancerSpec, nameTag)
		clusterScope.V(2).Info("Create the desired loadBalancer tag name", "loadBalancerName", loadBalancerName)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w cannot tag loadBalancer for OscCluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
		}
	}
	controlPlaneEndpoint := *loadbalancer.DnsName
	clusterScope.V(4).Info("Set controlPlaneEndpoint", "endpoint", controlPlaneEndpoint)

	controlPlanePort := loadBalancerSpec.Listener.LoadBalancerPort

	clusterScope.SetControlPlaneEndpoint(clusterv1.APIEndpoint{
		Host: controlPlaneEndpoint,
		Port: controlPlanePort,
	})
	return reconcile.Result{}, nil

}

// reconcileDeleteLoadBalancer reconcile the destruction of the LoadBalancer of the cluster.

func reconcileDeleteLoadBalancer(ctx context.Context, clusterScope *scope.ClusterScope, loadBalancerSvc service.OscLoadBalancerInterface) (reconcile.Result, error) {
	osccluster := clusterScope.OscCluster
	loadBalancerSpec := clusterScope.GetLoadBalancer()
	loadBalancerSpec.SetDefaultValue()
	loadBalancerName := loadBalancerSpec.LoadBalancerName

	loadbalancer, err := loadBalancerSvc.GetLoadBalancer(loadBalancerSpec)
	if err != nil {
		return reconcile.Result{}, err
	}
	if loadbalancer == nil {
		clusterScope.V(4).Info("The desired loadBalancer does not exist anymore", "loadBalancerName", loadBalancerName)
		controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
		return reconcile.Result{}, nil
	}
	name := loadBalancerName + "-" + clusterScope.GetUID()
	nameTag := osc.ResourceTag{
		Key:   "Name",
		Value: name,
	}
	clusterScope.V(4).Info("Delete the desired loadBalancer", "loadBalancerName", loadBalancerName)
	loadBalancerTag, err := loadBalancerSvc.GetLoadBalancerTag(loadBalancerSpec)
	if err != nil {
		return reconcile.Result{}, err
	}
	if loadBalancerTag != nil && *loadBalancerTag.Key == nameTag.Key && *loadBalancerTag.Value != nameTag.Value {
		clusterScope.V(4).Info("cannot delete LoadBalancer that already exists by other cluster", "loadBalancer", loadBalancerName)
		return reconcile.Result{}, nil
	}

	err = loadBalancerSvc.CheckLoadBalancerDeregisterVm(20, 120, loadBalancerSpec)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("%w VmBackend is not deregister in loadBalancer %s for OscCluster %s/%s", err, loadBalancerSpec.LoadBalancerName, clusterScope.GetNamespace(), clusterScope.GetName())
	}

	loadBalancerTagKey := osc.ResourceLoadBalancerTag{
		Key: nameTag.Key,
	}
	err = loadBalancerSvc.DeleteLoadBalancerTag(loadBalancerSpec, loadBalancerTagKey)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("%w cannot delete loadBalancer Tag for OscCluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
	}

	err = loadBalancerSvc.DeleteLoadBalancer(loadBalancerSpec)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("%w cannot delete loadBalancer for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
	}
	return reconcile.Result{}, nil
}
