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

package security

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	tag "github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	"github.com/outscale/cluster-api-provider-outscale/cloud/utils"
	"github.com/outscale/cluster-api-provider-outscale/util/reconciler"
	osc "github.com/outscale/osc-sdk-go/v2"
	"k8s.io/apimachinery/pkg/util/wait"
)

// ErrResourceConflict is returned by DeleteSecurityGroup when SG cannot be deleted because another resource requires it.
var ErrResourceConflict = errors.New("conflict with existing resource")

//go:generate ../../../bin/mockgen -destination mock_security/securitygroup_mock.go -package mock_security -source ./securitygroup.go

type OscSecurityGroupInterface interface {
	CreateSecurityGroup(ctx context.Context, netId string, clusterName string, securityGroupName string, securityGroupDescription string, securityGroupTag string) (*osc.SecurityGroup, error)
	CreateSecurityGroupRule(ctx context.Context, securityGroupId string, flow string, ipProtocol string, ipRange string, securityGroupMemberId string, fromPortRange int32, toPortRange int32) (*osc.SecurityGroup, error)
	DeleteSecurityGroupRule(ctx context.Context, securityGroupId string, flow string, ipProtocol string, ipRange string, securityGroupMemberId string, fromPortRange int32, toPortRange int32) error
	DeleteSecurityGroup(ctx context.Context, securityGroupId string) error
	GetSecurityGroup(ctx context.Context, securityGroupId string) (*osc.SecurityGroup, error)
	SecurityGroupHasRule(ctx context.Context, securityGroupId string, flow string, ipProtocols string, ipRanges string, securityGroupMemberId string, fromPortRanges int32, toPortRanges int32) (bool, error)
	GetSecurityGroupIdsFromNetIds(ctx context.Context, netId string) ([]string, error)
}

// CreateSecurityGroup create the securitygroup associated with the net
func (s *Service) CreateSecurityGroup(ctx context.Context, netId string, clusterName string, securityGroupName string, securityGroupDescription string, securityGroupTag string) (*osc.SecurityGroup, error) {
	securityGroupRequest := osc.CreateSecurityGroupRequest{
		SecurityGroupName: securityGroupName,
		Description:       securityGroupDescription,
		NetId:             &netId,
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	var securityGroupResponse osc.CreateSecurityGroupResponse
	createSecurityGroupCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		securityGroupResponse, httpRes, err = oscApiClient.SecurityGroupApi.CreateSecurityGroup(oscAuthClient).CreateSecurityGroupRequest(securityGroupRequest).Execute()
		utils.LogAPICall(ctx, "CreateSecurityGroup", securityGroupRequest, httpRes, err)
		if err != nil {
			if httpRes != nil {
				fmt.Printf("Error with http result %s", httpRes.Status)
			}
			requestStr := fmt.Sprintf("%v", securityGroupRequest)
			if reconciler.KeepRetryWithError(
				requestStr,
				httpRes.StatusCode,
				reconciler.ThrottlingErrors) {
				return false, nil
			}
			return false, err
		}
		return true, err
	}
	backoff := reconciler.EnvBackoff()
	waitErr := wait.ExponentialBackoff(backoff, createSecurityGroupCallBack)
	if waitErr != nil {
		return nil, waitErr
	}
	securityGroup, ok := securityGroupResponse.GetSecurityGroupOk()
	if !ok {
		return nil, errors.New("Can not create securitygroup")
	}
	resourceIds := []string{*securityGroupResponse.SecurityGroup.SecurityGroupId}
	clusterTag := osc.ResourceTag{
		Key:   "OscK8sClusterID/" + clusterName,
		Value: "owned",
	}
	clusterSecurityGroupRequest := osc.CreateTagsRequest{
		ResourceIds: resourceIds,
		Tags:        []osc.ResourceTag{clusterTag},
	}
	err, httpRes := tag.AddTag(ctx, clusterSecurityGroupRequest, resourceIds, oscApiClient, oscAuthClient)
	if err != nil {
		if httpRes != nil {
			return nil, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
		} else {
			return nil, err
		}
	}
	if securityGroupTag == "OscK8sMainSG" {
		mainTag := osc.ResourceTag{
			Key:   "OscK8sMainSG/" + clusterName,
			Value: "True",
		}
		mainSecurityGroupTagRequest := osc.CreateTagsRequest{
			ResourceIds: resourceIds,
			Tags:        []osc.ResourceTag{mainTag},
		}
		err, httpRes := tag.AddTag(ctx, mainSecurityGroupTagRequest, resourceIds, oscApiClient, oscAuthClient)
		if err != nil {
			if httpRes != nil {
				return nil, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
			} else {
				return nil, err
			}
		}
	}

	return securityGroup, nil
}

// CreateSecurityGroupRule create the security group rule associated with the security group and the net
func (s *Service) CreateSecurityGroupRule(ctx context.Context, securityGroupId string, flow string, ipProtocol string, ipRange string, securityGroupMemberId string, fromPortRange int32, toPortRange int32) (*osc.SecurityGroup, error) {
	var rule osc.SecurityGroupRule
	if securityGroupMemberId != "" && ipRange == "" {
		securityGroupMember := osc.SecurityGroupsMember{
			SecurityGroupId: &securityGroupMemberId,
		}
		rule = osc.SecurityGroupRule{
			SecurityGroupsMembers: &[]osc.SecurityGroupsMember{securityGroupMember},
			IpProtocol:            &ipProtocol,
			FromPortRange:         &fromPortRange,
			ToPortRange:           &toPortRange,
		}
	} else if securityGroupMemberId != "" && ipRange != "" {
		return nil, errors.New("Get Both ipRange and securityGroupMemberId")
	} else {
		rule = osc.SecurityGroupRule{
			IpProtocol:    &ipProtocol,
			IpRanges:      &[]string{ipRange},
			FromPortRange: &fromPortRange,
			ToPortRange:   &toPortRange,
		}
	}
	createSecurityGroupRuleRequest := osc.CreateSecurityGroupRuleRequest{
		Flow:            flow,
		SecurityGroupId: securityGroupId,
		Rules:           &[]osc.SecurityGroupRule{rule},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()

	var securityGroupRuleResponse osc.CreateSecurityGroupRuleResponse
	createSecurityGroupRuleCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		securityGroupRuleResponse, httpRes, err = oscApiClient.SecurityGroupRuleApi.CreateSecurityGroupRule(oscAuthClient).CreateSecurityGroupRuleRequest(createSecurityGroupRuleRequest).Execute()
		utils.LogAPICall(ctx, "CreateSecurityGroupRule", createSecurityGroupRuleRequest, httpRes, err)
		if err != nil {
			if httpRes != nil {
				if httpRes.StatusCode == 409 {
					return true, nil
				}
				return false, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
			}
			requestStr := fmt.Sprintf("%v", createSecurityGroupRuleRequest)
			if reconciler.KeepRetryWithError(
				requestStr,
				httpRes.StatusCode,
				reconciler.ThrottlingErrors) {
				return false, nil
			}
			return false, err
		}
		return true, err
	}
	backoff := reconciler.EnvBackoff()
	waitErr := wait.ExponentialBackoff(backoff, createSecurityGroupRuleCallBack)
	if waitErr != nil {
		return nil, waitErr
	}
	securityGroupRule, ok := securityGroupRuleResponse.GetSecurityGroupOk()
	if !ok {
		// if CreateSecurityGroupRule return 409, the response not contain the conflicted SecurityGroup.
		// workarround to a Outscale API issue
		return s.GetSecurityGroup(ctx, securityGroupId)
	}
	return securityGroupRule, nil
}

// DeleteSecurityGroupRule delete the security group rule associated with the security group and the net
func (s *Service) DeleteSecurityGroupRule(ctx context.Context, securityGroupId string, flow string, ipProtocol string, ipRange string, securityGroupMemberId string, fromPortRange int32, toPortRange int32) error {
	var rule osc.SecurityGroupRule
	if securityGroupMemberId != "" && ipRange == "" {
		securityGroupMember := osc.SecurityGroupsMember{
			SecurityGroupId: &securityGroupMemberId,
		}
		rule = osc.SecurityGroupRule{
			SecurityGroupsMembers: &[]osc.SecurityGroupsMember{securityGroupMember},
			IpProtocol:            &ipProtocol,
			FromPortRange:         &fromPortRange,
			ToPortRange:           &toPortRange,
		}
	} else {
		rule = osc.SecurityGroupRule{
			IpProtocol:    &ipProtocol,
			IpRanges:      &[]string{ipRange},
			FromPortRange: &fromPortRange,
			ToPortRange:   &toPortRange,
		}
	}

	deleteSecurityGroupRuleRequest := osc.DeleteSecurityGroupRuleRequest{
		Flow:            flow,
		SecurityGroupId: securityGroupId,
		Rules:           &[]osc.SecurityGroupRule{rule},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()

	deleteSecurityGroupCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error

		_, httpRes, err = oscApiClient.SecurityGroupRuleApi.DeleteSecurityGroupRule(oscAuthClient).DeleteSecurityGroupRuleRequest(deleteSecurityGroupRuleRequest).Execute()
		utils.LogAPICall(ctx, "DeleteSecurityGroupRule", deleteSecurityGroupRuleRequest, httpRes, err)
		if err != nil {
			if httpRes != nil {
				return false, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
			}
			requestStr := fmt.Sprintf("%v", deleteSecurityGroupRuleRequest)
			if reconciler.KeepRetryWithError(
				requestStr,
				httpRes.StatusCode,
				reconciler.ThrottlingErrors) {
				return false, nil
			}
			return false, err
		}
		return true, err
	}
	backoff := reconciler.EnvBackoff()
	waitErr := wait.ExponentialBackoff(backoff, deleteSecurityGroupCallBack)
	if waitErr != nil {
		return waitErr
	}
	return nil
}

// DeleteSecurityGroup delete the securitygroup associated with the net
func (s *Service) DeleteSecurityGroup(ctx context.Context, securityGroupId string) error {
	deleteSecurityGroupRequest := osc.DeleteSecurityGroupRequest{SecurityGroupId: &securityGroupId}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscApiClient.SecurityGroupApi.DeleteSecurityGroup(oscAuthClient).DeleteSecurityGroupRequest(deleteSecurityGroupRequest).Execute()
	utils.LogAPICall(ctx, "DeleteSecurityGroup", deleteSecurityGroupRequest, httpRes, err)
	if err != nil {
		if httpRes != nil {
			fmt.Printf("Error with http result %s", httpRes.Status)
			if httpRes.StatusCode == http.StatusConflict {
				return ErrResourceConflict
			}
			return err
		}
	}
	return nil
}

// GetSecurityGroup retrieve security group object from the security group id
func (s *Service) GetSecurityGroup(ctx context.Context, securityGroupId string) (*osc.SecurityGroup, error) {
	readSecurityGroupRequest := osc.ReadSecurityGroupsRequest{
		Filters: &osc.FiltersSecurityGroup{
			SecurityGroupIds: &[]string{securityGroupId},
		},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	var readSecurityGroupsResponse osc.ReadSecurityGroupsResponse
	readSecurityGroupsCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		readSecurityGroupsResponse, httpRes, err = oscApiClient.SecurityGroupApi.ReadSecurityGroups(oscAuthClient).ReadSecurityGroupsRequest(readSecurityGroupRequest).Execute()
		utils.LogAPICall(ctx, "ReadSecurityGroups", readSecurityGroupRequest, httpRes, err)
		if err != nil {
			if httpRes != nil {
				return false, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
			}
			requestStr := fmt.Sprintf("%v", readSecurityGroupRequest)
			if reconciler.KeepRetryWithError(
				requestStr,
				httpRes.StatusCode,
				reconciler.ThrottlingErrors) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	}
	backoff := reconciler.EnvBackoff()
	waitErr := wait.ExponentialBackoff(backoff, readSecurityGroupsCallBack)
	if waitErr != nil {
		return nil, waitErr
	}
	securitygroups, ok := readSecurityGroupsResponse.GetSecurityGroupsOk()
	if !ok {
		return nil, errors.New("Can not get securityGroup")
	}
	if len(*securitygroups) == 0 {
		return nil, nil
	} else {
		securitygroup := *securitygroups
		return &securitygroup[0], nil
	}
}

// SecurityGroupHasRule checks if a security group has a specific rule.
func (s *Service) SecurityGroupHasRule(ctx context.Context, securityGroupId string, flow string, ipProtocols string, ipRanges string, securityGroupMemberId string, fromPortRanges int32, toPortRanges int32) (bool, error) {
	var readSecurityGroupRuleRequest osc.ReadSecurityGroupsRequest
	if ipProtocols == "-1" {
		fromPortRanges = -1
		toPortRanges = -1
	}

	switch {
	case flow == "Inbound":
		readSecurityGroupRuleRequest = osc.ReadSecurityGroupsRequest{
			Filters: &osc.FiltersSecurityGroup{
				SecurityGroupIds:          &[]string{securityGroupId},
				InboundRuleProtocols:      &[]string{ipProtocols},
				InboundRuleFromPortRanges: &[]int32{fromPortRanges},
				InboundRuleToPortRanges:   &[]int32{toPortRanges},
			},
		}

	case flow == "Outbound":
		readSecurityGroupRuleRequest = osc.ReadSecurityGroupsRequest{
			Filters: &osc.FiltersSecurityGroup{
				SecurityGroupIds:           &[]string{securityGroupId},
				OutboundRuleProtocols:      &[]string{ipProtocols},
				OutboundRuleFromPortRanges: &[]int32{fromPortRanges},
				OutboundRuleToPortRanges:   &[]int32{toPortRanges},
			},
		}
	default:
		return false, errors.New("Invalid Flow")
	}

	switch {
	case securityGroupMemberId != "" && ipRanges != "":
		return false, errors.New("Get Both IpRange and securityGroupMemberId")
	case securityGroupMemberId != "" && flow == "Inbound":
		readSecurityGroupRuleRequest.Filters.SetInboundRuleSecurityGroupIds([]string{securityGroupMemberId})
	case securityGroupMemberId != "" && flow == "Outbound":
		readSecurityGroupRuleRequest.Filters.SetOutboundRuleSecurityGroupIds([]string{securityGroupMemberId})
	case ipRanges != "" && flow == "Inbound":
		readSecurityGroupRuleRequest.Filters.SetInboundRuleIpRanges([]string{ipRanges})
	case ipRanges != "" && flow == "Outbound":
		readSecurityGroupRuleRequest.Filters.SetOutboundRuleIpRanges([]string{ipRanges})
	}

	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	var readSecurityGroupRulesResponse osc.ReadSecurityGroupsResponse
	readSecurityGroupRulesCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		readSecurityGroupRulesResponse, httpRes, err = oscApiClient.SecurityGroupApi.ReadSecurityGroups(oscAuthClient).ReadSecurityGroupsRequest(readSecurityGroupRuleRequest).Execute()
		utils.LogAPICall(ctx, "ReadSecurityGroups", readSecurityGroupRuleRequest, httpRes, err)
		if err != nil {
			if httpRes != nil {
				return false, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
			}
			requestStr := fmt.Sprintf("%v", readSecurityGroupRuleRequest)
			if reconciler.KeepRetryWithError(
				requestStr,
				httpRes.StatusCode,
				reconciler.ThrottlingErrors) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	}
	backoff := reconciler.EnvBackoff()
	waitErr := wait.ExponentialBackoff(backoff, readSecurityGroupRulesCallBack)
	if waitErr != nil {
		return false, waitErr
	}
	securityGroups, ok := readSecurityGroupRulesResponse.GetSecurityGroupsOk()
	if !ok {
		return false, errors.New("Can not get securityGroup")
	}
	return len(*securityGroups) > 0, nil
}

// GetSecurityGroupIdsFromNetIds return the security group id resource that exist from the net id
func (s *Service) GetSecurityGroupIdsFromNetIds(ctx context.Context, netId string) ([]string, error) {
	readSecurityGroupRequest := osc.ReadSecurityGroupsRequest{
		Filters: &osc.FiltersSecurityGroup{
			NetIds: &[]string{netId},
		},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	var readSecurityGroupsResponse osc.ReadSecurityGroupsResponse
	readSecurityGroupsCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		readSecurityGroupsResponse, httpRes, err = oscApiClient.SecurityGroupApi.ReadSecurityGroups(oscAuthClient).ReadSecurityGroupsRequest(readSecurityGroupRequest).Execute()
		utils.LogAPICall(ctx, "ReadSecurityGroups", readSecurityGroupRequest, httpRes, err)
		if err != nil {
			if httpRes != nil {
				return false, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
			}
			requestStr := fmt.Sprintf("%v", readSecurityGroupRequest)
			if reconciler.KeepRetryWithError(
				requestStr,
				httpRes.StatusCode,
				reconciler.ThrottlingErrors) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	}
	backoff := reconciler.EnvBackoff()
	waitErr := wait.ExponentialBackoff(backoff, readSecurityGroupsCallBack)
	if waitErr != nil {
		return nil, waitErr
	}
	var securityGroupIds []string
	securityGroups, ok := readSecurityGroupsResponse.GetSecurityGroupsOk()
	if !ok {
		return nil, errors.New("Can not get securityGroup")
	}
	if len(*securityGroups) != 0 {
		for _, securityGroup := range *securityGroups {
			securityGroupId := securityGroup.GetSecurityGroupId()
			securityGroupIds = append(securityGroupIds, securityGroupId)
		}
	}
	return securityGroupIds, nil
}
