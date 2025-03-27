package controllers_test

import (
	"context"
	"testing"

	"github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	"github.com/outscale/cluster-api-provider-outscale/controllers"
	osc "github.com/outscale/osc-sdk-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func runClusterTest(t *testing.T, tc testcase) {
	c, oc := loadClusterSpecs(t, tc.clusterSpec, tc.clusterBaseSpec)
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
	rec := controllers.OscClusterReconciler{
		Client: client,
		Tracker: &controllers.ClusterResourceTracker{
			Cloud: cs,
		},
		Cloud: cs,
	}
	nsn := types.NamespacedName{
		Namespace: oc.Namespace,
		Name:      oc.Name,
	}
	step := &tc
	for step != nil {
		for _, fn := range step.mockFuncs {
			fn(cs)
		}
		res, err := rec.Reconcile(context.TODO(), controllerruntime.Request{NamespacedName: nsn})
		if step.hasError {
			require.Error(t, err)
			assert.Zero(t, res)
		} else {
			require.NoError(t, err)
			assert.Equal(t, step.requeue, res.RequeueAfter > 0 || res.Requeue)
		}
		var out v1beta1.OscCluster
		err = client.Get(context.TODO(), nsn, &out)
		switch {
		case step.assertDeleted:
			require.True(t, apierrors.IsNotFound(err), "resource must have been deleted")
		default:
			require.NoError(t, err, "resource was not found")
			for _, fn := range step.clusterAsserts {
				fn(t, &out)
			}
		}
		step = step.next
	}
}

func TestReconcileOSCCluster_Create(t *testing.T) {
	tcs := []testcase{
		{
			name:        "creating a cluster with a v0.4 manual config",
			clusterSpec: "base-0.4",
			mockFuncs: []mockFunc{
				mockReadOwnedByTagNoneFound(tag.NetResourceType, "9e1db9c4-bf0a-4583-8999-203ec002c520"),
				mockReadTagByNameNoneFound(tag.NetResourceType, "test-cluster-api-net-9e1db9c4-bf0a-4583-8999-203ec002c520"),
				mockCreateNet(infrastructurev1beta1.OscNet{
					Name:        "test-cluster-api-net",
					IpRange:     "10.0.0.0/16",
					ClusterName: "test-cluster-api",
				}, "9e1db9c4-bf0a-4583-8999-203ec002c520", "test-cluster-api-net", "vpc-foo"),
				mockGetSubnetFromNet("vpc-foo", "10.0.4.0/24", nil),
				mockCreateSubnet(infrastructurev1beta1.OscSubnet{
					Name:          "test-cluster-api-subnet-kcp",
					IpSubnetRange: "10.0.4.0/24",
					SubregionName: "eu-west-2a",
				}, "vpc-foo", "9e1db9c4-bf0a-4583-8999-203ec002c520", "subnet-kcp"),
				mockGetSubnetFromNet("vpc-foo", "10.0.3.0/24", nil),
				mockCreateSubnet(infrastructurev1beta1.OscSubnet{
					Name:          "test-cluster-api-subnet-kw",
					IpSubnetRange: "10.0.3.0/24",
					SubregionName: "eu-west-2a",
				}, "vpc-foo", "9e1db9c4-bf0a-4583-8999-203ec002c520", "subnet-kw"),
				mockGetSubnetFromNet("vpc-foo", "10.0.2.0/24", nil),
				mockCreateSubnet(infrastructurev1beta1.OscSubnet{
					Name:          "test-cluster-api-subnet-public",
					IpSubnetRange: "10.0.2.0/24",
					SubregionName: "eu-west-2a",
				}, "vpc-foo", "9e1db9c4-bf0a-4583-8999-203ec002c520", "subnet-public"),
				mockGetInternetServiceForNet("vpc-foo", nil),
				mockCreateInternetService("test-cluster-api-internetservice", "9e1db9c4-bf0a-4583-8999-203ec002c520", "igw-foo"),
				mockLinkInternetService("igw-foo", "vpc-foo"),

				mockGetSecurityGroupFromName("test-cluster-api-securitygroup-kw-9e1db9c4-bf0a-4583-8999-203ec002c520", nil),
				mockCreateSecurityGroup("vpc-foo", "9e1db9c4-bf0a-4583-8999-203ec002c520", "test-cluster-api-securitygroup-kw-9e1db9c4-bf0a-4583-8999-203ec002c520",
					"Security Group Kw with cluster-api", "", "sg-kw"),
				mockCreateSecurityGroupRule("sg-kw", "Inbound", "tcp", "10.0.0.0/16", 179, 179),
				mockCreateSecurityGroupRule("sg-kw", "Inbound", "tcp", "10.0.3.0/24", 10250, 10250),
				mockCreateSecurityGroupRule("sg-kw", "Inbound", "tcp", "10.0.4.0/24", 10250, 10250),
				mockCreateSecurityGroupRule("sg-kw", "Inbound", "tcp", "10.0.3.0/24", 30000, 32767),
				mockCreateSecurityGroupRule("sg-kw", "Inbound", "tcp", "10.0.4.0/24", 30000, 32767),

				mockGetSecurityGroupFromName("test-cluster-api-securitygroup-kcp-9e1db9c4-bf0a-4583-8999-203ec002c520", nil),
				mockCreateSecurityGroup("vpc-foo", "9e1db9c4-bf0a-4583-8999-203ec002c520", "test-cluster-api-securitygroup-kcp-9e1db9c4-bf0a-4583-8999-203ec002c520",
					"Security Group Kcp with cluster-api", "", "sg-kcp"),
				mockCreateSecurityGroupRule("sg-kcp", "Inbound", "tcp", "10.0.0.0/16", 179, 179),
				mockCreateSecurityGroupRule("sg-kcp", "Inbound", "tcp", "10.0.4.0/24", 10250, 10252),
				mockCreateSecurityGroupRule("sg-kcp", "Inbound", "tcp", "10.0.3.0/24", 30000, 32767),
				mockCreateSecurityGroupRule("sg-kcp", "Inbound", "tcp", "10.0.4.0/24", 30000, 32767),
				mockCreateSecurityGroupRule("sg-kcp", "Inbound", "tcp", "10.0.3.0/24", 6443, 6443),
				mockCreateSecurityGroupRule("sg-kcp", "Inbound", "tcp", "10.0.4.0/24", 6443, 6443),
				mockCreateSecurityGroupRule("sg-kcp", "Inbound", "tcp", "10.0.4.0/24", 2378, 2379),

				mockGetSecurityGroupFromName("test-cluster-api-securitygroup-lb-9e1db9c4-bf0a-4583-8999-203ec002c520", nil),
				mockCreateSecurityGroup("vpc-foo", "9e1db9c4-bf0a-4583-8999-203ec002c520", "test-cluster-api-securitygroup-lb-9e1db9c4-bf0a-4583-8999-203ec002c520",
					"Security Group Lb with cluster-api", "", "sg-lb"),
				mockCreateSecurityGroupRule("sg-lb", "Inbound", "tcp", "0.0.0.0/0", 6443, 6443),

				mockGetSecurityGroupFromName("test-cluster-api-securitygroup-node-9e1db9c4-bf0a-4583-8999-203ec002c520", nil),
				mockCreateSecurityGroup("vpc-foo", "9e1db9c4-bf0a-4583-8999-203ec002c520", "test-cluster-api-securitygroup-node-9e1db9c4-bf0a-4583-8999-203ec002c520",
					"Security Group Node with cluster-api", "OscK8sMainSG", "sg-node"),
				mockCreateSecurityGroupRule("sg-node", "Inbound", "udp", "10.0.0.0/16", 4789, 4789),
				mockCreateSecurityGroupRule("sg-node", "Inbound", "udp", "10.0.0.0/16", 5473, 5473),
				mockCreateSecurityGroupRule("sg-node", "Inbound", "udp", "10.0.0.0/16", 51820, 51820),
				mockCreateSecurityGroupRule("sg-node", "Inbound", "udp", "10.0.0.0/16", 51821, 51821),
				mockCreateSecurityGroupRule("sg-node", "Inbound", "udp", "10.0.0.0/16", 8285, 8285),
				mockCreateSecurityGroupRule("sg-node", "Inbound", "udp", "10.0.0.0/16", 8472, 8472),

				mockGetRouteTablesFromNet("vpc-foo", nil),
				mockCreateRouteTable("vpc-foo", "9e1db9c4-bf0a-4583-8999-203ec002c520", "test-cluster-api-routetable-public", "rtb-public"),
				mockLinkRouteTable("rtb-public", "subnet-public"),
				mockCreateRoute("rtb-public", "0.0.0.0/0", "igw-foo", "gateway"),

				mockGetNatServiceFromClientToken("test-cluster-api-natservice-9e1db9c4-bf0a-4583-8999-203ec002c520", nil),
				mockReadTagByNameNoneFound(tag.NatResourceType, "test-cluster-api-natservice-9e1db9c4-bf0a-4583-8999-203ec002c520"),
				mockCreatePublicIp("test-cluster-api-natservice", "9e1db9c4-bf0a-4583-8999-203ec002c520", "ipalloc-nat", "1.2.3.4"),
				mockCreateNatService("ipalloc-nat", "subnet-public", "test-cluster-api-natservice-9e1db9c4-bf0a-4583-8999-203ec002c520", "test-cluster-api-natservice", "9e1db9c4-bf0a-4583-8999-203ec002c520", "nat-foo"),

				mockGetRouteTablesFromNet("vpc-foo", []osc.RouteTable{
					{
						RouteTableId: ptr.To("rtb-public"), LinkRouteTables: &[]osc.LinkRouteTable{{SubnetId: ptr.To("subnet-public")}},
						Routes: &[]osc.Route{{DestinationIpRange: ptr.To("0.0.0.0/0"), GatewayId: ptr.To("igw-foo")}},
					},
				}),
				mockCreateRouteTable("vpc-foo", "9e1db9c4-bf0a-4583-8999-203ec002c520", "test-cluster-api-routetable-kcp", "rtb-kcp"),
				mockLinkRouteTable("rtb-kcp", "subnet-kcp"),
				mockCreateRoute("rtb-kcp", "0.0.0.0/0", "nat-foo", "nat"),
				mockCreateRouteTable("vpc-foo", "9e1db9c4-bf0a-4583-8999-203ec002c520", "test-cluster-api-routetable-kw", "rtb-kw"),
				mockLinkRouteTable("rtb-kw", "subnet-kw"),
				mockCreateRoute("rtb-kw", "0.0.0.0/0", "nat-foo", "nat"),

				mockGetLoadBalancer("test-cluster-api-k8s", nil),
				mockCreateLoadBalancer("test-cluster-api-k8s", "internet-facing", "subnet-public", "sg-lb"),
				mockConfigureHealthCheck("test-cluster-api-k8s"),
				mockCreateLoadBalancerTag("test-cluster-api-k8s", "test-cluster-api-k8s-9e1db9c4-bf0a-4583-8999-203ec002c520"),
			},
			clusterAsserts: []assertOSCClusterFunc{
				assertHasClusterFinalizer(),
				assertStatusResources(infrastructurev1beta1.OscClusterResources{
					Net: map[string]string{
						"default": "vpc-foo",
					},
					Subnet: map[string]string{
						"10.0.2.0/24": "subnet-public",
						"10.0.3.0/24": "subnet-kw",
						"10.0.4.0/24": "subnet-kcp",
					},
					InternetService: map[string]string{
						"default": "igw-foo",
					},
					SecurityGroup: map[string]string{
						"test-cluster-api-securitygroup-kcp-9e1db9c4-bf0a-4583-8999-203ec002c520":  "sg-kcp",
						"test-cluster-api-securitygroup-kw-9e1db9c4-bf0a-4583-8999-203ec002c520":   "sg-kw",
						"test-cluster-api-securitygroup-lb-9e1db9c4-bf0a-4583-8999-203ec002c520":   "sg-lb",
						"test-cluster-api-securitygroup-node-9e1db9c4-bf0a-4583-8999-203ec002c520": "sg-node",
					},
					NatService: map[string]string{
						"test-cluster-api-natservice-9e1db9c4-bf0a-4583-8999-203ec002c520": "nat-foo",
					},
					PublicIPs: map[string]string{
						"test-cluster-api-natservice-9e1db9c4-bf0a-4583-8999-203ec002c520": "ipalloc-nat",
					},
				}),
				assertControlPlaneEndpoint("test-cluster-api-k8s.outscale.dev"),
			},
			next: &testcase{
				name: "A second run has all references in cache",
			},
		},
		{
			name:        "creating a cluster with a v0.5 automatic config",
			clusterSpec: "base-0.5",
			mockFuncs: []mockFunc{
				mockReadOwnedByTagNoneFound(tag.NetResourceType, "9e1db9c4-bf0a-4583-8999-203ec002c520"),
				mockCreateNet(infrastructurev1beta1.OscNet{
					IpRange: "10.0.0.0/16",
				}, "9e1db9c4-bf0a-4583-8999-203ec002c520", "Net for test-cluster-api", "vpc-foo"),
				mockGetSubnetFromNet("vpc-foo", "10.0.4.0/24", nil),
				mockCreateSubnet(infrastructurev1beta1.OscSubnet{
					Name:          "Controlplane subnet for test-cluster-api/eu-west-2a",
					IpSubnetRange: "10.0.4.0/24",
					SubregionName: "eu-west-2a",
					Roles:         []infrastructurev1beta1.OscRole{infrastructurev1beta1.RoleControlPlane},
				}, "vpc-foo", "9e1db9c4-bf0a-4583-8999-203ec002c520", "subnet-kcp"),
				mockGetSubnetFromNet("vpc-foo", "10.0.3.0/24", nil),
				mockCreateSubnet(infrastructurev1beta1.OscSubnet{
					Name:          "Worker subnet for test-cluster-api/eu-west-2a",
					IpSubnetRange: "10.0.3.0/24",
					SubregionName: "eu-west-2a",
					Roles:         []infrastructurev1beta1.OscRole{infrastructurev1beta1.RoleWorker},
				}, "vpc-foo", "9e1db9c4-bf0a-4583-8999-203ec002c520", "subnet-kw"),
				mockGetSubnetFromNet("vpc-foo", "10.0.2.0/24", nil),
				mockCreateSubnet(infrastructurev1beta1.OscSubnet{
					Name:          "Public subnet for test-cluster-api/eu-west-2a",
					IpSubnetRange: "10.0.2.0/24",
					SubregionName: "eu-west-2a",
					Roles:         []infrastructurev1beta1.OscRole{infrastructurev1beta1.RoleLoadBalancer, infrastructurev1beta1.RoleBastion, infrastructurev1beta1.RoleNat},
				}, "vpc-foo", "9e1db9c4-bf0a-4583-8999-203ec002c520", "subnet-public"),
				mockGetInternetServiceForNet("vpc-foo", nil),
				mockCreateInternetService("Internet Service for test-cluster-api", "9e1db9c4-bf0a-4583-8999-203ec002c520", "igw-foo"),
				mockLinkInternetService("igw-foo", "vpc-foo"),

				mockGetSecurityGroupFromName("test-cluster-api-worker-9e1db9c4-bf0a-4583-8999-203ec002c520", nil),
				mockCreateSecurityGroup("vpc-foo", "9e1db9c4-bf0a-4583-8999-203ec002c520", "test-cluster-api-worker-9e1db9c4-bf0a-4583-8999-203ec002c520",
					"Worker securityGroup for test-cluster-api", "", "sg-kw"),
				mockCreateSecurityGroupRule("sg-kw", "Inbound", "tcp", "10.0.3.0/24", 10250, 10250),
				mockCreateSecurityGroupRule("sg-kw", "Inbound", "tcp", "10.0.4.0/24", 10250, 10250),
				mockCreateSecurityGroupRule("sg-kw", "Inbound", "tcp", "10.0.3.0/24", 30000, 32767),
				mockCreateSecurityGroupRule("sg-kw", "Inbound", "tcp", "10.0.4.0/24", 30000, 32767),

				mockGetSecurityGroupFromName("test-cluster-api-controlplane-9e1db9c4-bf0a-4583-8999-203ec002c520", nil),
				mockCreateSecurityGroup("vpc-foo", "9e1db9c4-bf0a-4583-8999-203ec002c520", "test-cluster-api-controlplane-9e1db9c4-bf0a-4583-8999-203ec002c520",
					"Controlplane securityGroup for test-cluster-api", "", "sg-kcp"),
				mockCreateSecurityGroupRule("sg-kcp", "Inbound", "tcp", "10.0.4.0/24", 10250, 10252),
				mockCreateSecurityGroupRule("sg-kcp", "Inbound", "tcp", "10.0.3.0/24", 30000, 32767),
				mockCreateSecurityGroupRule("sg-kcp", "Inbound", "tcp", "10.0.4.0/24", 30000, 32767),
				mockCreateSecurityGroupRule("sg-kcp", "Inbound", "tcp", "10.0.0.0/16", 6443, 6443),
				mockCreateSecurityGroupRule("sg-kcp", "Inbound", "tcp", "10.0.4.0/24", 2378, 2380),

				mockGetSecurityGroupFromName("test-cluster-api-lb-9e1db9c4-bf0a-4583-8999-203ec002c520", nil),
				mockCreateSecurityGroup("vpc-foo", "9e1db9c4-bf0a-4583-8999-203ec002c520", "test-cluster-api-lb-9e1db9c4-bf0a-4583-8999-203ec002c520",
					"LB securityGroup for test-cluster-api", "", "sg-lb"),
				mockCreateSecurityGroupRule("sg-lb", "Inbound", "tcp", "0.0.0.0/0", 6443, 6443),
				mockCreateSecurityGroupRule("sg-lb", "Outbound", "tcp", "10.0.4.0/24", 6443, 6443),

				mockGetSecurityGroupFromName("test-cluster-api-node-9e1db9c4-bf0a-4583-8999-203ec002c520", nil),
				mockCreateSecurityGroup("vpc-foo", "9e1db9c4-bf0a-4583-8999-203ec002c520", "test-cluster-api-node-9e1db9c4-bf0a-4583-8999-203ec002c520",
					"Node securityGroup for test-cluster-api", "OscK8sMainSG", "sg-node"),
				mockCreateSecurityGroupRule("sg-node", "Inbound", "icmp", "10.0.0.0/16", 8, 8),
				mockCreateSecurityGroupRule("sg-node", "Inbound", "tcp", "10.0.0.0/16", 179, 179),
				mockCreateSecurityGroupRule("sg-node", "Inbound", "udp", "10.0.0.0/16", 4789, 4789),
				mockCreateSecurityGroupRule("sg-node", "Inbound", "udp", "10.0.0.0/16", 5473, 5473),
				mockCreateSecurityGroupRule("sg-node", "Inbound", "udp", "10.0.0.0/16", 51820, 51821),
				mockCreateSecurityGroupRule("sg-node", "Inbound", "udp", "10.0.0.0/16", 8285, 8285),
				mockCreateSecurityGroupRule("sg-node", "Inbound", "udp", "10.0.0.0/16", 8472, 8472),
				mockCreateSecurityGroupRule("sg-node", "Inbound", "tcp", "10.0.0.0/16", 4240, 4240),
				mockCreateSecurityGroupRule("sg-node", "Inbound", "tcp", "10.0.0.0/16", 4244, 4244),

				mockGetRouteTablesFromNet("vpc-foo", nil),
				mockCreateRouteTable("vpc-foo", "9e1db9c4-bf0a-4583-8999-203ec002c520", "Public subnet for test-cluster-api/eu-west-2a", "rtb-public"),
				mockLinkRouteTable("rtb-public", "subnet-public"),
				mockCreateRoute("rtb-public", "0.0.0.0/0", "igw-foo", "gateway"),

				mockGetNatServiceFromClientToken("eu-west-2a-9e1db9c4-bf0a-4583-8999-203ec002c520", nil),
				mockCreatePublicIp("Nat service for test-cluster-api", "9e1db9c4-bf0a-4583-8999-203ec002c520", "ipalloc-nat", "1.2.3.4"),
				mockCreateNatService("ipalloc-nat", "subnet-public", "eu-west-2a-9e1db9c4-bf0a-4583-8999-203ec002c520", "Nat service for test-cluster-api", "9e1db9c4-bf0a-4583-8999-203ec002c520", "nat-foo"),

				mockGetRouteTablesFromNet("vpc-foo", []osc.RouteTable{
					{
						RouteTableId: ptr.To("rtb-public"), LinkRouteTables: &[]osc.LinkRouteTable{{SubnetId: ptr.To("subnet-public")}},
						Routes: &[]osc.Route{{DestinationIpRange: ptr.To("0.0.0.0/0"), GatewayId: ptr.To("igw-foo")}},
					},
				}),
				mockCreateRouteTable("vpc-foo", "9e1db9c4-bf0a-4583-8999-203ec002c520", "Controlplane subnet for test-cluster-api/eu-west-2a", "rtb-kcp"),
				mockLinkRouteTable("rtb-kcp", "subnet-kcp"),
				mockCreateRoute("rtb-kcp", "0.0.0.0/0", "nat-foo", "nat"),
				mockCreateRouteTable("vpc-foo", "9e1db9c4-bf0a-4583-8999-203ec002c520", "Worker subnet for test-cluster-api/eu-west-2a", "rtb-kw"),
				mockLinkRouteTable("rtb-kw", "subnet-kw"),
				mockCreateRoute("rtb-kw", "0.0.0.0/0", "nat-foo", "nat"),

				mockGetLoadBalancer("test-cluster-api-k8s", nil),
				mockCreateLoadBalancer("test-cluster-api-k8s", "internet-facing", "subnet-public", "sg-lb"),
				mockConfigureHealthCheck("test-cluster-api-k8s"),
				mockCreateLoadBalancerTag("test-cluster-api-k8s", "test-cluster-api-k8s-9e1db9c4-bf0a-4583-8999-203ec002c520"),
			},
			clusterAsserts: []assertOSCClusterFunc{
				assertHasClusterFinalizer(),
				assertStatusResources(infrastructurev1beta1.OscClusterResources{
					Net: map[string]string{
						"default": "vpc-foo",
					},
					Subnet: map[string]string{
						"10.0.2.0/24": "subnet-public",
						"10.0.3.0/24": "subnet-kw",
						"10.0.4.0/24": "subnet-kcp",
					},
					InternetService: map[string]string{
						"default": "igw-foo",
					},
					SecurityGroup: map[string]string{
						"test-cluster-api-controlplane-9e1db9c4-bf0a-4583-8999-203ec002c520": "sg-kcp",
						"test-cluster-api-worker-9e1db9c4-bf0a-4583-8999-203ec002c520":       "sg-kw",
						"test-cluster-api-lb-9e1db9c4-bf0a-4583-8999-203ec002c520":           "sg-lb",
						"test-cluster-api-node-9e1db9c4-bf0a-4583-8999-203ec002c520":         "sg-node",
					},
					NatService: map[string]string{
						"eu-west-2a-9e1db9c4-bf0a-4583-8999-203ec002c520": "nat-foo",
					},
					PublicIPs: map[string]string{
						"eu-west-2a-9e1db9c4-bf0a-4583-8999-203ec002c520": "ipalloc-nat",
					},
				}),
				assertControlPlaneEndpoint("test-cluster-api-k8s.outscale.dev"),
			},
			next: &testcase{
				name: "A second run has all references in cache",
			},
		},
		{
			name:            "creating a cluster with a v0.5 automatic config and a bastion",
			clusterSpec:     "base-bastion-0.5",
			clusterBaseSpec: "base",
			mockFuncs: []mockFunc{
				mockReadOwnedByTagNoneFound(tag.NetResourceType, "9e1db9c4-bf0a-4583-8999-203ec002c520"),
				mockCreateNet(infrastructurev1beta1.OscNet{
					IpRange: "10.0.0.0/16",
				}, "9e1db9c4-bf0a-4583-8999-203ec002c520", "Net for test-cluster-api", "vpc-foo"),
				mockGetSubnetFromNet("vpc-foo", "10.0.4.0/24", nil),
				mockCreateSubnet(infrastructurev1beta1.OscSubnet{
					Name:          "Controlplane subnet for test-cluster-api/eu-west-2a",
					IpSubnetRange: "10.0.4.0/24",
					SubregionName: "eu-west-2a",
					Roles:         []infrastructurev1beta1.OscRole{infrastructurev1beta1.RoleControlPlane},
				}, "vpc-foo", "9e1db9c4-bf0a-4583-8999-203ec002c520", "subnet-kcp"),
				mockGetSubnetFromNet("vpc-foo", "10.0.3.0/24", nil),
				mockCreateSubnet(infrastructurev1beta1.OscSubnet{
					Name:          "Worker subnet for test-cluster-api/eu-west-2a",
					IpSubnetRange: "10.0.3.0/24",
					SubregionName: "eu-west-2a",
					Roles:         []infrastructurev1beta1.OscRole{infrastructurev1beta1.RoleWorker},
				}, "vpc-foo", "9e1db9c4-bf0a-4583-8999-203ec002c520", "subnet-kw"),
				mockGetSubnetFromNet("vpc-foo", "10.0.2.0/24", nil),
				mockCreateSubnet(infrastructurev1beta1.OscSubnet{
					Name:          "Public subnet for test-cluster-api/eu-west-2a",
					IpSubnetRange: "10.0.2.0/24",
					SubregionName: "eu-west-2a",
					Roles:         []infrastructurev1beta1.OscRole{infrastructurev1beta1.RoleLoadBalancer, infrastructurev1beta1.RoleBastion, infrastructurev1beta1.RoleNat},
				}, "vpc-foo", "9e1db9c4-bf0a-4583-8999-203ec002c520", "subnet-public"),
				mockGetInternetServiceForNet("vpc-foo", nil),
				mockCreateInternetService("Internet Service for test-cluster-api", "9e1db9c4-bf0a-4583-8999-203ec002c520", "igw-foo"),
				mockLinkInternetService("igw-foo", "vpc-foo"),

				mockGetSecurityGroupFromName("test-cluster-api-worker-9e1db9c4-bf0a-4583-8999-203ec002c520", nil),
				mockCreateSecurityGroup("vpc-foo", "9e1db9c4-bf0a-4583-8999-203ec002c520", "test-cluster-api-worker-9e1db9c4-bf0a-4583-8999-203ec002c520",
					"Worker securityGroup for test-cluster-api", "", "sg-kw"),
				mockCreateSecurityGroupRule("sg-kw", "Inbound", "tcp", "10.0.3.0/24", 10250, 10250),
				mockCreateSecurityGroupRule("sg-kw", "Inbound", "tcp", "10.0.4.0/24", 10250, 10250),
				mockCreateSecurityGroupRule("sg-kw", "Inbound", "tcp", "10.0.3.0/24", 30000, 32767),
				mockCreateSecurityGroupRule("sg-kw", "Inbound", "tcp", "10.0.4.0/24", 30000, 32767),

				mockGetSecurityGroupFromName("test-cluster-api-controlplane-9e1db9c4-bf0a-4583-8999-203ec002c520", nil),
				mockCreateSecurityGroup("vpc-foo", "9e1db9c4-bf0a-4583-8999-203ec002c520", "test-cluster-api-controlplane-9e1db9c4-bf0a-4583-8999-203ec002c520",
					"Controlplane securityGroup for test-cluster-api", "", "sg-kcp"),
				mockCreateSecurityGroupRule("sg-kcp", "Inbound", "tcp", "10.0.4.0/24", 10250, 10252),
				mockCreateSecurityGroupRule("sg-kcp", "Inbound", "tcp", "10.0.3.0/24", 30000, 32767),
				mockCreateSecurityGroupRule("sg-kcp", "Inbound", "tcp", "10.0.4.0/24", 30000, 32767),
				mockCreateSecurityGroupRule("sg-kcp", "Inbound", "tcp", "10.0.0.0/16", 6443, 6443),
				mockCreateSecurityGroupRule("sg-kcp", "Inbound", "tcp", "10.0.4.0/24", 2378, 2380),

				mockGetSecurityGroupFromName("test-cluster-api-lb-9e1db9c4-bf0a-4583-8999-203ec002c520", nil),
				mockCreateSecurityGroup("vpc-foo", "9e1db9c4-bf0a-4583-8999-203ec002c520", "test-cluster-api-lb-9e1db9c4-bf0a-4583-8999-203ec002c520",
					"LB securityGroup for test-cluster-api", "", "sg-lb"),
				mockCreateSecurityGroupRule("sg-lb", "Inbound", "tcp", "0.0.0.0/0", 6443, 6443),
				mockCreateSecurityGroupRule("sg-lb", "Outbound", "tcp", "10.0.4.0/24", 6443, 6443),

				mockGetSecurityGroupFromName("test-cluster-api-node-9e1db9c4-bf0a-4583-8999-203ec002c520", nil),
				mockCreateSecurityGroup("vpc-foo", "9e1db9c4-bf0a-4583-8999-203ec002c520", "test-cluster-api-node-9e1db9c4-bf0a-4583-8999-203ec002c520",
					"Node securityGroup for test-cluster-api", "OscK8sMainSG", "sg-node"),
				mockCreateSecurityGroupRule("sg-node", "Inbound", "icmp", "10.0.0.0/16", 8, 8),
				mockCreateSecurityGroupRule("sg-node", "Inbound", "tcp", "10.0.0.0/16", 179, 179),
				mockCreateSecurityGroupRule("sg-node", "Inbound", "udp", "10.0.0.0/16", 4789, 4789),
				mockCreateSecurityGroupRule("sg-node", "Inbound", "udp", "10.0.0.0/16", 5473, 5473),
				mockCreateSecurityGroupRule("sg-node", "Inbound", "udp", "10.0.0.0/16", 51820, 51821),
				mockCreateSecurityGroupRule("sg-node", "Inbound", "udp", "10.0.0.0/16", 8285, 8285),
				mockCreateSecurityGroupRule("sg-node", "Inbound", "udp", "10.0.0.0/16", 8472, 8472),
				mockCreateSecurityGroupRule("sg-node", "Inbound", "tcp", "10.0.0.0/16", 4240, 4240),
				mockCreateSecurityGroupRule("sg-node", "Inbound", "tcp", "10.0.0.0/16", 4244, 4244),
				mockCreateSecurityGroupRule("sg-node", "Inbound", "tcp", "10.0.2.0/24", 22, 22),

				mockGetSecurityGroupFromName("test-cluster-api-bastion-9e1db9c4-bf0a-4583-8999-203ec002c520", nil),
				mockCreateSecurityGroup("vpc-foo", "9e1db9c4-bf0a-4583-8999-203ec002c520", "test-cluster-api-bastion-9e1db9c4-bf0a-4583-8999-203ec002c520",
					"Bastion securityGroup for test-cluster-api", "", "sg-bastion"),
				mockCreateSecurityGroupRule("sg-bastion", "Inbound", "tcp", "0.0.0.0/0", 22, 22),
				mockCreateSecurityGroupRule("sg-bastion", "Outbound", "tcp", "10.0.0.0/16", 22, 22),

				mockGetRouteTablesFromNet("vpc-foo", nil),
				mockCreateRouteTable("vpc-foo", "9e1db9c4-bf0a-4583-8999-203ec002c520", "Public subnet for test-cluster-api/eu-west-2a", "rtb-public"),
				mockLinkRouteTable("rtb-public", "subnet-public"),
				mockCreateRoute("rtb-public", "0.0.0.0/0", "igw-foo", "gateway"),

				mockGetNatServiceFromClientToken("eu-west-2a-9e1db9c4-bf0a-4583-8999-203ec002c520", nil),
				mockCreatePublicIp("Nat service for test-cluster-api", "9e1db9c4-bf0a-4583-8999-203ec002c520", "ipalloc-nat", "1.2.3.4"),
				mockCreateNatService("ipalloc-nat", "subnet-public", "eu-west-2a-9e1db9c4-bf0a-4583-8999-203ec002c520", "Nat service for test-cluster-api", "9e1db9c4-bf0a-4583-8999-203ec002c520", "nat-foo"),

				mockGetRouteTablesFromNet("vpc-foo", []osc.RouteTable{
					{
						RouteTableId: ptr.To("rtb-public"), LinkRouteTables: &[]osc.LinkRouteTable{{SubnetId: ptr.To("subnet-public")}},
						Routes: &[]osc.Route{{DestinationIpRange: ptr.To("0.0.0.0/0"), GatewayId: ptr.To("igw-foo")}},
					},
				}),
				mockCreateRouteTable("vpc-foo", "9e1db9c4-bf0a-4583-8999-203ec002c520", "Controlplane subnet for test-cluster-api/eu-west-2a", "rtb-kcp"),
				mockLinkRouteTable("rtb-kcp", "subnet-kcp"),
				mockCreateRoute("rtb-kcp", "0.0.0.0/0", "nat-foo", "nat"),
				mockCreateRouteTable("vpc-foo", "9e1db9c4-bf0a-4583-8999-203ec002c520", "Worker subnet for test-cluster-api/eu-west-2a", "rtb-kw"),
				mockLinkRouteTable("rtb-kw", "subnet-kw"),
				mockCreateRoute("rtb-kw", "0.0.0.0/0", "nat-foo", "nat"),

				mockGetLoadBalancer("test-cluster-api-k8s", nil),
				mockCreateLoadBalancer("test-cluster-api-k8s", "internet-facing", "subnet-public", "sg-lb"),
				mockConfigureHealthCheck("test-cluster-api-k8s"),
				mockCreateLoadBalancerTag("test-cluster-api-k8s", "test-cluster-api-k8s-9e1db9c4-bf0a-4583-8999-203ec002c520"),

				mockGetVmFromClientToken("bastion-9e1db9c4-bf0a-4583-8999-203ec002c520", nil),
				mockCreatePublicIp("Bastion for test-cluster-api", "9e1db9c4-bf0a-4583-8999-203ec002c520", "ipalloc-bastion", "1.2.3.4"),
				mockCreateVmBastion("i-bastion", "subnet-public", []string{"sg-bastion"}, []string{}, "Bastion for test-cluster-api", "bastion-9e1db9c4-bf0a-4583-8999-203ec002c520", "ami-bastion", map[string]string{"osc.fcu.eip.auto-attach": "1.2.3.4"}),
			},
			hasError: true,
			clusterAsserts: []assertOSCClusterFunc{
				assertHasClusterFinalizer(),
				assertStatusResources(infrastructurev1beta1.OscClusterResources{
					Net: map[string]string{
						"default": "vpc-foo",
					},
					Subnet: map[string]string{
						"10.0.2.0/24": "subnet-public",
						"10.0.3.0/24": "subnet-kw",
						"10.0.4.0/24": "subnet-kcp",
					},
					InternetService: map[string]string{
						"default": "igw-foo",
					},
					SecurityGroup: map[string]string{
						"test-cluster-api-controlplane-9e1db9c4-bf0a-4583-8999-203ec002c520": "sg-kcp",
						"test-cluster-api-worker-9e1db9c4-bf0a-4583-8999-203ec002c520":       "sg-kw",
						"test-cluster-api-lb-9e1db9c4-bf0a-4583-8999-203ec002c520":           "sg-lb",
						"test-cluster-api-node-9e1db9c4-bf0a-4583-8999-203ec002c520":         "sg-node",
						"test-cluster-api-bastion-9e1db9c4-bf0a-4583-8999-203ec002c520":      "sg-bastion",
					},
					NatService: map[string]string{
						"eu-west-2a-9e1db9c4-bf0a-4583-8999-203ec002c520": "nat-foo",
					},
					PublicIPs: map[string]string{
						"bastion": "ipalloc-bastion",
						"eu-west-2a-9e1db9c4-bf0a-4583-8999-203ec002c520": "ipalloc-nat",
					},
					Bastion: map[string]string{
						"default": "i-bastion",
					},
				}),
			},
		},
		{
			name:            "reusing a network",
			clusterSpec:     "reuse-0.5",
			clusterBaseSpec: "base",
			mockFuncs: []mockFunc{
				mockNetFound("vpc-foo"),

				mockSubnetFound("subnet-kcp"),
				mockSubnetFound("subnet-kw"),
				mockSubnetFound("subnet-public"),

				mockGetLoadBalancer("test-cluster-api-k8s", nil),
				mockCreateLoadBalancer("test-cluster-api-k8s", "internet-facing", "subnet-public", "sg-lb"),
				mockConfigureHealthCheck("test-cluster-api-k8s"),
				mockCreateLoadBalancerTag("test-cluster-api-k8s", "test-cluster-api-k8s-9e1db9c4-bf0a-4583-8999-203ec002c520"),
			},
			clusterAsserts: []assertOSCClusterFunc{
				assertHasClusterFinalizer(),
				assertStatusResources(infrastructurev1beta1.OscClusterResources{}),
				assertControlPlaneEndpoint("test-cluster-api-k8s.outscale.dev"),
			},
		},
		{
			name:            "reusing a network, net is not created if missing",
			clusterSpec:     "reuse-0.5",
			clusterBaseSpec: "base",
			mockFuncs: []mockFunc{
				mockGetNet("vpc-foo", nil),
			},
			hasError: true,
		},
		{
			name:            "reusing a network, subnet is not created if missing",
			clusterSpec:     "reuse-0.5",
			clusterBaseSpec: "base",
			mockFuncs: []mockFunc{
				mockNetFound("vpc-foo"),
				mockGetSubnet("subnet-kcp", nil),
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

func TestReconcileOSCCluster_Update(t *testing.T) {
	tcs := []testcase{
		{
			name:        "An inbound rule may be added to a 0.4 cluster (IpRange)",
			clusterSpec: "ready-0.4",
			clusterPatches: []patchOSCClusterFunc{
				patchAddSGRule("test-cluster-api-securitygroup-kcp", infrastructurev1beta1.OscSecurityGroupRule{
					Flow: "Inbound", IpProtocol: "udp", FromPortRange: 32, ToPortRange: 32, IpRange: "1.2.3.4/32",
				}),
			},
			mockFuncs: []mockFunc{
				mockNetFound("vpc-24ba90ce"),
				mockSubnetFound("subnet-c1a282b0"),
				mockSubnetFound("subnet-1555ea91"),
				mockSubnetFound("subnet-174f5ec4"),
				mockInternetServiceFound("vpc-24ba90ce", "igw-c3c49899"),

				mockGetSecurityGroup("sg-750ae810", &osc.SecurityGroup{
					InboundRules: &[]osc.SecurityGroupRule{
						{IpProtocol: ptr.To("tcp"), FromPortRange: ptr.To[int32](179), ToPortRange: ptr.To[int32](179), IpRanges: &[]string{"10.0.0.0/16"}},
						{IpProtocol: ptr.To("tcp"), FromPortRange: ptr.To[int32](10250), ToPortRange: ptr.To[int32](10250), IpRanges: &[]string{"10.0.3.0/16"}},
						{IpProtocol: ptr.To("tcp"), FromPortRange: ptr.To[int32](6443), ToPortRange: ptr.To[int32](6443), IpRanges: &[]string{"10.0.3.0/16"}},
						{IpProtocol: ptr.To("tcp"), FromPortRange: ptr.To[int32](30000), ToPortRange: ptr.To[int32](32767), IpRanges: &[]string{"10.0.3.0/16"}},
						{IpProtocol: ptr.To("tcp"), FromPortRange: ptr.To[int32](2378), ToPortRange: ptr.To[int32](2379), IpRanges: &[]string{"10.0.4.0/16"}},
						{IpProtocol: ptr.To("tcp"), FromPortRange: ptr.To[int32](10250), ToPortRange: ptr.To[int32](10252), IpRanges: &[]string{"10.0.4.0/16"}},
					},
				}),
				mockCreateSecurityGroupRule("sg-750ae810", "Inbound", "udp", "1.2.3.4/32", 32, 32),
				mockGetSecurityGroup("sg-a093d014", &osc.SecurityGroup{
					InboundRules: &[]osc.SecurityGroupRule{
						{IpProtocol: ptr.To("tcp"), FromPortRange: ptr.To[int32](179), ToPortRange: ptr.To[int32](179), IpRanges: &[]string{"10.0.0.0/16"}},
						{IpProtocol: ptr.To("tcp"), FromPortRange: ptr.To[int32](10250), ToPortRange: ptr.To[int32](10250), IpRanges: &[]string{"10.0.3.0/16"}},
						{IpProtocol: ptr.To("tcp"), FromPortRange: ptr.To[int32](30000), ToPortRange: ptr.To[int32](32767), IpRanges: &[]string{"10.0.4.0/16"}},
					},
				}),
				mockGetSecurityGroup("sg-7eb16ccb", &osc.SecurityGroup{
					InboundRules: &[]osc.SecurityGroupRule{
						{IpProtocol: ptr.To("tcp"), FromPortRange: ptr.To[int32](6443), ToPortRange: ptr.To[int32](6443), IpRanges: &[]string{"0.0.0.0/0"}},
					},
				}),
				mockGetSecurityGroup("sg-0cd1f87e", &osc.SecurityGroup{
					InboundRules: &[]osc.SecurityGroupRule{
						{IpProtocol: ptr.To("udp"), FromPortRange: ptr.To[int32](4789), ToPortRange: ptr.To[int32](4789), IpRanges: &[]string{"10.0.0.0/16"}},
						{IpProtocol: ptr.To("udp"), FromPortRange: ptr.To[int32](5473), ToPortRange: ptr.To[int32](5473), IpRanges: &[]string{"10.0.0.0/16"}},
						{IpProtocol: ptr.To("udp"), FromPortRange: ptr.To[int32](51820), ToPortRange: ptr.To[int32](51820), IpRanges: &[]string{"10.0.0.0/16"}},
						{IpProtocol: ptr.To("udp"), FromPortRange: ptr.To[int32](51821), ToPortRange: ptr.To[int32](51821), IpRanges: &[]string{"10.0.0.0/16"}},
						{IpProtocol: ptr.To("udp"), FromPortRange: ptr.To[int32](8285), ToPortRange: ptr.To[int32](8285), IpRanges: &[]string{"10.0.0.0/16"}},
						{IpProtocol: ptr.To("udp"), FromPortRange: ptr.To[int32](8472), ToPortRange: ptr.To[int32](8472), IpRanges: &[]string{"10.0.0.0/16"}},
					},
				}),

				mockGetRouteTablesFromNet("vpc-24ba90ce", []osc.RouteTable{
					{
						RouteTableId:    ptr.To("rtb-0a4640a6"),
						LinkRouteTables: &[]osc.LinkRouteTable{{LinkRouteTableId: ptr.To("rtbassoc-643430b3"), SubnetId: ptr.To("subnet-1555ea91")}},
						Routes:          &[]osc.Route{{DestinationIpRange: ptr.To("0.0.0.0/0")}},
					},
					{
						RouteTableId:    ptr.To("rtb-194c971e"),
						LinkRouteTables: &[]osc.LinkRouteTable{{LinkRouteTableId: ptr.To("rtbassoc-09475c37"), SubnetId: ptr.To("subnet-c1a282b0")}},
						Routes:          &[]osc.Route{{DestinationIpRange: ptr.To("0.0.0.0/0")}},
					},
					{
						RouteTableId:    ptr.To("rtb-eeacfe8a"),
						LinkRouteTables: &[]osc.LinkRouteTable{{LinkRouteTableId: ptr.To("rtbassoc-90bda9c8"), SubnetId: ptr.To("subnet-174f5ec4")}},
						Routes:          &[]osc.Route{{DestinationIpRange: ptr.To("0.0.0.0/0")}},
					},
				}),
				mockGetNatServiceFromClientToken("test-cluster-api-natservice-9e1db9c4-bf0a-4583-8999-203ec002c520", nil),
				mockReadTagByNameFound(tag.NatResourceType, "test-cluster-api-natservice-9e1db9c4-bf0a-4583-8999-203ec002c520", "nat-223a4dd4"),
				mockNatServiceFound("nat-223a4dd4"),
				mockGetRouteTablesFromNet("vpc-24ba90ce", []osc.RouteTable{
					{
						RouteTableId:    ptr.To("rtb-0a4640a6"),
						LinkRouteTables: &[]osc.LinkRouteTable{{LinkRouteTableId: ptr.To("rtbassoc-643430b3"), SubnetId: ptr.To("subnet-1555ea91")}},
						Routes:          &[]osc.Route{{DestinationIpRange: ptr.To("0.0.0.0/0")}},
					},
					{
						RouteTableId:    ptr.To("rtb-194c971e"),
						LinkRouteTables: &[]osc.LinkRouteTable{{LinkRouteTableId: ptr.To("rtbassoc-09475c37"), SubnetId: ptr.To("subnet-c1a282b0")}},
						Routes:          &[]osc.Route{{DestinationIpRange: ptr.To("0.0.0.0/0")}},
					},
					{
						RouteTableId:    ptr.To("rtb-eeacfe8a"),
						LinkRouteTables: &[]osc.LinkRouteTable{{LinkRouteTableId: ptr.To("rtbassoc-90bda9c8"), SubnetId: ptr.To("subnet-174f5ec4")}},
						Routes:          &[]osc.Route{{DestinationIpRange: ptr.To("0.0.0.0/0")}},
					},
				}),

				mockLoadBalancerFound("test-cluster-api-k8s", "test-cluster-api-k8s-9e1db9c4-bf0a-4583-8999-203ec002c520"),
			},
		},
		{
			name:        "An outbound rule may be added to a 0.4 cluster (IpRange)",
			clusterSpec: "ready-0.4",
			clusterPatches: []patchOSCClusterFunc{
				patchAddSGRule("test-cluster-api-securitygroup-kcp", infrastructurev1beta1.OscSecurityGroupRule{
					Flow: "Outbound", IpProtocol: "udp", FromPortRange: 32, ToPortRange: 32, IpRange: "1.2.3.4/32",
				}),
			},
			mockFuncs: []mockFunc{
				mockNetFound("vpc-24ba90ce"),
				mockSubnetFound("subnet-c1a282b0"),
				mockSubnetFound("subnet-1555ea91"),
				mockSubnetFound("subnet-174f5ec4"),
				mockInternetServiceFound("vpc-24ba90ce", "igw-c3c49899"),

				mockGetSecurityGroup("sg-750ae810", &osc.SecurityGroup{
					InboundRules: &[]osc.SecurityGroupRule{
						{IpProtocol: ptr.To("tcp"), FromPortRange: ptr.To[int32](179), ToPortRange: ptr.To[int32](179), IpRanges: &[]string{"10.0.0.0/16"}},
						{IpProtocol: ptr.To("tcp"), FromPortRange: ptr.To[int32](10250), ToPortRange: ptr.To[int32](10250), IpRanges: &[]string{"10.0.3.0/16"}},
						{IpProtocol: ptr.To("tcp"), FromPortRange: ptr.To[int32](6443), ToPortRange: ptr.To[int32](6443), IpRanges: &[]string{"10.0.3.0/16"}},
						{IpProtocol: ptr.To("tcp"), FromPortRange: ptr.To[int32](30000), ToPortRange: ptr.To[int32](32767), IpRanges: &[]string{"10.0.3.0/16"}},
						{IpProtocol: ptr.To("tcp"), FromPortRange: ptr.To[int32](2378), ToPortRange: ptr.To[int32](2379), IpRanges: &[]string{"10.0.4.0/16"}},
						{IpProtocol: ptr.To("tcp"), FromPortRange: ptr.To[int32](10250), ToPortRange: ptr.To[int32](10252), IpRanges: &[]string{"10.0.4.0/16"}},
					},
				}),
				mockCreateSecurityGroupRule("sg-750ae810", "Outbound", "udp", "1.2.3.4/32", 32, 32),
				mockGetSecurityGroup("sg-a093d014", &osc.SecurityGroup{
					InboundRules: &[]osc.SecurityGroupRule{
						{IpProtocol: ptr.To("tcp"), FromPortRange: ptr.To[int32](179), ToPortRange: ptr.To[int32](179), IpRanges: &[]string{"10.0.0.0/16"}},
						{IpProtocol: ptr.To("tcp"), FromPortRange: ptr.To[int32](10250), ToPortRange: ptr.To[int32](10250), IpRanges: &[]string{"10.0.3.0/16"}},
						{IpProtocol: ptr.To("tcp"), FromPortRange: ptr.To[int32](30000), ToPortRange: ptr.To[int32](32767), IpRanges: &[]string{"10.0.4.0/16"}},
					},
				}),
				mockGetSecurityGroup("sg-7eb16ccb", &osc.SecurityGroup{
					InboundRules: &[]osc.SecurityGroupRule{
						{IpProtocol: ptr.To("tcp"), FromPortRange: ptr.To[int32](6443), ToPortRange: ptr.To[int32](6443), IpRanges: &[]string{"0.0.0.0/0"}},
					},
				}),
				mockGetSecurityGroup("sg-0cd1f87e", &osc.SecurityGroup{
					InboundRules: &[]osc.SecurityGroupRule{
						{IpProtocol: ptr.To("udp"), FromPortRange: ptr.To[int32](4789), ToPortRange: ptr.To[int32](4789), IpRanges: &[]string{"10.0.0.0/16"}},
						{IpProtocol: ptr.To("udp"), FromPortRange: ptr.To[int32](5473), ToPortRange: ptr.To[int32](5473), IpRanges: &[]string{"10.0.0.0/16"}},
						{IpProtocol: ptr.To("udp"), FromPortRange: ptr.To[int32](51820), ToPortRange: ptr.To[int32](51820), IpRanges: &[]string{"10.0.0.0/16"}},
						{IpProtocol: ptr.To("udp"), FromPortRange: ptr.To[int32](51821), ToPortRange: ptr.To[int32](51821), IpRanges: &[]string{"10.0.0.0/16"}},
						{IpProtocol: ptr.To("udp"), FromPortRange: ptr.To[int32](8285), ToPortRange: ptr.To[int32](8285), IpRanges: &[]string{"10.0.0.0/16"}},
						{IpProtocol: ptr.To("udp"), FromPortRange: ptr.To[int32](8472), ToPortRange: ptr.To[int32](8472), IpRanges: &[]string{"10.0.0.0/16"}},
					},
				}),

				mockGetRouteTablesFromNet("vpc-24ba90ce", []osc.RouteTable{
					{
						RouteTableId:    ptr.To("rtb-0a4640a6"),
						LinkRouteTables: &[]osc.LinkRouteTable{{LinkRouteTableId: ptr.To("rtbassoc-643430b3"), SubnetId: ptr.To("subnet-1555ea91")}},
						Routes:          &[]osc.Route{{DestinationIpRange: ptr.To("0.0.0.0/0")}},
					},
					{
						RouteTableId:    ptr.To("rtb-194c971e"),
						LinkRouteTables: &[]osc.LinkRouteTable{{LinkRouteTableId: ptr.To("rtbassoc-09475c37"), SubnetId: ptr.To("subnet-c1a282b0")}},
						Routes:          &[]osc.Route{{DestinationIpRange: ptr.To("0.0.0.0/0")}},
					},
					{
						RouteTableId:    ptr.To("rtb-eeacfe8a"),
						LinkRouteTables: &[]osc.LinkRouteTable{{LinkRouteTableId: ptr.To("rtbassoc-90bda9c8"), SubnetId: ptr.To("subnet-174f5ec4")}},
						Routes:          &[]osc.Route{{DestinationIpRange: ptr.To("0.0.0.0/0")}},
					},
				}),
				mockGetNatServiceFromClientToken("test-cluster-api-natservice-9e1db9c4-bf0a-4583-8999-203ec002c520", nil),
				mockReadTagByNameFound(tag.NatResourceType, "test-cluster-api-natservice-9e1db9c4-bf0a-4583-8999-203ec002c520", "nat-223a4dd4"),
				mockNatServiceFound("nat-223a4dd4"),
				mockGetRouteTablesFromNet("vpc-24ba90ce", []osc.RouteTable{
					{
						RouteTableId:    ptr.To("rtb-0a4640a6"),
						LinkRouteTables: &[]osc.LinkRouteTable{{LinkRouteTableId: ptr.To("rtbassoc-643430b3"), SubnetId: ptr.To("subnet-1555ea91")}},
						Routes:          &[]osc.Route{{DestinationIpRange: ptr.To("0.0.0.0/0")}},
					},
					{
						RouteTableId:    ptr.To("rtb-194c971e"),
						LinkRouteTables: &[]osc.LinkRouteTable{{LinkRouteTableId: ptr.To("rtbassoc-09475c37"), SubnetId: ptr.To("subnet-c1a282b0")}},
						Routes:          &[]osc.Route{{DestinationIpRange: ptr.To("0.0.0.0/0")}},
					},
					{
						RouteTableId:    ptr.To("rtb-eeacfe8a"),
						LinkRouteTables: &[]osc.LinkRouteTable{{LinkRouteTableId: ptr.To("rtbassoc-90bda9c8"), SubnetId: ptr.To("subnet-174f5ec4")}},
						Routes:          &[]osc.Route{{DestinationIpRange: ptr.To("0.0.0.0/0")}},
					},
				}),

				mockLoadBalancerFound("test-cluster-api-k8s", "test-cluster-api-k8s-9e1db9c4-bf0a-4583-8999-203ec002c520"),
			},
		},
		{
			name:        "An inbound rule may be added to a 0.4 cluster (IpRanges)",
			clusterSpec: "ready-0.4",
			clusterPatches: []patchOSCClusterFunc{
				patchAddSGRule("test-cluster-api-securitygroup-kcp", infrastructurev1beta1.OscSecurityGroupRule{
					Flow: "Inbound", IpProtocol: "udp", FromPortRange: 32, ToPortRange: 32, IpRanges: []string{"1.2.3.4/32", "1.2.3.5/32"},
				}),
			},
			mockFuncs: []mockFunc{
				mockNetFound("vpc-24ba90ce"),
				mockSubnetFound("subnet-c1a282b0"),
				mockSubnetFound("subnet-1555ea91"),
				mockSubnetFound("subnet-174f5ec4"),
				mockInternetServiceFound("vpc-24ba90ce", "igw-c3c49899"),

				mockGetSecurityGroup("sg-750ae810", &osc.SecurityGroup{
					InboundRules: &[]osc.SecurityGroupRule{
						{IpProtocol: ptr.To("tcp"), FromPortRange: ptr.To[int32](179), ToPortRange: ptr.To[int32](179), IpRanges: &[]string{"10.0.0.0/16"}},
						{IpProtocol: ptr.To("tcp"), FromPortRange: ptr.To[int32](10250), ToPortRange: ptr.To[int32](10250), IpRanges: &[]string{"10.0.3.0/16"}},
						{IpProtocol: ptr.To("tcp"), FromPortRange: ptr.To[int32](6443), ToPortRange: ptr.To[int32](6443), IpRanges: &[]string{"10.0.3.0/16"}},
						{IpProtocol: ptr.To("tcp"), FromPortRange: ptr.To[int32](30000), ToPortRange: ptr.To[int32](32767), IpRanges: &[]string{"10.0.3.0/16"}},
						{IpProtocol: ptr.To("tcp"), FromPortRange: ptr.To[int32](2378), ToPortRange: ptr.To[int32](2379), IpRanges: &[]string{"10.0.4.0/16"}},
						{IpProtocol: ptr.To("tcp"), FromPortRange: ptr.To[int32](10250), ToPortRange: ptr.To[int32](10252), IpRanges: &[]string{"10.0.4.0/16"}},
					},
				}),
				mockCreateSecurityGroupRule("sg-750ae810", "Inbound", "udp", "1.2.3.4/32", 32, 32),
				mockCreateSecurityGroupRule("sg-750ae810", "Inbound", "udp", "1.2.3.5/32", 32, 32),
				mockGetSecurityGroup("sg-a093d014", &osc.SecurityGroup{
					InboundRules: &[]osc.SecurityGroupRule{
						{IpProtocol: ptr.To("tcp"), FromPortRange: ptr.To[int32](179), ToPortRange: ptr.To[int32](179), IpRanges: &[]string{"10.0.0.0/16"}},
						{IpProtocol: ptr.To("tcp"), FromPortRange: ptr.To[int32](10250), ToPortRange: ptr.To[int32](10250), IpRanges: &[]string{"10.0.3.0/16"}},
						{IpProtocol: ptr.To("tcp"), FromPortRange: ptr.To[int32](30000), ToPortRange: ptr.To[int32](32767), IpRanges: &[]string{"10.0.4.0/16"}},
					},
				}),
				mockGetSecurityGroup("sg-7eb16ccb", &osc.SecurityGroup{
					InboundRules: &[]osc.SecurityGroupRule{
						{IpProtocol: ptr.To("tcp"), FromPortRange: ptr.To[int32](6443), ToPortRange: ptr.To[int32](6443), IpRanges: &[]string{"0.0.0.0/0"}},
					},
				}),
				mockGetSecurityGroup("sg-0cd1f87e", &osc.SecurityGroup{
					InboundRules: &[]osc.SecurityGroupRule{
						{IpProtocol: ptr.To("udp"), FromPortRange: ptr.To[int32](4789), ToPortRange: ptr.To[int32](4789), IpRanges: &[]string{"10.0.0.0/16"}},
						{IpProtocol: ptr.To("udp"), FromPortRange: ptr.To[int32](5473), ToPortRange: ptr.To[int32](5473), IpRanges: &[]string{"10.0.0.0/16"}},
						{IpProtocol: ptr.To("udp"), FromPortRange: ptr.To[int32](51820), ToPortRange: ptr.To[int32](51820), IpRanges: &[]string{"10.0.0.0/16"}},
						{IpProtocol: ptr.To("udp"), FromPortRange: ptr.To[int32](51821), ToPortRange: ptr.To[int32](51821), IpRanges: &[]string{"10.0.0.0/16"}},
						{IpProtocol: ptr.To("udp"), FromPortRange: ptr.To[int32](8285), ToPortRange: ptr.To[int32](8285), IpRanges: &[]string{"10.0.0.0/16"}},
						{IpProtocol: ptr.To("udp"), FromPortRange: ptr.To[int32](8472), ToPortRange: ptr.To[int32](8472), IpRanges: &[]string{"10.0.0.0/16"}},
					},
				}),

				mockGetRouteTablesFromNet("vpc-24ba90ce", []osc.RouteTable{
					{
						RouteTableId:    ptr.To("rtb-0a4640a6"),
						LinkRouteTables: &[]osc.LinkRouteTable{{LinkRouteTableId: ptr.To("rtbassoc-643430b3"), SubnetId: ptr.To("subnet-1555ea91")}},
						Routes:          &[]osc.Route{{DestinationIpRange: ptr.To("0.0.0.0/0")}},
					},
					{
						RouteTableId:    ptr.To("rtb-194c971e"),
						LinkRouteTables: &[]osc.LinkRouteTable{{LinkRouteTableId: ptr.To("rtbassoc-09475c37"), SubnetId: ptr.To("subnet-c1a282b0")}},
						Routes:          &[]osc.Route{{DestinationIpRange: ptr.To("0.0.0.0/0")}},
					},
					{
						RouteTableId:    ptr.To("rtb-eeacfe8a"),
						LinkRouteTables: &[]osc.LinkRouteTable{{LinkRouteTableId: ptr.To("rtbassoc-90bda9c8"), SubnetId: ptr.To("subnet-174f5ec4")}},
						Routes:          &[]osc.Route{{DestinationIpRange: ptr.To("0.0.0.0/0")}},
					},
				}),
				mockGetNatServiceFromClientToken("test-cluster-api-natservice-9e1db9c4-bf0a-4583-8999-203ec002c520", nil),
				mockReadTagByNameFound(tag.NatResourceType, "test-cluster-api-natservice-9e1db9c4-bf0a-4583-8999-203ec002c520", "nat-223a4dd4"),
				mockNatServiceFound("nat-223a4dd4"),
				mockGetRouteTablesFromNet("vpc-24ba90ce", []osc.RouteTable{
					{
						RouteTableId:    ptr.To("rtb-0a4640a6"),
						LinkRouteTables: &[]osc.LinkRouteTable{{LinkRouteTableId: ptr.To("rtbassoc-643430b3"), SubnetId: ptr.To("subnet-1555ea91")}},
						Routes:          &[]osc.Route{{DestinationIpRange: ptr.To("0.0.0.0/0")}},
					},
					{
						RouteTableId:    ptr.To("rtb-194c971e"),
						LinkRouteTables: &[]osc.LinkRouteTable{{LinkRouteTableId: ptr.To("rtbassoc-09475c37"), SubnetId: ptr.To("subnet-c1a282b0")}},
						Routes:          &[]osc.Route{{DestinationIpRange: ptr.To("0.0.0.0/0")}},
					},
					{
						RouteTableId:    ptr.To("rtb-eeacfe8a"),
						LinkRouteTables: &[]osc.LinkRouteTable{{LinkRouteTableId: ptr.To("rtbassoc-90bda9c8"), SubnetId: ptr.To("subnet-174f5ec4")}},
						Routes:          &[]osc.Route{{DestinationIpRange: ptr.To("0.0.0.0/0")}},
					},
				}),

				mockLoadBalancerFound("test-cluster-api-k8s", "test-cluster-api-k8s-9e1db9c4-bf0a-4583-8999-203ec002c520"),
			},
		},
		{
			name:           "A v0.4 cluster has been moved by clusterctl move, status is updated",
			clusterSpec:    "ready-0.4",
			clusterPatches: []patchOSCClusterFunc{patchMoveCluster()},
			mockFuncs: []mockFunc{
				mockNetFound("vpc-24ba90ce"),
				mockSubnetFound("subnet-c1a282b0"),
				mockSubnetFound("subnet-1555ea91"),
				mockSubnetFound("subnet-174f5ec4"),
				mockInternetServiceFound("vpc-24ba90ce", "igw-c3c49899"),

				mockGetSecurityGroup("sg-750ae810", &osc.SecurityGroup{
					InboundRules: &[]osc.SecurityGroupRule{
						{IpProtocol: ptr.To("tcp"), FromPortRange: ptr.To[int32](179), ToPortRange: ptr.To[int32](179), IpRanges: &[]string{"10.0.0.0/16"}},
						{IpProtocol: ptr.To("tcp"), FromPortRange: ptr.To[int32](10250), ToPortRange: ptr.To[int32](10250), IpRanges: &[]string{"10.0.3.0/16"}},
						{IpProtocol: ptr.To("tcp"), FromPortRange: ptr.To[int32](6443), ToPortRange: ptr.To[int32](6443), IpRanges: &[]string{"10.0.3.0/16"}},
						{IpProtocol: ptr.To("tcp"), FromPortRange: ptr.To[int32](30000), ToPortRange: ptr.To[int32](32767), IpRanges: &[]string{"10.0.3.0/16"}},
						{IpProtocol: ptr.To("tcp"), FromPortRange: ptr.To[int32](2378), ToPortRange: ptr.To[int32](2379), IpRanges: &[]string{"10.0.4.0/16"}},
						{IpProtocol: ptr.To("tcp"), FromPortRange: ptr.To[int32](10250), ToPortRange: ptr.To[int32](10252), IpRanges: &[]string{"10.0.4.0/16"}},
					},
				}),
				mockGetSecurityGroup("sg-a093d014", &osc.SecurityGroup{
					InboundRules: &[]osc.SecurityGroupRule{
						{IpProtocol: ptr.To("tcp"), FromPortRange: ptr.To[int32](179), ToPortRange: ptr.To[int32](179), IpRanges: &[]string{"10.0.0.0/16"}},
						{IpProtocol: ptr.To("tcp"), FromPortRange: ptr.To[int32](10250), ToPortRange: ptr.To[int32](10250), IpRanges: &[]string{"10.0.3.0/16"}},
						{IpProtocol: ptr.To("tcp"), FromPortRange: ptr.To[int32](30000), ToPortRange: ptr.To[int32](32767), IpRanges: &[]string{"10.0.4.0/16"}},
					},
				}),
				mockGetSecurityGroup("sg-7eb16ccb", &osc.SecurityGroup{
					InboundRules: &[]osc.SecurityGroupRule{
						{IpProtocol: ptr.To("tcp"), FromPortRange: ptr.To[int32](6443), ToPortRange: ptr.To[int32](6443), IpRanges: &[]string{"0.0.0.0/0"}},
					},
				}),
				mockGetSecurityGroup("sg-0cd1f87e", &osc.SecurityGroup{
					InboundRules: &[]osc.SecurityGroupRule{
						{IpProtocol: ptr.To("udp"), FromPortRange: ptr.To[int32](4789), ToPortRange: ptr.To[int32](4789), IpRanges: &[]string{"10.0.0.0/16"}},
						{IpProtocol: ptr.To("udp"), FromPortRange: ptr.To[int32](5473), ToPortRange: ptr.To[int32](5473), IpRanges: &[]string{"10.0.0.0/16"}},
						{IpProtocol: ptr.To("udp"), FromPortRange: ptr.To[int32](51820), ToPortRange: ptr.To[int32](51820), IpRanges: &[]string{"10.0.0.0/16"}},
						{IpProtocol: ptr.To("udp"), FromPortRange: ptr.To[int32](51821), ToPortRange: ptr.To[int32](51821), IpRanges: &[]string{"10.0.0.0/16"}},
						{IpProtocol: ptr.To("udp"), FromPortRange: ptr.To[int32](8285), ToPortRange: ptr.To[int32](8285), IpRanges: &[]string{"10.0.0.0/16"}},
						{IpProtocol: ptr.To("udp"), FromPortRange: ptr.To[int32](8472), ToPortRange: ptr.To[int32](8472), IpRanges: &[]string{"10.0.0.0/16"}},
					},
				}),

				mockGetRouteTablesFromNet("vpc-24ba90ce", []osc.RouteTable{
					{
						RouteTableId:    ptr.To("rtb-0a4640a6"),
						LinkRouteTables: &[]osc.LinkRouteTable{{LinkRouteTableId: ptr.To("rtbassoc-643430b3"), SubnetId: ptr.To("subnet-1555ea91")}},
						Routes:          &[]osc.Route{{DestinationIpRange: ptr.To("0.0.0.0/0")}},
					},
					{
						RouteTableId:    ptr.To("rtb-194c971e"),
						LinkRouteTables: &[]osc.LinkRouteTable{{LinkRouteTableId: ptr.To("rtbassoc-09475c37"), SubnetId: ptr.To("subnet-c1a282b0")}},
						Routes:          &[]osc.Route{{DestinationIpRange: ptr.To("0.0.0.0/0")}},
					},
					{
						RouteTableId:    ptr.To("rtb-eeacfe8a"),
						LinkRouteTables: &[]osc.LinkRouteTable{{LinkRouteTableId: ptr.To("rtbassoc-90bda9c8"), SubnetId: ptr.To("subnet-174f5ec4")}},
						Routes:          &[]osc.Route{{DestinationIpRange: ptr.To("0.0.0.0/0")}},
					},
				}),
				mockGetNatServiceFromClientToken("test-cluster-api-natservice-9e1db9c4-bf0a-4583-8999-203ec002c520", nil),
				mockReadTagByNameFound(tag.NatResourceType, "test-cluster-api-natservice-9e1db9c4-bf0a-4583-8999-203ec002c520", "nat-223a4dd4"),
				mockNatServiceFound("nat-223a4dd4"),
				mockGetRouteTablesFromNet("vpc-24ba90ce", []osc.RouteTable{
					{
						RouteTableId:    ptr.To("rtb-0a4640a6"),
						LinkRouteTables: &[]osc.LinkRouteTable{{LinkRouteTableId: ptr.To("rtbassoc-643430b3"), SubnetId: ptr.To("subnet-1555ea91")}},
						Routes:          &[]osc.Route{{DestinationIpRange: ptr.To("0.0.0.0/0")}},
					},
					{
						RouteTableId:    ptr.To("rtb-194c971e"),
						LinkRouteTables: &[]osc.LinkRouteTable{{LinkRouteTableId: ptr.To("rtbassoc-09475c37"), SubnetId: ptr.To("subnet-c1a282b0")}},
						Routes:          &[]osc.Route{{DestinationIpRange: ptr.To("0.0.0.0/0")}},
					},
					{
						RouteTableId:    ptr.To("rtb-eeacfe8a"),
						LinkRouteTables: &[]osc.LinkRouteTable{{LinkRouteTableId: ptr.To("rtbassoc-90bda9c8"), SubnetId: ptr.To("subnet-174f5ec4")}},
						Routes:          &[]osc.Route{{DestinationIpRange: ptr.To("0.0.0.0/0")}},
					},
				}),

				mockLoadBalancerFound("test-cluster-api-k8s", "test-cluster-api-k8s-9e1db9c4-bf0a-4583-8999-203ec002c520"),
			},
			clusterAsserts: []assertOSCClusterFunc{
				// All other resources have a ResourceId field, no need to store a ref in status.
				assertStatusResources(infrastructurev1beta1.OscClusterResources{
					InternetService: map[string]string{
						"default": "igw-c3c49899",
					},
					NatService: map[string]string{
						"test-cluster-api-natservice-9e1db9c4-bf0a-4583-8999-203ec002c520": "nat-223a4dd4",
					},
				}),
			},
			next: &testcase{
				name: "A second run has all references in cache",
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
			name:           "Deleting a v0.4 cluster",
			clusterSpec:    "ready-0.4",
			clusterPatches: []patchOSCClusterFunc{patchDeleteCluster()},
			mockFuncs: []mockFunc{
				mockLoadBalancerFound("test-cluster-api-k8s", "test-cluster-api-k8s-9e1db9c4-bf0a-4583-8999-203ec002c520"),
				mockDeleteLoadBalancer("test-cluster-api-k8s"),

				mockListNatServices("vpc-24ba90ce", "nat-223a4dd4"),
				mockDeleteNatService("nat-223a4dd4"),

				mockGetRouteTablesFromNet("vpc-24ba90ce", []osc.RouteTable{
					{RouteTableId: ptr.To("rtb-0a4640a6"), LinkRouteTables: &[]osc.LinkRouteTable{{LinkRouteTableId: ptr.To("rtbassoc-643430b3"), SubnetId: ptr.To("subnet-1555ea91")}}},
					{RouteTableId: ptr.To("rtb-194c971e"), LinkRouteTables: &[]osc.LinkRouteTable{{LinkRouteTableId: ptr.To("rtbassoc-09475c37"), SubnetId: ptr.To("subnet-c1a282b0")}}},
					{RouteTableId: ptr.To("rtb-eeacfe8a"), LinkRouteTables: &[]osc.LinkRouteTable{{LinkRouteTableId: ptr.To("rtbassoc-90bda9c8"), SubnetId: ptr.To("subnet-174f5ec4")}}},
				}),
				mockUnlinkRouteTable("rtbassoc-643430b3"),
				mockDeleteRouteTable("rtb-0a4640a6"),
				mockUnlinkRouteTable("rtbassoc-09475c37"),
				mockDeleteRouteTable("rtb-194c971e"),
				mockUnlinkRouteTable("rtbassoc-90bda9c8"),
				mockDeleteRouteTable("rtb-eeacfe8a"),

				mockGetSecurityGroupsFromNet("vpc-24ba90ce", []osc.SecurityGroup{
					{
						SecurityGroupId: ptr.To("sg-a093d014"), InboundRules: &[]osc.SecurityGroupRule{{}, {}}, OutboundRules: &[]osc.SecurityGroupRule{{}},
					},
					{
						SecurityGroupId: ptr.To("sg-750ae810"), InboundRules: &[]osc.SecurityGroupRule{{}}, OutboundRules: &[]osc.SecurityGroupRule{{}},
					},
				}),
				mockDeleteSecurityGroup("sg-a093d014", nil),
				mockDeleteSecurityGroup("sg-750ae810", nil),

				mockInternetServiceFound("vpc-24ba90ce", "igw-c3c49899"),
				mockUnlinkInternetService("igw-c3c49899", "vpc-24ba90ce"),
				mockDeleteInternetService("igw-c3c49899"),

				mockSubnetFound("subnet-c1a282b0"),
				mockDeleteSubnet("subnet-c1a282b0"),
				mockSubnetFound("subnet-1555ea91"),
				mockDeleteSubnet("subnet-1555ea91"),
				mockSubnetFound("subnet-174f5ec4"),
				mockDeleteSubnet("subnet-174f5ec4"),
				mockNetFound("vpc-24ba90ce"),
				mockDeleteNet("vpc-24ba90ce"),
			},
			assertDeleted: true,
		},
		{
			name:           "Cluster is deleted even if no resource has been created",
			clusterSpec:    "base-0.4",
			clusterPatches: []patchOSCClusterFunc{patchDeleteCluster()},
			mockFuncs: []mockFunc{
				mockGetLoadBalancer("test-cluster-api-k8s", nil),
				mockReadOwnedByTagNoneFound(tag.NetResourceType, "9e1db9c4-bf0a-4583-8999-203ec002c520"),
				mockReadTagByNameNoneFound(tag.NetResourceType, "test-cluster-api-net-9e1db9c4-bf0a-4583-8999-203ec002c520"),
			},
			assertDeleted: true,
		},
		{
			name:           "If LB is already deleted, continue with the rest",
			clusterSpec:    "ready-0.4",
			clusterPatches: []patchOSCClusterFunc{patchDeleteCluster()},
			mockFuncs: []mockFunc{
				mockGetLoadBalancer("test-cluster-api-k8s", nil),

				mockListNatServices("vpc-24ba90ce", "nat-223a4dd4"),
				mockDeleteNatService("nat-223a4dd4"),

				mockGetRouteTablesFromNet("vpc-24ba90ce", []osc.RouteTable{
					{RouteTableId: ptr.To("rtb-0a4640a6"), LinkRouteTables: &[]osc.LinkRouteTable{{LinkRouteTableId: ptr.To("rtbassoc-643430b3")}}},
					{RouteTableId: ptr.To("rtb-194c971e"), LinkRouteTables: &[]osc.LinkRouteTable{{LinkRouteTableId: ptr.To("rtbassoc-09475c37")}}},
				}),
				mockUnlinkRouteTable("rtbassoc-643430b3"),
				mockDeleteRouteTable("rtb-0a4640a6"),
				mockUnlinkRouteTable("rtbassoc-09475c37"),
				mockDeleteRouteTable("rtb-194c971e"),

				mockGetSecurityGroupsFromNet("vpc-24ba90ce", []osc.SecurityGroup{
					{
						SecurityGroupId: ptr.To("sg-a093d014"), InboundRules: &[]osc.SecurityGroupRule{{}, {}}, OutboundRules: &[]osc.SecurityGroupRule{{}},
					},
					{
						SecurityGroupId: ptr.To("sg-750ae810"), InboundRules: &[]osc.SecurityGroupRule{{}}, OutboundRules: &[]osc.SecurityGroupRule{{}},
					},
				}),
				mockDeleteSecurityGroup("sg-a093d014", nil),
				mockDeleteSecurityGroup("sg-750ae810", nil),

				mockInternetServiceFound("vpc-24ba90ce", "igw-c3c49899"),
				mockUnlinkInternetService("igw-c3c49899", "vpc-24ba90ce"),
				mockDeleteInternetService("igw-c3c49899"),

				mockSubnetFound("subnet-c1a282b0"),
				mockDeleteSubnet("subnet-c1a282b0"),
				mockSubnetFound("subnet-1555ea91"),
				mockDeleteSubnet("subnet-1555ea91"),
				mockSubnetFound("subnet-174f5ec4"),
				mockDeleteSubnet("subnet-174f5ec4"),
				mockNetFound("vpc-24ba90ce"),
				mockDeleteNet("vpc-24ba90ce"),
			},
			assertDeleted: true,
		},
		{
			name:           "Delete securityGroupRules with securityGroups before deleting securityGroups",
			clusterSpec:    "ready-0.4",
			clusterPatches: []patchOSCClusterFunc{patchDeleteCluster()},
			mockFuncs: []mockFunc{
				mockLoadBalancerFound("test-cluster-api-k8s", "test-cluster-api-k8s-9e1db9c4-bf0a-4583-8999-203ec002c520"),
				mockDeleteLoadBalancer("test-cluster-api-k8s"),

				mockListNatServices("vpc-24ba90ce", "nat-223a4dd4"),
				mockDeleteNatService("nat-223a4dd4"),

				mockGetRouteTablesFromNet("vpc-24ba90ce", []osc.RouteTable{
					{RouteTableId: ptr.To("rtb-0a4640a6"), LinkRouteTables: &[]osc.LinkRouteTable{{LinkRouteTableId: ptr.To("rtbassoc-643430b3")}}},
					{RouteTableId: ptr.To("rtb-194c971e"), LinkRouteTables: &[]osc.LinkRouteTable{{LinkRouteTableId: ptr.To("rtbassoc-09475c37")}}},
				}),
				mockUnlinkRouteTable("rtbassoc-643430b3"),
				mockDeleteRouteTable("rtb-0a4640a6"),
				mockUnlinkRouteTable("rtbassoc-09475c37"),
				mockDeleteRouteTable("rtb-194c971e"),

				mockGetSecurityGroupsFromNet("vpc-24ba90ce", []osc.SecurityGroup{
					{
						SecurityGroupId: ptr.To("sg-a093d014"), InboundRules: &[]osc.SecurityGroupRule{{
							FromPortRange: ptr.To(int32(33)), ToPortRange: ptr.To(int32(34)), IpProtocol: ptr.To("tcp"), SecurityGroupsMembers: &[]osc.SecurityGroupsMember{{SecurityGroupId: ptr.To("sg-foo")}, {SecurityGroupId: ptr.To("sg-bar")}},
						}}, OutboundRules: &[]osc.SecurityGroupRule{{
							FromPortRange: ptr.To(int32(35)), ToPortRange: ptr.To(int32(36)), IpProtocol: ptr.To("tcp"), SecurityGroupsMembers: &[]osc.SecurityGroupsMember{{SecurityGroupId: ptr.To("sg-foo")}, {SecurityGroupId: ptr.To("sg-bar")}},
						}},
					},
				}),
				mockDeleteSecurityGroupRule("sg-a093d014", "Inbound", "tcp", "", "sg-foo", int32(33), int32(34)),
				mockDeleteSecurityGroupRule("sg-a093d014", "Inbound", "tcp", "", "sg-bar", int32(33), int32(34)),
				mockDeleteSecurityGroupRule("sg-a093d014", "Outbound", "tcp", "", "sg-foo", int32(35), int32(36)),
				mockDeleteSecurityGroupRule("sg-a093d014", "Outbound", "tcp", "", "sg-bar", int32(35), int32(36)),
				mockDeleteSecurityGroup("sg-a093d014", nil),

				mockInternetServiceFound("vpc-24ba90ce", "igw-c3c49899"),
				mockUnlinkInternetService("igw-c3c49899", "vpc-24ba90ce"),
				mockDeleteInternetService("igw-c3c49899"),

				mockSubnetFound("subnet-c1a282b0"),
				mockDeleteSubnet("subnet-c1a282b0"),
				mockSubnetFound("subnet-1555ea91"),
				mockDeleteSubnet("subnet-1555ea91"),
				mockSubnetFound("subnet-174f5ec4"),
				mockDeleteSubnet("subnet-174f5ec4"),
				mockNetFound("vpc-24ba90ce"),
				mockDeleteNet("vpc-24ba90ce"),
			},
			assertDeleted: true,
		},
		{
			name:           "Deleting a cluster based on an existing network",
			clusterSpec:    "ready-0.4",
			clusterPatches: []patchOSCClusterFunc{patchDeleteCluster(), patchUseExistingNet()},
			mockFuncs: []mockFunc{
				mockLoadBalancerFound("test-cluster-api-k8s", "test-cluster-api-k8s-9e1db9c4-bf0a-4583-8999-203ec002c520"),
				mockDeleteLoadBalancer("test-cluster-api-k8s"),
			},
			assertDeleted: true,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			runClusterTest(t, tc)
		})
	}
}
