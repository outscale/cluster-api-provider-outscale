package security

import (
	"errors"
	"fmt"
	_nethttp "net/http"

	tag "github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/util/reconciler"
	osc "github.com/outscale/osc-sdk-go/v2"
	"k8s.io/apimachinery/pkg/util/wait"
)

//go:generate ../../../bin/mockgen -destination mock_security/nic_mock.go -package mock_security -source ./nic.go
type OscNicInterface interface {
	CreateNic(nicName string, subnetId string) (*osc.Nic, error)
	DeleteNic(nicId string) error
	GetNic(nicId string) (*osc.Nic, error)
}

// CreateNic retrieve a publicip associated with you account
func (s *Service) CreateNic(nicName, subnetId string) (*osc.Nic, error) {
	nicRequest := osc.CreateNicRequest{}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()

	var nicResponse osc.CreateNicResponse
	createNicCallBack := func() (bool, error) {
		var httpRes *_nethttp.Response
		var err error
		nicResponse, httpRes, err = oscApiClient.NicApi.CreateNic(oscAuthClient).CreateNicRequest(nicRequest).Execute()
		if err != nil {
			if httpRes == nil {
				return false, fmt.Errorf("error without http Response %w", err)
			}
			requestStr := fmt.Sprintf("%v", nicRequest)
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
	waitErr := wait.ExponentialBackoff(backoff, createNicCallBack)
	if waitErr != nil {
		return nil, waitErr
	}

	nicTag := osc.ResourceTag{
		Key:   "Name",
		Value: nicName,
	}
	resourceIds := []string{*nicResponse.Nic.NicId}
	nicTagRequest := osc.CreateTagsRequest{
		ResourceIds: resourceIds,
		Tags:        []osc.ResourceTag{nicTag},
	}

	err, httpRes := tag.AddTag(nicTagRequest, resourceIds, oscApiClient, oscAuthClient)
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}

	nic, ok := nicResponse.GetNicOk()
	if !ok {
		return nil, errors.New("can not create Nic")
	}
	return nic, nil
}

// DeleteNic release the nic
func (s *Service) DeleteNic(nicId string) error {
	deleteNicRequest := osc.DeleteNicRequest{
		NicId: nicId,
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	deleteNicCallBack := func() (bool, error) {
		var httpRes *_nethttp.Response
		var err error
		_, httpRes, err = oscApiClient.NicApi.DeleteNic(oscAuthClient).DeleteNicRequest(deleteNicRequest).Execute()
		if err != nil {
			if httpRes == nil {
				return false, fmt.Errorf("error without http Response %w", err)
			}
			requestStr := fmt.Sprintf("%v", deleteNicRequest)
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
	waitErr := wait.ExponentialBackoff(backoff, deleteNicCallBack)
	if waitErr != nil {
		return waitErr
	}
	return nil
}

// GetNic get a Nic object using a Nic id
func (s *Service) GetNic(nicId string) (*osc.Nic, error) {
	readNicRequest := osc.ReadNicsRequest{
		Filters: &osc.FiltersNic{
			NicIds: &[]string{nicId},
		},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()

	var readNicsResponse osc.ReadNicsResponse
	readNicCallBack := func() (bool, error) {
		var httpRes *_nethttp.Response
		var err error
		readNicsResponse, httpRes, err = oscApiClient.NicApi.ReadNics(oscAuthClient).ReadNicsRequest(readNicRequest).Execute()
		if err != nil {
			if httpRes == nil {
				return false, fmt.Errorf("error without http Response %w", err)
			}
			requestStr := fmt.Sprintf("%v", readNicRequest)
			if reconciler.KeepRetryWithError(
				requestStr,
				httpRes.StatusCode,
				reconciler.ThrottlingErrors) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	}
	backoff := reconciler.EnvBackoff()
	waitErr := wait.ExponentialBackoff(backoff, readNicCallBack)
	if waitErr != nil {
		return nil, waitErr
	}
	nics, ok := readNicsResponse.GetNicsOk()
	if !ok {
		return nil, errors.New("can not get publicIp")
	}
	if len(*nics) == 0 {
		return nil, nil
	} else {
		publicIp := *nics
		return &publicIp[0], nil
	}
}
