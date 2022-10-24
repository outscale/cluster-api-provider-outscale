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
	"fmt"

	"github.com/outscale-dev/cluster-api-provider-outscale.git/util/reconciler"
	osc "github.com/outscale/osc-sdk-go/v2"
	"k8s.io/apimachinery/pkg/util/wait"
	_nethttp "net/http"
)

type OscKeyPairInterface interface {
	CreateKeyPair(keypairName string) (*osc.KeypairCreated, error)
	DeleteKeyPair(keypairName string) error
	GetKeyPair(keyPairName string) (*osc.Keypair, error)
}

// CreateKeyPair create keypair with keypairName
func (s *Service) CreateKeyPair(keypairName string) (*osc.KeypairCreated, error) {

	keyPairRequest := osc.CreateKeypairRequest{
		KeypairName: keypairName,
	}

	oscAPIClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	var keyPairResponse osc.CreateKeypairResponse
	createKeyPairCallBack := func() (bool, error) {
		var httpRes *_nethttp.Response
		var err error
		keyPairResponse, httpRes, err = oscAPIClient.KeypairApi.CreateKeypair(oscAuthClient).CreateKeypairRequest(keyPairRequest).Execute()
		if err != nil {
			if httpRes != nil {
				return false, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
			}
			requestStr := fmt.Sprintf("%v", keyPairRequest)
			if reconciler.KeepRetryWithError(
				requestStr,
				httpRes.StatusCode,
				reconciler.ThrottlingErrors) {
				return false, nil
			}
			return false, err
		}
		return true, err
	}
	backoff := reconciler.EnvBackoff()
	waitErr := wait.ExponentialBackoff(backoff, createKeyPairCallBack)
	if waitErr != nil {
		return nil, waitErr
	}

	req := osc.ReadKeypairsRequest{
		Filters: &osc.FiltersKeypair{},
	}

	resp, _, err := oscAPIClient.KeypairApi.ReadKeypairs(oscAuthClient).ReadKeypairsRequest(req).Execute()
	if err != nil {
		return nil, fmt.Errorf("Unable to read keypairRequest")
	}

	if len(resp.GetKeypairs()) < 1 {
		return nil, fmt.Errorf("Unable to find key pair, please provide a better query criteria ")
	}

	keypair, ok := keyPairResponse.GetKeypairOk()
	if !ok {
		return nil, fmt.Errorf("Can not create keypair")
	}
	return keypair, nil
}

// GetKeypair retrieve keypair from keypairName
func (s *Service) GetKeyPair(keyPairName string) (*osc.Keypair, error) {
	readKeypairsRequest := osc.ReadKeypairsRequest{
		Filters: &osc.FiltersKeypair{
			KeypairNames: &[]string{keyPairName},
		},
	}
	oscAPIClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	var readKeypairsResponse osc.ReadKeypairsResponse
	readKeypairsCallBack := func() (bool, error) {
		var httpRes *_nethttp.Response
		var err error
		readKeypairsResponse, httpRes, err = oscAPIClient.KeypairApi.ReadKeypairs(oscAuthClient).ReadKeypairsRequest(readKeypairsRequest).Execute()
		if err != nil {
			if httpRes != nil {
				return false, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
			}
			requestStr := fmt.Sprintf("%v", readKeypairsRequest)
			if reconciler.KeepRetryWithError(
				requestStr,
				httpRes.StatusCode,
				reconciler.ThrottlingErrors) {
				return false, nil
			}
			return false, err
		}
		return true, err
	}
	backoff := reconciler.EnvBackoff()
	waitErr := wait.ExponentialBackoff(backoff, readKeypairsCallBack)
	if waitErr != nil {
		return nil, waitErr
	}
	keypairs, ok := readKeypairsResponse.GetKeypairsOk()
	if !ok {
		return nil, fmt.Errorf("Error retrieving KeyPair")
	}

	if len(*keypairs) == 0 {
		return nil, nil
	} else {
		keypaires := *keypairs
		return &keypaires[0], nil
	}
}

// DeleteKeyPair delete machine keypair
func (s *Service) DeleteKeyPair(keyPairName string) error {
	deleteKeypairRequest := osc.DeleteKeypairRequest{KeypairName: keyPairName}
	oscAPIClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	deleteKeypairCallBack := func() (bool, error) {
		var httpRes *_nethttp.Response
		var err error
		_, httpRes, err = oscAPIClient.KeypairApi.DeleteKeypair(oscAuthClient).DeleteKeypairRequest(deleteKeypairRequest).Execute()
		if err != nil {
			if httpRes != nil {
				return false, fmt.Errorf("error %w httpRes %s", err, httpRes.Status)
			}
			requestStr := fmt.Sprintf("%v", deleteKeypairRequest)
			if reconciler.KeepRetryWithError(
				requestStr,
				httpRes.StatusCode,
				reconciler.ThrottlingErrors) {
				return false, nil
			}
			return false, err
		}
		return true, err
	}
	backoff := reconciler.EnvBackoff()
	waitErr := wait.ExponentialBackoff(backoff, deleteKeypairCallBack)
	if waitErr != nil {
		return waitErr
	}
	return nil
}
