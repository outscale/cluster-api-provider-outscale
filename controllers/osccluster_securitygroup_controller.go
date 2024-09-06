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
	"time"

	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/security"
	tag "github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag"
	osc "github.com/outscale/osc-sdk-go/v2"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// getSecurityGroupResourceId return the SecurityGroupId from the resourceMap base on SecurityGroupName (tag name + cluster object uid)
func getSecurityGroupResourceId(resourceName string, clusterScope *scope.ClusterScope) (string, error) {
	securityGroupRef := clusterScope.GetSecurityGroupsRef()
	if securityGroupId, ok := securityGroupRef.ResourceMap[resourceName]; ok {
		return securityGroupId, nil
	} else {
		return "", fmt.Errorf("%s does not exist (yet)", resourceName)
	}
}

// getSecurityGroupRulesResourceId return the SecurityGroupId from the resourceMap base on SecurityGroupRuleName (tag name + cluster object uid)
func getSecurityGroupRulesResourceId(resourceName string, clusterScope *scope.ClusterScope) (string, error) {
	securityGroupRuleRef := clusterScope.GetSecurityGroupRuleRef()
	if securityGroupRuleId, ok := securityGroupRuleRef.ResourceMap[resourceName]; ok {
		return securityGroupRuleId, nil
	} else {
		return "", fmt.Errorf("%s does not exist", resourceName)
	}
}

// checkSecurityGroupOscDuplicateName check that there are not the same name for securityGroup
func checkSecurityGroupOscDuplicateName(clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	securityGroupsSpec := clusterScope.GetSecurityGroups()
	for _, securityGroupSpec := range securityGroupsSpec {
		resourceNameList = append(resourceNameList, securityGroupSpec.Name)
	}
	clusterScope.V(2).Info("Check unique security group")
	duplicateResourceErr := alertDuplicate(resourceNameList)
	if duplicateResourceErr != nil {
		return duplicateResourceErr
	} else {
		return nil
	}
}

// checkSecurityGroupRuleOscDuplicateName check that there are not the same name for securityGroupRule
func checkSecurityGroupRuleOscDuplicateName(clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	securityGroupsSpec := clusterScope.GetSecurityGroups()
	for _, securityGroupSpec := range securityGroupsSpec {
		securityGroupRulesSpec := clusterScope.GetSecurityGroupRule(securityGroupSpec.Name)
		for _, securityGroupRuleSpec := range *securityGroupRulesSpec {
			resourceNameList = append(resourceNameList, securityGroupRuleSpec.Name)
		}
		clusterScope.V(2).Info("Check unique security group rule")
		duplicateResourceErr := alertDuplicate(resourceNameList)
		if duplicateResourceErr != nil {
			return duplicateResourceErr
		} else {
			return nil
		}
	}
	return nil
}

// checkSecurityGroupFormatParameters check securityGroup parameters format (Tag format, cidr format, ..)
func checkSecurityGroupFormatParameters(clusterScope *scope.ClusterScope) (string, error) {
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
		clusterScope.V(2).Info("Check security group parameters")
		securityGroupTagName, err := tag.ValidateTagNameValue(securityGroupName)
		if err != nil {
			return securityGroupTagName, err
		}
		securityGroupDescription := securityGroupSpec.Description
		_, err = infrastructurev1beta1.ValidateDescription(securityGroupDescription)
		if err != nil {
			return securityGroupTagName, err
		}
	}
	return "", nil
}

// checkFormatParameters check every securityGroupRule parameters format (Tag format, cidr format, ..)
func checkSecurityGroupRuleFormatParameters(clusterScope *scope.ClusterScope) (string, error) {
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
			securityGroupRuleName := securityGroupRuleSpec.Name + "-" + clusterScope.GetUID()
			clusterScope.V(2).Info("Check security Group rule parameters")
			securityGroupRuleTagName, err := tag.ValidateTagNameValue(securityGroupRuleName)
			if err != nil {
				return securityGroupRuleTagName, err
			}
			securityGroupRuleFlow := securityGroupRuleSpec.Flow
			_, err = infrastructurev1beta1.ValidateFlow(securityGroupRuleFlow)
			if err != nil {
				return securityGroupRuleTagName, err
			}
			securityGroupRuleIpProtocol := securityGroupRuleSpec.IpProtocol
			_, err = infrastructurev1beta1.ValidateIpProtocol(securityGroupRuleIpProtocol)
			if err != nil {
				return securityGroupRuleTagName, err
			}
			securityGroupRuleIpRange := securityGroupRuleSpec.IpRange
			securityGroupTargetSecurityGroupName := securityGroupRuleSpec.TargetSecurityGroupName
			if securityGroupRuleIpRange == "" && securityGroupTargetSecurityGroupName == "" {
				return securityGroupRuleTagName, fmt.Errorf("ipRange or targetSecurityGroupName must be set")
			}
			if securityGroupRuleIpRange != "" {
				_, err = infrastructurev1beta1.ValidateCidr(securityGroupRuleIpRange)
				if err != nil {
					return securityGroupRuleTagName, err
				}
			}

			securityGroupRuleFromPortRange := securityGroupRuleSpec.FromPortRange
			_, err = infrastructurev1beta1.ValidatePort(securityGroupRuleFromPortRange)
			if err != nil {
				return securityGroupRuleTagName, err
			}
			securityGroupRuleToPortRange := securityGroupRuleSpec.ToPortRange
			_, err = infrastructurev1beta1.ValidatePort(securityGroupRuleToPortRange)
			if err != nil {
				return securityGroupRuleTagName, err
			}

		}
	}
	return "", nil
}

// deleteSecurityGroup reconcile the deletion of securityGroup of the cluster.
func deleteSecurityGroup(ctx context.Context, clusterScope *scope.ClusterScope, securityGroupId string, securityGroupSvc security.OscSecurityGroupInterface) (reconcile.Result, error) {
	err := securityGroupSvc.DeleteSecurityGroup(securityGroupId)
	if err != nil {
		return reconcile.Result{RequeueAfter: 30 * time.Second}, fmt.Errorf("cannot delete securityGroup for Osccluster %s/%s", clusterScope.GetNamespace(), clusterScope.GetName())
	}
	return reconcile.Result{}, nil
}

// reconcileSecurityGroup reconcile the securityGroup of the cluster.
func reconcileSecurityGroup(ctx context.Context, clusterScope *scope.ClusterScope, securityGroupSvc security.OscSecurityGroupInterface, tagSvc tag.OscTagInterface) (reconcile.Result, error) {
	clusterScope.V(4).Info("Reconciling security groups for OscCluster")
	securityGroupsSpec := clusterScope.GetSecurityGroups()

	netSpec := clusterScope.GetNet()
	netSpec.SetDefaultValue()
	netName := netSpec.Name + "-" + clusterScope.GetUID()
	netId, err := getNetResourceId(netName, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	networkSpec := clusterScope.GetNetwork()
	clusterName := networkSpec.ClusterName + "-" + clusterScope.GetUID()

	clusterScope.V(4).Info("Get list of all desired securitygroup in net", "netId", netId)
	securityGroupIds, err := securityGroupSvc.GetSecurityGroupIdsFromNetIds(netId)
	clusterScope.V(4).Info("Get securityGroup Ids", "securityGroups", securityGroupIds)
	if err != nil {
		return reconcile.Result{}, err
	}
	securityGroupsRef := clusterScope.GetSecurityGroupsRef()
	clusterScope.V(4).Info("Number of securityGroup", "securityGroupLength", len(securityGroupsSpec))
	for _, securityGroupSpec := range securityGroupsSpec {
		securityGroupName := securityGroupSpec.Name + "-" + clusterScope.GetUID()
		clusterScope.V(2).Info("Check if the desired securityGroup exists in net", "securityGroupName", securityGroupName)
		securityGroupDescription := securityGroupSpec.Description
		deleteDefaultOutboundRule := securityGroupSpec.DeleteDefaultOutboundRule

		tagKey := "Name"
		tagValue := securityGroupName
		tag, err := tagSvc.ReadTag(tagKey, tagValue)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w cannot get tag for OscCluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
		}
		securityGroupTag := securityGroupSpec.Tag
		if len(securityGroupsRef.ResourceMap) == 0 {
			securityGroupsRef.ResourceMap = make(map[string]string)
		}

		if securityGroupSpec.ResourceId != "" {
			securityGroupsRef.ResourceMap[securityGroupName] = securityGroupSpec.ResourceId
		}
		_, resourceMapExist := securityGroupsRef.ResourceMap[securityGroupName]
		if resourceMapExist {
			securityGroupSpec.ResourceId = securityGroupsRef.ResourceMap[securityGroupName]
		}
		var securityGroup *osc.SecurityGroup
		securityGroupId := securityGroupsRef.ResourceMap[securityGroupName]

		if !Contains(securityGroupIds, securityGroupId) && tag == nil {
			clusterScope.V(2).Info("Create the desired securitygroup", "securityGroupName", securityGroupName)
			if securityGroupTag == "OscK8sMainSG" {
				securityGroup, err = securityGroupSvc.CreateSecurityGroup(netId, clusterName, securityGroupName, securityGroupDescription, "OscK8sMainSG")
			} else {
				securityGroup, err = securityGroupSvc.CreateSecurityGroup(netId, clusterName, securityGroupName, securityGroupDescription, "")
			}
			clusterScope.V(4).Info("Get securityGroup", "securityGroup", securityGroup)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("%w cannot create securityGroup for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
			}
			securityGroupsRef.ResourceMap[securityGroupName] = *securityGroup.SecurityGroupId
			securityGroupSpec.ResourceId = *securityGroup.SecurityGroupId

			if deleteDefaultOutboundRule {
				clusterScope.V(2).Info("Delete default outbound rule for sg", "securityGroupName", securityGroupSpec.Name)
				err = securityGroupSvc.DeleteSecurityGroupRule(*securityGroup.SecurityGroupId, "Outbound", "-1", "0.0.0.0/0", "", 0, 0)
				if err != nil {
					return reconcile.Result{}, fmt.Errorf("%w Cannot delete default Outbound rule for sg %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
				}
			}
		}
	}

	for securityGroupName, securityGroupId := range securityGroupsRef.ResourceMap {
		if !Contains(securityGroupIds, securityGroupId) {
			clusterScope.V(4).Info("Deleting securityGroup and associated securityGroupRules", "securityGroupName", securityGroupName)

			securityGroupRulesSpec := clusterScope.GetSecurityGroupRule(securityGroupName)
			clusterScope.V(4).Info("Deleting securityGroupRules", "securityGroupName", securityGroupName)
			for _, securityGroupRuleSpec := range *securityGroupRulesSpec {
				reconcileDeleteSecurityGroupsRule, err := reconcileDeleteSecurityGroupsRule(ctx, clusterScope, securityGroupRuleSpec, securityGroupName, securityGroupSvc)
				if err != nil {
					return reconcileDeleteSecurityGroupsRule, err
				}
			}
			reconcileDeleteSecurityGroup, err := reconcileDeleteSecurityGroup(ctx, clusterScope, securityGroupId, securityGroupSvc)
			if err != nil {
				return reconcileDeleteSecurityGroup, err
			}
		}
	}
	return reconcile.Result{}, nil
}

// reconcileSecurityGroupRule reconcile the securityGroupRules of the cluster.
func reconcileSecurityGroupRule(ctx context.Context, clusterScope *scope.ClusterScope, securityGroupSvc security.OscSecurityGroupInterface, tagSvc tag.OscTagInterface) (reconcile.Result, error) {
	clusterScope.V(4).Info("Reconciling security group rules")
	securityGroupsSpec := clusterScope.GetSecurityGroups()

	securityGroupsRef := clusterScope.GetSecurityGroupsRef()
	if len(securityGroupsRef.ResourceMap) == 0 {
		return reconcile.Result{}, fmt.Errorf("securityGroupsRef.ResourceMap is empty, security groups should be reconciled first")
	}

	securityGroupRuleRef := clusterScope.GetSecurityGroupRuleRef()
	if len(securityGroupRuleRef.ResourceMap) == 0 {
		securityGroupRuleRef.ResourceMap = make(map[string]string)
	}

	for _, securityGroupSpec := range securityGroupsSpec {
		securityGroupName := securityGroupSpec.Name + "-" + clusterScope.GetUID()
		clusterScope.V(2).Info("Reconciling security group rules for securityGroup", "securityGroupName", securityGroupName)
		securityGroupRulesSpec := clusterScope.GetSecurityGroupRule(securityGroupSpec.Name)
		clusterScope.V(4).Info("Number of securityGroupRules", "securityGroupRuleLength", len(*securityGroupRulesSpec))
		var securityGroupRuleNames []string
		for _, securityGroupRuleSpec := range *securityGroupRulesSpec {
			securityGroupRuleName := securityGroupRuleSpec.Name + "-" + clusterScope.GetUID()
			securityGroupRuleNames = append(securityGroupRuleNames, securityGroupRuleName)
			clusterScope.V(4).Info("Reconciling securityGroupRule for the desired securityGroup", "securityGroupName", securityGroupName, "securityGroupRuleName", securityGroupRuleName)

			Flow := securityGroupRuleSpec.Flow
			IpProtocol := securityGroupRuleSpec.IpProtocol
			IpRange := securityGroupRuleSpec.IpRange
			FromPortRange := securityGroupRuleSpec.FromPortRange
			ToPortRange := securityGroupRuleSpec.ToPortRange
			associateSecurityGroupId := securityGroupsRef.ResourceMap[securityGroupName]

			targetSecurityGroupId := ""
			if securityGroupRuleSpec.TargetSecurityGroupName != "" {
				targetSecurityGroupName := securityGroupRuleSpec.TargetSecurityGroupName + "-" + clusterScope.GetUID()
				targetSecurityGroupId = securityGroupsRef.ResourceMap[targetSecurityGroupName]
				clusterScope.V(4).Info("Get targetSecurityGroupId", "securityGroup", targetSecurityGroupId)
				if targetSecurityGroupId == "" {
					return reconcile.Result{RequeueAfter: 30 * time.Second}, fmt.Errorf("the target securityGroup %s does not exist (yet) for OscCluster %s/%s", targetSecurityGroupName, clusterScope.GetNamespace(), clusterScope.GetName())
				}
			}

			// The GetSecurityGroupFromSecurityGroupRule function does not work for Rules containing a targetSecurityGroupId, for now we just try to create and ignore if it already exists (error 409)
			// clusterScope.V(4).Info("Check if the desired securityGroupRule exist", "securityGroupRuleName", securityGroupRuleName)
			// securityGroupFromSecurityGroupRule, err := securityGroupSvc.GetSecurityGroupFromSecurityGroupRule(associateSecurityGroupId, Flow, IpProtocol, IpRange, targetSecurityGroupId, FromPortRange, ToPortRange)
			// if err != nil {
			// 	return reconcile.Result{}, err
			// }
			// if securityGroupFromSecurityGroupRule == nil {
			// 	clusterScope.V(4).Info("Create the desired securityGroupRule", "securityGroupRuleName", securityGroupRuleName)
			// 	securityGroupFromSecurityGroupRule, err = securityGroupSvc.CreateSecurityGroupRule(associateSecurityGroupId, Flow, IpProtocol, IpRange, targetSecurityGroupId, FromPortRange, ToPortRange)
			// 	if err != nil {
			// 		return reconcile.Result{}, fmt.Errorf("%w cannot create securityGroupRule for OscCluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
			// 	}
			// }
			clusterScope.V(4).Info("Create the desired securityGroupRule", "securityGroupRuleName", securityGroupRuleName)
			securityGroupFromSecurityGroupRule, err := securityGroupSvc.CreateSecurityGroupRule(associateSecurityGroupId, Flow, IpProtocol, IpRange, targetSecurityGroupId, FromPortRange, ToPortRange)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("%w cannot create securityGroupRule for OscCluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
			}
			securityGroupRuleRef.ResourceMap[securityGroupRuleName] = securityGroupFromSecurityGroupRule.GetSecurityGroupId()
		}

		// for securityGroupRuleName, _ := range securityGroupRuleRef.ResourceMap {
		// 	if !Contains(securityGroupRuleNames, securityGroupRuleName) {
		// 		clusterScope.V(4).Info("Deleting securityGroupRule", "securityGroupRuleName", securityGroupRuleName)
		// 		clusterScope.V(2).Info("Deleting individual sg rules after they have been deleted from the spec is not supported yet")
		// We cannot delete securityGroupRules here while we should as we require the securityGroupRuleSpec, but that is already deleted from the osc cluster spec
		// reconcileDeleteSecurityGroupsRule, err := reconcileDeleteSecurityGroupsRule(ctx, clusterScope, securityGroupRuleSpec, securityGroupName, securityGroupSvc)
		// if err != nil {
		// 	return reconcileDeleteSecurityGroupsRule, err
		// }
		// 	}
		// }
	}

	return reconcile.Result{}, nil
}

// ReconcileRoute reconcile the RouteTable and the Route of the cluster.
func reconcileDeleteSecurityGroupsRule(ctx context.Context, clusterScope *scope.ClusterScope, securityGroupRuleSpec infrastructurev1beta1.OscSecurityGroupRule, securityGroupName string, securityGroupSvc security.OscSecurityGroupInterface) (reconcile.Result, error) {
	securityGroupsRef := clusterScope.GetSecurityGroupsRef()

	securityGroupRuleName := securityGroupRuleSpec.Name + "-" + clusterScope.GetUID()

	Flow := securityGroupRuleSpec.Flow
	IpProtocol := securityGroupRuleSpec.IpProtocol
	IpRange := securityGroupRuleSpec.IpRange
	FromPortRange := securityGroupRuleSpec.FromPortRange
	ToPortRange := securityGroupRuleSpec.ToPortRange
	associateSecurityGroupId := securityGroupsRef.ResourceMap[securityGroupName]
	targetSecurityGroupName := securityGroupRuleSpec.TargetSecurityGroupName
	targetSecurityGroupId := ""
	if targetSecurityGroupName != "" {
		targetSecurityGroupId = securityGroupsRef.ResourceMap[targetSecurityGroupName]
	}

	clusterScope.V(4).Info("Check if the desired securityGroupRule exist", "securityGroupRuleName", securityGroupRuleName)
	securityGroupFromSecurityGroupRule, err := securityGroupSvc.GetSecurityGroupFromSecurityGroupRule(associateSecurityGroupId, Flow, IpProtocol, IpRange, targetSecurityGroupId, FromPortRange, ToPortRange)
	if err != nil {
		return reconcile.Result{}, err
	}

	if securityGroupFromSecurityGroupRule == nil {
		clusterScope.V(2).Info("The desired securityGroupRule does not exist anymore", "securityGroupRuleName", securityGroupRuleName)
		return reconcile.Result{}, nil
	}
	clusterScope.V(2).Info("Delete the desired securityGroupRule", "securityGroupRuleName", securityGroupRuleName)
	err = securityGroupSvc.DeleteSecurityGroupRule(associateSecurityGroupId, Flow, IpProtocol, IpRange, targetSecurityGroupId, FromPortRange, ToPortRange)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("%s cannot delete securityGroupRule for OscCluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
	}
	return reconcile.Result{}, nil
}

// ReconcileRoute reconcile the RouteTable and the Route of the cluster.
func reconcileDeleteSecurityGroup(ctx context.Context, clusterScope *scope.ClusterScope, securityGroupId string, securityGroupSvc security.OscSecurityGroupInterface) (reconcile.Result, error) {
	securityGroupsRef := clusterScope.GetSecurityGroupsRef()

	clusterScope.V(4).Info("Check if the securityGroup exists", "securityGroupId", securityGroupId)
	securityGroup, err := securityGroupSvc.GetSecurityGroup(securityGroupId)
	if err != nil {
		return reconcile.Result{}, err
	}

	if securityGroup == nil {
		clusterScope.V(4).Info("The desired securityGroup does not exist anymore", "securityGroupId", securityGroupId)
		return reconcile.Result{}, nil
	}
	clusterScope.V(4).Info("Delete the desired securityGroup", "securityGroupId", securityGroupId)
	err = securityGroupSvc.DeleteSecurityGroup(securityGroupId)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("%s cannot delete securityGroup for OscCluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
	}
	clusterScope.V(4).Info("Deleted the desired securityGroup", "securityGroupId", securityGroupId)
	delete(securityGroupsRef.ResourceMap, *securityGroup.SecurityGroupName)
	return reconcile.Result{}, nil
}

// reconcileDeleteSecurityGroups reconcile the deletetion of securityGroup of the cluster.
func reconcileDeleteSecurityGroups(ctx context.Context, clusterScope *scope.ClusterScope, securityGroupSvc security.OscSecurityGroupInterface) (reconcile.Result, error) {
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
	netName := netSpec.Name + "-" + clusterScope.GetUID()
	netId, err := getNetResourceId(netName, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	securityGroupIds, err := securityGroupSvc.GetSecurityGroupIdsFromNetIds(netId)
	if err != nil {
		return reconcile.Result{}, err
	}
	clusterScope.V(4).Info("Number of securitGroup", "securityGroupLength", len(securityGroupsSpec))
	for _, securityGroupSpec := range securityGroupsSpec {
		securityGroupName := securityGroupSpec.Name + "-" + clusterScope.GetUID()
		securityGroupId := securityGroupsRef.ResourceMap[securityGroupName]
		if !Contains(securityGroupIds, securityGroupId) {
			clusterScope.V(2).Info("The desired securityGroup does not exist anymore", "securityGroupName", securityGroupName)
			return reconcile.Result{}, nil
		}
		securityGroupRulesSpec := clusterScope.GetSecurityGroupRule(securityGroupSpec.Name)
		clusterScope.V(4).Info("Number of securityGroupRule", "securityGroupLength", len(*securityGroupRulesSpec))
		for _, securityGroupRuleSpec := range *securityGroupRulesSpec {
			reconcileDeleteSecurityGroupsRule, err := reconcileDeleteSecurityGroupsRule(ctx, clusterScope, securityGroupRuleSpec, securityGroupName, securityGroupSvc)
			if err != nil {
				return reconcileDeleteSecurityGroupsRule, err
			}
		}
		clusterScope.V(2).Info("Delete the desired securityGroup", "securityGroupName", securityGroupName)
		reconcileDeleteSecurityGroup, err := reconcileDeleteSecurityGroup(ctx, clusterScope, securityGroupId, securityGroupSvc)
		if err != nil {
			return reconcileDeleteSecurityGroup, err
		}
	}
	return reconcile.Result{}, nil
}
