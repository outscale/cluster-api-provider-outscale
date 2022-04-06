package scope

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
        osc "github.com/outscale/osc-sdk-go/v2"
	infrastructurev1beta1 "github.com/outscale-vbr/cluster-api-provider-outscale.git/api/v1beta1"
	"k8s.io/klog/v2/klogr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/cluster-api/util/conditions"
        "github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud"
)

type ClusterScopeParams struct {
	OscClient *OscClient
	Client     client.Client
	Logger     logr.Logger
	Cluster    *clusterv1.Cluster
	OscCluster *infrastructurev1beta1.OscCluster
}

func NewClusterScope(params ClusterScopeParams) (*ClusterScope, error) {
	if params.Cluster == nil {
		return nil, errors.New("Cluster is required when creating a ClusterScope")
	}

	if params.OscCluster == nil {
		return nil, errors.New("OscCluster is required when creating a ClusterScope")
	}

	if params.Logger == (logr.Logger{}) {
		params.Logger = klogr.New()
	}

        client, err := newOscClient()

        if err != nil {
            return nil, errors.Wrap(err, "failed to create Osc Client")
        }

        if params.OscClient == nil {
            params.OscClient = client
        }

        if params.OscClient.api == nil {
            params.OscClient.api = client.api
        }

        if params.OscClient.auth == nil {
	    params.OscClient.auth = client.auth
        }
        
	helper, err := patch.NewHelper(params.OscCluster, params.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper")
	}
         
	return &ClusterScope{
		Logger:      params.Logger,
		client:      params.Client,
		Cluster:     params.Cluster,
		OscCluster:  params.OscCluster,
                OscClient:   params.OscClient,
		patchHelper: helper,
	}, nil
}

type ClusterScope struct {
	logr.Logger
	client      client.Client
	patchHelper *patch.Helper
        OscClient  *OscClient
	Cluster    *clusterv1.Cluster
	OscCluster *infrastructurev1beta1.OscCluster
}

func (s *ClusterScope) Close() error {
	return s.patchHelper.Patch(context.TODO(), s.OscCluster)
}

func (s *ClusterScope) Name() string {
	return s.Cluster.GetName()
}

func (s *ClusterScope) Namespace() string {
	return s.Cluster.GetNamespace()
}
func (s *ClusterScope) Region() string {
        return s.OscCluster.Spec.Network.LoadBalancer.SubregionName 
}
func (s *ClusterScope) UID() string {
	return string(s.Cluster.UID)
}
func (s *ClusterScope) Auth() context.Context {
        return s.OscClient.auth
}
func (s *ClusterScope) Api() *osc.APIClient {
        return s.OscClient.api
}
func (s *ClusterScope) InternetService() *infrastructurev1beta1.OscInternetService {
        return &s.OscCluster.Spec.Network.InternetService
}
func (s *ClusterScope) InternetServiceRef() *infrastructurev1beta1.OscResourceMapReference {
        return &s.OscCluster.Status.Network.InternetServiceRef
}
func (s *ClusterScope) NatService() *infrastructurev1beta1.OscNatService{
 	return &s.OscCluster.Spec.Network.NatService
}
func (s *ClusterScope) NatServiceRef() *infrastructurev1beta1.OscResourceMapReference {
	return &s.OscCluster.Status.Network.NatServiceRef
}
func (s *ClusterScope) LoadBalancer() *infrastructurev1beta1.OscLoadBalancer {
	return &s.OscCluster.Spec.Network.LoadBalancer
}
func (s *ClusterScope) Net() *infrastructurev1beta1.OscNet {
        return &s.OscCluster.Spec.Network.Net
}
func (s *ClusterScope) Network() *infrastructurev1beta1.OscNetwork {
	return &s.OscCluster.Spec.Network
}
func (s *ClusterScope) RouteTables() []*infrastructurev1beta1.OscRouteTable {
	return s.OscCluster.Spec.Network.RouteTables
}
func (s *ClusterScope) RouteTablesRef() *infrastructurev1beta1.OscResourceMapReference {
	return &s.OscCluster.Status.Network.RouteTablesRef
}
func (s *ClusterScope) RouteRef() *infrastructurev1beta1.OscResourceMapReference {
        return &s.OscCluster.Status.Network.RouteRef
}
func (s *ClusterScope) PublicIpRef() *infrastructurev1beta1.OscResourceMapReference {
	return &s.OscCluster.Status.Network.PublicIpRef
}
func (s ClusterScope) LinkRouteTablesRef() *infrastructurev1beta1.OscResourceMapReference {
        return &s.OscCluster.Status.Network.LinkRouteTableRef
}
func (s *ClusterScope) Route(Name string) *[]infrastructurev1beta1.OscRoute {
	routeTables := s.OscCluster.Spec.Network.RouteTables
        for _, routeTable := range routeTables {
            if routeTable.Name == Name {
                return &routeTable.Routes               
            }
        }
        return &routeTables[0].Routes
}
func (s *ClusterScope) PublicIp() []*infrastructurev1beta1.OscPublicIp {
        return s.OscCluster.Spec.Network.PublicIps
}
               
func (s *ClusterScope) NetRef() *infrastructurev1beta1.OscResourceMapReference {
        return &s.OscCluster.Status.Network.NetRef
}
func (s *ClusterScope) Subnet() []*infrastructurev1beta1.OscSubnet {
        return s.OscCluster.Spec.Network.Subnets
}
func (s *ClusterScope) SubnetRef() *infrastructurev1beta1.OscResourceMapReference {
        return &s.OscCluster.Status.Network.SubnetRef
}

func (s *ClusterScope) SetReady() {
	s.OscCluster.Status.Ready = true
}
func (s *ClusterScope) InfraCluster() cloud.ClusterObject {
    return s.OscCluster
}

func (s *ClusterScope) ClusterObj() cloud.ClusterObject {
    return s.Cluster
}

func (s *ClusterScope) PatchObject() error {
    setConditions := []clusterv1.ConditionType{
        infrastructurev1beta1.NetReadyCondition,
        infrastructurev1beta1.SubnetsReadyCondition,
        infrastructurev1beta1.LoadBalancerReadyCondition}
    setConditions = append(setConditions,
        infrastructurev1beta1.InternetServicesReadyCondition,
        infrastructurev1beta1.NatServicesReadyCondition,
        infrastructurev1beta1.RouteTablesReadyCondition)
    conditions.SetSummary(s.OscCluster,
        conditions.WithConditions(setConditions...),
        conditions.WithStepCounterIf(s.OscCluster.ObjectMeta.DeletionTimestamp.IsZero()),
        conditions.WithStepCounter(),
    )
    return s.patchHelper.Patch(
            context.TODO(),
            s.OscCluster,
            patch.WithOwnedConditions{Conditions: []clusterv1.ConditionType{
                clusterv1.ReadyCondition,
                infrastructurev1beta1.NetReadyCondition,
                infrastructurev1beta1.SubnetsReadyCondition,
                infrastructurev1beta1.InternetServicesReadyCondition,
                infrastructurev1beta1.NatServicesReadyCondition,
                infrastructurev1beta1.LoadBalancerReadyCondition,
            }})
}    
    
