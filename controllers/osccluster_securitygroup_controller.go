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
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// reconcileSecurityGroupAddRules reconciles rules for a securityGroup.
func (r *OscClusterReconciler) reconcileSecurityGroupAddRules(ctx context.Context, clusterScope *scope.ClusterScope, securityGroupRulesSpec []infrastructurev1beta1.OscSecurityGroupRule, sg *osc.SecurityGroup) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	for _, securityGroupRuleSpec := range securityGroupRulesSpec {
		var rules []osc.SecurityGroupRule
		switch strings.ToLower(securityGroupRuleSpec.Flow) {
		case "inbound":
			rules = sg.GetInboundRules()
		case "outbound":
			rules = sg.GetOutboundRules()
		}
		flow := securityGroupRuleSpec.Flow
		protocol := securityGroupRuleSpec.IpProtocol
		fromPort := securityGroupRuleSpec.FromPortRange
		toPort := securityGroupRuleSpec.ToPortRange
		var existingRanges []string
		for _, rule := range rules {
			if rule.GetFromPortRange() != fromPort || rule.GetToPortRange() != toPort || rule.GetIpProtocol() != protocol {
				continue
			}
			existingRanges = append(existingRanges, rule.GetIpRanges()...)
		}
		ipRanges := securityGroupRuleSpec.GetIpRanges()
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
	}
	return reconcile.Result{}, nil
}

// reconcileSecurityGroupDeleteRules deletes all rules not in spec for a securityGroup.
func (r *OscClusterReconciler) reconcileSecurityGroupDeleteRules(ctx context.Context, clusterScope *scope.ClusterScope, securityGroupRulesSpec []infrastructurev1beta1.OscSecurityGroupRule, sg *osc.SecurityGroup) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	var checkRules = func(flow string, rules []osc.SecurityGroupRule) error {
		for _, rule := range rules {
			// Skipping rule created by CCM. There is no way to be sure this comes from the CCM,
			// but the CCM creates rules with an associated SG, and the config from the CRD only uses ipRanges.
			if len(rule.GetSecurityGroupsMembers()) > 0 {
				log.V(5).Info("Skipping rule associated with another SG")
				continue
			}
			var okRanges []string
			for _, spec := range securityGroupRulesSpec {
				if flow != spec.Flow || rule.GetFromPortRange() != spec.FromPortRange || rule.GetToPortRange() != spec.ToPortRange || rule.GetIpProtocol() != spec.IpProtocol {
					continue
				}
				okRanges = append(okRanges, spec.GetIpRanges()...)
			}
			ipRanges := rule.GetIpRanges()
			for _, ipRange := range ipRanges {
				if slices.Contains(okRanges, ipRange) {
					continue
				}
				log.V(2).Info("Deleting securityGroupRule", "flow", flow, "ipRange", ipRange, "protocol", rule.GetIpProtocol(), "fromPort", rule.GetFromPortRange(), "toPort", rule.GetToPortRange())
				err := r.Cloud.SecurityGroup(ctx, *clusterScope).DeleteSecurityGroupRule(ctx, sg.GetSecurityGroupId(), flow, rule.GetIpProtocol(), ipRange, "", rule.GetFromPortRange(), rule.GetToPortRange())
				if err != nil {
					return fmt.Errorf("cannot create securityGroupRule: %w", err)
				}
			}
		}
		return nil
	}
	err := checkRules("Inbound", sg.GetInboundRules())
	if err != nil {
		return reconcile.Result{}, err
	}
	err = checkRules("Outbound", sg.GetOutboundRules())
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
	securityGroupSvc := r.Cloud.SecurityGroup(ctx, *clusterScope)
	securityGroupsSpec := clusterScope.GetSecurityGroups()
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
			log.V(2).Info("Created securityGroup", "securityGroupId", securityGroup.GetSecurityGroupId())
			r.Tracker.setSecurityGroupId(clusterScope, securityGroupSpec, securityGroup.GetSecurityGroupId())
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
		log.V(4).Info("Checking securityGroup rules", "securityGroupId", securityGroup.GetSecurityGroupId())
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
					return reconcile.Result{}, fmt.Errorf("cannot delete rule: %w", err)
				}
			}
		}
		log.V(2).Info("Deleting outbound rules", "securityGroupId", securityGroup.GetSecurityGroupId())
		for _, rule := range securityGroup.GetOutboundRules() {
			for _, member := range rule.GetSecurityGroupsMembers() {
				err = securityGroupSvc.DeleteSecurityGroupRule(ctx, securityGroup.GetSecurityGroupId(), "Outbound", rule.GetIpProtocol(), "", *member.SecurityGroupId, rule.GetFromPortRange(), rule.GetToPortRange())
				if err != nil {
					return reconcile.Result{}, fmt.Errorf("cannot delete rule: %w", err)
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
			sgerr = fmt.Errorf("cannot delete %s: %w", securityGroup.GetSecurityGroupId(), err)
		}
	}
	return reconcile.Result{}, sgerr
}
