/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package controllers_test

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	infrastructurev1beta2 "github.com/outscale/cluster-api-provider-outscale/api/v1beta2"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/compute"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/compute/mock_compute"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/net"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/net/mock_net"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/tag"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/tag/mock_tag"
	"github.com/outscale/cluster-api-provider-outscale/cloud/tenant"
	"github.com/outscale/osc-sdk-go/v3/pkg/osc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
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

	NetMock     *mock_net.MockServicer
	ComputeMock *mock_compute.MockServicer
	TagMock     *mock_tag.MockServicer
}

func newMockCloudServices(mockCtrl *gomock.Controller, region string) *MockCloudServices {
	return &MockCloudServices{
		defaultTenant: MockTenant{region: region},

		NetMock:     mock_net.NewMockServicer(mockCtrl),
		ComputeMock: mock_compute.NewMockServicer(mockCtrl),
		TagMock:     mock_tag.NewMockServicer(mockCtrl),
	}
}

func (s *MockCloudServices) DefaultTenant() (tenant.Tenant, error) {
	return s.defaultTenant, nil
}

func (s *MockCloudServices) Net(t tenant.Tenant) net.Servicer {
	s.tenant = t
	return s.NetMock
}

func (s *MockCloudServices) Compute(t tenant.Tenant) compute.Servicer {
	s.tenant = t
	return s.ComputeMock
}

func (s *MockCloudServices) Tag(t tenant.Tenant) tag.Servicer {
	s.tenant = t
	return s.TagMock
}

type (
	patchOSCClusterFunc func(m *infrastructurev1beta2.OscCluster)
	patchOSCMachineFunc func(m *infrastructurev1beta2.OscMachine)
)

type mockFunc func(s *MockCloudServices)

type (
	assertOSCMachineFunc func(t *testing.T, m *infrastructurev1beta2.OscMachine)
	assertOSCClusterFunc func(t *testing.T, c *infrastructurev1beta2.OscCluster)
	assertTenantFunc     func(t *testing.T, tnt tenant.Tenant)
)

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

func loadClusterSpecs(t *testing.T, spec, base string) (*clusterv1.Cluster, *infrastructurev1beta2.OscCluster) {
	var cluster clusterv1.Cluster
	var cname string
	switch base {
	case "-":
		cname = "doesnotexist"
	case "":
		decode(t, "cluster/"+trimVersion(spec)+".yaml", &cluster)
		cname = cluster.Name
	default:
		decode(t, "cluster/"+base+".yaml", &cluster)
		cname = cluster.Name
	}
	var osccluster infrastructurev1beta2.OscCluster
	decode(t, "osccluster/"+spec+".yaml", &osccluster)
	osccluster.Labels = map[string]string{clusterv1.ClusterNameLabel: osccluster.Name}
	osccluster.OwnerReferences = []metav1.OwnerReference{{
		APIVersion: "cluster.x-k8s.io/v1beta1",
		Kind:       "Cluster",
		Name:       cname,
	}}
	return &cluster, &osccluster
}

func loadMachineSpecs(t *testing.T, spec, base, clusterName string) (*clusterv1.Machine, *infrastructurev1beta2.OscMachine) {
	var machine clusterv1.Machine
	var mname string
	switch base {
	case "-":
		mname = "doesnotexist"
	case "":
		decode(t, "machine/"+trimVersion(spec)+".yaml", &machine)
		mname = machine.Name
	default:
		decode(t, "machine/"+base+".yaml", &machine)
		mname = machine.Name
	}
	var oscmachine infrastructurev1beta2.OscMachine
	decode(t, "oscmachine/"+spec+".yaml", &oscmachine)
	oscmachine.Labels = map[string]string{clusterv1.ClusterNameLabel: clusterName}
	oscmachine.OwnerReferences = []metav1.OwnerReference{{
		APIVersion: "cluster.x-k8s.io/v1beta1",
		Kind:       "Machine",
		Name:       mname,
	}}
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
				Value:      name,
				ResourceId: resourceId,
			}, nil)
	}
}

func mockGetSecurityGroupsFromNet(netId string, sgs []osc.SecurityGroup) mockFunc {
	return func(s *MockCloudServices) {
		s.ComputeMock.EXPECT().
			GetSecurityGroupsFromNet(gomock.Any(), gomock.Eq(netId)).
			Return(sgs, nil)
	}
}

func mockPublicIpFound(publicIpId string, pip ...*osc.PublicIp) mockFunc {
	if len(pip) > 0 {
		return func(s *MockCloudServices) {
			s.NetMock.EXPECT().
				GetPublicIp(gomock.Any(), gomock.Eq(publicIpId)).
				Return(pip[0], nil)
		}
	}
	return func(s *MockCloudServices) {
		s.NetMock.EXPECT().
			GetPublicIp(gomock.Any(), gomock.Eq(publicIpId)).
			Return(&osc.PublicIp{PublicIpId: publicIpId}, nil)
	}
}

func mockGetPublicIpByIp(publicIp, publicIpId string) mockFunc {
	return func(s *MockCloudServices) {
		s.NetMock.EXPECT().
			GetPublicIpByIp(gomock.Any(), gomock.Eq(publicIp)).
			Return(&osc.PublicIp{PublicIpId: publicIpId, PublicIp: publicIp}, nil)
	}
}

func mockCreatePublicIp(name, clusterID, publicIpId, publicIp string) mockFunc {
	return func(s *MockCloudServices) {
		s.NetMock.EXPECT().
			CreatePublicIp(gomock.Any(), gomock.Eq(name), gomock.Eq(clusterID)).
			Return(&osc.PublicIp{PublicIpId: publicIpId, PublicIp: publicIp}, nil)
	}
}

func mockListPublicIpsFromPool(pool string, ips []osc.PublicIp) mockFunc {
	return func(s *MockCloudServices) {
		s.NetMock.EXPECT().
			ListPublicIpsFromPool(gomock.Any(), gomock.Eq(pool)).
			Return(ips, nil)
	}
}

func mockDeletePublicIp(ip string) mockFunc {
	return func(s *MockCloudServices) {
		s.NetMock.EXPECT().
			DeletePublicIp(gomock.Any(), gomock.Eq(ip)).
			Return(nil)
	}
}

func mockDeleteVm(vmId string) mockFunc {
	return func(s *MockCloudServices) {
		s.ComputeMock.EXPECT().
			DeleteVm(gomock.Any(), gomock.Eq(vmId)).
			Return(nil)
	}
}

func assertTenant(ak, sk, region string) assertTenantFunc {
	return func(t *testing.T, found tenant.Tenant) {
		profile := found.Profile()
		assert.Equal(t, region, profile.Region)
		assert.Equal(t, ak, profile.AccessKey)
		assert.Equal(t, sk, profile.SecretKey)
	}
}

func assertDefaultTenant() assertTenantFunc {
	return func(t *testing.T, found tenant.Tenant) {
		assert.IsType(t, MockTenant{}, found)
	}
}
