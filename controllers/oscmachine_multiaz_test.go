/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package controllers_test

import (
	"math/rand/v2"
	"testing"

	infrastructurev1beta2 "github.com/outscale/cluster-api-provider-outscale/api/v1beta2"
	"github.com/outscale/cluster-api-provider-outscale/controllers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestMultiAZAllocator(t *testing.T) {
	allocated := infrastructurev1beta2.OscMachine{
		Name:      "foo1",
		Namespace: "foo",
		Labels: map[string]string{
			clusterv1.MachineDeploymentNameLabel: "foo",
		},
		Annotations: map[string]string{
			infrastructurev1beta2.NetRangeAnnotation: "10.0.0.0/8",
		},
		Spec: infrastructurev1beta2.OscMachineSpec{
			Role: infrastructurev1beta2.RoleWorker,
		},
		Status: infrastructurev1beta2.OscMachineStatus{
			FailureDomain: new("eu-west-2a"),
		},
	}
	nonallocated1 := infrastructurev1beta2.OscMachine{
		Name:      "foo2",
		Namespace: "foo",
		Labels: map[string]string{
			clusterv1.MachineDeploymentNameLabel: "foo",
		},
		Spec: infrastructurev1beta2.OscMachineSpec{
			Role: infrastructurev1beta2.RoleWorker,
		},
		Status: infrastructurev1beta2.OscMachineStatus{},
	}
	nonallocated2 := infrastructurev1beta2.OscMachine{
		Name:      "foo3",
		Namespace: "foo",
		Labels: map[string]string{
			clusterv1.MachineDeploymentNameLabel: "foo",
		},
		Spec: infrastructurev1beta2.OscMachineSpec{
			Role: infrastructurev1beta2.RoleWorker,
		},
		Status: infrastructurev1beta2.OscMachineStatus{},
	}
	testClient := func() client.Client {
		oms := &infrastructurev1beta2.OscMachineList{
			Items: []infrastructurev1beta2.OscMachine{
				allocated,
				nonallocated1,
				nonallocated2,
			},
		}
		fakeScheme := runtime.NewScheme()
		_ = clientgoscheme.AddToScheme(fakeScheme)
		_ = clusterv1.AddToScheme(fakeScheme)
		_ = apiextensionsv1.AddToScheme(fakeScheme)
		_ = infrastructurev1beta2.AddToScheme(fakeScheme)
		return fake.NewClientBuilder().WithScheme(fakeScheme).WithLists(oms).Build()
	}
	singleNet := []infrastructurev1beta2.OscNet{{
		IpRange: "10.0.0.0/8",
		Subnets: []infrastructurev1beta2.OscSubnet{
			{Subregion: "eu-west-2a", Roles: []infrastructurev1beta2.OscRole{infrastructurev1beta2.RoleWorker}},
			{Subregion: "eu-west-2b", Roles: []infrastructurev1beta2.OscRole{infrastructurev1beta2.RoleWorker}},
			{Subregion: "eu-west-2c", Roles: []infrastructurev1beta2.OscRole{infrastructurev1beta2.RoleWorker}},
		},
	}}
	multiNet := []infrastructurev1beta2.OscNet{{
		IpRange: "10.0.0.0/8",
		Subnets: []infrastructurev1beta2.OscSubnet{
			{Subregion: "eu-west-2a", Roles: []infrastructurev1beta2.OscRole{infrastructurev1beta2.RoleWorker}},
			{Subregion: "eu-west-2b", Roles: []infrastructurev1beta2.OscRole{infrastructurev1beta2.RoleWorker}},
		},
	}, {
		IpRange: "10.1.0.0/8",
		Subnets: []infrastructurev1beta2.OscSubnet{
			{Subregion: "eu-west-2b", Roles: []infrastructurev1beta2.OscRole{infrastructurev1beta2.RoleWorker}},
			{Subregion: "eu-west-2c", Roles: []infrastructurev1beta2.OscRole{infrastructurev1beta2.RoleWorker}},
		},
	}}
	t.Run("If already configured, the allocated az is returned (LeastNodes)", func(t *testing.T) {
		m := allocated
		a := controllers.NewMultiAZAllocator(testClient())
		_, sn, az, err := a.AllocateAZ(t.Context(), &m, infrastructurev1beta2.AllocationModeLeastNodes, infrastructurev1beta2.RoleWorker, singleNet, []infrastructurev1beta2.OscSubRegion{"eu-west-2a", "eu-west-2b", "eu-west-2c"})
		require.NoError(t, err)
		assert.Equal(t, infrastructurev1beta2.OscSubRegion(*m.Status.FailureDomain), az)
		assert.Equal(t, az, sn.Subregion)
	})
	t.Run("A non allocated machines is allocated to new azs (LeastNodes)", func(t *testing.T) {
		m := nonallocated1
		a := controllers.NewMultiAZAllocator(testClient())
		_, sn, az, err := a.AllocateAZ(t.Context(), &m, infrastructurev1beta2.AllocationModeLeastNodes, infrastructurev1beta2.RoleWorker, singleNet, []infrastructurev1beta2.OscSubRegion{"eu-west-2a", "eu-west-2b"})
		require.NoError(t, err)
		assert.Contains(t, []infrastructurev1beta2.OscSubRegion{"eu-west-2b", "eu-west-2c"}, az)
		assert.Equal(t, az, sn.Subregion)
	})
	t.Run("Non allocated machiness are allocated to all new azs in order (LeastNodes)", func(t *testing.T) {
		a := controllers.NewMultiAZAllocator(testClient())
		m1 := nonallocated1
		_, sn1, az1, err := a.AllocateAZ(t.Context(), &m1, infrastructurev1beta2.AllocationModeLeastNodes, infrastructurev1beta2.RoleWorker, singleNet, []infrastructurev1beta2.OscSubRegion{"eu-west-2a", "eu-west-2b", "eu-west-2c"})
		require.NoError(t, err)
		assert.Equal(t, az1, sn1.Subregion)
		m2 := nonallocated2
		_, sn2, az2, err := a.AllocateAZ(t.Context(), &m2, infrastructurev1beta2.AllocationModeLeastNodes, infrastructurev1beta2.RoleWorker, singleNet, []infrastructurev1beta2.OscSubRegion{"eu-west-2a", "eu-west-2b", "eu-west-2c"})
		require.NoError(t, err)
		assert.Equal(t, az2, sn2.Subregion)
		assert.NotEqual(t, az1, az2)
		assert.Equal(t, infrastructurev1beta2.OscSubRegion("eu-west-2b"), az1)
		assert.Equal(t, infrastructurev1beta2.OscSubRegion("eu-west-2c"), az2)
	})
	t.Run("Non allocated machines are randomly allocated (Random)", func(t *testing.T) {
		rnd := 0
		controllers.RandIntN = func(n int) (ret int) {
			ret = rnd
			rnd++
			return
		}
		defer func() {
			controllers.RandIntN = rand.IntN
		}()
		a := controllers.NewMultiAZAllocator(testClient())
		m1 := nonallocated1
		_, sn1, az1, err := a.AllocateAZ(t.Context(), &m1, infrastructurev1beta2.AllocationModeRandom, infrastructurev1beta2.RoleWorker, singleNet, []infrastructurev1beta2.OscSubRegion{"eu-west-2a", "eu-west-2b", "eu-west-2c"})
		require.NoError(t, err)
		assert.Equal(t, az1, sn1.Subregion)
		m2 := nonallocated2
		_, sn2, az2, err := a.AllocateAZ(t.Context(), &m2, infrastructurev1beta2.AllocationModeRandom, infrastructurev1beta2.RoleWorker, singleNet, []infrastructurev1beta2.OscSubRegion{"eu-west-2a", "eu-west-2b", "eu-west-2c"})
		require.NoError(t, err)
		assert.Equal(t, az2, sn2.Subregion)
		assert.Equal(t, infrastructurev1beta2.OscSubRegion("eu-west-2a"), az1)
		assert.Equal(t, infrastructurev1beta2.OscSubRegion("eu-west-2b"), az2)
	})
	t.Run("By default, leastNodes is used", func(t *testing.T) {
		// make the random allocator fail...
		controllers.RandIntN = func(n int) (ret int) {
			t.FailNow()
			return 0
		}
		defer func() {
			controllers.RandIntN = rand.IntN
		}()
		a := controllers.NewMultiAZAllocator(testClient())
		m1 := nonallocated1
		_, _, _, err := a.AllocateAZ(t.Context(), &m1, "", infrastructurev1beta2.RoleWorker, singleNet, []infrastructurev1beta2.OscSubRegion{"eu-west-2a", "eu-west-2b", "eu-west-2c"})
		require.NoError(t, err)
	})
	t.Run("Multiple nets can be allocated", func(t *testing.T) {
		m1 := nonallocated1
		m2 := nonallocated2
		a := controllers.NewMultiAZAllocator(testClient())
		n1, sn1, az1, err := a.AllocateAZ(t.Context(), &m1, infrastructurev1beta2.AllocationModeLeastNodes, infrastructurev1beta2.RoleWorker, multiNet, []infrastructurev1beta2.OscSubRegion{"eu-west-2a", "eu-west-2b", "eu-west-2c"})
		require.NoError(t, err)
		assert.Equal(t, infrastructurev1beta2.OscSubRegion("eu-west-2b"), az1)
		assert.Equal(t, az1, sn1.Subregion)
		assert.Equal(t, multiNet[1], n1) // can be in two nets, but the second one has less nodes
		n2, sn2, az2, err := a.AllocateAZ(t.Context(), &m2, infrastructurev1beta2.AllocationModeLeastNodes, infrastructurev1beta2.RoleWorker, multiNet, []infrastructurev1beta2.OscSubRegion{"eu-west-2a", "eu-west-2b", "eu-west-2c"})
		require.NoError(t, err)
		assert.Equal(t, az2, sn2.Subregion)
		assert.Equal(t, infrastructurev1beta2.OscSubRegion("eu-west-2c"), az2)
		assert.Equal(t, multiNet[1], n2) // subregion is only present in the second net
	})
}
