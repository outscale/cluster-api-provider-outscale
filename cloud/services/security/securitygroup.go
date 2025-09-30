/*
SPDX-FileCopyrightText: 2022 The Kubernetes Authors

SPDX-License-Identifier: Apache-2.0
*/

package security

import (
	"context"
	"errors"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	tag "github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	"github.com/outscale/cluster-api-provider-outscale/cloud/utils"
	osc "github.com/outscale/osc-sdk-go/v2"
)

// ErrResourceConflict is returned by DeleteSecurityGroup when SG cannot be deleted because another resource requires it.
var ErrResourceConflict = errors.New("conflict with existing resource")

//go:generate ../../../bin/mockgen -destination mock_security/securitygroup_mock.go -package mock_security -source ./securitygroup.go

type OscSecurityGroupInterface interface {
	CreateSecurityGroup(ctx context.Context, netId, clusterID, securityGroupName, securityGroupDescription, securityGroupTag string, roles []infrastructurev1beta1.OscRole) (*osc.SecurityGroup, error)
	CreateSecurityGroupRule(ctx context.Context, securityGroupId, flow, ipProtocol, ipRange, securityGroupMemberId string, fromPortRange int32, toPortRange int32) (*osc.SecurityGroup, error)
	DeleteSecurityGroupRule(ctx context.Context, securityGroupId, flow, ipProtocol, ipRange, securityGroupMemberId string, fromPortRange int32, toPortRange int32) error
	DeleteSecurityGroup(ctx context.Context, securityGroupId string) error
	GetSecurityGroup(ctx context.Context, securityGroupId string) (*osc.SecurityGroup, error)
	SecurityGroupHasRule(ctx context.Context, securityGroupId, flow, ipProtocols, ipRanges, securityGroupMemberId string, fromPortRanges int32, toPortRanges int32) (bool, error)
	GetSecurityGroupsFromNet(ctx context.Context, netId string) ([]osc.SecurityGroup, error)
	GetSecurityGroupFromName(ctx context.Context, name string) (*osc.SecurityGroup, error)
}

// CreateSecurityGroup create the securitygroup associated with the net
func (s *Service) CreateSecurityGroup(ctx context.Context, netId string, clusterID string, securityGroupName string, securityGroupDescription string, securityGroupTag string, roles []infrastructurev1beta1.OscRole) (*osc.SecurityGroup, error) {
	securityGroupRequest := osc.CreateSecurityGroupRequest{
		SecurityGroupName: securityGroupName,
		Description:       securityGroupDescription,
		NetId:             &netId,
	}

	securityGroupResponse, httpRes, err := s.tenant.Client().SecurityGroupApi.CreateSecurityGroup(s.tenant.ContextWithAuth(ctx)).CreateSecurityGroupRequest(securityGroupRequest).Execute()
	err = utils.LogAndExtractError(ctx, "CreateSecurityGroup", securityGroupRequest, httpRes, err)
	if err != nil {
		return nil, err
	}
	securityGroup, ok := securityGroupResponse.GetSecurityGroupOk()
	if !ok {
		return nil, errors.New("cannot create securitygroup")
	}
	resourceIds := []string{*securityGroupResponse.SecurityGroup.SecurityGroupId}
	clusterTag := osc.ResourceTag{
		Key:   "OscK8sClusterID/" + clusterID,
		Value: "owned",
	}
	clusterSecurityGroupRequest := osc.CreateTagsRequest{
		ResourceIds: resourceIds,
		Tags:        append(utils.RoleTags(roles), clusterTag),
	}
	err = tag.AddTag(ctx, clusterSecurityGroupRequest, resourceIds, s.tenant.Client(), s.tenant.ContextWithAuth(ctx))
	if err != nil {
		return nil, err
	}
	if securityGroupTag == "OscK8sMainSG" {
		mainTag := osc.ResourceTag{
			Key:   "OscK8sMainSG/" + clusterID,
			Value: "True",
		}
		mainSecurityGroupTagRequest := osc.CreateTagsRequest{
			ResourceIds: resourceIds,
			Tags:        []osc.ResourceTag{mainTag},
		}
		err := tag.AddTag(ctx, mainSecurityGroupTagRequest, resourceIds, s.tenant.Client(), s.tenant.ContextWithAuth(ctx))
		if err != nil {
			return nil, err
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

	securityGroupRuleResponse, httpRes, err := s.tenant.Client().SecurityGroupRuleApi.CreateSecurityGroupRule(s.tenant.ContextWithAuth(ctx)).CreateSecurityGroupRuleRequest(createSecurityGroupRuleRequest).Execute()
	err = utils.LogAndExtractError(ctx, "CreateSecurityGroupRule", createSecurityGroupRuleRequest, httpRes, err)
	if err != nil {
		return nil, err
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

	_, httpRes, err := s.tenant.Client().SecurityGroupRuleApi.DeleteSecurityGroupRule(s.tenant.ContextWithAuth(ctx)).DeleteSecurityGroupRuleRequest(deleteSecurityGroupRuleRequest).Execute()
	err = utils.LogAndExtractError(ctx, "DeleteSecurityGroupRule", deleteSecurityGroupRuleRequest, httpRes, err)
	return err
}

// DeleteSecurityGroup delete the securitygroup associated with the net
func (s *Service) DeleteSecurityGroup(ctx context.Context, securityGroupId string) error {
	deleteSecurityGroupRequest := osc.DeleteSecurityGroupRequest{SecurityGroupId: &securityGroupId}

	_, httpRes, err := s.tenant.Client().SecurityGroupApi.DeleteSecurityGroup(s.tenant.ContextWithAuth(ctx)).DeleteSecurityGroupRequest(deleteSecurityGroupRequest).Execute()
	err = utils.LogAndExtractError(ctx, "DeleteSecurityGroup", deleteSecurityGroupRequest, httpRes, err)
	return err
}

// GetSecurityGroup retrieve security group object from the security group id
func (s *Service) GetSecurityGroup(ctx context.Context, securityGroupId string) (*osc.SecurityGroup, error) {
	readSecurityGroupRequest := osc.ReadSecurityGroupsRequest{
		Filters: &osc.FiltersSecurityGroup{
			SecurityGroupIds: &[]string{securityGroupId},
		},
	}

	readSecurityGroupsResponse, httpRes, err := s.tenant.Client().SecurityGroupApi.ReadSecurityGroups(s.tenant.ContextWithAuth(ctx)).ReadSecurityGroupsRequest(readSecurityGroupRequest).Execute()
	err = utils.LogAndExtractError(ctx, "ReadSecurityGroups", readSecurityGroupRequest, httpRes, err)
	if err != nil {
		return nil, err
	}
	securitygroups, ok := readSecurityGroupsResponse.GetSecurityGroupsOk()
	if !ok {
		return nil, errors.New("cannot get securityGroup")
	}
	if len(*securitygroups) == 0 {
		return nil, nil
	} else {
		securitygroup := *securitygroups
		return &securitygroup[0], nil
	}
}

// GetSecurityGroupFromName retrieve security group object from the security group id
func (s *Service) GetSecurityGroupFromName(ctx context.Context, name string) (*osc.SecurityGroup, error) {
	readSecurityGroupRequest := osc.ReadSecurityGroupsRequest{
		Filters: &osc.FiltersSecurityGroup{
			SecurityGroupNames: &[]string{name},
		},
	}

	readSecurityGroupsResponse, httpRes, err := s.tenant.Client().SecurityGroupApi.ReadSecurityGroups(s.tenant.ContextWithAuth(ctx)).ReadSecurityGroupsRequest(readSecurityGroupRequest).Execute()
	err = utils.LogAndExtractError(ctx, "ReadSecurityGroups", readSecurityGroupRequest, httpRes, err)
	if err != nil {
		return nil, err
	}
	securitygroups, ok := readSecurityGroupsResponse.GetSecurityGroupsOk()
	if !ok {
		return nil, errors.New("cannot get securityGroup")
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

	switch flow {
	case "Inbound":
		readSecurityGroupRuleRequest = osc.ReadSecurityGroupsRequest{
			Filters: &osc.FiltersSecurityGroup{
				SecurityGroupIds:          &[]string{securityGroupId},
				InboundRuleProtocols:      &[]string{ipProtocols},
				InboundRuleFromPortRanges: &[]int32{fromPortRanges},
				InboundRuleToPortRanges:   &[]int32{toPortRanges},
			},
		}

	case "Outbound":
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

	readSecurityGroupRulesResponse, httpRes, err := s.tenant.Client().SecurityGroupApi.ReadSecurityGroups(s.tenant.ContextWithAuth(ctx)).ReadSecurityGroupsRequest(readSecurityGroupRuleRequest).Execute()
	err = utils.LogAndExtractError(ctx, "ReadSecurityGroups", readSecurityGroupRuleRequest, httpRes, err)
	if err != nil {
		return false, err
	}
	securityGroups, ok := readSecurityGroupRulesResponse.GetSecurityGroupsOk()
	if !ok {
		return false, errors.New("cannot get securityGroup")
	}
	return len(*securityGroups) > 0, nil
}

// GetSecurityGroupsFromNet return the security group id resource that exist from the net id
func (s *Service) GetSecurityGroupsFromNet(ctx context.Context, netId string) ([]osc.SecurityGroup, error) {
	readSecurityGroupRequest := osc.ReadSecurityGroupsRequest{
		Filters: &osc.FiltersSecurityGroup{
			NetIds: &[]string{netId},
		},
	}

	readSecurityGroupsResponse, httpRes, err := s.tenant.Client().SecurityGroupApi.ReadSecurityGroups(s.tenant.ContextWithAuth(ctx)).ReadSecurityGroupsRequest(readSecurityGroupRequest).Execute()
	err = utils.LogAndExtractError(ctx, "ReadSecurityGroups", readSecurityGroupRequest, httpRes, err)
	if err != nil {
		return nil, err
	}
	return readSecurityGroupsResponse.GetSecurityGroups(), nil
}
