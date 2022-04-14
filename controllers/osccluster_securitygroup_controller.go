package controllers

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
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
		securityGroupRef := clusterScope.SecurityGroupsRef()
		if securityGroupId, ok := securityGroupRef.ResourceMap[resourceName]; ok {
			return securityGroupId, nil
		} else {
			return "", fmt.Errorf("%s is not exist", resourceName)
		}
}

// GetResourceId return the resourceId from the resourceMap base on resourceName (tag name + cluster object uid) and resourceType (net, subnet, gateway, route, route-table, public-ip)
func GetSecurityGroupRulesResourceId(resourceName string, clusterScope *scope.ClusterScope) (string, error) {
                securityGroupRuleRef := clusterScope.SecurityGroupRuleRef()
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
		return nil
}

// CheckOscDuplicateName check that there are not the same name for resource with the same kind (route-table, subnet, ..).
func CheckSecurityGroupRuleOscDuplicateName(clusterScope *scope.ClusterScope) error {
                clusterScope.Info("check unique security group rule")
         	var resourceNameList []string
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
                return nil
}

// CheckFormatParameters check every resource (net, subnet, ...) parameters format (Tag format, cidr format, ..)
func CheckSecurityGroupFormatParameters( clusterScope *scope.ClusterScope) (string, error) {
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
		return "", nil
}


// CheckFormatParameters check every resource (net, subnet, ...) parameters format (Tag format, cidr format, ..)
func CheckSecurityGroupRuleFormatParameters( clusterScope *scope.ClusterScope) (string, error) {
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
                return "", nil
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
	netId, err := GetNetResourceId(netName, clusterScope)
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
	netId, err := GetNetResourceId(netName, clusterScope)
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
