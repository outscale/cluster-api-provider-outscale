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

package security

import (
	"context"
	"errors"

	"github.com/outscale/cluster-api-provider-outscale/cloud/utils"
	osc "github.com/outscale/osc-sdk-go/v2"
)

//go:generate ../../../bin/mockgen -destination mock_security/keypair_mock.go -package mock_security -source ./keypair.go
type OscKeyPairInterface interface {
	CreateKeyPair(ctx context.Context, keypairName string) (*osc.KeypairCreated, error)
	DeleteKeyPair(ctx context.Context, keypairName string) error
	GetKeyPair(ctx context.Context, keyPairName string) (*osc.Keypair, error)
}

// CreateKeyPair create keypair with keypairName
func (s *Service) CreateKeyPair(ctx context.Context, keypairName string) (*osc.KeypairCreated, error) {
	keyPairRequest := osc.CreateKeypairRequest{
		KeypairName: keypairName,
	}

	oscAPIClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	keyPairResponse, httpRes, err := oscAPIClient.KeypairApi.CreateKeypair(oscAuthClient).CreateKeypairRequest(keyPairRequest).Execute()
	utils.LogAPICall(ctx, "CreateKeypair", keyPairRequest, httpRes, err)
	if err != nil {
		return nil, utils.ExtractOAPIError(err, httpRes)
	}

	req := osc.ReadKeypairsRequest{
		Filters: &osc.FiltersKeypair{},
	}

	resp, httpRes, err := oscAPIClient.KeypairApi.ReadKeypairs(oscAuthClient).ReadKeypairsRequest(req).Execute()
	utils.LogAPICall(ctx, "ReadKeypairs", req, httpRes, err)
	if err != nil {
		return nil, errors.New("Unable to read keypairRequest")
	}

	if len(resp.GetKeypairs()) < 1 {
		return nil, errors.New("Unable to find key pair, please provide a better query criteria ")
	}

	keypair, ok := keyPairResponse.GetKeypairOk()
	if !ok {
		return nil, errors.New("Can not create keypair")
	}
	return keypair, nil
}

// GetKeypair retrieve keypair from keypairName
func (s *Service) GetKeyPair(ctx context.Context, keyPairName string) (*osc.Keypair, error) {
	readKeypairsRequest := osc.ReadKeypairsRequest{
		Filters: &osc.FiltersKeypair{
			KeypairNames: &[]string{keyPairName},
		},
	}
	oscAPIClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	readKeypairsResponse, httpRes, err := oscAPIClient.KeypairApi.ReadKeypairs(oscAuthClient).ReadKeypairsRequest(readKeypairsRequest).Execute()
	utils.LogAPICall(ctx, "ReadKeypairs", readKeypairsRequest, httpRes, err)
	if err != nil {
		return nil, utils.ExtractOAPIError(err, httpRes)
	}
	keypairs, ok := readKeypairsResponse.GetKeypairsOk()
	if !ok {
		return nil, errors.New("error retrieving KeyPair")
	}

	if len(*keypairs) == 0 {
		return nil, nil
	} else {
		keypaires := *keypairs
		return &keypaires[0], nil
	}
}

// DeleteKeyPair delete machine keypair
func (s *Service) DeleteKeyPair(ctx context.Context, keyPairName string) error {
	deleteKeypairRequest := osc.DeleteKeypairRequest{KeypairName: &keyPairName}
	oscAPIClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	_, httpRes, err := oscAPIClient.KeypairApi.DeleteKeypair(oscAuthClient).DeleteKeypairRequest(deleteKeypairRequest).Execute()
	utils.LogAPICall(ctx, "DeleteKeypair", deleteKeypairRequest, httpRes, err)
	return utils.ExtractOAPIError(err, httpRes)
}
