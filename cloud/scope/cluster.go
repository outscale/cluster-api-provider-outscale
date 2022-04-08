package scope

import (
	"context"

	"github.com/go-logr/logr"
	infrastructurev1beta1 "github.com/outscale-vbr/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud"
	osc "github.com/outscale/osc-sdk-go/v2"
	"github.com/pkg/errors"
	"k8s.io/klog/v2/klogr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterScopeParams is a collection of input parameters to create a new scope
type ClusterScopeParams struct {
	OscClient  *OscClient
	Client     client.Client
	Logger     logr.Logger
	Cluster    *clusterv1.Cluster
	OscCluster *infrastructurev1beta1.OscCluster
}

// NewClusterScope create new clusterScope from parameters which is called at each reconciliation iteration
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

// ClusterScope is the basic context of the actuator that will be used
type ClusterScope struct {
	logr.Logger
	client      client.Client
	patchHelper *patch.Helper
	OscClient   *OscClient
	Cluster     *clusterv1.Cluster
	OscCluster  *infrastructurev1beta1.OscCluster
}

// Close closes the scope of the cluster configuration and status
func (s *ClusterScope) Close() error {
	return s.patchHelper.Patch(context.TODO(), s.OscCluster)
}

// Name return the name of the cluster
func (s *ClusterScope) Name() string {
	return s.Cluster.GetName()
}

// Namespace return the namespace of the cluster
func (s *ClusterScope) Namespace() string {
	return s.Cluster.GetNamespace()
}

// Region return the region of the cluster
func (s *ClusterScope) Region() string {
	return s.OscCluster.Spec.Network.LoadBalancer.SubregionName
}

// UID return the uid of the cluster
func (s *ClusterScope) UID() string {
	return string(s.Cluster.UID)
}

// Auth return outscale api context
func (s *ClusterScope) Auth() context.Context {
	return s.OscClient.auth
}

// api return outscale api credential
func (s *ClusterScope) Api() *osc.APIClient {
	return s.OscClient.api
}

// InternetService return the internetService of the cluster
func (s *ClusterScope) InternetService() *infrastructurev1beta1.OscInternetService {
	return &s.OscCluster.Spec.Network.InternetService
}

// InternetServiceRef get the status of internetService (a Map with tag name with cluster uid associate with resource response id)
func (s *ClusterScope) InternetServiceRef() *infrastructurev1beta1.OscResourceMapReference {
	return &s.OscCluster.Status.Network.InternetServiceRef
}

// NatService return the natService of the cluster
func (s *ClusterScope) NatService() *infrastructurev1beta1.OscNatService {
	return &s.OscCluster.Spec.Network.NatService
}

// NatServiceRef get the status of natService (a Map with tag name with cluster uid associate with resource response id)
func (s *ClusterScope) NatServiceRef() *infrastructurev1beta1.OscResourceMapReference {
	return &s.OscCluster.Status.Network.NatServiceRef
}

// LoadBalancer return the loadbalanacer of the cluster
func (s *ClusterScope) LoadBalancer() *infrastructurev1beta1.OscLoadBalancer {
	return &s.OscCluster.Spec.Network.LoadBalancer
}

// Net return the net of the cluster
func (s *ClusterScope) Net() *infrastructurev1beta1.OscNet {
	return &s.OscCluster.Spec.Network.Net
}

// Network return the network of the cluster
func (s *ClusterScope) Network() *infrastructurev1beta1.OscNetwork {
	return &s.OscCluster.Spec.Network
}

// RouteTables return the routeTables of the cluster
func (s *ClusterScope) RouteTables() []*infrastructurev1beta1.OscRouteTable {
	return s.OscCluster.Spec.Network.RouteTables
}

// RouteTablesRef get the status of routeTable (a Map with tag name with cluster uid associate with resource response id)
func (s *ClusterScope) RouteTablesRef() *infrastructurev1beta1.OscResourceMapReference {
	return &s.OscCluster.Status.Network.RouteTablesRef
}

// RouteRef get the status of route (a Map with tag name with cluster uid associate with resource response id)
func (s *ClusterScope) RouteRef() *infrastructurev1beta1.OscResourceMapReference {
	return &s.OscCluster.Status.Network.RouteRef
}

// PublicIpRef get the status of publicip (a Map with tag name with cluster uid associate with resource response id)
func (s *ClusterScope) PublicIpRef() *infrastructurev1beta1.OscResourceMapReference {
	return &s.OscCluster.Status.Network.PublicIpRef
}

// LinkRouteTablesRef get the status of route associate with a routeTables (a Map with tag name with cluster uid associate with resource response id)
func (s ClusterScope) LinkRouteTablesRef() *infrastructurev1beta1.OscResourceMapReference {
	return &s.OscCluster.Status.Network.LinkRouteTableRef
}

// Route return slices of routes asscoiated with routetable Name
func (s *ClusterScope) Route(Name string) *[]infrastructurev1beta1.OscRoute {
	routeTables := s.OscCluster.Spec.Network.RouteTables
	for _, routeTable := range routeTables {
		if routeTable.Name == Name {
			return &routeTable.Routes
		}
	}
	return &routeTables[0].Routes
}

// PublicIp return the public ip of the cluster
func (s *ClusterScope) PublicIp() []*infrastructurev1beta1.OscPublicIp {
	return s.OscCluster.Spec.Network.PublicIps
}

// LinkRouteTablesRef get the status of route associate with a routeTables (a Map with tag name with cluster uid associate with resource response id)
func (s *ClusterScope) NetRef() *infrastructurev1beta1.OscResourceMapReference {
	return &s.OscCluster.Status.Network.NetRef
}

// Subnet return the subnet of the cluster
func (s *ClusterScope) Subnet() []*infrastructurev1beta1.OscSubnet {
	return s.OscCluster.Spec.Network.Subnets
}

// SubnetRef get the subnet (a Map with tag name with cluster uid associate with resource response id)
func (s *ClusterScope) SubnetRef() *infrastructurev1beta1.OscResourceMapReference {
	return &s.OscCluster.Status.Network.SubnetRef
}

// SetReady set the ready status of the cluster
func (s *ClusterScope) SetReady() {
	s.OscCluster.Status.Ready = true
}

// InfraCluster return the Outscale infrastructure cluster
func (s *ClusterScope) InfraCluster() cloud.ClusterObject {
	return s.OscCluster
}

// ClusterObj return the cluster object
func (s *ClusterScope) ClusterObj() cloud.ClusterObject {
	return s.Cluster
}

// PatchObject keep the cluster configuration and status
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
