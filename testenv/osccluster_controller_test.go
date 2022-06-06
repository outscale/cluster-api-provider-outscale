package test

import (
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/net"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/security"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/service"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/controllers"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func deploySecret(ctx context.Context, name string, namespace string) (client.Object, client.ObjectKey) {
	By("Deploy Secret")
	access_key := os.Getenv("OSC_ACCESS_KEY")
	secret_key := os.Getenv("OSC_SECRET_KEY")

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			access_key: []byte(access_key),
			secret_key: []byte(secret_key),
		},
	}
	Expect(k8sClient.Create(ctx, secret)).To(Succeed())
	secretKey := client.ObjectKey{Namespace: namespace, Name: name}
	return secret, secretKey
}

func deployOscInfraCluster(ctx context.Context, infraClusterSpec infrastructurev1beta1.OscClusterSpec, name string, namespace string) (client.Object, client.ObjectKey) {
	By("Deploy oscInfraCluster")
	oscInfraCluster := &infrastructurev1beta1.OscCluster{
		Spec: infraClusterSpec,
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	Expect(k8sClient.Create(ctx, oscInfraCluster)).To(Succeed())
	oscInfraClusterKey := client.ObjectKey{Namespace: namespace, Name: name}
	return oscInfraCluster, oscInfraClusterKey
}

func createCheckDeleteOscCluster(ctx context.Context, infraClusterSpec infrastructurev1beta1.OscClusterSpec) {
	secret, secretKey := deploySecret(ctx, "cluster-api-provider-outscale", "cluster-api-provider-outscale-system")
	oscInfraCluster, oscInfraClusterKey := deployOscInfraCluster(ctx, infraClusterSpec, "cluster-api", "default")
	capoCluster, capoClusterKey := deployCapoCluster(ctx, "cluster-api", "default")
	waitOscInfraClusterToBeReady(ctx, oscInfraClusterKey)
	waitOscClusterToProvision(ctx, capoClusterKey)
	clusterScope, err := getClusterScope(ctx, capoClusterKey, oscInfraClusterKey)
	Expect(err).ShouldNot(HaveOccurred())
	checkOscNetToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscSubnetToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscInternetServiceToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscNatServiceToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscPublicIpToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscRouteTableToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscRouteToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscSecurityGroupToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscSecurityGroupRuleToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscLoadBalancerToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	By("Delete cluster")
	deleteObj(ctx, secret, secretKey, "secret", "cluster-api-provider-outscale")
	deleteObj(ctx, oscInfraCluster, oscInfraClusterKey, "oscInfraCluster", "cluster-api")
	deleteObj(ctx, capoCluster, capoClusterKey, "capoCluster", "cluster-api")
}

func deleteObj(ctx context.Context, obj client.Object, key client.ObjectKey, kind string, name string) {
	Expect(k8sClient.Delete(ctx, obj)).To(Succeed())
	EventuallyWithOffset(1, func() error {
		fmt.Fprintf(GinkgoWriter, "Wait %s %s to be delete\n", kind, name)
		return k8sClient.Get(ctx, key, obj)
	}, 2*time.Minute, 3*time.Second).ShouldNot(Succeed())
}

func deployCapoCluster(ctx context.Context, name string, namespace string) (client.Object, client.ObjectKey) {
	By("Deploy capoCluster")
	capoCluster := &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: clusterv1.ClusterSpec{
			InfrastructureRef: &corev1.ObjectReference{
				APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
				Kind:       "OscCluster",
				Name:       name,
				Namespace:  namespace,
			},
		},
	}
	Expect(k8sClient.Create(ctx, capoCluster)).To(Succeed())
	capoClusterKey := client.ObjectKey{Namespace: namespace, Name: name}
	return capoCluster, capoClusterKey
}

func waitOscClusterToProvision(ctx context.Context, capoClusterKey client.ObjectKey) {
	By("Wait capoCluster to be in provisioned phase")
	Eventually(func() (string, error) {
		capoCluster := &clusterv1.Cluster{}
		k8sClient.Get(ctx, capoClusterKey, capoCluster)
		fmt.Fprintf(GinkgoWriter, "capoClusterPhase: %v\n", capoCluster.Status.Phase)
		return capoCluster.Status.Phase, nil
	}, 2*time.Minute, 3*time.Second).Should(Equal("Provisioned"))
}

func waitOscInfraClusterToBeReady(ctx context.Context, oscInfraClusterKey client.ObjectKey) {
	By("Wait OscInfraCluster to be in ready status")
	EventuallyWithOffset(1, func() bool {
		oscInfraCluster := &infrastructurev1beta1.OscCluster{}
		k8sClient.Get(ctx, oscInfraClusterKey, oscInfraCluster)
		fmt.Fprintf(GinkgoWriter, "oscInfraClusterReady: %v\n", oscInfraCluster.Status.Ready)
		return oscInfraCluster.Status.Ready
	}, 2*time.Minute, 3*time.Second).Should(BeTrue())
}

func checkOscNetToBeProvisioned(ctx context.Context, oscInfraClusterKey client.ObjectKey, clusterScope *scope.ClusterScope) {
	By("Check OscNet is provisioned")
	Eventually(func() error {
		netsvc := net.NewService(ctx, clusterScope)
		netSpec := clusterScope.GetNet()
		netId := netSpec.ResourceId
		fmt.Fprintf(GinkgoWriter, "Check NetId %s\n", netId)
		net, err := netsvc.GetNet(netId)
		if err != nil {
			return err
		}
		fmt.Fprintf(GinkgoWriter, "Check NetId received %s\n", net.GetNetId())
		if netId != net.GetNetId() {
			return fmt.Errorf("Net %s does not exist", netId)
		}
		fmt.Fprintf(GinkgoWriter, "Found OscNet \n")
		return nil
	}, 1*time.Minute, 1*time.Second).Should(BeNil())
}

func checkOscSubnetToBeProvisioned(ctx context.Context, oscInfraClusterKey client.ObjectKey, clusterScope *scope.ClusterScope) {
	By("Check OscSubnet is provisioned")
	Eventually(func() error {
		netsvc := net.NewService(ctx, clusterScope)
		netSpec := clusterScope.GetNet()
		subnetsSpec := clusterScope.GetSubnet()
		netId := netSpec.ResourceId
		var subnetIds []string
		subnetIds, err := netsvc.GetSubnetIdsFromNetIds(netId)
		if err != nil {
			return err
		}
		fmt.Fprintf(GinkgoWriter, "Check SubnetIds received %v \n", subnetIds)
		for _, subnetSpec := range subnetsSpec {
			subnetId := subnetSpec.ResourceId

			fmt.Fprintf(GinkgoWriter, "Check SubnetId %s\n", subnetId)
			if !controllers.Contains(subnetIds, subnetId) {
				return fmt.Errorf("Subnet %s does not exist", subnetId)
			}
		}
		fmt.Fprintf(GinkgoWriter, "Found OscSubnet \n")
		return nil
	}, 1*time.Minute, 1*time.Second).Should(BeNil())
}

func checkOscInternetServiceToBeProvisioned(ctx context.Context, oscInfraClusterKey client.ObjectKey, clusterScope *scope.ClusterScope) {
	By("Check OscInternetService is provisioned")
	Eventually(func() error {
		netsvc := net.NewService(ctx, clusterScope)
		internetServiceSpec := clusterScope.GetInternetService()
		internetServiceId := internetServiceSpec.ResourceId
		fmt.Fprintf(GinkgoWriter, "Check InternetServiceId %s\n", internetServiceId)
		internetService, err := netsvc.GetInternetService(internetServiceId)
		if err != nil {
			return err
		}
		fmt.Fprintf(GinkgoWriter, "Check InternetServiceId received %s\n", internetService.GetInternetServiceId())
		if internetServiceId != internetService.GetInternetServiceId() {
			return fmt.Errorf("InternetService %s does not exist", internetServiceId)
		}
		fmt.Fprintf(GinkgoWriter, "Found OscInternetService \n")
		return nil
	}, 1*time.Minute, 1*time.Second).Should(BeNil())
}

func checkOscNatServiceToBeProvisioned(ctx context.Context, oscInfraClusterKey client.ObjectKey, clusterScope *scope.ClusterScope) {
	By("Check OscNatService is provisioned")
	Eventually(func() error {
		netsvc := net.NewService(ctx, clusterScope)
		natServiceSpec := clusterScope.GetNatService()
		natServiceId := natServiceSpec.ResourceId
		fmt.Fprintf(GinkgoWriter, "Check NatServiceId %s\n", natServiceId)
		natService, err := netsvc.GetNatService(natServiceId)
		if err != nil {
			return err
		}
		fmt.Fprintf(GinkgoWriter, "Check NatServiceId received %s\n", natService.GetNatServiceId())
		if natServiceId != natService.GetNatServiceId() {
			return fmt.Errorf("NatService %s does not exist", natServiceId)
		}
		fmt.Fprintf(GinkgoWriter, "Found OscNatService \n")
		return nil
	}, 1*time.Minute, 1*time.Second).Should(BeNil())
}

func checkOscPublicIpToBeProvisioned(ctx context.Context, oscInfraClusterKey client.ObjectKey, clusterScope *scope.ClusterScope) {
	By("Check OscPublicIp is provisioned")
	Eventually(func() error {
		securitysvc := security.NewService(ctx, clusterScope)
		var publicIpsSpec []*infrastructurev1beta1.OscPublicIp
		publicIpsSpec = clusterScope.GetPublicIp()
		var publicIpId string
		var publicIpIds []string
		for _, publicIpSpec := range publicIpsSpec {
			publicIpId = publicIpSpec.ResourceId
			publicIpIds = append(publicIpIds, publicIpId)
		}
		validPublicIpIds, err := securitysvc.ValidatePublicIpIds(publicIpIds)
		fmt.Fprintf(GinkgoWriter, "Check PublicIpIds received %s\n", validPublicIpIds)
		if err != nil {
			return err
		}
		for _, publicIpSpec := range publicIpsSpec {
			publicIpId = publicIpSpec.ResourceId
			fmt.Fprintf(GinkgoWriter, "Check PublicIpId %s\n", publicIpId)
		}
		if !controllers.Contains(validPublicIpIds, publicIpId) {
			return fmt.Errorf("PublicIpId %s does not exist", publicIpId)
		}
		fmt.Fprintf(GinkgoWriter, "Found OscPublicIp \n")
		return nil
	}, 1*time.Minute, 1*time.Second).Should(BeNil())
}

func checkOscRouteTableToBeProvisioned(ctx context.Context, oscInfraClusterKey client.ObjectKey, clusterScope *scope.ClusterScope) {
	By("Check OscRouteTable is provisioned")
	Eventually(func() error {
		netSpec := clusterScope.GetNet()
		netId := netSpec.ResourceId
		securitysvc := security.NewService(ctx, clusterScope)
		routeTablesSpec := clusterScope.GetRouteTables()
		routeTableIds, err := securitysvc.GetRouteTableIdsFromNetIds(netId)
		fmt.Fprintf(GinkgoWriter, "Check RouteTableIds received %v \n", routeTableIds)

		if err != nil {
			return err
		}

		for _, routeTableSpec := range routeTablesSpec {
			routeTableId := routeTableSpec.ResourceId
			fmt.Fprintf(GinkgoWriter, "Check RouteTableId %s\n", routeTableId)
			if !controllers.Contains(routeTableIds, routeTableId) {
				return fmt.Errorf("RouteTableId %s does not exist", routeTableId)
			}
		}
		fmt.Fprintf(GinkgoWriter, "Found OscRouteTable \n")
		return nil
	}, 1*time.Minute, 1*time.Second).Should(BeNil())
}

func checkOscRouteToBeProvisioned(ctx context.Context, oscInfraClusterKey client.ObjectKey, clusterScope *scope.ClusterScope) {
	By("Check OscRoute is provisioned")
	Eventually(func() error {
		securitysvc := security.NewService(ctx, clusterScope)
		routeTablesSpec := clusterScope.GetRouteTables()
		natServiceSpec := clusterScope.GetNatService()
		natServiceId := natServiceSpec.ResourceId
		internetServiceSpec := clusterScope.GetInternetService()
		internetServiceId := internetServiceSpec.ResourceId
		var resourceId string
		for _, routeTableSpec := range routeTablesSpec {
			routeTableId := routeTableSpec.ResourceId
			fmt.Fprintf(GinkgoWriter, "Check RouteTableId %s\n", routeTableId)
			routesSpec := clusterScope.GetRoute(routeTableSpec.Name)
			for _, routeSpec := range *routesSpec {
				routeName := routeSpec.Name + clusterScope.GetUID()
				fmt.Fprintf(GinkgoWriter, "Check Route %s exist \n", routeName)
				resourceType := routeSpec.TargetType
				if resourceType == "gateway" {
					resourceId = internetServiceId
				} else {
					resourceId = natServiceId
				}
				fmt.Fprintf(GinkgoWriter, "Check RouteTable %s %s %s\n", routeTableId, resourceId, resourceType)

				routeTableFromRoute, err := securitysvc.GetRouteTableFromRoute(routeTableId, resourceId, resourceType)
				if err != nil {
					return err
				}
				fmt.Fprintf(GinkgoWriter, "Check RouteTableId received %s\n", routeTableFromRoute.GetRouteTableId())
				if routeTableId != routeTableFromRoute.GetRouteTableId() {
					return fmt.Errorf("Route %s does not exist", routeName)
				}
			}
		}
		fmt.Fprintf(GinkgoWriter, "Found OscRoute \n")
		return nil
	}, 1*time.Minute, 1*time.Second).Should(BeNil())
}

func checkOscSecurityGroupToBeProvisioned(ctx context.Context, oscInfraClusterKey client.ObjectKey, clusterScope *scope.ClusterScope) {
	By("Check OscSecurityGroup is provisioned")
	Eventually(func() error {
		netSpec := clusterScope.GetNet()
		netId := netSpec.ResourceId
		securitysvc := security.NewService(ctx, clusterScope)
		securityGroupsSpec := clusterScope.GetSecurityGroups()
		securityGroupIds, err := securitysvc.GetSecurityGroupIdsFromNetIds(netId)
		fmt.Fprintf(GinkgoWriter, "Check SecurityGroupIds received %v \n", securityGroupIds)
		if err != nil {
			return err
		}
		for _, securityGroupSpec := range securityGroupsSpec {
			securityGroupId := securityGroupSpec.ResourceId
			fmt.Fprintf(GinkgoWriter, "Check SecurityGroupId %s\n", securityGroupId)
			if !controllers.Contains(securityGroupIds, securityGroupId) {
				return fmt.Errorf("SecurityGroupId %s does not exist", securityGroupId)
			}
		}
		fmt.Fprintf(GinkgoWriter, "Found OscSecurityGroup \n")
		return nil

	}, 1*time.Minute, 1*time.Second).Should(BeNil())
}

func checkOscSecurityGroupRuleToBeProvisioned(ctx context.Context, oscInfraClusterKey client.ObjectKey, clusterScope *scope.ClusterScope) {
	By("Check OscSecurityGroupRule is provisioned")
	Eventually(func() error {
		securitysvc := security.NewService(ctx, clusterScope)
		securityGroupsSpec := clusterScope.GetSecurityGroups()
		for _, securityGroupSpec := range securityGroupsSpec {
			securityGroupId := securityGroupSpec.ResourceId
			fmt.Fprintf(GinkgoWriter, "Check SecurityGroupId %s\n", securityGroupId)
			securityGroupRulesSpec := clusterScope.GetSecurityGroupRule(securityGroupSpec.Name)
			for _, securityGroupRuleSpec := range *securityGroupRulesSpec {
				securityGroupRuleName := securityGroupRuleSpec.Name + "-" + clusterScope.GetUID()
				fmt.Fprintf(GinkgoWriter, "Check SecurityGroupRule %s exist \n", securityGroupRuleName)
				Flow := securityGroupRuleSpec.Flow
				IpProtocol := securityGroupRuleSpec.IpProtocol
				IpRange := securityGroupRuleSpec.IpRange
				FromPortRange := securityGroupRuleSpec.FromPortRange
				ToPortRange := securityGroupRuleSpec.ToPortRange
				securityGroupFromSecurityGroupRule, err := securitysvc.GetSecurityGroupFromSecurityGroupRule(securityGroupId, Flow, IpProtocol, IpRange, FromPortRange, ToPortRange)
				if err != nil {
					return err
				}
				fmt.Fprintf(GinkgoWriter, "Check SecurityGroupId received %s\n", securityGroupFromSecurityGroupRule.GetSecurityGroupId())
				if securityGroupId != securityGroupFromSecurityGroupRule.GetSecurityGroupId() {
					return fmt.Errorf("SecurityGroupRule %s does not exist", securityGroupRuleName)
				}

			}
		}
		fmt.Fprintf(GinkgoWriter, "Found OscSecurityGroupRule \n")
		return nil
	}, 1*time.Minute, 1*time.Second).Should(BeNil())
}

func checkOscLoadBalancerToBeProvisioned(ctx context.Context, oscInfraClusterKey client.ObjectKey, clusterScope *scope.ClusterScope) {
	By("Check OscLoadBalancer is provisioned")
	Eventually(func() error {
		servicesvc := service.NewService(ctx, clusterScope)
		loadBalancerSpec := clusterScope.GetLoadBalancer()
		loadbalancer, err := servicesvc.GetLoadBalancer(loadBalancerSpec)
		loadBalancerName := loadBalancerSpec.LoadBalancerName
		fmt.Fprintf(GinkgoWriter, "Check loadBalancer %s\n", loadBalancerName)
		if err != nil {
			return err
		}
		if loadbalancer == nil {
			return fmt.Errorf("LoadBalancer %s does not exist", loadBalancerName)
		}
		fmt.Fprintf(GinkgoWriter, "Check loadBalancer received %s\n", *loadbalancer.LoadBalancerName)
		fmt.Fprintf(GinkgoWriter, "Found OscLoadBalancer \n")
		return nil
	}, 1*time.Minute, 1*time.Second).Should(BeNil())
}

func getClusterScope(ctx context.Context, capoClusterKey client.ObjectKey, oscInfraClusterKey client.ObjectKey) (clusterScope *scope.ClusterScope, err error) {
	By("Get ClusterScope")
	capoCluster := &clusterv1.Cluster{}
	k8sClient.Get(ctx, capoClusterKey, capoCluster)
	oscInfraCluster := &infrastructurev1beta1.OscCluster{}
	k8sClient.Get(ctx, oscInfraClusterKey, oscInfraCluster)
	clusterScope, err = scope.NewClusterScope(scope.ClusterScopeParams{
		Client:     k8sClient,
		Cluster:    capoCluster,
		OscCluster: oscInfraCluster,
	})
	return clusterScope, err
}

var _ = Describe("Outscale Cluster Reconciler", func() {
	BeforeEach(func() {})
	AfterEach(func() {})
	Context("Reconcile an Outscale cluster", func() {
		It("should create a simple cluster", func() {
			ctx := context.Background()
			infraClusterSpec := infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "cluster-api-net",
						IpRange: "10.0.0.0/16",
					},
					Subnets: []*infrastructurev1beta1.OscSubnet{
						{
							Name:          "cluster-api-subnet",
							IpSubnetRange: "10.0.0.0/24",
						},
					},
					InternetService: infrastructurev1beta1.OscInternetService{
						Name: "cluster-api-internetservice",
					},
					NatService: infrastructurev1beta1.OscNatService{
						Name:         "cluster-api-natservice",
						PublicIpName: "cluster-api-publicip",
						SubnetName:   "cluster-api-subnet",
					},
					RouteTables: []*infrastructurev1beta1.OscRouteTable{
						{
							Name:       "cluster-api-routetable",
							SubnetName: "cluster-api-subnet",
							Routes: []infrastructurev1beta1.OscRoute{
								{
									Name:        "cluster-api-routes",
									TargetName:  "cluster-api-internetservice",
									TargetType:  "gateway",
									Destination: "0.0.0.0/0",
								},
							},
						},
					},
					PublicIps: []*infrastructurev1beta1.OscPublicIp{
						{
							Name: "cluster-api-publicip",
						},
					},
					SecurityGroups: []*infrastructurev1beta1.OscSecurityGroup{
						{
							Name:        "cluster-api-securitygroups",
							Description: "securitygroup",
							SecurityGroupRules: []infrastructurev1beta1.OscSecurityGroupRule{
								{
									Name:          "cluster-api-securitygrouprule",
									Flow:          "Inbound",
									IpProtocol:    "tcp",
									IpRange:       "0.0.0.0/0",
									FromPortRange: 6443,
									ToPortRange:   6443,
								},
							},
						},
					},
					LoadBalancer: infrastructurev1beta1.OscLoadBalancer{
						LoadBalancerName:  "OscSdkExample-8",
						LoadBalancerType:  "internet-facing",
						SubnetName:        "cluster-api-subnet",
						SecurityGroupName: "cluster-api-securitygroups",
					},
				},
			}
			createCheckDeleteOscCluster(ctx, infraClusterSpec)

		})
		It("should create a simple cluster with multi subnet, routeTable, securityGroup", func() {
			ctx := context.Background()
			infraClusterSpec := infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "cluster-api-net",
						IpRange: "10.0.0.0/16",
					},
					Subnets: []*infrastructurev1beta1.OscSubnet{
						{
							Name:          "cluster-api-subnet",
							IpSubnetRange: "10.0.0.0/24",
						},
						{
							Name:          "cluster-api-sub",
							IpSubnetRange: "10.0.1.0/24",
						},
					},
					InternetService: infrastructurev1beta1.OscInternetService{
						Name: "cluster-api-internetservice",
					},
					NatService: infrastructurev1beta1.OscNatService{
						Name:         "cluster-api-natservice",
						PublicIpName: "cluster-api-publicip",
						SubnetName:   "cluster-api-subnet",
					},
					RouteTables: []*infrastructurev1beta1.OscRouteTable{
						{
							Name:       "cluster-api-routetable",
							SubnetName: "cluster-api-subnet",
							Routes: []infrastructurev1beta1.OscRoute{
								{
									Name:        "cluster-api-routes",
									TargetName:  "cluster-api-internetservice",
									TargetType:  "gateway",
									Destination: "0.0.0.0/0",
								},
							},
						},
						{
							Name:       "cluster-api-rt",
							SubnetName: "cluster-api-sub",
							Routes: []infrastructurev1beta1.OscRoute{
								{
									Name:        "cluster-api-r",
									TargetName:  "cluster-api-natservice",
									TargetType:  "nat",
									Destination: "0.0.0.0/0",
								},
							},
						},
					},
					PublicIps: []*infrastructurev1beta1.OscPublicIp{
						{
							Name: "cluster-api-publicip",
						},
					},
					SecurityGroups: []*infrastructurev1beta1.OscSecurityGroup{
						{
							Name:        "cluster-api-securitygroups",
							Description: "securitygroup",
							SecurityGroupRules: []infrastructurev1beta1.OscSecurityGroupRule{
								{
									Name:          "cluster-api-securitygrouprule",
									Flow:          "Inbound",
									IpProtocol:    "tcp",
									IpRange:       "0.0.0.0/0",
									FromPortRange: 6443,
									ToPortRange:   6443,
								},
								{
									Name:          "cluster-api-securitygrouprule-http",
									Flow:          "Inbound",
									IpProtocol:    "tcp",
									IpRange:       "0.0.0.0/0",
									FromPortRange: 80,
									ToPortRange:   80,
								},
							},
						},
					},
					LoadBalancer: infrastructurev1beta1.OscLoadBalancer{
						LoadBalancerName:  "OscSdkExample-10",
						LoadBalancerType:  "internet-facing",
						SubnetName:        "cluster-api-subnet",
						SecurityGroupName: "cluster-api-securitygroups",
					},
				},
			}
			createCheckDeleteOscCluster(ctx, infraClusterSpec)

		})
		It("should create a simple cluster with default values", func() {
			ctx := context.Background()
			infraClusterSpec := infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name: "cluster-api-net",
					},
				},
			}
			createCheckDeleteOscCluster(ctx, infraClusterSpec)

		})

	})
})
