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
	"errors"
	"fmt"
	"net/http"

	osc "github.com/outscale/osc-sdk-go/v2"
)

//go:generate ../../../bin/mockgen -destination mock_security/securitygroup_mock.go -package mock_security -source ./securitygroup.go
type OscSecurityGroupInterface interface {
	CreateSecurityGroup(netID string, securityGroupName string, securityGroupDescription string) (*osc.SecurityGroup, error)
	CreateSecurityGroupRule(securityGroupID string, flow string, ipProtocol string, ipRange string, securityGroupMemberID string, fromPortRange int32, toPortRange int32) (*osc.SecurityGroup, error)
	DeleteSecurityGroupRule(securityGroupID string, flow string, ipProtocol string, ipRange string, securityGroupMemberID string, fromPortRange int32, toPortRange int32) error
	DeleteSecurityGroup(securityGroupID string) (*http.Response, error)
	GetSecurityGroup(securityGroupID string) (*osc.SecurityGroup, error)
	GetSecurityGroupFromSecurityGroupRule(securityGroupID string, Flow string, IPProtocols string, IPRanges string, FromPortRanges int32, ToPortRanges int32) (*osc.SecurityGroup, error)
	GetSecurityGroupIdsFromNetIds(netID string) ([]string, error)
}

// CreateSecurityGroup create the securitygroup associated with the net.
func (s *Service) CreateSecurityGroup(netID string, securityGroupName string, securityGroupDescription string) (*osc.SecurityGroup, error) {
	securityGroupRequest := osc.CreateSecurityGroupRequest{
		SecurityGroupName: securityGroupName,
		Description:       securityGroupDescription,
		NetId:             &netID,
	}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	securityGroupResponse, httpRes, err := oscAPIClient.SecurityGroupApi.CreateSecurityGroup(oscAuthClient).CreateSecurityGroupRequest(securityGroupRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	securityGroup, ok := securityGroupResponse.GetSecurityGroupOk()
	if !ok {
		return nil, errors.New("can not create securitygroup")
	}
	return securityGroup, nil
}

// CreateSecurityGroupRule create the security group rule associated with the security group and the net.
func (s *Service) CreateSecurityGroupRule(securityGroupID string, flow string, ipProtocol string, ipRange string, securityGroupMemberID string, fromPortRange int32, toPortRange int32) (*osc.SecurityGroup, error) {
	var rule osc.SecurityGroupRule
	if securityGroupMemberID != "" && ipRange == "" {
		securityGroupMember := osc.SecurityGroupsMember{
			SecurityGroupId: &securityGroupMemberID,
		}
		rule = osc.SecurityGroupRule{
			SecurityGroupsMembers: &[]osc.SecurityGroupsMember{securityGroupMember},
			IpProtocol:            &ipProtocol,
			FromPortRange:         &fromPortRange,
			ToPortRange:           &toPortRange,
		}
	} else if securityGroupMemberID != "" && ipRange != "" {
		return nil, errors.New("get Both ipRange and securityGroupMemberID")
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
		SecurityGroupId: securityGroupID,
		Rules:           &[]osc.SecurityGroupRule{rule},
	}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	securityGroupRuleResponse, httpRes, err := oscAPIClient.SecurityGroupRuleApi.CreateSecurityGroupRule(oscAuthClient).CreateSecurityGroupRuleRequest(createSecurityGroupRuleRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	securityGroupRule, ok := securityGroupRuleResponse.GetSecurityGroupOk()
	if !ok {
		return nil, errors.New("can not get securityGroup")
	}
	return securityGroupRule, nil
}

// DeleteSecurityGroupRule delete the security group rule associated with the security group and the net.
func (s *Service) DeleteSecurityGroupRule(securityGroupID string, flow string, ipProtocol string, ipRange string, securityGroupMemberID string, fromPortRange int32, toPortRange int32) error {
	var rule osc.SecurityGroupRule
	if securityGroupMemberID != "" && ipRange == "" {
		securityGroupMember := osc.SecurityGroupsMember{
			SecurityGroupId: &securityGroupMemberID,
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
		SecurityGroupId: securityGroupID,
		Rules:           &[]osc.SecurityGroupRule{rule},
	}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscAPIClient.SecurityGroupRuleApi.DeleteSecurityGroupRule(oscAuthClient).DeleteSecurityGroupRuleRequest(deleteSecurityGroupRuleRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return err
	}
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	return nil
}

// DeleteSecurityGroup delete the securitygroup associated with the net.
func (s *Service) DeleteSecurityGroup(securityGroupID string) (*http.Response, error) {
	deleteSecurityGroupRequest := osc.DeleteSecurityGroupRequest{SecurityGroupId: &securityGroupID}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscAPIClient.SecurityGroupApi.DeleteSecurityGroup(oscAuthClient).DeleteSecurityGroupRequest(deleteSecurityGroupRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return httpRes, err
	}
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	return httpRes, nil
}

// GetSecurityGroup retrieve security group object from the security group id.
func (s *Service) GetSecurityGroup(securityGroupID string) (*osc.SecurityGroup, error) {
	readSecurityGroupRequest := osc.ReadSecurityGroupsRequest{
		Filters: &osc.FiltersSecurityGroup{
			SecurityGroupIds: &[]string{securityGroupID},
		},
	}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	readSecurityGroupsResponse, httpRes, err := oscAPIClient.SecurityGroupApi.ReadSecurityGroups(oscAuthClient).ReadSecurityGroupsRequest(readSecurityGroupRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	securitygroups, ok := readSecurityGroupsResponse.GetSecurityGroupsOk()
	if !ok {
		return nil, errors.New("can not get securityGroup")
	}
	if len(*securitygroups) == 0 {
		return nil, nil
	}
	securitygroup := *securitygroups
	return &securitygroup[0], nil
}

// GetSecurityGroupFromSecurityGroupRule retrieve security group rule object from the security group id.
func (s *Service) GetSecurityGroupFromSecurityGroupRule(securityGroupID string, flow string, ipProtocols string, ipRanges string, fromPortRanges int32, toPortRanges int32) (*osc.SecurityGroup, error) {
	var readSecurityGroupRuleRequest osc.ReadSecurityGroupsRequest
	switch {
	case flow == "Inbound":
		readSecurityGroupRuleRequest = osc.ReadSecurityGroupsRequest{
			Filters: &osc.FiltersSecurityGroup{
				SecurityGroupIds:          &[]string{securityGroupID},
				InboundRuleProtocols:      &[]string{ipProtocols},
				InboundRuleIpRanges:       &[]string{ipRanges},
				InboundRuleFromPortRanges: &[]int32{fromPortRanges},
				InboundRuleToPortRanges:   &[]int32{toPortRanges},
			},
		}
	case flow == "Outbound":
		readSecurityGroupRuleRequest = osc.ReadSecurityGroupsRequest{
			Filters: &osc.FiltersSecurityGroup{
				SecurityGroupIds:           &[]string{securityGroupID},
				OutboundRuleProtocols:      &[]string{ipProtocols},
				OutboundRuleIpRanges:       &[]string{ipRanges},
				OutboundRuleFromPortRanges: &[]int32{fromPortRanges},
				OutboundRuleToPortRanges:   &[]int32{toPortRanges},
			},
		}
	default:
		return nil, errors.New("invalid Flow")
	}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	readSecurityGroupRuleResponse, httpRes, err := oscAPIClient.SecurityGroupApi.ReadSecurityGroups(oscAuthClient).ReadSecurityGroupsRequest(readSecurityGroupRuleRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	securityGroups, ok := readSecurityGroupRuleResponse.GetSecurityGroupsOk()
	if !ok {
		return nil, errors.New("can not get securityGroup")
	}
	if len(*securityGroups) == 0 {
		return nil, nil
	}
	securityGroup := *securityGroups
	return &securityGroup[0], nil
}

// GetSecurityGroupIdsFromNetIds return the security group id resource that exist from the net id.
func (s *Service) GetSecurityGroupIdsFromNetIds(netID string) ([]string, error) {
	readSecurityGroupRequest := osc.ReadSecurityGroupsRequest{
		Filters: &osc.FiltersSecurityGroup{
			NetIds: &[]string{netID},
		},
	}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	readSecurityGroupsResponse, httpRes, err := oscAPIClient.SecurityGroupApi.ReadSecurityGroups(oscAuthClient).ReadSecurityGroupsRequest(readSecurityGroupRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}

	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	var securityGroupIDs []string
	securityGroups, ok := readSecurityGroupsResponse.GetSecurityGroupsOk()
	if !ok {
		return nil, errors.New("can not get securityGroup")
	}
	if len(*securityGroups) != 0 {
		for _, securityGroup := range *securityGroups {
			securityGroupID := securityGroup.GetSecurityGroupId()
			securityGroupIDs = append(securityGroupIDs, securityGroupID)
		}
	}
	return securityGroupIDs, nil
}
