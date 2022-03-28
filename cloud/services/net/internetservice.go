package net

import(
    infrastructurev1beta1 "github.com/outscale-vbr/cluster-api-provider-outscale.git/api/v1beta1"
    osc "github.com/outscale/osc-sdk-go/v2"
    tag "github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/tag"

    "fmt"
    
)

func (s *Service) CreateInternetService(spec *infrastructurev1beta1.OscInternetService) (*osc.InternetService, error) {
    internetServiceRequest := osc.CreateInternetServiceRequest{}
    OscApiClient := s.scope.Api()
    OscAuthClient := s.scope.Auth()
    internetServiceResponse, httpRes, err := OscApiClient.InternetServiceApi.CreateInternetService(OscAuthClient).CreateInternetServiceRequest(internetServiceRequest).Execute()    
    if err != nil {
        fmt.Sprintf("Error with http result %s", httpRes.Status)
        return nil, err
    }
    resourceIds := []string{*internetServiceResponse.InternetService.InternetServiceId}
    err = tag.AddTag("Name", "cluster-api-igw", resourceIds, OscApiClient, OscAuthClient)
    if err != nil {
        fmt.Sprintf("Error with http result %s", httpRes.Status)
        return nil, err
    }
    return internetServiceResponse.InternetService, nil
} 

func (s *Service) DeleteInternetService (internetServiceId string) (error) {
    deleteInternetServiceRequest := osc.DeleteInternetServiceRequest{InternetServiceId: internetServiceId}
    OscApiClient := s.scope.Api()
    OscAuthClient := s.scope.Auth()
    _, httpRes, err :=  OscApiClient.InternetServiceApi.DeleteInternetService(OscAuthClient).DeleteInternetServiceRequest(deleteInternetServiceRequest).Execute()
    if err != nil {
        fmt.Sprintf("Error with http result %s", httpRes.Status)
        return err
    }
    return nil
}

func (s *Service) LinkInternetService(internetServiceId string, netId string) (error) {
    linkInternetServiceRequest := osc.LinkInternetServiceRequest{
        InternetServiceId: internetServiceId,
        NetId: netId,
    }
    OscApiClient := s.scope.Api()
    OscAuthClient := s.scope.Auth()
    _, httpRes, err := OscApiClient.InternetServiceApi.LinkInternetService(OscAuthClient).LinkInternetServiceRequest(linkInternetServiceRequest).Execute()
    if err != nil {
        fmt.Sprintf("Error with http result %s", httpRes.Status)
        return err
    }
    return nil
}

func (s *Service) UnlinkInternetService(internetServiceId string, netId string) (error) {
    unlinkInternetServiceRequest := osc.UnlinkInternetServiceRequest{
        InternetServiceId: internetServiceId,
        NetId: netId,
    }
    OscApiClient := s.scope.Api()
    OscAuthClient := s.scope.Auth()
    _, httpRes, err := OscApiClient.InternetServiceApi.UnlinkInternetService(OscAuthClient).UnlinkInternetServiceRequest(unlinkInternetServiceRequest).Execute()
    if err != nil {
        fmt.Sprintf("Error with http result %s", httpRes.Status)
        return err
    }
    return nil    
} 

func (s *Service) GetInternetService(internetServiceId []string) (*osc.InternetService, error) {
    readInternetServiceRequest := osc.ReadInternetServicesRequest{
        Filters: &osc.FiltersInternetService{
            InternetServiceIds: &internetServiceId,
        },
    }
    OscApiClient := s.scope.Api()
    OscAuthClient := s.scope.Auth()
    readInternetServiceResponse, httpRes, err := OscApiClient.InternetServiceApi.ReadInternetServices(OscAuthClient).ReadInternetServicesRequest(readInternetServiceRequest).Execute()
    if err != nil {
        fmt.Sprintf("Error with http result %s", httpRes.Status)
        return nil, err
    }
    var internetservice []osc.InternetService
    internetservices := *readInternetServiceResponse.InternetServices
    if len(internetservices) == 0 {
        return nil, nil
    } else {
        internetservice = append(internetservice, internetservices...)
        return &internetservice[0],nil
    }
}
