package net

import (
	"fmt"
	"net"
	"strings"
	"strconv"
	infrastructurev1beta1 "github.com/outscale-vbr/cluster-api-provider-outscale.git/api/v1beta1"
	tag "github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/tag"
	osc "github.com/outscale/osc-sdk-go/v2"
	"github.com/pkg/errors"
)

// ValidateCidr check that the cidr string is a valide CIDR
func ValidateCidr(cidr string) (string, error) {
	if !strings.Contains(cidr, "/") {
		return cidr, errors.New("Invalid Not A CIDR")
	}
	_, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return cidr, err
	}
	return cidr, nil
}


// CreateNet create the net from spec (in order to retrieve ip range)
func (s *Service) CreateNet(spec *infrastructurev1beta1.OscNet, netName string) (*osc.Net, error) {
	ipRange, err := ValidateCidr(spec.IpRange)
	if err != nil {
		return nil, err
	}
	netRequest := osc.CreateNetRequest{
		IpRange: ipRange,
	}
	oscApiClient := s.scope.Api()
	oscAuthClient := s.scope.Auth()
	netResponse, httpRes, err := oscApiClient.NetApi.CreateNet(oscAuthClient).CreateNetRequest(netRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	resourceIds := []string{*netResponse.Net.NetId}
	err = tag.AddTag("Name", netName, resourceIds, oscApiClient, oscAuthClient)
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	return netResponse.Net, nil
}

// DeleteNet delete the net
func (s *Service) DeleteNet(netId string) error {
	deleteNetRequest := osc.DeleteNetRequest{NetId: netId}
	oscApiClient := s.scope.Api()
	oscAuthClient := s.scope.Auth()
	_, httpRes, err := oscApiClient.NetApi.DeleteNet(oscAuthClient).DeleteNetRequest(deleteNetRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return err
	}
	return nil
}

// GetNet retrieve the net object using the net id
func (s *Service) GetNet(netId string) (*osc.Net, error) {
	readNetsRequest := osc.ReadNetsRequest{
		Filters: &osc.FiltersNet{
			NetIds: &[]string{netId},
		},
	}
	oscApiClient := s.scope.Api()
	oscAuthClient := s.scope.Auth()
	readNetsResponse, httpRes, err := oscApiClient.NetApi.ReadNets(oscAuthClient).ReadNetsRequest(readNetsRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	nets := *readNetsResponse.Nets
	if len(nets) == 0 {
		return nil, nil
	} else {
		return &nets[0], nil
	}
}
