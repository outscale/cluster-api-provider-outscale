package scope

import (
	"context"
	"os"

	osc "github.com/outscale/osc-sdk-go/v2"
	"github.com/pkg/errors"
)

type OscClient struct {
	auth context.Context
	api  *osc.APIClient
}

func newOscClient() (*OscClient, error) {
	accessKey := os.Getenv("OSC_ACCESS_KEY")
	if accessKey == "" {
		return nil, errors.New("env var accessKey is required")
	}
	secretKey := os.Getenv("OSC_SECRET_KEY")
	if secretKey == "" {
		return nil, errors.New("env var secretKey is required")
	}
	oscClient := &OscClient{
		api: osc.NewAPIClient(osc.NewConfiguration()),
		auth: context.WithValue(context.Background(), osc.ContextAWSv4, osc.AWSv4{
			AccessKey: os.Getenv("OSC_ACCESS_KEY"),
			SecretKey: os.Getenv("OSC_SECRET_KEY"),
		}),
	}
	return oscClient, nil
}
