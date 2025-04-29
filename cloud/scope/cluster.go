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
	"net"
	"slices"
	"strings"

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

// GetNetwork return the network of the cluster
func (s *ClusterScope) GetNetwork() *infrastructurev1beta1.OscNetwork {
	return &s.OscCluster.Spec.Network
}

// GetNet return the net of the cluster
func (s *ClusterScope) GetNet() infrastructurev1beta1.OscNet {
	if s.OscCluster.Spec.Network.Net.IsZero() {
		return infrastructurev1beta1.DefaultNet
	}
	return s.OscCluster.Spec.Network.Net
}

// GetNetName return the name of the net
func (s *ClusterScope) GetNetName() string {
	if s.OscCluster.Spec.Network.Net.Name != "" {
		return s.OscCluster.Spec.Network.Net.Name
	}
	return "Net for " + s.OscCluster.ObjectMeta.Name
}

// GetDefaultSubregion returns the default subregion.
func (s *ClusterScope) GetDefaultSubregion() string {
	if len(s.GetNetwork().Subregions) > 0 {
		return s.GetNetwork().Subregions[0]
	}
	return s.GetNetwork().SubregionName
}

// GetSubregions returns the subregions where to deploy the cluster.
func (s *ClusterScope) GetSubregions() []string {
	if len(s.GetNetwork().Subregions) > 0 {
		return s.GetNetwork().Subregions
	}
	return []string{s.GetNetwork().SubregionName}
}

// GetSubnets returns the subnets of the cluster.
func (s *ClusterScope) GetSubnets() []infrastructurev1beta1.OscSubnet {
	if len(s.OscCluster.Spec.Network.Subnets) > 0 {
		return s.OscCluster.Spec.Network.Subnets
	}
	_, net, err := net.ParseCIDR(s.GetNet().IpRange)
	if err != nil {
		return nil
	}
	fds := s.GetSubregions()
	subnets := make([]infrastructurev1beta1.OscSubnet, 0, 3*len(fds))
	net.IP[2]++
	net.Mask[1], net.Mask[2] = 255, 255
	for _, fd := range fds {
		for _, roles := range [][]infrastructurev1beta1.OscRole{
			{infrastructurev1beta1.RoleLoadBalancer, infrastructurev1beta1.RoleBastion, infrastructurev1beta1.RoleNat},
			{infrastructurev1beta1.RoleWorker}, {infrastructurev1beta1.RoleControlPlane}} {
			net.IP[2]++
			subnet := infrastructurev1beta1.OscSubnet{
				IpSubnetRange: net.String(),
				Roles:         roles,
				SubregionName: fd,
			}
			subnets = append(subnets, subnet)
		}
	}
	return subnets
}

var ErrNoSubnetFound = errors.New("subnet not found")

func (s *ClusterScope) GetSubnet(name string, role infrastructurev1beta1.OscRole, subregion string) (infrastructurev1beta1.OscSubnet, error) {
	if subregion == "" {
		subregion = s.GetDefaultSubregion()
	}
	for _, spec := range s.GetSubnets() {
		switch {
		case name != "" && spec.Name == name:
			return spec, nil
		case !s.SubnetHasRole(spec, role):
		case s.GetSubnetSubregion(spec) == subregion:
			return spec, nil
		}
	}
	return infrastructurev1beta1.OscSubnet{}, ErrNoSubnetFound
}

func (s *ClusterScope) SubnetHasRole(spec infrastructurev1beta1.OscSubnet, role infrastructurev1beta1.OscRole) bool {
	if len(spec.Roles) > 0 {
		return slices.Contains(spec.Roles, role)
	}
	if slices.Contains(s.OscCluster.Spec.Network.ControlPlaneSubnets, spec.Name) || strings.Contains(spec.Name, "kcp") {
		return role == infrastructurev1beta1.RoleControlPlane
	}
	if s.OscCluster.Spec.Network.LoadBalancer.SubnetName != "" && spec.Name == s.OscCluster.Spec.Network.LoadBalancer.SubnetName {
		return role == infrastructurev1beta1.RoleLoadBalancer || role == infrastructurev1beta1.RoleBastion || role == infrastructurev1beta1.RoleNat
	}
	return role == infrastructurev1beta1.RoleWorker
}

func (s *ClusterScope) SubnetIsPublic(spec infrastructurev1beta1.OscSubnet) bool {
	return s.SubnetHasRole(spec, infrastructurev1beta1.RoleBastion) || s.SubnetHasRole(spec, infrastructurev1beta1.RoleLoadBalancer) || s.SubnetHasRole(spec, infrastructurev1beta1.RoleNat)
}

func (s *ClusterScope) GetSubnetSubregion(spec infrastructurev1beta1.OscSubnet) string {
	if spec.SubregionName != "" {
		return spec.SubregionName
	}
	return s.GetDefaultSubregion()
}

func (s *ClusterScope) GetSubnetName(spec infrastructurev1beta1.OscSubnet) string {
	if spec.Name != "" {
		return spec.Name
	}
	fd := s.GetSubnetSubregion(spec)
	switch {
	case s.SubnetIsPublic(spec):
		return "Public subnet for " + s.OscCluster.ObjectMeta.Name + "/" + fd
	case s.SubnetHasRole(spec, infrastructurev1beta1.RoleControlPlane):
		return "Controlplane subnet for " + s.OscCluster.ObjectMeta.Name + "/" + fd
	case s.SubnetHasRole(spec, infrastructurev1beta1.RoleWorker):
		return "Worker subnet for " + s.OscCluster.ObjectMeta.Name + "/" + fd
	default:
		return "Subnet for " + s.OscCluster.ObjectMeta.Name + "/" + fd
	}
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
	return "Internet Service for " + s.OscCluster.ObjectMeta.Name
}

var ErrNoNatFound = errors.New("natService not found")

// GetNatServices return the natServices of the cluster
func (s *ClusterScope) GetNatServices() []infrastructurev1beta1.OscNatService {
	switch {
	case s.OscCluster.Spec.Network.UseExisting.Net:
		return nil
	case len(s.OscCluster.Spec.Network.NatServices) > 0:
		return s.OscCluster.Spec.Network.NatServices
	case s.OscCluster.Spec.Network.NatService != infrastructurev1beta1.OscNatService{}:
		return []infrastructurev1beta1.OscNatService{s.OscCluster.Spec.Network.NatService}
	default:
		var nss []infrastructurev1beta1.OscNatService
		for _, subnet := range s.GetSubnets() {
			if !s.SubnetHasRole(subnet, infrastructurev1beta1.RoleNat) {
				continue
			}
			nss = append(nss, infrastructurev1beta1.OscNatService{
				SubregionName: s.GetSubnetSubregion(subnet),
				SubnetName:    subnet.Name,
			})
		}
		return nss
	}
}

// GetNatService return the natService of the cluster
func (s *ClusterScope) GetNatService(name string, subregion string) (infrastructurev1beta1.OscNatService, error) {
	nats := s.GetNatServices()
	if len(nats) == 1 {
		return nats[0], nil
	}
	for _, spec := range nats {
		if spec.SubregionName == "" {
			spec.SubregionName = s.GetDefaultSubregion()
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

// GetNatServiceName return the name of a nat service
func (s *ClusterScope) GetNatServiceName(nat infrastructurev1beta1.OscNatService) string {
	if nat.Name != "" {
		return nat.Name
	}
	name := "Nat service for " + s.OscCluster.ObjectMeta.Name
	if nat.SubregionName != "" {
		name += "/" + nat.SubregionName
	}
	return name
}

// GetNatServiceClientToken return the client token for a nat service
func (s *ClusterScope) GetNatServiceClientToken(nat infrastructurev1beta1.OscNatService) string {
	if nat.Name != "" {
		ct := nat.Name + "-" + s.GetUID()
		if len(ct) > 64 {
			ct = ct[len(ct)-64:]
		}
		return ct
	}
	return nat.SubregionName + "-" + s.GetUID()
}

// GetRouteTables return the routeTables of the cluster
func (s *ClusterScope) GetRouteTables() []infrastructurev1beta1.OscRouteTable {
	if len(s.OscCluster.Spec.Network.RouteTables) > 0 {
		return s.OscCluster.Spec.Network.RouteTables
	}
	subnets := s.GetSubnets()
	rtbls := make([]infrastructurev1beta1.OscRouteTable, 0, len(subnets))
	for _, subnet := range subnets {
		rtbl := infrastructurev1beta1.OscRouteTable{
			Name:          s.GetSubnetName(subnet),
			SubregionName: s.GetSubnetSubregion(subnet),
		}
		if subnet.Name != "" {
			rtbl.Subnets = []string{subnet.Name}
		}
		if len(subnet.Roles) > 0 {
			rtbl.Role = subnet.Roles[0]
		}
		if s.SubnetIsPublic(subnet) {
			rtbl.Routes = []infrastructurev1beta1.OscRoute{{Destination: "0.0.0.0/0", TargetType: "gateway"}}
		} else {
			rtbl.Routes = []infrastructurev1beta1.OscRoute{{Destination: "0.0.0.0/0", TargetType: "nat"}}
		}
		rtbls = append(rtbls, rtbl)
	}
	return rtbls
}

// HasIPRestriction returns true if AllowFromIps is set.
func (s *ClusterScope) HasIPRestriction() bool {
	return len(s.OscCluster.Spec.Network.AllowFromIPRanges) > 0
}

func (s *ClusterScope) getAdditionalRules(roles ...infrastructurev1beta1.OscRole) []infrastructurev1beta1.OscSecurityGroupRule {
	for _, ar := range s.GetNetwork().AdditionalSecurityRules {
		if slices.Equal(roles, ar.Roles) {
			return ar.Rules
		}
	}
	return nil
}

// GetSecurityGroups returns the list of all security groups for the cluster.
func (s *ClusterScope) GetSecurityGroups() []infrastructurev1beta1.OscSecurityGroup {
	if len(s.OscCluster.Spec.Network.SecurityGroups) > 0 {
		return s.OscCluster.Spec.Network.SecurityGroups
	}
	var allSN, allSNCP, allSNBastion []string
	for _, sn := range s.GetSubnets() {
		if s.SubnetHasRole(sn, infrastructurev1beta1.RoleBastion) {
			allSNBastion = append(allSNBastion, sn.IpSubnetRange)
		}
		if s.SubnetIsPublic(sn) {
			continue
		}
		if s.SubnetHasRole(sn, infrastructurev1beta1.RoleControlPlane) {
			allSNCP = append(allSNCP, sn.IpSubnetRange)
		}
		allSN = append(allSN, sn.IpSubnetRange)
	}
	allowedIn := s.OscCluster.Spec.Network.AllowFromIPRanges
	if len(allowedIn) == 0 {
		allowedIn = []string{"0.0.0.0/0"}
	}
	lb := infrastructurev1beta1.OscSecurityGroup{
		Name:        s.GetName() + "-lb",
		Description: "LB securityGroup for " + s.GetName(),
		Roles:       []infrastructurev1beta1.OscRole{infrastructurev1beta1.RoleLoadBalancer},
		SecurityGroupRules: []infrastructurev1beta1.OscSecurityGroupRule{
			{Flow: "Inbound", IpProtocol: "tcp", FromPortRange: 6443, ToPortRange: 6443, IpRanges: allowedIn},
			{Flow: "Outbound", IpProtocol: "tcp", FromPortRange: 6443, ToPortRange: 6443, IpRanges: allSNCP},
		},
		Authoritative: true,
	}
	lb.SecurityGroupRules = append(lb.SecurityGroupRules, s.getAdditionalRules(infrastructurev1beta1.RoleLoadBalancer)...)

	worker := infrastructurev1beta1.OscSecurityGroup{
		Name:        s.GetName() + "-worker",
		Description: "Worker securityGroup for " + s.GetName(),
		Roles:       []infrastructurev1beta1.OscRole{infrastructurev1beta1.RoleWorker},
		SecurityGroupRules: []infrastructurev1beta1.OscSecurityGroupRule{
			{Flow: "Inbound", IpProtocol: "tcp", FromPortRange: 30000, ToPortRange: 32767, IpRanges: allSN}, // NodePort
			{Flow: "Inbound", IpProtocol: "tcp", FromPortRange: 10250, ToPortRange: 10250, IpRanges: allSN}, // Kubelet
		},
		Authoritative: true,
	}
	worker.SecurityGroupRules = append(worker.SecurityGroupRules, s.getAdditionalRules(infrastructurev1beta1.RoleWorker)...)

	controlplane := infrastructurev1beta1.OscSecurityGroup{
		Name:        s.GetName() + "-controlplane",
		Description: "Controlplane securityGroup for " + s.GetName(),
		Roles:       []infrastructurev1beta1.OscRole{infrastructurev1beta1.RoleControlPlane},
		SecurityGroupRules: []infrastructurev1beta1.OscSecurityGroupRule{
			{Flow: "Inbound", IpProtocol: "tcp", FromPortRange: 6443, ToPortRange: 6443, IpRange: s.GetNet().IpRange}, // API
			{Flow: "Inbound", IpProtocol: "tcp", FromPortRange: 30000, ToPortRange: 32767, IpRanges: allSN},           // NodePort
			{Flow: "Inbound", IpProtocol: "tcp", FromPortRange: 2378, ToPortRange: 2380, IpRanges: allSNCP},           // etcd
			{Flow: "Inbound", IpProtocol: "tcp", FromPortRange: 10250, ToPortRange: 10252, IpRanges: allSNCP},         // Kubelet
		},
		Authoritative: true,
	}
	controlplane.SecurityGroupRules = append(controlplane.SecurityGroupRules, s.getAdditionalRules(infrastructurev1beta1.RoleControlPlane)...)

	allowedOut := s.OscCluster.Spec.Network.AllowToIPRanges
	switch {
	case len(allowedOut) == 0:
		allowedOut = []string{"0.0.0.0/0"}
	case allowedOut[0] == "":
		allowedOut = nil
	}

	node := infrastructurev1beta1.OscSecurityGroup{
		Name:        s.GetName() + "-node",
		Description: "Node securityGroup for " + s.GetName(),
		Roles:       []infrastructurev1beta1.OscRole{infrastructurev1beta1.RoleControlPlane, infrastructurev1beta1.RoleWorker},
		SecurityGroupRules: []infrastructurev1beta1.OscSecurityGroupRule{
			{Flow: "Inbound", IpProtocol: "icmp", FromPortRange: 8, ToPortRange: 8, IpRange: s.GetNet().IpRange},        // ICMP
			{Flow: "Inbound", IpProtocol: "tcp", FromPortRange: 179, ToPortRange: 179, IpRange: s.GetNet().IpRange},     // BGP
			{Flow: "Inbound", IpProtocol: "udp", FromPortRange: 4789, ToPortRange: 4789, IpRange: s.GetNet().IpRange},   // Calico VXLAN
			{Flow: "Inbound", IpProtocol: "udp", FromPortRange: 5473, ToPortRange: 5473, IpRange: s.GetNet().IpRange},   // Typha
			{Flow: "Inbound", IpProtocol: "udp", FromPortRange: 51820, ToPortRange: 51821, IpRange: s.GetNet().IpRange}, // Wiregard
			{Flow: "Inbound", IpProtocol: "udp", FromPortRange: 8285, ToPortRange: 8285, IpRange: s.GetNet().IpRange},   // Flannel
			{Flow: "Inbound", IpProtocol: "udp", FromPortRange: 8472, ToPortRange: 8472, IpRange: s.GetNet().IpRange},   // Flannel VXLAN
			{Flow: "Inbound", IpProtocol: "tcp", FromPortRange: 4240, ToPortRange: 4240, IpRange: s.GetNet().IpRange},   // Cillium health
			{Flow: "Inbound", IpProtocol: "tcp", FromPortRange: 4244, ToPortRange: 4244, IpRange: s.GetNet().IpRange},   // Cillium hubble
		},
		Tag:           "OscK8sMainSG",
		Authoritative: true,
	}
	// Outbound traffic
	if len(allowedOut) > 0 {
		node.SecurityGroupRules = append(node.SecurityGroupRules,
			infrastructurev1beta1.OscSecurityGroupRule{Flow: "Outbound", IpProtocol: "-1", FromPortRange: -1, ToPortRange: -1, IpRanges: allowedOut},
		)
	}
	node.SecurityGroupRules = append(node.SecurityGroupRules, s.getAdditionalRules(infrastructurev1beta1.RoleControlPlane, infrastructurev1beta1.RoleWorker)...)

	if !s.OscCluster.Spec.Network.Bastion.Enable {
		return []infrastructurev1beta1.OscSecurityGroup{lb, worker, controlplane, node}
	}
	node.SecurityGroupRules = append(node.SecurityGroupRules, infrastructurev1beta1.OscSecurityGroupRule{
		Flow: "Inbound", IpProtocol: "tcp", FromPortRange: 22, ToPortRange: 22, IpRanges: allSNBastion,
	})
	bastion := infrastructurev1beta1.OscSecurityGroup{
		Name:        s.GetName() + "-bastion",
		Description: "Bastion securityGroup for " + s.GetName(),
		Roles:       []infrastructurev1beta1.OscRole{infrastructurev1beta1.RoleBastion},
		SecurityGroupRules: []infrastructurev1beta1.OscSecurityGroupRule{
			{Flow: "Inbound", IpProtocol: "tcp", FromPortRange: 22, ToPortRange: 22, IpRanges: allowedIn},
			{Flow: "Outbound", IpProtocol: "tcp", FromPortRange: 22, ToPortRange: 22, IpRange: s.GetNet().IpRange},
		},
		Authoritative: true,
	}
	// Outbound traffic
	if len(allowedOut) > 0 {
		bastion.SecurityGroupRules = append(bastion.SecurityGroupRules,
			infrastructurev1beta1.OscSecurityGroupRule{Flow: "Outbound", IpProtocol: "-1", FromPortRange: -1, ToPortRange: -1, IpRanges: allowedOut},
		)
	}
	bastion.SecurityGroupRules = append(bastion.SecurityGroupRules, s.getAdditionalRules(infrastructurev1beta1.RoleBastion)...)

	return []infrastructurev1beta1.OscSecurityGroup{lb, worker, controlplane, node, bastion}
}

var ErrNoSecurityGroupFound = errors.New("no security group found")

// getSecurityGroupsForNames returns the security group having names.
func (s *ClusterScope) getSecurityGroupsForNames(names []infrastructurev1beta1.OscSecurityGroupElement) ([]infrastructurev1beta1.OscSecurityGroup, error) {
	var sgs []infrastructurev1beta1.OscSecurityGroup
LOOPNAMES:
	for _, name := range names {
		for _, spec := range s.GetSecurityGroups() {
			if spec.Name == name.Name {
				sgs = append(sgs, spec)
				continue LOOPNAMES
			}
		}
		return nil, ErrNoSecurityGroupFound
	}
	return sgs, nil
}

// GetSecurityGroupsFor returns the security groups having names or a role.
func (s *ClusterScope) GetSecurityGroupsFor(names []infrastructurev1beta1.OscSecurityGroupElement, role infrastructurev1beta1.OscRole) ([]infrastructurev1beta1.OscSecurityGroup, error) {
	if len(names) > 0 {
		return s.getSecurityGroupsForNames(names)
	}
	var sgs []infrastructurev1beta1.OscSecurityGroup
	for _, spec := range s.GetSecurityGroups() {
		if spec.HasRole(role) {
			sgs = append(sgs, spec)
		}
	}
	return sgs, nil
}

// GetSecurityGroupName returns the SecurityGroupName attribute value for a security group.
func (s *ClusterScope) GetSecurityGroupName(sg infrastructurev1beta1.OscSecurityGroup) string {
	if sg.Name != "" {
		return sg.Name + "-" + s.GetUID()
	}
	name := s.GetName() + "-"
	for _, role := range sg.Roles {
		name += string(role) + "-"
	}
	return name + s.GetUID()
}

// SetFailureDomain sets the infrastructure provider failure domain key to the spec given as input.
func (s *ClusterScope) SetFailureDomain(id string, spec clusterv1.FailureDomainSpec) {
	if s.OscCluster.Status.FailureDomains == nil {
		s.OscCluster.Status.FailureDomains = make(clusterv1.FailureDomains)
	}
	s.OscCluster.Status.FailureDomains[id] = spec
}

// GetLoadBalancer return the loadbalanacer of the cluster
func (s *ClusterScope) GetLoadBalancer() infrastructurev1beta1.OscLoadBalancer {
	lb := s.OscCluster.Spec.Network.LoadBalancer
	if lb.LoadBalancerName == "" {
		lb.LoadBalancerName = s.GetName() + "-k8s"
	}
	lb.SetDefaultValue()
	return lb
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

// GetPublicIp return the public ip of the cluster
func (s *ClusterScope) GetPublicIp() []*infrastructurev1beta1.OscPublicIp {
	return s.OscCluster.Spec.Network.PublicIps
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
func (s *ClusterScope) GetBastion() infrastructurev1beta1.OscBastion {
	if !s.OscCluster.Spec.Network.Bastion.Enable {
		return infrastructurev1beta1.OscBastion{}
	}
	bastionSpec := s.OscCluster.Spec.Network.Bastion
	bastionSpec.SetDefaultValue()
	return bastionSpec
}

// GetBastionName return the name of the bastion
func (s *ClusterScope) GetBastionName() string {
	if s.OscCluster.Spec.Network.Bastion.Name != "" {
		return s.OscCluster.Spec.Network.Bastion.Name
	}
	return "Bastion for " + s.GetName()
}

func (s *ClusterScope) GetBastionClientToken() string {
	if s.OscCluster.Spec.Network.Bastion.Name != "" {
		ct := s.OscCluster.Spec.Network.Bastion.Name + "-" + s.GetUID()
		if len(ct) > 64 {
			ct = ct[len(ct)-64:]
		}
		return ct
	}
	return "bastion-" + s.GetUID()
}

// GetBastionPrivateIps return the bastion privateIps
func (s *ClusterScope) GetBastionPrivateIps() []infrastructurev1beta1.OscPrivateIpElement {
	return s.GetBastion().PrivateIps
}

// GetBastionSecurityGroups return the bastion securityGroups
func (s *ClusterScope) GetBastionSecurityGroups() []infrastructurev1beta1.OscSecurityGroupElement {
	return s.GetBastion().SecurityGroupNames
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
