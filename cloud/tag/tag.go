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

package tag

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/outscale/cluster-api-provider-outscale/cloud/utils"
	osc "github.com/outscale/osc-sdk-go/v2"
	"k8s.io/apimachinery/pkg/util/wait"
)

type ResourceType string

const (
	InternetServiceResourceType ResourceType = "internet-service"
	NatResourceType             ResourceType = "natgateway"
	VmResourceType              ResourceType = "instance"
	RouteTableResourceType      ResourceType = "route-table"
	SecurityGroupResourceType   ResourceType = "security-group"
	NetResourceType             ResourceType = "vpc"
	SubnetResourceType          ResourceType = "subnet"
	PublicIPResourceType        ResourceType = "public-ip"
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
func AddTag(ctx context.Context, createTagRequest osc.CreateTagsRequest, resourceIds []string, api *osc.APIClient, auth context.Context) error {
	addTagNameCallBack := func() (bool, error) {
		_, httpRes, err := api.TagApi.CreateTags(auth).CreateTagsRequest(createTagRequest).Execute()
		err = utils.LogAndExtractError(ctx, "CreateTags", createTagRequest, httpRes, err)
		if err != nil {
			// we wish to retry on TCP errors, but not on 400 errors.
			if utils.RetryIf(httpRes) || httpRes == nil {
				return false, nil
			}
			return false, err
		}
		return true, nil
	}
	backoff := utils.EnvBackoff()
	return wait.ExponentialBackoff(backoff, addTagNameCallBack)
}

// ReadTag read a tag of a resource
func (s *Service) ReadTag(ctx context.Context, rsrcType ResourceType, key, value string) (*osc.Tag, error) {
	readTagsRequest := osc.ReadTagsRequest{
		Filters: &osc.FiltersTag{
			ResourceTypes: &[]string{string(rsrcType)},
			Keys:          &[]string{key},
			Values:        &[]string{value},
		},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	readTagsResponse, httpRes, err := oscApiClient.TagApi.ReadTags(oscAuthClient).ReadTagsRequest(readTagsRequest).Execute()
	err = utils.LogAndExtractError(ctx, "ReadTags", readTagsRequest, httpRes, err)
	if err != nil {
		if httpRes != nil {
			fmt.Printf("Error with http result %s", httpRes.Status)
		}
		return nil, err
	}
	tags, ok := readTagsResponse.GetTagsOk()
	if !ok {
		return nil, errors.New("Can not get tag")
	}
	if len(*tags) == 0 {
		return nil, nil
	} else {
		tag := *tags
		return &tag[0], nil
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

func GetTagValue(key string, tags []osc.ResourceTag) string {
	for _, tg := range tags {
		if key == tg.GetKey() {
			return tg.GetValue()
		}
	}
	return ""
}
