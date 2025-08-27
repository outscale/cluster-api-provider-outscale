package controllers_test

import (
	"testing"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	tag "github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	"github.com/outscale/cluster-api-provider-outscale/controllers"
	osc "github.com/outscale/osc-sdk-go/v2"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func patchMoveCluster() patchOSCClusterFunc {
	return func(m *infrastructurev1beta1.OscCluster) {
		m.Status = infrastructurev1beta1.OscClusterStatus{}
	}
}

func patchDeleteCluster() patchOSCClusterFunc {
	return func(m *infrastructurev1beta1.OscCluster) {
		m.DeletionTimestamp = ptr.To(metav1.Now())
		if len(m.Finalizers) == 0 {
			m.Finalizers = []string{controllers.OscClusterFinalizer}
		}
	}
}

func patchUseExistingNet() patchOSCClusterFunc {
	return func(m *infrastructurev1beta1.OscCluster) {
		m.Spec.Network.UseExisting.Net = true
	}
}

func patchUseExistingSecurityGroups() patchOSCClusterFunc {
	return func(m *infrastructurev1beta1.OscCluster) {
		m.Spec.Network.UseExisting.SecurityGroups = true
	}
}

func patchRestrictFromIP(ips ...string) patchOSCClusterFunc {
	return func(m *infrastructurev1beta1.OscCluster) {
		m.Spec.Network.AllowFromIPRanges = ips
	}
}

func patchRestrictToIP(ips ...string) patchOSCClusterFunc {
	return func(m *infrastructurev1beta1.OscCluster) {
		m.Spec.Network.AllowToIPRanges = ips
	}
}

func patchAddSGRule(name string, r infrastructurev1beta1.OscSecurityGroupRule) patchOSCClusterFunc {
	return func(m *infrastructurev1beta1.OscCluster) {
		m.Generation++
		for i, sg := range m.Spec.Network.SecurityGroups {
			if sg.Name == name {
				m.Spec.Network.SecurityGroups[i].SecurityGroupRules = append(m.Spec.Network.SecurityGroups[i].SecurityGroupRules, r)
				return
			}
		}
	}
}

func patchAdditionalSGRule(add infrastructurev1beta1.OscAdditionalSecurityRules) patchOSCClusterFunc {
	return func(m *infrastructurev1beta1.OscCluster) {
		m.Generation++
		m.Spec.Network.AdditionalSecurityRules = append(m.Spec.Network.AdditionalSecurityRules, add)
	}
}

func patchIncrementGeneration() patchOSCClusterFunc {
	return func(m *infrastructurev1beta1.OscCluster) {
		m.Generation++
	}
}

func patchSubregions(subregions ...string) patchOSCClusterFunc {
	return func(m *infrastructurev1beta1.OscCluster) {
		m.Spec.Network.Subregions = subregions
	}
}

func patchNATIPFromPool(name string) patchOSCClusterFunc {
	return func(m *infrastructurev1beta1.OscCluster) {
		m.Spec.Network.NatPublicIpPool = name
	}
}

func mockNetFound(id string) mockFunc {
	return func(s *MockCloudServices) {
		s.NetMock.EXPECT().
			GetNet(gomock.Any(), gomock.Eq(id)).
			Return(&osc.Net{NetId: ptr.To(id)}, nil)
	}
}

func mockGetNet(id string, net *osc.Net) mockFunc {
	return func(s *MockCloudServices) {
		s.NetMock.EXPECT().
			GetNet(gomock.Any(), gomock.Eq(id)).
			Return(net, nil)
	}
}

func mockCreateNet(spec infrastructurev1beta1.OscNet, clusterID, netName, netId string) mockFunc {
	return func(s *MockCloudServices) {
		s.NetMock.EXPECT().
			CreateNet(gomock.Any(), gomock.Eq(spec), gomock.Eq(clusterID), gomock.Eq(netName)).
			Return(&osc.Net{NetId: &netId}, nil)
	}
}

func mockDeleteNet(id string) mockFunc {
	return func(s *MockCloudServices) {
		s.NetMock.EXPECT().
			DeleteNet(gomock.Any(), gomock.Eq(id)).
			Return(nil)
	}
}

func mockGetSubnetFromNet(netId, ipRange string, sn *osc.Subnet) mockFunc {
	return func(s *MockCloudServices) {
		s.SubnetMock.EXPECT().
			GetSubnetFromNet(gomock.Any(), gomock.Eq(netId), gomock.Eq(ipRange)).
			Return(sn, nil)
	}
}

func mockGetSubnet(id string, subnet *osc.Subnet) mockFunc {
	return func(s *MockCloudServices) {
		s.SubnetMock.EXPECT().
			GetSubnet(gomock.Any(), gomock.Eq(id)).
			Return(subnet, nil)
	}
}

func mockSubnetFound(id string) mockFunc {
	return func(s *MockCloudServices) {
		s.SubnetMock.EXPECT().
			GetSubnet(gomock.Any(), gomock.Eq(id)).
			Return(&osc.Subnet{SubnetId: ptr.To(id)}, nil)
	}
}

func mockCreateSubnet(spec infrastructurev1beta1.OscSubnet, netId, clusterID, name, subnetId string) mockFunc {
	return func(s *MockCloudServices) {
		s.SubnetMock.EXPECT().
			CreateSubnet(gomock.Any(), gomock.Eq(spec), gomock.Eq(netId), gomock.Eq(clusterID), gomock.Eq(name)).
			Return(&osc.Subnet{SubnetId: &subnetId, NetId: &netId, IpRange: &spec.IpSubnetRange}, nil)
	}
}

func mockDeleteSubnet(id string) mockFunc {
	return func(s *MockCloudServices) {
		s.SubnetMock.EXPECT().
			DeleteSubnet(gomock.Any(), gomock.Eq(id)).
			Return(nil)
	}
}

func mockInternetServiceFound(netId, id string) mockFunc {
	return func(s *MockCloudServices) {
		s.InternetServiceMock.EXPECT().
			GetInternetServiceForNet(gomock.Any(), gomock.Eq(netId)).
			Return(&osc.InternetService{
				InternetServiceId: ptr.To(id),
				NetId:             ptr.To(netId),
			}, nil)
	}
}

func mockGetInternetServiceForNet(netId string, is *osc.InternetService) mockFunc {
	return func(s *MockCloudServices) {
		s.InternetServiceMock.EXPECT().
			GetInternetServiceForNet(gomock.Any(), gomock.Eq(netId)).
			Return(is, nil)
	}
}

func mockCreateInternetService(name, clusterId, internetServiceId string) mockFunc {
	return func(s *MockCloudServices) {
		s.InternetServiceMock.EXPECT().
			CreateInternetService(gomock.Any(), gomock.Eq(name), gomock.Eq(clusterId)).
			Return(&osc.InternetService{
				InternetServiceId: &internetServiceId,
			}, nil)
	}
}

func mockLinkInternetService(id, netId string) mockFunc {
	return func(s *MockCloudServices) {
		s.InternetServiceMock.EXPECT().
			LinkInternetService(gomock.Any(), gomock.Eq(id), gomock.Eq(netId)).
			Return(nil)
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

func mockGetSecurityGroupFromName(name string, sg *osc.SecurityGroup) mockFunc {
	return func(s *MockCloudServices) {
		s.SecurityGroupMock.EXPECT().
			GetSecurityGroupFromName(gomock.Any(), gomock.Eq(name)).
			Return(sg, nil)
	}
}

func mockGetSecurityGroup(sgId string, sg *osc.SecurityGroup) mockFunc {
	return func(s *MockCloudServices) {
		sg.SecurityGroupId = &sgId
		s.SecurityGroupMock.EXPECT().
			GetSecurityGroup(gomock.Any(), gomock.Eq(sgId)).
			Return(sg, nil)
	}
}

func mockCreateSecurityGroup(netId, clusterId, name, description, tag string, roles []infrastructurev1beta1.OscRole, sgId string) mockFunc {
	return func(s *MockCloudServices) {
		s.SecurityGroupMock.EXPECT().
			CreateSecurityGroup(gomock.Any(), gomock.Eq(netId), gomock.Eq(clusterId), gomock.Eq(name), gomock.Eq(description), gomock.Eq(tag), gomock.Eq(roles)).
			Return(&osc.SecurityGroup{SecurityGroupId: &sgId}, nil)
	}
}

func mockCreateSecurityGroupRule(sg, flow, proto, ipRanges string, portFrom, portTo int32) mockFunc {
	return func(s *MockCloudServices) {
		s.SecurityGroupMock.EXPECT().
			CreateSecurityGroupRule(gomock.Any(), sg, flow, proto, ipRanges, "", portFrom, portTo).
			Return(&osc.SecurityGroup{SecurityGroupId: ptr.To(sg)}, nil)
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

func mockGetRouteTablesFromNet(netId string, rts []osc.RouteTable) mockFunc {
	return func(s *MockCloudServices) {
		s.RouteTableMock.EXPECT().
			GetRouteTablesFromNet(gomock.Any(), gomock.Eq(netId)).
			Return(rts, nil)
	}
}

func mockCreateRouteTable(netId, clusterID, name, routeTableId string) mockFunc {
	return func(s *MockCloudServices) {
		s.RouteTableMock.EXPECT().
			CreateRouteTable(gomock.Any(), gomock.Eq(netId), gomock.Eq(clusterID), gomock.Eq(name)).
			Return(&osc.RouteTable{RouteTableId: &routeTableId}, nil)
	}
}

func mockLinkRouteTable(routeTableId, subnetId string) mockFunc {
	return func(s *MockCloudServices) {
		s.RouteTableMock.EXPECT().
			LinkRouteTable(gomock.Any(), gomock.Eq(routeTableId), gomock.Eq(subnetId)).
			Return("", nil)
	}
}

func mockCreateRoute(routeTableId, dest, resourceId, resourceType string) mockFunc {
	return func(s *MockCloudServices) {
		s.RouteTableMock.EXPECT().
			CreateRoute(gomock.Any(), gomock.Eq(dest), gomock.Eq(routeTableId), gomock.Eq(resourceId), gomock.Eq(resourceType)).
			Return(&osc.RouteTable{}, nil)
	}
}

func mockUnlinkRouteTable(id string) mockFunc {
	return func(s *MockCloudServices) {
		s.RouteTableMock.EXPECT().
			UnlinkRouteTable(gomock.Any(), gomock.Eq(id)).
			Return(nil)
	}
}

func mockDeleteRouteTable(routeTableId string) mockFunc {
	return func(s *MockCloudServices) {
		s.RouteTableMock.EXPECT().
			DeleteRouteTable(gomock.Any(), gomock.Eq(routeTableId)).
			Return(nil)
	}
}

func mockListNatServices(netId string, nats []osc.NatService) mockFunc {
	return func(s *MockCloudServices) {
		s.NatServiceMock.EXPECT().
			ListNatServices(gomock.Any(), gomock.Eq(netId)).
			Return(nats, nil)
	}
}

func mockGetNatServiceFromClientToken(token string, ns *osc.NatService) mockFunc {
	return func(s *MockCloudServices) {
		s.NatServiceMock.EXPECT().
			GetNatServiceFromClientToken(gomock.Any(), gomock.Eq(token)).
			Return(ns, nil)
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

func mockCreateNatService(publicIpId, subnetId, clientToken, name, clusterID, natServiceId string) mockFunc {
	return func(s *MockCloudServices) {
		s.NatServiceMock.EXPECT().
			CreateNatService(gomock.Any(), gomock.Eq(publicIpId), gomock.Eq(subnetId), gomock.Eq(clientToken), gomock.Eq(name), gomock.Eq(clusterID)).
			Return(&osc.NatService{NatServiceId: &natServiceId}, nil)
	}
}

func mockDeleteNatService(id string) mockFunc {
	return func(s *MockCloudServices) {
		s.NatServiceMock.EXPECT().
			DeleteNatService(gomock.Any(), gomock.Eq(id)).
			Return(nil)
	}
}

func mockGetLoadBalancer(name string, lb *osc.LoadBalancer) mockFunc {
	return func(s *MockCloudServices) {
		s.LoadBalancerMock.EXPECT().
			GetLoadBalancer(gomock.Any(), gomock.Eq(name)).
			Return(lb, nil)
	}
}

func mockLoadBalancerFound(name, nameTag string) mockFunc {
	return func(s *MockCloudServices) {
		s.LoadBalancerMock.EXPECT().
			GetLoadBalancer(gomock.Any(), gomock.Eq(name)).
			Return(&osc.LoadBalancer{
				LoadBalancerName: ptr.To(name),
				DnsName:          ptr.To(name + ".lbu.outscale.com"),
				Tags:             &[]osc.ResourceTag{{Key: tag.NameKey, Value: nameTag}},
				HealthCheck:      &osc.HealthCheck{},
			}, nil)
	}
}

func mockCreateLoadBalancer(loadBalancerName, loadBalancerType, subnetId, securityGroupId string) mockFunc {
	return func(s *MockCloudServices) {
		s.LoadBalancerMock.EXPECT().
			CreateLoadBalancer(gomock.Any(), gomock.Cond(func(spec *infrastructurev1beta1.OscLoadBalancer) bool {
				return spec.LoadBalancerName == loadBalancerName && spec.LoadBalancerType == loadBalancerType
			}), gomock.Eq(subnetId), gomock.Eq(securityGroupId)).
			Return(&osc.LoadBalancer{
				LoadBalancerName: &loadBalancerName,
				DnsName:          ptr.To(loadBalancerName + ".outscale.dev"),
				Listeners:        &[]osc.Listener{{LoadBalancerPort: ptr.To[int32](6443)}},
			}, nil)
	}
}

func mockConfigureHealthCheck(loadBalancerName string) mockFunc {
	return func(s *MockCloudServices) {
		s.LoadBalancerMock.EXPECT().
			ConfigureHealthCheck(gomock.Any(), gomock.Cond(func(spec *infrastructurev1beta1.OscLoadBalancer) bool {
				return spec.LoadBalancerName == loadBalancerName
			})).
			Return(&osc.LoadBalancer{
				LoadBalancerName: &loadBalancerName,
				DnsName:          ptr.To(loadBalancerName + ".outscale.dev"),
				Listeners:        &[]osc.Listener{{LoadBalancerPort: ptr.To[int32](6443)}},
			}, nil)
	}
}

func mockCreateLoadBalancerTag(loadBalancerName, nameTag string) mockFunc {
	return func(s *MockCloudServices) {
		s.LoadBalancerMock.EXPECT().
			CreateLoadBalancerTag(gomock.Any(), gomock.Cond(func(spec *infrastructurev1beta1.OscLoadBalancer) bool {
				return spec.LoadBalancerName == loadBalancerName
			}), gomock.Eq(&osc.ResourceTag{Key: tag.NameKey, Value: nameTag})).
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

func mockReadOwnedByTag(rsrcType tag.ResourceType, clusterID string, tag *osc.Tag) mockFunc {
	return func(s *MockCloudServices) {
		s.TagMock.EXPECT().
			ReadOwnedByTag(gomock.Any(), gomock.Eq(rsrcType), gomock.Eq(clusterID)).
			Return(tag, nil).MinTimes(1)
	}
}

func mockCreateVmBastion(vmId, subnetId string, securityGroupIds, privateIps []string, vmName, clientToken, imageId string, vmTags map[string]string) mockFunc {
	created := []osc.BlockDeviceMappingCreated{{
		DeviceName: ptr.To("/dev/sda1"),
		Bsu: &osc.BsuCreated{
			VolumeId: ptr.To(string(defaultRootVolumeId)),
		},
	}}
	return func(s *MockCloudServices) {
		s.VMMock.
			EXPECT().
			CreateVmBastion(gomock.Any(), gomock.Any(),
				gomock.Eq(subnetId), gomock.Eq(securityGroupIds), gomock.Eq(privateIps), gomock.Eq(vmName), gomock.Eq(clientToken), gomock.Eq(imageId), gomock.Eq(vmTags)).
			Return(&osc.Vm{
				VmId:                ptr.To(vmId),
				PrivateDnsName:      ptr.To(defaultPrivateDnsName),
				PrivateIp:           ptr.To(defaultPrivateIp),
				BlockDeviceMappings: &created,
				State:               ptr.To("pending"),
			}, nil)
	}
}

func assertStatusClusterResources(rsrcs infrastructurev1beta1.OscClusterResources) assertOSCClusterFunc {
	return func(t *testing.T, c *infrastructurev1beta1.OscCluster) {
		assert.Equal(t, rsrcs, c.Status.Resources)
	}
}

func assertControlPlaneEndpoint(endpoint string) assertOSCClusterFunc {
	return func(t *testing.T, c *infrastructurev1beta1.OscCluster) {
		assert.Equal(t, endpoint, c.Spec.ControlPlaneEndpoint.Host)
	}
}

func assertHasClusterFinalizer() assertOSCClusterFunc {
	return func(t *testing.T, m *infrastructurev1beta1.OscCluster) {
		assert.True(t, controllerutil.ContainsFinalizer(m, controllers.OscClusterFinalizer))
	}
}
