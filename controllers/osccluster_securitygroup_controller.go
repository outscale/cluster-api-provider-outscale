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
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/benbjohnson/clock"
	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/security"
	tag "github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	osc "github.com/outscale/osc-sdk-go/v2"
	ctrl "sigs.k8s.io/controller-runtime"
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
			_, err = infrastructurev1beta1.ValidateCidr(securityGroupRuleIpRange)
			if err != nil {
				return securityGroupRuleTagName, err
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

// reconcileSecurityGroupRule reconcile the securityGroupRule of the cluster.
func reconcileSecurityGroupRule(ctx context.Context, clusterScope *scope.ClusterScope, securityGroupRuleSpec infrastructurev1beta1.OscSecurityGroupRule, securityGroupName string, securityGroupSvc security.OscSecurityGroupInterface) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
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
	log.V(4).Info("Checking securityGroupRule", "securityGroup", associateSecurityGroupId, "securityGroupRuleName", securityGroupRuleName)
	hasRule, err := securityGroupSvc.SecurityGroupHasRule(ctx, associateSecurityGroupId, Flow, IpProtocol, IpRange, "", FromPortRange, ToPortRange)
	if err != nil {
		return reconcile.Result{}, err
	}
	if !hasRule {
		log.V(2).Info("Creating securityGroupRule", "securityGroupRuleName", securityGroupRuleName)
		sg, err := securityGroupSvc.CreateSecurityGroupRule(ctx, associateSecurityGroupId, Flow, IpProtocol, IpRange, "", FromPortRange, ToPortRange)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot create securityGroupRule: %w", err)
		}
		securityGroupRuleRef.ResourceMap[securityGroupRuleName] = sg.GetSecurityGroupId()
	}
	return reconcile.Result{}, nil
}

// deleteSecurityGroup reconcile the deletion of securityGroup of the cluster.
func deleteSecurityGroup(ctx context.Context, clusterScope *scope.ClusterScope, securityGroupId string, securityGroupSvc security.OscSecurityGroupInterface, clock_time clock.Clock) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	currentTimeout := clock_time.Now().Add(time.Second * 600)
	for {
		err := securityGroupSvc.DeleteSecurityGroup(ctx, securityGroupId)
		if err == nil {
			return reconcile.Result{}, nil
		}
		if !errors.Is(err, security.ErrResourceConflict) {
			return reconcile.Result{}, fmt.Errorf("cannot delete securityGroup: %w", err)
		}
		log.V(2).Info("LoadBalancer is not deleted yet")

		clock_time.Sleep(20 * time.Second)
		if clock_time.Now().After(currentTimeout) {
			return reconcile.Result{}, fmt.Errorf("timeout trying to delete securityGroup: %w", err)
		}
	}
}

// reconcileSecurityGroup reconcile the securityGroup of the cluster.
func reconcileSecurityGroup(ctx context.Context, clusterScope *scope.ClusterScope, securityGroupSvc security.OscSecurityGroupInterface, tagSvc tag.OscTagInterface) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
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
	extraSecurityGroupRule := clusterScope.GetExtraSecurityGroupRule()

	log.V(4).Info("List all securitygroups in net", "netId", netId)
	securityGroupIds, err := securityGroupSvc.GetSecurityGroupIdsFromNetIds(ctx, netId)
	log.V(4).Info("Get securityGroup Id", "securityGroup", securityGroupIds)
	if err != nil {
		return reconcile.Result{}, err
	}
	securityGroupsRef := clusterScope.GetSecurityGroupsRef()
	log.V(4).Info("Number of securityGroup", "securityGroupLength", len(securityGroupsSpec))
	for _, securityGroupSpec := range securityGroupsSpec {
		securityGroupName := securityGroupSpec.Name + "-" + clusterScope.GetUID()
		log.V(4).Info("Checking securityGroup", "securityGroupName", securityGroupName)
		securityGroupDescription := securityGroupSpec.Description

		tagKey := "Name"
		tagValue := securityGroupName
		tag, err := tagSvc.ReadTag(ctx, tagKey, tagValue)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot get tag: %w", err)
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

		if !slices.Contains(securityGroupIds, securityGroupId) && tag == nil {
			if extraSecurityGroupRule && (len(securityGroupsRef.ResourceMap) == len(securityGroupsSpec)) {
				log.V(4).Info("Extra Security Group Rule activated")
			} else {
				log.V(2).Info("Creating securitygroup", "securityGroupName", securityGroupName)
				if securityGroupTag == "OscK8sMainSG" {
					securityGroup, err = securityGroupSvc.CreateSecurityGroup(ctx, netId, clusterName, securityGroupName, securityGroupDescription, "OscK8sMainSG")
				} else {
					securityGroup, err = securityGroupSvc.CreateSecurityGroup(ctx, netId, clusterName, securityGroupName, securityGroupDescription, "")
				}
				if err != nil {
					return reconcile.Result{}, fmt.Errorf("cannot create securityGroup: %w", err)
				}
				securityGroupsRef.ResourceMap[securityGroupName] = *securityGroup.SecurityGroupId
				securityGroupSpec.ResourceId = *securityGroup.SecurityGroupId
			}

			log.V(3).Info("Checking securityGroupRules")
			securityGroupRulesSpec := clusterScope.GetSecurityGroupRule(securityGroupSpec.Name)
			log.V(4).Info("Number of securityGroupRule", "securityGroupRuleLength", len(*securityGroupRulesSpec))
			for _, securityGroupRuleSpec := range *securityGroupRulesSpec {
				log.V(4).Info("Create securityGroupRule for securityGroup", "securityGroupName", securityGroupName)
				_, err = reconcileSecurityGroupRule(ctx, clusterScope, securityGroupRuleSpec, securityGroupName, securityGroupSvc)
				if err != nil {
					return reconcile.Result{}, err
				}
			}
		}

		if slices.Contains(securityGroupIds, securityGroupId) && extraSecurityGroupRule {
			log.V(4).Info("Extra Security Group Rule activated")
			securityGroupRulesSpec := clusterScope.GetSecurityGroupRule(securityGroupSpec.Name)
			log.V(4).Info("Number of securityGroupRule", "securityGroupRuleLength", len(*securityGroupRulesSpec))
			for _, securityGroupRuleSpec := range *securityGroupRulesSpec {
				log.V(4).Info("Get sgrule", "sgRuleName", securityGroupRuleSpec.Name)
				log.V(4).Info("Create securityGroupRule for securityGroup", "securityGroupName", securityGroupName)
				_, err = reconcileSecurityGroupRule(ctx, clusterScope, securityGroupRuleSpec, securityGroupName, securityGroupSvc)
				if err != nil {
					return reconcile.Result{}, err
				}
			}
		}
	}

	clusterScope.SetExtraSecurityGroupRule(false)
	return reconcile.Result{}, nil
}

// ReconcileRoute reconcile the RouteTable and the Route of the cluster.
func reconcileDeleteSecurityGroupRule(ctx context.Context, clusterScope *scope.ClusterScope, securityGroupRuleSpec infrastructurev1beta1.OscSecurityGroupRule, securityGroupName string, securityGroupSvc security.OscSecurityGroupInterface) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	osccluster := clusterScope.OscCluster
	securityGroupsRef := clusterScope.GetSecurityGroupsRef()

	securityGroupRuleName := securityGroupRuleSpec.Name + "-" + clusterScope.GetUID()

	Flow := securityGroupRuleSpec.Flow
	IpProtocol := securityGroupRuleSpec.IpProtocol
	IpRange := securityGroupRuleSpec.IpRange
	FromPortRange := securityGroupRuleSpec.FromPortRange
	ToPortRange := securityGroupRuleSpec.ToPortRange
	associateSecurityGroupId := securityGroupsRef.ResourceMap[securityGroupName]
	log.V(4).Info("Check if securityGroupRule exist", "securityGroupRuleName", securityGroupRuleName)
	hasRule, err := securityGroupSvc.SecurityGroupHasRule(ctx, associateSecurityGroupId, Flow, IpProtocol, IpRange, "", FromPortRange, ToPortRange)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !hasRule {
		log.V(2).Info("securityGroupRule does not exist anymore", "securityGroupRuleName", securityGroupRuleName)
		controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
		return reconcile.Result{}, nil
	}
	log.V(2).Info("Deleting securityGroupRule", "securityGroupRuleName", securityGroupRuleName)
	err = securityGroupSvc.DeleteSecurityGroupRule(ctx, associateSecurityGroupId, Flow, IpProtocol, IpRange, "", FromPortRange, ToPortRange)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot delete securityGroupRule: %w", err)
	}
	return reconcile.Result{}, nil
}

// reconcileDeleteSecurityGroup reconcile the deletetion of securityGroup of the cluster.
func reconcileDeleteSecurityGroup(ctx context.Context, clusterScope *scope.ClusterScope, securityGroupSvc security.OscSecurityGroupInterface) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

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
		log.V(3).Info("No net found, skipping security group deletion")
		return reconcile.Result{}, nil //nolint: nilerr
	}
	securityGroupIds, err := securityGroupSvc.GetSecurityGroupIdsFromNetIds(ctx, netId)
	if err != nil {
		return reconcile.Result{}, err
	}
	clock_time := clock.New()
	log.V(4).Info("Number of securitGroup", "securityGroupLength", len(securityGroupsSpec))
	for _, securityGroupSpec := range securityGroupsSpec {
		securityGroupName := securityGroupSpec.Name + "-" + clusterScope.GetUID()
		securityGroupId := securityGroupsRef.ResourceMap[securityGroupName]
		if !slices.Contains(securityGroupIds, securityGroupId) {
			log.V(2).Info("securityGroup does not exist anymore", "securityGroupName", securityGroupName)
			return reconcile.Result{}, nil
		}
		securityGroupRulesSpec := clusterScope.GetSecurityGroupRule(securityGroupSpec.Name)
		log.V(4).Info("Number of securityGroupRule", "securityGroupLength", len(*securityGroupRulesSpec))
		for _, securityGroupRuleSpec := range *securityGroupRulesSpec {
			_, err = reconcileDeleteSecurityGroupRule(ctx, clusterScope, securityGroupRuleSpec, securityGroupName, securityGroupSvc)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		log.V(2).Info("Deleting securityGroup", "securityGroupName", securityGroupName)
		_, err := deleteSecurityGroup(ctx, clusterScope, securityGroupsRef.ResourceMap[securityGroupName], securityGroupSvc, clock_time)
		if err != nil {
			return reconcile.Result{}, err
		}
	}
	return reconcile.Result{}, nil
}
