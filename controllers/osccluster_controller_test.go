package controllers_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/controllers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func runClusterTest(t *testing.T, tc testcase) {
	c, oc := loadClusterSpecs(t, tc.clusterSpec)
	oc.Labels = map[string]string{clusterv1.ClusterNameLabel: oc.Name}
	oc.OwnerReferences = []metav1.OwnerReference{{
		APIVersion: "cluster.x-k8s.io/v1beta1",
		Kind:       "Cluster",
		Name:       c.Name,
	}}
	for _, fn := range tc.clusterPatches {
		fn(oc)
	}
	fakeScheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(fakeScheme)
	_ = clusterv1.AddToScheme(fakeScheme)
	_ = apiextensionsv1.AddToScheme(fakeScheme)
	_ = infrastructurev1beta1.AddToScheme(fakeScheme)
	client := fake.NewClientBuilder().WithScheme(fakeScheme).
		WithStatusSubresource(oc).WithObjects(c, oc).Build()
	mockCtrl := gomock.NewController(t)
	cs := newMockCloudServices(mockCtrl)
	for _, fn := range tc.mockFuncs {
		fn(cs)
	}
	rec := controllers.OscClusterReconciler{
		Client: client,
		Cloud:  cs,
	}
	nsn := types.NamespacedName{
		Namespace: oc.Namespace,
		Name:      oc.Name,
	}
	res, err := rec.Reconcile(context.TODO(), controllerruntime.Request{NamespacedName: nsn})
	if tc.hasError {
		require.Error(t, err)
		assert.Zero(t, res)
	} else {
		require.NoError(t, err)
		assert.Equal(t, tc.requeue, res.RequeueAfter > 0 || res.Requeue)
	}
	var out v1beta1.OscCluster
	err = client.Get(context.TODO(), nsn, &out)
	require.NoError(t, err, "resource was not found")
	for _, fn := range tc.clusterAsserts {
		fn(t, &out)
	}
}

func TestReconcileOSCCluster_Update(t *testing.T) {
	tcs := []testcase{
		{
			// FIXME: this is a bug, cluster should be ok.
			name:           "cluster has been moved by clusterctl move, status is updated",
			clusterSpec:    "ready",
			clusterPatches: []patchOSCClusterFunc{patchMoveCluster()},
			mockFuncs: []mockFunc{
				mockReadTagByNameFound("test-cluster-api-net-9e1db9c4-bf0a-4583-8999-203ec002c520"),
				mockNetFound("vpc-24ba90ce"), mockGetSubnetIdsFromNetIds("vpc-24ba90ce", []string{
					"subnet-c1a282b0", "subnet-1555ea91", "subnet-174f5ec4",
				}),
				mockReadTagByNameFound("test-cluster-api-subnet-kw-9e1db9c4-bf0a-4583-8999-203ec002c520"),
				mockReadTagByNameFound("test-cluster-api-subnet-kcp-9e1db9c4-bf0a-4583-8999-203ec002c520"),
				mockReadTagByNameFound("test-cluster-api-subnet-public-9e1db9c4-bf0a-4583-8999-203ec002c520"),

				mockReadTagByNameFound("test-cluster-api-internetservice-9e1db9c4-bf0a-4583-8999-203ec002c520"),
				mockInternetServiceFound("igw-c3c49899"),

				mockValidatePublicIpIdsOk([]string{"eipalloc-da72a57c"}),
				mockReadTagByNameFound("test-cluster-api-publicip-nat-9e1db9c4-bf0a-4583-8999-203ec002c520"),

				mockSecurityGroupsForNetFound("vpc-24ba90ce", []string{"sg-750ae810", "sg-a093d014", "sg-7eb16ccb", "sg-0cd1f87e"}),
				mockReadTagByNameFound("test-cluster-api-securitygroup-kw-9e1db9c4-bf0a-4583-8999-203ec002c520"),
				mockReadTagByNameFound("test-cluster-api-securitygroup-kcp-9e1db9c4-bf0a-4583-8999-203ec002c520"),
				mockReadTagByNameFound("test-cluster-api-securitygroup-lb-9e1db9c4-bf0a-4583-8999-203ec002c520"),
				mockReadTagByNameFound("test-cluster-api-securitygroup-node-9e1db9c4-bf0a-4583-8999-203ec002c520"),

				mockRouteTablesForNetFound("vpc-24ba90ce", []string{"rtb-194c971e", "rtb-0a4640a6", "rtb-eeacfe8a"}),
				mockReadTagByNameFound("test-cluster-api-routetable-kcp-9e1db9c4-bf0a-4583-8999-203ec002c520"),
				mockReadTagByNameFound("test-cluster-api-routetable-kw-9e1db9c4-bf0a-4583-8999-203ec002c520"),
				mockReadTagByNameFound("test-cluster-api-routetable-public-9e1db9c4-bf0a-4583-8999-203ec002c520"),

				mockReadTagByNameFound("test-cluster-api-natservice-9e1db9c4-bf0a-4583-8999-203ec002c520"),
				mockNatServiceFound("nat-223a4dd4"),

				mockRouteTablesForNetFound("vpc-24ba90ce", []string{"rtb-194c971e", "rtb-0a4640a6", "rtb-eeacfe8a"}),
				mockReadTagByNameFound("test-cluster-api-routetable-kcp-9e1db9c4-bf0a-4583-8999-203ec002c520"),
				mockReadTagByNameFound("test-cluster-api-routetable-kw-9e1db9c4-bf0a-4583-8999-203ec002c520"),
				mockReadTagByNameFound("test-cluster-api-routetable-public-9e1db9c4-bf0a-4583-8999-203ec002c520"),

				mockLoadBalancerFound("test-cluster-api-k8s"),
				mockLoadBalancerTagFound("test-cluster-api-k8s"),
			},
			hasError: true,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			runClusterTest(t, tc)
		})
	}
}
