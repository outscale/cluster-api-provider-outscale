package controllers_test

import (
	"github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	osc "github.com/outscale/osc-sdk-go/v2"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func patchMoveCluster() patchOSCClusterFunc {
	return func(m *v1beta1.OscCluster) {
		m.Status = v1beta1.OscClusterStatus{}
	}
}

func patchClusterNoResourceId() patchOSCClusterFunc {
	return func(m *v1beta1.OscCluster) {
		m.Spec.Network.Bastion.ResourceId = ""
		m.Spec.Network.InternetService.ResourceId = ""
		m.Spec.Network.NatService.ResourceId = ""
		m.Spec.Network.Net.ResourceId = ""
		for i := range m.Spec.Network.Subnets {
			m.Spec.Network.Subnets[i].ResourceId = ""
		}
		for i := range m.Spec.Network.PublicIps {
			m.Spec.Network.PublicIps[i].ResourceId = ""
		}
		for i := range m.Spec.Network.RouteTables {
			m.Spec.Network.RouteTables[i].ResourceId = ""
		}
		for i := range m.Spec.Network.SecurityGroups {
			m.Spec.Network.SecurityGroups[i].ResourceId = ""
		}
	}
}

func patchDeleteCluster() patchOSCClusterFunc {
	return func(m *v1beta1.OscCluster) {
		m.DeletionTimestamp = ptr.To(metav1.Now())
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

func mockDeleteNet(id string) mockFunc {
	return func(s *MockCloudServices) {
		s.NetMock.EXPECT().
			DeleteNet(gomock.Any(), gomock.Eq(id)).
			Return(nil)
	}
}

func mockDeleteSubnet(id string) mockFunc {
	return func(s *MockCloudServices) {
		s.SubnetMock.EXPECT().
			DeleteSubnet(gomock.Any(), gomock.Eq(id)).
			Return(nil)
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

func mockUnlinkInternetService(id, netId string) mockFunc {
	return func(s *MockCloudServices) {
		s.InternetServiceMock.EXPECT().
			UnlinkInternetService(gomock.Any(), gomock.Eq(id), gomock.Eq(netId)).
			Return(nil)
	}
}

func mockDeleteInternetService(id string) mockFunc {
	return func(s *MockCloudServices) {
		s.InternetServiceMock.EXPECT().
			DeleteInternetService(gomock.Any(), gomock.Eq(id)).
			Return(nil)
	}
}

func mockValidatePublicIpIdsOk(ips []string) mockFunc {
	return func(s *MockCloudServices) {
		s.PublicIpMock.EXPECT().
			ValidatePublicIpIds(gomock.Any(), gomock.Eq(ips)).
			Return(ips, nil)
	}
}

func mockCheckPublicIpUnlink(ip string) mockFunc {
	return func(s *MockCloudServices) {
		s.PublicIpMock.EXPECT().
			CheckPublicIpUnlink(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Eq(ip)).
			Return(nil)
	}
}

func mockDeletePublicIp(ip string) mockFunc {
	return func(s *MockCloudServices) {
		s.PublicIpMock.EXPECT().
			DeletePublicIp(gomock.Any(), gomock.Eq(ip)).
			Return(nil)
	}
}

func mockSecurityGroupsForNetFound(netId string, sgs []string) mockFunc {
	return func(s *MockCloudServices) {
		s.SecurityGroupMock.EXPECT().
			GetSecurityGroupIdsFromNetIds(gomock.Any(), gomock.Eq(netId)).
			Return(sgs, nil)
	}
}

func mockDeleteSecurityGroupRule(sg, flow, proto, ipRange, sgMember string, fromPort int32, toPort int32) mockFunc {
	return func(s *MockCloudServices) {
		s.SecurityGroupMock.EXPECT().
			DeleteSecurityGroupRule(gomock.Any(), gomock.Eq(sg), gomock.Eq(flow), gomock.Eq(proto), gomock.Eq(ipRange), gomock.Eq(sgMember), gomock.Eq(fromPort), gomock.Eq(toPort)).
			Return(nil)
	}
}

func mockDeleteSecurityGroup(sg string, err error) mockFunc {
	return func(s *MockCloudServices) {
		s.SecurityGroupMock.EXPECT().
			DeleteSecurityGroup(gomock.Any(), gomock.Eq(sg)).
			Return(err)
	}
}

func mockRouteTablesForNetFound(netId string, rts []string) mockFunc {
	return func(s *MockCloudServices) {
		s.RouteTableMock.EXPECT().
			GetRouteTableIdsFromNetIds(gomock.Any(), gomock.Eq(netId)).
			Return(rts, nil)
	}
}

func mockDeleteRoute(id, dest string) mockFunc {
	return func(s *MockCloudServices) {
		s.RouteTableMock.EXPECT().
			DeleteRoute(gomock.Any(), gomock.Eq(dest), gomock.Eq(id)).
			Return(nil)
	}
}

func mockUnlinkRouteTable(id string) mockFunc {
	return func(s *MockCloudServices) {
		s.RouteTableMock.EXPECT().
			UnlinkRouteTable(gomock.Any(), gomock.Eq(id)).
			Return(nil)
	}
}

func mockDeleteRouteTable(id string) mockFunc {
	return func(s *MockCloudServices) {
		s.RouteTableMock.EXPECT().
			DeleteRouteTable(gomock.Any(), gomock.Eq(id)).
			Return(nil)
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

func mockDeleteNatService(id string) mockFunc {
	return func(s *MockCloudServices) {
		s.NatServiceMock.EXPECT().
			DeleteNatService(gomock.Any(), gomock.Eq(id)).
			Return(nil)
	}
}

func mockLoadBalancerFound(name string, found bool) mockFunc {
	if !found {
		return func(s *MockCloudServices) {
			s.LoadBalancerMock.EXPECT().
				GetLoadBalancer(gomock.Any(), gomock.Eq(name)).
				Return(nil, nil)
		}
	}
	return func(s *MockCloudServices) {
		s.LoadBalancerMock.EXPECT().
			GetLoadBalancer(gomock.Any(), gomock.Eq(name)).
			Return(&osc.LoadBalancer{
				LoadBalancerName: ptr.To(name),
				DnsName:          ptr.To(name + ".lbu.outscale.com"),
			}, nil)
	}
}

func mockLoadBalancerTagFound(name, tagValue string) mockFunc {
	return func(s *MockCloudServices) {
		s.LoadBalancerMock.EXPECT().
			GetLoadBalancerTag(gomock.Any(), gomock.Cond(func(spec *infrastructurev1beta1.OscLoadBalancer) bool {
				return spec.LoadBalancerName == name
			})).
			Return(&osc.LoadBalancerTag{
				LoadBalancerName: ptr.To(name),
				Key:              ptr.To("Name"),
				Value:            ptr.To(tagValue),
			}, nil)
	}
}

func mockCheckLoadBalancerDeregisterVm(name string) mockFunc {
	return func(s *MockCloudServices) {
		s.LoadBalancerMock.EXPECT().
			CheckLoadBalancerDeregisterVm(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Cond(func(spec *infrastructurev1beta1.OscLoadBalancer) bool {
				return spec.LoadBalancerName == name
			})).
			Return(nil)
	}
}

func mockDeleteLoadBalancerTag(name string) mockFunc {
	return func(s *MockCloudServices) {
		s.LoadBalancerMock.EXPECT().
			DeleteLoadBalancerTag(gomock.Any(), gomock.Cond(func(spec *infrastructurev1beta1.OscLoadBalancer) bool {
				return spec.LoadBalancerName == name
			}), gomock.Any()).
			Return(nil)
	}
}

func mockDeleteLoadBalancer(name string) mockFunc {
	return func(s *MockCloudServices) {
		s.LoadBalancerMock.EXPECT().
			DeleteLoadBalancer(gomock.Any(), gomock.Cond(func(spec *infrastructurev1beta1.OscLoadBalancer) bool {
				return spec.LoadBalancerName == name
			})).
			Return(nil)
	}
}
