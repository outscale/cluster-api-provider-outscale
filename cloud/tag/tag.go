package tag

import(
    osc "github.com/outscale/osc-sdk-go/v2"
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
