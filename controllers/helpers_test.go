package controllers_test

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/compute"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/compute/mock_compute"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/net"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/net/mock_net"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/security"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/security/mock_security"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/service"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/service/mock_service"
	tag "github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	"github.com/outscale/cluster-api-provider-outscale/cloud/tag/mock_tag"
	osc "github.com/outscale/osc-sdk-go/v2"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

func decode(t *testing.T, file string, into runtime.Object) {
	scheme := runtime.NewScheme()
	codecs := serializer.NewCodecFactory(scheme)
	decode := codecs.UniversalDeserializer().Decode

	buf, err := os.ReadFile(filepath.Join("testdata", file))
	require.NoError(t, err)
	_, _, err = decode(buf, nil, into)
	require.NoError(t, err)
}

type MockCloudServices struct {
	Cloud             *cloud.OscClient
	NetMock           *mock_net.MockOscNetInterface
	SubnetMock        *mock_net.MockOscSubnetInterface
	SecurityGroupMock *mock_security.MockOscSecurityGroupInterface

	InternetServiceMock *mock_net.MockOscInternetServiceInterface
	RouteTableMock      *mock_security.MockOscRouteTableInterface
	NatServiceMock      *mock_net.MockOscNatServiceInterface
	PublicIpMock        *mock_security.MockOscPublicIpInterface
	LoadBalancerMock    *mock_service.MockOscLoadBalancerInterface

	VMMock    *mock_compute.MockOscVmInterface
	ImageMock *mock_compute.MockOscImageInterface

	TagMock *mock_tag.MockOscTagInterface
}

func newMockCloudServices(mockCtrl *gomock.Controller, region string) *MockCloudServices {
	return &MockCloudServices{
		Cloud:             &cloud.OscClient{Region: region},
		NetMock:           mock_net.NewMockOscNetInterface(mockCtrl),
		SubnetMock:        mock_net.NewMockOscSubnetInterface(mockCtrl),
		SecurityGroupMock: mock_security.NewMockOscSecurityGroupInterface(mockCtrl),

		InternetServiceMock: mock_net.NewMockOscInternetServiceInterface(mockCtrl),
		RouteTableMock:      mock_security.NewMockOscRouteTableInterface(mockCtrl),
		NatServiceMock:      mock_net.NewMockOscNatServiceInterface(mockCtrl),
		PublicIpMock:        mock_security.NewMockOscPublicIpInterface(mockCtrl),
		LoadBalancerMock:    mock_service.NewMockOscLoadBalancerInterface(mockCtrl),

		VMMock:    mock_compute.NewMockOscVmInterface(mockCtrl),
		ImageMock: mock_compute.NewMockOscImageInterface(mockCtrl),

		TagMock: mock_tag.NewMockOscTagInterface(mockCtrl),
	}
}

func (s *MockCloudServices) OscClient() *cloud.OscClient {
	return s.Cloud
}

func (s *MockCloudServices) Net(ctx context.Context, scope scope.ClusterScope) net.OscNetInterface {
	return s.NetMock
}

func (s *MockCloudServices) Subnet(ctx context.Context, scope scope.ClusterScope) net.OscSubnetInterface {
	return s.SubnetMock
}

func (s *MockCloudServices) SecurityGroup(ctx context.Context, scope scope.ClusterScope) security.OscSecurityGroupInterface {
	return s.SecurityGroupMock
}

func (s *MockCloudServices) InternetService(ctx context.Context, scope scope.ClusterScope) net.OscInternetServiceInterface {
	return s.InternetServiceMock
}

func (s *MockCloudServices) RouteTable(ctx context.Context, scope scope.ClusterScope) security.OscRouteTableInterface {
	return s.RouteTableMock
}

func (s *MockCloudServices) NatService(ctx context.Context, scope scope.ClusterScope) net.OscNatServiceInterface {
	return s.NatServiceMock
}

func (s *MockCloudServices) PublicIp(ctx context.Context, scope scope.ClusterScope) security.OscPublicIpInterface {
	return s.PublicIpMock
}

func (s *MockCloudServices) LoadBalancer(ctx context.Context, scope scope.ClusterScope) service.OscLoadBalancerInterface {
	return s.LoadBalancerMock
}

func (s *MockCloudServices) VM(ctx context.Context, scope scope.ClusterScope) compute.OscVmInterface {
	return s.VMMock
}

func (s *MockCloudServices) Image(ctx context.Context, scope scope.ClusterScope) compute.OscImageInterface {
	return s.ImageMock
}

func (s *MockCloudServices) Tag(ctx context.Context, scope scope.ClusterScope) tag.OscTagInterface {
	return s.TagMock
}

type patchOSCClusterFunc func(m *v1beta1.OscCluster)
type patchOSCMachineFunc func(m *v1beta1.OscMachine)

type mockFunc func(s *MockCloudServices)

type assertOSCMachineFunc func(t *testing.T, m *v1beta1.OscMachine)
type assertOSCClusterFunc func(t *testing.T, c *v1beta1.OscCluster)

type testcase struct {
	name                             string
	region                           string
	clusterSpec, machineSpec         string
	clusterBaseSpec, machineBaseSpec string
	clusterPatches                   []patchOSCClusterFunc
	machinePatches                   []patchOSCMachineFunc
	mockFuncs                        []mockFunc
	hasError                         bool
	requeue                          bool
	assertDeleted                    bool
	clusterAsserts                   []assertOSCClusterFunc
	machineAsserts                   []assertOSCMachineFunc

	next *testcase
}

var reVersion = regexp.MustCompile("-[0-9.]+$")

func trimVersion(spec string) string {
	return reVersion.ReplaceAllString(spec, "")
}

func loadClusterSpecs(t *testing.T, spec, base string) (*clusterv1.Cluster, *v1beta1.OscCluster) {
	var cluster clusterv1.Cluster
	if base != "" {
		decode(t, "cluster/"+base+".yaml", &cluster)
	} else {
		decode(t, "cluster/"+trimVersion(spec)+".yaml", &cluster)
	}
	var osccluster v1beta1.OscCluster
	decode(t, "osccluster/"+spec+".yaml", &osccluster)
	return &cluster, &osccluster
}

func loadMachineSpecs(t *testing.T, spec, base string) (*clusterv1.Machine, *v1beta1.OscMachine) {
	var machine clusterv1.Machine
	if base != "" {
		decode(t, "machine/"+base+".yaml", &machine)
	} else {
		decode(t, "machine/"+trimVersion(spec)+".yaml", &machine)
	}
	var oscmachine v1beta1.OscMachine
	decode(t, "oscmachine/"+spec+".yaml", &oscmachine)
	return &machine, &oscmachine
}

func mockReadTagByNameNoneFound(typ tag.ResourceType, name string) mockFunc {
	return func(s *MockCloudServices) {
		s.TagMock.EXPECT().
			ReadTag(gomock.Any(), gomock.Eq(typ), gomock.Eq(tag.NameKey), gomock.Eq(name)).
			Return(nil, nil).MinTimes(1)
	}
}

func mockReadTagByNameFound(typ tag.ResourceType, name, resourceId string) mockFunc {
	return func(s *MockCloudServices) {
		s.TagMock.EXPECT().
			ReadTag(gomock.Any(), gomock.Eq(typ), gomock.Eq(tag.NameKey), gomock.Eq(name)).
			Return(&osc.Tag{
				Value:      &name,
				ResourceId: &resourceId,
			}, nil)
	}
}

func mockGetSecurityGroupsFromNet(netId string, sgs []osc.SecurityGroup) mockFunc {
	return func(s *MockCloudServices) {
		s.SecurityGroupMock.EXPECT().
			GetSecurityGroupsFromNet(gomock.Any(), gomock.Eq(netId)).
			Return(sgs, nil)
	}
}

func mockPublicIpFound(publicIpId string) mockFunc {
	return func(s *MockCloudServices) {
		s.PublicIpMock.EXPECT().
			GetPublicIp(gomock.Any(), gomock.Eq(publicIpId)).
			Return(&osc.PublicIp{PublicIpId: &publicIpId}, nil)
	}
}

func mockGetPublicIpByIp(publicIp, publicIpId string) mockFunc {
	return func(s *MockCloudServices) {
		s.PublicIpMock.EXPECT().
			GetPublicIpByIp(gomock.Any(), gomock.Eq(publicIp)).
			Return(&osc.PublicIp{PublicIpId: &publicIpId, PublicIp: &publicIp}, nil)
	}
}

func mockCreatePublicIp(name, clusterID, publicIpId, publicIp string) mockFunc {
	return func(s *MockCloudServices) {
		s.PublicIpMock.EXPECT().
			CreatePublicIp(gomock.Any(), gomock.Eq(name), gomock.Eq(clusterID)).
			Return(&osc.PublicIp{PublicIpId: &publicIpId, PublicIp: &publicIp}, nil)
	}
}

func mockListPublicIpsFromPool(pool string, ips []osc.PublicIp) mockFunc {
	return func(s *MockCloudServices) {
		s.PublicIpMock.EXPECT().
			ListPublicIpsFromPool(gomock.Any(), gomock.Eq(pool)).
			Return(ips, nil)
	}
}

func mockDeletePublicIp(ip string) mockFunc {
	return func(s *MockCloudServices) {
		s.PublicIpMock.EXPECT().
			DeletePublicIp(gomock.Any(), gomock.Eq(ip)).
			Return(nil)
	}
}

func mockDeleteVm(vmId string) mockFunc {
	return func(s *MockCloudServices) {
		s.VMMock.EXPECT().
			DeleteVm(gomock.Any(), gomock.Eq(vmId)).
			Return(nil)
	}
}
