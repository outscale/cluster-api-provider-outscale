package controllers

import (
	"context"
	"fmt"

	infrastructurev1beta1 "github.com/outscale-vbr/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/services/net"
	"github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/services/security"
	"github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/services/service"
	tag "github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/tag"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// GetResourceId return the resourceId from the resourceMap base on resourceName (tag name + cluster object uid) and resourceType (net, subnet, gateway, route, route-table, public-ip)
func GetSecurityGroupResourceId(resourceName string, clusterScope *scope.ClusterScope) (string, error) {
	securityGroupRef := clusterScope.GetSecurityGroupsRef()
	if securityGroupId, ok := securityGroupRef.ResourceMap[resourceName]; ok {
		return securityGroupId, nil
	} else {
		return "", fmt.Errorf("%s is not exist", resourceName)
	}
}

// GetResourceId return the resourceId from the resourceMap base on resourceName (tag name + cluster object uid) and resourceType (net, subnet, gateway, route, route-table, public-ip)
func GetSecurityGroupRulesResourceId(resourceName string, clusterScope *scope.ClusterScope) (string, error) {
	securityGroupRuleRef := clusterScope.GetSecurityGroupRuleRef()
	if securityGroupRuleId, ok := securityGroupRuleRef.ResourceMap[resourceName]; ok {
		return securityGroupRuleId, nil
	} else {
		return "", fmt.Errorf("%s is not exist", resourceName)
	}
}

// CheckOscDuplicateName check that there are not the same name for resource with the same kind (route-table, subnet, ..).
func CheckSecurityGroupOscDuplicateName(clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	clusterScope.Info("check unique security group rule")
	securityGroupsSpec := clusterScope.GetSecurityGroups()
	for _, securityGroupSpec := range securityGroupsSpec {
		securityGroupRulesSpec := clusterScope.GetSecurityGroupRule(securityGroupSpec.Name)
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
	return nil
}

// CheckOscDuplicateName check that there are not the same name for resource with the same kind (route-table, subnet, ..).
func CheckSecurityGroupRuleOscDuplicateName(clusterScope *scope.ClusterScope) error {
	clusterScope.Info("check unique security group rule")
	var resourceNameList []string
	securityGroupsSpec := clusterScope.GetSecurityGroups()
	for _, securityGroupSpec := range securityGroupsSpec {
		securityGroupRulesSpec := clusterScope.GetSecurityGroupRule(securityGroupSpec.Name)
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
	return nil
}

// CheckFormatParameters check every resource (net, subnet, ...) parameters format (Tag format, cidr format, ..)
func CheckSecurityGroupFormatParameters(clusterScope *scope.ClusterScope) (string, error) {
	clusterScope.Info("Check security group parameters")
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
	return "", nil
}

// CheckFormatParameters check every resource (net, subnet, ...) parameters format (Tag format, cidr format, ..)
func CheckSecurityGroupRuleFormatParameters(clusterScope *scope.ClusterScope) (string, error) {
	clusterScope.Info("Check security Group rule parameters")
	var securityGroupsSpec []*infrastructurev1beta1.OscSecurityGroup
	networkSpec := clusterScope.GetNetwork()
	if networkSpec.SecurityGroups == nil {
		networkSpec.SetSecurityGroupDefaultValue()
		securityGroupsSpec = networkSpec.SecurityGroups
	} else {
		securityGroupsSpec = clusterScope.GetSecurityGroups()
	}
	for _, securityGroupSpec := range securityGroupsSpec {
		securityGroupRulesSpec := clusterScope.GetSecurityGroupRule(securityGroupSpec.Name)
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
	return "", nil
}

func reconcileSecurityGroupRule(ctx context.Context, clusterScope *scope.ClusterScope, securityGroupRuleSpec infrastructurev1beta1.OscSecurityGroupRule, securityGroupName string) (reconcile.Result, error) {
	securitysvc := security.NewService(ctx, clusterScope)
	osccluster := clusterScope.OscCluster

	securityGroupsRef := clusterScope.GetSecurityGroupsRef()
	securityGroupRuleRef := clusterScope.GetSecurityGroupRuleRef()

	securityGroupRuleName := securityGroupRuleSpec.Name + "-" + clusterScope.UID()
	if len(securityGroupRuleRef.ResourceMap) == 0 {
		securityGroupRuleRef.ResourceMap = make(map[string]string)
	}
	if securityGroupRuleSpec.ResourceId != "" {
		securityGroupRuleRef.ResourceMap[securityGroupRuleName] = securityGroupRuleSpec.ResourceId
	}
	Flow := securityGroupRuleSpec.Flow
	IpProtocol := securityGroupRuleSpec.IpProtocol
	IpRange := securityGroupRuleSpec.IpRange
	FromPortRange := securityGroupRuleSpec.FromPortRange
	ToPortRange := securityGroupRuleSpec.ToPortRange
	associateSecurityGroupId := securityGroupsRef.ResourceMap[securityGroupName]

	securityGroupFromSecurityGroupRule, err := securitysvc.GetSecurityGroupFromSecurityGroupRule(associateSecurityGroupId, Flow, IpProtocol, IpRange, FromPortRange, ToPortRange)

	if err != nil {
		return reconcile.Result{}, err
	}
	if securityGroupFromSecurityGroupRule == nil {
		clusterScope.Info("### Create securityGroupRule")
		securityGroupFromSecurityGroupRule, err = securitysvc.CreateSecurityGroupRule(securityGroupsRef.ResourceMap[securityGroupName], Flow, IpProtocol, IpRange, FromPortRange, ToPortRange)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w: Can not create  securityGroupRule for Osccluster %s/%s", err, osccluster.Namespace, osccluster.Name)
		}
	}
	securityGroupRuleRef.ResourceMap[securityGroupRuleName] = *securityGroupFromSecurityGroupRule.SecurityGroupId
	securityGroupRuleSpec.ResourceId = *securityGroupFromSecurityGroupRule.SecurityGroupId
	return reconcile.Result{}, nil
}

func reconcileSecurityGroup(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	securitysvc := security.NewService(ctx, clusterScope)
	osccluster := clusterScope.OscCluster

	clusterScope.Info("Create SecurityGroup")
	var securityGroupsSpec []*infrastructurev1beta1.OscSecurityGroup
	networkSpec := clusterScope.GetNetwork()
	if networkSpec.SecurityGroups == nil {
		networkSpec.SetSecurityGroupDefaultValue()
		securityGroupsSpec = networkSpec.SecurityGroups
	} else {
		securityGroupsSpec = clusterScope.GetSecurityGroups()
	}

	netSpec := clusterScope.GetNet()
	netSpec.SetDefaultValue()
	netName := netSpec.Name + "-" + clusterScope.UID()
	netId, err := GetNetResourceId(netName, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}

	securityGroupIds, err := securitysvc.GetSecurityGroupIdsFromNetIds(netId)
	if err != nil {
		return reconcile.Result{}, err
	}
	securityGroupsRef := clusterScope.GetSecurityGroupsRef()
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
				return reconcile.Result{}, fmt.Errorf("%w: Can not create securitygroup for Osccluster %s/%s", err, osccluster.Namespace, osccluster.Name)
			}
			clusterScope.Info("### Get securityGroup", "securityGroup", securityGroup)
			securityGroupsRef.ResourceMap[securityGroupName] = *securityGroup.SecurityGroupId
			securityGroupSpec.ResourceId = *securityGroup.SecurityGroupId

			clusterScope.Info("check securityGroupRule")
			securityGroupRulesSpec := clusterScope.GetSecurityGroupRule(securityGroupSpec.Name)
			for _, securityGroupRuleSpec := range *securityGroupRulesSpec {
				_, err = reconcileSecurityGroupRule(ctx, clusterScope, securityGroupRuleSpec, securityGroupName)
				if err != nil {
					return reconcile.Result{}, err
				}
			}
		}
	}
	return reconcile.Result{}, nil
}

// ReconcileRoute reconcile the RouteTable and the Route of the cluster.
func reconcileDeleteSecurityGroupRule(ctx context.Context, clusterScope *scope.ClusterScope, securityGroupRuleSpec infrastructurev1beta1.OscSecurityGroupRule, securityGroupName string) (reconcile.Result, error) {
	securitysvc := security.NewService(ctx, clusterScope)
	osccluster := clusterScope.OscCluster
	securityGroupsRef := clusterScope.GetSecurityGroupsRef()

	Flow := securityGroupRuleSpec.Flow
	IpProtocol := securityGroupRuleSpec.IpProtocol
	IpRange := securityGroupRuleSpec.IpRange
	FromPortRange := securityGroupRuleSpec.FromPortRange
	ToPortRange := securityGroupRuleSpec.ToPortRange
	associateSecurityGroupId := securityGroupsRef.ResourceMap[securityGroupName]
	clusterScope.Info("Delete SecurityGroupRule")
	securityGroupFromSecurityGroupRule, err := securitysvc.GetSecurityGroupFromSecurityGroupRule(associateSecurityGroupId, Flow, IpProtocol, IpRange, FromPortRange, ToPortRange)
	if err != nil {
		return reconcile.Result{}, err
	}
	if securityGroupFromSecurityGroupRule == nil {
		controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
		return reconcile.Result{}, nil
	}
	err = securitysvc.DeleteSecurityGroupRule(securityGroupsRef.ResourceMap[securityGroupName], Flow, IpProtocol, IpRange, FromPortRange, ToPortRange)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("%w: Can not delete securityGroupRule for Osccluster %s/%s", err, osccluster.Namespace, osccluster.Name)
	}
	return reconcile.Result{}, nil
}

func reconcileDeleteSecurityGroup(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	osccluster := clusterScope.OscCluster
	securitysvc := security.NewService(ctx, clusterScope)

	clusterScope.Info("Delete SecurityGroup")
	var securityGroupsSpec []*infrastructurev1beta1.OscSecurityGroup
	networkSpec := clusterScope.GetNetwork()
	if networkSpec.SecurityGroups == nil {
		networkSpec.SetSecurityGroupDefaultValue()
		securityGroupsSpec = networkSpec.SecurityGroups
	} else {
		securityGroupsSpec = clusterScope.GetSecurityGroups()
	}
	securityGroupsRef := clusterScope.GetSecurityGroupsRef()

	netSpec := clusterScope.GetNet()
	netSpec.SetDefaultValue()
	netName := netSpec.Name + "-" + clusterScope.UID()
	netId, err := GetNetResourceId(netName, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	securityGroupIds, err := securitysvc.GetSecurityGroupIdsFromNetIds(netId)
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
		securityGroupRulesSpec := clusterScope.GetSecurityGroupRule(securityGroupSpec.Name)
		for _, securityGroupRuleSpec := range *securityGroupRulesSpec {
			_, err = reconcileDeleteSecurityGroupRule(ctx, clusterScope, securityGroupRuleSpec, securityGroupName)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		clusterScope.Info("Delete SecurityGroup")
		err = securitysvc.DeleteSecurityGroup(securityGroupsRef.ResourceMap[securityGroupName])
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w: Can not delete securityGroup  for Osccluster %s/%s", err, osccluster.Namespace, osccluster.Name)
		}
	}
	return reconcile.Result{}, nil
}
