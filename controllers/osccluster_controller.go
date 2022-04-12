/*
Copyright 2022.

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
	"time"

	infrastructurev1beta1 "github.com/outscale-vbr/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"

	//      "k8s.io/apimachinery/pkg/runtime"
	"github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/services/net"
	"github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/services/security"
	"github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/services/service"
	tag "github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/tag"
	"github.com/outscale-vbr/cluster-api-provider-outscale.git/util/reconciler"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/conditions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// OscClusterReconciler reconciles a OscCluster object
type OscClusterReconciler struct {
	client.Client
	Recorder         record.EventRecorder
	ReconcileTimeout time.Duration
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=oscclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=oscclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=oscclusters/finalizers,verbs=update
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters,verbs=get;list;watch
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters/status,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=events,verbs=create;get;list;patch;update;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the OscCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *OscClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	_ = log.FromContext(ctx)
	ctx, cancel := context.WithTimeout(ctx, reconciler.DefaultedLoopTimeout(r.ReconcileTimeout))
	defer cancel()
	log := ctrl.LoggerFrom(ctx)
	oscCluster := &infrastructurev1beta1.OscCluster{}

	log.Info("Please WAIT !!!!")

	if err := r.Get(ctx, req.NamespacedName, oscCluster); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("object was not found")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	cluster, err := util.GetOwnerCluster(ctx, r.Client, oscCluster.ObjectMeta)
	if err != nil {
		return reconcile.Result{}, err
	}
	if cluster == nil {
		log.Info("Cluster Controller has not yet set OwnerRef")
		return reconcile.Result{}, nil
	}

	// Return early if the object or Cluster is paused.
	if annotations.IsPaused(cluster, oscCluster) {
		log.Info("oscCluster or linked Cluster is marked as paused. Won't reconcile")
		return ctrl.Result{}, nil
	}

	// Create the cluster scope.
	clusterScope, err := scope.NewClusterScope(scope.ClusterScopeParams{
		Client:     r.Client,
		Logger:     log,
		Cluster:    cluster,
		OscCluster: oscCluster,
	})
	if err != nil {
		return reconcile.Result{}, errors.Errorf("failed to create scope: %+v", err)
	}
	defer func() {
		if err := clusterScope.Close(); err != nil && reterr == nil {
			reterr = err
		}
	}()
	osccluster := clusterScope.OscCluster
	if !osccluster.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, clusterScope)
	}
	loadBalancerSpec := clusterScope.LoadBalancer()
	loadBalancerSpec.SetDefaultValue()
	log.Info("Create loadBalancer", "loadBalancerName", loadBalancerSpec.LoadBalancerName)
	return r.reconcile(ctx, clusterScope)
}

// GetResourceId return the resourceId from the resourceMap base on resourceName (tag name + cluster object uid) and resourceType (net, subnet, gateway, route, route-table, public-ip)
func GetResourceId(resourceName string, resourceType string, clusterScope *scope.ClusterScope) (string, error) {
	switch {
	case resourceType == "net":
		netRef := clusterScope.NetRef()
		if netId, ok := netRef.ResourceMap[resourceName]; ok {
			return netId, nil
		} else {
			return "", fmt.Errorf("%s is not exist", resourceName)
		}
	case resourceType == "subnet":
		subnetRef := clusterScope.SubnetRef()
		if subnetId, ok := subnetRef.ResourceMap[resourceName]; ok {
			return subnetId, nil
		} else {
			return "", fmt.Errorf("%s is not exist", resourceName)
		}
	case resourceType == "gateway":
		internetServiceRef := clusterScope.InternetServiceRef()
		if internetServiceId, ok := internetServiceRef.ResourceMap[resourceName]; ok {
			return internetServiceId, nil
		} else {
			return "", fmt.Errorf("%s is not exist", resourceName)
		}
	case resourceType == "route-table":
		routeTableRef := clusterScope.RouteTablesRef()
		if routeTableId, ok := routeTableRef.ResourceMap[resourceName]; ok {
			return routeTableId, nil
		} else {
			return "", fmt.Errorf("%s is not exist", resourceName)
		}
	case resourceType == "security-group":
		securityGroupRef := clusterScope.SecurityGroupsRef()
		if securityGroupId, ok := securityGroupRef.ResourceMap[resourceName]; ok {
			return securityGroupId, nil
		} else {
			return "", fmt.Errorf("%s is not exist", resourceName)
		}
	case resourceType == "route":
		routeRef := clusterScope.RouteRef()
		if routeId, ok := routeRef.ResourceMap[resourceName]; ok {
			return routeId, nil
		} else {
			return "", fmt.Errorf("%s is not exist", resourceName)
		}
	case resourceType == "security-group-rule":
		securityGroupRuleRef := clusterScope.SecurityGroupRuleRef()
		if securityGroupRuleId, ok := securityGroupRuleRef.ResourceMap[resourceName]; ok {
			return securityGroupRuleId, nil
		} else {
			return "", fmt.Errorf("%s is not exist", resourceName)
		}
	case resourceType == "public-ip":
		publicIpRef := clusterScope.PublicIpRef()
		if publicIpId, ok := publicIpRef.ResourceMap[resourceName]; ok {
			return publicIpId, nil
		} else {
			return "", fmt.Errorf("%s is not exist", resourceName)
		}
	case resourceType == "nat":
		natServiceRef := clusterScope.NatServiceRef()
		if natServiceId, ok := natServiceRef.ResourceMap[resourceName]; ok {
			return natServiceId, nil
		} else {
			return "", fmt.Errorf("%s is not exist", resourceName)
		}

	default:
		return "", fmt.Errorf("%s does not exist", resourceType)
	}
}

// CheckAssociate return if the resourcename is an item of firstResourceNameArray
func CheckAssociate(resourceName string, firstResourceNameArray []string) bool {
	for i := 0; i < len(firstResourceNameArray); i++ {
		if firstResourceNameArray[i] == resourceName {
			return true
		}
	}
	return false
}

// CheckFormatParameters check every resource (net, subnet, ...) parameters format (Tag format, cidr format, ..)
func CheckFormatParameters(resourceType string, clusterScope *scope.ClusterScope) (string, error) {
	switch {
	case resourceType == "loadbalancer":
		clusterScope.Info("Check LoadBalancer name parameters")
		loadBalancerSpec := clusterScope.LoadBalancer()
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

	case resourceType == "net":
		clusterScope.Info("Check Net name parameters ")
		netSpec := clusterScope.Net()
		netSpec.SetDefaultValue()
		netName := netSpec.Name + "-" + clusterScope.UID()
		netTagName, err := tag.ValidateTagNameValue(netName)
		if err != nil {
			return netTagName, err
		}
		clusterScope.Info("Check Net IpRange parameters")
		netIpRange := netSpec.IpRange
		_, err = net.ValidateCidr(netIpRange)
		if err != nil {
			return netTagName, err
		}
	case resourceType == "subnet":
		clusterScope.Info("Check subnet name parameters")
		var subnetsSpec []*infrastructurev1beta1.OscSubnet
		networkSpec := clusterScope.Network()
		if networkSpec.Subnets == nil {
			networkSpec.SetSubnetDefaultValue()
			subnetsSpec = networkSpec.Subnets
		} else {
			subnetsSpec = clusterScope.Subnet()
		}
		for _, subnetSpec := range subnetsSpec {
			subnetName := subnetSpec.Name + "-" + clusterScope.UID()
			subnetTagName, err := tag.ValidateTagNameValue(subnetName)
			if err != nil {
				return subnetTagName, err
			}
		}
	case resourceType == "internet-service":
		clusterScope.Info("Check Internet Service parameters")
		internetServiceSpec := clusterScope.InternetService()
		internetServiceSpec.SetDefaultValue()
		internetServiceName := internetServiceSpec.Name + "-" + clusterScope.UID()
		internetServiceTagName, err := tag.ValidateTagNameValue(internetServiceName)
		if err != nil {
			return internetServiceTagName, err
		}
	case resourceType == "public-ip":
		clusterScope.Info("Check Public Ip parameters")
		var publicIpsSpec []*infrastructurev1beta1.OscPublicIp
		networkSpec := clusterScope.Network()
		if networkSpec.PublicIps == nil {
			networkSpec.SetPublicIpDefaultValue()
			publicIpsSpec = networkSpec.PublicIps
		} else {
			publicIpsSpec = clusterScope.PublicIp()
		}
		for _, publicIpSpec := range publicIpsSpec {
			publicIpName := publicIpSpec.Name + "-" + clusterScope.UID()
			publicIpTagName, err := tag.ValidateTagNameValue(publicIpName)
			if err != nil {
				return publicIpTagName, err
			}
		}
	case resourceType == "route-table":
		clusterScope.Info("Check Route table parameters")
		var routeTablesSpec []*infrastructurev1beta1.OscRouteTable
		networkSpec := clusterScope.Network()
		if networkSpec.RouteTables == nil {
			networkSpec.SetRouteTableDefaultValue()
			routeTablesSpec = networkSpec.RouteTables
		} else {
			routeTablesSpec = clusterScope.RouteTables()
		}
		for _, routeTableSpec := range routeTablesSpec {
			routeTableName := routeTableSpec.Name + "-" + clusterScope.UID()
			routeTableTagName, err := tag.ValidateTagNameValue(routeTableName)
			if err != nil {
				return routeTableTagName, err
			}
		}
	case resourceType == "security-group":
		clusterScope.Info("Check security group parameters")
		var securityGroupsSpec []*infrastructurev1beta1.OscSecurityGroup
		networkSpec := clusterScope.Network()
		if networkSpec.SecurityGroups == nil {
			networkSpec.SetSecurityGroupDefaultValue()
			securityGroupsSpec = networkSpec.SecurityGroups
		} else {
			securityGroupsSpec = clusterScope.SecurityGroups()
		}
		for _, securityGroupSpec := range securityGroupsSpec {
			securityGroupName := securityGroupSpec.Name + "-" + clusterScope.UID()
			securityGroupTagName, err := tag.ValidateTagNameValue(securityGroupName)
			if err != nil {
				return securityGroupTagName, err
			}
			securityGroupDescription := securityGroupSpec.Description
			_, err = security.ValidateDescription(securityGroupDescription)
			if err != nil {
				return securityGroupTagName, err
			}
		}

	case resourceType == "route":
		clusterScope.Info("Check Route parameters")
		var routeTablesSpec []*infrastructurev1beta1.OscRouteTable
		networkSpec := clusterScope.Network()
		if networkSpec.RouteTables == nil {
			networkSpec.SetRouteTableDefaultValue()
			routeTablesSpec = networkSpec.RouteTables
		} else {
			routeTablesSpec = clusterScope.RouteTables()
		}
		for _, routeTableSpec := range routeTablesSpec {
			routesSpec := clusterScope.Route(routeTableSpec.Name)
			for _, routeSpec := range *routesSpec {
				routeName := routeSpec.Name + "-" + clusterScope.UID()
				routeTagName, err := tag.ValidateTagNameValue(routeName)
				if err != nil {
					return routeTagName, err
				}
				clusterScope.Info("Check route destination IpRange parameters")
				destinationIpRange := routeSpec.Destination
				_, err = net.ValidateCidr(destinationIpRange)
				if err != nil {
					return routeTagName, err
				}
			}
		}
	case resourceType == "security-group-rule":
		clusterScope.Info("Check security Group rule parameters")
		var securityGroupsSpec []*infrastructurev1beta1.OscSecurityGroup
		networkSpec := clusterScope.Network()
		if networkSpec.SecurityGroups == nil {
			networkSpec.SetSecurityGroupDefaultValue()
			securityGroupsSpec = networkSpec.SecurityGroups
		} else {
			securityGroupsSpec = clusterScope.SecurityGroups()
		}
		for _, securityGroupSpec := range securityGroupsSpec {
			securityGroupRulesSpec := clusterScope.SecurityGroupRule(securityGroupSpec.Name)
			for _, securityGroupRuleSpec := range *securityGroupRulesSpec {
				securityGroupRuleName := securityGroupRuleSpec.Name + "-" + clusterScope.UID()
				securityGroupRuleTagName, err := tag.ValidateTagNameValue(securityGroupRuleName)
				if err != nil {
					return securityGroupRuleTagName, err
				}
				securityGroupRuleFlow := securityGroupRuleSpec.Flow
				_, err = security.ValidateFlow(securityGroupRuleFlow)
				if err != nil {
					return securityGroupRuleTagName, err
				}
				securityGroupRuleIpProtocol := securityGroupRuleSpec.IpProtocol
				_, err = security.ValidateIpProtocol(securityGroupRuleIpProtocol)
				if err != nil {
					return securityGroupRuleTagName, err
				}
				securityGroupRuleIpRange := securityGroupRuleSpec.IpRange
				_, err = net.ValidateCidr(securityGroupRuleIpRange)
				if err != nil {
					return securityGroupRuleTagName, err
				}
				securityGroupRuleFromPortRange := securityGroupRuleSpec.FromPortRange
				_, err = service.ValidatePort(securityGroupRuleFromPortRange)
				if err != nil {
					return securityGroupRuleTagName, err
				}
				securityGroupRuleToPortRange := securityGroupRuleSpec.ToPortRange
				_, err = service.ValidatePort(securityGroupRuleToPortRange)
				if err != nil {
					return securityGroupRuleTagName, err
				}

			}
		}

	}
	return "", nil
}

// CheckOscAssociateResourceName check that resourceType dependancies tag name in both resource configuration are the same.
func CheckOscAssociateResourceName(resourceType string, clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	switch {
	case resourceType == "public-ip":
		clusterScope.Info("check match public ip with nat service")
		natServiceSpec := clusterScope.NatService()
		natServiceSpec.SetDefaultValue()
		natPublicIpName := natServiceSpec.PublicIpName + "-" + clusterScope.UID()
		var publicIpsSpec []*infrastructurev1beta1.OscPublicIp
		networkSpec := clusterScope.Network()
		if networkSpec.PublicIps == nil {
			networkSpec.SetPublicIpDefaultValue()
			publicIpsSpec = networkSpec.PublicIps
		} else {
			publicIpsSpec = clusterScope.PublicIp()
		}
		for _, publicIpSpec := range publicIpsSpec {
			publicIpName := publicIpSpec.Name + "-" + clusterScope.UID()
			resourceNameList = append(resourceNameList, publicIpName)
		}
		checkOscAssociate := CheckAssociate(natPublicIpName, resourceNameList)
		if checkOscAssociate {
			return nil
		} else {
			return fmt.Errorf("publicIp %s does not exist in natService ", natPublicIpName)
		}
	case resourceType == "natSubnet":
		clusterScope.Info("check match subnet with nat service")
		natServiceSpec := clusterScope.NatService()
		natServiceSpec.SetDefaultValue()
		natSubnetName := natServiceSpec.SubnetName + "-" + clusterScope.UID()
		var subnetsSpec []*infrastructurev1beta1.OscSubnet
		networkSpec := clusterScope.Network()
		if networkSpec.Subnets == nil {
			networkSpec.SetSubnetDefaultValue()
			subnetsSpec = networkSpec.Subnets
		} else {
			subnetsSpec = clusterScope.Subnet()
		}
		for _, subnetSpec := range subnetsSpec {
			subnetName := subnetSpec.Name + "-" + clusterScope.UID()
			resourceNameList = append(resourceNameList, subnetName)
		}
		checkOscAssociate := CheckAssociate(natSubnetName, resourceNameList)
		if checkOscAssociate {
			return nil
		} else {
			return fmt.Errorf("%s subnet does not exist in natService", natSubnetName)
		}
	case resourceType == "loadBalancerSubnet":
		clusterScope.Info("check match subnet with loadBalancer")
		loadBalancerSpec := clusterScope.LoadBalancer()
		loadBalancerSpec.SetDefaultValue()
		loadBalancerName := loadBalancerSpec.LoadBalancerName
		loadBalancerSubnetName := loadBalancerSpec.SubnetName + "-" + clusterScope.UID()
		var subnetsSpec []*infrastructurev1beta1.OscSubnet
		networkSpec := clusterScope.Network()
		if networkSpec.Subnets == nil {
			networkSpec.SetSubnetDefaultValue()
			subnetsSpec = networkSpec.Subnets
		} else {
			subnetsSpec = clusterScope.Subnet()
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
	case resourceType == "loadBalancerSecurityGroup":
		clusterScope.Info("check match securityGroup with loadBalancer")
		loadBalancerSpec := clusterScope.LoadBalancer()
		loadBalancerSpec.SetDefaultValue()
		loadBalancerName := loadBalancerSpec.LoadBalancerName
		loadBalancerSecurityGroupName := loadBalancerSpec.SecurityGroupName + "-" + clusterScope.UID()
		var securityGroupsSpec []*infrastructurev1beta1.OscSecurityGroup
		networkSpec := clusterScope.Network()
		if networkSpec.SecurityGroups == nil {
			networkSpec.SetSecurityGroupDefaultValue()
			securityGroupsSpec = networkSpec.SecurityGroups
		} else {
			securityGroupsSpec = clusterScope.SecurityGroups()
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

	case resourceType == "routeTableSubnet":
		clusterScope.Info("check match subnet with route table service")
		var routeTablesSpec []*infrastructurev1beta1.OscRouteTable
		networkSpec := clusterScope.Network()
		if networkSpec.RouteTables == nil {
			networkSpec.SetRouteTableDefaultValue()
			routeTablesSpec = networkSpec.RouteTables
		} else {
			routeTablesSpec = clusterScope.RouteTables()
		}
		resourceNameList = resourceNameList[:0]
		var subnetsSpec []*infrastructurev1beta1.OscSubnet
		if networkSpec.Subnets == nil {
			networkSpec.SetSubnetDefaultValue()
			subnetsSpec = networkSpec.Subnets
		} else {
			subnetsSpec = clusterScope.Subnet()
		}
		for _, subnetSpec := range subnetsSpec {
			subnetName := subnetSpec.Name + "-" + clusterScope.UID()
			resourceNameList = append(resourceNameList, subnetName)
		}
		for _, routeTableSpec := range routeTablesSpec {
			routeTableSubnetName := routeTableSpec.SubnetName + "-" + clusterScope.UID()
			checkOscAssociate := CheckAssociate(routeTableSubnetName, resourceNameList)
			if checkOscAssociate {
				return nil
			} else {
				return fmt.Errorf("%s subnet dooes not exist in routeTable", routeTableSubnetName)
			}
		}
	}

	return nil
}

// CheckOscDuplicateName check that there are not the same name for resource with the same kind of resourceType (route-table, subnet, ..).
func CheckOscDuplicateName(resourceType string, clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	switch {
	case resourceType == "route-table":
		clusterScope.Info("check unique routetable")
		var routeTablesSpec []*infrastructurev1beta1.OscRouteTable
		networkSpec := clusterScope.Network()
		if networkSpec.RouteTables == nil {
			networkSpec.SetRouteTableDefaultValue()
			routeTablesSpec = networkSpec.RouteTables
		} else {
			routeTablesSpec = clusterScope.RouteTables()
		}
		for _, routeTableSpec := range routeTablesSpec {
			resourceNameList = append(resourceNameList, routeTableSpec.Name)
		}
		duplicateResourceErr := AlertDuplicate(resourceNameList)
		if duplicateResourceErr != nil {
			return duplicateResourceErr
		} else {
			return nil
		}
		return nil
	case resourceType == "security-group":
		clusterScope.Info("check unique securityGroup")
		var securityGroupsSpec []*infrastructurev1beta1.OscSecurityGroup
		networkSpec := clusterScope.Network()
		if networkSpec.SecurityGroups == nil {
			networkSpec.SetSecurityGroupDefaultValue()
			securityGroupsSpec = networkSpec.SecurityGroups
		} else {
			securityGroupsSpec = clusterScope.SecurityGroups()
		}
		for _, securityGroupSpec := range securityGroupsSpec {
			resourceNameList = append(resourceNameList, securityGroupSpec.Name)
		}
		duplicateResourceErr := AlertDuplicate(resourceNameList)
		if duplicateResourceErr != nil {
			return duplicateResourceErr
		} else {
			return nil
		}

	case resourceType == "route":
		clusterScope.Info("check unique route")
		routeTablesSpec := clusterScope.RouteTables()
		for _, routeTableSpec := range routeTablesSpec {
			routesSpec := clusterScope.Route(routeTableSpec.Name)
			for _, routeSpec := range *routesSpec {
				resourceNameList = append(resourceNameList, routeSpec.Name)
			}
			duplicateResourceErr := AlertDuplicate(resourceNameList)
			if duplicateResourceErr != nil {
				return duplicateResourceErr
			} else {
				return nil
			}
		}
		return nil
	case resourceType == "security-group-rule":
		clusterScope.Info("check unique security group rule")
		securityGroupsSpec := clusterScope.SecurityGroups()
		for _, securityGroupSpec := range securityGroupsSpec {
			securityGroupRulesSpec := clusterScope.SecurityGroupRule(securityGroupSpec.Name)
			for _, securityGroupRuleSpec := range *securityGroupRulesSpec {
				resourceNameList = append(resourceNameList, securityGroupRuleSpec.Name)
			}
			duplicateResourceErr := AlertDuplicate(resourceNameList)
			if duplicateResourceErr != nil {
				return duplicateResourceErr
			} else {
				return nil
			}
		}
	case resourceType == "public-ip":
		clusterScope.Info("Check unique name publicIp")
		var publicIpsSpec []*infrastructurev1beta1.OscPublicIp
		networkSpec := clusterScope.Network()
		if networkSpec.PublicIps == nil {
			networkSpec.SetPublicIpDefaultValue()
			publicIpsSpec = networkSpec.PublicIps
		} else {
			publicIpsSpec = clusterScope.PublicIp()
		}
		for _, publicIpSpec := range publicIpsSpec {
			resourceNameList = append(resourceNameList, publicIpSpec.Name)
		}
		duplicateResourceErr := AlertDuplicate(resourceNameList)
		if duplicateResourceErr != nil {
			return duplicateResourceErr
		} else {
			return nil
		}
		return nil
	case resourceType == "subnet":
		clusterScope.Info("Check unique subnet")
		var subnetsSpec []*infrastructurev1beta1.OscSubnet
		networkSpec := clusterScope.Network()
		if networkSpec.Subnets == nil {
			networkSpec.SetSubnetDefaultValue()
			subnetsSpec = networkSpec.Subnets
		} else {
			subnetsSpec = clusterScope.Subnet()
		}
		for _, subnetSpec := range subnetsSpec {
			resourceNameList = append(resourceNameList, subnetSpec.Name)
		}
		duplicateResourceErr := AlertDuplicate(resourceNameList)
		if duplicateResourceErr != nil {
			return duplicateResourceErr
		} else {
			return nil
		}

	default:
		return nil
	}

	return nil
}

// AlertDuplicate alert if item is present more than once in array
func AlertDuplicate(nameArray []string) error {
	checkMap := make(map[string]bool, 0)
	for i := 0; i < len(nameArray); i++ {
		if checkMap[nameArray[i]] == true {
			return fmt.Errorf("%s already exist", nameArray[i])
		} else {
			checkMap[nameArray[i]] = true
		}
	}
	return nil
}

// contains check if item is present in slice
func contains(slice []string, item string) bool {
	for _, val := range slice {
		if val == item {
			return true
		}
	}
	return false
}

// ReconcileLoadBalancer reconciles the loadBalancer of the cluster.
func reconcileLoadBalancer(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	servicesvc := service.NewService(ctx, clusterScope)
	osccluster := clusterScope.OscCluster

	clusterScope.Info("Create Loadbalancer")
	loadBalancerSpec := clusterScope.LoadBalancer()
	loadBalancerSpec.SetDefaultValue()
	loadbalancer, err := servicesvc.GetLoadBalancer(loadBalancerSpec)
	if err != nil {
		return reconcile.Result{}, err
	}
	subnetName := loadBalancerSpec.SubnetName + "-" + clusterScope.UID()
	subnetId, err := GetResourceId(subnetName, "subnet", clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	securityGroupName := loadBalancerSpec.SecurityGroupName + "-" + clusterScope.UID()
	securityGroupId, err := GetResourceId(securityGroupName, "security-group", clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	if loadbalancer == nil {
		clusterScope.Info("### Get lb subnetId ###", "subnet", subnetId)
		clusterScope.Info("### Get lb  sgId ###", "sg", securityGroupId)

		_, err := servicesvc.CreateLoadBalancer(loadBalancerSpec, subnetId, securityGroupId)
		if err != nil {
			return reconcile.Result{}, errors.Wrapf(err, "Can not create load balancer for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
		}
		loadbalancer, err = servicesvc.ConfigureHealthCheck(loadBalancerSpec)
		if err != nil {
			return reconcile.Result{}, errors.Wrapf(err, "Can not configure healthcheck for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
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

// ReconcileNet reconcile the Net of the cluster.
func reconcileNet(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {

	netsvc := net.NewService(ctx, clusterScope)
	osccluster := clusterScope.OscCluster

	clusterScope.Info("Create Net")
	netSpec := clusterScope.Net()
	netSpec.SetDefaultValue()
	netRef := clusterScope.NetRef()
	netName := netSpec.Name + "-" + clusterScope.UID()
	if len(netRef.ResourceMap) == 0 {
		netRef.ResourceMap = make(map[string]string)
	}
	if netSpec.ResourceId != "" {
		netRef.ResourceMap[netName] = netSpec.ResourceId
	}
	var netIds = []string{netRef.ResourceMap[netName]}
	clusterScope.Info("### Get netId ###", "net", netRef.ResourceMap)
	net, err := netsvc.GetNet(netIds)
	if err != nil {
		return reconcile.Result{}, err
	}
	if net == nil {
		net, err = netsvc.CreateNet(netSpec, netName)
		if err != nil {
			return reconcile.Result{}, errors.Wrapf(err, "Can not create net for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
		}
		clusterScope.Info("### Get net ###", "net", net)
		netRef.ResourceMap[netName] = *net.NetId
		netSpec.ResourceId = *net.NetId

	}
	netRef.ResourceMap[netName] = *net.NetId
	netSpec.ResourceId = *net.NetId
	clusterScope.Info("Info net", "net", net)
	return reconcile.Result{}, nil
}

// ReconcileSubnet reconcile the subnet of the cluster.
func reconcileSubnet(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	netsvc := net.NewService(ctx, clusterScope)
	osccluster := clusterScope.OscCluster

	clusterScope.Info("Create Subnet")

	netSpec := clusterScope.Net()
	netSpec.SetDefaultValue()
	netName := netSpec.Name + "-" + clusterScope.UID()
	netId, err := GetResourceId(netName, "net", clusterScope)
	netIds := []string{netId}
	if err != nil {
		return reconcile.Result{}, err
	}
	var subnetsSpec []*infrastructurev1beta1.OscSubnet
	networkSpec := clusterScope.Network()
	if networkSpec.Subnets == nil {
		networkSpec.SetSubnetDefaultValue()
		subnetsSpec = networkSpec.Subnets
	} else {
		subnetsSpec = clusterScope.Subnet()
	}
	subnetRef := clusterScope.SubnetRef()
	clusterScope.Info("### Get subnetId ###", "subnet", subnetRef.ResourceMap)
	var subnetIds []string
	subnetIds, err = netsvc.GetSubnetIdsFromNetIds(netIds)
	if err != nil {
		return reconcile.Result{}, err
	}
	for _, subnetSpec := range subnetsSpec {
		subnetName := subnetSpec.Name + "-" + clusterScope.UID()
		subnetId := subnetRef.ResourceMap[subnetName]
		if len(subnetRef.ResourceMap) == 0 {
			subnetRef.ResourceMap = make(map[string]string)
		}
		if subnetSpec.ResourceId != "" {
			subnetRef.ResourceMap[subnetName] = subnetSpec.ResourceId
		}
		if !contains(subnetIds, subnetId) {
			subnet, err := netsvc.CreateSubnet(subnetSpec, netId, subnetName)
			if err != nil {
				return reconcile.Result{}, errors.Wrapf(err, "Can not create subnet for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
			}
			clusterScope.Info("### Get subnet ###", "subnet", subnet)
			subnetRef.ResourceMap[subnetName] = *subnet.SubnetId
			subnetSpec.ResourceId = *subnet.SubnetId
		}
	}
	return reconcile.Result{}, nil
}

// ReconcileInternetService reconcile the InternetService of the cluster.
func reconcileInternetService(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	netsvc := net.NewService(ctx, clusterScope)
	osccluster := clusterScope.OscCluster

	clusterScope.Info("Create InternetGateway")
	internetServiceSpec := clusterScope.InternetService()
	internetServiceSpec.SetDefaultValue()
	internetServiceRef := clusterScope.InternetServiceRef()
	internetServiceName := internetServiceSpec.Name + "-" + clusterScope.UID()

	netSpec := clusterScope.Net()
	netSpec.SetDefaultValue()
	netName := netSpec.Name + "-" + clusterScope.UID()
	netId, err := GetResourceId(netName, "net", clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	if len(internetServiceRef.ResourceMap) == 0 {
		internetServiceRef.ResourceMap = make(map[string]string)
	}
	if internetServiceSpec.ResourceId != "" {
		internetServiceRef.ResourceMap[internetServiceName] = internetServiceSpec.ResourceId
	}
	var internetServiceIds = []string{internetServiceRef.ResourceMap[internetServiceName]}
	clusterScope.Info("### Get internetServiceId ###", "internetservice", internetServiceRef.ResourceMap)
	internetService, err := netsvc.GetInternetService(internetServiceIds)
	if err != nil {
		return reconcile.Result{}, err
	}
	if internetService == nil {
		internetService, err = netsvc.CreateInternetService(internetServiceName)
		if err != nil {
			return reconcile.Result{}, errors.Wrapf(err, "Can not create internetservice for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
		}
		err = netsvc.LinkInternetService(*internetService.InternetServiceId, netId)
		if err != nil {
			return reconcile.Result{}, errors.Wrapf(err, "Can not link internetService with net for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
		}
		clusterScope.Info("### Get internetService ###", "internetservice", internetService)
		internetServiceRef.ResourceMap[internetServiceName] = *internetService.InternetServiceId
		internetServiceSpec.ResourceId = *internetService.InternetServiceId

	}
	return reconcile.Result{}, nil
}

// ReconcilePublicIp reconcile the PublicIp of the cluster.
func reconcilePublicIp(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	securitysvc := security.NewService(ctx, clusterScope)
	osccluster := clusterScope.OscCluster

	clusterScope.Info("Create PublicIp")
	var publicIpsSpec []*infrastructurev1beta1.OscPublicIp
	networkSpec := clusterScope.Network()
	if networkSpec.PublicIps == nil {
		networkSpec.SetPublicIpDefaultValue()
		publicIpsSpec = networkSpec.PublicIps
	} else {
		publicIpsSpec = clusterScope.PublicIp()
	}
	publicIpRef := clusterScope.PublicIpRef()
	var publicIpsId []string
	for _, publicIpSpec := range publicIpsSpec {
		publicIpName := publicIpSpec.Name + "-" + clusterScope.UID()
		publicIpId := publicIpRef.ResourceMap[publicIpName]
		publicIpsId = append(publicIpsId, publicIpId)
	}
	publicIpIds, err := securitysvc.ValidatePublicIpIds(publicIpsId)
	if err != nil {
		return reconcile.Result{}, err
	}
	clusterScope.Info("### Check Id  ###", "publicip", publicIpIds)
	for _, publicIpSpec := range publicIpsSpec {
		publicIpName := publicIpSpec.Name + "-" + clusterScope.UID()
		publicIpId := publicIpRef.ResourceMap[publicIpName]
		clusterScope.Info("### Get publicIp Id ###", "publicip", publicIpRef.ResourceMap)
		if publicIpSpec.ResourceId != "" {
			publicIpRef.ResourceMap[publicIpName] = publicIpSpec.ResourceId
		}
		if len(publicIpRef.ResourceMap) == 0 {
			publicIpRef.ResourceMap = make(map[string]string)
		}
		if !contains(publicIpIds, publicIpId) {
			publicIp, err := securitysvc.CreatePublicIp(publicIpName)
			if err != nil {
				return reconcile.Result{}, errors.Wrapf(err, "Can not create publicIp for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
			}
			clusterScope.Info("### Get publicIp  ###", "publicip", publicIp)
			publicIpRef.ResourceMap[publicIpName] = *publicIp.PublicIpId
			publicIpSpec.ResourceId = *publicIp.PublicIpId
		}
	}
	return reconcile.Result{}, nil
}

func reconcileSecurityGroup(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	securitysvc := security.NewService(ctx, clusterScope)
	osccluster := clusterScope.OscCluster

	clusterScope.Info("Create SecurityGroup")
	var securityGroupsSpec []*infrastructurev1beta1.OscSecurityGroup
	networkSpec := clusterScope.Network()
	if networkSpec.SecurityGroups == nil {
		networkSpec.SetSecurityGroupDefaultValue()
		securityGroupsSpec = networkSpec.SecurityGroups
	} else {
		securityGroupsSpec = clusterScope.SecurityGroups()
	}

	netSpec := clusterScope.Net()
	netSpec.SetDefaultValue()
	netName := netSpec.Name + "-" + clusterScope.UID()
	netId, err := GetResourceId(netName, "net", clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	netIds := []string{netId}
	clusterScope.Info("### Get net Id ###", "net", netIds)

	securityGroupIds, err := securitysvc.GetSecurityGroupIdsFromNetIds(netIds)
	if err != nil {
		return reconcile.Result{}, err
	}
	securityGroupsRef := clusterScope.SecurityGroupsRef()
	securityGroupRuleRef := clusterScope.SecurityGroupRuleRef()
	for _, securityGroupSpec := range securityGroupsSpec {
		securityGroupName := securityGroupSpec.Name + "-" + clusterScope.UID()
		securityGroupDescription := securityGroupSpec.Description
		securityGroupId := securityGroupsRef.ResourceMap[securityGroupName]
		clusterScope.Info("### Get securityGroup Id ###", "securityGroup", securityGroupIds)
		if len(securityGroupsRef.ResourceMap) == 0 {
			securityGroupsRef.ResourceMap = make(map[string]string)
		}
		if securityGroupSpec.ResourceId != "" {
			securityGroupsRef.ResourceMap[securityGroupName] = securityGroupSpec.ResourceId
		}
		if !contains(securityGroupIds, securityGroupId) {
			securityGroup, err := securitysvc.CreateSecurityGroup(netId, securityGroupName, securityGroupDescription)
			if err != nil {
				return reconcile.Result{}, errors.Wrapf(err, "Can not create securitygroup for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
			}
			clusterScope.Info("### Get securityGroup", "securityGroup", securityGroup)
			securityGroupsRef.ResourceMap[securityGroupName] = *securityGroup.SecurityGroupId
			securityGroupSpec.ResourceId = *securityGroup.SecurityGroupId

			clusterScope.Info("check securityGroupRule")
			securityGroupRulesSpec := clusterScope.SecurityGroupRule(securityGroupSpec.Name)
			for _, securityGroupRuleSpec := range *securityGroupRulesSpec {
				securityGroupRuleName := securityGroupRuleSpec.Name + "-" + clusterScope.UID()
				if len(securityGroupRuleRef.ResourceMap) == 0 {
					securityGroupRuleRef.ResourceMap = make(map[string]string)
				}
				if securityGroupRuleSpec.ResourceId != "" {
					securityGroupRuleRef.ResourceMap[securityGroupRuleName] = securityGroupRuleSpec.ResourceId
				}
				if err != nil {
					return reconcile.Result{}, err
				}
				Flow := securityGroupRuleSpec.Flow
				IpProtocol := securityGroupRuleSpec.IpProtocol
				IpProtocols := []string{IpProtocol}
				IpRange := securityGroupRuleSpec.IpRange
				IpRanges := []string{IpRange}
				FromPortRange := securityGroupRuleSpec.FromPortRange
				FromPortRanges := []int32{FromPortRange}
				ToPortRange := securityGroupRuleSpec.ToPortRange
				ToPortRanges := []int32{ToPortRange}
				associateSecurityGroupIds := []string{securityGroupsRef.ResourceMap[securityGroupName]}

				securityGroupFromSecurityGroupRule, err := securitysvc.GetSecurityGroupFromSecurityGroupRule(associateSecurityGroupIds, Flow, IpProtocols, IpRanges, FromPortRanges, ToPortRanges)
				clusterScope.Info("### Retrieve securityGroup", "securityGroup", securityGroupFromSecurityGroupRule)
				clusterScope.Info("### Retrieve sg info", "securityGroup", associateSecurityGroupIds)
				clusterScope.Info("### Retrieve sg info", "securityGroup", Flow)
				clusterScope.Info("### Retrieve sg info", "securityGroup", IpProtocols)
				clusterScope.Info("### Retrieve sg info", "securityGroup", IpRanges)
				clusterScope.Info("### Retrieve sg info", "securityGroup", FromPortRanges)
				clusterScope.Info("### Retrieve sg info", "securityGroup", ToPortRanges)

				if err != nil {
					return reconcile.Result{}, err
				}
				if securityGroupFromSecurityGroupRule == nil {
					clusterScope.Info("### Create securityGroupRule")
					securityGroupFromSecurityGroupRule, err = securitysvc.CreateSecurityGroupRule(securityGroupsRef.ResourceMap[securityGroupName], Flow, IpProtocol, IpRange, FromPortRange, ToPortRange)
					if err != nil {
						return reconcile.Result{}, errors.Wrapf(err, "Can not create  securityGroupRule for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
					}
				}
				securityGroupRuleRef.ResourceMap[securityGroupRuleName] = *securityGroupFromSecurityGroupRule.SecurityGroupId
				securityGroupRuleSpec.ResourceId = *securityGroupFromSecurityGroupRule.SecurityGroupId
			}
		}
	}
	return reconcile.Result{}, nil
}

// ReconcileRouteTable reconcile the RouteTable and the Route of the cluster.
func reconcileRouteTable(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	securitysvc := security.NewService(ctx, clusterScope)
	osccluster := clusterScope.OscCluster

	clusterScope.Info("Create RouteTable")
	var routeTablesSpec []*infrastructurev1beta1.OscRouteTable
	networkSpec := clusterScope.Network()
	if networkSpec.RouteTables == nil {
		networkSpec.SetRouteTableDefaultValue()
		routeTablesSpec = networkSpec.RouteTables
	} else {
		routeTablesSpec = clusterScope.RouteTables()
	}
	routeTablesRef := clusterScope.RouteTablesRef()
	routeRef := clusterScope.RouteRef()
	linkRouteTablesRef := clusterScope.LinkRouteTablesRef()

	netSpec := clusterScope.Net()
	netSpec.SetDefaultValue()
	netName := netSpec.Name + "-" + clusterScope.UID()
	netId, err := GetResourceId(netName, "net", clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	netIds := []string{netId}

	var resourceIds []string
	routeTableIds, err := securitysvc.GetRouteTableIdsFromNetIds(netIds)
	if err != nil {
		return reconcile.Result{}, err
	}
	for _, routeTableSpec := range routeTablesSpec {
		routeTableName := routeTableSpec.Name + "-" + clusterScope.UID()
		routeTableId := routeTablesRef.ResourceMap[routeTableName]
		clusterScope.Info("### Get routeTable Id ###", "routeTable", routeTablesRef.ResourceMap)
		subnetName := routeTableSpec.SubnetName + "-" + clusterScope.UID()
		subnetId, err := GetResourceId(subnetName, "subnet", clusterScope)
		if err != nil {
			return reconcile.Result{}, err
		}

		if len(routeTablesRef.ResourceMap) == 0 {
			routeTablesRef.ResourceMap = make(map[string]string)
		}
		if len(linkRouteTablesRef.ResourceMap) == 0 {
			linkRouteTablesRef.ResourceMap = make(map[string]string)
		}
		if routeTableSpec.ResourceId != "" {
			routeTablesRef.ResourceMap[routeTableName] = routeTableSpec.ResourceId
		}
		var natRouteTable bool = false
		if !contains(routeTableIds, routeTableId) {
			clusterScope.Info("check Nat RouteTable")
			routesSpec := clusterScope.Route(routeTableSpec.Name)

			for _, routeSpec := range *routesSpec {
				resourceType := routeSpec.TargetType
				if resourceType == "nat" {
					natServiceRef := clusterScope.NatServiceRef()
					clusterScope.Info("### Get Nat ###", "Nat", natServiceRef.ResourceMap)
					if len(natServiceRef.ResourceMap) == 0 {
						natRouteTable = true
					}
				}
			}
			if natRouteTable {
				continue
			}

			routeTable, err := securitysvc.CreateRouteTable(netId, routeTableName)
			if err != nil {
				return reconcile.Result{}, errors.Wrapf(err, "Can not create routetable for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
			}
			linkRouteTableId, err := securitysvc.LinkRouteTable(*routeTable.RouteTableId, subnetId)
			if err != nil {
				return reconcile.Result{}, errors.Wrapf(err, "Can not link routetable with net for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
			}
			clusterScope.Info("### Get routeTable ###", "routeTable", routeTable)
			routeTablesRef.ResourceMap[routeTableName] = *routeTable.RouteTableId
			routeTableSpec.ResourceId = *routeTable.RouteTableId
			linkRouteTablesRef.ResourceMap[routeTableName] = linkRouteTableId

			clusterScope.Info("check route")
			//			routesSpec := clusterScope.Route(routeTableSpec.Name)
			for _, routeSpec := range *routesSpec {
				resourceName := routeSpec.TargetName + "-" + clusterScope.UID()
				resourceType := routeSpec.TargetType
				routeName := routeSpec.Name + "-" + clusterScope.UID()
				if len(routeRef.ResourceMap) == 0 {
					routeRef.ResourceMap = make(map[string]string)
				}
				if routeSpec.ResourceId != "" {
					routeRef.ResourceMap[routeName] = routeSpec.ResourceId
				}

				resourceId, err := GetResourceId(resourceName, resourceType, clusterScope)
				if err != nil {
					return reconcile.Result{}, err
				}
				resourceIds = []string{resourceId}
				destinationIpRange := routeSpec.Destination
				associateRouteTableIds := []string{routeTablesRef.ResourceMap[routeTableName]}
				routeTableFromRoute, err := securitysvc.GetRouteTableFromRoute(associateRouteTableIds, resourceIds, resourceType)
				if err != nil {
					return reconcile.Result{}, err
				}
				if routeTableFromRoute == nil {
					clusterScope.Info("### Create Route ###", "Route", resourceId)
					routeTableFromRoute, err = securitysvc.CreateRoute(destinationIpRange, routeTablesRef.ResourceMap[routeTableName], resourceId, resourceType)
					if err != nil {
						return reconcile.Result{}, errors.Wrapf(err, "Can not create route for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
					}
				}
				routeRef.ResourceMap[routeName] = *routeTableFromRoute.RouteTableId
				routeSpec.ResourceId = *routeTableFromRoute.RouteTableId
			}
		}
	}
	return reconcile.Result{}, nil
}

// ReconcileNatService reconcile the NatService of the cluster.
func reconcileNatService(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	netsvc := net.NewService(ctx, clusterScope)
	osccluster := clusterScope.OscCluster

	clusterScope.Info("Create NatService")
	natServiceSpec := clusterScope.NatService()
	natServiceRef := clusterScope.NatServiceRef()
	natServiceSpec.SetDefaultValue()
	natServiceName := natServiceSpec.Name + "-" + clusterScope.UID()

	publicIpName := natServiceSpec.PublicIpName + "-" + clusterScope.UID()
	publicIpId, err := GetResourceId(publicIpName, "public-ip", clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}

	subnetName := natServiceSpec.SubnetName + "-" + clusterScope.UID()

	subnetId, err := GetResourceId(subnetName, "subnet", clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	if len(natServiceRef.ResourceMap) == 0 {
		natServiceRef.ResourceMap = make(map[string]string)
	}
	if natServiceSpec.ResourceId != "" {
		natServiceRef.ResourceMap[natServiceName] = natServiceSpec.ResourceId
	}
	var natServiceIds = []string{natServiceRef.ResourceMap[natServiceName]}
	clusterScope.Info("### Get natService Id ###", "natservice", natServiceRef.ResourceMap)
	natService, err := netsvc.GetNatService(natServiceIds)
	if err != nil {
		return reconcile.Result{}, err
	}

	if natService == nil {

		natService, err = netsvc.CreateNatService(publicIpId, subnetId, natServiceName)
		if err != nil {
			return reconcile.Result{}, errors.Wrapf(err, "Can not create natservice for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
		}
		clusterScope.Info("### Get natService ###", "natservice", natService)
		natServiceRef.ResourceMap[natServiceName] = *natService.NatServiceId
		natServiceSpec.ResourceId = *natService.NatServiceId
	}
	return reconcile.Result{}, nil
}
func (r *OscClusterReconciler) reconcile(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	clusterScope.Info("Reconcile OscCluster")
	osccluster := clusterScope.OscCluster
	controllerutil.AddFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
	if err := clusterScope.PatchObject(); err != nil {
		return reconcile.Result{}, err
	}

	duplicateResourceRouteTableErr := CheckOscDuplicateName("route-table", clusterScope)
	if duplicateResourceRouteTableErr != nil {
		return reconcile.Result{}, duplicateResourceRouteTableErr
	}

	duplicateResourceSecurityGroupErr := CheckOscDuplicateName("security-group", clusterScope)
	if duplicateResourceSecurityGroupErr != nil {
		return reconcile.Result{}, duplicateResourceSecurityGroupErr
	}

	duplicateResourceRouteErr := CheckOscDuplicateName("route", clusterScope)
	if duplicateResourceRouteErr != nil {
		return reconcile.Result{}, duplicateResourceRouteErr
	}

	duplicateResourceSecurityGroupRuleErr := CheckOscDuplicateName("security-group-rule", clusterScope)
	if duplicateResourceSecurityGroupRuleErr != nil {
		return reconcile.Result{}, duplicateResourceSecurityGroupRuleErr
	}

	duplicateResourcePublicIpErr := CheckOscDuplicateName("public-ip", clusterScope)
	if duplicateResourcePublicIpErr != nil {
		return reconcile.Result{}, duplicateResourcePublicIpErr
	}

	duplicateResourceSubnetErr := CheckOscDuplicateName("subnet", clusterScope)
	if duplicateResourceSubnetErr != nil {
		return reconcile.Result{}, duplicateResourceSubnetErr
	}

	CheckOscAssociatePublicIpErr := CheckOscAssociateResourceName("public-ip", clusterScope)
	if CheckOscAssociatePublicIpErr != nil {
		return reconcile.Result{}, CheckOscAssociatePublicIpErr
	}

	CheckOscAssociateRouteTableSubnetErr := CheckOscAssociateResourceName("routeTableSubnet", clusterScope)
	if CheckOscAssociateRouteTableSubnetErr != nil {
		return reconcile.Result{}, CheckOscAssociateRouteTableSubnetErr
	}

	CheckOscAssociateNatSubnetErr := CheckOscAssociateResourceName("natSubnet", clusterScope)
	if CheckOscAssociateNatSubnetErr != nil {
		return reconcile.Result{}, CheckOscAssociateNatSubnetErr
	}

	CheckOscAssociateLoadBalancerSubnetErr := CheckOscAssociateResourceName("loadBalancerSubnet", clusterScope)
	if CheckOscAssociateLoadBalancerSubnetErr != nil {
		return reconcile.Result{}, CheckOscAssociateLoadBalancerSubnetErr
	}

	CheckOscAssociateLoadBalancerSecurityGroupErr := CheckOscAssociateResourceName("loadBalancerSecurityGroup", clusterScope)
	if CheckOscAssociateLoadBalancerSecurityGroupErr != nil {
		return reconcile.Result{}, CheckOscAssociateLoadBalancerSecurityGroupErr
	}

	netName, err := CheckFormatParameters("net", clusterScope)
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "Can not create net %s for OscCluster %s/%s", netName, osccluster.Namespace, osccluster.Name)
	}
	subnetName, err := CheckFormatParameters("subnet", clusterScope)
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "Can not create subnet %s for OscCluster %s/%s", subnetName, osccluster.Namespace, osccluster.Name)
	}

	internetServiceName, err := CheckFormatParameters("internet-service", clusterScope)
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "Can not create internetService %s for OscCluster %s/%s", internetServiceName, osccluster.Namespace, osccluster.Name)
	}

	publicIpName, err := CheckFormatParameters("public-ip", clusterScope)
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "Can not create internetService %s for OscCluster %s/%s", publicIpName, osccluster.Namespace, osccluster.Name)
	}

	routeTableName, err := CheckFormatParameters("route-table", clusterScope)
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "Can not create routeTable %s for OscCluster %s/%s", routeTableName, osccluster.Namespace, osccluster.Name)
	}

	securityGroupName, err := CheckFormatParameters("security-group", clusterScope)
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "Can not create securityGroup %s for OscCluster %s/%s", securityGroupName, osccluster.Namespace, osccluster.Name)
	}

	routeName, err := CheckFormatParameters("route", clusterScope)
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "Can not create route %s for OscCluster %s/%s", routeName, osccluster.Namespace, osccluster.Name)
	}

	securityGroupRuleName, err := CheckFormatParameters("security-group-rule", clusterScope)
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "Can not create security group rule %s for OscCluster %s/%s", securityGroupRuleName, osccluster.Namespace, osccluster.Name)
	}
	reconcileLoadBalancerName, err := CheckFormatParameters("loadbalancer", clusterScope)
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "Can not create loadBalancer %s for OscCluster %s/%s", reconcileLoadBalancerName, osccluster.Namespace, osccluster.Name)
	}
	reconcileNet, err := reconcileNet(ctx, clusterScope)
	if err != nil {
		clusterScope.Error(err, "failed to reconcile net")
		conditions.MarkFalse(osccluster, infrastructurev1beta1.NetReadyCondition, infrastructurev1beta1.NetReconciliationFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
		return reconcileNet, err
	}
	conditions.MarkTrue(osccluster, infrastructurev1beta1.NetReadyCondition)

	reconcileSubnets, err := reconcileSubnet(ctx, clusterScope)
	if err != nil {
		clusterScope.Error(err, "failed to reconcile subnet")
		conditions.MarkFalse(osccluster, infrastructurev1beta1.SubnetsReadyCondition, infrastructurev1beta1.SubnetsReconciliationFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
		return reconcileSubnets, err
	}
	conditions.MarkTrue(osccluster, infrastructurev1beta1.SubnetsReadyCondition)

	reconcileInternetService, err := reconcileInternetService(ctx, clusterScope)
	if err != nil {
		clusterScope.Error(err, "failed to reconcile internetService")
		conditions.MarkFalse(osccluster, infrastructurev1beta1.InternetServicesReadyCondition, infrastructurev1beta1.InternetServicesFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
		return reconcileInternetService, err
	}
	conditions.MarkTrue(osccluster, infrastructurev1beta1.InternetServicesReadyCondition)

	reconcilePublicIp, err := reconcilePublicIp(ctx, clusterScope)
	if err != nil {
		clusterScope.Error(err, "failed to reconcile publicIp")
		conditions.MarkFalse(osccluster, infrastructurev1beta1.PublicIpsReadyCondition, infrastructurev1beta1.PublicIpsFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
		return reconcilePublicIp, err
	}
	conditions.MarkTrue(osccluster, infrastructurev1beta1.PublicIpsReadyCondition)

	reconcileRouteTables, err := reconcileRouteTable(ctx, clusterScope)
	if err != nil {
		clusterScope.Error(err, "failed to reconcile routeTable")
		conditions.MarkFalse(osccluster, infrastructurev1beta1.RouteTablesReadyCondition, infrastructurev1beta1.RouteTableReconciliationFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
		return reconcileRouteTables, err
	}
	conditions.MarkTrue(osccluster, infrastructurev1beta1.RouteTablesReadyCondition)

	reconcileSecurityGroups, err := reconcileSecurityGroup(ctx, clusterScope)
	if err != nil {
		clusterScope.Error(err, "failed to reconcile securityGroup")
		conditions.MarkFalse(osccluster, infrastructurev1beta1.SecurityGroupReadyCondition, infrastructurev1beta1.SecurityGroupReconciliationFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
		return reconcileSecurityGroups, err
	}
	conditions.MarkTrue(osccluster, infrastructurev1beta1.SecurityGroupReadyCondition)

	reconcileLoadBalancer, err := reconcileLoadBalancer(ctx, clusterScope)
	if err != nil {
		clusterScope.Error(err, "failed to reconcile load balancer")
		conditions.MarkFalse(osccluster, infrastructurev1beta1.LoadBalancerReadyCondition, infrastructurev1beta1.LoadBalancerFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
		return reconcileLoadBalancer, err
	}
	conditions.MarkTrue(osccluster, infrastructurev1beta1.LoadBalancerReadyCondition)

	reconcileNatService, err := reconcileNatService(ctx, clusterScope)
	if err != nil {
		clusterScope.Error(err, "failed to reconcile natservice")
		conditions.MarkFalse(osccluster, infrastructurev1beta1.NatServicesReadyCondition, infrastructurev1beta1.NatServicesReconciliationFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
		return reconcileNatService, nil
	}
	conditions.MarkTrue(osccluster, infrastructurev1beta1.NatServicesReadyCondition)

	reconcileNatRouteTable, err := reconcileRouteTable(ctx, clusterScope)
	if err != nil {
		clusterScope.Error(err, "failed to reconcile NatRouteTable")
		conditions.MarkFalse(osccluster, infrastructurev1beta1.RouteTablesReadyCondition, infrastructurev1beta1.RouteTableReconciliationFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
		return reconcileNatRouteTable, err
	}
	conditions.MarkTrue(osccluster, infrastructurev1beta1.RouteTablesReadyCondition)

	clusterScope.Info("Set OscCluster status to ready")
	clusterScope.SetReady()
	return reconcile.Result{}, nil
}

// ReconcileDeleteLoadBalancer reconcile the destruction of the LoadBalancer of the cluster.

func reconcileDeleteLoadBalancer(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	osccluster := clusterScope.OscCluster
	servicesvc := service.NewService(ctx, clusterScope)

	clusterScope.Info("Delete LoadBalancer")
	loadBalancerSpec := clusterScope.LoadBalancer()
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
		return reconcile.Result{}, errors.Wrapf(err, "Can not delete load balancer for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
	}
	clusterScope.Info("Wait LoadBalancer Delete")
	time.Sleep(15 * time.Second)
	return reconcile.Result{RequeueAfter: 15 * time.Second}, nil
}

// ReconcileDeleteNatService reconcile the destruction of the NatService of the cluster.
func reconcileDeleteNatService(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	osccluster := clusterScope.OscCluster
	netsvc := net.NewService(ctx, clusterScope)

	clusterScope.Info("Delete natService")
	natServiceSpec := clusterScope.NatService()
	natServiceSpec.SetDefaultValue()
	natServiceRef := clusterScope.NatServiceRef()
	natServiceName := natServiceSpec.Name + "-" + clusterScope.UID()
	var natServiceIds = []string{natServiceRef.ResourceMap[natServiceName]}
	natservice, err := netsvc.GetNatService(natServiceIds)
	if err != nil {
		return reconcile.Result{}, err
	}
	if natservice == nil {
		controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
		return reconcile.Result{}, nil
	}
	err = netsvc.DeleteNatService(natServiceRef.ResourceMap[natServiceName])
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "Can not delete natService for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
	}
	return reconcile.Result{}, err
}

// ReconcileDeletePublicIp reconcile the destruction of the PublicIp of the cluster.
func reconcileDeletePublicIp(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	securitysvc := security.NewService(ctx, clusterScope)
	osccluster := clusterScope.OscCluster

	clusterScope.Info("Delete PublicIp")
	var publicIpsSpec []*infrastructurev1beta1.OscPublicIp
	networkSpec := clusterScope.Network()
	if networkSpec.PublicIps == nil {
		networkSpec.SetPublicIpDefaultValue()
		publicIpsSpec = networkSpec.PublicIps
	} else {
		publicIpsSpec = clusterScope.PublicIp()
	}
	publicIpRef := clusterScope.PublicIpRef()
	var publicIpsId []string
	for _, publicIpSpec := range publicIpsSpec {
		publicIpName := publicIpSpec.Name + "-" + clusterScope.UID()
		publicIpId := publicIpRef.ResourceMap[publicIpName]
		publicIpsId = append(publicIpsId, publicIpId)
	}
	publicIpIds, err := securitysvc.ValidatePublicIpIds(publicIpsId)
	if err != nil {
		return reconcile.Result{}, err
	}
	clusterScope.Info("### Check Id  ###", "publicip", publicIpIds)
	for _, publicIpSpec := range publicIpsSpec {
		publicIpName := publicIpSpec.Name + "-" + clusterScope.UID()
		publicIpId := publicIpRef.ResourceMap[publicIpName]
		if !contains(publicIpIds, publicIpId) {
			controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
			return reconcile.Result{}, nil
		}
		clusterScope.Info("Remove publicip")
		err = securitysvc.DeletePublicIp(publicIpRef.ResourceMap[publicIpName])
		if err != nil {
			return reconcile.Result{}, errors.Wrapf(err, "Can not delete publicIp for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
		}

	}
	return reconcile.Result{}, nil
}

func reconcileDeleteSecurityGroup(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	osccluster := clusterScope.OscCluster
	securitysvc := security.NewService(ctx, clusterScope)

	clusterScope.Info("Delete SecurityGroup")
	var securityGroupsSpec []*infrastructurev1beta1.OscSecurityGroup
	networkSpec := clusterScope.Network()
	if networkSpec.SecurityGroups == nil {
		networkSpec.SetSecurityGroupDefaultValue()
		securityGroupsSpec = networkSpec.SecurityGroups
	} else {
		securityGroupsSpec = clusterScope.SecurityGroups()
	}
	securityGroupsRef := clusterScope.SecurityGroupsRef()

	netSpec := clusterScope.Net()
	netSpec.SetDefaultValue()
	netName := netSpec.Name + "-" + clusterScope.UID()
	netId, err := GetResourceId(netName, "net", clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	netIds := []string{netId}
	clusterScope.Info("### Get net Id ###", "net", netIds)
	securityGroupIds, err := securitysvc.GetSecurityGroupIdsFromNetIds(netIds)
	if err != nil {
		return reconcile.Result{}, err
	}
	for _, securityGroupSpec := range securityGroupsSpec {
		securityGroupName := securityGroupSpec.Name + "-" + clusterScope.UID()
		securityGroupId := securityGroupsRef.ResourceMap[securityGroupName]
		if !contains(securityGroupIds, securityGroupId) {
			controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
			return reconcile.Result{}, nil
		}
		clusterScope.Info("Remove securityGroupRule")
		securityGroupRulesSpec := clusterScope.SecurityGroupRule(securityGroupSpec.Name)
		for _, securityGroupRuleSpec := range *securityGroupRulesSpec {
			Flow := securityGroupRuleSpec.Flow
			IpProtocol := securityGroupRuleSpec.IpProtocol
			IpProtocols := []string{IpProtocol}
			IpRange := securityGroupRuleSpec.IpRange
			IpRanges := []string{IpRange}
			FromPortRange := securityGroupRuleSpec.FromPortRange
			FromPortRanges := []int32{FromPortRange}
			ToPortRange := securityGroupRuleSpec.ToPortRange
			ToPortRanges := []int32{ToPortRange}
			associateSecurityGroupIds := []string{securityGroupsRef.ResourceMap[securityGroupName]}
			clusterScope.Info("Delete SecurityGroupRule")
			securityGroupFromSecurityGroupRule, err := securitysvc.GetSecurityGroupFromSecurityGroupRule(associateSecurityGroupIds, Flow, IpProtocols, IpRanges, FromPortRanges, ToPortRanges)
			if err != nil {
				return reconcile.Result{}, err
			}
			if securityGroupFromSecurityGroupRule == nil {
				controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
				return reconcile.Result{}, nil
			}
			err = securitysvc.DeleteSecurityGroupRule(securityGroupsRef.ResourceMap[securityGroupName], Flow, IpProtocol, IpRange, FromPortRange, ToPortRange)
			if err != nil {
				return reconcile.Result{}, errors.Wrapf(err, "Can not delete securityGroupRule for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
			}
		}
		clusterScope.Info("Delete SecurityGroup")
		err = securitysvc.DeleteSecurityGroup(securityGroupsRef.ResourceMap[securityGroupName])
		if err != nil {
			return reconcile.Result{}, errors.Wrapf(err, "Can not delete securityGroup  for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
		}
	}
	return reconcile.Result{}, nil
}

// ReconcileDeleteRouteTable reconcile the destruction of the RouteTable of the cluster.
func reconcileDeleteRouteTable(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	osccluster := clusterScope.OscCluster
	securitysvc := security.NewService(ctx, clusterScope)

	clusterScope.Info("Delete RouteTable")
	var routeTablesSpec []*infrastructurev1beta1.OscRouteTable
	networkSpec := clusterScope.Network()
	if networkSpec.RouteTables == nil {
		networkSpec.SetRouteTableDefaultValue()
		routeTablesSpec = networkSpec.RouteTables
	} else {
		routeTablesSpec = clusterScope.RouteTables()
	}
	routeTablesRef := clusterScope.RouteTablesRef()
	linkRouteTablesRef := clusterScope.LinkRouteTablesRef()

	netSpec := clusterScope.Net()
	netSpec.SetDefaultValue()
	netName := netSpec.Name + "-" + clusterScope.UID()
	netId, err := GetResourceId(netName, "net", clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	netIds := []string{netId}

	var resourceIds []string
	routeTableIds, err := securitysvc.GetRouteTableIdsFromNetIds(netIds)
	if err != nil {
		return reconcile.Result{}, err
	}
	for _, routeTableSpec := range routeTablesSpec {
		routeTableSpec.SetDefaultValue()
		routeTableName := routeTableSpec.Name + "-" + clusterScope.UID()
		routeTableId := routeTablesRef.ResourceMap[routeTableName]
		clusterScope.Info("### delete routeTable Id ###", "routeTable", routeTableId)

		if !contains(routeTableIds, routeTableId) {
			controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
			return reconcile.Result{}, nil
		}
		clusterScope.Info("Remove Route")
		routesSpec := clusterScope.Route(routeTableSpec.Name)
		for _, routeSpec := range *routesSpec {
			resourceName := routeSpec.TargetName + "-" + clusterScope.UID()
			resourceType := routeSpec.TargetType
			routeName := routeSpec.Name + "-" + clusterScope.UID()
			resourceId, err := GetResourceId(resourceName, resourceType, clusterScope)
			if err != nil {
				return reconcile.Result{}, err
			}
			routeTableId, err := GetResourceId(routeName, "route", clusterScope)
			if err != nil {
				return reconcile.Result{}, err
			}
			resourceIds = []string{resourceId}
			destinationIpRange := routeSpec.Destination
			associateRouteTableIds := []string{routeTablesRef.ResourceMap[routeTableName]}
			routeTableFromRoute, err := securitysvc.GetRouteTableFromRoute(associateRouteTableIds, resourceIds, resourceType)
			if err != nil {
				return reconcile.Result{}, err
			}
			if routeTableFromRoute == nil {
				controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
				return reconcile.Result{}, nil
			}
			clusterScope.Info("Delete Route")
			clusterScope.Info("### delete destinationIpRange###", "routeTable", destinationIpRange)

			err = securitysvc.DeleteRoute(destinationIpRange, routeTableId)
			if err != nil {
				return reconcile.Result{}, errors.Wrapf(err, "Can not delete route for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
			}
		}
		clusterScope.Info("Unlink Routetable")

		err = securitysvc.UnlinkRouteTable(linkRouteTablesRef.ResourceMap[routeTableName])
		if err != nil {
			return reconcile.Result{}, errors.Wrapf(err, "Can not delete routeTable for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
		}
		clusterScope.Info("Delete RouteTable")

		err = securitysvc.DeleteRouteTable(routeTablesRef.ResourceMap[routeTableName])
		if err != nil {
			return reconcile.Result{}, errors.Wrapf(err, "Can not delete internetService for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
		}
	}
	return reconcile.Result{}, nil
}

// ReconcileDeleteInternetService reconcile the destruction of the InternetService of the cluster.
func reconcileDeleteInternetService(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	osccluster := clusterScope.OscCluster
	netsvc := net.NewService(ctx, clusterScope)

	clusterScope.Info("Delete internetService")

	internetServiceSpec := clusterScope.InternetService()
	internetServiceSpec.SetDefaultValue()
	internetServiceRef := clusterScope.InternetServiceRef()
	internetServiceName := internetServiceSpec.Name + "-" + clusterScope.UID()

	netSpec := clusterScope.Net()
	netSpec.SetDefaultValue()
	netName := netSpec.Name + "-" + clusterScope.UID()

	netId, err := GetResourceId(netName, "net", clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}

	var internetServiceIds = []string{internetServiceRef.ResourceMap[internetServiceName]}
	internetservice, err := netsvc.GetInternetService(internetServiceIds)
	if err != nil {
		return reconcile.Result{}, err
	}
	if internetservice == nil {
		controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
		return reconcile.Result{}, nil
	}
	err = netsvc.UnlinkInternetService(internetServiceRef.ResourceMap[internetServiceName], netId)
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "Can not unlink internetService and net for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
	}
	err = netsvc.DeleteInternetService(internetServiceRef.ResourceMap[internetServiceName])
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "Can not delete internetService for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
	}
	return reconcile.Result{}, nil
}

// ReconcileDeleteSubnet reconcile the destruction of the Subnet of the cluster.
func reconcileDeleteSubnet(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	osccluster := clusterScope.OscCluster
	netsvc := net.NewService(ctx, clusterScope)

	clusterScope.Info("Delete subnet")

	var subnetsSpec []*infrastructurev1beta1.OscSubnet
	networkSpec := clusterScope.Network()
	if networkSpec.Subnets == nil {
		networkSpec.SetSubnetDefaultValue()
		subnetsSpec = networkSpec.Subnets
	} else {
		subnetsSpec = clusterScope.Subnet()
	}
	netSpec := clusterScope.Net()
	netSpec.SetDefaultValue()
	netName := netSpec.Name + "-" + clusterScope.UID()

	netId, err := GetResourceId(netName, "net", clusterScope)
	netIds := []string{netId}
	if err != nil {
		return reconcile.Result{}, err
	}
	subnetRef := clusterScope.SubnetRef()
	subnetIds, err := netsvc.GetSubnetIdsFromNetIds(netIds)
	if err != nil {
		return reconcile.Result{}, err
	}
	for _, subnetSpec := range subnetsSpec {
		subnetName := subnetSpec.Name + "-" + clusterScope.UID()
		subnetId := subnetRef.ResourceMap[subnetName]
		if !contains(subnetIds, subnetId) {
			controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
			return reconcile.Result{}, nil
		}
		err = netsvc.DeleteSubnet(subnetRef.ResourceMap[subnetName])
		if err != nil {
			return reconcile.Result{}, errors.Wrapf(err, "Can not delete subnet for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
		}
	}
	return reconcile.Result{}, nil
}

// ReconcileDeleteNet reconcile the destruction of the Net of the cluster.
func reconcileDeleteNet(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	osccluster := clusterScope.OscCluster
	netsvc := net.NewService(ctx, clusterScope)

	netSpec := clusterScope.Net()
	netSpec.SetDefaultValue()
	netRef := clusterScope.NetRef()
	netName := netSpec.Name + "-" + clusterScope.UID()
	var netIds = []string{netRef.ResourceMap[netName]}

	clusterScope.Info("Delete net")
	net, err := netsvc.GetNet(netIds)
	if err != nil {
		return reconcile.Result{}, err
	}
	if net == nil {
		controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
		return reconcile.Result{}, nil
	}
	err = netsvc.DeleteNet(netRef.ResourceMap[netName])
	if err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "Can not delete net for Osccluster %s/%s", osccluster.Namespace, osccluster.Name)
	}
	return reconcile.Result{}, nil
}

func (r *OscClusterReconciler) reconcileDelete(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	clusterScope.Info("Reconcile OscCluster")
	osccluster := clusterScope.OscCluster

	reconcileDeleteLoadBalancer, err := reconcileDeleteLoadBalancer(ctx, clusterScope)
	if err != nil {
		return reconcileDeleteLoadBalancer, err
	}

	reconcileDeleteNatService, err := reconcileDeleteNatService(ctx, clusterScope)
	if err != nil {
		return reconcileDeleteNatService, err
	}

	reconcileDeletePublicIp, err := reconcileDeletePublicIp(ctx, clusterScope)
	if err != nil {
		return reconcileDeletePublicIp, err
	}

	reconcileDeleteRouteTable, err := reconcileDeleteRouteTable(ctx, clusterScope)
	if err != nil {
		return reconcileDeleteRouteTable, err
	}

	reconcileDeleteSecurityGroup, err := reconcileDeleteSecurityGroup(ctx, clusterScope)
	if err != nil {
		return reconcileDeleteSecurityGroup, err
	}

	reconcileDeleteInternetService, err := reconcileDeleteInternetService(ctx, clusterScope)
	if err != nil {
		return reconcileDeleteInternetService, err
	}

	reconcileDeleteSubnet, err := reconcileDeleteSubnet(ctx, clusterScope)
	if err != nil {
		return reconcileDeleteSubnet, err
	}
	reconcileDeleteNet, err := reconcileDeleteNet(ctx, clusterScope)
	if err != nil {
		return reconcileDeleteNet, err
	}

	controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OscClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1beta1.OscCluster{}).
		Complete(r)
}
