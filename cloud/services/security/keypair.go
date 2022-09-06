package security

import (
	"fmt"

	tag "github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag"
	osc "github.com/outscale/osc-sdk-go/v2"
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

	keyPairResponse, httpRes, err := oscAPIClient.KeypairApi.CreateKeypair(oscAuthClient).CreateKeypairRequest(keyPairRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
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

	if len(resp.GetKeypairs()) > 1 {
		return nil, fmt.Errorf("too many key pair, please provide a better query criteria ")
	}

	resourceIds := []string{*keyPairResponse.Keypair.KeypairName}
	err = tag.AddTag("Name", keypairName, resourceIds, oscAPIClient, oscAuthClient)
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}

	keypair, ok := keyPairResponse.GetKeypairOk()
	if !ok {
		return nil, fmt.Errorf("Can not create keypair")
	}
	return keypair, nil
}

// GetKeypair retrieve keypair from keypairName
func (s *Service) GetKeyPair(keyPairName string) (*osc.Keypair, error) {
	readKeypairRequest := osc.ReadKeypairsRequest{
		Filters: &osc.FiltersKeypair{
			KeypairNames: &[]string{keyPairName},
		},
	}
	oscAPIClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	readKeypairResponse, httpRes, err := oscAPIClient.KeypairApi.ReadKeypairs(oscAuthClient).ReadKeypairsRequest(readKeypairRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	keypairs, ok := readKeypairResponse.GetKeypairsOk()
	if !ok {
		return nil, fmt.Errorf("Error retrieving KeyPair")
	}

	if len(*keypairs) == 0 {
		return nil, fmt.Errorf("No keypair found")
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

	_, httpRes, err := oscAPIClient.KeypairApi.DeleteKeypair(oscAuthClient).DeleteKeypairRequest(deleteKeypairRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return err
	}
	return nil
}
