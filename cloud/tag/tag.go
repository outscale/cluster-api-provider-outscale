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
	"github.com/outscale-dev/cluster-api-provider-outscale.git/util/reconciler"
	osc "github.com/outscale/osc-sdk-go/v2"
	"k8s.io/apimachinery/pkg/util/wait"
	"net/http"
	"regexp"
)

// AddTag add a tag to a resource
func AddTag(createTagRequest osc.CreateTagsRequest, resourceIds []string, api *osc.APIClient, auth context.Context) (error, *http.Response) {
	var httpRes *http.Response
	addTagNameCallBack := func() (bool, error) {
		_, httpRes, err := api.TagApi.CreateTags(auth).CreateTagsRequest(createTagRequest).Execute()
		if err != nil {
			if httpRes != nil {
				fmt.Printf("Error with http result %s", httpRes.Status)
			}
			requestStr := fmt.Sprintf("%v", createTagRequest)
			if reconciler.KeepRetryWithError(
				requestStr,
				httpRes.StatusCode,
				reconciler.ThrottlingErrors) {
				return false, nil
			}
			return false, fmt.Errorf("%w failed to add Tag Name", err)
		}
		return true, err
	}
	backoff := reconciler.EnvBackoff()
	waitErr := wait.ExponentialBackoff(backoff, addTagNameCallBack)
	if waitErr != nil {
		return waitErr, httpRes
	}
	return nil, httpRes
}

// ValidateTagNameValue check that tag name value is a valide name
func ValidateTagNameValue(tagValue string) (string, error) {
	isValidateTagNameValue := regexp.MustCompile(`^[0-9A-Za-z\-]{0,255}$`).MatchString
	if isValidateTagNameValue(tagValue) {
		return tagValue, nil
	} else {
		return tagValue, errors.New("Invalid Tag Name")
	}
}
