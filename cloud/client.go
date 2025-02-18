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

package cloud

import (
	"context"
	"errors"
	"os"

	osc "github.com/outscale/osc-sdk-go/v2"
)

// OscClient contains input client to use outscale api
type OscClient struct {
	Auth context.Context
	API  *osc.APIClient
}

// newOscClient return OscLient using secret credentials
func NewOscClient() (*OscClient, error) {
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
		version = "DEV"
	}
	config := osc.NewConfiguration()
	// config.Debug = true
	config.UserAgent = "cluster-api-provider-outscale/" + version
	auth := context.WithValue(context.Background(), osc.ContextAWSv4, osc.AWSv4{
		AccessKey: os.Getenv("OSC_ACCESS_KEY"),
		SecretKey: os.Getenv("OSC_SECRET_KEY"),
	})
	auth = context.WithValue(auth, osc.ContextServerIndex, 0)
	auth = context.WithValue(auth, osc.ContextServerVariables, map[string]string{"region": os.Getenv("OSC_REGION")})

	oscClient := &OscClient{
		API:  osc.NewAPIClient(config),
		Auth: auth,
	}
	return oscClient, nil
}
