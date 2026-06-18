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
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/tag"
	"github.com/outscale/osc-sdk-go/v3/pkg/osc"
)

type FGPUInterface interface {
	GetFGPU(ctx context.Context, id string) (*osc.FlexibleGpu, error)
	AllocateFGPU(ctx context.Context, model, az string, machineScope *scope.MachineScope) (*osc.FlexibleGpu, error)
	LinkFGPU(ctx context.Context, fGPUId, vmId string) error
}

func (s *Service) AllocateFGPU(ctx context.Context, model, az string, machineScope *scope.MachineScope) (*osc.FlexibleGpu, error) {
	req := osc.CreateFlexibleGpuRequest{
		ModelName:          model,
		SubregionName:      az,
		DeleteOnVmDeletion: new(true),
	}
	if after, ok := strings.CutPrefix(machineScope.GetVm().VmType, "tina"); ok {
		gen, _, _ := strings.Cut(after, ".")
		req.Generation = &gen
	}
	resp, err := s.tenant.Client().CreateFlexibleGpu(ctx, req)
	if err != nil {
		return nil, err
	}
	resourceIds := []string{resp.FlexibleGpu.FlexibleGpuId}
	nodeTag := osc.ResourceTag{
		Key:   tag.NameKey,
		Value: machineScope.GetName(),
	}
	tagRequest := osc.CreateTagsRequest{
		ResourceIds: resourceIds,
		Tags:        []osc.ResourceTag{nodeTag},
	}
	err = s.tags.AddTag(ctx, tagRequest, resourceIds)
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
	resp, err := s.tenant.Client().ReadFlexibleGpus(ctx, req)
	switch {
	case err != nil:
		return nil, err
	case len(*resp.FlexibleGpus) == 0:
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
	_, err := s.tenant.Client().LinkFlexibleGpu(ctx, req)
	return err
}
