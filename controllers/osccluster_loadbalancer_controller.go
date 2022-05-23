package controllers

import (
	"context"
	"fmt"

	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/service"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// checkLoadBalancerSubneOscAssociateResourceName check that LoadBalancer Subnet dependancies tag name in both resource configuration are the same.
func checkLoadBalancerSubnetOscAssociateResourceName(clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	clusterScope.Info("check match subnet with loadBalancer")
	loadBalancerSpec := clusterScope.GetLoadBalancer()
	loadBalancerSubnetName := loadBalancerSpec.SubnetName + "-" + clusterScope.GetUID()
	subnetsSpec := clusterScope.GetSubnet()
	for _, subnetSpec := range subnetsSpec {
		subnetName := subnetSpec.Name + "-" + clusterScope.GetUID()
		resourceNameList = append(resourceNameList, subnetName)
	}
	checkOscAssociate := contains(resourceNameList, loadBalancerSubnetName)
	if checkOscAssociate {
		return nil
	} else {
		return fmt.Errorf("%s subnet does not exist in loadBalancer", loadBalancerSubnetName)
	}
}

// checkLoadBalancerFormatParameters check LoadBalancer parameters format (Tag format, cidr format, ..)
func checkLoadBalancerFormatParameters(clusterScope *scope.ClusterScope) (string, error) {
	clusterScope.Info("Check LoadBalancer name parameters")
	loadBalancerSpec := clusterScope.GetLoadBalancer()
	loadBalancerSpec.SetDefaultValue()
	loadBalancerName := loadBalancerSpec.LoadBalancerName

	validLoadBalancerName := service.ValidateLoadBalancerName(loadBalancerName)
	if !validLoadBalancerName {
		return loadBalancerName, fmt.Errorf("%s is an invalid loadBalancer name", loadBalancerName)
	}

	loadBalancerType := loadBalancerSpec.LoadBalancerType
	validLoadBalancerType := service.ValidateLoadBalancerType(loadBalancerType)
	if !validLoadBalancerType {
		return loadBalancerName, fmt.Errorf("%s is an invalid loadBalancer type", loadBalancerType)
	}

	loadBalancerBackendPort := loadBalancerSpec.Listener.BackendPort
	_, err := service.ValidatePort(loadBalancerBackendPort)
	if err != nil {
		return loadBalancerName, fmt.Errorf("%d is an %w for loadBalancer backend", loadBalancerBackendPort, err)
	}

	loadBalancerBackendProtocol := loadBalancerSpec.Listener.BackendProtocol
	_, err = service.ValidateProtocol(loadBalancerBackendProtocol)
	if err != nil {
		return loadBalancerName, fmt.Errorf("%s is an %w for loadBalancer backend", loadBalancerBackendProtocol, err)
	}

	loadBalancerPort := loadBalancerSpec.Listener.LoadBalancerPort
	_, err = service.ValidatePort(loadBalancerPort)
	if err != nil {
		return loadBalancerName, fmt.Errorf("%d is an %w for loadBalancer", loadBalancerPort, err)
	}

	loadBalancerProtocol := loadBalancerSpec.Listener.LoadBalancerProtocol
	_, err = service.ValidateProtocol(loadBalancerProtocol)
	if err != nil {
		return loadBalancerName, fmt.Errorf("%s is an %w for loadBalancer", loadBalancerProtocol, err)
	}

	loadBalancerCheckInterval := loadBalancerSpec.HealthCheck.CheckInterval
	_, err = service.ValidateInterval(loadBalancerCheckInterval)
	if err != nil {
		return loadBalancerName, fmt.Errorf("%d is an %w for loadBalancer", loadBalancerCheckInterval, err)
	}

	loadBalancerHealthyThreshold := loadBalancerSpec.HealthCheck.HealthyThreshold
	_, err = service.ValidateThreshold(loadBalancerHealthyThreshold)
	if err != nil {
		return loadBalancerName, fmt.Errorf("%d is an %w for loadBalancer", loadBalancerHealthyThreshold, err)
	}

	loadBalancerHealthCheckPort := loadBalancerSpec.HealthCheck.Port
	_, err = service.ValidatePort(loadBalancerHealthCheckPort)
	if err != nil {
		return loadBalancerName, fmt.Errorf("%d is an %w for loadBalancer", loadBalancerHealthCheckPort, err)
	}

	loadBalancerHealthCheckProtocol := loadBalancerSpec.HealthCheck.Protocol
	_, err = service.ValidateProtocol(loadBalancerHealthCheckProtocol)
	if err != nil {
		return loadBalancerName, fmt.Errorf("%s is an %w for loadBalancer", loadBalancerHealthCheckProtocol, err)
	}

	loadBalancerTimeout := loadBalancerSpec.HealthCheck.Timeout
	_, err = service.ValidateTimeout(loadBalancerTimeout)
	if err != nil {
		return loadBalancerName, fmt.Errorf("%d is an %w for loadBalancer", loadBalancerTimeout, err)
	}

	loadBalancerUnhealthyThreshold := loadBalancerSpec.HealthCheck.UnhealthyThreshold
	_, err = service.ValidateThreshold(loadBalancerUnhealthyThreshold)
	if err != nil {
		return loadBalancerName, fmt.Errorf("%d is an %w for loadBalancer", loadBalancerUnhealthyThreshold, err)
	}

	return "", nil
}

// checkLoadBalancerSecurityOscAssociateResourceName check that LoadBalancer SecurityGroup dependancies tag name in both resource configuration are the same.
func checkLoadBalancerSecurityGroupOscAssociateResourceName(clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	clusterScope.Info("check match securityGroup with loadBalancer")
	loadBalancerSpec := clusterScope.GetLoadBalancer()
	loadBalancerSecurityGroupName := loadBalancerSpec.SecurityGroupName + "-" + clusterScope.GetUID()
	securityGroupsSpec := clusterScope.GetSecurityGroups()
	for _, securityGroupSpec := range securityGroupsSpec {
		securityGroupName := securityGroupSpec.Name + "-" + clusterScope.GetUID()
		resourceNameList = append(resourceNameList, securityGroupName)
	}
	checkOscAssociate := contains(resourceNameList, loadBalancerSecurityGroupName)
	if checkOscAssociate {
		return nil
	} else {
		return fmt.Errorf("%s securityGroup does not exist in loadBalancer", loadBalancerSecurityGroupName)
	}
}

// reconcileLoadBalancer reconciles the loadBalancer of the cluster.
func reconcileLoadBalancer(ctx context.Context, clusterScope *scope.ClusterScope, loadBalancerSvc service.OscLoadBalancerInterface) (reconcile.Result, error) {

	clusterScope.Info("Create Loadbalancer")
	loadBalancerSpec := clusterScope.GetLoadBalancer()
	loadBalancerName := loadBalancerSpec.LoadBalancerName
	clusterScope.Info("Check if the desired loadbalancer exist", "loadBalancerName", loadBalancerName)
	loadbalancer, err := loadBalancerSvc.GetLoadBalancer(loadBalancerSpec)
	if err != nil {
		return reconcile.Result{}, err
	}
	subnetName := loadBalancerSpec.SubnetName + "-" + clusterScope.GetUID()
	subnetId, err := getSubnetResourceId(subnetName, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	securityGroupName := loadBalancerSpec.SecurityGroupName + "-" + clusterScope.GetUID()
	securityGroupId, err := getSecurityGroupResourceId(securityGroupName, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	if loadbalancer == nil {
		clusterScope.Info("### Get lb subnetId ###", "subnet", subnetId)
		clusterScope.Info("### Get lb  sgId ###", "sg", securityGroupId)
		clusterScope.Info("Create the desired loadBalancer", "loadBalancerName", loadBalancerName)
		_, err := loadBalancerSvc.CreateLoadBalancer(loadBalancerSpec, subnetId, securityGroupId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not create loadBalancer for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
		}
		clusterScope.Info("Configure the desired loadBalancer", "loadBalancerName", loadBalancerName)
		loadbalancer, err = loadBalancerSvc.ConfigureHealthCheck(loadBalancerSpec)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not configure healthcheck for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
		}
		clusterScope.Info("### Get lb ###", "loadbalancer", loadbalancer)

	}
	controlPlaneEndpoint := *loadbalancer.DnsName
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

	clusterScope.Info("Delete LoadBalancer")
	loadBalancerSpec := clusterScope.GetLoadBalancer()
	loadBalancerSpec.SetDefaultValue()
	loadBalancerName := loadBalancerSpec.LoadBalancerName

	loadbalancer, err := loadBalancerSvc.GetLoadBalancer(loadBalancerSpec)
	if err != nil {
		return reconcile.Result{}, err
	}
	if loadbalancer == nil {
		clusterScope.Info("the desired loadBalancer does not exist anymore", "loadBalancerName", loadBalancerName)
		controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
		return reconcile.Result{}, nil
	}
	err = loadBalancerSvc.DeleteLoadBalancer(loadBalancerSpec)
	if err != nil {
		clusterScope.Info("Delete the desired loadBalancer", "loadBalancerName", loadBalancerName)
		return reconcile.Result{}, fmt.Errorf("%w Can not delete loadBalancer for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
	}
	return reconcile.Result{}, nil
}
