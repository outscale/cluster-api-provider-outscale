package controllers_test

import (
	"context"
	"errors"
	"testing"

	"github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	"github.com/outscale/cluster-api-provider-outscale/controllers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
		Tracker: &controllers.ResourceTracker{
			Cloud: cs,
		},
		Cloud: cs,
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
	switch {
	case tc.assertDeleted:
		require.True(t, apierrors.IsNotFound(err), "resource must have been deleted")
	default:
		require.NoError(t, err, "resource was not found")
		for _, fn := range tc.clusterAsserts {
			fn(t, &out)
		}
	}
}

func TestReconcileOSCCluster_Update(t *testing.T) {
	tcs := []testcase{
		{
			// TODO: assert that status is correctly setup
			name:           "cluster has been moved by clusterctl move, status is updated",
			clusterSpec:    "ready",
			clusterPatches: []patchOSCClusterFunc{patchMoveCluster()},
			mockFuncs: []mockFunc{
				mockReadTagByNameFound(tag.NetResourceType, "test-cluster-api-net-9e1db9c4-bf0a-4583-8999-203ec002c520", "vpc-24ba90ce"),
				mockNetFound("vpc-24ba90ce"), mockGetSubnetIdsFromNetIds("vpc-24ba90ce", []string{
					"subnet-c1a282b0", "subnet-1555ea91", "subnet-174f5ec4",
				}),
				mockReadTagByNameFound(tag.SubnetResourceType, "test-cluster-api-subnet-kw-9e1db9c4-bf0a-4583-8999-203ec002c520", ""),
				mockReadTagByNameFound(tag.SubnetResourceType, "test-cluster-api-subnet-kcp-9e1db9c4-bf0a-4583-8999-203ec002c520", ""),
				mockReadTagByNameFound(tag.SubnetResourceType, "test-cluster-api-subnet-public-9e1db9c4-bf0a-4583-8999-203ec002c520", ""),

				mockReadTagByNameFound(tag.InternetServiceResourceType, "test-cluster-api-internetservice-9e1db9c4-bf0a-4583-8999-203ec002c520", "igw-c3c49899"),
				mockInternetServiceFound("igw-c3c49899"),

				mockValidatePublicIpIdsOk([]string{"eipalloc-da72a57c"}),
				mockReadTagByNameFound(tag.PublicIPResourceType, "test-cluster-api-publicip-nat-9e1db9c4-bf0a-4583-8999-203ec002c520", "eipalloc-da72a57c"),

				mockSecurityGroupsForNetFound("vpc-24ba90ce", []string{"sg-750ae810", "sg-a093d014", "sg-7eb16ccb", "sg-0cd1f87e"}),
				mockReadTagByNameFound(tag.SecurityGroupResourceType, "test-cluster-api-securitygroup-kw-9e1db9c4-bf0a-4583-8999-203ec002c520", "sg-750ae810"),
				mockReadTagByNameFound(tag.SecurityGroupResourceType, "test-cluster-api-securitygroup-kcp-9e1db9c4-bf0a-4583-8999-203ec002c520", "sg-a093d014"),
				mockReadTagByNameFound(tag.SecurityGroupResourceType, "test-cluster-api-securitygroup-lb-9e1db9c4-bf0a-4583-8999-203ec002c520", "sg-7eb16ccb"),
				mockReadTagByNameFound(tag.SecurityGroupResourceType, "test-cluster-api-securitygroup-node-9e1db9c4-bf0a-4583-8999-203ec002c520", "sg-0cd1f87e"),

				mockReadTagByNameFound(tag.NatResourceType, "test-cluster-api-natservice-9e1db9c4-bf0a-4583-8999-203ec002c520", "nat-223a4dd4"),
				mockNatServiceFound("nat-223a4dd4"),

				mockRouteTablesForNetFound("vpc-24ba90ce", []string{"rtb-194c971e", "rtb-0a4640a6", "rtb-eeacfe8a"}),
				mockReadTagByNameFound(tag.NatResourceType, "test-cluster-api-routetable-kcp-9e1db9c4-bf0a-4583-8999-203ec002c520", "rtb-194c971e"),
				mockReadTagByNameFound(tag.NatResourceType, "test-cluster-api-routetable-kw-9e1db9c4-bf0a-4583-8999-203ec002c520", "rtb-0a4640a6"),
				mockReadTagByNameFound(tag.NatResourceType, "test-cluster-api-routetable-public-9e1db9c4-bf0a-4583-8999-203ec002c520", "rtb-eeacfe8a"),

				mockLoadBalancerFound("test-cluster-api-k8s", true),
				mockLoadBalancerTagFound("test-cluster-api-k8s", "test-cluster-api-k8s-9e1db9c4-bf0a-4583-8999-203ec002c520"),
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			runClusterTest(t, tc)
		})
	}
}

func TestReconcileOSCCluster_Delete(t *testing.T) {
	tcs := []testcase{
		{
			name:           "Cluster is deleted with all sg rules",
			clusterSpec:    "ready",
			clusterPatches: []patchOSCClusterFunc{patchDeleteCluster()},
			mockFuncs: []mockFunc{
				mockLoadBalancerFound("test-cluster-api-k8s", true),
				mockLoadBalancerTagFound("test-cluster-api-k8s", "test-cluster-api-k8s-9e1db9c4-bf0a-4583-8999-203ec002c520"),
				mockCheckLoadBalancerDeregisterVm("test-cluster-api-k8s"),
				mockDeleteLoadBalancerTag("test-cluster-api-k8s"),
				mockDeleteLoadBalancer("test-cluster-api-k8s"),

				mockNatServiceFound("nat-223a4dd4"),
				mockDeleteNatService("nat-223a4dd4"),

				mockValidatePublicIpIdsOk([]string{"eipalloc-da72a57c"}),
				mockCheckPublicIpUnlink("eipalloc-da72a57c"),
				mockDeletePublicIp("eipalloc-da72a57c"),

				mockRouteTablesForNetFound("vpc-24ba90ce", []string{"rtb-194c971e", "rtb-0a4640a6", "rtb-eeacfe8a"}),
				mockDeleteRoute("rtb-0a4640a6", "0.0.0.0/0"),
				mockUnlinkRouteTable("rtbassoc-643430b3"),
				mockDeleteRouteTable("rtb-0a4640a6"),
				mockDeleteRoute("rtb-194c971e", "0.0.0.0/0"),
				mockUnlinkRouteTable("rtbassoc-09475c37"),
				mockDeleteRouteTable("rtb-194c971e"),
				mockDeleteRoute("rtb-eeacfe8a", "0.0.0.0/0"),
				mockUnlinkRouteTable("rtbassoc-90bda9c8"),
				mockDeleteRouteTable("rtb-eeacfe8a"),

				mockSecurityGroupsForNetFound("vpc-24ba90ce", []string{"sg-750ae810", "sg-a093d014", "sg-7eb16ccb", "sg-0cd1f87e"}),

				mockSecurityGroupHasRule("sg-a093d014", "Inbound", "tcp", "10.0.0.0/16", "", 179, 179, true),
				mockDeleteSecurityGroupRule("sg-a093d014", "Inbound", "tcp", "10.0.0.0/16", "", 179, 179),
				mockSecurityGroupHasRule("sg-a093d014", "Inbound", "tcp", "10.0.3.0/24", "", 10250, 10250, true),
				mockDeleteSecurityGroupRule("sg-a093d014", "Inbound", "tcp", "10.0.3.0/24", "", 10250, 10250),
				mockSecurityGroupHasRule("sg-a093d014", "Inbound", "tcp", "10.0.3.0/24", "", 30000, 32767, true),
				mockDeleteSecurityGroupRule("sg-a093d014", "Inbound", "tcp", "10.0.3.0/24", "", 30000, 32767),
				mockSecurityGroupHasRule("sg-a093d014", "Inbound", "tcp", "10.0.4.0/24", "", 10250, 10250, true),
				mockDeleteSecurityGroupRule("sg-a093d014", "Inbound", "tcp", "10.0.4.0/24", "", 10250, 10250),
				mockSecurityGroupHasRule("sg-a093d014", "Inbound", "tcp", "10.0.4.0/24", "", 30000, 32767, true),
				mockDeleteSecurityGroupRule("sg-a093d014", "Inbound", "tcp", "10.0.4.0/24", "", 30000, 32767),
				mockDeleteSecurityGroup("sg-a093d014", nil),

				mockSecurityGroupHasRule("sg-750ae810", "Inbound", "tcp", "10.0.0.0/16", "", 179, 179, true),
				mockDeleteSecurityGroupRule("sg-750ae810", "Inbound", "tcp", "10.0.0.0/16", "", 179, 179),
				mockSecurityGroupHasRule("sg-750ae810", "Inbound", "tcp", "10.0.3.0/24", "", 6443, 6443, true),
				mockDeleteSecurityGroupRule("sg-750ae810", "Inbound", "tcp", "10.0.3.0/24", "", 6443, 6443),
				mockSecurityGroupHasRule("sg-750ae810", "Inbound", "tcp", "10.0.3.0/24", "", 30000, 32767, true),
				mockDeleteSecurityGroupRule("sg-750ae810", "Inbound", "tcp", "10.0.3.0/24", "", 30000, 32767),
				mockSecurityGroupHasRule("sg-750ae810", "Inbound", "tcp", "10.0.4.0/24", "", 6443, 6443, true),
				mockDeleteSecurityGroupRule("sg-750ae810", "Inbound", "tcp", "10.0.4.0/24", "", 6443, 6443),
				mockSecurityGroupHasRule("sg-750ae810", "Inbound", "tcp", "10.0.4.0/24", "", 30000, 32767, true),
				mockDeleteSecurityGroupRule("sg-750ae810", "Inbound", "tcp", "10.0.4.0/24", "", 30000, 32767),
				mockSecurityGroupHasRule("sg-750ae810", "Inbound", "tcp", "10.0.4.0/24", "", 2378, 2379, true),
				mockDeleteSecurityGroupRule("sg-750ae810", "Inbound", "tcp", "10.0.4.0/24", "", 2378, 2379),
				mockSecurityGroupHasRule("sg-750ae810", "Inbound", "tcp", "10.0.4.0/24", "", 10250, 10252, true),
				mockDeleteSecurityGroupRule("sg-750ae810", "Inbound", "tcp", "10.0.4.0/24", "", 10250, 10252),
				mockDeleteSecurityGroup("sg-750ae810", nil),

				mockSecurityGroupHasRule("sg-7eb16ccb", "Inbound", "tcp", "0.0.0.0/0", "", 6443, 6443, true),
				mockDeleteSecurityGroupRule("sg-7eb16ccb", "Inbound", "tcp", "0.0.0.0/0", "", 6443, 6443),
				mockDeleteSecurityGroup("sg-7eb16ccb", nil),

				mockSecurityGroupHasRule("sg-0cd1f87e", "Inbound", "udp", "10.0.0.0/16", "", 4789, 4789, true),
				mockDeleteSecurityGroupRule("sg-0cd1f87e", "Inbound", "udp", "10.0.0.0/16", "", 4789, 4789),
				mockSecurityGroupHasRule("sg-0cd1f87e", "Inbound", "udp", "10.0.0.0/16", "", 5473, 5473, true),
				mockDeleteSecurityGroupRule("sg-0cd1f87e", "Inbound", "udp", "10.0.0.0/16", "", 5473, 5473),
				mockSecurityGroupHasRule("sg-0cd1f87e", "Inbound", "udp", "10.0.0.0/16", "", 51820, 51820, true),
				mockDeleteSecurityGroupRule("sg-0cd1f87e", "Inbound", "udp", "10.0.0.0/16", "", 51820, 51820),
				mockSecurityGroupHasRule("sg-0cd1f87e", "Inbound", "udp", "10.0.0.0/16", "", 51821, 51821, true),
				mockDeleteSecurityGroupRule("sg-0cd1f87e", "Inbound", "udp", "10.0.0.0/16", "", 51821, 51821),
				mockSecurityGroupHasRule("sg-0cd1f87e", "Inbound", "udp", "10.0.0.0/16", "", 8285, 8285, true),
				mockDeleteSecurityGroupRule("sg-0cd1f87e", "Inbound", "udp", "10.0.0.0/16", "", 8285, 8285),
				mockSecurityGroupHasRule("sg-0cd1f87e", "Inbound", "udp", "10.0.0.0/16", "", 8472, 8472, true),
				mockDeleteSecurityGroupRule("sg-0cd1f87e", "Inbound", "udp", "10.0.0.0/16", "", 8472, 8472),
				mockDeleteSecurityGroup("sg-0cd1f87e", nil),

				mockInternetServiceFound("igw-c3c49899"),
				mockUnlinkInternetService("igw-c3c49899", "vpc-24ba90ce"),
				mockDeleteInternetService("igw-c3c49899"),

				mockGetSubnetIdsFromNetIds("vpc-24ba90ce", []string{
					"subnet-c1a282b0", "subnet-1555ea91", "subnet-174f5ec4",
				}),
				mockDeleteSubnet("subnet-c1a282b0"),
				mockDeleteSubnet("subnet-1555ea91"),
				mockDeleteSubnet("subnet-174f5ec4"),
				mockNetFound("vpc-24ba90ce"),
				mockDeleteNet("vpc-24ba90ce"),
			},
			assertDeleted: true,
		},
		{
			name:           "Cluster is deleted event if no resource has been created",
			clusterSpec:    "base",
			clusterPatches: []patchOSCClusterFunc{patchDeleteCluster()},
			mockFuncs: []mockFunc{
				mockLoadBalancerFound("test-cluster-api-k8s", false),
			},
			assertDeleted: true,
		},
		{
			name:           "If LB is already deleted, continue with the rest",
			clusterSpec:    "ready",
			clusterPatches: []patchOSCClusterFunc{patchDeleteCluster()},
			mockFuncs: []mockFunc{
				mockLoadBalancerFound("test-cluster-api-k8s", false),

				mockNatServiceFound("nat-223a4dd4"),
				mockDeleteNatService("nat-223a4dd4"),

				mockValidatePublicIpIdsOk([]string{"eipalloc-da72a57c"}),
				mockCheckPublicIpUnlink("eipalloc-da72a57c"),
				mockDeletePublicIp("eipalloc-da72a57c"),

				mockRouteTablesForNetFound("vpc-24ba90ce", []string{"rtb-194c971e", "rtb-0a4640a6", "rtb-eeacfe8a"}),
				mockDeleteRoute("rtb-0a4640a6", "0.0.0.0/0"),
				mockUnlinkRouteTable("rtbassoc-643430b3"),
				mockDeleteRouteTable("rtb-0a4640a6"),
				mockDeleteRoute("rtb-194c971e", "0.0.0.0/0"),
				mockUnlinkRouteTable("rtbassoc-09475c37"),
				mockDeleteRouteTable("rtb-194c971e"),
				mockDeleteRoute("rtb-eeacfe8a", "0.0.0.0/0"),
				mockUnlinkRouteTable("rtbassoc-90bda9c8"),
				mockDeleteRouteTable("rtb-eeacfe8a"),

				mockSecurityGroupsForNetFound("vpc-24ba90ce", []string{"sg-750ae810", "sg-a093d014", "sg-7eb16ccb", "sg-0cd1f87e"}),

				mockSecurityGroupCleanAllRules("sg-a093d014"),
				mockDeleteSecurityGroup("sg-a093d014", nil),

				mockSecurityGroupCleanAllRules("sg-750ae810"),
				mockDeleteSecurityGroup("sg-750ae810", nil),

				mockSecurityGroupCleanAllRules("sg-7eb16ccb"),
				mockDeleteSecurityGroup("sg-7eb16ccb", nil),

				mockSecurityGroupCleanAllRules("sg-0cd1f87e"),
				mockDeleteSecurityGroup("sg-0cd1f87e", nil),

				mockInternetServiceFound("igw-c3c49899"),
				mockUnlinkInternetService("igw-c3c49899", "vpc-24ba90ce"),
				mockDeleteInternetService("igw-c3c49899"),

				mockGetSubnetIdsFromNetIds("vpc-24ba90ce", []string{
					"subnet-c1a282b0", "subnet-1555ea91", "subnet-174f5ec4",
				}),
				mockDeleteSubnet("subnet-c1a282b0"),
				mockDeleteSubnet("subnet-1555ea91"),
				mockDeleteSubnet("subnet-174f5ec4"),
				mockNetFound("vpc-24ba90ce"),
				mockDeleteNet("vpc-24ba90ce"),
			},
			assertDeleted: true,
		},
		{
			name:           "Trying to delete all sg, even if some cannot be",
			clusterSpec:    "ready",
			clusterPatches: []patchOSCClusterFunc{patchDeleteCluster()},
			mockFuncs: []mockFunc{
				mockLoadBalancerFound("test-cluster-api-k8s", false),

				mockNatServiceFound("nat-223a4dd4"),
				mockDeleteNatService("nat-223a4dd4"),

				mockValidatePublicIpIdsOk([]string{"eipalloc-da72a57c"}),
				mockCheckPublicIpUnlink("eipalloc-da72a57c"),
				mockDeletePublicIp("eipalloc-da72a57c"),

				mockRouteTablesForNetFound("vpc-24ba90ce", []string{"rtb-194c971e", "rtb-0a4640a6", "rtb-eeacfe8a"}),
				mockDeleteRoute("rtb-0a4640a6", "0.0.0.0/0"),
				mockUnlinkRouteTable("rtbassoc-643430b3"),
				mockDeleteRouteTable("rtb-0a4640a6"),
				mockDeleteRoute("rtb-194c971e", "0.0.0.0/0"),
				mockUnlinkRouteTable("rtbassoc-09475c37"),
				mockDeleteRouteTable("rtb-194c971e"),
				mockDeleteRoute("rtb-eeacfe8a", "0.0.0.0/0"),
				mockUnlinkRouteTable("rtbassoc-90bda9c8"),
				mockDeleteRouteTable("rtb-eeacfe8a"),

				mockSecurityGroupsForNetFound("vpc-24ba90ce", []string{"sg-750ae810", "sg-a093d014", "sg-7eb16ccb", "sg-0cd1f87e"}),

				mockSecurityGroupCleanAllRules("sg-a093d014"),
				mockDeleteSecurityGroup("sg-a093d014", errors.New("foo")),

				mockSecurityGroupCleanAllRules("sg-750ae810"),
				mockDeleteSecurityGroup("sg-750ae810", nil),

				mockSecurityGroupCleanAllRules("sg-7eb16ccb"),
				mockDeleteSecurityGroup("sg-7eb16ccb", errors.New("foo")),

				mockSecurityGroupCleanAllRules("sg-0cd1f87e"),
				mockDeleteSecurityGroup("sg-0cd1f87e", nil),
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
