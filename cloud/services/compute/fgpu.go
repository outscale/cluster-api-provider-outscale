/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package compute

import (
	"context"
	"fmt"
	"strings"

	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	tag "github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	"github.com/outscale/cluster-api-provider-outscale/cloud/utils"
	"github.com/outscale/osc-sdk-go/v2"
	"k8s.io/utils/ptr"
)

//go:generate ../../../bin/mockgen -destination mock_compute/fgpu_mock.go -package mock_compute -source ./fgpu.go
type OscFGPUInterface interface {
	GetFGPU(ctx context.Context, id string) (*osc.FlexibleGpu, error)
	AllocateFGPU(ctx context.Context, model, az string, machineScope *scope.MachineScope) (*osc.FlexibleGpu, error)
	LinkFGPU(ctx context.Context, fGPUId, vmId string) error
}

func (s *Service) AllocateFGPU(ctx context.Context, model, az string, machineScope *scope.MachineScope) (*osc.FlexibleGpu, error) {
	req := osc.CreateFlexibleGpuRequest{
		ModelName:          model,
		SubregionName:      az,
		DeleteOnVmDeletion: ptr.To(true),
	}
	if after, ok := strings.CutPrefix(machineScope.GetVm().VmType, "tina"); ok {
		gen, _, _ := strings.Cut(after, ".")
		req.Generation = &gen
	}
	resp, httpRes, err := s.tenant.Client().FlexibleGpuApi.CreateFlexibleGpu(s.tenant.ContextWithAuth(ctx)).CreateFlexibleGpuRequest(req).Execute()
	err = utils.LogAndExtractError(ctx, "CreateFlexibleGpu", req, httpRes, err)
	if err != nil {
		return nil, err
	}
	resourceIds := []string{resp.FlexibleGpu.GetFlexibleGpuId()}
	nodeTag := osc.ResourceTag{
		Key:   tag.NameKey,
		Value: machineScope.GetName(),
	}
	tagRequest := osc.CreateTagsRequest{
		ResourceIds: resourceIds,
		Tags:        []osc.ResourceTag{nodeTag},
	}
	err = tag.AddTag(ctx, tagRequest, resourceIds, s.tenant.Client(), s.tenant.ContextWithAuth(ctx))
	if err != nil {
		return nil, fmt.Errorf("tag fgpu: %w", err)
	}
	return resp.FlexibleGpu, nil
}

func (s *Service) GetFGPU(ctx context.Context, id string) (*osc.FlexibleGpu, error) {
	req := osc.ReadFlexibleGpusRequest{
		Filters: &osc.FiltersFlexibleGpu{
			FlexibleGpuIds: &[]string{id},
		},
	}
	resp, httpRes, err := s.tenant.Client().FlexibleGpuApi.ReadFlexibleGpus(s.tenant.ContextWithAuth(ctx)).ReadFlexibleGpusRequest(req).Execute()
	err = utils.LogAndExtractError(ctx, "ReadFlexibleGpus", req, httpRes, err)
	switch {
	case err != nil:
		return nil, err
	case len(resp.GetFlexibleGpus()) == 0:
		return nil, nil
	default:
		return &(*resp.FlexibleGpus)[0], nil
	}
}

func (s *Service) LinkFGPU(ctx context.Context, fGPUId, vmId string) error {
	req := osc.LinkFlexibleGpuRequest{
		FlexibleGpuId: fGPUId,
		VmId:          vmId,
	}
	_, httpRes, err := s.tenant.Client().FlexibleGpuApi.LinkFlexibleGpu(s.tenant.ContextWithAuth(ctx)).LinkFlexibleGpuRequest(req).Execute()
	return utils.LogAndExtractError(ctx, "LinkFlexibleGpu", req, httpRes, err)
}
