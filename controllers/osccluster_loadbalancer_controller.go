package controllers

import (
	"context"
	"fmt"

	infrastructurev1beta1 "github.com/outscale-vbr/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/services/service"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// CheckLoadBalancerSubneOscAssociateResourceName check that LoadBalancer Subnet dependancies tag name in both resource configuration are the same.
func CheckLoadBalancerSubnetOscAssociateResourceName(clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	clusterScope.Info("check match subnet with loadBalancer")
	loadBalancerSpec := clusterScope.GetLoadBalancer()
	loadBalancerName := loadBalancerSpec.LoadBalancerName
	loadBalancerSubnetName := loadBalancerSpec.SubnetName + "-" + clusterScope.GetUID()
	var subnetsSpec []*infrastructurev1beta1.OscSubnet
	networkSpec := clusterScope.GetNetwork()
	if networkSpec.Subnets == nil {
		networkSpec.SetSubnetDefaultValue()
		subnetsSpec = networkSpec.Subnets
	} else {
		subnetsSpec = clusterScope.GetSubnet()
	}
	for _, subnetSpec := range subnetsSpec {
		subnetName := subnetSpec.Name + "-" + clusterScope.GetUID()
		resourceNameList = append(resourceNameList, subnetName)
	}
	checkOscAssociate := CheckAssociate(loadBalancerSubnetName, resourceNameList)
	if checkOscAssociate {
		return nil
	} else {
		return fmt.Errorf("%s subnet does not exist in loadBalancer", loadBalancerName)
	}
}

// CheckLoadBalancerFormatParameters check LoadBalancer parameters format (Tag format, cidr format, ..)
func CheckLoadBalancerFormatParameters(clusterScope *scope.ClusterScope) (string, error) {
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
		return loadBalancerName, fmt.Errorf("%s is and invalid loadbalancer type", loadBalancerType)
	}
	return "", nil
}

// CheckLoadBalancerSecurityOscAssociateResourceName check that LoadBalancer SecurityGroup dependancies tag name in both resource configuration are the same.
func CheckLoadBalancerSecurityGroupOscAssociateResourceName(clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	clusterScope.Info("check match securityGroup with loadBalancer")
	loadBalancerSpec := clusterScope.GetLoadBalancer()
	loadBalancerName := loadBalancerSpec.LoadBalancerName
	loadBalancerSecurityGroupName := loadBalancerSpec.SecurityGroupName + "-" + clusterScope.GetUID()
	var securityGroupsSpec []*infrastructurev1beta1.OscSecurityGroup
	networkSpec := clusterScope.GetNetwork()
	if networkSpec.SecurityGroups == nil {
		networkSpec.SetSecurityGroupDefaultValue()
		securityGroupsSpec = networkSpec.SecurityGroups
	} else {
		securityGroupsSpec = clusterScope.GetSecurityGroups()
	}
	for _, securityGroupSpec := range securityGroupsSpec {
		securityGroupName := securityGroupSpec.Name + "-" + clusterScope.GetUID()
		resourceNameList = append(resourceNameList, securityGroupName)
	}
	checkOscAssociate := CheckAssociate(loadBalancerSecurityGroupName, resourceNameList)
	if checkOscAssociate {
		return nil
	} else {
		return fmt.Errorf("%s securityGroup does not exist in loadBalancer", loadBalancerName)
	}
}

// ReconcileLoadBalancer reconciles the loadBalancer of the cluster.
func reconcileLoadBalancer(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	servicesvc := service.NewService(ctx, clusterScope)

	clusterScope.Info("Create Loadbalancer")
	loadBalancerSpec := clusterScope.GetLoadBalancer()
	loadbalancer, err := servicesvc.GetLoadBalancer(loadBalancerSpec)
	if err != nil {
		return reconcile.Result{}, err
	}
	subnetName := loadBalancerSpec.SubnetName + "-" + clusterScope.GetUID()
	subnetId, err := GetSubnetResourceId(subnetName, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	securityGroupName := loadBalancerSpec.SecurityGroupName + "-" + clusterScope.GetUID()
	securityGroupId, err := GetSecurityGroupResourceId(securityGroupName, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	if loadbalancer == nil {
		clusterScope.Info("### Get lb subnetId ###", "subnet", subnetId)
		clusterScope.Info("### Get lb  sgId ###", "sg", securityGroupId)

		_, err := servicesvc.CreateLoadBalancer(loadBalancerSpec, subnetId, securityGroupId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not create load balancer for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
		}
		loadbalancer, err = servicesvc.ConfigureHealthCheck(loadBalancerSpec)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not configure healthcheck for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
		}

	}
	controlPlaneEndpoint := *loadbalancer.DnsName
	controlPlanePort := loadBalancerSpec.Listener.LoadBalancerPort
	clusterScope.SetControlPlaneEndpoint(clusterv1.APIEndpoint{
		Host: controlPlaneEndpoint,
		Port: controlPlanePort,
	})
	return reconcile.Result{}, nil

}

// ReconcileDeleteLoadBalancer reconcile the destruction of the LoadBalancer of the cluster.

func reconcileDeleteLoadBalancer(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	osccluster := clusterScope.OscCluster
	servicesvc := service.NewService(ctx, clusterScope)

	clusterScope.Info("Delete LoadBalancer")
	loadBalancerSpec := clusterScope.GetLoadBalancer()
	loadbalancer, err := servicesvc.GetLoadBalancer(loadBalancerSpec)
	if err != nil {
		return reconcile.Result{}, err
	}
	if loadbalancer == nil {
		controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
		return reconcile.Result{}, nil
	}
	err = servicesvc.DeleteLoadBalancer(loadBalancerSpec)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("%w Can not delete load balancer for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
	}
	clusterScope.Info("Wait LoadBalancer Delete")
	return reconcile.Result{}, nil
}
