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
	"fmt"
	"net/http"

	"errors"

	tag "github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag"

	osc "github.com/outscale/osc-sdk-go/v2"
)

//go:generate ../../../bin/mockgen -destination mock_security/securitygroup_mock.go -package mock_security -source ./securitygroup.go

type OscSecurityGroupInterface interface {
	CreateSecurityGroup(netId string, clusterName string, securityGroupName string, securityGroupDescription string, securityGroupTag string) (*osc.SecurityGroup, error)
	CreateSecurityGroupRule(securityGroupId string, flow string, ipProtocol string, ipRange string, securityGroupMemberId string, fromPortRange int32, toPortRange int32) (*osc.SecurityGroup, error)
	DeleteSecurityGroupRule(securityGroupId string, flow string, ipProtocol string, ipRange string, securityGroupMemberId string, fromPortRange int32, toPortRange int32) error
	DeleteSecurityGroup(securityGroupId string) (error, *http.Response)
	GetSecurityGroup(securityGroupId string) (*osc.SecurityGroup, error)
	GetSecurityGroupFromSecurityGroupRule(securityGroupId string, Flow string, IpProtocols string, IpRanges string, securityGroupMemberId string, FromPortRanges int32, ToPortRanges int32) (*osc.SecurityGroup, error)
	GetSecurityGroupIdsFromNetIds(netId string) ([]string, error)
}

// CreateSecurityGroup create the securitygroup associated with the net
func (s *Service) CreateSecurityGroup(netId string, clusterName string, securityGroupName string, securityGroupDescription string, securityGroupTag string) (*osc.SecurityGroup, error) {
	securityGroupRequest := osc.CreateSecurityGroupRequest{
		SecurityGroupName: securityGroupName,
		Description:       securityGroupDescription,
		NetId:             &netId,
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	securityGroupResponse, httpRes, err := oscApiClient.SecurityGroupApi.CreateSecurityGroup(oscAuthClient).CreateSecurityGroupRequest(securityGroupRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	securityGroup, ok := securityGroupResponse.GetSecurityGroupOk()
	if !ok {
		return nil, errors.New("Can not create securitygroup")
	}
	resourceIds := []string{*securityGroupResponse.SecurityGroup.SecurityGroupId}
	err = tag.AddTag("OscK8sClusterID/"+clusterName, "owned", resourceIds, oscApiClient, oscAuthClient)
	if err != nil {
		if httpRes != nil {
			return nil, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
		} else {
			return nil, err
		}
	}
	if securityGroupTag == "OscK8sMainSG" {
		err = tag.AddTag("OscK8sMainSG/"+clusterName, "True", resourceIds, oscApiClient, oscAuthClient)
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
func (s *Service) CreateSecurityGroupRule(securityGroupId string, flow string, ipProtocol string, ipRange string, securityGroupMemberId string, fromPortRange int32, toPortRange int32) (*osc.SecurityGroup, error) {
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
	securityGroupRuleResponse, httpRes, err := oscApiClient.SecurityGroupRuleApi.CreateSecurityGroupRule(oscAuthClient).CreateSecurityGroupRuleRequest(createSecurityGroupRuleRequest).Execute()
	if err != nil {
		if httpRes != nil {
			return nil, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
		} else {
			return nil, err
		}
	}
	securityGroupRule, ok := securityGroupRuleResponse.GetSecurityGroupOk()
	if !ok {
		return nil, errors.New("Can not get securityGroup")
	}
	return securityGroupRule, nil
}

// DeleteSecurityGroupRule delete the security group rule associated with the security group and the net
func (s *Service) DeleteSecurityGroupRule(securityGroupId string, flow string, ipProtocol string, ipRange string, securityGroupMemberId string, fromPortRange int32, toPortRange int32) error {
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
	_, httpRes, err := oscApiClient.SecurityGroupRuleApi.DeleteSecurityGroupRule(oscAuthClient).DeleteSecurityGroupRuleRequest(deleteSecurityGroupRuleRequest).Execute()
	if err != nil {
		if httpRes != nil {
			return fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
		} else {
			return err
		}
	}
	return nil
}

// DeleteSecurityGroup delete the securitygroup associated with the net
func (s *Service) DeleteSecurityGroup(securityGroupId string) (error, *http.Response) {
	deleteSecurityGroupRequest := osc.DeleteSecurityGroupRequest{SecurityGroupId: &securityGroupId}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscApiClient.SecurityGroupApi.DeleteSecurityGroup(oscAuthClient).DeleteSecurityGroupRequest(deleteSecurityGroupRequest).Execute()
	if err != nil {
		if httpRes != nil {
			fmt.Printf("Error with http result %s", httpRes.Status)
			return err, httpRes
		}
	}
	return nil, httpRes
}

// GetSecurityGroup retrieve security group object from the security group id
func (s *Service) GetSecurityGroup(securityGroupId string) (*osc.SecurityGroup, error) {
	readSecurityGroupRequest := osc.ReadSecurityGroupsRequest{
		Filters: &osc.FiltersSecurityGroup{
			SecurityGroupIds: &[]string{securityGroupId},
		},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	readSecurityGroupsResponse, httpRes, err := oscApiClient.SecurityGroupApi.ReadSecurityGroups(oscAuthClient).ReadSecurityGroupsRequest(readSecurityGroupRequest).Execute()
	if err != nil {
		if httpRes != nil {
			return nil, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
		} else {
			return nil, err
		}
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

// GetSecurityGroupFromSecurityGroupRule retrieve security group rule object from the security group id
func (s *Service) GetSecurityGroupFromSecurityGroupRule(securityGroupId string, flow string, ipProtocols string, ipRanges string, securityGroupMemberId string, fromPortRanges int32, toPortRanges int32) (*osc.SecurityGroup, error) {
	var readSecurityGroupRuleRequest osc.ReadSecurityGroupsRequest
	switch {
	case flow == "Inbound":
		readSecurityGroupRuleRequest = osc.ReadSecurityGroupsRequest{
			Filters: &osc.FiltersSecurityGroup{
				SecurityGroupIds:          &[]string{securityGroupId},
				InboundRuleProtocols:      &[]string{ipProtocols},
				InboundRuleIpRanges:       &[]string{ipRanges},
				InboundRuleFromPortRanges: &[]int32{fromPortRanges},
				InboundRuleToPortRanges:   &[]int32{toPortRanges},
			},
		}

	case flow == "Outbound":
		readSecurityGroupRuleRequest = osc.ReadSecurityGroupsRequest{
			Filters: &osc.FiltersSecurityGroup{
				SecurityGroupIds:           &[]string{securityGroupId},
				OutboundRuleProtocols:      &[]string{ipProtocols},
				OutboundRuleIpRanges:       &[]string{ipRanges},
				OutboundRuleFromPortRanges: &[]int32{fromPortRanges},
				OutboundRuleToPortRanges:   &[]int32{toPortRanges},
			},
		}
	default:
		return nil, errors.New("Invalid Flow")
	}

	if securityGroupMemberId != "" && ipRanges == "" && flow == "Inbound" {
		readSecurityGroupRuleRequest.Filters.SetInboundRuleSecurityGroupIds([]string{securityGroupMemberId})
	} else if securityGroupMemberId != "" && ipRanges == "" && flow == "Outbound" {
		readSecurityGroupRuleRequest.Filters.SetOutboundRuleSecurityGroupIds([]string{securityGroupMemberId})
	} else if securityGroupMemberId != "" && ipRanges != "" {
		return nil, errors.New("Get Both IpRange and securityGroupMemberId")
	} else {
		fmt.Printf("Have IpRange and no securityGroupMemberId\n")
	}

	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	readSecurityGroupRuleResponse, httpRes, err := oscApiClient.SecurityGroupApi.ReadSecurityGroups(oscAuthClient).ReadSecurityGroupsRequest(readSecurityGroupRuleRequest).Execute()
	if err != nil {
		if httpRes != nil {
			return nil, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
		} else {
			return nil, err
		}
	}
	securityGroups, ok := readSecurityGroupRuleResponse.GetSecurityGroupsOk()
	if !ok {
		return nil, errors.New("Can not get securityGroup")
	}
	if len(*securityGroups) == 0 {
		return nil, nil
	} else {
		securityGroup := *securityGroups
		return &securityGroup[0], nil
	}
}

// GetSecurityGroupIdsFromNetIds return the security group id resource that exist from the net id
func (s *Service) GetSecurityGroupIdsFromNetIds(netId string) ([]string, error) {
	readSecurityGroupRequest := osc.ReadSecurityGroupsRequest{
		Filters: &osc.FiltersSecurityGroup{
			NetIds: &[]string{netId},
		},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	readSecurityGroupsResponse, httpRes, err := oscApiClient.SecurityGroupApi.ReadSecurityGroups(oscAuthClient).ReadSecurityGroupsRequest(readSecurityGroupRequest).Execute()
	if err != nil {
		if httpRes != nil {
			return nil, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
		} else {
			return nil, err
		}
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
