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

package controllers

import (
	"context"
	"testing"

	"fmt"

	"github.com/stretchr/testify/assert"

	"github.com/golang/mock/gomock"
	infrastructurev1beta2 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta2"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/security/mock_security"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag/mock_tag"
	osc "github.com/outscale/osc-sdk-go/v2"
)

var (
	defaultRouteTableGatewayInitialize = infrastructurev1beta2.OscClusterSpec{
		Network: infrastructurev1beta2.OscNetwork{
			ClusterName: "test-cluster",
			Net: infrastructurev1beta2.OscNet{
				Name:    "test-net",
				IpRange: "10.0.0.0/16",
			},
			Subnets: []*infrastructurev1beta2.OscSubnet{
				{
					Name:          "test-subnet",
					IpSubnetRange: "10.0.0.0/24",
				},
			},
			InternetService: infrastructurev1beta2.OscInternetService{
				Name: "test-internetservice",
			},
			RouteTables: []*infrastructurev1beta2.OscRouteTable{
				{
					Name: "test-routetable",
					Subnets: []string{
						"test-subnet",
					},
					Routes: []infrastructurev1beta2.OscRoute{
						{
							Name:        "test-route",
							TargetName:  "test-internetservice",
							TargetType:  "gateway",
							Destination: "0.0.0.0/0",
						},
					},
				},
			},
		},
	}
	defaultRouteTableGatewayReconcile = infrastructurev1beta2.OscClusterSpec{
		Network: infrastructurev1beta2.OscNetwork{
			ClusterName: "test-cluster",
			Net: infrastructurev1beta2.OscNet{
				Name:       "test-net",
				IpRange:    "10.0.0.0/16",
				ResourceId: "vpc-test-net-uid",
			},
			Subnets: []*infrastructurev1beta2.OscSubnet{
				{
					Name:          "test-subnet",
					IpSubnetRange: "10.0.0.0/24",
					ResourceId:    "subnet-test-subnet-uid",
				},
			},
			InternetService: infrastructurev1beta2.OscInternetService{
				Name:       "test-internetservice",
				ResourceId: "igw-test-interneetservice-uid",
			},
			NatService: infrastructurev1beta2.OscNatService{
				Name:         "test-natservice",
				PublicIpName: "test-publicip",
				SubnetName:   "test-subnet",
				ResourceId:   "nat-test-natservice-uid",
			},
			RouteTables: []*infrastructurev1beta2.OscRouteTable{
				{
					Name: "test-routetable",
					Subnets: []string{
						"test-subnet",
					},
					ResourceId: "rtb-test-routetable-uid",
					Routes: []infrastructurev1beta2.OscRoute{
						{
							Name:        "test-route",
							TargetName:  "test-natservice",
							TargetType:  "nat",
							Destination: "0.0.0.0/0",
						},
					},
				},
			},
		},
	}

	defaultRouteTableNatInitialize = infrastructurev1beta2.OscClusterSpec{
		Network: infrastructurev1beta2.OscNetwork{
			ClusterName: "test-cluster",
			Net: infrastructurev1beta2.OscNet{
				Name:    "test-net",
				IpRange: "10.0.0.0/16",
			},
			Subnets: []*infrastructurev1beta2.OscSubnet{
				{
					Name:          "test-subnet",
					IpSubnetRange: "10.0.0.0/24",
				},
			},
			InternetService: infrastructurev1beta2.OscInternetService{
				Name: "test-internetservice",
			},
			NatService: infrastructurev1beta2.OscNatService{
				Name:         "test-natservice",
				PublicIpName: "test-publicip",
				SubnetName:   "test-subnet",
			},
			RouteTables: []*infrastructurev1beta2.OscRouteTable{
				{
					Name: "test-routetable",
					Subnets: []string{
						"test-subnet",
					},
					Routes: []infrastructurev1beta2.OscRoute{
						{
							Name:        "test-route",
							TargetName:  "test-natservice",
							TargetType:  "nat",
							Destination: "0.0.0.0/0",
						},
					},
				},
			},
		},
	}

	defaultRouteTableNatReconcile = infrastructurev1beta2.OscClusterSpec{
		Network: infrastructurev1beta2.OscNetwork{
			ClusterName: "test-cluster",
			Net: infrastructurev1beta2.OscNet{
				Name:       "test-net",
				IpRange:    "10.0.0.0/16",
				ResourceId: "vpc-test-net-uid",
			},
			Subnets: []*infrastructurev1beta2.OscSubnet{
				{
					Name:          "test-subnet",
					IpSubnetRange: "10.0.0.0/24",
					ResourceId:    "subnet-test-subnet-uid",
				},
			},
			InternetService: infrastructurev1beta2.OscInternetService{
				Name:       "test-internetservice",
				ResourceId: "igw-test-interneetservice-uid",
			},
			NatService: infrastructurev1beta2.OscNatService{
				Name:         "test-natservice",
				PublicIpName: "test-publicip",
				SubnetName:   "test-subnet",
				ResourceId:   "nat-test-natservice-uid",
			},
			RouteTables: []*infrastructurev1beta2.OscRouteTable{
				{
					Name: "test-routetable",
					Subnets: []string{
						"test-subnet",
					},
					ResourceId: "rtb-test-routetable-uid",
					Routes: []infrastructurev1beta2.OscRoute{
						{
							Name:        "test-route",
							TargetName:  "test-natservice",
							TargetType:  "nat",
							Destination: "0.0.0.0/0",
						},
					},
				},
			},
		},
	}

	defaultRouteTableGatewayNatInitialize = infrastructurev1beta2.OscClusterSpec{
		Network: infrastructurev1beta2.OscNetwork{
			ClusterName: "test-cluster",
			Net: infrastructurev1beta2.OscNet{
				Name:    "test-net",
				IpRange: "10.0.0.0/16",
			},
			Subnets: []*infrastructurev1beta2.OscSubnet{
				{
					Name:          "test-subnet",
					IpSubnetRange: "10.0.0.0/24",
				},
			},
			InternetService: infrastructurev1beta2.OscInternetService{
				Name: "test-internetservice",
			},
			NatService: infrastructurev1beta2.OscNatService{
				Name:         "test-natservice",
				PublicIpName: "test-publicip",
				SubnetName:   "test-subnet",
			},
			RouteTables: []*infrastructurev1beta2.OscRouteTable{
				{
					Name: "test-routetable",
					Subnets: []string{
						"test-subnet",
					},
					Routes: []infrastructurev1beta2.OscRoute{
						{
							Name:        "test-route-nat",
							TargetName:  "test-natservice",
							TargetType:  "nat",
							Destination: "0.0.0.0/0",
						},
						{
							Name:        "test-route-igw",
							TargetName:  "test-internetservice",
							TargetType:  "gateway",
							Destination: "0.0.0.0/0",
						},
					},
				},
			},
		},
	}

	defaultRouteTableGatewayNatReconcile = infrastructurev1beta2.OscClusterSpec{
		Network: infrastructurev1beta2.OscNetwork{
			ClusterName: "test-cluster",
			Net: infrastructurev1beta2.OscNet{
				Name:       "test-net",
				IpRange:    "10.0.0.0/16",
				ResourceId: "vpc-test-net",
			},
			Subnets: []*infrastructurev1beta2.OscSubnet{
				{
					Name:          "test-subnet",
					IpSubnetRange: "10.0.0.0/24",
					ResourceId:    "subnet-test-subnet-uid",
				},
			},
			InternetService: infrastructurev1beta2.OscInternetService{
				Name:       "test-internetservice",
				ResourceId: "igw-test-internetservice-uid",
			},
			NatService: infrastructurev1beta2.OscNatService{
				Name:         "test-natservice",
				PublicIpName: "test-publicip",
				SubnetName:   "test-subnet",
				ResourceId:   "nat-test-natservice-uid",
			},
			RouteTables: []*infrastructurev1beta2.OscRouteTable{
				{
					Name: "test-routetable",
					Subnets: []string{
						"test-subnet",
					},
					ResourceId: "rtb-test-routetable-uid",
					Routes: []infrastructurev1beta2.OscRoute{
						{
							Name:        "test-route-nat",
							TargetName:  "test-natservice",
							TargetType:  "nat",
							Destination: "0.0.0.0/0",
						},
						{
							Name:        "test-route-igw",
							TargetName:  "test-internetservice",
							TargetType:  "gateway",
							Destination: "0.0.0.0/0",
						},
					},
				},
			},
		},
	}
)

// SetupWithRouteTableMock set routeTableMock with clusterScope and osccluster
func SetupWithRouteTableMock(t *testing.T, name string, spec infrastructurev1beta2.OscClusterSpec) (clusterScope *scope.ClusterScope, ctx context.Context, mockOscRouteTableInterface *mock_security.MockOscRouteTableInterface, mockOscTagInterface *mock_tag.MockOscTagInterface) {
	clusterScope = Setup(t, name, spec)
	mockCtrl := gomock.NewController(t)
	mockOscRouteTableInterface = mock_security.NewMockOscRouteTableInterface(mockCtrl)
	mockOscTagInterface = mock_tag.NewMockOscTagInterface(mockCtrl)
	ctx = context.Background()
	return clusterScope, ctx, mockOscRouteTableInterface, mockOscTagInterface
}

// TestGettRouteTableResourceId has several tests to cover the code of the function getRouteTableResourceId
func TestGetRouteTableResourceId(t *testing.T) {
	routeTableTestCases := []struct {
		name                          string
		spec                          infrastructurev1beta2.OscClusterSpec
		expRouteTablesFound           bool
		expGetRouteTableResourceIdErr error
	}{
		{
			name:                          "get RouteTableId",
			spec:                          defaultRouteTableGatewayInitialize,
			expRouteTablesFound:           true,
			expGetRouteTableResourceIdErr: nil,
		},
		{
			name:                          "can not get RouteTableId",
			spec:                          defaultRouteTableGatewayInitialize,
			expRouteTablesFound:           false,
			expGetRouteTableResourceIdErr: fmt.Errorf("test-routetable-uid does not exist"),
		},
	}
	for _, rttc := range routeTableTestCases {
		t.Run(rttc.name, func(t *testing.T) {
			clusterScope := Setup(t, rttc.name, rttc.spec)
			routeTablesRef := clusterScope.GetRouteTablesRef()
			routeTablesRef.ResourceMap = make(map[string]string)

			routeTablesSpec := rttc.spec.Network.RouteTables
			for _, routeTableSpec := range routeTablesSpec {
				routeTableName := routeTableSpec.Name + "-uid"
				routeTableId := "rtb-" + routeTableName
				if rttc.expRouteTablesFound {
					routeTablesRef.ResourceMap[routeTableName] = routeTableId
				}
				routeTableResourceId, err := getRouteTableResourceId(routeTableName, clusterScope)
				if err != nil {
					assert.Equal(t, rttc.expGetRouteTableResourceIdErr, err, "GetRouteTableResourceId() should return the same error")
				} else {
					assert.Nil(t, rttc.expGetRouteTableResourceIdErr)
				}
				t.Logf("Find routeTableResourceId %s\n", routeTableResourceId)
			}
		})
	}
}

// TestGettRouteResourceId has several tests to cover the code of the function getRouteResourceId
func TestGetRouteResourceId(t *testing.T) {
	routeTestCases := []struct {
		name                     string
		spec                     infrastructurev1beta2.OscClusterSpec
		expRouteFound            bool
		expGetRouteResourceIdErr error
	}{
		{
			name:                     "get RouteId",
			spec:                     defaultRouteTableGatewayInitialize,
			expRouteFound:            true,
			expGetRouteResourceIdErr: nil,
		},
		{
			name:                     "can not get RouteId",
			spec:                     defaultRouteTableGatewayInitialize,
			expRouteFound:            false,
			expGetRouteResourceIdErr: fmt.Errorf("test-route-uid does not exist"),
		},
	}
	for _, rtc := range routeTestCases {
		t.Run(rtc.name, func(t *testing.T) {
			clusterScope := Setup(t, rtc.name, rtc.spec)
			routeRef := clusterScope.GetRouteRef()
			routeRef.ResourceMap = make(map[string]string)
			routeTablesSpec := rtc.spec.Network.RouteTables
			for _, routeTableSpec := range routeTablesSpec {
				routesSpec := routeTableSpec.Routes
				routeTableName := routeTableSpec.Name + "-uid"
				routeTableId := "rtb-" + routeTableName
				for _, routeSpec := range routesSpec {
					routeName := routeSpec.Name + "-uid"
					if rtc.expRouteFound {
						routeRef.ResourceMap[routeName] = routeTableId
					}
					routeResourceId, err := getRouteResourceId(routeName, clusterScope)
					if err != nil {
						assert.Equal(t, rtc.expGetRouteResourceIdErr, err, "GetRouteResourceId() should return the same error")
					} else {
						assert.Nil(t, rtc.expGetRouteResourceIdErr)
					}
					t.Logf("Find routeResourceId %s\n", routeResourceId)
				}
			}
		})
	}
}

// TestCheckRouteTableSubnetOscAssociateResourceName has several tests to cover the code of the func checkRouteTableSubnetOscAssociateResourceName
func TestCheckRouteTableSubnetOscAssociateResourceName(t *testing.T) {
	routeTableTestCases := []struct {
		name                                                string
		spec                                                infrastructurev1beta2.OscClusterSpec
		expCheckRouteTableSubnetOscAssociateResourceNameErr error
	}{
		{
			name: "check work without net, routetable and route spec (with default values)",
			spec: infrastructurev1beta2.OscClusterSpec{
				Network: infrastructurev1beta2.OscNetwork{},
			},
			expCheckRouteTableSubnetOscAssociateResourceNameErr: nil,
		},
		{
			name: "check routetable association with subnet",
			spec: defaultRouteTableGatewayInitialize,
			expCheckRouteTableSubnetOscAssociateResourceNameErr: nil,
		},
		{
			name: "check routetable association with bad subnet",
			spec: infrastructurev1beta2.OscClusterSpec{
				Network: infrastructurev1beta2.OscNetwork{
					Net: infrastructurev1beta2.OscNet{
						Name:    "test-net",
						IpRange: "10.0.0.0/16",
					},
					Subnets: []*infrastructurev1beta2.OscSubnet{
						{
							Name:          "test-subnet",
							IpSubnetRange: "10.0.0.0/24",
						},
					},
					InternetService: infrastructurev1beta2.OscInternetService{
						Name: "test-internetservice",
					},
					RouteTables: []*infrastructurev1beta2.OscRouteTable{
						{
							Name: "test-routetable",
							Subnets: []string{
								"test-subnet-test",
							},
							Routes: []infrastructurev1beta2.OscRoute{
								{
									Name:        "test-route",
									TargetName:  "test-internetservice",
									TargetType:  "gateway",
									Destination: "0.0.0.0/0",
								},
							},
						},
					},
				},
			},
			expCheckRouteTableSubnetOscAssociateResourceNameErr: fmt.Errorf("test-subnet-test-uid subnet does not exist in routeTable"),
		},
	}
	for _, rttc := range routeTableTestCases {
		t.Run(rttc.name, func(t *testing.T) {
			clusterScope := Setup(t, rttc.name, rttc.spec)
			err := checkRouteTableSubnetOscAssociateResourceName(clusterScope)
			if err != nil {
				assert.Equal(t, rttc.expCheckRouteTableSubnetOscAssociateResourceNameErr, err, "CheckRouteTableSubnetOscAssociateResourceName() should return the same error")
			} else {
				assert.Nil(t, rttc.expCheckRouteTableSubnetOscAssociateResourceNameErr)
			}
		})
	}
}

// TestCheckRouteTableFormatParameters has several tests to cover the code of the func checkRouteTableFormatParameters
func TestCheckRouteTableFormatParameters(t *testing.T) {
	routeTableTestCases := []struct {
		name                                  string
		spec                                  infrastructurev1beta2.OscClusterSpec
		expCheckRouteTableFormatParametersErr error
	}{
		{
			name: "check work without net, routable and route spec (with default values)",
			spec: infrastructurev1beta2.OscClusterSpec{
				Network: infrastructurev1beta2.OscNetwork{},
			},
			expCheckRouteTableFormatParametersErr: nil,
		},
		{
			name:                                  "check routetable format",
			spec:                                  defaultRouteTableGatewayInitialize,
			expCheckRouteTableFormatParametersErr: nil,
		},
		{
			name: "check Bad Name routetable",
			spec: infrastructurev1beta2.OscClusterSpec{
				Network: infrastructurev1beta2.OscNetwork{
					Net: infrastructurev1beta2.OscNet{
						Name:    "test-net",
						IpRange: "10.0.0.0/16",
					},
					Subnets: []*infrastructurev1beta2.OscSubnet{
						{
							Name:          "test-subnet",
							IpSubnetRange: "10.0.0.0/24",
						},
					},
					InternetService: infrastructurev1beta2.OscInternetService{
						Name: "test-internetservice",
					},
					RouteTables: []*infrastructurev1beta2.OscRouteTable{
						{
							Name: "test-routetable@test",
							Subnets: []string{
								"test-subnet",
							},
							Routes: []infrastructurev1beta2.OscRoute{
								{
									Name:        "test-route",
									TargetName:  "test-internetservice",
									TargetType:  "gateway",
									Destination: "0.0.0.0/0",
								},
							},
						},
					},
				},
			},
			expCheckRouteTableFormatParametersErr: fmt.Errorf("Invalid Tag Name"),
		},
	}
	for _, rttc := range routeTableTestCases {
		t.Run(rttc.name, func(t *testing.T) {
			clusterScope := Setup(t, rttc.name, rttc.spec)
			_, err := checkRouteTableFormatParameters(clusterScope)
			if err != nil {
				assert.Equal(t, err, rttc.expCheckRouteTableFormatParametersErr, "CheckRouteTableFormatParameters() should return the same error")
			} else {
				assert.Nil(t, rttc.expCheckRouteTableFormatParametersErr)
			}
			t.Logf("find all routetablename ")
		})
	}
}

// TestCheckRouteFormatParameters has several tests to cover the code of the func checkRouteFormatParameters
func TestCheckRouteFormatParameters(t *testing.T) {
	routeTestCases := []struct {
		name                             string
		spec                             infrastructurev1beta2.OscClusterSpec
		expCheckRouteFormatParametersErr error
	}{
		{
			name: "check work without net, routetable and route spec (with default values)",
			spec: infrastructurev1beta2.OscClusterSpec{
				Network: infrastructurev1beta2.OscNetwork{},
			},
			expCheckRouteFormatParametersErr: nil,
		},
		{
			name:                             "check route format",
			spec:                             defaultRouteTableGatewayInitialize,
			expCheckRouteFormatParametersErr: nil,
		},
		{
			name: "check Bad Name route",
			spec: infrastructurev1beta2.OscClusterSpec{
				Network: infrastructurev1beta2.OscNetwork{
					Net: infrastructurev1beta2.OscNet{
						Name:    "test-net",
						IpRange: "10.0.0.0/16",
					},
					Subnets: []*infrastructurev1beta2.OscSubnet{
						{
							Name:          "test-subnet",
							IpSubnetRange: "10.0.0.0/24",
						},
					},
					InternetService: infrastructurev1beta2.OscInternetService{
						Name: "test-internetservice",
					},
					RouteTables: []*infrastructurev1beta2.OscRouteTable{
						{
							Name: "test-routetable",
							Subnets: []string{
								"test-subnet",
							},
							Routes: []infrastructurev1beta2.OscRoute{
								{
									Name:        "test-route@test",
									TargetName:  "test-internetservice",
									TargetType:  "gateway",
									Destination: "0.0.0.0/0",
								},
							},
						},
					},
				},
			},
			expCheckRouteFormatParametersErr: fmt.Errorf("Invalid Tag Name"),
		},
		{
			name: "check Bad Ip Range IP route",
			spec: infrastructurev1beta2.OscClusterSpec{
				Network: infrastructurev1beta2.OscNetwork{
					Net: infrastructurev1beta2.OscNet{
						Name:    "test-net",
						IpRange: "10.0.0.0/16",
					},
					Subnets: []*infrastructurev1beta2.OscSubnet{
						{
							Name:          "test-subnet",
							IpSubnetRange: "10.0.0.0/24",
						},
					},
					InternetService: infrastructurev1beta2.OscInternetService{
						Name: "test-internetservice",
					},
					RouteTables: []*infrastructurev1beta2.OscRouteTable{
						{
							Name: "test-routetable",
							Subnets: []string{
								"test-subnet",
							},
							Routes: []infrastructurev1beta2.OscRoute{
								{
									Name:        "test-route",
									TargetName:  "test-internetservice",
									TargetType:  "gateway",
									Destination: "10.0.0.256/16",
								},
							},
						},
					},
				},
			},
			expCheckRouteFormatParametersErr: fmt.Errorf("invalid CIDR address: 10.0.0.256/16"),
		},
		{
			name: "check Bad Ip Range IP route",
			spec: infrastructurev1beta2.OscClusterSpec{
				Network: infrastructurev1beta2.OscNetwork{
					Net: infrastructurev1beta2.OscNet{
						Name:    "test-net",
						IpRange: "10.0.0.0/16",
					},
					Subnets: []*infrastructurev1beta2.OscSubnet{
						{
							Name:          "test-subnet",
							IpSubnetRange: "10.0.0.0/24",
						},
					},
					InternetService: infrastructurev1beta2.OscInternetService{
						Name: "test-internetservice",
					},
					RouteTables: []*infrastructurev1beta2.OscRouteTable{
						{
							Name: "test-routetable",
							Subnets: []string{
								"test-subnet",
							},
							Routes: []infrastructurev1beta2.OscRoute{
								{
									Name:        "test-route",
									TargetName:  "test-internetservice",
									TargetType:  "gateway",
									Destination: "10.0.0.0/36",
								},
							},
						},
					},
				},
			},
			expCheckRouteFormatParametersErr: fmt.Errorf("invalid CIDR address: 10.0.0.0/36"),
		},
	}
	for _, rtc := range routeTestCases {
		t.Run(rtc.name, func(t *testing.T) {
			clusterScope := Setup(t, rtc.name, rtc.spec)

			_, err := checkRouteFormatParameters(clusterScope)
			if err != nil {
				assert.Equal(t, rtc.expCheckRouteFormatParametersErr.Error(), err.Error(), "CheckRouteFormatParameters() should return the same error")
			} else {
				assert.Nil(t, rtc.expCheckRouteFormatParametersErr)
			}
			t.Logf("find all routeName")
		})
	}
}

// TestCheckRouteTableOscDuplicateName has several tests to cover the code of the func checkRouteTableOscDuplicateName
func TestCheckRouteTableOscDuplicateName(t *testing.T) {
	routeTableTestCases := []struct {
		name                                  string
		spec                                  infrastructurev1beta2.OscClusterSpec
		expCheckRouteTableOscDuplicateNameErr error
	}{
		{
			name:                                  "get no duplicate routeTable Name",
			spec:                                  defaultRouteTableGatewayInitialize,
			expCheckRouteTableOscDuplicateNameErr: nil,
		},
		{
			name: "get duplicate routeTable Name",
			spec: infrastructurev1beta2.OscClusterSpec{
				Network: infrastructurev1beta2.OscNetwork{
					Net: infrastructurev1beta2.OscNet{
						Name:    "test-net",
						IpRange: "10.0.0.0/16",
					},
					Subnets: []*infrastructurev1beta2.OscSubnet{
						{
							Name:          "test-subnet",
							IpSubnetRange: "10.0.0.0/24",
						},
					},
					InternetService: infrastructurev1beta2.OscInternetService{
						Name: "test-internetservice",
					},
					RouteTables: []*infrastructurev1beta2.OscRouteTable{
						{
							Name: "test-routetable",
							Subnets: []string{
								"test-subnet",
							},
							Routes: []infrastructurev1beta2.OscRoute{
								{
									Name:        "test-route",
									TargetName:  "test-internetservice",
									TargetType:  "gateway",
									Destination: "0.0.0.0/0",
								},
							},
						},
						{
							Name: "test-routetable",
							Subnets: []string{
								"test-subnet",
							},
							Routes: []infrastructurev1beta2.OscRoute{
								{
									Name:        "test-route",
									TargetName:  "test-internetservice",
									TargetType:  "gateway",
									Destination: "0.0.0.0/0",
								},
							},
						},
					},
				},
			},
			expCheckRouteTableOscDuplicateNameErr: fmt.Errorf("test-routetable already exist"),
		},
	}
	for _, rttc := range routeTableTestCases {
		t.Run(rttc.name, func(t *testing.T) {
			clusterScope := Setup(t, rttc.name, rttc.spec)
			duplicateResourceRouteTableNameErr := checkRouteTableOscDuplicateName(clusterScope)
			if duplicateResourceRouteTableNameErr != nil {
				assert.Equal(t, rttc.expCheckRouteTableOscDuplicateNameErr, duplicateResourceRouteTableNameErr, "checkRouteTableOscDuplicateName() should return the same error")
			} else {
				assert.Nil(t, rttc.expCheckRouteTableOscDuplicateNameErr)
			}
		})
	}
}

// TestCheckRouteOscDuplicateName has several tests to cover the code of the func checkRouteOscDuplicateName
func TestCheckRouteOscDuplicateName(t *testing.T) {
	routeTestCases := []struct {
		name                             string
		spec                             infrastructurev1beta2.OscClusterSpec
		expCheckRouteOscDuplicateNameErr error
	}{
		{
			name: "check work without net, routetable and route spec (with default values)",
			spec: infrastructurev1beta2.OscClusterSpec{
				Network: infrastructurev1beta2.OscNetwork{},
			},
			expCheckRouteOscDuplicateNameErr: nil,
		},
		{
			name:                             "check route duplicate name",
			spec:                             defaultRouteTableGatewayInitialize,
			expCheckRouteOscDuplicateNameErr: nil,
		},
		{
			name:                             "get no route duplicate name",
			spec:                             defaultRouteTableGatewayInitialize,
			expCheckRouteOscDuplicateNameErr: nil,
		},
		{
			name: "check route duplicate  internet service name",
			spec: infrastructurev1beta2.OscClusterSpec{
				Network: infrastructurev1beta2.OscNetwork{
					Net: infrastructurev1beta2.OscNet{
						Name:    "test-net",
						IpRange: "10.0.0.0/16",
					},
					Subnets: []*infrastructurev1beta2.OscSubnet{
						{
							Name:          "test-subnet",
							IpSubnetRange: "10.0.0.0/24",
						},
					},
					InternetService: infrastructurev1beta2.OscInternetService{
						Name: "test-internetservice",
					},
					RouteTables: []*infrastructurev1beta2.OscRouteTable{
						{
							Name: "test-routetable",
							Subnets: []string{
								"test-subnet",
							},
							Routes: []infrastructurev1beta2.OscRoute{
								{
									Name:        "test-route",
									TargetName:  "test-internetservice",
									TargetType:  "gateway",
									Destination: "0.0.0.0/0",
								},
								{
									Name:        "test-route",
									TargetName:  "test-internetservice",
									TargetType:  "gateway",
									Destination: "0.0.0.0/0",
								},
							},
						},
					},
				},
			},
			expCheckRouteOscDuplicateNameErr: fmt.Errorf("test-route already exist"),
		},
		{
			name: "check route duplicate  nat service name",
			spec: infrastructurev1beta2.OscClusterSpec{
				Network: infrastructurev1beta2.OscNetwork{
					Net: infrastructurev1beta2.OscNet{
						Name:    "test-net",
						IpRange: "10.0.0.0/16",
					},
					Subnets: []*infrastructurev1beta2.OscSubnet{
						{
							Name:          "test-subnet",
							IpSubnetRange: "10.0.0.0/24",
						},
					},
					InternetService: infrastructurev1beta2.OscInternetService{
						Name: "test-internetservice",
					},
					RouteTables: []*infrastructurev1beta2.OscRouteTable{
						{
							Name: "test-routetable",
							Subnets: []string{
								"test-subnet",
							},
							Routes: []infrastructurev1beta2.OscRoute{
								{
									Name:        "test-route",
									TargetName:  "test-natservice",
									TargetType:  "nat",
									Destination: "0.0.0.0/0",
								},
								{
									Name:        "test-route",
									TargetName:  "test-natservice",
									TargetType:  "nat",
									Destination: "0.0.0.0/0",
								},
							},
						},
					},
				},
			},
			expCheckRouteOscDuplicateNameErr: fmt.Errorf("test-route already exist"),
		},
		{
			name: "check no routetable",
			spec: infrastructurev1beta2.OscClusterSpec{
				Network: infrastructurev1beta2.OscNetwork{
					Net: infrastructurev1beta2.OscNet{
						Name:    "test-net",
						IpRange: "10.0.0.0/16",
					},
					Subnets: []*infrastructurev1beta2.OscSubnet{
						{
							Name:          "test-subnet",
							IpSubnetRange: "10.0.0.0/24",
						},
					},
					InternetService: infrastructurev1beta2.OscInternetService{
						Name: "test-internetservice",
					},
					RouteTables: []*infrastructurev1beta2.OscRouteTable{
						{},
					},
				},
			},
			expCheckRouteOscDuplicateNameErr: nil,
		},
	}
	for _, rtc := range routeTestCases {
		t.Run(rtc.name, func(t *testing.T) {
			clusterScope := Setup(t, rtc.name, rtc.spec)
			duplicateResourceRouteNameErr := checkRouteOscDuplicateName(clusterScope)
			if duplicateResourceRouteNameErr != nil {
				assert.Equal(t, rtc.expCheckRouteOscDuplicateNameErr, duplicateResourceRouteNameErr, "CheckRouteOscDuplicateName() should return the same error")
			} else {
				assert.Nil(t, rtc.expCheckRouteOscDuplicateNameErr)
			}
		})
	}
}

// TestReconcilerRouteCreate has several tests to cover the code of the function reconcilerRouteCreate
func TestReconcilerRouteCreate(t *testing.T) {
	routeTestCases := []struct {
		name                         string
		spec                         infrastructurev1beta2.OscClusterSpec
		expRouteFound                bool
		expTagFound                  bool
		expInternetServiceFound      bool
		expNatServiceFound           bool
		expCreateRouteFound          bool
		expCreateRouteErr            error
		expGetRouteTableFromRouteErr error
		expReadTagErr                error
		expReconcileRouteErr         error
	}{
		{
			name:                         "create route with internet service (first time reconcile loop)",
			spec:                         defaultRouteTableGatewayInitialize,
			expRouteFound:                false,
			expInternetServiceFound:      true,
			expNatServiceFound:           false,
			expCreateRouteFound:          true,
			expTagFound:                  false,
			expCreateRouteErr:            nil,
			expGetRouteTableFromRouteErr: nil,
			expReadTagErr:                nil,
			expReconcileRouteErr:         nil,
		},
		{
			name:                         "create route with natservice (first time reconcile loop)",
			spec:                         defaultRouteTableNatInitialize,
			expRouteFound:                false,
			expInternetServiceFound:      false,
			expNatServiceFound:           true,
			expTagFound:                  false,
			expCreateRouteFound:          true,
			expCreateRouteErr:            nil,
			expGetRouteTableFromRouteErr: nil,
			expReadTagErr:                nil,
			expReconcileRouteErr:         nil,
		},
		{
			name:                         "create multi route  (first time reconcile loop)",
			spec:                         defaultRouteTableGatewayNatInitialize,
			expRouteFound:                false,
			expTagFound:                  false,
			expInternetServiceFound:      true,
			expNatServiceFound:           true,
			expCreateRouteFound:          true,
			expCreateRouteErr:            nil,
			expGetRouteTableFromRouteErr: nil,
			expReadTagErr:                nil,
			expReconcileRouteErr:         nil,
		},
		{
			name:                         "user delete route without cluster-api",
			spec:                         defaultRouteTableNatReconcile,
			expRouteFound:                false,
			expTagFound:                  false,
			expInternetServiceFound:      false,
			expNatServiceFound:           true,
			expCreateRouteFound:          true,
			expCreateRouteErr:            nil,
			expGetRouteTableFromRouteErr: nil,
			expReadTagErr:                nil,
			expReconcileRouteErr:         nil,
		},
		{
			name:                         "failed to create route",
			spec:                         defaultRouteTableNatInitialize,
			expRouteFound:                false,
			expTagFound:                  false,
			expInternetServiceFound:      false,
			expNatServiceFound:           true,
			expCreateRouteFound:          false,
			expCreateRouteErr:            fmt.Errorf("CreateRoute generic error"),
			expGetRouteTableFromRouteErr: nil,
			expReadTagErr:                nil,
			expReconcileRouteErr:         fmt.Errorf("CreateRoute generic error Can not create route for Osccluster test-system/test-osc"),
		},
	}
	for _, rttc := range routeTestCases {
		t.Run(rttc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscRouteTableInterface, mockOscTagInterface := SetupWithRouteTableMock(t, rttc.name, rttc.spec)

			routeTablesRef := clusterScope.GetRouteTablesRef()
			routeTablesRef.ResourceMap = make(map[string]string)

			internetServiceName := rttc.spec.Network.InternetService.Name + "-uid"
			internetServiceId := "igw-" + internetServiceName
			internetServiceRef := clusterScope.GetInternetServiceRef()
			internetServiceRef.ResourceMap = make(map[string]string)
			if rttc.expInternetServiceFound {
				internetServiceRef.ResourceMap[internetServiceName] = internetServiceId
			}

			natServiceName := rttc.spec.Network.NatService.Name + "-uid"
			natServiceId := "nat-" + natServiceName
			natServiceRef := clusterScope.GetNatServiceRef()
			natServiceRef.ResourceMap = make(map[string]string)
			if rttc.expNatServiceFound {
				natServiceRef.ResourceMap[natServiceName] = natServiceId
			}

			routeRef := clusterScope.GetRouteRef()
			routeRef.ResourceMap = make(map[string]string)

			var associateRouteTableId string
			var resourceId string
			routeTablesSpec := rttc.spec.Network.RouteTables
			for _, routeTableSpec := range routeTablesSpec {
				routeTableName := routeTableSpec.Name + "-uid"
				routeTableId := "rtb-" + routeTableName
				tag := osc.Tag{
					ResourceId: &routeTableId,
				}
				if rttc.expTagFound {
					mockOscTagInterface.
						EXPECT().
						ReadTag(gomock.Eq("Name"), gomock.Eq(routeTableName)).
						Return(&tag, rttc.expReadTagErr)
				}
				routeTablesRef.ResourceMap[routeTableName] = routeTableId
				associateRouteTableId = routeTableId
				routesSpec := routeTableSpec.Routes
				for _, routeSpec := range routesSpec {
					destinationIpRange := routeSpec.Destination
					resourceType := routeSpec.TargetType
					if resourceType == "gateway" {
						resourceId = internetServiceId
					} else {
						resourceId = natServiceId
					}

					route := osc.CreateRouteResponse{
						RouteTable: &osc.RouteTable{
							RouteTableId: &routeTableId,
						},
					}

					readRouteTables := osc.ReadRouteTablesResponse{
						RouteTables: &[]osc.RouteTable{
							*route.RouteTable,
						},
					}
					readRouteTable := *readRouteTables.RouteTables
					if rttc.expRouteFound {
						mockOscRouteTableInterface.
							EXPECT().
							GetRouteTableFromRoute(gomock.Eq(associateRouteTableId), gomock.Eq(resourceId), gomock.Eq(resourceType)).
							Return(&readRouteTable[0], rttc.expGetRouteTableFromRouteErr)
					} else {
						mockOscRouteTableInterface.
							EXPECT().
							GetRouteTableFromRoute(gomock.Eq(associateRouteTableId), gomock.Eq(resourceId), gomock.Eq(resourceType)).
							Return(nil, rttc.expGetRouteTableFromRouteErr)
					}
					if rttc.expCreateRouteFound {
						mockOscRouteTableInterface.
							EXPECT().
							CreateRoute(gomock.Eq(destinationIpRange), gomock.Eq(associateRouteTableId), gomock.Eq(resourceId), gomock.Eq(resourceType)).
							Return(route.RouteTable, rttc.expCreateRouteErr)

					} else {
						mockOscRouteTableInterface.
							EXPECT().
							CreateRoute(gomock.Eq(destinationIpRange), gomock.Eq(associateRouteTableId), gomock.Eq(resourceId), gomock.Eq(resourceType)).
							Return(nil, rttc.expCreateRouteErr)
					}
					reconcileRoute, err := reconcileRoute(ctx, clusterScope, routeSpec, routeTableName, mockOscRouteTableInterface)
					if err != nil {
						assert.Equal(t, rttc.expReconcileRouteErr.Error(), err.Error(), "reconcileRoute() should return the same error")
					} else {
						assert.Nil(t, rttc.expReconcileRouteErr)
					}
					t.Logf("find reconcileRoute %v\n", reconcileRoute)
				}
			}
		})
	}
}

// TestReconcileRouteGet has several tests to cover the code of the function reconcileRouteGet
func TestReconcileRouteGet(t *testing.T) {
	routeTestCases := []struct {
		name                         string
		spec                         infrastructurev1beta2.OscClusterSpec
		expRouteFound                bool
		expTagFound                  bool
		expInternetServiceFound      bool
		expNatServiceFound           bool
		expGetRouteTableFromRouteErr error
		expReadTagErr                error
		expReconcileRouteErr         error
	}{
		{
			name:                         "check reconcile multi route (second time reconcile loop)",
			spec:                         defaultRouteTableGatewayNatReconcile,
			expRouteFound:                true,
			expTagFound:                  false,
			expInternetServiceFound:      true,
			expNatServiceFound:           true,
			expGetRouteTableFromRouteErr: nil,
			expReadTagErr:                nil,
			expReconcileRouteErr:         nil,
		},
		{
			name:                         "check reconcile route with natservice (second time reconcile loop)",
			spec:                         defaultRouteTableNatReconcile,
			expRouteFound:                true,
			expTagFound:                  false,
			expInternetServiceFound:      false,
			expNatServiceFound:           true,
			expGetRouteTableFromRouteErr: nil,
			expReadTagErr:                nil,
			expReconcileRouteErr:         nil,
		},
		{
			name:                         "failed to get route",
			spec:                         defaultRouteTableNatInitialize,
			expRouteFound:                false,
			expTagFound:                  false,
			expInternetServiceFound:      false,
			expNatServiceFound:           true,
			expGetRouteTableFromRouteErr: fmt.Errorf("GetRouteTableFromRoute generic error"),
			expReadTagErr:                nil,
			expReconcileRouteErr:         fmt.Errorf("GetRouteTableFromRoute generic error"),
		},
	}
	for _, rttc := range routeTestCases {
		t.Run(rttc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscRouteTableInterface, mockOscTagInterface := SetupWithRouteTableMock(t, rttc.name, rttc.spec)

			routeTablesRef := clusterScope.GetRouteTablesRef()
			routeTablesRef.ResourceMap = make(map[string]string)

			internetServiceName := rttc.spec.Network.InternetService.Name + "-uid"
			internetServiceId := "igw-" + internetServiceName
			internetServiceRef := clusterScope.GetInternetServiceRef()
			internetServiceRef.ResourceMap = make(map[string]string)
			if rttc.expInternetServiceFound {
				internetServiceRef.ResourceMap[internetServiceName] = internetServiceId
			}

			natServiceName := rttc.spec.Network.NatService.Name + "-uid"
			natServiceId := "nat-" + natServiceName
			natServiceRef := clusterScope.GetNatServiceRef()
			natServiceRef.ResourceMap = make(map[string]string)
			if rttc.expNatServiceFound {
				natServiceRef.ResourceMap[natServiceName] = natServiceId
			}

			routeRef := clusterScope.GetRouteRef()
			routeRef.ResourceMap = make(map[string]string)
			var associateRouteTableId string
			var resourceId string

			routeTablesSpec := rttc.spec.Network.RouteTables
			for _, routeTableSpec := range routeTablesSpec {
				routeTableName := routeTableSpec.Name + "-uid"
				routeTableId := "rtb-" + routeTableName
				routeTablesRef.ResourceMap[routeTableName] = routeTableId
				tag := osc.Tag{
					ResourceId: &routeTableId,
				}
				if rttc.expTagFound {
					mockOscTagInterface.
						EXPECT().
						ReadTag(gomock.Eq("Name"), gomock.Eq(routeTableName)).
						Return(&tag, rttc.expReadTagErr)
				}
				associateRouteTableId = routeTableId
				routesSpec := routeTableSpec.Routes
				for _, routeSpec := range routesSpec {
					resourceType := routeSpec.TargetType
					if resourceType == "gateway" {
						resourceId = internetServiceId
					} else {
						resourceId = natServiceId
					}

					route := osc.CreateRouteResponse{
						RouteTable: &osc.RouteTable{
							RouteTableId: &routeTableId,
						},
					}

					readRouteTables := osc.ReadRouteTablesResponse{
						RouteTables: &[]osc.RouteTable{
							*route.RouteTable,
						},
					}
					readRouteTable := *readRouteTables.RouteTables
					if rttc.expRouteFound {
						mockOscRouteTableInterface.
							EXPECT().
							GetRouteTableFromRoute(gomock.Eq(associateRouteTableId), gomock.Eq(resourceId), gomock.Eq(resourceType)).
							Return(&readRouteTable[0], rttc.expGetRouteTableFromRouteErr)
					} else {
						mockOscRouteTableInterface.
							EXPECT().
							GetRouteTableFromRoute(gomock.Eq(associateRouteTableId), gomock.Eq(resourceId), gomock.Eq(resourceType)).
							Return(nil, rttc.expGetRouteTableFromRouteErr)
					}
					reconcileRoute, err := reconcileRoute(ctx, clusterScope, routeSpec, routeTableName, mockOscRouteTableInterface)
					if err != nil {
						assert.Equal(t, rttc.expReconcileRouteErr, err, "reconcileRoute() should return the same error")
					} else {
						assert.Nil(t, rttc.expReconcileRouteErr)
					}
					t.Logf("find reconcileRoute %v\n", reconcileRoute)
				}
			}
		})
	}
}

// TestReconcileRouteResourceId has several tests to cover the code of the function reconcileRouteResourceId
func TestReconcileRouteResourceId(t *testing.T) {
	routeTestCases := []struct {
		name                    string
		spec                    infrastructurev1beta2.OscClusterSpec
		expInternetServiceFound bool
		expNatServiceFound      bool
		expTagFound             bool
		expReadTagErr           error
		expReconcileRouteErr    error
	}{
		{
			name:                    "natService does not exist",
			spec:                    defaultRouteTableNatInitialize,
			expInternetServiceFound: false,
			expNatServiceFound:      false,
			expTagFound:             false,
			expReadTagErr:           nil,
			expReconcileRouteErr:    fmt.Errorf("test-natservice-uid does not exist"),
		},
		{
			name:                    "internetService does not exist",
			spec:                    defaultRouteTableGatewayInitialize,
			expInternetServiceFound: false,
			expNatServiceFound:      false,
			expTagFound:             false,
			expReadTagErr:           nil,
			expReconcileRouteErr:    fmt.Errorf("test-internetservice-uid does not exist"),
		},
	}
	for _, rttc := range routeTestCases {
		t.Run(rttc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscRouteTableInterface, mockOscTagInterface := SetupWithRouteTableMock(t, rttc.name, rttc.spec)

			internetServiceName := rttc.spec.Network.InternetService.Name + "-uid"
			internetServiceId := "igw-" + internetServiceName
			internetServiceRef := clusterScope.GetInternetServiceRef()
			internetServiceRef.ResourceMap = make(map[string]string)
			if rttc.expInternetServiceFound {
				internetServiceRef.ResourceMap[internetServiceName] = internetServiceId
			}

			natServiceName := rttc.spec.Network.NatService.Name + "-uid"
			natServiceId := "nat-" + natServiceName
			natServiceRef := clusterScope.GetNatServiceRef()
			natServiceRef.ResourceMap = make(map[string]string)
			if rttc.expNatServiceFound {
				natServiceRef.ResourceMap[natServiceName] = natServiceId
			}

			routeTablesSpec := rttc.spec.Network.RouteTables
			for _, routeTableSpec := range routeTablesSpec {
				routeTableName := routeTableSpec.Name + "-uid"
				routeTableId := "rtb-" + routeTableName
				tag := osc.Tag{
					ResourceId: &routeTableId,
				}
				if rttc.expTagFound {
					mockOscTagInterface.
						EXPECT().
						ReadTag(gomock.Eq("Name"), gomock.Eq(routeTableName)).
						Return(&tag, rttc.expReadTagErr)
				}
				routesSpec := routeTableSpec.Routes
				for _, routeSpec := range routesSpec {
					reconcileRoute, err := reconcileRoute(ctx, clusterScope, routeSpec, routeTableName, mockOscRouteTableInterface)
					if err != nil {
						assert.Equal(t, rttc.expReconcileRouteErr, err, "reconcileRoute() should return the same error")
					} else {
						assert.Nil(t, rttc.expReconcileRouteErr)
					}
					t.Logf("find reconcileRoute %v\n", reconcileRoute)
				}
			}
		})
	}
}

// TestReconcileRouteTableCreate has several tests to cover the code of the function reconcileRouteTableCreate
func TestReconcileRouteTableCreate(t *testing.T) {
	routeTestCases := []struct {
		name                             string
		spec                             infrastructurev1beta2.OscClusterSpec
		expNetFound                      bool
		expSubnetFound                   bool
		expRouteFound                    bool
		expRouteTableFound               bool
		expInternetServiceFound          bool
		expNatServiceFound               bool
		expCreateRouteFound              bool
		expCreateRouteTableFound         bool
		expLinkRouteTableFound           bool
		expTagFound                      bool
		expCreateRouteErr                error
		expCreateRouteTableErr           error
		expLinkRouteTableErr             error
		expGetRouteTableFromRouteErr     error
		expGetRouteTableIdsFromNetIdsErr error
		expReadTagErr                    error
		expReconcileRouteTableErr        error
	}{
		{
			name:                             "create routetable with internet service route (first time reconcile loop)",
			spec:                             defaultRouteTableGatewayInitialize,
			expNetFound:                      true,
			expSubnetFound:                   true,
			expRouteFound:                    false,
			expRouteTableFound:               false,
			expInternetServiceFound:          true,
			expNatServiceFound:               false,
			expCreateRouteFound:              true,
			expCreateRouteTableFound:         true,
			expLinkRouteTableFound:           true,
			expTagFound:                      false,
			expCreateRouteErr:                nil,
			expCreateRouteTableErr:           nil,
			expLinkRouteTableErr:             nil,
			expGetRouteTableFromRouteErr:     nil,
			expGetRouteTableIdsFromNetIdsErr: nil,
			expReadTagErr:                    nil,
			expReconcileRouteTableErr:        nil,
		},
		{
			name:                             "failed to create route",
			spec:                             defaultRouteTableGatewayInitialize,
			expNetFound:                      true,
			expSubnetFound:                   true,
			expRouteFound:                    false,
			expRouteTableFound:               false,
			expTagFound:                      false,
			expInternetServiceFound:          true,
			expNatServiceFound:               false,
			expCreateRouteFound:              true,
			expCreateRouteTableFound:         true,
			expLinkRouteTableFound:           true,
			expCreateRouteErr:                fmt.Errorf("CreateRoute generic error"),
			expCreateRouteTableErr:           nil,
			expLinkRouteTableErr:             nil,
			expGetRouteTableFromRouteErr:     nil,
			expGetRouteTableIdsFromNetIdsErr: nil,
			expReadTagErr:                    nil,
			expReconcileRouteTableErr:        fmt.Errorf("CreateRoute generic error Can not create route for Osccluster test-system/test-osc"),
		},
	}
	for _, rttc := range routeTestCases {
		t.Run(rttc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscRouteTableInterface, mockOscTagInterface := SetupWithRouteTableMock(t, rttc.name, rttc.spec)

			netName := rttc.spec.Network.Net.Name + "-uid"
			netId := "vpc-" + netName
			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			if rttc.expNetFound {
				netRef.ResourceMap[netName] = netId
			}
			clusterName := rttc.spec.Network.ClusterName + "-uid"
			routeTablesRef := clusterScope.GetRouteTablesRef()
			routeTablesRef.ResourceMap = make(map[string]string)

			linkRouteTableRef := clusterScope.GetLinkRouteTablesRef()
			if len(linkRouteTableRef) == 0 {
				linkRouteTableRef = make(map[string][]string)
			}
			subnetRef := clusterScope.GetSubnetRef()
			subnetRef.ResourceMap = make(map[string]string)

			internetServiceName := rttc.spec.Network.InternetService.Name + "-uid"
			internetServiceId := "igw-" + internetServiceName
			internetServiceRef := clusterScope.GetInternetServiceRef()
			internetServiceRef.ResourceMap = make(map[string]string)
			if rttc.expInternetServiceFound {
				internetServiceRef.ResourceMap[internetServiceName] = internetServiceId
			}

			natServiceName := rttc.spec.Network.NatService.Name + "-uid"
			natServiceId := "nat-" + natServiceName
			natServiceRef := clusterScope.GetNatServiceRef()
			natServiceRef.ResourceMap = make(map[string]string)
			if rttc.expNatServiceFound {
				natServiceRef.ResourceMap[natServiceName] = natServiceId
			}

			routeRef := clusterScope.GetRouteRef()
			routeRef.ResourceMap = make(map[string]string)

			var associateRouteTableId string
			var routeTableIds []string
			var resourceId string

			routeTablesSpec := rttc.spec.Network.RouteTables
			for _, routeTableSpec := range routeTablesSpec {
				routeTableName := routeTableSpec.Name + "-uid"
				routeTableId := "rtb-" + routeTableName
				tag := osc.Tag{
					ResourceId: &routeTableId,
				}
				if rttc.expTagFound {
					mockOscTagInterface.
						EXPECT().
						ReadTag(gomock.Eq("Name"), gomock.Eq(routeTableName)).
						Return(&tag, rttc.expReadTagErr)
				} else {
					mockOscTagInterface.
						EXPECT().
						ReadTag(gomock.Eq("Name"), gomock.Eq(routeTableName)).
						Return(nil, rttc.expReadTagErr)
				}
				routeTableIds = append(routeTableIds, routeTableId)
				linkRouteTableId := "eipalloc-" + routeTableName
				subnetsSpec := routeTableSpec.Subnets
				for _, subnet := range subnetsSpec {
					subnetName := subnet + "-uid"
					subnetId := "subnet-" + subnetName

					if rttc.expSubnetFound {
						subnetRef.ResourceMap[subnetName] = subnetId
					}

					if rttc.expLinkRouteTableFound {
						linkRouteTableRef[routeTableName] = []string{linkRouteTableId}
					}

					routeTable := osc.CreateRouteTableResponse{
						RouteTable: &osc.RouteTable{
							RouteTableId: &routeTableId,
						},
					}

					linkRouteTable := osc.LinkRouteTableResponse{
						LinkRouteTableId: &linkRouteTableId,
					}

					readRouteTables := osc.ReadRouteTablesResponse{
						RouteTables: &[]osc.RouteTable{
							*routeTable.RouteTable,
						},
					}
					readRouteTable := *readRouteTables.RouteTables
					if rttc.expRouteTableFound {
						mockOscRouteTableInterface.
							EXPECT().
							GetRouteTableIdsFromNetIds(gomock.Eq(netId)).
							Return(routeTableIds, rttc.expGetRouteTableIdsFromNetIdsErr)
					} else {
						mockOscRouteTableInterface.
							EXPECT().
							GetRouteTableIdsFromNetIds(gomock.Eq(netId)).
							Return(nil, rttc.expGetRouteTableIdsFromNetIdsErr)
					}
					if rttc.expCreateRouteTableFound {
						associateRouteTableId = routeTableId
						routeTablesRef.ResourceMap[routeTableName] = routeTableId
						mockOscRouteTableInterface.
							EXPECT().
							CreateRouteTable(gomock.Eq(netId), gomock.Eq(clusterName), gomock.Eq(routeTableName)).
							Return(routeTable.RouteTable, rttc.expCreateRouteTableErr)
					} else {
						mockOscRouteTableInterface.
							EXPECT().
							CreateRouteTable(gomock.Eq(netId), gomock.Eq(netName), gomock.Eq(routeTableName)).
							Return(nil, rttc.expCreateRouteTableErr)
					}

					if rttc.expLinkRouteTableFound {
						mockOscRouteTableInterface.
							EXPECT().
							LinkRouteTable(gomock.Eq(routeTableId), gomock.Eq(subnetId)).
							Return(*linkRouteTable.LinkRouteTableId, rttc.expLinkRouteTableErr)
					} else {
						mockOscRouteTableInterface.
							EXPECT().
							LinkRouteTable(gomock.Eq(routeTableId), gomock.Eq(subnetId)).
							Return("", rttc.expLinkRouteTableErr)
					}

					routesSpec := routeTableSpec.Routes
					for _, routeSpec := range routesSpec {
						destinationIpRange := routeSpec.Destination
						resourceType := routeSpec.TargetType
						if resourceType == "gateway" {
							resourceId = internetServiceId
						} else {
							resourceId = natServiceId
						}

						route := osc.CreateRouteResponse{
							RouteTable: &osc.RouteTable{
								RouteTableId: &routeTableId,
							},
						}
						if rttc.expRouteFound {
							mockOscRouteTableInterface.
								EXPECT().
								GetRouteTableFromRoute(gomock.Eq(associateRouteTableId), gomock.Eq(resourceId), gomock.Eq(resourceType)).
								Return(&readRouteTable[0], rttc.expGetRouteTableFromRouteErr)
						} else {
							mockOscRouteTableInterface.
								EXPECT().
								GetRouteTableFromRoute(gomock.Eq(associateRouteTableId), gomock.Eq(resourceId), gomock.Eq(resourceType)).
								Return(nil, rttc.expGetRouteTableFromRouteErr)
						}
						if rttc.expCreateRouteFound {
							mockOscRouteTableInterface.
								EXPECT().
								CreateRoute(gomock.Eq(destinationIpRange), gomock.Eq(associateRouteTableId), gomock.Eq(resourceId), gomock.Eq(resourceType)).
								Return(route.RouteTable, rttc.expCreateRouteErr)
						} else {
							mockOscRouteTableInterface.
								EXPECT().
								CreateRoute(gomock.Eq(destinationIpRange), gomock.Eq(associateRouteTableId), gomock.Eq(resourceId), gomock.Eq(resourceType)).
								Return(nil, rttc.expCreateRouteErr)
						}
					}
					reconcileRouteTable, err := reconcileRouteTable(ctx, clusterScope, mockOscRouteTableInterface, mockOscTagInterface)
					if err != nil {
						assert.Equal(t, rttc.expReconcileRouteTableErr.Error(), err.Error(), "reconcileRouteTable() should return the same error")
					} else {
						assert.Nil(t, rttc.expReconcileRouteTableErr)
					}
					t.Logf("find reconcileRoute %v\n", reconcileRouteTable)
				}
			}
		})
	}
}

// reconcileRouteTableGet has several tests to cover the code of the function reconcileRouteTableGet
func TestReconcileRouteTableGet(t *testing.T) {
	routeTestCases := []struct {
		name                             string
		spec                             infrastructurev1beta2.OscClusterSpec
		expNetFound                      bool
		expTagFound                      bool
		expSubnetFound                   bool
		expRouteTableFound               bool
		expInternetServiceFound          bool
		expNatServiceFound               bool
		expGetRouteTableIdsFromNetIdsErr error
		expReadTagErr                    error
		expReconcileRouteTableErr        error
	}{
		{
			name:                             "check reconcile routetable  with internet service route (second time reconcile loop)",
			spec:                             defaultRouteTableGatewayReconcile,
			expNetFound:                      true,
			expSubnetFound:                   true,
			expRouteTableFound:               true,
			expInternetServiceFound:          true,
			expNatServiceFound:               false,
			expTagFound:                      true,
			expGetRouteTableIdsFromNetIdsErr: nil,
			expReadTagErr:                    nil,
			expReconcileRouteTableErr:        nil,
		},
		{
			name:                             "failed to get routetable",
			spec:                             defaultRouteTableGatewayInitialize,
			expNetFound:                      true,
			expSubnetFound:                   true,
			expRouteTableFound:               false,
			expInternetServiceFound:          true,
			expNatServiceFound:               false,
			expTagFound:                      false,
			expGetRouteTableIdsFromNetIdsErr: fmt.Errorf("GetRouteTableIdsFromNetIds generic errors"),
			expReadTagErr:                    nil,
			expReconcileRouteTableErr:        fmt.Errorf("GetRouteTableIdsFromNetIds generic errors"),
		},
		{
			name:                             "create routetable with natservice (first time reconcile loop)",
			spec:                             defaultRouteTableNatInitialize,
			expNetFound:                      true,
			expSubnetFound:                   true,
			expTagFound:                      true,
			expRouteTableFound:               true,
			expInternetServiceFound:          false,
			expNatServiceFound:               false,
			expGetRouteTableIdsFromNetIdsErr: nil,
			expReadTagErr:                    nil,
			expReconcileRouteTableErr:        nil,
		},
	}
	for _, rttc := range routeTestCases {
		t.Run(rttc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscRouteTableInterface, mockOscTagInterface := SetupWithRouteTableMock(t, rttc.name, rttc.spec)

			netName := rttc.spec.Network.Net.Name + "-uid"
			netId := "vpc-" + netName
			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			if rttc.expNetFound {
				netRef.ResourceMap[netName] = netId
			}

			routeTablesRef := clusterScope.GetRouteTablesRef()
			routeTablesRef.ResourceMap = make(map[string]string)

			subnetRef := clusterScope.GetSubnetRef()
			subnetRef.ResourceMap = make(map[string]string)

			internetServiceName := rttc.spec.Network.InternetService.Name + "-uid"
			internetServiceId := "igw-" + internetServiceName
			internetServiceRef := clusterScope.GetInternetServiceRef()
			internetServiceRef.ResourceMap = make(map[string]string)
			if rttc.expInternetServiceFound {
				internetServiceRef.ResourceMap[internetServiceName] = internetServiceId
			}

			natServiceName := rttc.spec.Network.NatService.Name + "-uid"
			natServiceId := "nat-" + natServiceName
			natServiceRef := clusterScope.GetNatServiceRef()
			natServiceRef.ResourceMap = make(map[string]string)
			if rttc.expNatServiceFound {
				natServiceRef.ResourceMap[natServiceName] = natServiceId
			}

			var routeTableIds []string

			routeTablesSpec := rttc.spec.Network.RouteTables
			for _, routeTableSpec := range routeTablesSpec {
				routeTableName := routeTableSpec.Name + "-uid"
				routeTableId := "rtb-" + routeTableName
				routeTableIds = append(routeTableIds, routeTableId)
				subnetsSpec := routeTableSpec.Subnets
				for _, subnet := range subnetsSpec {
					subnetName := subnet + "-uid"
					subnetId := "subnet-" + subnetName
					tag := osc.Tag{
						ResourceId: &subnetId,
					}
					if rttc.expTagFound {
						if rttc.expRouteTableFound {
							mockOscTagInterface.
								EXPECT().
								ReadTag(gomock.Eq("Name"), gomock.Eq(routeTableName)).
								Return(&tag, rttc.expReadTagErr)
						}
					}
					if rttc.expSubnetFound {
						subnetRef.ResourceMap[subnetName] = subnetId
					}
					if rttc.expRouteTableFound {
						routeTablesRef.ResourceMap[routeTableName] = routeTableId
					}

					if rttc.expRouteTableFound {
						mockOscRouteTableInterface.
							EXPECT().
							GetRouteTableIdsFromNetIds(gomock.Eq(netId)).
							Return(routeTableIds, rttc.expGetRouteTableIdsFromNetIdsErr)
					} else {
						mockOscRouteTableInterface.
							EXPECT().
							GetRouteTableIdsFromNetIds(gomock.Eq(netId)).
							Return(nil, rttc.expGetRouteTableIdsFromNetIdsErr)
					}
				}

				reconcileRouteTable, err := reconcileRouteTable(ctx, clusterScope, mockOscRouteTableInterface, mockOscTagInterface)
				if err != nil {
					assert.Equal(t, rttc.expReconcileRouteTableErr, err, "reconcileRouteTable() should return the same error")
				} else {
					assert.Nil(t, rttc.expReconcileRouteTableErr)
				}
				t.Logf("find reconcileRoute %v\n", reconcileRouteTable)
			}
		})
	}
}

// TestReconcileRouteTableResourceId has several tests to cover the code of the function reconcileRouteTable
func TestReconcileRouteTableResourceId(t *testing.T) {
	routeTestCases := []struct {
		name                             string
		spec                             infrastructurev1beta2.OscClusterSpec
		expTagFound                      bool
		expNetFound                      bool
		expReadTagErr                    error
		expReconcileRouteTableErr        error
		expGetRouteTableIdsFromNetIdsErr error
	}{
		{
			name:                             "net does not exist",
			spec:                             defaultRouteTableGatewayInitialize,
			expTagFound:                      false,
			expNetFound:                      false,
			expReadTagErr:                    nil,
			expGetRouteTableIdsFromNetIdsErr: nil,
			expReconcileRouteTableErr:        fmt.Errorf("test-net-uid does not exist"),
		},
		{
			name:                             "failed to get tag",
			spec:                             defaultRouteTableGatewayInitialize,
			expTagFound:                      true,
			expNetFound:                      true,
			expGetRouteTableIdsFromNetIdsErr: nil,
			expReadTagErr:                    fmt.Errorf("ReadTag generic error"),
			expReconcileRouteTableErr:        fmt.Errorf("ReadTag generic error Can not get tag for OscCluster test-system/test-osc"),
		},
	}
	for _, rttc := range routeTestCases {
		t.Run(rttc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscRouteTableInterface, mockOscTagInterface := SetupWithRouteTableMock(t, rttc.name, rttc.spec)

			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			netName := rttc.spec.Network.Net.Name + "-uid"
			netId := "vpc-" + netName
			if rttc.expNetFound {
				netRef.ResourceMap[netName] = netId
			}
			routeTablesSpec := rttc.spec.Network.RouteTables
			var routeTableIds []string
			if rttc.expTagFound {
				for _, routeTableSpec := range routeTablesSpec {
					routeTableName := routeTableSpec.Name + "-uid"
					routeTableId := "rtb-" + routeTableName
					routeTableIds = append(routeTableIds, routeTableId)
					mockOscTagInterface.
						EXPECT().
						ReadTag(gomock.Eq("Name"), gomock.Eq(routeTableName)).
						Return(nil, rttc.expReadTagErr)

				}

				mockOscRouteTableInterface.
					EXPECT().
					GetRouteTableIdsFromNetIds(netId).
					Return(routeTableIds, rttc.expGetRouteTableIdsFromNetIdsErr)
			}
			reconcileRouteTable, err := reconcileRouteTable(ctx, clusterScope, mockOscRouteTableInterface, mockOscTagInterface)
			if err != nil {
				assert.Equal(t, rttc.expReconcileRouteTableErr.Error(), err.Error(), "reconcileRouteTable() should return the same error")
			} else {
				assert.Nil(t, rttc.expReconcileRouteTableErr)
			}
			t.Logf("find reconcileRoute %v\n", reconcileRouteTable)
		})
	}
}

// TestReconcileCreateRouteTable has several tests to cover the code of the function reconcileRouteTable
func TestReconcileCreateRouteTable(t *testing.T) {
	routeTestCases := []struct {
		name                             string
		spec                             infrastructurev1beta2.OscClusterSpec
		expTagFound                      bool
		expCreateRouteTableErr           error
		expGetRouteTableIdsFromNetIdsErr error
		expReadTagErr                    error
		expReconcileRouteTableErr        error
	}{
		{
			name: "failed to create routeTable",
			spec: infrastructurev1beta2.OscClusterSpec{
				Network: infrastructurev1beta2.OscNetwork{
					Net: infrastructurev1beta2.OscNet{
						Name:    "test-net",
						IpRange: "10.0.0.0/16",
					},
					Subnets: []*infrastructurev1beta2.OscSubnet{
						{
							Name:          "test-subnet",
							IpSubnetRange: "10.0.0.0/24",
						},
					},
					InternetService: infrastructurev1beta2.OscInternetService{
						Name: "test-internetservice",
					},
					RouteTables: []*infrastructurev1beta2.OscRouteTable{
						{
							Name: "test-routetable",
							Subnets: []string{
								"test-subnet",
							},
							Routes: []infrastructurev1beta2.OscRoute{
								{
									Name:        "test-route",
									TargetName:  "test-internetservice",
									TargetType:  "gateway",
									Destination: "0.0.0.0/0",
								},
							},
						},
					},
				},
			},
			expCreateRouteTableErr:           fmt.Errorf("CreateRouteTable generic error"),
			expGetRouteTableIdsFromNetIdsErr: nil,
			expTagFound:                      false,
			expReadTagErr:                    nil,
			expReconcileRouteTableErr:        fmt.Errorf("CreateRouteTable generic error Can not create routetable for Osccluster test-system/test-osc"),
		},
	}
	for _, rttc := range routeTestCases {
		t.Run(rttc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscRouteTableInterface, mockOscTagInterface := SetupWithRouteTableMock(t, rttc.name, rttc.spec)

			netName := rttc.spec.Network.Net.Name + "-uid"
			clusterName := rttc.spec.Network.ClusterName + "-uid"
			netId := "vpc-" + netName
			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			netRef.ResourceMap[netName] = netId

			routeTablesRef := clusterScope.GetRouteTablesRef()
			routeTablesRef.ResourceMap = make(map[string]string)

			subnetRef := clusterScope.GetSubnetRef()
			subnetRef.ResourceMap = make(map[string]string)
			var routeTableIds []string

			routeTablesSpec := rttc.spec.Network.RouteTables
			for _, routeTableSpec := range routeTablesSpec {
				routeTableName := routeTableSpec.Name + "-uid"
				routeTableId := "rtb-" + routeTableName
				routeTableIds = append(routeTableIds, routeTableId)
				tag := osc.Tag{
					ResourceId: &routeTableId,
				}
				if rttc.expTagFound {
					mockOscTagInterface.
						EXPECT().
						ReadTag(gomock.Eq("Name"), gomock.Eq(routeTableName)).
						Return(&tag, rttc.expReadTagErr)
				} else {
					mockOscTagInterface.
						EXPECT().
						ReadTag(gomock.Eq("Name"), gomock.Eq(routeTableName)).
						Return(nil, rttc.expReadTagErr)
				}

				subnetsSpec := routeTableSpec.Subnets
				for _, subnet := range subnetsSpec {
					subnetName := subnet + "-uid"
					subnetId := "subnet-" + subnetName
					subnetRef.ResourceMap[subnetName] = subnetId
					mockOscRouteTableInterface.
						EXPECT().
						GetRouteTableIdsFromNetIds(gomock.Eq(netId)).
						Return(routeTableIds, rttc.expGetRouteTableIdsFromNetIdsErr)

					mockOscRouteTableInterface.
						EXPECT().
						CreateRouteTable(gomock.Eq(netId), gomock.Eq(clusterName), gomock.Eq(routeTableName)).
						Return(nil, rttc.expCreateRouteTableErr)
				}
				reconcileRouteTable, err := reconcileRouteTable(ctx, clusterScope, mockOscRouteTableInterface, mockOscTagInterface)
				if err != nil {
					assert.Equal(t, rttc.expReconcileRouteTableErr.Error(), err.Error(), "reconcileRouteTable() should return the same error")
				} else {
					assert.Nil(t, rttc.expReconcileRouteTableErr)
				}
				t.Logf("find reconcileRoute %v\n", reconcileRouteTable)
			}
		})
	}
}

// TestReconcileRouteTableLink has several tests to cover the code of the function reconcileRouteTable
func TestReconcileRouteTableLink(t *testing.T) {
	routeTestCases := []struct {
		name                             string
		spec                             infrastructurev1beta2.OscClusterSpec
		expSubnetFound                   bool
		expTagFound                      bool
		expLinkRouteTableFound           bool
		expCreateRouteTableErr           error
		expLinkRouteTableErr             error
		expGetRouteTableIdsFromNetIdsErr error
		expReadTagErr                    error
		expReconcileRouteTableErr        error
	}{
		{
			name:                             "failed to link routeTable",
			spec:                             defaultRouteTableGatewayInitialize,
			expSubnetFound:                   true,
			expTagFound:                      false,
			expLinkRouteTableFound:           true,
			expCreateRouteTableErr:           nil,
			expLinkRouteTableErr:             fmt.Errorf("LinkRouteTable generic error"),
			expGetRouteTableIdsFromNetIdsErr: nil,
			expReadTagErr:                    nil,
			expReconcileRouteTableErr:        fmt.Errorf("LinkRouteTable generic error Can not link routetable with net for Osccluster test-system/test-osc"),
		},
		{
			name:                             "failed to get subnet",
			spec:                             defaultRouteTableGatewayInitialize,
			expSubnetFound:                   false,
			expLinkRouteTableFound:           false,
			expTagFound:                      false,
			expCreateRouteTableErr:           nil,
			expLinkRouteTableErr:             nil,
			expGetRouteTableIdsFromNetIdsErr: nil,
			expReadTagErr:                    nil,
			expReconcileRouteTableErr:        fmt.Errorf("test-subnet-uid does not exist"),
		},
	}
	for _, rttc := range routeTestCases {
		t.Run(rttc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscRouteTableInterface, mockOscTagInterface := SetupWithRouteTableMock(t, rttc.name, rttc.spec)

			netName := rttc.spec.Network.Net.Name + "-uid"
			netId := "vpc-" + netName
			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			netRef.ResourceMap[netName] = netId

			clusterName := rttc.spec.Network.ClusterName + "-uid"
			routeTablesRef := clusterScope.GetRouteTablesRef()
			routeTablesRef.ResourceMap = make(map[string]string)

			linkRouteTableRef := clusterScope.GetLinkRouteTablesRef()
			if len(linkRouteTableRef) == 0 {
				linkRouteTableRef = make(map[string][]string)
			}
			subnetRef := clusterScope.GetSubnetRef()
			subnetRef.ResourceMap = make(map[string]string)

			routeTablesSpec := rttc.spec.Network.RouteTables
			for _, routeTableSpec := range routeTablesSpec {
				routeTableName := routeTableSpec.Name + "-uid"
				routeTableId := "rtb-" + routeTableName
				tag := osc.Tag{
					ResourceId: &routeTableId,
				}
				if rttc.expTagFound {
					mockOscTagInterface.
						EXPECT().
						ReadTag(gomock.Eq("Name"), gomock.Eq(routeTableName)).
						Return(&tag, rttc.expReadTagErr)
				} else {
					mockOscTagInterface.
						EXPECT().
						ReadTag(gomock.Eq("Name"), gomock.Eq(routeTableName)).
						Return(nil, rttc.expReadTagErr)
				}
				subnetsSpec := routeTableSpec.Subnets
				for _, subnet := range subnetsSpec {
					subnetName := subnet + "-uid"
					subnetId := "subnet-" + subnetName
					if rttc.expSubnetFound {
						subnetRef.ResourceMap[subnetName] = subnetId
					}

					routeTable := osc.CreateRouteTableResponse{
						RouteTable: &osc.RouteTable{
							RouteTableId: &routeTableId,
						},
					}

					mockOscRouteTableInterface.
						EXPECT().
						CreateRouteTable(gomock.Eq(netId), gomock.Eq(clusterName), gomock.Eq(routeTableName)).
						Return(routeTable.RouteTable, rttc.expCreateRouteTableErr)
					if rttc.expLinkRouteTableFound {
						mockOscRouteTableInterface.
							EXPECT().
							LinkRouteTable(gomock.Eq(routeTableId), gomock.Eq(subnetId)).
							Return("", rttc.expLinkRouteTableErr)
					}
					mockOscRouteTableInterface.
						EXPECT().
						GetRouteTableIdsFromNetIds(gomock.Eq(netId)).
						Return(nil, rttc.expGetRouteTableIdsFromNetIdsErr)
				}
				reconcileRouteTable, err := reconcileRouteTable(ctx, clusterScope, mockOscRouteTableInterface, mockOscTagInterface)
				if err != nil {
					assert.Equal(t, rttc.expReconcileRouteTableErr.Error(), err.Error(), "reconcileRouteTable() should return the same error")
				} else {
					assert.Nil(t, rttc.expReconcileRouteTableErr)
				}
				t.Logf("find reconcileRoute %v\n", reconcileRouteTable)
			}
		})
	}
}

// TestReconcileDeleteRouteDelete has several tests to cover the code of the function reconcileDeleteRoute
func TestReconcileDeleteRouteDelete(t *testing.T) {
	routeTestCases := []struct {
		name                         string
		spec                         infrastructurev1beta2.OscClusterSpec
		expRouteFound                bool
		expInternetServiceFound      bool
		expNatServiceFound           bool
		expDeleteRouteErr            error
		expGetRouteTableFromRouteErr error
		expReconcileDeleteRouteErr   error
	}{
		{
			name:                         "delete Route with internetservice (first time reconcile loop)",
			spec:                         defaultRouteTableGatewayInitialize,
			expRouteFound:                true,
			expInternetServiceFound:      true,
			expNatServiceFound:           false,
			expDeleteRouteErr:            nil,
			expGetRouteTableFromRouteErr: nil,
			expReconcileDeleteRouteErr:   nil,
		},
		{
			name:                         "delete Route with natservice (first time reconcile loop)",
			spec:                         defaultRouteTableNatReconcile,
			expRouteFound:                true,
			expInternetServiceFound:      false,
			expNatServiceFound:           true,
			expDeleteRouteErr:            nil,
			expGetRouteTableFromRouteErr: nil,
			expReconcileDeleteRouteErr:   nil,
		},
		{
			name:                         "delete Route with internetservice  and gatewayservice (first time reconcile loop)",
			spec:                         defaultRouteTableGatewayNatReconcile,
			expRouteFound:                true,
			expInternetServiceFound:      true,
			expNatServiceFound:           true,
			expDeleteRouteErr:            nil,
			expGetRouteTableFromRouteErr: nil,
			expReconcileDeleteRouteErr:   nil,
		},
		{
			name:                         "failed to delete route",
			spec:                         defaultRouteTableGatewayInitialize,
			expRouteFound:                true,
			expInternetServiceFound:      true,
			expNatServiceFound:           false,
			expDeleteRouteErr:            fmt.Errorf("DeleteRoute generic error"),
			expGetRouteTableFromRouteErr: nil,
			expReconcileDeleteRouteErr:   fmt.Errorf("DeleteRoute generic error Can not delete route for Osccluster test-system/test-osc"),
		},
	}
	for _, rttc := range routeTestCases {
		t.Run(rttc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscRouteTableInterface, _ := SetupWithRouteTableMock(t, rttc.name, rttc.spec)

			internetServiceName := rttc.spec.Network.InternetService.Name + "-uid"
			internetServiceId := "igw-" + internetServiceName
			internetServiceRef := clusterScope.GetInternetServiceRef()
			internetServiceRef.ResourceMap = make(map[string]string)
			if rttc.expInternetServiceFound {
				internetServiceRef.ResourceMap[internetServiceName] = internetServiceId
			}

			natServiceName := rttc.spec.Network.NatService.Name + "-uid"
			natServiceId := "nat-" + natServiceName
			natServiceRef := clusterScope.GetNatServiceRef()
			natServiceRef.ResourceMap = make(map[string]string)
			if rttc.expNatServiceFound {
				natServiceRef.ResourceMap[natServiceName] = natServiceId
			}

			routeRef := clusterScope.GetRouteRef()
			routeRef.ResourceMap = make(map[string]string)

			routeTablesRef := clusterScope.GetRouteTablesRef()
			routeTablesRef.ResourceMap = make(map[string]string)

			var associateRouteTableId string
			var resourceId string
			routeTablesSpec := rttc.spec.Network.RouteTables
			for _, routeTableSpec := range routeTablesSpec {
				routeTableName := routeTableSpec.Name + "-uid"
				routeTableId := "rtb-" + routeTableName
				routeTablesRef.ResourceMap[routeTableName] = routeTableId
				associateRouteTableId = routeTableId
				routesSpec := routeTableSpec.Routes
				for _, routeSpec := range routesSpec {
					destinationIpRange := routeSpec.Destination
					resourceType := routeSpec.TargetType
					routeName := routeSpec.Name + "-uid"
					routeRef.ResourceMap[routeName] = routeTableId
					if resourceType == "gateway" {
						resourceId = internetServiceId
					} else {
						resourceId = natServiceId
					}
					route := osc.CreateRouteResponse{
						RouteTable: &osc.RouteTable{
							RouteTableId: &routeTableId,
						},
					}

					readRouteTables := osc.ReadRouteTablesResponse{
						RouteTables: &[]osc.RouteTable{
							*route.RouteTable,
						},
					}

					readRouteTable := *readRouteTables.RouteTables
					if rttc.expRouteFound {
						mockOscRouteTableInterface.
							EXPECT().
							GetRouteTableFromRoute(gomock.Eq(associateRouteTableId), gomock.Eq(resourceId), gomock.Eq(resourceType)).
							Return(&readRouteTable[0], rttc.expGetRouteTableFromRouteErr)
					} else {
						mockOscRouteTableInterface.
							EXPECT().
							GetRouteTableFromRoute(gomock.Eq(associateRouteTableId), gomock.Eq(resourceId), gomock.Eq(resourceType)).
							Return(nil, rttc.expGetRouteTableFromRouteErr)
					}
					mockOscRouteTableInterface.
						EXPECT().
						DeleteRoute(gomock.Eq(destinationIpRange), gomock.Eq(routeTableId)).
						Return(rttc.expDeleteRouteErr)

					reconcileDeleteRoute, err := reconcileDeleteRoute(ctx, clusterScope, routeSpec, routeTableName, mockOscRouteTableInterface)
					if err != nil {
						assert.Equal(t, rttc.expReconcileDeleteRouteErr.Error(), err.Error(), "reconcileDeleteRoute() should return the same error")
					} else {
						assert.Nil(t, rttc.expReconcileDeleteRouteErr)
					}
					t.Logf("Find reconcileDeleteRoute %v\n", reconcileDeleteRoute)

				}
			}
		})
	}
}

// TestReconcileDeleteRouteGet has several tests to cover the code of the function reconcileDeleteRoute
func TestReconcileDeleteRouteGet(t *testing.T) {
	routeTestCases := []struct {
		name                         string
		spec                         infrastructurev1beta2.OscClusterSpec
		expRouteFound                bool
		expInternetServiceFound      bool
		expNatServiceFound           bool
		expGetRouteTableFromRouteErr error
		expReconcileDeleteRouteErr   error
	}{
		{
			name:                         "failed to get route",
			spec:                         defaultRouteTableGatewayInitialize,
			expRouteFound:                false,
			expInternetServiceFound:      true,
			expNatServiceFound:           false,
			expGetRouteTableFromRouteErr: fmt.Errorf("GetRouteTable generic error"),
			expReconcileDeleteRouteErr:   fmt.Errorf("GetRouteTable generic error"),
		},
		{
			name:                         "remove finalizer (user delete route without cluster-api)",
			spec:                         defaultRouteTableGatewayInitialize,
			expRouteFound:                false,
			expInternetServiceFound:      true,
			expNatServiceFound:           true,
			expGetRouteTableFromRouteErr: nil,
			expReconcileDeleteRouteErr:   nil,
		},
	}
	for _, rttc := range routeTestCases {
		t.Run(rttc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscRouteTableInterface, _ := SetupWithRouteTableMock(t, rttc.name, rttc.spec)

			internetServiceName := rttc.spec.Network.InternetService.Name + "-uid"
			internetServiceId := "igw-" + internetServiceName
			internetServiceRef := clusterScope.GetInternetServiceRef()
			internetServiceRef.ResourceMap = make(map[string]string)
			if rttc.expInternetServiceFound {
				internetServiceRef.ResourceMap[internetServiceName] = internetServiceId
			}

			natServiceName := rttc.spec.Network.NatService.Name + "-uid"
			natServiceId := "nat-" + natServiceName
			natServiceRef := clusterScope.GetNatServiceRef()
			natServiceRef.ResourceMap = make(map[string]string)
			if rttc.expNatServiceFound {
				natServiceRef.ResourceMap[natServiceName] = natServiceId
			}

			routeRef := clusterScope.GetRouteRef()
			routeRef.ResourceMap = make(map[string]string)

			routeTablesRef := clusterScope.GetRouteTablesRef()
			routeTablesRef.ResourceMap = make(map[string]string)

			var associateRouteTableId string
			var resourceId string
			routeTablesSpec := rttc.spec.Network.RouteTables
			for _, routeTableSpec := range routeTablesSpec {
				routeTableName := routeTableSpec.Name + "-uid"
				routeTableId := "rtb-" + routeTableName
				routeTablesRef.ResourceMap[routeTableName] = routeTableId
				associateRouteTableId = routeTableId
				routesSpec := routeTableSpec.Routes
				for _, routeSpec := range routesSpec {
					resourceType := routeSpec.TargetType
					routeName := routeSpec.Name + "-uid"
					routeRef.ResourceMap[routeName] = routeTableId

					if resourceType == "gateway" {
						resourceId = internetServiceId
					} else {
						resourceId = natServiceId
					}
					route := osc.CreateRouteResponse{
						RouteTable: &osc.RouteTable{
							RouteTableId: &routeTableId,
						},
					}

					readRouteTables := osc.ReadRouteTablesResponse{
						RouteTables: &[]osc.RouteTable{
							*route.RouteTable,
						},
					}

					readRouteTable := *readRouteTables.RouteTables
					if rttc.expRouteFound {
						mockOscRouteTableInterface.
							EXPECT().
							GetRouteTableFromRoute(gomock.Eq(associateRouteTableId), gomock.Eq(resourceId), gomock.Eq(resourceType)).
							Return(&readRouteTable[0], rttc.expGetRouteTableFromRouteErr)
					} else {
						mockOscRouteTableInterface.
							EXPECT().
							GetRouteTableFromRoute(gomock.Eq(associateRouteTableId), gomock.Eq(resourceId), gomock.Eq(resourceType)).
							Return(nil, rttc.expGetRouteTableFromRouteErr)
					}

					reconcileDeleteRoute, err := reconcileDeleteRoute(ctx, clusterScope, routeSpec, routeTableName, mockOscRouteTableInterface)
					if err != nil {
						assert.Equal(t, rttc.expReconcileDeleteRouteErr, err, "reconcileDeleteRoute() should return the same error")
					} else {
						assert.Nil(t, rttc.expReconcileDeleteRouteErr)
					}
					t.Logf("Find reconcileDeleteRoute %v\n", reconcileDeleteRoute)

				}
			}
		})
	}
}

// TestReconcileDeleteRouteResourceId has several tests to cover the code of the function reconcileDeleteRoute
func TestReconcileDeleteRouteResourceId(t *testing.T) {
	routeTestCases := []struct {
		name                       string
		spec                       infrastructurev1beta2.OscClusterSpec
		expInternetServiceFound    bool
		expNatServiceFound         bool
		expReconcileDeleteRouteErr error
	}{
		{
			name:                       "natService does not exist",
			spec:                       defaultRouteTableNatReconcile,
			expInternetServiceFound:    false,
			expNatServiceFound:         false,
			expReconcileDeleteRouteErr: fmt.Errorf("test-natservice-uid does not exist"),
		},
		{
			name:                       "internetService does not exist",
			spec:                       defaultRouteTableGatewayInitialize,
			expInternetServiceFound:    false,
			expNatServiceFound:         false,
			expReconcileDeleteRouteErr: fmt.Errorf("test-internetservice-uid does not exist"),
		},
		{
			name:                       "route does not exist",
			spec:                       defaultRouteTableGatewayInitialize,
			expInternetServiceFound:    true,
			expNatServiceFound:         true,
			expReconcileDeleteRouteErr: fmt.Errorf("test-route-uid does not exist"),
		},
	}
	for _, rttc := range routeTestCases {
		t.Run(rttc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscRouteTableInterface, _ := SetupWithRouteTableMock(t, rttc.name, rttc.spec)

			internetServiceName := rttc.spec.Network.InternetService.Name + "-uid"
			internetServiceRef := clusterScope.GetInternetServiceRef()
			internetServiceRef.ResourceMap = make(map[string]string)
			internetServiceId := "igw-" + internetServiceName
			if rttc.expInternetServiceFound {
				internetServiceRef.ResourceMap[internetServiceName] = internetServiceId
			}

			natServiceName := rttc.spec.Network.NatService.Name + "-uid"
			natServiceId := "nat-" + natServiceName
			natServiceRef := clusterScope.GetNatServiceRef()
			natServiceRef.ResourceMap = make(map[string]string)
			if rttc.expNatServiceFound {
				natServiceRef.ResourceMap[natServiceName] = natServiceId
			}

			routeTablesSpec := rttc.spec.Network.RouteTables
			for _, routeTableSpec := range routeTablesSpec {
				routeTableName := routeTableSpec.Name + "-uid"
				routesSpec := routeTableSpec.Routes
				for _, routeSpec := range routesSpec {

					reconcileDeleteRoute, err := reconcileDeleteRoute(ctx, clusterScope, routeSpec, routeTableName, mockOscRouteTableInterface)
					if err != nil {
						assert.Equal(t, rttc.expReconcileDeleteRouteErr, err, "reconcileDeleteRoute() should return the same error")
					} else {
						assert.Nil(t, rttc.expReconcileDeleteRouteErr)
					}
					t.Logf("Find reconcileDeleteRoute %v\n", reconcileDeleteRoute)

				}
			}
		})
	}
}

// TestReconcileDeleteRouteTableDelete has several tests to cover the code of the function reconcileDeleteRouteTable
func TestReconcileDeleteRouteTableDeleteWithoutSpec(t *testing.T) {
	routeTableTestCases := []struct {
		name                             string
		spec                             infrastructurev1beta2.OscClusterSpec
		expUnlinkRouteTableErr           error
		expDeleteRouteErr                error
		expDeleteRouteTableErr           error
		expGetRouteTableFromRouteErr     error
		expGetRouteTableIdsFromNetIdsErr error
		expReconcileDeleteRouteTableErr  error
	}{
		{
			name:                             "delete Routetable with internetservice route (first time reconcile loop) without spec (with default values)",
			expUnlinkRouteTableErr:           nil,
			expDeleteRouteErr:                nil,
			expDeleteRouteTableErr:           nil,
			expGetRouteTableFromRouteErr:     nil,
			expGetRouteTableIdsFromNetIdsErr: nil,
			expReconcileDeleteRouteTableErr:  nil,
		},
	}
	for _, rttc := range routeTableTestCases {
		t.Run(rttc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscRouteTableInterface, _ := SetupWithRouteTableMock(t, rttc.name, rttc.spec)

			netName := "cluster-api-net-uid"
			netId := "vpc-" + netName
			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			netRef.ResourceMap[netName] = netId

			internetServiceName := "cluster-api-internetservice-uid"
			internetServiceId := "igw-" + internetServiceName
			internetServiceRef := clusterScope.GetInternetServiceRef()
			internetServiceRef.ResourceMap = make(map[string]string)
			internetServiceRef.ResourceMap[internetServiceName] = internetServiceId

			natServiceName := "cluster-api-natservice-uid"
			natServiceId := "nat-" + natServiceName
			natServiceRef := clusterScope.GetNatServiceRef()
			natServiceRef.ResourceMap = make(map[string]string)
			natServiceRef.ResourceMap[natServiceName] = natServiceId

			routeRef := clusterScope.GetRouteRef()
			routeRef.ResourceMap = make(map[string]string)

			routeTablesRef := clusterScope.GetRouteTablesRef()
			routeTablesRef.ResourceMap = make(map[string]string)

			linkRouteTableRef := clusterScope.GetLinkRouteTablesRef()
			if len(linkRouteTableRef) == 0 {
				linkRouteTableRef = make(map[string][]string)
			}
			var associateRouteTableId string
			var resourceId string
			var routeTableIds []string
			routeTableName := "cluster-api-routetable-kw-uid"
			routeTableId := "rtb-" + routeTableName
			routeTableIds = append(routeTableIds, routeTableId)
			linkRouteTableId := "eipalloc-" + routeTableName
			routeTablesRef.ResourceMap[routeTableName] = routeTableId
			associateRouteTableId = routeTableId

			linkRouteTableRef[routeTableName] = []string{linkRouteTableId}
			clusterScope.SetLinkRouteTablesRef(linkRouteTableRef)
			mockOscRouteTableInterface.
				EXPECT().
				GetRouteTableIdsFromNetIds(gomock.Eq(netId)).
				Return(routeTableIds, rttc.expGetRouteTableIdsFromNetIdsErr)
			mockOscRouteTableInterface.
				EXPECT().
				UnlinkRouteTable(gomock.Eq(linkRouteTableId)).
				Return(rttc.expUnlinkRouteTableErr)

			mockOscRouteTableInterface.
				EXPECT().
				DeleteRouteTable(gomock.Eq(routeTableId)).
				Return(rttc.expDeleteRouteTableErr)

			destinationIpRange := "0.0.0.0/0"
			resourceType := "nat"
			routeName := "cluster-api-route-kw-uid"
			routeRef.ResourceMap[routeName] = routeTableId

			if resourceType == "gateway" {
				resourceId = internetServiceId
			} else {
				resourceId = natServiceId
			}
			route := osc.CreateRouteResponse{
				RouteTable: &osc.RouteTable{
					RouteTableId: &routeTableId,
				},
			}

			readRouteTables := osc.ReadRouteTablesResponse{
				RouteTables: &[]osc.RouteTable{
					*route.RouteTable,
				},
			}

			readRouteTable := *readRouteTables.RouteTables
			mockOscRouteTableInterface.
				EXPECT().
				GetRouteTableFromRoute(gomock.Eq(associateRouteTableId), gomock.Eq(resourceId), gomock.Eq(resourceType)).
				Return(&readRouteTable[0], rttc.expGetRouteTableFromRouteErr)

			mockOscRouteTableInterface.
				EXPECT().
				DeleteRoute(gomock.Eq(destinationIpRange), gomock.Eq(routeTableId)).
				Return(rttc.expDeleteRouteErr)
			reconcileDeleteRouteTable, err := reconcileDeleteRouteTable(ctx, clusterScope, mockOscRouteTableInterface)
			if err != nil {
				assert.Equal(t, rttc.expReconcileDeleteRouteTableErr, err, "reconcileDeleteRouteTable() should return the same error")
			} else {
				assert.Nil(t, rttc.expReconcileDeleteRouteTableErr)
			}
			t.Logf("Find reconcileDeleteRouteTable %v\n", reconcileDeleteRouteTable)

		})
	}
}

// TestReconcileDeleteRouteTableDelete has several tests to cover the code of the function reconcileDeleteRouteTable
func TestReconcileDeleteRouteTableDelete(t *testing.T) {
	routeTableTestCases := []struct {
		name                             string
		spec                             infrastructurev1beta2.OscClusterSpec
		expNetFound                      bool
		expRouteFound                    bool
		expRouteTableFound               bool
		expInternetServiceFound          bool
		expNatServiceFound               bool
		expLinkRouteTableFound           bool
		expUnlinkRouteTableErr           error
		expDeleteRouteErr                error
		expDeleteRouteTableErr           error
		expGetRouteTableFromRouteErr     error
		expGetRouteTableIdsFromNetIdsErr error
		expReconcileDeleteRouteTableErr  error
	}{
		{
			name:                             "delete Routetable with internetservice route (first time reconcile loop)",
			spec:                             defaultRouteTableGatewayInitialize,
			expNetFound:                      true,
			expRouteFound:                    true,
			expRouteTableFound:               true,
			expInternetServiceFound:          true,
			expNatServiceFound:               false,
			expLinkRouteTableFound:           true,
			expUnlinkRouteTableErr:           nil,
			expDeleteRouteErr:                nil,
			expDeleteRouteTableErr:           nil,
			expGetRouteTableFromRouteErr:     nil,
			expGetRouteTableIdsFromNetIdsErr: nil,
			expReconcileDeleteRouteTableErr:  nil,
		},
		{
			name:                             "failed to delete routetable",
			spec:                             defaultRouteTableGatewayInitialize,
			expNetFound:                      true,
			expRouteFound:                    true,
			expRouteTableFound:               true,
			expInternetServiceFound:          true,
			expNatServiceFound:               false,
			expLinkRouteTableFound:           true,
			expUnlinkRouteTableErr:           nil,
			expDeleteRouteErr:                nil,
			expDeleteRouteTableErr:           fmt.Errorf("DeleteRoutetable generic error"),
			expGetRouteTableFromRouteErr:     nil,
			expGetRouteTableIdsFromNetIdsErr: nil,
			expReconcileDeleteRouteTableErr:  fmt.Errorf("DeleteRoutetable generic error Can not delete routeTable for Osccluster test-system/test-osc"),
		},
	}
	for _, rttc := range routeTableTestCases {
		t.Run(rttc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscRouteTableInterface, _ := SetupWithRouteTableMock(t, rttc.name, rttc.spec)

			netName := rttc.spec.Network.Net.Name + "-uid"
			netId := "vpc-" + netName
			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			if rttc.expNetFound {
				netRef.ResourceMap[netName] = netId
			}

			internetServiceName := rttc.spec.Network.InternetService.Name + "-uid"
			internetServiceId := "igw-" + internetServiceName
			internetServiceRef := clusterScope.GetInternetServiceRef()
			internetServiceRef.ResourceMap = make(map[string]string)
			if rttc.expInternetServiceFound {
				internetServiceRef.ResourceMap[internetServiceName] = internetServiceId
			}

			natServiceName := rttc.spec.Network.NatService.Name + "-uid"
			natServiceId := "nat-" + natServiceName
			natServiceRef := clusterScope.GetNatServiceRef()
			natServiceRef.ResourceMap = make(map[string]string)
			if rttc.expNatServiceFound {
				natServiceRef.ResourceMap[natServiceName] = natServiceId
			}

			routeRef := clusterScope.GetRouteRef()
			routeRef.ResourceMap = make(map[string]string)

			linkRouteTableRef := clusterScope.GetLinkRouteTablesRef()
			if len(linkRouteTableRef) == 0 {
				linkRouteTableRef = make(map[string][]string)
			}

			routeTablesRef := clusterScope.GetRouteTablesRef()
			routeTablesRef.ResourceMap = make(map[string]string)

			var associateRouteTableId string
			var resourceId string
			var routeTableIds []string
			routeTablesSpec := rttc.spec.Network.RouteTables

			for _, routeTableSpec := range routeTablesSpec {
				routeTableName := routeTableSpec.Name + "-uid"
				routeTableId := "rtb-" + routeTableName
				routeTableIds = append(routeTableIds, routeTableId)
				linkRouteTableId := "eipalloc-" + routeTableName
				if rttc.expRouteTableFound {
					routeTablesRef.ResourceMap[routeTableName] = routeTableId
					associateRouteTableId = routeTableId
				}

				if rttc.expLinkRouteTableFound {
					linkRouteTableRef[routeTableName] = []string{linkRouteTableId}
					clusterScope.SetLinkRouteTablesRef(linkRouteTableRef)
				}

				if rttc.expRouteTableFound {
					mockOscRouteTableInterface.
						EXPECT().
						GetRouteTableIdsFromNetIds(gomock.Eq(netId)).
						Return(routeTableIds, rttc.expGetRouteTableIdsFromNetIdsErr)
				} else {
					mockOscRouteTableInterface.
						EXPECT().
						GetRouteTableIdsFromNetIds(gomock.Eq(netId)).
						Return(nil, rttc.expGetRouteTableIdsFromNetIdsErr)
				}
				mockOscRouteTableInterface.
					EXPECT().
					UnlinkRouteTable(gomock.Eq(linkRouteTableId)).
					Return(rttc.expUnlinkRouteTableErr)
				mockOscRouteTableInterface.
					EXPECT().
					DeleteRouteTable(gomock.Eq(routeTableId)).
					Return(rttc.expDeleteRouteTableErr)

				routesSpec := routeTableSpec.Routes
				for _, routeSpec := range routesSpec {
					destinationIpRange := routeSpec.Destination
					resourceType := routeSpec.TargetType
					routeName := routeSpec.Name + "-uid"
					if rttc.expRouteFound {
						routeRef.ResourceMap[routeName] = routeTableId
					}
					if resourceType == "gateway" {
						resourceId = internetServiceId
					} else {
						resourceId = natServiceId
					}
					route := osc.CreateRouteResponse{
						RouteTable: &osc.RouteTable{
							RouteTableId: &routeTableId,
						},
					}

					readRouteTables := osc.ReadRouteTablesResponse{
						RouteTables: &[]osc.RouteTable{
							*route.RouteTable,
						},
					}

					readRouteTable := *readRouteTables.RouteTables
					if rttc.expRouteTableFound {
						mockOscRouteTableInterface.
							EXPECT().
							GetRouteTableFromRoute(gomock.Eq(associateRouteTableId), gomock.Eq(resourceId), gomock.Eq(resourceType)).
							Return(&readRouteTable[0], rttc.expGetRouteTableFromRouteErr)
					} else {
						mockOscRouteTableInterface.
							EXPECT().
							GetRouteTableFromRoute(gomock.Eq(associateRouteTableId), gomock.Eq(resourceId), gomock.Eq(resourceType)).
							Return(nil, rttc.expGetRouteTableFromRouteErr)
					}
					mockOscRouteTableInterface.
						EXPECT().
						DeleteRoute(gomock.Eq(destinationIpRange), gomock.Eq(routeTableId)).
						Return(rttc.expDeleteRouteErr)
				}
			}
			reconcileDeleteRouteTable, err := reconcileDeleteRouteTable(ctx, clusterScope, mockOscRouteTableInterface)
			if err != nil {
				assert.Equal(t, rttc.expReconcileDeleteRouteTableErr.Error(), err.Error(), "reconcileDeleteRouteTable() should return the same error")
			} else {
				assert.Nil(t, rttc.expReconcileDeleteRouteTableErr)
			}
			t.Logf("Find reconcileDeleteRouteTable %v\n", reconcileDeleteRouteTable)

		})
	}
}

// TestReconcileDeleteRouteTableGet has several tests to cover the code of the function reconcileDeleteRouteTable
func TestReconcileDeleteRouteTableGet(t *testing.T) {
	routeTableTestCases := []struct {
		name                             string
		spec                             infrastructurev1beta2.OscClusterSpec
		expNetFound                      bool
		expRouteFound                    bool
		expRouteTableFound               bool
		expInternetServiceFound          bool
		expNatServiceFound               bool
		expGetRouteTableIdsFromNetIdsErr error
		expReconcileDeleteRouteTableErr  error
	}{
		{
			name:                             "failed to get routetable",
			spec:                             defaultRouteTableGatewayInitialize,
			expNetFound:                      true,
			expRouteTableFound:               false,
			expInternetServiceFound:          false,
			expNatServiceFound:               false,
			expGetRouteTableIdsFromNetIdsErr: fmt.Errorf("GetRouteTableIdsFromNetIds generic error"),
			expReconcileDeleteRouteTableErr:  fmt.Errorf("GetRouteTableIdsFromNetIds generic error"),
		},
		{
			name:                             "remove finalizer (delete routetable without cluster-api)",
			spec:                             defaultRouteTableGatewayInitialize,
			expNetFound:                      true,
			expRouteTableFound:               false,
			expInternetServiceFound:          false,
			expNatServiceFound:               false,
			expGetRouteTableIdsFromNetIdsErr: nil,
			expReconcileDeleteRouteTableErr:  nil,
		},
	}
	for _, rttc := range routeTableTestCases {
		t.Run(rttc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscRouteTableInterface, _ := SetupWithRouteTableMock(t, rttc.name, rttc.spec)

			netName := rttc.spec.Network.Net.Name + "-uid"
			netId := "vpc-" + netName
			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			if rttc.expNetFound {
				netRef.ResourceMap[netName] = netId
			}

			internetServiceName := rttc.spec.Network.InternetService.Name + "-uid"
			internetServiceId := "igw-" + internetServiceName
			internetServiceRef := clusterScope.GetInternetServiceRef()
			internetServiceRef.ResourceMap = make(map[string]string)
			if rttc.expInternetServiceFound {
				internetServiceRef.ResourceMap[internetServiceName] = internetServiceId
			}

			natServiceName := rttc.spec.Network.NatService.Name + "-uid"
			natServiceId := "nat-" + natServiceName
			natServiceRef := clusterScope.GetNatServiceRef()
			natServiceRef.ResourceMap = make(map[string]string)
			if rttc.expNatServiceFound {
				natServiceRef.ResourceMap[natServiceName] = natServiceId
			}

			routeTablesRef := clusterScope.GetRouteTablesRef()
			routeTablesRef.ResourceMap = make(map[string]string)

			routeTablesSpec := rttc.spec.Network.RouteTables
			for _, routeTableSpec := range routeTablesSpec {
				routeTableName := routeTableSpec.Name + "-uid"
				routeTableId := "rtb-" + routeTableName
				if rttc.expRouteTableFound {
					routeTablesRef.ResourceMap[routeTableName] = routeTableId
				}

				if rttc.expRouteTableFound {
					mockOscRouteTableInterface.
						EXPECT().
						GetRouteTableIdsFromNetIds(gomock.Eq(netId)).
						Return([]string{routeTableId}, rttc.expGetRouteTableIdsFromNetIdsErr)
				} else {
					mockOscRouteTableInterface.
						EXPECT().
						GetRouteTableIdsFromNetIds(gomock.Eq(netId)).
						Return(nil, rttc.expGetRouteTableIdsFromNetIdsErr)
				}
			}
			reconcileDeleteRouteTable, err := reconcileDeleteRouteTable(ctx, clusterScope, mockOscRouteTableInterface)
			if err != nil {
				assert.Equal(t, rttc.expReconcileDeleteRouteTableErr, err, "reconcileDeleteRouteTable() should return the same error")
			} else {
				assert.Nil(t, rttc.expReconcileDeleteRouteTableErr)
			}
			t.Logf("Find reconcileDeleteRouteTable %v\n", reconcileDeleteRouteTable)

		})
	}
}

// TestReconcileDeleteRouteTableUnlink has several tests to cover the code of the function reconcileDeleteRouteTable
func TestReconcileDeleteRouteTableUnlink(t *testing.T) {
	routeTableTestCases := []struct {
		name                             string
		spec                             infrastructurev1beta2.OscClusterSpec
		expUnlinkRouteTableErr           error
		expDeleteRouteErr                error
		expGetRouteTableFromRouteErr     error
		expGetRouteTableIdsFromNetIdsErr error
		expReconcileDeleteRouteTableErr  error
	}{
		{
			name:                             "failed to unlink routetable",
			spec:                             defaultRouteTableGatewayInitialize,
			expUnlinkRouteTableErr:           fmt.Errorf("UnlinkRouteTable generic error"),
			expDeleteRouteErr:                nil,
			expGetRouteTableFromRouteErr:     nil,
			expGetRouteTableIdsFromNetIdsErr: nil,
			expReconcileDeleteRouteTableErr:  fmt.Errorf("UnlinkRouteTable generic error Can not unlink routeTable for Osccluster test-system/test-osc"),
		},
	}
	for _, rttc := range routeTableTestCases {
		t.Run(rttc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscRouteTableInterface, _ := SetupWithRouteTableMock(t, rttc.name, rttc.spec)

			netName := rttc.spec.Network.Net.Name + "-uid"
			netId := "vpc-" + netName
			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			netRef.ResourceMap[netName] = netId

			internetServiceName := rttc.spec.Network.InternetService.Name + "-uid"
			internetServiceId := "igw-" + internetServiceName
			internetServiceRef := clusterScope.GetInternetServiceRef()
			internetServiceRef.ResourceMap = make(map[string]string)
			internetServiceRef.ResourceMap[internetServiceName] = internetServiceId

			natServiceName := rttc.spec.Network.NatService.Name + "-uid"
			natServiceId := "nat-" + natServiceName
			natServiceRef := clusterScope.GetNatServiceRef()
			natServiceRef.ResourceMap = make(map[string]string)
			natServiceRef.ResourceMap[natServiceName] = natServiceId

			linkRouteTableRef := clusterScope.GetLinkRouteTablesRef()
			linkRouteTableRef = make(map[string][]string)

			routeTablesRef := clusterScope.GetRouteTablesRef()
			routeTablesRef.ResourceMap = make(map[string]string)
			routeRef := clusterScope.GetRouteRef()
			routeRef.ResourceMap = make(map[string]string)

			var associateRouteTableId string
			var resourceId string
			routeTablesSpec := rttc.spec.Network.RouteTables
			for _, routeTableSpec := range routeTablesSpec {
				routeTableName := routeTableSpec.Name + "-uid"
				routeTableId := "rtb-" + routeTableName
				linkRouteTableId := "eipalloc-" + routeTableName
				routeTablesRef.ResourceMap[routeTableName] = routeTableId
				associateRouteTableId = routeTableId

				linkRouteTableRef[routeTableName] = []string{linkRouteTableId}
				clusterScope.SetLinkRouteTablesRef(linkRouteTableRef)
				mockOscRouteTableInterface.
					EXPECT().
					GetRouteTableIdsFromNetIds(gomock.Eq(netId)).
					Return([]string{routeTableId}, rttc.expGetRouteTableIdsFromNetIdsErr)

				mockOscRouteTableInterface.
					EXPECT().
					UnlinkRouteTable(gomock.Eq(linkRouteTableId)).
					Return(rttc.expUnlinkRouteTableErr)

				routesSpec := routeTableSpec.Routes
				for _, routeSpec := range routesSpec {
					destinationIpRange := routeSpec.Destination
					resourceType := routeSpec.TargetType
					routeName := routeSpec.Name + "-uid"
					routeRef.ResourceMap[routeName] = routeTableId
					if resourceType == "gateway" {
						resourceId = internetServiceId
					} else {
						resourceId = natServiceId
					}
					route := osc.CreateRouteResponse{
						RouteTable: &osc.RouteTable{
							RouteTableId: &routeTableId,
						},
					}

					readRouteTables := osc.ReadRouteTablesResponse{
						RouteTables: &[]osc.RouteTable{
							*route.RouteTable,
						},
					}

					readRouteTable := *readRouteTables.RouteTables
					mockOscRouteTableInterface.
						EXPECT().
						GetRouteTableFromRoute(gomock.Eq(associateRouteTableId), gomock.Eq(resourceId), gomock.Eq(resourceType)).
						Return(&readRouteTable[0], rttc.expGetRouteTableFromRouteErr)
					mockOscRouteTableInterface.
						EXPECT().
						DeleteRoute(gomock.Eq(destinationIpRange), gomock.Eq(routeTableId)).
						Return(rttc.expDeleteRouteErr).AnyTimes()
				}
			}
			reconcileDeleteRouteTable, err := reconcileDeleteRouteTable(ctx, clusterScope, mockOscRouteTableInterface)
			if err != nil {
				assert.Equal(t, rttc.expReconcileDeleteRouteTableErr.Error(), err.Error(), "reconcileDeleteRouteTable() should return the same error")
			} else {
				assert.Nil(t, rttc.expReconcileDeleteRouteTableErr)
			}
			t.Logf("Find reconcileDeleteRouteTable %v\n", reconcileDeleteRouteTable)

		})
	}
}

// TestReconcileDeleteRouteDeleteRouteTable has several tests to cover the code of the function reconcileDeleteRouteTable
func TestReconcileDeleteRouteDeleteRouteTable(t *testing.T) {
	routeTableTestCases := []struct {
		name                             string
		spec                             infrastructurev1beta2.OscClusterSpec
		expDeleteRouteErr                error
		expGetRouteTableFromRouteErr     error
		expGetRouteTableIdsFromNetIdsErr error
		expReconcileDeleteRouteTableErr  error
	}{
		{
			name:                             "failed to delete route",
			spec:                             defaultRouteTableGatewayInitialize,
			expDeleteRouteErr:                fmt.Errorf("DeleteRoute generic error"),
			expGetRouteTableFromRouteErr:     nil,
			expGetRouteTableIdsFromNetIdsErr: nil,
			expReconcileDeleteRouteTableErr:  fmt.Errorf("DeleteRoute generic error Can not delete route for Osccluster test-system/test-osc"),
		},
	}
	for _, rttc := range routeTableTestCases {
		t.Run(rttc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscRouteTableInterface, _ := SetupWithRouteTableMock(t, rttc.name, rttc.spec)

			netName := rttc.spec.Network.Net.Name + "-uid"
			netId := "vpc-" + netName
			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			netRef.ResourceMap[netName] = netId

			internetServiceName := rttc.spec.Network.InternetService.Name + "-uid"
			internetServiceId := "igw-" + internetServiceName
			internetServiceRef := clusterScope.GetInternetServiceRef()
			internetServiceRef.ResourceMap = make(map[string]string)
			internetServiceRef.ResourceMap[internetServiceName] = internetServiceId

			natServiceName := rttc.spec.Network.NatService.Name + "-uid"
			natServiceId := "nat-" + natServiceName
			natServiceRef := clusterScope.GetNatServiceRef()
			natServiceRef.ResourceMap = make(map[string]string)

			routeRef := clusterScope.GetRouteRef()
			routeRef.ResourceMap = make(map[string]string)

			linkRouteTableRef := clusterScope.GetLinkRouteTablesRef()
			linkRouteTableRef = make(map[string][]string)

			routeTablesRef := clusterScope.GetRouteTablesRef()
			routeTablesRef.ResourceMap = make(map[string]string)

			var associateRouteTableId string
			var resourceId string
			routeTablesSpec := rttc.spec.Network.RouteTables
			for _, routeTableSpec := range routeTablesSpec {
				routeTableName := routeTableSpec.Name + "-uid"
				routeTableId := "rtb-" + routeTableName
				linkRouteTableId := "eipalloc-" + routeTableName
				routeTablesRef.ResourceMap[routeTableName] = routeTableId
				associateRouteTableId = routeTableId

				linkRouteTableRef[routeTableName] = []string{linkRouteTableId}
				clusterScope.SetLinkRouteTablesRef(linkRouteTableRef)

				mockOscRouteTableInterface.
					EXPECT().
					GetRouteTableIdsFromNetIds(gomock.Eq(netId)).
					Return([]string{routeTableId}, rttc.expGetRouteTableIdsFromNetIdsErr)

				routesSpec := routeTableSpec.Routes
				for _, routeSpec := range routesSpec {
					destinationIpRange := routeSpec.Destination
					resourceType := routeSpec.TargetType
					routeName := routeSpec.Name + "-uid"
					routeRef.ResourceMap[routeName] = routeTableId
					if resourceType == "gateway" {
						resourceId = internetServiceId
					} else {
						resourceId = natServiceId
					}
					route := osc.CreateRouteResponse{
						RouteTable: &osc.RouteTable{
							RouteTableId: &routeTableId,
						},
					}

					readRouteTables := osc.ReadRouteTablesResponse{
						RouteTables: &[]osc.RouteTable{
							*route.RouteTable,
						},
					}

					readRouteTable := *readRouteTables.RouteTables
					mockOscRouteTableInterface.
						EXPECT().
						GetRouteTableFromRoute(gomock.Eq(associateRouteTableId), gomock.Eq(resourceId), gomock.Eq(resourceType)).
						Return(&readRouteTable[0], rttc.expGetRouteTableFromRouteErr)
					mockOscRouteTableInterface.
						EXPECT().
						DeleteRoute(destinationIpRange, routeTableId).
						Return(rttc.expDeleteRouteErr)
				}
			}
			reconcileDeleteRouteTable, err := reconcileDeleteRouteTable(ctx, clusterScope, mockOscRouteTableInterface)
			if err != nil {
				assert.Equal(t, rttc.expReconcileDeleteRouteTableErr.Error(), err.Error(), "reconcileDeleteRouteTable() should return the same error")
			} else {
				assert.Nil(t, rttc.expReconcileDeleteRouteTableErr)
			}
			t.Logf("Find reconcileDeleteRouteTable %v\n", reconcileDeleteRouteTable)

		})
	}
}

// TestReconcileDeleteRouteTableResourceId has several tests to cover the code of the function reconcileDeleteRouteTable
func TestReconcileDeleteRouteTableResourceId(t *testing.T) {
	routeTableTestCases := []struct {
		name                            string
		spec                            infrastructurev1beta2.OscClusterSpec
		expNetFound                     bool
		expReconcileDeleteRouteTableErr error
	}{
		{
			name: "check work without net, routeTable and route spec (with default values)",
			spec: infrastructurev1beta2.OscClusterSpec{
				Network: infrastructurev1beta2.OscNetwork{},
			},
			expNetFound:                     false,
			expReconcileDeleteRouteTableErr: fmt.Errorf("cluster-api-net-uid does not exist"),
		},
		{
			name:                            "net does not exist",
			spec:                            defaultRouteTableNatReconcile,
			expNetFound:                     false,
			expReconcileDeleteRouteTableErr: fmt.Errorf("test-net-uid does not exist"),
		},
	}
	for _, rttc := range routeTableTestCases {
		t.Run(rttc.name, func(t *testing.T) {
			clusterScope, ctx, mockOscRouteTableInterface, _ := SetupWithRouteTableMock(t, rttc.name, rttc.spec)

			netName := rttc.spec.Network.Net.Name + "-uid"
			netId := "vpc-" + netName

			netRef := clusterScope.GetNetRef()
			netRef.ResourceMap = make(map[string]string)
			if rttc.expNetFound {
				netRef.ResourceMap[netName] = netId
			}

			reconcileDeleteRouteTable, err := reconcileDeleteRouteTable(ctx, clusterScope, mockOscRouteTableInterface)
			if err != nil {
				assert.Equal(t, rttc.expReconcileDeleteRouteTableErr, err, "reconcileDeleteRouteTable() should return the same error")
			} else {
				assert.Nil(t, rttc.expReconcileDeleteRouteTableErr)

			}
			t.Logf("Find reconcileDeleteRouteTable %v\n", reconcileDeleteRouteTable)

		})
	}
}
