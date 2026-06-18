/*
SPDX-FileCopyrightText: 2022 The Kubernetes Authors

SPDX-License-Identifier: Apache-2.0
*/

package compute

import (
	"context"
	"errors"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/utils"
	"github.com/outscale/osc-sdk-go/v3/pkg/osc"
)

// ErrResourceConflict is returned by DeleteSecurityGroup when SG cannot be deleted because another resource requires it.
var ErrResourceConflict = errors.New("conflict with existing resource")

type SecurityGroupInterface interface {
	CreateSecurityGroup(ctx context.Context, netId, clusterID, securityGroupName, securityGroupDescription, securityGroupTag string, roles []infrastructurev1beta1.OscRole) (*osc.SecurityGroup, error)
	CreateSecurityGroupRule(ctx context.Context, securityGroupId, flow, ipProtocol, ipRange, securityGroupMemberId string, fromPortRange, toPortRange int) (*osc.SecurityGroup, error)
	DeleteSecurityGroupRule(ctx context.Context, securityGroupId, flow, ipProtocol, ipRange, securityGroupMemberId string, fromPortRange, toPortRange int) error
	DeleteSecurityGroup(ctx context.Context, securityGroupId string) error
	GetSecurityGroup(ctx context.Context, securityGroupId string) (*osc.SecurityGroup, error)
	SecurityGroupHasRule(ctx context.Context, securityGroupId, flow, ipProtocols, ipRanges, securityGroupMemberId string, fromPortRanges, toPortRanges int) (bool, error)
	GetSecurityGroupsFromNet(ctx context.Context, netId string) ([]osc.SecurityGroup, error)
	GetSecurityGroupFromName(ctx context.Context, name string) (*osc.SecurityGroup, error)
}

// CreateSecurityGroup create the securitygroup associated with the net
func (s *Service) CreateSecurityGroup(ctx context.Context, netId string, clusterID string, securityGroupName string, securityGroupDescription string, securityGroupTag string, roles []infrastructurev1beta1.OscRole) (*osc.SecurityGroup, error) {
	req := osc.CreateSecurityGroupRequest{
		SecurityGroupName: securityGroupName,
		Description:       securityGroupDescription,
		NetId:             &netId,
	}

	resp, err := s.tenant.Client().CreateSecurityGroup(ctx, req)
	if err != nil {
		return nil, err
	}
	resourceIds := []string{resp.SecurityGroup.SecurityGroupId}
	clusterTag := osc.ResourceTag{
		Key:   "OscK8sClusterID/" + clusterID,
		Value: "owned",
	}
	clusterSecurityGroupRequest := osc.CreateTagsRequest{
		ResourceIds: resourceIds,
		Tags:        append(utils.RoleTags(roles), clusterTag),
	}
	err = s.tags.AddTag(ctx, clusterSecurityGroupRequest, resourceIds)
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
		err := s.tags.AddTag(ctx, mainSecurityGroupTagRequest, resourceIds)
		if err != nil {
			return nil, err
		}
	}

	return resp.SecurityGroup, nil
}

// CreateSecurityGroupRule create the security group rule associated with the security group and the net
func (s *Service) CreateSecurityGroupRule(ctx context.Context, securityGroupId string, flow string, ipProtocol string, ipRange string, securityGroupMemberId string, fromPortRange int, toPortRange int) (*osc.SecurityGroup, error) {
	var rule osc.SecurityGroupRule
	switch {
	case securityGroupMemberId != "" && ipRange == "":
		securityGroupMember := osc.SecurityGroupsMember{
			SecurityGroupId: securityGroupMemberId,
		}
		rule = osc.SecurityGroupRule{
			SecurityGroupsMembers: []osc.SecurityGroupsMember{securityGroupMember},
			IpProtocol:            ipProtocol,
			FromPortRange:         fromPortRange,
			ToPortRange:           toPortRange,
		}
	case securityGroupMemberId != "" && ipRange != "":
		return nil, errors.New("ipRange and securityGroupMemberId are incompatible")
	default:
		rule = osc.SecurityGroupRule{
			IpProtocol:    ipProtocol,
			IpRanges:      []string{ipRange},
			FromPortRange: fromPortRange,
			ToPortRange:   toPortRange,
		}
	}
	req := osc.CreateSecurityGroupRuleRequest{
		Flow:            flow,
		SecurityGroupId: securityGroupId,
		Rules:           []osc.SecurityGroupRule{rule},
	}

	resp, err := s.tenant.Client().CreateSecurityGroupRule(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.SecurityGroup, nil
}

// DeleteSecurityGroupRule delete the security group rule associated with the security group and the net
func (s *Service) DeleteSecurityGroupRule(ctx context.Context, securityGroupId string, flow string, ipProtocol string, ipRange string, securityGroupMemberId string, fromPortRange int, toPortRange int) error {
	var rule osc.SecurityGroupRule
	if securityGroupMemberId != "" && ipRange == "" {
		securityGroupMember := osc.SecurityGroupsMember{
			SecurityGroupId: securityGroupMemberId,
		}
		rule = osc.SecurityGroupRule{
			SecurityGroupsMembers: []osc.SecurityGroupsMember{securityGroupMember},
			IpProtocol:            ipProtocol,
			FromPortRange:         fromPortRange,
			ToPortRange:           toPortRange,
		}
	} else {
		rule = osc.SecurityGroupRule{
			IpProtocol:    ipProtocol,
			IpRanges:      []string{ipRange},
			FromPortRange: fromPortRange,
			ToPortRange:   toPortRange,
		}
	}

	req := osc.DeleteSecurityGroupRuleRequest{
		Flow:            flow,
		SecurityGroupId: securityGroupId,
		Rules:           []osc.SecurityGroupRule{rule},
	}

	_, err := s.tenant.Client().DeleteSecurityGroupRule(ctx, req)
	return err
}

// DeleteSecurityGroup delete the securitygroup associated with the net
func (s *Service) DeleteSecurityGroup(ctx context.Context, securityGroupId string) error {
	req := osc.DeleteSecurityGroupRequest{SecurityGroupId: &securityGroupId}

	_, err := s.tenant.Client().DeleteSecurityGroup(ctx, req)
	return err
}

// GetSecurityGroup retrieve security group object from the security group id
func (s *Service) GetSecurityGroup(ctx context.Context, securityGroupId string) (*osc.SecurityGroup, error) {
	req := osc.ReadSecurityGroupsRequest{
		Filters: &osc.FiltersSecurityGroup{
			SecurityGroupIds: &[]string{securityGroupId},
		},
	}

	resp, err := s.tenant.Client().ReadSecurityGroups(ctx, req)
	switch {
	case err != nil:
		return nil, err
	case len(*resp.SecurityGroups) == 0:
		return nil, nil
	default:
		return &(*resp.SecurityGroups)[0], nil
	}
}

// GetSecurityGroupFromName retrieve security group object from the security group id
func (s *Service) GetSecurityGroupFromName(ctx context.Context, name string) (*osc.SecurityGroup, error) {
	req := osc.ReadSecurityGroupsRequest{
		Filters: &osc.FiltersSecurityGroup{
			SecurityGroupNames: &[]string{name},
		},
	}

	resp, err := s.tenant.Client().ReadSecurityGroups(ctx, req)
	switch {
	case err != nil:
		return nil, err
	case len(*resp.SecurityGroups) == 0:
		return nil, nil
	default:
		return &(*resp.SecurityGroups)[0], nil
	}
}

// SecurityGroupHasRule checks if a security group has a specific rule.
func (s *Service) SecurityGroupHasRule(ctx context.Context, securityGroupId string, flow string, ipProtocols string, ipRanges string, securityGroupMemberId string, fromPortRanges, toPortRanges int) (bool, error) {
	var req osc.ReadSecurityGroupsRequest
	if ipProtocols == "-1" {
		fromPortRanges = -1
		toPortRanges = -1
	}

	switch flow {
	case "Inbound":
		req = osc.ReadSecurityGroupsRequest{
			Filters: &osc.FiltersSecurityGroup{
				SecurityGroupIds:          &[]string{securityGroupId},
				InboundRuleProtocols:      &[]string{ipProtocols},
				InboundRuleFromPortRanges: &[]int{fromPortRanges},
				InboundRuleToPortRanges:   &[]int{toPortRanges},
			},
		}
	case "Outbound":
		req = osc.ReadSecurityGroupsRequest{
			Filters: &osc.FiltersSecurityGroup{
				SecurityGroupIds:           &[]string{securityGroupId},
				OutboundRuleProtocols:      &[]string{ipProtocols},
				OutboundRuleFromPortRanges: &[]int{fromPortRanges},
				OutboundRuleToPortRanges:   &[]int{toPortRanges},
			},
		}
	default:
		return false, errors.New("invalid Flow")
	}

	switch {
	case securityGroupMemberId != "" && ipRanges != "":
		return false, errors.New("ipRange and securityGroupMemberId are incompatible")
	case securityGroupMemberId != "" && flow == "Inbound":
		req.Filters.InboundRuleSecurityGroupIds = &[]string{securityGroupMemberId}
	case securityGroupMemberId != "" && flow == "Outbound":
		req.Filters.OutboundRuleSecurityGroupIds = &[]string{securityGroupMemberId}
	case ipRanges != "" && flow == "Inbound":
		req.Filters.InboundRuleIpRanges = &[]string{ipRanges}
	case ipRanges != "" && flow == "Outbound":
		req.Filters.OutboundRuleIpRanges = &[]string{ipRanges}
	}

	resp, err := s.tenant.Client().ReadSecurityGroups(ctx, req)
	if err != nil {
		return false, err
	}
	return len(*resp.SecurityGroups) > 0, nil
}

// GetSecurityGroupsFromNet return the security group id resource that exist from the net id
func (s *Service) GetSecurityGroupsFromNet(ctx context.Context, netId string) ([]osc.SecurityGroup, error) {
	req := osc.ReadSecurityGroupsRequest{
		Filters: &osc.FiltersSecurityGroup{
			NetIds: &[]string{netId},
		},
	}
	resp, err := s.tenant.Client().ReadSecurityGroups(ctx, req)
	if err != nil {
		return nil, err
	}
	return *resp.SecurityGroups, nil
}
