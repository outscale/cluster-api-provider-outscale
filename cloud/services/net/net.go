/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package net

import (
	"context"
	"errors"
	"net/http"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	tag "github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	"github.com/outscale/cluster-api-provider-outscale/cloud/utils"
	osc "github.com/outscale/osc-sdk-go/v2"
	"k8s.io/apimachinery/pkg/util/wait"
)

//go:generate ../../../bin/mockgen -destination mock_net/net_mock.go -package mock_net -source ./net.go

type OscNetInterface interface {
	CreateNet(ctx context.Context, spec infrastructurev1beta1.OscNet, clusterID, netName string) (*osc.Net, error)
	DeleteNet(ctx context.Context, netId string) error
	GetNet(ctx context.Context, netId string) (*osc.Net, error)
}

// CreateNet create the net from spec (in order to retrieve ip range)
func (s *Service) CreateNet(ctx context.Context, spec infrastructurev1beta1.OscNet, clusterID, netName string) (*osc.Net, error) {
	netRequest := osc.CreateNetRequest{
		IpRange: spec.IpRange,
	}

	var netResponse osc.CreateNetResponse
	netResponse, httpRes, err := s.tenant.Client().NetApi.CreateNet(s.tenant.ContextWithAuth(ctx)).CreateNetRequest(netRequest).Execute()
	err = utils.LogAndExtractError(ctx, "CreateNet", netRequest, httpRes, err)
	if err != nil {
		return nil, err
	}
	resourceIds := []string{*netResponse.Net.NetId}
	netTag := osc.ResourceTag{
		Key:   "Name",
		Value: netName,
	}
	clusterTag := osc.ResourceTag{
		Key:   "OscK8sClusterID/" + clusterID,
		Value: "owned",
	}
	netTagRequest := osc.CreateTagsRequest{
		ResourceIds: resourceIds,
		Tags:        []osc.ResourceTag{netTag, clusterTag},
	}

	err = tag.AddTag(ctx, netTagRequest, resourceIds, s.tenant.Client(), s.tenant.ContextWithAuth(ctx))
	if err != nil {
		return nil, err
	}
	net, ok := netResponse.GetNetOk()
	if !ok {
		return nil, errors.New("Can not create net")
	}
	return net, nil
}

// DeleteNet delete the net
func (s *Service) DeleteNet(ctx context.Context, netId string) error {
	deleteNetRequest := osc.DeleteNetRequest{NetId: netId}

	deleteNetCallBack := func() (bool, error) {
		var httpRes *http.Response
		var err error
		_, httpRes, err = s.tenant.Client().NetApi.DeleteNet(s.tenant.ContextWithAuth(ctx)).DeleteNetRequest(deleteNetRequest).Execute()
		err = utils.LogAndExtractError(ctx, "DeleteNet", deleteNetRequest, httpRes, err)
		if err != nil {
			if utils.RetryIf(httpRes) {
				return false, nil
			}
			return false, err
		}
		return true, err
	}
	backoff := utils.EnvBackoff()
	waitErr := wait.ExponentialBackoff(backoff, deleteNetCallBack)
	if waitErr != nil {
		return waitErr
	}
	return nil
}

// GetNet retrieve the net object using the net id
func (s *Service) GetNet(ctx context.Context, netId string) (*osc.Net, error) {
	readNetsRequest := osc.ReadNetsRequest{
		Filters: &osc.FiltersNet{
			NetIds: &[]string{netId},
		},
	}

	readNetsResponse, httpRes, err := s.tenant.Client().NetApi.ReadNets(s.tenant.ContextWithAuth(ctx)).ReadNetsRequest(readNetsRequest).Execute()
	err = utils.LogAndExtractError(ctx, "ReadNets", readNetsRequest, httpRes, err)
	if err != nil {
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
