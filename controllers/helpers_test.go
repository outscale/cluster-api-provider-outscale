package controllers_test

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/compute"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/compute/mock_compute"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/loadbalancer"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/loadbalancer/mock_loadbalancer"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/net"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/net/mock_net"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/security"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/security/mock_security"
	tag "github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	"github.com/outscale/cluster-api-provider-outscale/cloud/tag/mock_tag"
	"github.com/outscale/cluster-api-provider-outscale/cloud/tenant"
	osc "github.com/outscale/osc-sdk-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

type MockTenant struct {
	region string
	tenant.Tenant
}

func (t MockTenant) Region() string {
	return t.region
}

type MockCloudServices struct {
	defaultTenant MockTenant
	tenant        tenant.Tenant

	NetMock           *mock_net.MockOscNetInterface
	SubnetMock        *mock_net.MockOscSubnetInterface
	SecurityGroupMock *mock_security.MockOscSecurityGroupInterface

	InternetServiceMock *mock_net.MockOscInternetServiceInterface
	RouteTableMock      *mock_security.MockOscRouteTableInterface
	NatServiceMock      *mock_net.MockOscNatServiceInterface
	PublicIpMock        *mock_security.MockOscPublicIpInterface
	LoadBalancerMock    *mock_loadbalancer.MockOscLoadBalancerInterface

	VMMock    *mock_compute.MockOscVmInterface
	ImageMock *mock_compute.MockOscImageInterface

	TagMock *mock_tag.MockOscTagInterface
}

func newMockCloudServices(mockCtrl *gomock.Controller, region string) *MockCloudServices {
	return &MockCloudServices{
		defaultTenant: MockTenant{region: region},

		NetMock:           mock_net.NewMockOscNetInterface(mockCtrl),
		SubnetMock:        mock_net.NewMockOscSubnetInterface(mockCtrl),
		SecurityGroupMock: mock_security.NewMockOscSecurityGroupInterface(mockCtrl),

		InternetServiceMock: mock_net.NewMockOscInternetServiceInterface(mockCtrl),
		RouteTableMock:      mock_security.NewMockOscRouteTableInterface(mockCtrl),
		NatServiceMock:      mock_net.NewMockOscNatServiceInterface(mockCtrl),
		PublicIpMock:        mock_security.NewMockOscPublicIpInterface(mockCtrl),
		LoadBalancerMock:    mock_loadbalancer.NewMockOscLoadBalancerInterface(mockCtrl),

		VMMock:    mock_compute.NewMockOscVmInterface(mockCtrl),
		ImageMock: mock_compute.NewMockOscImageInterface(mockCtrl),

		TagMock: mock_tag.NewMockOscTagInterface(mockCtrl),
	}
}

func (s *MockCloudServices) DefaultTenant() (tenant.Tenant, error) {
	return s.defaultTenant, nil
}

func (s *MockCloudServices) Net(t tenant.Tenant) net.OscNetInterface {
	s.tenant = t
	return s.NetMock
}

func (s *MockCloudServices) Subnet(t tenant.Tenant) net.OscSubnetInterface {
	s.tenant = t
	return s.SubnetMock
}

func (s *MockCloudServices) SecurityGroup(t tenant.Tenant) security.OscSecurityGroupInterface {
	s.tenant = t
	return s.SecurityGroupMock
}

func (s *MockCloudServices) InternetService(t tenant.Tenant) net.OscInternetServiceInterface {
	s.tenant = t
	return s.InternetServiceMock
}

func (s *MockCloudServices) RouteTable(t tenant.Tenant) security.OscRouteTableInterface {
	s.tenant = t
	return s.RouteTableMock
}

func (s *MockCloudServices) NatService(t tenant.Tenant) net.OscNatServiceInterface {
	s.tenant = t
	return s.NatServiceMock
}

func (s *MockCloudServices) PublicIp(t tenant.Tenant) security.OscPublicIpInterface {
	s.tenant = t
	return s.PublicIpMock
}

func (s *MockCloudServices) LoadBalancer(t tenant.Tenant) loadbalancer.OscLoadBalancerInterface {
	s.tenant = t
	return s.LoadBalancerMock
}

func (s *MockCloudServices) VM(t tenant.Tenant) compute.OscVmInterface {
	s.tenant = t
	return s.VMMock
}

func (s *MockCloudServices) Image(t tenant.Tenant) compute.OscImageInterface {
	s.tenant = t
	return s.ImageMock
}

func (s *MockCloudServices) Tag(t tenant.Tenant) tag.OscTagInterface {
	s.tenant = t
	return s.TagMock
}

type patchOSCClusterFunc func(m *v1beta1.OscCluster)
type patchOSCMachineFunc func(m *v1beta1.OscMachine)

type mockFunc func(s *MockCloudServices)

type assertOSCMachineFunc func(t *testing.T, m *v1beta1.OscMachine)
type assertOSCClusterFunc func(t *testing.T, c *v1beta1.OscCluster)
type assertTenantFunc func(t *testing.T, tnt tenant.Tenant)

type testcase struct {
	name                             string
	region                           string
	clusterSpec, machineSpec         string
	clusterBaseSpec, machineBaseSpec string
	clusterPatches                   []patchOSCClusterFunc
	machinePatches                   []patchOSCMachineFunc
	mockFuncs                        []mockFunc
	kubeObjects                      []client.Object
	hasError                         bool
	requeue                          bool
	assertDeleted                    bool
	clusterAsserts                   []assertOSCClusterFunc
	machineAsserts                   []assertOSCMachineFunc
	tenantAsserts                    []assertTenantFunc

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

func assertTenant(ak, sk, region string) assertTenantFunc {
	return func(t *testing.T, found tenant.Tenant) {
		assert.Equal(t, region, found.Region())
		awsv4 := found.ContextWithAuth(context.TODO()).Value(osc.ContextAWSv4).(osc.AWSv4)
		assert.Equal(t, ak, awsv4.AccessKey)
		assert.Equal(t, sk, awsv4.SecretKey)
	}
}

func assertDefaultTenant() assertTenantFunc {
	return func(t *testing.T, found tenant.Tenant) {
		assert.IsType(t, MockTenant{}, found)
	}
}
