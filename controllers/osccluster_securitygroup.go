/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package controllers

import (
	"context"
	"fmt"
	"slices"
	"strings"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/osc-sdk-go/v3/pkg/osc"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// reconcileSecurityGroupAddRules reconciles rules for a securityGroup.
func (r *OscClusterReconciler) reconcileSecurityGroupAddRules(ctx context.Context, clusterScope *scope.ClusterScope, securityGroupRulesSpec []infrastructurev1beta1.OscSecurityGroupRule, sg *osc.SecurityGroup) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	svc := r.Cloud.Compute(clusterScope.Tenant)
	for _, securityGroupRuleSpec := range securityGroupRulesSpec {
		var rules []osc.SecurityGroupRule
		switch strings.ToLower(securityGroupRuleSpec.Flow) {
		case "inbound":
			rules = sg.InboundRules
		case "outbound":
			rules = sg.OutboundRules
		}
		flow := securityGroupRuleSpec.Flow
		protocol := securityGroupRuleSpec.IpProtocol
		fromPort := int(securityGroupRuleSpec.FromPortRange)
		toPort := int(securityGroupRuleSpec.ToPortRange)
		var existingRanges []string
		for _, rule := range rules {
			if rule.FromPortRange != fromPort || rule.ToPortRange != toPort || rule.IpProtocol != protocol {
				continue
			}
			existingRanges = append(existingRanges, rule.IpRanges...)
		}
		ipRanges := securityGroupRuleSpec.GetIpRanges()
		for _, ipRange := range ipRanges {
			if slices.Contains(existingRanges, ipRange) {
				continue
			}
			log.V(2).Info("Creating securityGroupRule", "flow", flow, "ipRange", ipRange, "protocol", protocol, "fromPort", fromPort, "toPort", toPort)
			_, err := svc.CreateSecurityGroupRule(ctx, sg.SecurityGroupId, flow, protocol, ipRange, "", fromPort, toPort)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("cannot create securityGroupRule: %w", err)
			}
		}
	}
	return reconcile.Result{}, nil
}

// reconcileSecurityGroupDeleteRules deletes all rules not in spec for a securityGroup.
func (r *OscClusterReconciler) reconcileSecurityGroupDeleteRules(ctx context.Context, clusterScope *scope.ClusterScope, securityGroupRulesSpec []infrastructurev1beta1.OscSecurityGroupRule, sg *osc.SecurityGroup) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	svc := r.Cloud.Compute(clusterScope.Tenant)
	checkRules := func(flow string, rules []osc.SecurityGroupRule) error {
		for _, rule := range rules {
			// Skipping rule created by CCM. There is no way to be sure this comes from the CCM,
			// but the CCM creates rules with an associated SG, and the config from the CRD only uses ipRanges.
			if len(rule.SecurityGroupsMembers) > 0 {
				log.V(5).Info("Skipping rule associated with another SG")
				continue
			}
			var okRanges []string
			for _, spec := range securityGroupRulesSpec {
				if flow != spec.Flow ||
					rule.FromPortRange != int(spec.FromPortRange) || rule.ToPortRange != int(spec.ToPortRange) ||
					rule.IpProtocol != spec.IpProtocol {
					continue
				}
				okRanges = append(okRanges, spec.GetIpRanges()...)
			}
			ipRanges := rule.IpRanges
			for _, ipRange := range ipRanges {
				if slices.Contains(okRanges, ipRange) {
					continue
				}
				log.V(2).Info("Deleting securityGroupRule", "flow", flow, "ipRange", ipRange, "protocol", rule.IpProtocol, "fromPort", rule.FromPortRange, "toPort", rule.ToPortRange)
				err := svc.DeleteSecurityGroupRule(ctx, sg.SecurityGroupId, flow, rule.IpProtocol, ipRange, "", rule.FromPortRange, rule.ToPortRange)
				if err != nil {
					return fmt.Errorf("cannot create securityGroupRule: %w", err)
				}
			}
		}
		return nil
	}
	err := checkRules("Inbound", sg.InboundRules)
	if err != nil {
		return reconcile.Result{}, err
	}
	err = checkRules("Outbound", sg.OutboundRules)
	if err != nil {
		return reconcile.Result{}, err
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

	if clusterScope.GetNetwork().UseExisting.SecurityGroups {
		log.V(3).Info("Using existing securityGroups")
		return reconcile.Result{}, nil
	}

	netId, err := r.Tracker.getNetId(ctx, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	securityGroupSvc := r.Cloud.Compute(clusterScope.Tenant)
	securityGroupsSpec := clusterScope.GetSecurityGroups()
	for _, securityGroupSpec := range securityGroupsSpec {
		securityGroup, err := r.Tracker.getSecurityGroup(ctx, securityGroupSpec, clusterScope)
		switch {
		case IsNotFound(err):
			log.V(3).Info("Creating securityGroup", "securityGroupName", securityGroupSpec.Name)
			name := clusterScope.GetSecurityGroupName(securityGroupSpec)
			securityGroup, err = securityGroupSvc.CreateSecurityGroup(ctx, netId, clusterScope.GetUID(), name, securityGroupSpec.Description, securityGroupSpec.Tag, securityGroupSpec.Roles)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("cannot create securityGroup: %w", err)
			}
			log.V(2).Info("Created securityGroup", "securityGroupId", securityGroup.SecurityGroupId)
			r.Tracker.setSecurityGroupId(clusterScope, securityGroupSpec, securityGroup.SecurityGroupId)
			r.Recorder.Eventf(clusterScope.OscCluster, corev1.EventTypeNormal, infrastructurev1beta1.SecurityGroupCreatedReason, "Security group created %v", securityGroupSpec.Roles)
		case err != nil:
			return reconcile.Result{}, fmt.Errorf("get existing: %w", err)
		}
		securityGroupRulesSpec := securityGroupSpec.SecurityGroupRules
		if securityGroupSpec.HasRole(infrastructurev1beta1.RoleLoadBalancer) && clusterScope.HasIPRestriction() {
			ips, err := r.listNATPublicIPs(ctx, clusterScope, true)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("cannot list NAT public IPs: %w", err)
			}
			securityGroupRulesSpec = append(securityGroupRulesSpec, infrastructurev1beta1.OscSecurityGroupRule{
				Flow:          "Inbound",
				IpProtocol:    "tcp",
				FromPortRange: infrastructurev1beta1.APIPort,
				ToPortRange:   infrastructurev1beta1.APIPort,
				IpRanges:      ips,
			})
		}
		log.V(4).Info("Checking securityGroup rules", "securityGroupId", securityGroup.SecurityGroupId)
		_, err = r.reconcileSecurityGroupAddRules(ctx, clusterScope, securityGroupRulesSpec, securityGroup)
		if err == nil && securityGroupSpec.Authoritative {
			_, err = r.reconcileSecurityGroupDeleteRules(ctx, clusterScope, securityGroupRulesSpec, securityGroup)
		}
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("check rules: %w", err)
		}
	}
	clusterScope.SetReconciliationGeneration(infrastructurev1beta1.ReconcilerSecurityGroup)
	return reconcile.Result{}, nil
}

// reconcileDeleteSecurityGroup reconcile the deletetion of securityGroup of the cluster.
func (r *OscClusterReconciler) reconcileDeleteSecurityGroup(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	if clusterScope.GetNetwork().UseExisting.SecurityGroups {
		log.V(3).Info("Not deleting existing securityGroups")
		return reconcile.Result{}, nil
	}

	netId, err := r.Tracker.getNetId(ctx, clusterScope)
	switch {
	case IsNotFound(err):
		log.V(4).Info("The net is already deleted, no securityGroup expected")
		return reconcile.Result{}, nil
	case err != nil:
		return reconcile.Result{}, fmt.Errorf("get net: %w", err)
	}

	securityGroupSvc := r.Cloud.Compute(clusterScope.Tenant)
	securityGroups, err := securityGroupSvc.GetSecurityGroupsFromNet(ctx, netId)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("list securityGroups: %w", err)
	}
	var sgerr error
	for _, securityGroup := range securityGroups {
		if securityGroup.SecurityGroupName == "default" {
			continue
		}
		log.V(2).Info("Deleting inbound rules", "securityGroupId", securityGroup.SecurityGroupId)
		for _, rule := range securityGroup.InboundRules {
			for _, member := range rule.SecurityGroupsMembers {
				err = securityGroupSvc.DeleteSecurityGroupRule(ctx, securityGroup.SecurityGroupId, "Inbound", rule.IpProtocol, "", member.SecurityGroupId, rule.FromPortRange, rule.ToPortRange)
				if err != nil {
					return reconcile.Result{}, fmt.Errorf("cannot delete rule: %w", err)
				}
			}
		}
		log.V(2).Info("Deleting outbound rules", "securityGroupId", securityGroup.SecurityGroupId)
		for _, rule := range securityGroup.OutboundRules {
			for _, member := range rule.SecurityGroupsMembers {
				err = securityGroupSvc.DeleteSecurityGroupRule(ctx, securityGroup.SecurityGroupId, "Outbound", rule.IpProtocol, "", member.SecurityGroupId, rule.FromPortRange, rule.ToPortRange)
				if err != nil {
					return reconcile.Result{}, fmt.Errorf("cannot delete rule: %w", err)
				}
			}
		}
	}
	for _, securityGroup := range securityGroups {
		if securityGroup.SecurityGroupName == "default" {
			continue
		}
		log.V(2).Info("Deleting securityGroup", "securityGroupId", securityGroup.SecurityGroupId)
		err := securityGroupSvc.DeleteSecurityGroup(ctx, securityGroup.SecurityGroupId)
		if err != nil {
			sgerr = fmt.Errorf("cannot delete %s: %w", securityGroup.SecurityGroupId, err)
		}
	}
	return reconcile.Result{}, sgerr
}
