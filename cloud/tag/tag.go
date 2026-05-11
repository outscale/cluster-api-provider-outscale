/*
SPDX-FileCopyrightText: 2022 The Kubernetes Authors

SPDX-License-Identifier: Apache-2.0
*/

package tag

import (
	"context"
	"errors"
	"regexp"

	"github.com/outscale/osc-sdk-go/v3/pkg/osc"
)

type ResourceType string

const (
	NetResourceType             ResourceType = "vpc"
	NetPeeringResourceType      ResourceType = "vpc-peering-connection"
	SubnetResourceType          ResourceType = "subnet"
	InternetServiceResourceType ResourceType = "internet-service"
	NetAccessPointResourceType  ResourceType = "internet-service"
	NatResourceType             ResourceType = "natgateway"
	VmResourceType              ResourceType = "instance"
	RouteTableResourceType      ResourceType = "route-table"
	SecurityGroupResourceType   ResourceType = "security-group"
	PublicIPResourceType        ResourceType = "public-ip"
	FlexibleGPUResourceType     ResourceType = "flexible-gpu"
)

const (
	NameKey = "Name"

	ClusterKeyPrefix = "OscK8sClusterID/"
	OwnedValue       = "owned"
)

//go:generate ../../bin/mockgen -destination mock_tag/tag_mock.go -package mock_tag -source ./tag.go
type OscTagInterface interface {
	ReadTag(ctx context.Context, rsrcType ResourceType, key, value string) (*osc.Tag, error)
	ReadOwnedByTag(ctx context.Context, rsrcType ResourceType, cluster string) (*osc.Tag, error)
}

// AddTag add a tag to a resource
func AddTag(ctx context.Context, req osc.CreateTagsRequest, resourceIds []string, api *osc.Client) error {
	_, err := api.CreateTags(ctx, req)
	return err
}

// ReadTag read a tag of a resource
func (s *Service) ReadTag(ctx context.Context, rsrcType ResourceType, key, value string) (*osc.Tag, error) {
	req := osc.ReadTagsRequest{
		Filters: &osc.FiltersTag{
			ResourceTypes: &[]string{string(rsrcType)},
			Keys:          &[]string{key},
			Values:        &[]string{value},
		},
	}

	resp, err := s.tenant.Client().ReadTags(ctx, req)
	switch {
	case err != nil:
		return nil, err
	case len(*resp.Tags) == 0:
		return nil, nil
	default:
		return &(*resp.Tags)[0], nil
	}
}

func (s *Service) ReadOwnedByTag(ctx context.Context, rsrcType ResourceType, cluster string) (*osc.Tag, error) {
	return s.ReadTag(ctx, rsrcType, ClusterKeyPrefix+cluster, OwnedValue)
}

// ValidateTagNameValue check that tag name value is a valid name
func ValidateTagNameValue(tagValue string) (string, error) {
	isValidateTagNameValue := regexp.MustCompile(`^[0-9A-Za-z\-]{0,255}$`).MatchString
	if isValidateTagNameValue(tagValue) {
		return tagValue, nil
	} else {
		return tagValue, errors.New("Invalid Tag Name")
	}
}
