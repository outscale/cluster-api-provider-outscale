package net

import (
	"fmt"
	"net"
	"regexp"
	"strings"

	infrastructurev1beta1 "github.com/outscale-vbr/cluster-api-provider-outscale.git/api/v1beta1"
	tag "github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/tag"
	osc "github.com/outscale/osc-sdk-go/v2"
	"github.com/pkg/errors"
)

func ValidateCidr(cidr string) (string, error) {
	if !strings.Contains(cidr, "/") {
		return cidr, errors.New("Invalid Not A CIDR")
	}
	cidr_split := strings.Split(cidr, "/")
	ip := cidr_split[0]
	prefix := cidr_split[1]
	if net.ParseIP(ip) == nil {
		return cidr, errors.New("Invalid Cidr Ip")
	}
	isValidatePrefix := regexp.MustCompile(`^([0-9]|[1-2][0-9]|3[0-1]|32)$`).MatchString
	if !isValidatePrefix(prefix) {
		return cidr, errors.New("Invalid Cidr Prefix")
	}
	return cidr, nil
}

func (s *Service) CreateNet(spec *infrastructurev1beta1.OscNet, tagValue string) (*osc.Net, error) {
	IpRange, err := ValidateCidr(spec.IpRange)
	if err != nil {
		return nil, err
	}
	netRequest := osc.CreateNetRequest{
		IpRange: IpRange,
	}
	OscApiClient := s.scope.Api()
	OscAuthClient := s.scope.Auth()
	netResponse, httpRes, err := OscApiClient.NetApi.CreateNet(OscAuthClient).CreateNetRequest(netRequest).Execute()
	if err != nil {
		fmt.Sprintf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	netName, err := tag.ValidateTagNameValue(tagValue)
	if err != nil {
		return nil, err
	}
	resourceIds := []string{*netResponse.Net.NetId}
	err = tag.AddTag("Name", netName, resourceIds, OscApiClient, OscAuthClient)
	if err != nil {
		fmt.Sprintf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	return netResponse.Net, nil
}

func (s *Service) DeleteNet(netId string) error {
	deleteNetRequest := osc.DeleteNetRequest{NetId: netId}
	OscApiClient := s.scope.Api()
	OscAuthClient := s.scope.Auth()
	_, httpRes, err := OscApiClient.NetApi.DeleteNet(OscAuthClient).DeleteNetRequest(deleteNetRequest).Execute()
	if err != nil {
		fmt.Sprintf("Error with http result %s", httpRes.Status)
		return err
	}
	return nil
}

func (s *Service) GetNet(netId []string) (*osc.Net, error) {
	readNetsRequest := osc.ReadNetsRequest{
		Filters: &osc.FiltersNet{
			NetIds: &netId,
		},
	}
	OscApiClient := s.scope.Api()
	OscAuthClient := s.scope.Auth()
	readNetsResponse, httpRes, err := OscApiClient.NetApi.ReadNets(OscAuthClient).ReadNetsRequest(readNetsRequest).Execute()
	if err != nil {
		fmt.Sprintf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	var net []osc.Net
	nets := *readNetsResponse.Nets
	if len(nets) == 0 {
		return nil, nil
	} else {
		net = append(net, nets...)
		return &net[0], nil
	}
}
