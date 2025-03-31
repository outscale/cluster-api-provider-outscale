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

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/security"
	tag "github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	"github.com/outscale/cluster-api-provider-outscale/cloud/utils"
	ctrl "sigs.k8s.io/controller-runtime"
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
	return utils.CheckDuplicates(clusterScope.GetSecurityGroups(), func(sg *infrastructurev1beta1.OscSecurityGroup) string {
		return sg.Name
	})
}

// checkSecurityGroupRuleOscDuplicateName check that there are not the same name for securityGroupRule
func checkSecurityGroupRuleOscDuplicateName(clusterScope *scope.ClusterScope) error {
	securityGroupsSpec := clusterScope.GetSecurityGroups()
	for _, securityGroupSpec := range securityGroupsSpec {
		err := utils.CheckDuplicates(clusterScope.GetSecurityGroupRule(securityGroupSpec.Name), func(sgr infrastructurev1beta1.OscSecurityGroupRule) string {
			return sgr.Name
		})
		if err != nil {
			return err
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
		err = infrastructurev1beta1.ValidateDescription(securityGroupDescription)
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
		for _, securityGroupRuleSpec := range securityGroupRulesSpec {
			securityGroupRuleName := securityGroupRuleSpec.Name + "-" + clusterScope.GetUID()
			securityGroupRuleTagName, err := tag.ValidateTagNameValue(securityGroupRuleName)
			if err != nil {
				return securityGroupRuleTagName, err
			}
			securityGroupRuleFlow := securityGroupRuleSpec.Flow
			err = infrastructurev1beta1.ValidateFlow(securityGroupRuleFlow)
			if err != nil {
				return securityGroupRuleTagName, err
			}
			securityGroupRuleIpProtocol := securityGroupRuleSpec.IpProtocol
			err = infrastructurev1beta1.ValidateIpProtocol(securityGroupRuleIpProtocol)
			if err != nil {
				return securityGroupRuleTagName, err
			}
			// FIXME
			// securityGroupRuleIpRange := securityGroupRuleSpec.IpRange
			// err = infrastructurev1beta1.ValidateCidr(securityGroupRuleIpRange)
			// if err != nil {
			// 	return securityGroupRuleTagName, err
			// }
			securityGroupRuleFromPortRange := securityGroupRuleSpec.FromPortRange
			err = infrastructurev1beta1.ValidatePort(securityGroupRuleFromPortRange)
			if err != nil {
				return securityGroupRuleTagName, err
			}
			securityGroupRuleToPortRange := securityGroupRuleSpec.ToPortRange
			err = infrastructurev1beta1.ValidatePort(securityGroupRuleToPortRange)
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
func deleteSecurityGroup(ctx context.Context, clusterScope *scope.ClusterScope, securityGroupId string, securityGroupSvc security.OscSecurityGroupInterface) (reconcile.Result, error) {
	err := securityGroupSvc.DeleteSecurityGroup(ctx, securityGroupId)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot delete securityGroup: %w", err)
	}
	return reconcile.Result{}, nil
}

// reconcileSecurityGroup reconcile the securityGroup of the cluster.
func (r *OscClusterReconciler) reconcileSecurityGroup(ctx context.Context, clusterScope *scope.ClusterScope, securityGroupSvc security.OscSecurityGroupInterface, tagSvc tag.OscTagInterface) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	securityGroupsSpec := clusterScope.GetSecurityGroups()

	netId, err := r.Tracker.getNetId(ctx, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}

	for _, securityGroupSpec := range securityGroupsSpec {
		securityGroup, err := r.Tracker.getSecurityGroup(ctx, securityGroupSpec, clusterScope)
		switch {
		case errors.Is(err, ErrNoResourceFound):
			securityGroup, err = securityGroupSvc.CreateSecurityGroup(ctx, netId, clusterScope.GetName(), securityGroupSpec.Name, *securityGroup.Description, securityGroupSpec.Tag)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("cannot create securityGroup: %w", err)
			}
		case err != nil:
			return reconcile.Result{}, fmt.Errorf("reconcile securityGroup: %w", err)
		}

		log.V(3).Info("Checking securityGroupRules")
		securityGroupRulesSpec := clusterScope.GetSecurityGroupRule(securityGroupSpec.Name)
		log.V(4).Info("Number of securityGroupRule", "securityGroupRuleLength", len(securityGroupRulesSpec))
		for _, securityGroupRuleSpec := range securityGroupRulesSpec {
			log.V(4).Info("Create securityGroupRule for securityGroup", "securityGroupName", securityGroupName)
			_, err = reconcileSecurityGroupRule(ctx, clusterScope, securityGroupRuleSpec, securityGroupName, securityGroupSvc)
			if err != nil {
				return reconcile.Result{}, err
			}
		}

		if slices.Contains(securityGroupIds, securityGroupId) && extraSecurityGroupRule {
			log.V(4).Info("Extra Security Group Rule activated")
			securityGroupRulesSpec := clusterScope.GetSecurityGroupRule(securityGroupSpec.Name)
			log.V(4).Info("Number of securityGroupRule", "securityGroupRuleLength", len(securityGroupRulesSpec))
			for _, securityGroupRuleSpec := range securityGroupRulesSpec {
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
	securityGroupsRef := clusterScope.GetSecurityGroupsRef()

	securityGroupRuleName := securityGroupRuleSpec.Name + "-" + clusterScope.GetUID()

	Flow := securityGroupRuleSpec.Flow
	IpProtocol := securityGroupRuleSpec.IpProtocol
	IpRange := securityGroupRuleSpec.IpRange
	FromPortRange := securityGroupRuleSpec.FromPortRange
	ToPortRange := securityGroupRuleSpec.ToPortRange
	associateSecurityGroupId := securityGroupsRef.ResourceMap[securityGroupName]
	hasRule, err := securityGroupSvc.SecurityGroupHasRule(ctx, associateSecurityGroupId, Flow, IpProtocol, IpRange, "", FromPortRange, ToPortRange)
	if err != nil {
		return reconcile.Result{}, err
	}
	if !hasRule {
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
func (r *OscClusterReconciler) reconcileDeleteSecurityGroup(ctx context.Context, clusterScope *scope.ClusterScope, securityGroupSvc security.OscSecurityGroupInterface) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	if !clusterScope.NeedReconciliation(infrastructurev1beta1.ReconcilerSecurityGroup) {
		log.V(4).Info("No need for securityGroup reconciliation")
		return reconcile.Result{}, nil
	}
	netSpec := clusterScope.GetNet()
	if netSpec.UseExisting {
		log.V(3).Info("Using existing securityGroups")
		return reconcile.Result{}, nil
	}

	securityGroupsSpec := clusterScope.GetSecurityGroups()

	netId, err := r.Tracker.getNetId(ctx, clusterScope)
	if err != nil {
		log.V(3).Info("No net found, skipping security group deletion")
		return reconcile.Result{}, nil //nolint: nilerr
	}
	securityGroupIds, err := securityGroupSvc.GetSecurityGroupIdsFromNetIds(ctx, netId)
	if err != nil {
		return reconcile.Result{}, err
	}
	var sgerr error
	for _, securityGroupSpec := range securityGroupsSpec {
		sgName := securityGroupSpec.Name + "-" + clusterScope.GetUID()
		sgId := securityGroupsRef.ResourceMap[sgName]
		if !slices.Contains(securityGroupIds, sgId) {
			log.V(4).Info("securityGroup does not exist anymore", "securityGroupName", sgName)
			continue
		}
		sgRulesSpec := clusterScope.GetSecurityGroupRule(securityGroupSpec.Name)
		log.V(4).Info("Number of securityGroupRule", "securityGroupLength", len(sgRulesSpec))
		for _, securityGroupRuleSpec := range sgRulesSpec {
			_, err = reconcileDeleteSecurityGroupRule(ctx, clusterScope, securityGroupRuleSpec, sgName, securityGroupSvc)
			if err != nil {
				log.V(4).Error(err, "cannot delete security group rule", "securityGroupName", sgName)
				sgerr = err
			}
		}
		log.V(2).Info("Deleting securityGroup", "securityGroupName", sgName)
		_, err := deleteSecurityGroup(ctx, clusterScope, sgId, securityGroupSvc)
		if err != nil {
			log.V(4).Error(err, "cannot delete security group", "securityGroupName", sgName)
			sgerr = err
		}
	}
	return reconcile.Result{}, sgerr
}
