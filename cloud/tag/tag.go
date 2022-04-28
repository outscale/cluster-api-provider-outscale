package tag

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	osc "github.com/outscale/osc-sdk-go/v2"
)

// AddTag add a tag to a resource
func AddTag(tagKey string, tagValue string, resourceIds []string, api *osc.APIClient, auth context.Context) error {
	tag := osc.ResourceTag{
		Key:   tagKey,
		Value: tagValue,
	}
	createTagRequest := osc.CreateTagsRequest{
		ResourceIds: resourceIds,
		Tags:        []osc.ResourceTag{tag},
	}
	_, httpRes, err := api.TagApi.CreateTags(auth).CreateTagsRequest(createTagRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return err
	}
	return nil
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
