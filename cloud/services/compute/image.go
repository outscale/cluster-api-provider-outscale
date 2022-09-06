package compute

import (
	"fmt"

	osc "github.com/outscale/osc-sdk-go/v2"
)

//go:generate ../../../bin/mockgen -destination mock_compute/image_mock.go -package mock_compute -source ./image.go
type OscImageInterface interface {
	GetImage(imageId string) (*osc.Image, error)
}

// GetImage retrieve image from imageId
func (s *Service) GetImage(imageId string) (*osc.Image, error) {
	readImageRequest := osc.ReadImagesRequest{
		Filters: &osc.FiltersImage{ImageIds: &[]string{imageId}},
	}
	oscApiClient := s.scope.GetApi()
	oscAuthClient := s.scope.GetAuth()
	readImageResponse, httpRes, err := oscApiClient.ImageApi.ReadImages(oscAuthClient).ReadImagesRequest(readImageRequest).Execute()
	if err != nil {
		fmt.Printf("Error with http result %s", httpRes.Status)
		return nil, err
	}
	if len(readImageResponse.GetImages()) == 0 {
		return nil, nil
	}
	image := readImageResponse.GetImages()[0]

	return &image, nil
}
