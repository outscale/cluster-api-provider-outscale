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
	"io"
	"strings"
	"time"

	"github.com/Jeffail/gabs"
	"github.com/benbjohnson/clock"
	infrastructurev1beta2 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta2"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/security"
	tag "github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag"
	osc "github.com/outscale/osc-sdk-go/v2"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// getSecurityGroupResourceId return the SecurityGroupId from the resourceMap base on SecurityGroupName (tag name + cluster object uid)
func getSecurityGroupResourceId(resourceName string, clusterScope *scope.ClusterScope) (string, error) {
	securityGroupRef := clusterScope.GetSecurityGroupsRef()
	if securityGroupId, ok := securityGroupRef.ResourceMap[resourceName]; ok {
		return securityGroupId, nil
	} else {
		return "", fmt.Errorf("%s does not exist", resourceName)
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
	clusterScope.V(2).Info("Check unique security group rule")
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
	var securityGroupsSpec []*infrastructurev1beta2.OscSecurityGroup
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
		_, err = infrastructurev1beta2.ValidateDescription(securityGroupDescription)
		if err != nil {
			return securityGroupTagName, err
		}
	}
	return "", nil
}

// checkFormatParameters check every securityGroupRule parameters format (Tag format, cidr format, ..)
func checkSecurityGroupRuleFormatParameters(clusterScope *scope.ClusterScope) (string, error) {
	var securityGroupsSpec []*infrastructurev1beta2.OscSecurityGroup
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
			_, err = infrastructurev1beta2.ValidateFlow(securityGroupRuleFlow)
			if err != nil {
				return securityGroupRuleTagName, err
			}
			securityGroupRuleIpProtocol := securityGroupRuleSpec.IpProtocol
			_, err = infrastructurev1beta2.ValidateIpProtocol(securityGroupRuleIpProtocol)
			if err != nil {
				return securityGroupRuleTagName, err
			}
			securityGroupRuleIpRange := securityGroupRuleSpec.IpRange
			_, err = infrastructurev1beta2.ValidateCidr(securityGroupRuleIpRange)
			if err != nil {
				return securityGroupRuleTagName, err
			}
			securityGroupRuleFromPortRange := securityGroupRuleSpec.FromPortRange
			_, err = infrastructurev1beta2.ValidatePort(securityGroupRuleFromPortRange)
			if err != nil {
				return securityGroupRuleTagName, err
			}
			securityGroupRuleToPortRange := securityGroupRuleSpec.ToPortRange
			_, err = infrastructurev1beta2.ValidatePort(securityGroupRuleToPortRange)
			if err != nil {
				return securityGroupRuleTagName, err
			}

		}
	}
	return "", nil
}

// reconcileSecurityGroupRule reconcile the securityGroupRule of the cluster.
func reconcileSecurityGroupRule(ctx context.Context, clusterScope *scope.ClusterScope, securityGroupRuleSpec infrastructurev1beta2.OscSecurityGroupRule, securityGroupName string, securityGroupSvc security.OscSecurityGroupInterface) (reconcile.Result, error) {
	//osccluster := clusterScope.OscCluster

	securityGroupsRef := clusterScope.GetSecurityGroupsRef()
	securityGroupRuleRef := clusterScope.GetSecurityGroupRuleRef()

	securityGroupRuleName := securityGroupRuleSpec.Name + "-" + clusterScope.GetUID()
	if len(securityGroupRuleRef.ResourceMap) == 0 {
		securityGroupRuleRef.ResourceMap = make(map[string]string)
	}
	Flow := securityGroupRuleSpec.Flow
	IpProtocol := securityGroupRuleSpec.IpProtocol
	IpRange := securityGroupRuleSpec.IpRange
	FromPortRange := securityGroupRuleSpec.FromPortRange
	ToPortRange := securityGroupRuleSpec.ToPortRange
	associateSecurityGroupId := securityGroupsRef.ResourceMap[securityGroupName]
	clusterScope.V(4).Info("Get associateSecurityGroupId", "securityGroup", associateSecurityGroupId)
	clusterScope.V(4).Info("Check if the desired securityGroupRule exist", "securityGroupRuleName", securityGroupRuleName)
	securityGroupFromSecurityGroupRule, err := securityGroupSvc.GetSecurityGroupFromSecurityGroupRule(associateSecurityGroupId, Flow, IpProtocol, IpRange, "", FromPortRange, ToPortRange)
	if err != nil {
		return reconcile.Result{}, err
	}
	if securityGroupFromSecurityGroupRule == nil {
		clusterScope.V(4).Info("Create the desired securityGroupRule", "securityGroupRuleName", securityGroupRuleName)
		securityGroupFromSecurityGroupRule, err = securityGroupSvc.CreateSecurityGroupRule(associateSecurityGroupId, Flow, IpProtocol, IpRange, "", FromPortRange, ToPortRange)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not create securityGroupRule for OscCluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
		}

	}
	securityGroupRuleRef.ResourceMap[securityGroupRuleName] = securityGroupFromSecurityGroupRule.GetSecurityGroupId()
	return reconcile.Result{}, nil
}

// deleteSecurityGroup reconcile the deletion of securityGroup of the cluster.
func deleteSecurityGroup(ctx context.Context, clusterScope *scope.ClusterScope, securityGroupId string, securityGroupSvc security.OscSecurityGroupInterface, clock_time clock.Clock) (reconcile.Result, error) {

	currentTimeout := clock_time.Now().Add(time.Second * 600)
	var loadbalancer_delete = false
	for !loadbalancer_delete {
		err, httpRes := securityGroupSvc.DeleteSecurityGroup(securityGroupId)
		if err != nil {
			time.Sleep(20 * time.Second)
			buffer := new(strings.Builder)
			_, err := io.Copy(buffer, httpRes.Body)
			if err != nil {
				return reconcile.Result{}, nil
			}
			httpResBody := buffer.String()
			clusterScope.V(4).Info("Find body", "httpResBody", httpResBody)
			httpResBodyData := []byte(httpResBody)
			httpResBodyParsed, err := gabs.ParseJSON(httpResBodyData)

			if err != nil {
				return reconcile.Result{}, fmt.Errorf("%w Can not delete securityGroup for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
			}
			httpResCode := strings.Replace(strings.Replace(fmt.Sprintf("%v", httpResBodyParsed.Path("Errors.Code").Data()), "[", "", 1), "]", "", 1)
			httpResType := strings.Replace(strings.Replace(fmt.Sprintf("%v", httpResBodyParsed.Path("Errors.Type").Data()), "[", "", 1), "]", "", 1)
			var unexpectedErr bool = true

			if httpResCode == "9085" && httpResType == "ResourceConflict" {
				clusterScope.V(2).Info("LoadBalancer is not deleting yet")
				unexpectedErr = false
			}
			if unexpectedErr {
				return reconcile.Result{}, fmt.Errorf(" Can not delete securityGroup because of the uncatch error for Osccluster %s/%s", clusterScope.GetNamespace(), clusterScope.GetName())
			}
			clusterScope.V(2).Info("Wait until loadBalancer is deleting")
		} else {
			loadbalancer_delete = true
		}

		if clock_time.Now().After(currentTimeout) {
			return reconcile.Result{}, fmt.Errorf("%w Can not delete securityGroup because to waiting loadbalancer to be delete timeout  for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
		}

	}
	return reconcile.Result{}, nil
}

// reconcileSecurityGroup reconcile the securityGroup of the cluster.
func reconcileSecurityGroup(ctx context.Context, clusterScope *scope.ClusterScope, securityGroupSvc security.OscSecurityGroupInterface, tagSvc tag.OscTagInterface) (reconcile.Result, error) {
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
	clusterScope.V(4).Info("Get securityGroup Id", "securityGroup", securityGroupIds)
	if err != nil {
		return reconcile.Result{}, err
	}
	securityGroupsRef := clusterScope.GetSecurityGroupsRef()
	clusterScope.V(4).Info("Number of securityGroup", "securityGroupLength", len(securityGroupsSpec))
	for _, securityGroupSpec := range securityGroupsSpec {
		securityGroupName := securityGroupSpec.Name + "-" + clusterScope.GetUID()
		clusterScope.V(2).Info("Check if the desired securityGroup exist in net", "securityGroupName", securityGroupName)
		securityGroupDescription := securityGroupSpec.Description

		tagKey := "Name"
		tagValue := securityGroupName
		tag, err := tagSvc.ReadTag(tagKey, tagValue)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not get tag for OscCluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
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
				return reconcile.Result{}, fmt.Errorf("%w Can not create securityGroup for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
			}
			securityGroupsRef.ResourceMap[securityGroupName] = *securityGroup.SecurityGroupId
			securityGroupSpec.ResourceId = *securityGroup.SecurityGroupId

			clusterScope.V(2).Info("Check securityGroupRule")
			securityGroupRulesSpec := clusterScope.GetSecurityGroupRule(securityGroupSpec.Name)
			clusterScope.V(4).Info("Number of securityGroupRule", "securityGroupRuleLength", len(*securityGroupRulesSpec))
			for _, securityGroupRuleSpec := range *securityGroupRulesSpec {
				clusterScope.V(4).Info("Create securityGroupRule for the desired securityGroup", "securityGroupName", securityGroupName)
				_, err = reconcileSecurityGroupRule(ctx, clusterScope, securityGroupRuleSpec, securityGroupName, securityGroupSvc)
				if err != nil {
					return reconcile.Result{}, err
				}
			}
		}
	}
	return reconcile.Result{}, nil
}

// ReconcileRoute reconcile the RouteTable and the Route of the cluster.
func reconcileDeleteSecurityGroupRule(ctx context.Context, clusterScope *scope.ClusterScope, securityGroupRuleSpec infrastructurev1beta2.OscSecurityGroupRule, securityGroupName string, securityGroupSvc security.OscSecurityGroupInterface) (reconcile.Result, error) {
	osccluster := clusterScope.OscCluster
	securityGroupsRef := clusterScope.GetSecurityGroupsRef()

	securityGroupRuleName := securityGroupRuleSpec.Name + "-" + clusterScope.GetUID()

	Flow := securityGroupRuleSpec.Flow
	IpProtocol := securityGroupRuleSpec.IpProtocol
	IpRange := securityGroupRuleSpec.IpRange
	FromPortRange := securityGroupRuleSpec.FromPortRange
	ToPortRange := securityGroupRuleSpec.ToPortRange
	associateSecurityGroupId := securityGroupsRef.ResourceMap[securityGroupName]
	clusterScope.V(4).Info("Check if the desired securityGroupRule exist", "securityGroupRuleName", securityGroupRuleName)
	securityGroupFromSecurityGroupRule, err := securityGroupSvc.GetSecurityGroupFromSecurityGroupRule(associateSecurityGroupId, Flow, IpProtocol, IpRange, "", FromPortRange, ToPortRange)
	if err != nil {
		return reconcile.Result{}, err
	}

	if securityGroupFromSecurityGroupRule == nil {
		clusterScope.V(2).Info("The desired securityGroupRule does not exist anymore", "securityGroupRuleName", securityGroupRuleName)
		controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
		return reconcile.Result{}, nil
	}
	clusterScope.V(2).Info("Delete the desired securityGroupRule", "securityGroupRuleName", securityGroupRuleName)
	err = securityGroupSvc.DeleteSecurityGroupRule(associateSecurityGroupId, Flow, IpProtocol, IpRange, "", FromPortRange, ToPortRange)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("%s Can not delete securityGroupRule for OscCluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
	}
	return reconcile.Result{}, nil
}

// reconcileDeleteSecurityGroup reconcile the deletetion of securityGroup of the cluster.
func reconcileDeleteSecurityGroup(ctx context.Context, clusterScope *scope.ClusterScope, securityGroupSvc security.OscSecurityGroupInterface) (reconcile.Result, error) {
	osccluster := clusterScope.OscCluster

	var securityGroupsSpec []*infrastructurev1beta2.OscSecurityGroup
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
	clock_time := clock.New()
	clusterScope.V(4).Info("Number of securitGroup", "securityGroupLength", len(securityGroupsSpec))
	for _, securityGroupSpec := range securityGroupsSpec {
		securityGroupName := securityGroupSpec.Name + "-" + clusterScope.GetUID()
		securityGroupId := securityGroupsRef.ResourceMap[securityGroupName]
		if !Contains(securityGroupIds, securityGroupId) {
			clusterScope.V(2).Info("The desired securityGroup does not exist anymore", "securityGroupName", securityGroupName)
			controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
			return reconcile.Result{}, nil
		}
		securityGroupRulesSpec := clusterScope.GetSecurityGroupRule(securityGroupSpec.Name)
		clusterScope.V(4).Info("Number of securityGroupRule", "securityGroupLength", len(*securityGroupRulesSpec))
		for _, securityGroupRuleSpec := range *securityGroupRulesSpec {
			_, err = reconcileDeleteSecurityGroupRule(ctx, clusterScope, securityGroupRuleSpec, securityGroupName, securityGroupSvc)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		clusterScope.V(2).Info("Delete the desired securityGroup", "securityGroupName", securityGroupName)
		_, err := deleteSecurityGroup(ctx, clusterScope, securityGroupsRef.ResourceMap[securityGroupName], securityGroupSvc, clock_time)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not delete securityGroup  for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
		}
	}
	return reconcile.Result{}, nil
}
