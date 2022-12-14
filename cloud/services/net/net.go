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
	"fmt"

	"errors"

	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	tag "github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/util/reconciler"
	osc "github.com/outscale/osc-sdk-go/v2"
	"k8s.io/apimachinery/pkg/util/wait"
	_nethttp "net/http"
)

//go:generate ../../../bin/mockgen -destination mock_net/net_mock.go -package mock_net -source ./net.go

type OscNetInterface interface {
	CreateNet(spec *infrastructurev1beta1.OscNet, clusterName string, netName string) (*osc.Net, error)
	DeleteNet(netId string) error
	GetNet(netId string) (*osc.Net, error)
}

// CreateNet create the net from spec (in order to retrieve ip range)
func (s *Service) CreateNet(spec *infrastructurev1beta1.OscNet, clusterName string, netName string) (*osc.Net, error) {
	ipRange, err := infrastructurev1beta1.ValidateCidr(spec.IpRange)
	if err != nil {
		return nil, err
	}
	netRequest := osc.CreateNetRequest{
		IpRange: ipRange,
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	var netResponse osc.CreateNetResponse
	netResponse, httpRes, err := oscApiClient.NetApi.CreateNet(oscAuthClient).CreateNetRequest(netRequest).Execute()
	if err != nil {
		if httpRes != nil {
			return nil, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
		}
		return nil, err
	}
	resourceIds := []string{*netResponse.Net.NetId}
	netTag := osc.ResourceTag{
		Key:   "Name",
		Value: netName,
	}
	netTagRequest := osc.CreateTagsRequest{
		ResourceIds: resourceIds,
		Tags:        []osc.ResourceTag{netTag},
	}

	err, httpRes = tag.AddTag(netTagRequest, resourceIds, oscApiClient, oscAuthClient)
	if err != nil {
		if httpRes != nil {
			return nil, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
		} else {
			return nil, err
		}
	}
	clusterNetTag := osc.ResourceTag{
		Key:   "OscK8sClusterID/" + clusterName,
		Value: "owned",
	}
	netTagRequest = osc.CreateTagsRequest{
		ResourceIds: resourceIds,
		Tags:        []osc.ResourceTag{clusterNetTag},
	}

	err, httpRes = tag.AddTag(netTagRequest, resourceIds, oscApiClient, oscAuthClient)
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
	deleteNetCallBack := func() (bool, error) {
		var httpRes *_nethttp.Response
		var err error
		_, httpRes, err = oscApiClient.NetApi.DeleteNet(oscAuthClient).DeleteNetRequest(deleteNetRequest).Execute()
		if err != nil {
			if httpRes != nil {
				return false, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
			}
			requestStr := fmt.Sprintf("%v", deleteNetRequest)
			if reconciler.KeepRetryWithError(
				requestStr,
				httpRes.StatusCode,
				reconciler.ThrottlingErrors) {
				return false, nil
			}
			return false, err
		}
		return true, err

	}
	backoff := reconciler.EnvBackoff()
	waitErr := wait.ExponentialBackoff(backoff, deleteNetCallBack)
	if waitErr != nil {
		return waitErr
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
		if httpRes != nil {
			fmt.Printf("Error with http result %s", httpRes.Status)
			return nil, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
		} else {
			return nil, err
		}
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
