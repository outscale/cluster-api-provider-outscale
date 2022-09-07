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

package net

import (
	"errors"
	"fmt"

	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	tag "github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag"
	osc "github.com/outscale/osc-sdk-go/v2"
)

//go:generate ../../../bin/mockgen -destination mock_net/net_mock.go -package mock_net -source ./net.go

// OscNetInterface is the net interface.
type OscNetInterface interface {
	CreateNet(spec *infrastructurev1beta1.OscNet, netName string) (*osc.Net, error)
	DeleteNet(netID string) error
	GetNet(netID string) (*osc.Net, error)
}

// CreateNet create the net from spec (in order to retrieve ip range).
func (s *Service) CreateNet(spec *infrastructurev1beta1.OscNet, netName string) (*osc.Net, error) {
	ipRange, err := infrastructurev1beta1.ValidateCidr(spec.IPRange)
	if err != nil {
		return nil, err
	}
	netRequest := osc.CreateNetRequest{
		IpRange: ipRange,
	}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	netResponse, httpRes, err := oscAPIClient.NetApi.CreateNet(oscAuthClient).CreateNetRequest(netRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	resourceIds := []string{*netResponse.Net.NetId}
	err = tag.AddTag(oscAuthClient, "Name", netName, resourceIds, oscAPIClient)
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	net, ok := netResponse.GetNetOk()
	if !ok {
		return nil, errors.New("can not create net")
	}
	return net, nil
}

// DeleteNet delete the net.
func (s *Service) DeleteNet(netID string) error {
	deleteNetRequest := osc.DeleteNetRequest{NetId: netID}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscAPIClient.NetApi.DeleteNet(oscAuthClient).DeleteNetRequest(deleteNetRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return err
	}
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	return nil
}

// GetNet retrieve the net object using the net id.
func (s *Service) GetNet(netID string) (*osc.Net, error) {
	readNetsRequest := osc.ReadNetsRequest{
		Filters: &osc.FiltersNet{
			NetIds: &[]string{netID},
		},
	}
	oscAPIClient := s.scope.GetAPI()
	oscAuthClient := s.scope.GetAuth()
	readNetsResponse, httpRes, err := oscAPIClient.NetApi.ReadNets(oscAuthClient).ReadNetsRequest(readNetsRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	nets, ok := readNetsResponse.GetNetsOk()
	if !ok {
		return nil, errors.New("can not get net")
	}
	if len(*nets) == 0 {
		return nil, nil
	}
	net := *nets
	return &net[0], nil
}
