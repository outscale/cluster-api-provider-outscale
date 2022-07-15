package controllers

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Jeffail/gabs"
	"github.com/benbjohnson/clock"
	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/security"
	tag "github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag"
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
	clusterScope.Info("check unique security group rule")
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
	clusterScope.Info("check unique security group rule")
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
	clusterScope.Info("### Get associateSecurityGroupId###", "securityGroup", associateSecurityGroupId)
	clusterScope.Info("check if the desired securityGroupRule exist", "securityGroupRuleName", securityGroupRuleName)
	securityGroupFromSecurityGroupRule, err := securityGroupSvc.GetSecurityGroupFromSecurityGroupRule(associateSecurityGroupId, Flow, IpProtocol, IpRange, FromPortRange, ToPortRange)
	if err != nil {
		return reconcile.Result{}, err
	}
	if securityGroupFromSecurityGroupRule == nil {
		clusterScope.Info("### Create securityGroupRule")
		clusterScope.Info("Create the desired securityGroupRule", "securityGroupRuleName", securityGroupRuleName)
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
	clusterScope.Info("Check loadbalancer deletion")

	currentTimeout := clock_time.Now().Add(time.Second * 20)
	var loadbalancer_delete = false
	for !loadbalancer_delete {
		err, httpRes := securityGroupSvc.DeleteSecurityGroup(securityGroupId)
		if err != nil {
			buffer := new(strings.Builder)
			_, err := io.Copy(buffer, httpRes.Body)
			httpResBody := buffer.String()
			clusterScope.Info("Find body", "httpResBody", httpResBody)
			httpResBodyData := []byte(httpResBody)
			httpResBodyParsed, err := gabs.ParseJSON(httpResBodyData)

			if err != nil {
				return reconcile.Result{}, fmt.Errorf("%w Can not delete securityGroup for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
			}
			httpResCode := strings.Replace(strings.Replace(fmt.Sprintf("%v", httpResBodyParsed.Path("Errors.Code").Data()), "[", "", 1), "]", "", 1)
			httpResType := strings.Replace(strings.Replace(fmt.Sprintf("%v", httpResBodyParsed.Path("Errors.Type").Data()), "[", "", 1), "]", "", 1)
			var unexpectedErr bool = true

			if httpResCode == "9085" && httpResType == "ResourceConflict" {
				clusterScope.Info("LoadBalancer is not deleting yet")
				unexpectedErr = false
			}
			if unexpectedErr {
				return reconcile.Result{}, fmt.Errorf(" Can not delete securityGroup because of the uncatch error for Osccluster %s/%s", clusterScope.GetNamespace(), clusterScope.GetName())
			}
			clusterScope.Info("Wait until loadBalancer is deleting")
			time.Sleep(5 * time.Second)
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
func reconcileSecurityGroup(ctx context.Context, clusterScope *scope.ClusterScope, securityGroupSvc security.OscSecurityGroupInterface) (reconcile.Result, error) {

	clusterScope.Info("Create SecurityGroup")
	securityGroupsSpec := clusterScope.GetSecurityGroups()

	netSpec := clusterScope.GetNet()
	netSpec.SetDefaultValue()
	netName := netSpec.Name + "-" + clusterScope.GetUID()
	netId, err := getNetResourceId(netName, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}

	clusterScope.Info("Get list of all desired securitygroup in net", "netId", netId)
	securityGroupIds, err := securityGroupSvc.GetSecurityGroupIdsFromNetIds(netId)
	if err != nil {
		return reconcile.Result{}, err
	}
	securityGroupsRef := clusterScope.GetSecurityGroupsRef()

	for _, securityGroupSpec := range securityGroupsSpec {
		securityGroupName := securityGroupSpec.Name + "-" + clusterScope.GetUID()
		clusterScope.Info("Check if the desired securityGroup exist in net", "securityGroupName", securityGroupName)
		securityGroupDescription := securityGroupSpec.Description
		clusterScope.Info("### Get securityGroup Id ###", "securityGroup", securityGroupIds)
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

		securityGroupId := securityGroupsRef.ResourceMap[securityGroupName]
		if !Contains(securityGroupIds, securityGroupId) {
			clusterScope.Info("Find securitygroup", "securityGroup", securityGroupId)
			clusterScope.Info("Create the desired securitygroup", "securityGroupName", securityGroupName)
			securityGroup, err := securityGroupSvc.CreateSecurityGroup(netId, securityGroupName, securityGroupDescription)
			clusterScope.Info("### Get securityGroup", "securityGroup", securityGroup)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("%w Can not create securityGroup for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
			}
			securityGroupsRef.ResourceMap[securityGroupName] = *securityGroup.SecurityGroupId
			securityGroupSpec.ResourceId = *securityGroup.SecurityGroupId

			clusterScope.Info("check securityGroupRule")
			securityGroupRulesSpec := clusterScope.GetSecurityGroupRule(securityGroupSpec.Name)
			for _, securityGroupRuleSpec := range *securityGroupRulesSpec {
				clusterScope.Info("Create securityGroupRule for the desired securityGroup", "securityGroupName", securityGroupName)
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
func reconcileDeleteSecurityGroupRule(ctx context.Context, clusterScope *scope.ClusterScope, securityGroupRuleSpec infrastructurev1beta1.OscSecurityGroupRule, securityGroupName string, securityGroupSvc security.OscSecurityGroupInterface) (reconcile.Result, error) {
	osccluster := clusterScope.OscCluster
	securityGroupsRef := clusterScope.GetSecurityGroupsRef()

	securityGroupRuleName := securityGroupRuleSpec.Name + "-" + clusterScope.GetUID()

	Flow := securityGroupRuleSpec.Flow
	IpProtocol := securityGroupRuleSpec.IpProtocol
	IpRange := securityGroupRuleSpec.IpRange
	FromPortRange := securityGroupRuleSpec.FromPortRange
	ToPortRange := securityGroupRuleSpec.ToPortRange
	associateSecurityGroupId := securityGroupsRef.ResourceMap[securityGroupName]
	clusterScope.Info("Delete SecurityGroupRule")
	clusterScope.Info("Check if the desired securityGroupRule exist", "securityGroupRuleName", securityGroupRuleName)
	securityGroupFromSecurityGroupRule, err := securityGroupSvc.GetSecurityGroupFromSecurityGroupRule(associateSecurityGroupId, Flow, IpProtocol, IpRange, FromPortRange, ToPortRange)
	if err != nil {
		return reconcile.Result{}, err
	}
	if securityGroupFromSecurityGroupRule == nil {
		clusterScope.Info("the desired securityGroupRule does not exist anymore", "securityGroupRuleName", securityGroupRuleName)
		controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
		return reconcile.Result{}, nil
	}
	clusterScope.Info("Delete the desired securityGroupRule", "securityGroupRuleName", securityGroupRuleName)
	err = securityGroupSvc.DeleteSecurityGroupRule(associateSecurityGroupId, Flow, IpProtocol, IpRange, "", FromPortRange, ToPortRange)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("%s Can not delete securityGroupRule for OscCluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
	}
	return reconcile.Result{}, nil
}

// reconcileDeleteSecurityGroup reconcile the deletetion of securityGroup of the cluster.
func reconcileDeleteSecurityGroup(ctx context.Context, clusterScope *scope.ClusterScope, securityGroupSvc security.OscSecurityGroupInterface) (reconcile.Result, error) {
	osccluster := clusterScope.OscCluster

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
	netName := netSpec.Name + "-" + clusterScope.GetUID()
	netId, err := getNetResourceId(netName, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	securityGroupIds, err := securityGroupSvc.GetSecurityGroupIdsFromNetIds(netId)
	if err != nil {
		return reconcile.Result{}, err
	}
	clusterScope.Info("Delete SecurityGroup Info")
	clock_time := clock.New()
	for _, securityGroupSpec := range securityGroupsSpec {
		securityGroupName := securityGroupSpec.Name + "-" + clusterScope.GetUID()
		securityGroupId := securityGroupsRef.ResourceMap[securityGroupName]
		if !Contains(securityGroupIds, securityGroupId) {
			clusterScope.Info("the desired securityGroup does not exist anymore", "securityGroupName", securityGroupName)
			controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
			return reconcile.Result{}, nil
		}
		clusterScope.Info("Remove securityGroupRule")
		securityGroupRulesSpec := clusterScope.GetSecurityGroupRule(securityGroupSpec.Name)
		for _, securityGroupRuleSpec := range *securityGroupRulesSpec {
			_, err = reconcileDeleteSecurityGroupRule(ctx, clusterScope, securityGroupRuleSpec, securityGroupName, securityGroupSvc)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		clusterScope.Info("Delete SecurityGroup")
		clusterScope.Info("delete the desired securityGroup", "securityGroupName", securityGroupName)
		_, err := deleteSecurityGroup(ctx, clusterScope, securityGroupsRef.ResourceMap[securityGroupName], securityGroupSvc, clock_time)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not delete securityGroup  for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
		}
	}
	return reconcile.Result{}, nil
}
