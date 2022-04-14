package controllers

import (
	"context"
	"fmt"
	"time"

	infrastructurev1beta1 "github.com/outscale-vbr/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/services/service"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// CheckOscAssociateResourceName check that resourceType dependancies tag name in both resource configuration are the same.
func CheckLoadBalancerSubnetOscAssociateResourceName(clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	clusterScope.Info("check match subnet with loadBalancer")
	loadBalancerSpec := clusterScope.GetLoadBalancer()
	loadBalancerSpec.SetDefaultValue()
	loadBalancerName := loadBalancerSpec.LoadBalancerName
	loadBalancerSubnetName := loadBalancerSpec.SubnetName + "-" + clusterScope.UID()
	var subnetsSpec []*infrastructurev1beta1.OscSubnet
	networkSpec := clusterScope.GetNetwork()
	if networkSpec.Subnets == nil {
		networkSpec.SetSubnetDefaultValue()
		subnetsSpec = networkSpec.Subnets
	} else {
		subnetsSpec = clusterScope.GetSubnet()
	}
	for _, subnetSpec := range subnetsSpec {
		subnetName := subnetSpec.Name + "-" + clusterScope.UID()
		resourceNameList = append(resourceNameList, subnetName)
	}
	checkOscAssociate := CheckAssociate(loadBalancerSubnetName, resourceNameList)
	if checkOscAssociate {
		return nil
	} else {
		return fmt.Errorf("%s subnet does not exist in loadBalancer", loadBalancerName)
	}
}

// CheckFormatParameters check every resource (net, subnet, ...) parameters format (Tag format, cidr format, ..)
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

// CheckOscAssociateResourceName check that resourceType dependancies tag name in both resource configuration are the same.
func CheckLoadBalancerSecurityGroupOscAssociateResourceName(clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	clusterScope.Info("check match securityGroup with loadBalancer")
	loadBalancerSpec := clusterScope.GetLoadBalancer()
	loadBalancerSpec.SetDefaultValue()
	loadBalancerName := loadBalancerSpec.LoadBalancerName
	loadBalancerSecurityGroupName := loadBalancerSpec.SecurityGroupName + "-" + clusterScope.UID()
	var securityGroupsSpec []*infrastructurev1beta1.OscSecurityGroup
	networkSpec := clusterScope.GetNetwork()
	if networkSpec.SecurityGroups == nil {
		networkSpec.SetSecurityGroupDefaultValue()
		securityGroupsSpec = networkSpec.SecurityGroups
	} else {
		securityGroupsSpec = clusterScope.GetSecurityGroups()
	}
	for _, securityGroupSpec := range securityGroupsSpec {
		securityGroupName := securityGroupSpec.Name + "-" + clusterScope.UID()
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
	osccluster := clusterScope.OscCluster

	clusterScope.Info("Create Loadbalancer")
	loadBalancerSpec := clusterScope.GetLoadBalancer()
	loadBalancerSpec.SetDefaultValue()
	loadbalancer, err := servicesvc.GetLoadBalancer(loadBalancerSpec)
	if err != nil {
		return reconcile.Result{}, err
	}
	subnetName := loadBalancerSpec.SubnetName + "-" + clusterScope.UID()
	subnetId, err := GetSubnetResourceId(subnetName, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	securityGroupName := loadBalancerSpec.SecurityGroupName + "-" + clusterScope.UID()
	securityGroupId, err := GetSecurityGroupResourceId(securityGroupName, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	if loadbalancer == nil {
		clusterScope.Info("### Get lb subnetId ###", "subnet", subnetId)
		clusterScope.Info("### Get lb  sgId ###", "sg", securityGroupId)

		_, err := servicesvc.CreateLoadBalancer(loadBalancerSpec, subnetId, securityGroupId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w: Can not create load balancer for Osccluster %s/%s", err, osccluster.Namespace, osccluster.Name)
		}
		loadbalancer, err = servicesvc.ConfigureHealthCheck(loadBalancerSpec)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w: Can not configure healthcheck for Osccluster %s/%s", err, osccluster.Namespace, osccluster.Name)
		}

	}
	controlPlaneEndpoint := *loadbalancer.DnsName
	controlPlanePort := loadBalancerSpec.Listener.LoadBalancerPort
	clusterScope.SetControlPlaneEndpoint(clusterv1.APIEndpoint{
		Host: controlPlaneEndpoint,
		Port: controlPlanePort,
	})
	clusterScope.Info("Waiting on Dns Name")
	return reconcile.Result{}, nil

}

// ReconcileDeleteLoadBalancer reconcile the destruction of the LoadBalancer of the cluster.

func reconcileDeleteLoadBalancer(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	osccluster := clusterScope.OscCluster
	servicesvc := service.NewService(ctx, clusterScope)

	clusterScope.Info("Delete LoadBalancer")
	loadBalancerSpec := clusterScope.GetLoadBalancer()
	loadBalancerSpec.SetDefaultValue()
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
		return reconcile.Result{}, fmt.Errorf("%w: Can not delete load balancer for Osccluster %s/%s", err, osccluster.Namespace, osccluster.Name)
	}
	clusterScope.Info("Wait LoadBalancer Delete")
	time.Sleep(45 * time.Second)
	return reconcile.Result{RequeueAfter: 45 * time.Second}, nil
}
