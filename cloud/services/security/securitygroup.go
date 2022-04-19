package security

import (
	"fmt"
	"net/http"
	"regexp"

	osc "github.com/outscale/osc-sdk-go/v2"
	"github.com/pkg/errors"
)

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

func (s *Service) CreateSecurityGroupRule(securityGroupId string, flow string, ipProtocol string, ipRange string, fromPortRange int32, toPortRange int32) (*osc.SecurityGroup, error) {
	createSecurityGroupRuleRequest := osc.CreateSecurityGroupRuleRequest{
		Flow:            flow,
		SecurityGroupId: securityGroupId,
		IpProtocol:      &ipProtocol,
		IpRange:         &ipRange,
		FromPortRange:   &fromPortRange,
		ToPortRange:     &toPortRange,
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

func (s *Service) DeleteSecurityGroupRule(securityGroupId string, flow string, ipProtocol string, ipRange string, fromPortRange int32, toPortRange int32) error {
	deleteSecurityGroupRuleRequest := osc.DeleteSecurityGroupRuleRequest{
		Flow:            flow,
		SecurityGroupId: securityGroupId,
		IpProtocol:      &ipProtocol,
		IpRange:         &ipRange,
		FromPortRange:   &fromPortRange,
		ToPortRange:     &toPortRange,
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

func (s *Service) GetSecurityGroupFromSecurityGroupRule(securityGroupId string, Flow string, IpProtocols string, IpRanges string, FromPortRanges int32, ToPortRanges int32) (*osc.SecurityGroup, error) {
	var readSecurityGroupRuleRequest osc.ReadSecurityGroupsRequest
	switch {
	case Flow == "Inbound":
		readSecurityGroupRuleRequest = osc.ReadSecurityGroupsRequest{
			Filters: &osc.FiltersSecurityGroup{
				SecurityGroupIds:          &[]string{securityGroupId},
				InboundRuleProtocols:      &[]string{IpProtocols},
				InboundRuleIpRanges:       &[]string{IpRanges},
				InboundRuleFromPortRanges: &[]int32{FromPortRanges},
				InboundRuleToPortRanges:   &[]int32{ToPortRanges},
			},
		}
	case Flow == "Outbound":
		readSecurityGroupRuleRequest = osc.ReadSecurityGroupsRequest{
			Filters: &osc.FiltersSecurityGroup{
				SecurityGroupIds:           &[]string{securityGroupId},
				OutboundRuleProtocols:      &[]string{IpProtocols},
				OutboundRuleIpRanges:       &[]string{IpRanges},
				OutboundRuleFromPortRanges: &[]int32{FromPortRanges},
				OutboundRuleToPortRanges:   &[]int32{ToPortRanges},
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
