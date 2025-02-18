package controllers_test

import (
	"github.com/golang/mock/gomock"
	"github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	osc "github.com/outscale/osc-sdk-go/v2"
	"k8s.io/utils/ptr"
)

func patchMoveCluster() patchOSCClusterFunc {
	return func(m *v1beta1.OscCluster) {
		m.Status = v1beta1.OscClusterStatus{}
	}
}

func mockNetFound(id string) mockFunc {
	return func(s *MockCloudServices) {
		s.NetMock.EXPECT().
			GetNet(gomock.Any(), gomock.Eq(id)).
			Return(&osc.Net{NetId: ptr.To(id)}, nil)
	}
}

func mockGetSubnetIdsFromNetIds(net string, subnets []string) mockFunc {
	return func(s *MockCloudServices) {
		s.SubnetMock.EXPECT().
			GetSubnetIdsFromNetIds(gomock.Any(), gomock.Eq(net)).
			Return(subnets, nil)
	}
}

func mockInternetServiceFound(id string) mockFunc {
	return func(s *MockCloudServices) {
		s.InternetServiceMock.EXPECT().
			GetInternetService(gomock.Any(), gomock.Eq(id)).
			Return(&osc.InternetService{
				InternetServiceId: ptr.To(id),
			}, nil)
	}
}

func mockValidatePublicIpIdsOk(ips []string) mockFunc {
	return func(s *MockCloudServices) {
		s.PublicIpMock.EXPECT().
			ValidatePublicIpIds(gomock.Any(), gomock.Eq(ips)).
			Return(ips, nil)
	}
}

func mockSecurityGroupsForNetFound(netId string, sgs []string) mockFunc {
	return func(s *MockCloudServices) {
		s.SecurityGroupMock.EXPECT().
			GetSecurityGroupIdsFromNetIds(gomock.Any(), gomock.Eq(netId)).
			Return(sgs, nil)
	}
}

func mockRouteTablesForNetFound(netId string, rts []string) mockFunc {
	return func(s *MockCloudServices) {
		s.RouteTableMock.EXPECT().
			GetRouteTableIdsFromNetIds(gomock.Any(), gomock.Eq(netId)).
			Return(rts, nil)
	}
}

func mockNatServiceFound(id string) mockFunc {
	return func(s *MockCloudServices) {
		s.NatServiceMock.EXPECT().
			GetNatService(gomock.Any(), gomock.Eq(id)).
			Return(&osc.NatService{
				NatServiceId: ptr.To(id),
			}, nil)
	}
}

func mockLoadBalancerFound(name string) mockFunc {
	return func(s *MockCloudServices) {
		s.LoadBalancerMock.EXPECT().
			GetLoadBalancer(gomock.Any(), gomock.Eq(name)).
			Return(&osc.LoadBalancer{
				LoadBalancerName: ptr.To(name),
			}, nil)
	}
}

func mockLoadBalancerTagFound(name string) mockFunc {
	return func(s *MockCloudServices) {
		s.LoadBalancerMock.EXPECT().
			GetLoadBalancerTag(gomock.Any(), gomock.Any()).
			Return(&osc.LoadBalancerTag{
				LoadBalancerName: ptr.To(name),
				Key:              ptr.To("Name"),
				Value:            ptr.To(name),
			}, nil)
	}
}
