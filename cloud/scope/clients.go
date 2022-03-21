package scope

import (
    "context"
    "os"
    "github.com/pkg/errors"
    osc "github.com/outscale/osc-sdk-go/v2"
)

type OscApiClient struct {
   auth context.Context
   api *osc.APIClient
}

func newOscApiClient() (*OscApiClient, error) {
    accessKey := os.Getenv("OSC_ACCESS_KEY")
    if accessKey == "" {
        return nil, errors.New("env var accessKey is required")
    }
    secretKey := os.Getenv("OSC_SECRET_KEY")
    if secretKey == "" {
        return nil, errors.New("env var secretKey is required")
    }
    oscApiClient := &OscApiClient{
    			api: osc.NewAPIClient(osc.NewConfiguration()),
                        auth: context.WithValue(context.Background(), osc.ContextAWSv4, osc.AWSv4{
				AccessKey: os.Getenv("OSC_ACCESS_KEY"),
				SecretKey: os.Getenv("OSC_SECRET_KEY"),
                        }),
    }
    return oscApiClient, nil
}
