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
	"strings"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/osc-sdk-go/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// reconcileSecurityGroupRules reconciles a rule for a securityGroup.
func (r *OscClusterReconciler) reconcileSecurityGroupRules(ctx context.Context, clusterScope *scope.ClusterScope, securityGroupRuleSpec infrastructurev1beta1.OscSecurityGroupRule, sg *osc.SecurityGroup) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	var rules []osc.SecurityGroupRule
	switch strings.ToLower(securityGroupRuleSpec.Flow) {
	case "inbound":
		rules = sg.GetInboundRules()
	case "outbound":
		rules = sg.GetOutboundRules()
	}
	flow := securityGroupRuleSpec.Flow
	protocol := securityGroupRuleSpec.IpProtocol
	ipRanges := securityGroupRuleSpec.GetIpRanges()
	fromPort := securityGroupRuleSpec.FromPortRange
	toPort := securityGroupRuleSpec.ToPortRange
	var existingRanges []string
	for _, rule := range rules {
		for _, ipRange := range securityGroupRuleSpec.GetIpRanges() {
			if rule.GetFromPortRange() == fromPort && rule.GetToPortRange() == toPort && rule.GetIpProtocol() == protocol && slices.Contains(ipRanges, ipRange) {
				existingRanges = append(existingRanges, ipRange)
			}
		}
	}
	if len(ipRanges) == len(existingRanges) {
		return reconcile.Result{}, nil
	}
	for _, ipRange := range ipRanges {
		if slices.Contains(existingRanges, ipRange) {
			continue
		}
		log.V(2).Info("Creating securityGroupRule", "flow", flow, "ipRange", ipRange, "protocol", protocol, "fromPort", fromPort, "toPort", toPort)
		_, err := r.Cloud.SecurityGroup(ctx, *clusterScope).CreateSecurityGroupRule(ctx, sg.GetSecurityGroupId(), flow, protocol, ipRange, "", fromPort, toPort)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot create securityGroupRule: %w", err)
		}
	}
	return reconcile.Result{}, nil
}

// reconcileSecurityGroup reconcile the securityGroup of the cluster.
func (r *OscClusterReconciler) reconcileSecurityGroup(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
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
	log.V(5).Info(fmt.Sprintf("%+v", securityGroupsSpec))
	errs := infrastructurev1beta1.ValidateSecurityGroups(securityGroupsSpec, netSpec)
	if len(errs) > 0 {
		return reconcile.Result{}, errs.ToAggregate()
	}

	netId, err := r.Tracker.getNetId(ctx, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	securityGroupSvc := r.Cloud.SecurityGroup(ctx, *clusterScope)
	for _, securityGroupSpec := range securityGroupsSpec {
		securityGroup, err := r.Tracker.getSecurityGroup(ctx, securityGroupSpec, clusterScope)
		switch {
		case errors.Is(err, ErrNoResourceFound):
			log.V(3).Info("Creating securityGroup", "securityGroupName", securityGroupSpec.Name)
			name := clusterScope.GetSecurityGroupName(securityGroupSpec)
			securityGroup, err = securityGroupSvc.CreateSecurityGroup(ctx, netId, clusterScope.GetUID(), name, securityGroupSpec.Description, securityGroupSpec.Tag)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("cannot create securityGroup: %w", err)
			}
			log.V(3).Info("Created securityGroup", "securityGroupId", securityGroup.GetSecurityGroupId())
			r.Tracker.setSecurityGroupId(clusterScope, securityGroupSpec, securityGroup.GetSecurityGroupId())
		case err != nil:
			return reconcile.Result{}, fmt.Errorf("get existing: %w", err)
		}
		securityGroupRulesSpec := securityGroupSpec.SecurityGroupRules
		if len(securityGroupRulesSpec) <= len(securityGroup.GetInboundRules())+len(securityGroup.GetOutboundRules()) {
			log.V(4).Info("Same number of rules, not checking securityGroup rules", "securityGroupId", securityGroup.GetSecurityGroupId())
			continue
		}
		log.V(4).Info("Checking securityGroup rules", "securityGroupId", securityGroup.GetSecurityGroupId())
		for _, securityGroupRuleSpec := range securityGroupRulesSpec {
			_, err = r.reconcileSecurityGroupRules(ctx, clusterScope, securityGroupRuleSpec, securityGroup)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
	}
	clusterScope.SetReconciliationGeneration(infrastructurev1beta1.ReconcilerSecurityGroup)
	return reconcile.Result{}, nil
}

// reconcileDeleteSecurityGroup reconcile the deletetion of securityGroup of the cluster.
func (r *OscClusterReconciler) reconcileDeleteSecurityGroup(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	netSpec := clusterScope.GetNet()
	if netSpec.UseExisting {
		log.V(3).Info("Not deleting existing securityGroups")
		return reconcile.Result{}, nil
	}

	netId, err := r.Tracker.getNetId(ctx, clusterScope)
	switch {
	case errors.Is(err, ErrNoResourceFound) || errors.Is(err, ErrMissingResource):
		log.V(4).Info("The net is already deleted, no securityGroup expected")
		return reconcile.Result{}, nil
	case err != nil:
		return reconcile.Result{}, fmt.Errorf("get net: %w", err)
	}

	securityGroupSvc := r.Cloud.SecurityGroup(ctx, *clusterScope)
	securityGroups, err := securityGroupSvc.GetSecurityGroupsFromNet(ctx, netId)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("list securityGroups: %w", err)
	}
	var sgerr error
	for _, securityGroup := range securityGroups {
		if securityGroup.GetSecurityGroupName() == "default" {
			continue
		}
		log.V(2).Info("Deleting inbound rules", "securityGroupId", securityGroup.GetSecurityGroupId())
		for _, rule := range securityGroup.GetInboundRules() {
			for _, member := range rule.GetSecurityGroupsMembers() {
				err = securityGroupSvc.DeleteSecurityGroupRule(ctx, securityGroup.GetSecurityGroupId(), "Inbound", rule.GetIpProtocol(), "", *member.SecurityGroupId, rule.GetFromPortRange(), rule.GetToPortRange())
				if err != nil {
					return reconcile.Result{}, fmt.Errorf("cannot delete securityGroupRule: %w", err)
				}
			}
		}
		log.V(2).Info("Deleting outbound rules", "securityGroupId", securityGroup.GetSecurityGroupId())
		for _, rule := range securityGroup.GetOutboundRules() {
			for _, member := range rule.GetSecurityGroupsMembers() {
				err = securityGroupSvc.DeleteSecurityGroupRule(ctx, securityGroup.GetSecurityGroupId(), "Outbound", rule.GetIpProtocol(), "", *member.SecurityGroupId, rule.GetFromPortRange(), rule.GetToPortRange())
				if err != nil {
					return reconcile.Result{}, fmt.Errorf("cannot delete securityGroupRule: %w", err)
				}
			}
		}
	}
	for _, securityGroup := range securityGroups {
		if securityGroup.GetSecurityGroupName() == "default" {
			continue
		}
		log.V(2).Info("Deleting securityGroup", "securityGroupId", securityGroup.GetSecurityGroupId())
		err := securityGroupSvc.DeleteSecurityGroup(ctx, securityGroup.GetSecurityGroupId())
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot delete securityGroup: %w", err)
		}
	}
	return reconcile.Result{}, sgerr
}
