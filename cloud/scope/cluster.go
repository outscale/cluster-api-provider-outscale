/*
Copyright 2022 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package scope

import (
	"context"

	osc "github.com/outscale/osc-sdk-go/v2"

	"errors"
	"fmt"

	"github.com/go-logr/logr"
	infrastructurev1beta2 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta2"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud"
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
	OscCluster *infrastructurev1beta2.OscCluster
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
		return nil, fmt.Errorf("%w failed to create Osc Client", err)
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
		return nil, fmt.Errorf("%w failed to init patch helper", err)
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
	OscCluster  *infrastructurev1beta2.OscCluster
}

// Close closes the scope of the cluster configuration and status
func (s *ClusterScope) Close() error {
	return s.patchHelper.Patch(context.TODO(), s.OscCluster)
}

// GetName return the name of the cluster
func (s *ClusterScope) GetName() string {
	return s.Cluster.GetName()
}

// GetNamespace return the namespace of the cluster
func (s *ClusterScope) GetNamespace() string {
	return s.Cluster.GetNamespace()
}

// GetUID return the uid of the cluster
func (s *ClusterScope) GetUID() string {
	return string(s.Cluster.UID)
}

// GetAuth return outscale api context
func (s *ClusterScope) GetAuth() context.Context {
	return s.OscClient.auth
}

// GetApi return outscale api credential
func (s *ClusterScope) GetApi() *osc.APIClient {
	return s.OscClient.api
}

// GetInternetService return the internetService of the cluster
func (s *ClusterScope) GetInternetService() *infrastructurev1beta2.OscInternetService {
	return &s.OscCluster.Spec.Network.InternetService
}

// GetInternetServiceRef get the status of internetService (a Map with tag name with cluster uid associate with resource response id)
func (s *ClusterScope) GetInternetServiceRef() *infrastructurev1beta2.OscResourceReference {
	return &s.OscCluster.Status.Network.InternetServiceRef
}

// GetNatService return the natService of the cluster
func (s *ClusterScope) GetNatService() *infrastructurev1beta2.OscNatService {
	return &s.OscCluster.Spec.Network.NatService
}

// GetNatServiceRef get the status of natService (a Map with tag name with cluster uid associate with resource response id)
func (s *ClusterScope) GetNatServiceRef() *infrastructurev1beta2.OscResourceReference {
	return &s.OscCluster.Status.Network.NatServiceRef
}

// GetLoadBalancer return the loadbalanacer of the cluster
func (s *ClusterScope) GetLoadBalancer() *infrastructurev1beta2.OscLoadBalancer {
	return &s.OscCluster.Spec.Network.LoadBalancer
}

// GetNet return the net of the cluster
func (s *ClusterScope) GetNet() *infrastructurev1beta2.OscNet {
	return &s.OscCluster.Spec.Network.Net
}

// GetNetwork return the network of the cluster
func (s *ClusterScope) GetNetwork() *infrastructurev1beta2.OscNetwork {
	return &s.OscCluster.Spec.Network
}

// GetRouteTables return the routeTables of the cluster
func (s *ClusterScope) GetRouteTables() []*infrastructurev1beta2.OscRouteTable {
	return s.OscCluster.Spec.Network.RouteTables
}

// GetSecurityGroups return the securitygroup of the cluster
func (s *ClusterScope) GetSecurityGroups() []*infrastructurev1beta2.OscSecurityGroup {
	return s.OscCluster.Spec.Network.SecurityGroups
}

// GetRouteTablesRef get the status of routeTable (a Map with tag name with cluster uid associate with resource response id)
func (s *ClusterScope) GetRouteTablesRef() *infrastructurev1beta2.OscResourceReference {
	return &s.OscCluster.Status.Network.RouteTablesRef
}

// GetSecurityGroupsRef get the status of securityGroup
func (s *ClusterScope) GetSecurityGroupsRef() *infrastructurev1beta2.OscResourceReference {
	return &s.OscCluster.Status.Network.SecurityGroupsRef
}

// GetRouteRef get the status of route (a Map with tag name with cluster uid associate with resource response id)
func (s *ClusterScope) GetRouteRef() *infrastructurev1beta2.OscResourceReference {
	return &s.OscCluster.Status.Network.RouteRef
}

// GetSecurityGroupRuleRef get the status of securityGroup rule
func (s *ClusterScope) GetSecurityGroupRuleRef() *infrastructurev1beta2.OscResourceReference {
	return &s.OscCluster.Status.Network.SecurityGroupRuleRef
}

// PublicIpRef get the status of publicip (a Map with tag name with cluster uid associate with resource response id)
func (s *ClusterScope) GetPublicIpRef() *infrastructurev1beta2.OscResourceReference {
	return &s.OscCluster.Status.Network.PublicIpRef
}

// LinkRouteTablesRef get the status of route associate with a routeTables (a Map with tag name with cluster uid associate with resource response id)
func (s ClusterScope) GetLinkRouteTablesRef() map[string][]string {
	return s.OscCluster.Status.Network.LinkRouteTableRef
}

// SetLinkRouteTableRef set the status of route associate with a routeTables (a Map with tag name with cluster uid associate with resource response id)
func (s ClusterScope) SetLinkRouteTablesRef(linkRouteTableRef map[string][]string) {
	s.OscCluster.Status.Network.LinkRouteTableRef = linkRouteTableRef
}

// Route return slices of routes associated with routetable Name
func (s *ClusterScope) GetRoute(Name string) *[]infrastructurev1beta2.OscRoute {
	routeTables := s.OscCluster.Spec.Network.RouteTables
	for _, routeTable := range routeTables {
		if routeTable.Name == Name {
			return &routeTable.Routes
		}
	}
	return nil
}

// GetIpSubnetRange return IpSubnetRang from the subnet
func (s *ClusterScope) GetIpSubnetRange(Name string) string {
	subnets := s.OscCluster.Spec.Network.Subnets
	for _, subnet := range subnets {
		if subnet.Name == Name {
			return subnet.IpSubnetRange
		}
	}
	return ""
}

// GetSecurityGroupRule return slices of securityGroupRule asscociated with securityGroup Name
func (s *ClusterScope) GetSecurityGroupRule(Name string) *[]infrastructurev1beta2.OscSecurityGroupRule {
	securityGroups := s.OscCluster.Spec.Network.SecurityGroups
	for _, securityGroup := range securityGroups {
		if securityGroup.Name == Name {
			return &securityGroup.SecurityGroupRules
		}
	}
	return nil
}

// GetLinkPublicIpRef get the status of linkPublicIpRef (a Map with tag name with bastion uid associate with resource response id)
func (s *ClusterScope) GetLinkPublicIpRef() *infrastructurev1beta2.OscResourceReference {
	return &s.OscCluster.Status.Network.LinkPublicIpRef
}

// GetPublicIp return the public ip of the cluster
func (s *ClusterScope) GetPublicIp() []*infrastructurev1beta2.OscPublicIp {
	return s.OscCluster.Spec.Network.PublicIps
}

// GetLinkRouteTablesRef get the status of route associate with a routeTables (a Map with tag name with cluster uid associate with resource response id)
func (s *ClusterScope) GetNetRef() *infrastructurev1beta2.OscResourceReference {
	return &s.OscCluster.Status.Network.NetRef
}

// GetSubnet return the subnet of the cluster
func (s *ClusterScope) GetSubnet() []*infrastructurev1beta2.OscSubnet {
	return s.OscCluster.Spec.Network.Subnets
}

// GetSubnetRef get the subnet (a Map with tag name with cluster uid associate with resource response id)
func (s *ClusterScope) GetSubnetRef() *infrastructurev1beta2.OscResourceReference {
	return &s.OscCluster.Status.Network.SubnetRef
}

// SetControlPlaneEndpoint set controlPlane endpoint
func (s *ClusterScope) SetControlPlaneEndpoint(apiEndpoint clusterv1.APIEndpoint) {
	s.OscCluster.Spec.ControlPlaneEndpoint = apiEndpoint
}

// GetControlPlaneEndpoint get controlPlane endpoint
func (s *ClusterScope) GetControlPlaneEndpoint() clusterv1.APIEndpoint {
	return s.OscCluster.Spec.ControlPlaneEndpoint
}

// GetControlPlaneEndpointHost get controlPlane endpoint host
func (s *ClusterScope) GetControlPlaneEndpointHost() string {
	return s.OscCluster.Spec.ControlPlaneEndpoint.Host
}

// GetControlPlaneEndpointPort get controlPlane endpoint port
func (s *ClusterScope) GetControlPlaneEndpointPort() int32 {
	return s.OscCluster.Spec.ControlPlaneEndpoint.Port
}

// SetNotReady set not ready status
func (s *ClusterScope) SetNotReady() {
	s.OscCluster.Status.Ready = false
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

// GetBastion return the vm bastion
func (s *ClusterScope) GetBastion() *infrastructurev1beta2.OscBastion {
	return &s.OscCluster.Spec.Network.Bastion
}

// GetBastionRef get the bastion (a Map with cluster uid associate with resource response id)
func (s *ClusterScope) GetBastionRef() *infrastructurev1beta2.OscResourceReference {
	return &s.OscCluster.Status.Network.BastionRef
}

// GetBastionPrivateIps return the bastion privateIps
func (s *ClusterScope) GetBastionPrivateIps() *[]infrastructurev1beta2.OscPrivateIpElement {
	return &s.GetBastion().PrivateIps
}

// GetBastionSecurityGroups return the bastion securityGroups
func (s *ClusterScope) GetBastionSecurityGroups() *[]infrastructurev1beta2.OscSecurityGroupElement {
	return &s.GetBastion().SecurityGroupNames
}

// SetVmState set vmstate
func (s *ClusterScope) SetVmState(v infrastructurev1beta2.VmState) {
	s.OscCluster.Status.VmState = &v
}

// PatchObject keep the cluster configuration and status
func (s *ClusterScope) PatchObject() error {
	setConditions := []clusterv1.ConditionType{
		infrastructurev1beta2.NetReadyCondition,
		infrastructurev1beta2.SubnetsReadyCondition,
		infrastructurev1beta2.LoadBalancerReadyCondition}
	setConditions = append(setConditions,
		infrastructurev1beta2.InternetServicesReadyCondition,
		infrastructurev1beta2.NatServicesReadyCondition,
		infrastructurev1beta2.RouteTablesReadyCondition)
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
			infrastructurev1beta2.NetReadyCondition,
			infrastructurev1beta2.SubnetsReadyCondition,
			infrastructurev1beta2.InternetServicesReadyCondition,
			infrastructurev1beta2.NatServicesReadyCondition,
			infrastructurev1beta2.LoadBalancerReadyCondition,
		}})
}

func (s *ClusterScope) ListMachines(ctx context.Context) ([]*clusterv1.Machine, []*infrastructurev1beta2.OscMachine, error) {
	var machineListRaw clusterv1.MachineList
	var machineByOscMachineName = make(map[string]*clusterv1.Machine)
	if err := s.client.List(ctx, &machineListRaw, client.InNamespace(s.GetNamespace())); err != nil {
		return nil, nil, err
	}
	expectedGk := infrastructurev1beta2.GroupVersion.WithKind("OscMachine").GroupKind()
	for pos := range machineListRaw.Items {
		m := &machineListRaw.Items[pos]
		actualGk := m.Spec.InfrastructureRef.GroupVersionKind().GroupKind()
		if m.Spec.ClusterName != s.Cluster.Name || actualGk.String() != expectedGk.String() {
			continue
		}
		machineByOscMachineName[m.Spec.InfrastructureRef.Name] = m
	}
	var oscMachineListRaw infrastructurev1beta2.OscMachineList
	if err := s.client.List(ctx, &oscMachineListRaw, client.InNamespace(s.GetNamespace())); err != nil {
		return nil, nil, err
	}
	machineList := make([]*clusterv1.Machine, 0, len(oscMachineListRaw.Items))

	oscMachineList := make([]*infrastructurev1beta2.OscMachine, 0, len(oscMachineListRaw.Items))
	for pos := range oscMachineListRaw.Items {
		oscm := &oscMachineListRaw.Items[pos]
		m, ok := machineByOscMachineName[oscm.Name]
		if !ok {
			continue
		}
		machineList = append(machineList, m)
		oscMachineList = append(oscMachineList, oscm)
	}
	return machineList, oscMachineList, nil
}
