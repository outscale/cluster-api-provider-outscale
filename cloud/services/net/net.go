package net

import (
	"fmt"

	"errors"

	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	tag "github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag"
	osc "github.com/outscale/osc-sdk-go/v2"
)

//go:generate ../../../bin/mockgen -destination mock_net/net_mock.go -package mock_net -source ./net.go

type OscNetInterface interface {
	CreateNet(spec *infrastructurev1beta1.OscNet, netName string) (*osc.Net, error)
	DeleteNet(netId string) error
	GetNet(netId string) (*osc.Net, error)
}

// CreateNet create the net from spec (in order to retrieve ip range)
func (s *Service) CreateNet(spec *infrastructurev1beta1.OscNet, netName string) (*osc.Net, error) {
	ipRange, err := infrastructurev1beta1.ValidateCidr(spec.IpRange)
	if err != nil {
		return nil, err
	}
	netRequest := osc.CreateNetRequest{
		IpRange: ipRange,
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
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
	err = tag.AddTag("OscK8sClusterID/"+netName, "owned", resourceIds, oscApiClient, oscAuthClient)
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	net, ok := netResponse.GetNetOk()
	if !ok {
		return nil, errors.New("Can not create net")
	}
	return net, nil
}

// DeleteNet delete the net
func (s *Service) DeleteNet(netId string) error {
	deleteNetRequest := osc.DeleteNetRequest{NetId: netId}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
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
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	readNetsResponse, httpRes, err := oscApiClient.NetApi.ReadNets(oscAuthClient).ReadNetsRequest(readNetsRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	nets, ok := readNetsResponse.GetNetsOk()
	if !ok {
		return nil, errors.New("Can not get net")
	}
	if len(*nets) == 0 {
		return nil, nil
	} else {
		net := *nets
		return &net[0], nil
	}
}
