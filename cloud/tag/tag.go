package tag

import(
    osc "github.com/outscale/osc-sdk-go/v2"
    "regexp"
    "github.com/pkg/errors"
    "context"
    "fmt"
)
func AddTag(tagKey string, tagValue string, resourceIds []string, api *osc.APIClient, auth context.Context) (error) {
    tag := osc.ResourceTag{
        Key: tagKey,
        Value: tagValue,
    }
    createTagRequest := osc.CreateTagsRequest{
        ResourceIds: resourceIds,
        Tags: []osc.ResourceTag{tag},
    } 
    _, httpRes, err := api.TagApi.CreateTags(auth).CreateTagsRequest(createTagRequest).Execute()
    if err != nil {
        fmt.Sprintf("Error with http result %s", httpRes.Status)
        return  err
    }
    return nil      
}

func ValidateTagNameValue(tagValue string) (string, error) {
   isValidateTagNameValue := regexp.MustCompile(`^[0-9A-Za-z\-]{0,255}$`).MatchString
   if isValidateTagNameValue(tagValue) {
       return tagValue, nil
   } else {
       return tagValue, errors.New("Invalid Tag Name")
   }   
}
