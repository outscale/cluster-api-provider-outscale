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
	"errors"
	"fmt"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud"
	osc "github.com/outscale/osc-sdk-go/v2"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const clusterUIDLabel = "outscale.com/clusterUID"

// ClusterScopeParams is a collection of input parameters to create a new scope
type ClusterScopeParams struct {
	OscClient  *cloud.OscClient
	Client     client.Client
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

	helper, err := patch.NewHelper(params.OscCluster, params.Client)
	if err != nil {
		return nil, fmt.Errorf("failed to init patch helper: %w", err)
	}

	return &ClusterScope{
		Client:      params.Client,
		Cluster:     params.Cluster,
		OscCluster:  params.OscCluster,
		OscClient:   params.OscClient,
		patchHelper: helper,
	}, nil
}

// ClusterScope is the basic context of the actuator that will be used
type ClusterScope struct {
	Client      client.Client
	patchHelper *patch.Helper
	OscClient   *cloud.OscClient
	Cluster     *clusterv1.Cluster
	OscCluster  *infrastructurev1beta1.OscCluster
}

// Close closes the scope of the cluster configuration and status
func (s *ClusterScope) Close(ctx context.Context) error {
	return s.patchHelper.Patch(ctx, s.OscCluster)
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
	explicitUID, hasExplicitUID := s.OscCluster.Labels[clusterUIDLabel]
	if hasExplicitUID {
		return explicitUID
	} else {
		return string(s.Cluster.UID)
	}
}

// EnsureExplicitUID creates the cluster UID label if missing
func (s *ClusterScope) EnsureExplicitUID() {
	_, hasExplicitUID := s.OscCluster.Labels[clusterUIDLabel]
	if !hasExplicitUID {
		s.OscCluster.Labels[clusterUIDLabel] = string(s.Cluster.UID)
	}
}

// GetAuth return outscale api context
func (s *ClusterScope) GetAuth() context.Context {
	return s.OscClient.Auth
}

// GetApi return outscale api credential
func (s *ClusterScope) GetApi() *osc.APIClient {
	return s.OscClient.API
}

// GetInternetService return the internetService of the cluster
func (s *ClusterScope) GetInternetService() infrastructurev1beta1.OscInternetService {
	return s.OscCluster.Spec.Network.InternetService
}

// GetInternetServiceName return the name of the net
func (s *ClusterScope) GetInternetServiceName() string {
	if s.OscCluster.Spec.Network.InternetService.Name != "" {
		return s.OscCluster.Spec.Network.InternetService.Name
	}
	return "Internet Service for " + s.OscCluster.ObjectMeta.Name + " cluster"
}

var ErrNoNatFound = errors.New("natService not found")

// GetNatService return the natService of the cluster
func (s *ClusterScope) GetNatService(name string, subregion string) (infrastructurev1beta1.OscNatService, error) {
	nats := s.GetNatServices()
	if len(nats) == 1 {
		return nats[0], nil
	}
	for _, spec := range nats {
		if spec.SubregionName == "" {
			spec.SubregionName = s.OscCluster.Spec.Network.SubregionName
		}
		switch {
		case name != "" && spec.Name == name:
			return spec, nil
		case spec.SubregionName == subregion || subregion == "":
			return spec, nil
		}
	}
	return infrastructurev1beta1.OscNatService{}, ErrNoNatFound
}

// GetNatServices return the natServices of the cluster
func (s *ClusterScope) GetNatServices() []infrastructurev1beta1.OscNatService {
	if len(s.OscCluster.Spec.Network.NatServices) == 0 && !s.OscCluster.Spec.Network.Net.UseExisting {
		var nss []infrastructurev1beta1.OscNatService
		srdone := map[string]bool{}
		for _, subnet := range s.OscCluster.Spec.Network.Subnets {
			sr := subnet.SubregionName
			if sr == "" {
				sr = s.OscCluster.Spec.Network.SubregionName
			}
			if srdone[sr] {
				continue
			}
			nss = append(nss, infrastructurev1beta1.OscNatService{
				SubregionName: sr,
			})
			srdone[sr] = true
		}
		return nss
	}
	return s.OscCluster.Spec.Network.NatServices
}

// GetNatServiceClientToken return the name of the net
func (s *ClusterScope) GetNatServiceClientToken(nat infrastructurev1beta1.OscNatService) string {
	if nat.Name != "" {
		return nat.Name + "-" + s.GetUID()
	}
	return nat.SubregionName + "-" + s.GetUID()
}

// GetNatServiceClientToken return the name of the net
func (s *ClusterScope) GetNatServiceName(nat infrastructurev1beta1.OscNatService) string {
	if nat.Name != "" {
		return nat.Name
	}
	return "Nat service in for " + s.OscCluster.ObjectMeta.Name + " cluster"
}

// GetLoadBalancer return the loadbalanacer of the cluster
func (s *ClusterScope) GetLoadBalancer() *infrastructurev1beta1.OscLoadBalancer {
	return &s.OscCluster.Spec.Network.LoadBalancer
}

// GetNet return the net of the cluster
func (s *ClusterScope) GetNet() infrastructurev1beta1.OscNet {
	if s.OscCluster.Spec.Network.Net.IsZero() {
		d := infrastructurev1beta1.DefaultNet
		d.Name = s.GetNetName()
		return d
	}
	return s.OscCluster.Spec.Network.Net
}

// GetNetName return the name of the net
func (s *ClusterScope) GetNetName() string {
	if s.OscCluster.Spec.Network.Net.Name != "" {
		return s.OscCluster.Spec.Network.Net.Name
	}
	return "Net for " + s.OscCluster.ObjectMeta.Name + " cluster"
}

// GetNetName return the name of the net
func (s *ClusterScope) GetSubnetName(spec infrastructurev1beta1.OscSubnet) string {
	if spec.Name != "" {
		return spec.Name
	}

	return string(spec.GetRole()) + " subnet for " + s.OscCluster.ObjectMeta.Name + " cluster"
}

var ErrNoSubnetFound = errors.New("subnet not found")

func (s *ClusterScope) GetSubnet(name string, role infrastructurev1beta1.OscRole, subregion string) (infrastructurev1beta1.OscSubnet, error) {
	for _, spec := range s.OscCluster.Spec.Network.Subnets {
		if spec.SubregionName == "" {
			spec.SubregionName = s.OscCluster.Spec.Network.SubregionName
		}
		switch {
		case name != "" && spec.Name == name:
			return spec, nil
		case spec.GetRole() != role:
		case spec.SubregionName == subregion || subregion == "":
			return spec, nil
		}
	}
	return infrastructurev1beta1.OscSubnet{}, ErrNoSubnetFound
}

// GetExtraSecurityGroupRule return the extraSecurityGroupRule
func (s *ClusterScope) GetExtraSecurityGroupRule() bool {
	return s.OscCluster.Spec.Network.ExtraSecurityGroupRule
}

// SetExtraSecurityGroupRule set the extraSecurityGroupRule
func (s *ClusterScope) SetExtraSecurityGroupRule(extraSecurityGroupRule bool) {
	s.OscCluster.Spec.Network.ExtraSecurityGroupRule = extraSecurityGroupRule
}

// GetNetwork return the network of the cluster
func (s *ClusterScope) GetNetwork() *infrastructurev1beta1.OscNetwork {
	return &s.OscCluster.Spec.Network
}

// GetRouteTables return the routeTables of the cluster
func (s *ClusterScope) GetRouteTables() []infrastructurev1beta1.OscRouteTable {
	if len(s.OscCluster.Spec.Network.RouteTables) == 0 {
		rtbls := []infrastructurev1beta1.OscRouteTable{
			{
				Role:   infrastructurev1beta1.RolePublic,
				Routes: []infrastructurev1beta1.OscRoute{{Destination: "0.0.0.0/0", TargetType: "gateway"}},
			},
			{
				Role:   infrastructurev1beta1.RoleControlPlane,
				Routes: []infrastructurev1beta1.OscRoute{{Destination: "0.0.0.0/0", TargetType: "nat"}},
			},
		}
		for _, subnet := range s.OscCluster.Spec.Network.Subnets {
			rtbls = append(rtbls, infrastructurev1beta1.OscRouteTable{
				Role:          infrastructurev1beta1.RoleWorker,
				SubregionName: subnet.SubregionName,
				Routes:        []infrastructurev1beta1.OscRoute{{Destination: "0.0.0.0/0", TargetType: "nat"}},
			})
		}
		return rtbls
	}
	return s.OscCluster.Spec.Network.RouteTables
}

// GetSecurityGroups return the securitygroup of the cluster
func (s *ClusterScope) GetSecurityGroups() []infrastructurev1beta1.OscSecurityGroup {
	return s.OscCluster.Spec.Network.SecurityGroups
}

// GetSecurityGroupsRef get the status of securityGroup
// Deprecated
func (s *ClusterScope) GetSecurityGroupsRef() *infrastructurev1beta1.OscResourceReference {
	return &s.OscCluster.Status.Network.SecurityGroupsRef
}

// GetSecurityGroupRuleRef get the status of securityGroup rule
// Deprecated
func (s *ClusterScope) GetSecurityGroupRuleRef() *infrastructurev1beta1.OscResourceReference {
	return &s.OscCluster.Status.Network.SecurityGroupRuleRef
}

// PublicIpRef get the status of publicip (a Map with tag name with cluster uid associate with resource response id)
// Deprecated
func (s *ClusterScope) GetPublicIpRef() *infrastructurev1beta1.OscResourceReference {
	return &s.OscCluster.Status.Network.PublicIpRef
}

// LinkRouteTablesRef get the status of route associate with a routeTables (a Map with tag name with cluster uid associate with resource response id)
// Deprecated
func (s ClusterScope) GetLinkRouteTablesRef() map[string][]string {
	return s.OscCluster.Status.Network.LinkRouteTableRef
}

// SetFailureDomain sets the infrastructure provider failure domain key to the spec given as input.
func (s *ClusterScope) SetFailureDomain(id string, spec clusterv1.FailureDomainSpec) {
	if s.OscCluster.Status.FailureDomains == nil {
		s.OscCluster.Status.FailureDomains = make(clusterv1.FailureDomains)
	}
	s.OscCluster.Status.FailureDomains[id] = spec
}

// SetLinkRouteTableRef set the status of route associate with a routeTables (a Map with tag name with cluster uid associate with resource response id)
// Deprecated
func (s ClusterScope) SetLinkRouteTablesRef(linkRouteTableRef map[string][]string) {
	s.OscCluster.Status.Network.LinkRouteTableRef = linkRouteTableRef
}

// Route return slices of routes associated with routetable Name
func (s *ClusterScope) GetRoute(name string) []infrastructurev1beta1.OscRoute {
	routeTables := s.OscCluster.Spec.Network.RouteTables
	for _, routeTable := range routeTables {
		if routeTable.Name == name {
			return routeTable.Routes
		}
	}
	return nil
}

// GetIpSubnetRange return IpSubnetRang from the subnet
func (s *ClusterScope) GetIpSubnetRange(name string) string {
	subnets := s.OscCluster.Spec.Network.Subnets
	for _, subnet := range subnets {
		if subnet.Name == name {
			return subnet.IpSubnetRange
		}
	}
	return ""
}

// GetSecurityGroupRule return slices of securityGroupRule asscociated with securityGroup Name
func (s *ClusterScope) GetSecurityGroupRule(name string) []infrastructurev1beta1.OscSecurityGroupRule {
	securityGroups := s.OscCluster.Spec.Network.SecurityGroups
	for _, securityGroup := range securityGroups {
		if securityGroup.Name == name {
			return securityGroup.SecurityGroupRules
		}
	}
	return nil
}

// GetLinkPublicIpRef get the status of linkPublicIpRef (a Map with tag name with bastion uid associate with resource response id)
// Deprecated
func (s *ClusterScope) GetLinkPublicIpRef() *infrastructurev1beta1.OscResourceReference {
	return &s.OscCluster.Status.Network.LinkPublicIpRef
}

// GetPublicIp return the public ip of the cluster
func (s *ClusterScope) GetPublicIp() []*infrastructurev1beta1.OscPublicIp {
	return s.OscCluster.Spec.Network.PublicIps
}

// GetLinkRouteTablesRef get the status of route associate with a routeTables (a Map with tag name with cluster uid associate with resource response id)
// Deprecated
func (s *ClusterScope) GetNetRef() *infrastructurev1beta1.OscResourceReference {
	return &s.OscCluster.Status.Network.NetRef
}

// GetSubnet return the subnet of the cluster
func (s *ClusterScope) GetSubnets() []infrastructurev1beta1.OscSubnet {
	return s.OscCluster.Spec.Network.Subnets
}

// GetSubnetRef get the subnet (a Map with tag name with cluster uid associate with resource response id)
func (s *ClusterScope) GetSubnetRef() *infrastructurev1beta1.OscResourceReference {
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

// GetReady get ready status
func (s *ClusterScope) GetReady() bool {
	return s.OscCluster.Status.Ready
}

// SetNotReady set not ready status
func (s *ClusterScope) SetNotReady() {
	s.OscCluster.Status.Ready = false
}

// SetReady set the ready status of the cluster
func (s *ClusterScope) SetReady() {
	s.OscCluster.Status.Ready = true
}

// GetBastion return the vm bastion
func (s *ClusterScope) GetBastion() *infrastructurev1beta1.OscBastion {
	return &s.OscCluster.Spec.Network.Bastion
}

// GetBastionName return the name of the bastion
func (s *ClusterScope) GetBastionName() string {
	if s.OscCluster.Spec.Network.Bastion.Name != "" {
		return s.OscCluster.Spec.Network.Bastion.Name
	}
	return "Bastion for " + s.OscCluster.ObjectMeta.Name + " cluster"
}

// GetBastionRef get the bastion (a Map with cluster uid associate with resource response id)
func (s *ClusterScope) GetBastionRef() *infrastructurev1beta1.OscResourceReference {
	return &s.OscCluster.Status.Network.BastionRef
}

// GetBastionPrivateIps return the bastion privateIps
func (s *ClusterScope) GetBastionPrivateIps() *[]infrastructurev1beta1.OscPrivateIpElement {
	return &s.GetBastion().PrivateIps
}

// GetBastionSecurityGroups return the bastion securityGroups
func (s *ClusterScope) GetBastionSecurityGroups() *[]infrastructurev1beta1.OscSecurityGroupElement {
	return &s.GetBastion().SecurityGroupNames
}

// GetImage return the image
func (s *ClusterScope) GetImage() *infrastructurev1beta1.OscImage {
	return &s.OscCluster.Spec.Network.Image
}

// SetVmState set vmstate
func (s *ClusterScope) SetVmState(v infrastructurev1beta1.VmState) {
	s.OscCluster.Status.VmState = &v
}

// SetVmState set vmstate
func (s *ClusterScope) GetVmState() *infrastructurev1beta1.VmState {
	return s.OscCluster.Status.VmState
}

// GetResources returns the resource list from the OscCluster status.
func (s *ClusterScope) GetResources() *infrastructurev1beta1.OscClusterResources {
	return &s.OscCluster.Status.Resources
}

// NeedReconciliation returns true if a reconciler needs to run.
func (s *ClusterScope) NeedReconciliation(reconciler infrastructurev1beta1.Reconciler) bool {
	if s.OscCluster.Status.ReconcilerGeneration == nil {
		return true
	}
	return s.OscCluster.Status.ReconcilerGeneration[reconciler] < s.OscCluster.Generation
}

// SetReconciliationGeneration marks a reconciler as having finished its job for a specific cluster generation.
func (s *ClusterScope) SetReconciliationGeneration(reconciler infrastructurev1beta1.Reconciler) {
	if s.OscCluster.Status.ReconcilerGeneration == nil {
		s.OscCluster.Status.ReconcilerGeneration = map[infrastructurev1beta1.Reconciler]int64{}
	}
	s.OscCluster.Status.ReconcilerGeneration[reconciler] = s.OscCluster.Generation
}

// PatchObject keep the cluster configuration and status
func (s *ClusterScope) PatchObject(ctx context.Context) error {
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
		ctx,
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

func (s *ClusterScope) ListMachines(ctx context.Context) ([]*clusterv1.Machine, []*infrastructurev1beta1.OscMachine, error) {
	var machineListRaw clusterv1.MachineList
	machineByOscMachineName := make(map[string]*clusterv1.Machine)
	if err := s.Client.List(ctx, &machineListRaw, client.InNamespace(s.GetNamespace())); err != nil {
		return nil, nil, err
	}
	expectedGk := infrastructurev1beta1.GroupVersion.WithKind("OscMachine").GroupKind()
	for pos := range machineListRaw.Items {
		m := &machineListRaw.Items[pos]
		actualGk := m.Spec.InfrastructureRef.GroupVersionKind().GroupKind()
		if m.Spec.ClusterName != s.Cluster.Name || actualGk.String() != expectedGk.String() {
			continue
		}
		machineByOscMachineName[m.Spec.InfrastructureRef.Name] = m
	}
	var oscMachineListRaw infrastructurev1beta1.OscMachineList
	if err := s.Client.List(ctx, &oscMachineListRaw, client.InNamespace(s.GetNamespace())); err != nil {
		return nil, nil, err
	}
	machineList := make([]*clusterv1.Machine, 0, len(oscMachineListRaw.Items))

	oscMachineList := make([]*infrastructurev1beta1.OscMachine, 0, len(oscMachineListRaw.Items))
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
