package scope

import (
	"context"
	"errors"
	"fmt"
	osc "github.com/outscale/osc-sdk-go/v2"
	"os"
)

// OscClient contains input client to use outscale api
type OscClient struct {
	auth context.Context
	api  *osc.APIClient
}

// newOscClient return OscLient using secret credentials
func newOscClient() (*OscClient, error) {
	accessKey := os.Getenv("OSC_ACCESS_KEY")
	if accessKey == "" {
		return nil, errors.New("environment variable OSC_ACCESS_KEY is required")
	}
	secretKey := os.Getenv("OSC_SECRET_KEY")
	if secretKey == "" {
		return nil, errors.New("environment variable OSC_SECRET_KEY is required")
	}
	version := os.Getenv("VERSION")
	if version == "" {
		return nil, errors.New("Version is required")
	}
	config := osc.NewConfiguration()
	config.Debug = true
	config.UserAgent = fmt.Sprintf("cluster-api-provider-outscale/%s", version)
	auth := context.WithValue(context.Background(), osc.ContextAWSv4, osc.AWSv4{
		AccessKey: os.Getenv("OSC_ACCESS_KEY"),
		SecretKey: os.Getenv("OSC_SECRET_KEY"),
	})
	auth = context.WithValue(auth, osc.ContextServerIndex, 0)
	auth = context.WithValue(auth, osc.ContextServerVariables, map[string]string{"region": os.Getenv("OSC_REGION")})

	oscClient := &OscClient{
		api:  osc.NewAPIClient(config),
		auth: auth,
	}
	return oscClient, nil
}
