/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package services

import (
	"errors"
	"fmt"
	"unicode"

	"github.com/aws/aws-sdk-go/aws"             //nolint
	"github.com/aws/aws-sdk-go/aws/ec2metadata" //nolint
	"github.com/aws/aws-sdk-go/aws/endpoints"   //nolint
	"github.com/aws/aws-sdk-go/aws/request"     //nolint
	"github.com/aws/aws-sdk-go/aws/session"     //nolint
	"github.com/outscale/cluster-api-provider-outscale/cloud/utils"
	"k8s.io/klog/v2"
)

type Metadata struct {
	Region    string
	AccountID string
	NetID     string
}

// FetchMetadata queries the metadata server.
func FetchMetadata() (Metadata, error) {
	awsConfig := &aws.Config{
		EndpointResolver: MetadataResolver(),
	}
	// awsConfig.WithLogLevel(aws.LogDebugWithSigning | aws.LogDebugWithHTTPBody | aws.LogDebugWithRequestRetries | aws.LogDebugWithRequestErrors)
	sess, err := session.NewSession(awsConfig)
	if err != nil {
		return Metadata{}, fmt.Errorf("unable to fetch metadata: %w", err)
	}
	addHandlers(&sess.Handlers)
	svc := ec2metadata.New(sess)

	if !svc.Available() {
		return Metadata{}, errors.New("EC2 instance metadata is not available")
	}

	mac, err := svc.GetMetadata("mac")
	if err != nil || mac == "" {
		return Metadata{}, errors.New("could not get primary MAC from metadata")
	}

	netID, err := svc.GetMetadata("network/interfaces/macs/" + mac + "/vpc-id")
	if err != nil || netID == "" {
		return Metadata{}, errors.New("could not get net id")
	}
	accountID, err := svc.GetMetadata("network/interfaces/macs/" + mac + "/owner-id")
	if err != nil || accountID == "" {
		return Metadata{}, errors.New("could not get account id")
	}

	availabilityZone, err := svc.GetMetadata("placement/availability-zone")
	if err != nil || len(availabilityZone) < 2 {
		return Metadata{}, errors.New("could not get valid VM availability zone")
	}
	region := availabilityZone[0 : len(availabilityZone)-1]

	return Metadata{
		Region:    region,
		AccountID: accountID,
		NetID:     netID,
	}, nil
}

// MetadataResolver resolver for osc metadata service
func MetadataResolver() endpoints.ResolverFunc {
	return func(service, region string, optFns ...func(*endpoints.Options)) (endpoints.ResolvedEndpoint, error) {
		return endpoints.ResolvedEndpoint{
			URL:           "http://169.254.169.254/latest",
			SigningRegion: "custom-signing-region",
		}, nil
	}
}
func addHandlers(h *request.Handlers) {
	h.Build.PushFrontNamed(request.NamedHandler{
		Name: "cluster-api-provider-outscale/user-agent",
		Fn:   request.MakeAddToUserAgentHandler("cluster-api-provider-outscale", utils.GetVersion()),
	})

	// h.Sign.PushFrontNamed(request.NamedHandler{
	// 	Name: "k8s/logger",
	// 	Fn:   awsHandlerLogger,
	// })

	h.Send.PushBackNamed(request.NamedHandler{
		Name: "k8s/api-request",
		Fn:   awsSendHandlerLogger,
	})

	h.ValidateResponse.PushFrontNamed(request.NamedHandler{
		Name: "k8s/api-validate-response",
		Fn:   awsValidateResponseHandlerLogger,
	})
}

func awsSendHandlerLogger(req *request.Request) {
	_, call := awsServiceAndName(req)
	logger := klog.FromContext(req.HTTPRequest.Context())
	if logger.V(5).Enabled() {
		logger.Info("LBU request: "+cleanAws(req.Params), "LBU", call)
	}
}

func awsValidateResponseHandlerLogger(req *request.Request) {
	_, call := awsServiceAndName(req)
	logger := klog.FromContext(req.HTTPRequest.Context())
	switch {
	case req.Error != nil && req.HTTPResponse == nil:
		logger.V(3).Error(req.Error, "LBU error", "LBU", call)
	case req.HTTPResponse == nil:
	case req.HTTPResponse.StatusCode > 299:
		logger.V(3).Info("LBU error response: "+cleanAws(req.Data), "LBU", call, "http_status", req.HTTPResponse.Status)
	case logger.V(5).Enabled(): // no error
		logger.Info("LBU response: "+cleanAws(req.Data), "LBU", call)
	}
}

func awsServiceAndName(req *request.Request) (string, string) {
	service := req.ClientInfo.ServiceName

	name := "?"
	if req.Operation != nil {
		name = req.Operation.Name
	}
	return service, name
}

// cleanAws cleans a aws log
// - merges all multiple unicode spaces (\n, \r, \t, ...) into a single ascii space.
// - removes all spaces after unicode punctuations [ ] { } : , etc.
// - removes all "
func cleanAws(i any) string {
	str := fmt.Sprintf("%v", i)
	var prev rune
	return string(utils.Map([]rune(str), func(r rune) (rune, bool) {
		defer func() {
			prev = r
		}()
		switch {
		case unicode.IsSpace(r) && (unicode.IsPunct(prev) || unicode.IsSpace(prev)):
			return ' ', false
		case unicode.IsSpace(r):
			return ' ', true
		case r == '"':
			return ' ', false
		default:
			return r, true
		}
	}))
}
