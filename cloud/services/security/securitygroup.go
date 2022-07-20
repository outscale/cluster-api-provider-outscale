package security

import (
	"fmt"
	"net/http"
	"regexp"

	"errors"

	osc "github.com/outscale/osc-sdk-go/v2"
)

//go:generate ../../../bin/mockgen -destination mock_security/securitygroup_mock.go -package mock_security -source ./securitygroup.go

type OscSecurityGroupInterface interface {
	CreateSecurityGroup(netId string, securityGroupName string, securityGroupDescription string) (*osc.SecurityGroup, error)
	CreateSecurityGroupRule(securityGroupId string, flow string, ipProtocol string, ipRange string, securityGroupMemberId string, fromPortRange int32, toPortRange int32) (*osc.SecurityGroup, error)
	DeleteSecurityGroupRule(securityGroupId string, flow string, ipProtocol string, ipRange string, securityGroupMemberId string, fromPortRange int32, toPortRange int32) error
	DeleteSecurityGroup(securityGroupId string) (error, *http.Response)
	GetSecurityGroup(securityGroupId string) (*osc.SecurityGroup, error)
	GetSecurityGroupFromSecurityGroupRule(securityGroupId string, Flow string, IpProtocols string, IpRanges string, FromPortRanges int32, ToPortRanges int32) (*osc.SecurityGroup, error)
	GetSecurityGroupIdsFromNetIds(netId string) ([]string, error)
}

func (s *Service) CreateSecurityGroup(netId string, securityGroupName string, securityGroupDescription string) (*osc.SecurityGroup, error) {
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
	return securityGroup, nil
}

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
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	securityGroupRule, ok := securityGroupRuleResponse.GetSecurityGroupOk()
	if !ok {
		return nil, errors.New("Can not get securityGroup")
	}
	return securityGroupRule, nil
}

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
		fmt.Printf("Error with http result %s", httpRes.Status)
		return err
	}
	return nil
}

func (s *Service) DeleteSecurityGroup(securityGroupId string) (error, *http.Response) {
	deleteSecurityGroupRequest := osc.DeleteSecurityGroupRequest{SecurityGroupId: &securityGroupId}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscApiClient.SecurityGroupApi.DeleteSecurityGroup(oscAuthClient).DeleteSecurityGroupRequest(deleteSecurityGroupRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return err, httpRes
	}
	return nil, httpRes
}

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
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
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

func (s *Service) GetSecurityGroupFromSecurityGroupRule(securityGroupId string, flow string, ipProtocols string, ipRanges string, fromPortRanges int32, toPortRanges int32) (*osc.SecurityGroup, error) {
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
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	readSecurityGroupRuleResponse, httpRes, err := oscApiClient.SecurityGroupApi.ReadSecurityGroups(oscAuthClient).ReadSecurityGroupsRequest(readSecurityGroupRuleRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
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
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
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

func ValidateIpProtocol(protocol string) (string, error) {
	switch {
	case protocol == "tcp" || protocol == "udp" || protocol == "icmp" || protocol == "-1":
		return protocol, nil
	default:
		return protocol, errors.New("Invalid protocol")
	}
}

func ValidateFlow(flow string) (string, error) {
	switch {
	case flow == "Inbound" || flow == "Outbound":
		return flow, nil
	default:
		return flow, errors.New("Invalid flow")
	}
}

func ValidateDescription(description string) (string, error) {
	isValidateDescription := regexp.MustCompile("^[\x20-\x7E]{0,255}$").MatchString
	if isValidateDescription(description) {
		return description, nil
	} else {
		return description, errors.New("Invalid Description")
	}
}
