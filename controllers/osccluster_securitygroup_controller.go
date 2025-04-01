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
	"strings"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/osc-sdk-go/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// reconcileSecurityGroupRule reconcile the securityGroupRule of the cluster.
func (r *OscClusterReconciler) reconcileSecurityGroupRule(ctx context.Context, clusterScope *scope.ClusterScope, securityGroupRuleSpec infrastructurev1beta1.OscSecurityGroupRule, sg *osc.SecurityGroup) (reconcile.Result, error) {
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
	ipRange := securityGroupRuleSpec.IpRange
	fromPort := securityGroupRuleSpec.FromPortRange
	toPort := securityGroupRuleSpec.ToPortRange
	for _, rule := range rules {
		if rule.GetFromPortRange() == fromPort && rule.GetToPortRange() == toPort && rule.GetIpProtocol() == protocol && len(rule.GetIpRanges()) == 1 && ipRange == rule.GetIpRanges()[0] {
			return reconcile.Result{}, nil
		}
	}
	log.V(2).Info("Creating securityGroupRule", "flow", flow, "ipRange", ipRange)
	_, err := r.Cloud.SecurityGroup(ctx, *clusterScope).CreateSecurityGroupRule(ctx, sg.GetSecurityGroupId(), flow, protocol, ipRange, "", fromPort, toPort)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot create securityGroupRule: %w", err)
	}
	return reconcile.Result{}, nil
}

// reconcileSecurityGroup reconcile the securityGroup of the cluster.
func (r *OscClusterReconciler) reconcileSecurityGroup(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	if !clusterScope.NeedReconciliation(infrastructurev1beta1.ReconcilerRouteTable) {
		log.V(4).Info("No need for securityGroup reconciliation")
		return reconcile.Result{}, nil
	}
	netSpec := clusterScope.GetNet()
	if netSpec.UseExisting {
		log.V(3).Info("Using existing securityGroups")
		return reconcile.Result{}, nil
	}

	netId, err := r.Tracker.getNetId(ctx, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	securityGroupSvc := r.Cloud.SecurityGroup(ctx, *clusterScope)
	securityGroupsSpec := clusterScope.GetSecurityGroups()
	for _, securityGroupSpec := range securityGroupsSpec {
		securityGroup, err := r.Tracker.getSecurityGroup(ctx, securityGroupSpec, clusterScope)
		switch {
		case errors.Is(err, ErrNoResourceFound):
			log.V(3).Info("Creating securityGroup")
			securityGroup, err = securityGroupSvc.CreateSecurityGroup(ctx, netId, clusterScope.GetName(), securityGroupSpec.Name, *securityGroup.Description, securityGroupSpec.Tag)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("cannot create securityGroup: %w", err)
			}
			log.V(3).Info("Created securityGroup", "securityGroupId", securityGroup.GetSecurityGroupId())
			r.Tracker.setSecurityGroupId(clusterScope, securityGroupSpec, securityGroup.GetSecurityGroupId())
		case err != nil:
			return reconcile.Result{}, fmt.Errorf("reconcile securityGroup: %w", err)
		}
		securityGroupRulesSpec := securityGroupSpec.SecurityGroupRules
		if len(securityGroupRulesSpec) <= len(securityGroup.GetInboundRules())+len(securityGroup.GetOutboundRules()) {
			log.V(4).Info("Same number of rules, not checking")
			continue
		}
		for _, securityGroupRuleSpec := range securityGroupRulesSpec {
			_, err = r.reconcileSecurityGroupRule(ctx, clusterScope, securityGroupRuleSpec, securityGroup)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
	}

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
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot find net: %w", err)
	}

	securityGroupSvc := r.Cloud.SecurityGroup(ctx, *clusterScope)
	securityGroups, err := securityGroupSvc.GetSecurityGroupsFromNet(ctx, netId)
	if err != nil {
		return reconcile.Result{}, err
	}
	var sgerr error
	for _, securityGroup := range securityGroups {
		log.V(2).Info("Deleting inbound rules", "securityGroupId", securityGroup.GetSecurityGroupId())
		for _, rule := range securityGroup.GetInboundRules() {
			for _, rng := range rule.GetIpRanges() {
				err = securityGroupSvc.DeleteSecurityGroupRule(ctx, securityGroup.GetSecurityGroupId(), "Inbound", rule.GetIpProtocol(), rng, "", rule.GetFromPortRange(), rule.GetToPortRange())
				if err != nil {
					return reconcile.Result{}, fmt.Errorf("cannot delete securityGroupRule: %w", err)
				}
			}
		}
		log.V(2).Info("Deleting outbound rules", "securityGroupId", securityGroup.GetSecurityGroupId())
		for _, rule := range securityGroup.GetOutboundRules() {
			for _, rng := range rule.GetIpRanges() {
				err = securityGroupSvc.DeleteSecurityGroupRule(ctx, securityGroup.GetSecurityGroupId(), "Outbound", rule.GetIpProtocol(), rng, "", rule.GetFromPortRange(), rule.GetToPortRange())
				if err != nil {
					return reconcile.Result{}, fmt.Errorf("cannot delete securityGroupRule: %w", err)
				}
			}
		}
	}
	for _, securityGroup := range securityGroups {
		log.V(2).Info("Deleting securityGroup", "securityGroupId", securityGroup.GetSecurityGroupId())
		err := securityGroupSvc.DeleteSecurityGroup(ctx, securityGroup.GetSecurityGroupId())
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("cannot delete securityGroup: %w", err)
		}
	}
	return reconcile.Result{}, sgerr
}
