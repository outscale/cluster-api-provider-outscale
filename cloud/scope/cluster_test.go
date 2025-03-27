package scope_test

import (
	"testing"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

func TestClusterScope_GetSubnets(t *testing.T) {
	t.Run("Default subnets are computed on a default net if none set", func(t *testing.T) {
		clusterScope := scope.ClusterScope{OscCluster: &infrastructurev1beta1.OscCluster{}}
		clusterScope.OscCluster.Spec.Network.SubregionName = "eu-west2a"
		subnets := clusterScope.GetSubnets()
		assert.Equal(t, []infrastructurev1beta1.OscSubnet{
			{IpSubnetRange: "10.0.2.0/24", Roles: []infrastructurev1beta1.OscRole{infrastructurev1beta1.RoleLoadBalancer, infrastructurev1beta1.RoleBastion}, SubregionName: "eu-west2a"},
			{IpSubnetRange: "10.0.3.0/24", Roles: []infrastructurev1beta1.OscRole{infrastructurev1beta1.RoleWorker}, SubregionName: "eu-west2a"},
			{IpSubnetRange: "10.0.4.0/24", Roles: []infrastructurev1beta1.OscRole{infrastructurev1beta1.RoleControlPlane}, SubregionName: "eu-west2a"},
		}, subnets)
	})
	t.Run("Default subnets are computed on a custom net if not set", func(t *testing.T) {
		clusterScope := scope.ClusterScope{OscCluster: &infrastructurev1beta1.OscCluster{}}
		clusterScope.OscCluster.Spec.Network.Net.IpRange = "10.1.0.0/16"
		clusterScope.OscCluster.Spec.Network.SubregionName = "eu-west2a"
		subnets := clusterScope.GetSubnets()
		assert.Equal(t, []infrastructurev1beta1.OscSubnet{
			{IpSubnetRange: "10.1.2.0/24", Roles: []infrastructurev1beta1.OscRole{infrastructurev1beta1.RoleLoadBalancer, infrastructurev1beta1.RoleBastion}, SubregionName: "eu-west2a"},
			{IpSubnetRange: "10.1.3.0/24", Roles: []infrastructurev1beta1.OscRole{infrastructurev1beta1.RoleWorker}, SubregionName: "eu-west2a"},
			{IpSubnetRange: "10.1.4.0/24", Roles: []infrastructurev1beta1.OscRole{infrastructurev1beta1.RoleControlPlane}, SubregionName: "eu-west2a"},
		}, subnets)
	})
}

func TestClusterScope_GetSubnet(t *testing.T) {
	tts := []struct {
		subnets         []infrastructurev1beta1.OscSubnet
		searchName      string
		searchRole      infrastructurev1beta1.OscRole
		searchSubregion string
		expectName      string
	}{
		{
			subnets: []infrastructurev1beta1.OscSubnet{
				{Name: "1", Roles: []infrastructurev1beta1.OscRole{infrastructurev1beta1.RoleLoadBalancer, infrastructurev1beta1.RoleBastion}},
				{Name: "2", Roles: []infrastructurev1beta1.OscRole{infrastructurev1beta1.RoleControlPlane}},
				{Name: "3", Roles: []infrastructurev1beta1.OscRole{infrastructurev1beta1.RoleWorker}},
			},
			searchRole: infrastructurev1beta1.RoleLoadBalancer,
			expectName: "1",
		},
		{
			subnets: []infrastructurev1beta1.OscSubnet{
				{Name: "1", Roles: []infrastructurev1beta1.OscRole{infrastructurev1beta1.RoleLoadBalancer, infrastructurev1beta1.RoleBastion}},
				{Name: "2", Roles: []infrastructurev1beta1.OscRole{infrastructurev1beta1.RoleControlPlane}},
				{Name: "3", Roles: []infrastructurev1beta1.OscRole{infrastructurev1beta1.RoleWorker}},
			},
			searchRole: infrastructurev1beta1.RoleBastion,
			expectName: "1",
		},
		{
			subnets: []infrastructurev1beta1.OscSubnet{
				{Name: "1", Roles: []infrastructurev1beta1.OscRole{infrastructurev1beta1.RoleLoadBalancer, infrastructurev1beta1.RoleBastion}},
				{Name: "2", Roles: []infrastructurev1beta1.OscRole{infrastructurev1beta1.RoleControlPlane}},
				{Name: "3", Roles: []infrastructurev1beta1.OscRole{infrastructurev1beta1.RoleWorker}},
			},
			searchRole: infrastructurev1beta1.RoleControlPlane,
			expectName: "2",
		},
		{
			subnets: []infrastructurev1beta1.OscSubnet{
				{Name: "1", Roles: []infrastructurev1beta1.OscRole{infrastructurev1beta1.RoleLoadBalancer, infrastructurev1beta1.RoleBastion}},
				{Name: "2", Roles: []infrastructurev1beta1.OscRole{infrastructurev1beta1.RoleControlPlane}},
				{Name: "3", Roles: []infrastructurev1beta1.OscRole{infrastructurev1beta1.RoleWorker}},
			},
			searchRole: infrastructurev1beta1.RoleWorker,
			expectName: "3",
		},
		{
			subnets: []infrastructurev1beta1.OscSubnet{
				{Name: "1", Roles: []infrastructurev1beta1.OscRole{infrastructurev1beta1.RoleLoadBalancer, infrastructurev1beta1.RoleBastion}},
				{Name: "2", Roles: []infrastructurev1beta1.OscRole{infrastructurev1beta1.RoleControlPlane}},
				{Name: "3", Roles: []infrastructurev1beta1.OscRole{infrastructurev1beta1.RoleWorker}},
			},
			searchRole:      infrastructurev1beta1.RoleWorker,
			searchSubregion: "eu-west2a",
			expectName:      "3",
		},
		{
			subnets: []infrastructurev1beta1.OscSubnet{
				{Name: "1", Roles: []infrastructurev1beta1.OscRole{infrastructurev1beta1.RoleLoadBalancer, infrastructurev1beta1.RoleBastion}},
				{Name: "2", Roles: []infrastructurev1beta1.OscRole{infrastructurev1beta1.RoleControlPlane}},
				{Name: "3", Roles: []infrastructurev1beta1.OscRole{infrastructurev1beta1.RoleWorker}, SubregionName: "eu-west2a"},
			},
			searchRole: infrastructurev1beta1.RoleWorker,
			expectName: "3",
		},
		{
			subnets: []infrastructurev1beta1.OscSubnet{
				{Name: "1", Roles: []infrastructurev1beta1.OscRole{infrastructurev1beta1.RoleLoadBalancer, infrastructurev1beta1.RoleBastion}},
				{Name: "2", Roles: []infrastructurev1beta1.OscRole{infrastructurev1beta1.RoleControlPlane}},
				{Name: "3", Roles: []infrastructurev1beta1.OscRole{infrastructurev1beta1.RoleWorker}},
			},
			searchRole:      infrastructurev1beta1.RoleWorker,
			searchSubregion: "eu-west2b",
			expectName:      "3",
		},
	}

	for _, tt := range tts {
		clusterScope := &scope.ClusterScope{
			OscCluster: &infrastructurev1beta1.OscCluster{
				Spec: infrastructurev1beta1.OscClusterSpec{
					Network: infrastructurev1beta1.OscNetwork{
						SubregionName: "eu-west2a",
						Subnets:       tt.subnets,
					},
				},
			},
		}

		found, err := clusterScope.GetSubnet(tt.searchName, tt.searchRole, tt.searchSubregion)
		if found.Name == "" {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			assert.Equal(t, tt.expectName, found.Name)
		}
	}
}

func TestClusterScope_GetSecurityGroupsFor(t *testing.T) {
	tts := []struct {
		searchRole  infrastructurev1beta1.OscRole
		expectNames []string
	}{
		{
			searchRole:  infrastructurev1beta1.RoleLoadBalancer,
			expectNames: []string{"foo-lb"},
		},
		{
			searchRole:  infrastructurev1beta1.RoleControlPlane,
			expectNames: []string{"foo-controlplane", "foo-node"},
		},
		{
			searchRole:  infrastructurev1beta1.RoleWorker,
			expectNames: []string{"foo-worker", "foo-node"},
		},
	}

	for _, tt := range tts {
		clusterScope := &scope.ClusterScope{
			Cluster: &clusterv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
					UID:  "abcd",
				},
			},
			OscCluster: &infrastructurev1beta1.OscCluster{},
		}

		sgs, err := clusterScope.GetSecurityGroupsFor(nil, tt.searchRole)
		require.NoError(t, err)
		require.Len(t, sgs, len(tt.expectNames))
		for i, expectName := range tt.expectNames {
			assert.Equal(t, expectName, sgs[i].Name)
		}
	}
}
