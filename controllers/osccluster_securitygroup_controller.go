package controllers

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Jeffail/gabs"
	infrastructurev1beta1 "github.com/outscale-vbr/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/services/net"
	"github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/services/security"
	"github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/services/service"
	tag "github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/tag"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// GetSecurityGroupResourceId return the SecurityGroupId from the resourceMap base on SecurityGroupName (tag name + cluster object uid)
func GetSecurityGroupResourceId(resourceName string, clusterScope *scope.ClusterScope) (string, error) {
	securityGroupRef := clusterScope.GetSecurityGroupsRef()
	if securityGroupId, ok := securityGroupRef.ResourceMap[resourceName]; ok {
		return securityGroupId, nil
	} else {
		return "", fmt.Errorf("%s is not exist", resourceName)
	}
}

// GetSecurityGroupRulesResourceId return the SecurityGroupId from the resourceMap base on SecurityGroupRuleName (tag name + cluster object uid)
func GetSecurityGroupRulesResourceId(resourceName string, clusterScope *scope.ClusterScope) (string, error) {
	securityGroupRuleRef := clusterScope.GetSecurityGroupRuleRef()
	if securityGroupRuleId, ok := securityGroupRuleRef.ResourceMap[resourceName]; ok {
		return securityGroupRuleId, nil
	} else {
		return "", fmt.Errorf("%s is not exist", resourceName)
	}
}

// CheckSecurityGroupOscDuplicateName check that there are not the same name for securityGroup
func CheckSecurityGroupOscDuplicateName(clusterScope *scope.ClusterScope) error {
	var resourceNameList []string
	clusterScope.Info("check unique security group rule")
	securityGroupsSpec := clusterScope.GetSecurityGroups()
	for _, securityGroupSpec := range securityGroupsSpec {
		securityGroupRulesSpec := clusterScope.GetSecurityGroupRule(securityGroupSpec.Name)
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

// CheckSecurityGroupRuleOscDuplicateName check that there are not the same name for securityGroupRule
func CheckSecurityGroupRuleOscDuplicateName(clusterScope *scope.ClusterScope) error {
	clusterScope.Info("check unique security group rule")
	var resourceNameList []string
	securityGroupsSpec := clusterScope.GetSecurityGroups()
	for _, securityGroupSpec := range securityGroupsSpec {
		securityGroupRulesSpec := clusterScope.GetSecurityGroupRule(securityGroupSpec.Name)
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

// CheckSecurityGroupFormatParameters check securityGroup parameters format (Tag format, cidr format, ..)
func CheckSecurityGroupFormatParameters(clusterScope *scope.ClusterScope) (string, error) {
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
		_, err = security.ValidateDescription(securityGroupDescription)
		if err != nil {
			return securityGroupTagName, err
		}
	}
	return "", nil
}

// CheckFormatParameters check every securityGroupRule parameters format (Tag format, cidr format, ..)
func CheckSecurityGroupRuleFormatParameters(clusterScope *scope.ClusterScope) (string, error) {
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

// reconcileSecurityGroupRule reconcile the securityGroupRule of the cluster.
func reconcileSecurityGroupRule(ctx context.Context, clusterScope *scope.ClusterScope, securityGroupRuleSpec infrastructurev1beta1.OscSecurityGroupRule, securityGroupName string) (reconcile.Result, error) {
	securitysvc := security.NewService(ctx, clusterScope)
	//osccluster := clusterScope.OscCluster

	securityGroupsRef := clusterScope.GetSecurityGroupsRef()
	securityGroupRuleRef := clusterScope.GetSecurityGroupRuleRef()

	securityGroupRuleName := securityGroupRuleSpec.Name + "-" + clusterScope.GetUID()
	if len(securityGroupRuleRef.ResourceMap) == 0 {
		securityGroupRuleRef.ResourceMap = make(map[string]string)
	}
	if securityGroupRuleSpec.ResourceId != "" {
		securityGroupRuleRef.ResourceMap[securityGroupRuleName] = securityGroupRuleSpec.ResourceId
	}
	Flow := securityGroupRuleSpec.Flow
	IpProtocol := securityGroupRuleSpec.IpProtocol
	IpRange := securityGroupRuleSpec.IpRange
	FromPortRange := securityGroupRuleSpec.FromPortRange
	ToPortRange := securityGroupRuleSpec.ToPortRange
	associateSecurityGroupId := securityGroupsRef.ResourceMap[securityGroupName]

	securityGroupFromSecurityGroupRule, err := securitysvc.GetSecurityGroupFromSecurityGroupRule(associateSecurityGroupId, Flow, IpProtocol, IpRange, FromPortRange, ToPortRange)

	if err != nil {
		return reconcile.Result{}, err
	}
	if securityGroupFromSecurityGroupRule == nil {
		clusterScope.Info("### Create securityGroupRule")
		securityGroupFromSecurityGroupRule, err = securitysvc.CreateSecurityGroupRule(securityGroupsRef.ResourceMap[securityGroupName], Flow, IpProtocol, IpRange, FromPortRange, ToPortRange)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not create  securityGroupRule for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
		}
	}
	securityGroupRuleRef.ResourceMap[securityGroupRuleName] = securityGroupFromSecurityGroupRule.GetSecurityGroupId()
	securityGroupRuleSpec.ResourceId = securityGroupFromSecurityGroupRule.GetSecurityGroupId()
	return reconcile.Result{}, nil
}

// reconcileSecurityGroupRule reconcile the deletion of securityGroup of the cluster.
func DeleteSecurityGroup(ctx context.Context, clusterScope *scope.ClusterScope, securityGroupId string) (reconcile.Result, error) {
	securitysvc := security.NewService(ctx, clusterScope)
	clusterScope.Info("Check loadbalancer deletion")

	currentTimeout := time.Now().Add(time.Second * 120)
	var loadbalancer_delete = false
	for !loadbalancer_delete {
		err, httpRes := securitysvc.DeleteSecurityGroup(securityGroupId)
		if err != nil {
			buffer := new(strings.Builder)
			_, err := io.Copy(buffer, httpRes.Body)
			httpResBody := buffer.String()
			httpResBodyData := []byte(httpResBody)
			httpResBodyParsed, err := gabs.ParseJSON(httpResBodyData)

			if err != nil {
				return reconcile.Result{}, fmt.Errorf("%w Can not delete publicIp for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
			}
			httpResCode := strings.Replace(strings.Replace(fmt.Sprintf("%v", httpResBodyParsed.Path("Errors.Code").Data()), "[", "", 1), "]", "", 1)
			httpResType := strings.Replace(strings.Replace(fmt.Sprintf("%v", httpResBodyParsed.Path("Errors.Type").Data()), "[", "", 1), "]", "", 1)
			var unexpectedErr bool = true

			if httpResCode == "9029" && httpResType == "ResourceConflict" {
				clusterScope.Info("LoadBalancer is not deleting yet")
				unexpectedErr = false
			}
			if unexpectedErr {
				return reconcile.Result{}, fmt.Errorf(" Can not delete publicIp because of the uncatch error for Osccluster %s/%s", clusterScope.GetNamespace(), clusterScope.GetName())
			}
		} else {
			loadbalancer_delete = true
		}
		clusterScope.Info("Wait until loadBalancer is deleting")

		time.Sleep(10 * time.Second)

		if time.Now().After(currentTimeout) {
			return reconcile.Result{}, fmt.Errorf("%w Can not delete publicIp because to waiting loadbalancer to be delete timeout  for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
		}

	}
	return reconcile.Result{}, nil
}

// reconcileSecurityGroup reconcile the securityGroup of the cluster.
func reconcileSecurityGroup(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	securitysvc := security.NewService(ctx, clusterScope)

	clusterScope.Info("Create SecurityGroup")
	var securityGroupsSpec []*infrastructurev1beta1.OscSecurityGroup
	networkSpec := clusterScope.GetNetwork()
	if networkSpec.SecurityGroups == nil {
		networkSpec.SetSecurityGroupDefaultValue()
		securityGroupsSpec = networkSpec.SecurityGroups
	} else {
		securityGroupsSpec = clusterScope.GetSecurityGroups()
	}

	netSpec := clusterScope.GetNet()
	netSpec.SetDefaultValue()
	netName := netSpec.Name + "-" + clusterScope.GetUID()
	netId, err := GetNetResourceId(netName, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}

	securityGroupIds, err := securitysvc.GetSecurityGroupIdsFromNetIds(netId)
	if err != nil {
		return reconcile.Result{}, err
	}
	securityGroupsRef := clusterScope.GetSecurityGroupsRef()
	for _, securityGroupSpec := range securityGroupsSpec {
		securityGroupName := securityGroupSpec.Name + "-" + clusterScope.GetUID()
		securityGroupDescription := securityGroupSpec.Description
		securityGroupId := securityGroupsRef.ResourceMap[securityGroupName]
		clusterScope.Info("### Get securityGroup Id ###", "securityGroup", securityGroupIds)
		if len(securityGroupsRef.ResourceMap) == 0 {
			securityGroupsRef.ResourceMap = make(map[string]string)
		}
		if securityGroupSpec.ResourceId != "" {
			securityGroupsRef.ResourceMap[securityGroupName] = securityGroupSpec.ResourceId
		}
		if !Contains(securityGroupIds, securityGroupId) {
			securityGroup, err := securitysvc.CreateSecurityGroup(netId, securityGroupName, securityGroupDescription)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("%w Can not create securitygroup for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
			}
			clusterScope.Info("### Get securityGroup", "securityGroup", securityGroup)
			securityGroupsRef.ResourceMap[securityGroupName] = *securityGroup.SecurityGroupId
			securityGroupSpec.ResourceId = *securityGroup.SecurityGroupId

			clusterScope.Info("check securityGroupRule")
			securityGroupRulesSpec := clusterScope.GetSecurityGroupRule(securityGroupSpec.Name)
			for _, securityGroupRuleSpec := range *securityGroupRulesSpec {
				_, err = reconcileSecurityGroupRule(ctx, clusterScope, securityGroupRuleSpec, securityGroupName)
				if err != nil {
					return reconcile.Result{}, err
				}
			}
		}
	}
	return reconcile.Result{}, nil
}

// ReconcileRoute reconcile the RouteTable and the Route of the cluster.
func reconcileDeleteSecurityGroupRule(ctx context.Context, clusterScope *scope.ClusterScope, securityGroupRuleSpec infrastructurev1beta1.OscSecurityGroupRule, securityGroupName string) (reconcile.Result, error) {
	securitysvc := security.NewService(ctx, clusterScope)
	osccluster := clusterScope.OscCluster
	securityGroupsRef := clusterScope.GetSecurityGroupsRef()

	Flow := securityGroupRuleSpec.Flow
	IpProtocol := securityGroupRuleSpec.IpProtocol
	IpRange := securityGroupRuleSpec.IpRange
	FromPortRange := securityGroupRuleSpec.FromPortRange
	ToPortRange := securityGroupRuleSpec.ToPortRange
	associateSecurityGroupId := securityGroupsRef.ResourceMap[securityGroupName]
	clusterScope.Info("Delete SecurityGroupRule")
	securityGroupFromSecurityGroupRule, err := securitysvc.GetSecurityGroupFromSecurityGroupRule(associateSecurityGroupId, Flow, IpProtocol, IpRange, FromPortRange, ToPortRange)
	if err != nil {
		return reconcile.Result{}, err
	}
	if securityGroupFromSecurityGroupRule == nil {
		controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
		return reconcile.Result{}, nil
	}
	err = securitysvc.DeleteSecurityGroupRule(securityGroupsRef.ResourceMap[securityGroupName], Flow, IpProtocol, IpRange, FromPortRange, ToPortRange)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("%w Can not delete securityGroupRule for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
	}
	return reconcile.Result{}, nil
}

// reconcileDeleteSecurityGroup reconcile the deletetion of securityGroup of the cluster.
func reconcileDeleteSecurityGroup(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	osccluster := clusterScope.OscCluster
	securitysvc := security.NewService(ctx, clusterScope)

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
	netId, err := GetNetResourceId(netName, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	securityGroupIds, err := securitysvc.GetSecurityGroupIdsFromNetIds(netId)
	if err != nil {
		return reconcile.Result{}, err
	}
	for _, securityGroupSpec := range securityGroupsSpec {
		securityGroupName := securityGroupSpec.Name + "-" + clusterScope.GetUID()
		securityGroupId := securityGroupsRef.ResourceMap[securityGroupName]
		if !Contains(securityGroupIds, securityGroupId) {
			controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
			return reconcile.Result{}, nil
		}
		clusterScope.Info("Remove securityGroupRule")
		securityGroupRulesSpec := clusterScope.GetSecurityGroupRule(securityGroupSpec.Name)
		for _, securityGroupRuleSpec := range *securityGroupRulesSpec {
			_, err = reconcileDeleteSecurityGroupRule(ctx, clusterScope, securityGroupRuleSpec, securityGroupName)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		clusterScope.Info("Delete SecurityGroup")
		_, err := DeleteSecurityGroup(ctx, clusterScope, securityGroupsRef.ResourceMap[securityGroupName])
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not delete securityGroup  for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
		}
	}
	return reconcile.Result{}, nil
}
