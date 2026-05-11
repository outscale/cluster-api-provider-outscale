/*
SPDX-FileCopyrightText: 2022 The Kubernetes Authors

SPDX-License-Identifier: Apache-2.0
*/

package scope

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"net"
	"slices"

	infrastructurev1beta2 "github.com/outscale/cluster-api-provider-outscale/api/v1beta2"
	"github.com/outscale/cluster-api-provider-outscale/cloud/tenant"
	"github.com/outscale/goutils/sdk/ptr"
	"github.com/outscale/osc-sdk-go/v3/pkg/osc"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const clusterUIDLabel = "outscale.com/clusterUID"

// ClusterScopeParams is a collection of input parameters to create a new scope
type ClusterScopeParams struct {
	Client     client.Client
	Cluster    *clusterv1.Cluster
	OscCluster *infrastructurev1beta2.OscCluster
	Tenant     tenant.Tenant
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
		patchHelper: helper,
		Tenant:      params.Tenant,
	}, nil
}

// ClusterScope is the basic context of the actuator that will be used
type ClusterScope struct {
	Client      client.Client
	patchHelper *patch.Helper
	Cluster     *clusterv1.Cluster
	OscCluster  *infrastructurev1beta2.OscCluster
	Tenant      tenant.Tenant
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
func (s *ClusterScope) GetRegion() string {
	return s.Tenant.Region()
}

// GetSpec return the network of the cluster
func (s *ClusterScope) GetSpec() *infrastructurev1beta2.OscClusterSpec {
	return &s.OscCluster.Spec
}

// GetNet return the net of the cluster
func (s *ClusterScope) GetNet() infrastructurev1beta2.OscNet {
	if s.OscCluster.Spec.Net.IsZero() {
		return infrastructurev1beta2.DefaultNet
	}
	return s.OscCluster.Spec.Net
}

func (s *ClusterScope) GetNetName() string {
	return "Net for " + s.OscCluster.Name
}

// GetDefaultSubregion returns the default subregion.
func (s *ClusterScope) GetDefaultSubregion() string {
	if len(s.GetSpec().Subregions) > 0 {
		return s.GetSpec().Subregions[0]
	}
	// TODO: ???
	return ""
}

// GetSubregions returns the subregions where to deploy the cluster.
func (s *ClusterScope) GetSubregions() []string {
	return s.GetSpec().Subregions
}

// GetSubnets returns the subnets of the cluster.
func (s *ClusterScope) GetSubnets() []infrastructurev1beta2.OscSubnet {
	if len(s.OscCluster.Spec.Subnets) > 0 {
		return s.OscCluster.Spec.Subnets
	}
	_, net, err := net.ParseCIDR(s.GetNet().IpRange)
	if err != nil {
		return nil
	}
	fds := s.GetSubregions()
	subnets := make([]infrastructurev1beta2.OscSubnet, 0, 3*len(fds))
	net.IP[2]++
	net.Mask[1], net.Mask[2] = 255, 255
	for _, fd := range fds {
		for _, roles := range [][]infrastructurev1beta2.OscRole{
			{infrastructurev1beta2.RoleLoadBalancer, infrastructurev1beta2.RoleBastion, infrastructurev1beta2.RoleNat},
			{infrastructurev1beta2.RoleWorker},
			{infrastructurev1beta2.RoleControlPlane},
		} {
			net.IP[2]++
			subnet := infrastructurev1beta2.OscSubnet{
				IpRange: net.String(),
				Roles:         roles,
				Subregion: fd,
			}
			subnets = append(subnets, subnet)
		}
	}
	return subnets
}

var ErrNoSubnetFound = errors.New("subnet not found")

func (s *ClusterScope) GetSubnet(role infrastructurev1beta2.OscRole, subregion string) (infrastructurev1beta2.OscSubnet, error) {
	if subregion == "" {
		subregion = s.GetDefaultSubregion()
	}
	for _, spec := range s.GetSubnets() {
		switch {
		case !s.SubnetHasRole(spec, role):
		case s.GetSubnetSubregion(spec) == subregion:
			return spec, nil
		}
	}
	return infrastructurev1beta2.OscSubnet{}, ErrNoSubnetFound
}

func (s *ClusterScope) SubnetHasRole(spec infrastructurev1beta2.OscSubnet, role infrastructurev1beta2.OscRole) bool {
	if len(spec.Roles) > 0 {
		return slices.Contains(spec.Roles, role)
	}
	return role == infrastructurev1beta2.RoleWorker
}

func (s *ClusterScope) SubnetIsPublic(spec infrastructurev1beta2.OscSubnet) bool {
	return s.SubnetHasRole(spec, infrastructurev1beta2.RoleBastion) || s.SubnetHasRole(spec, infrastructurev1beta2.RoleLoadBalancer) || s.SubnetHasRole(spec, infrastructurev1beta2.RoleNat)
}

func (s *ClusterScope) GetSubnetSubregion(spec infrastructurev1beta2.OscSubnet) string {
	if spec.Subregion != "" {
		return spec.Subregion
	}
	return s.GetDefaultSubregion()
}

func (s *ClusterScope) GetSubnetName(spec infrastructurev1beta2.OscSubnet) string {
	if spec.Name != "" {
		return spec.Name
	}
	fd := s.GetSubnetSubregion(spec)
	switch {
	case s.SubnetIsPublic(spec):
		return "Public subnet for " + s.OscCluster.Name + "/" + fd
	case s.SubnetHasRole(spec, infrastructurev1beta2.RoleControlPlane):
		return "Controlplane subnet for " + s.OscCluster.Name + "/" + fd
	case s.SubnetHasRole(spec, infrastructurev1beta2.RoleWorker):
		return "Worker subnet for " + s.OscCluster.Name + "/" + fd
	default:
		return "Subnet for " + s.OscCluster.Name + "/" + fd
	}
}

// IsInternetDisabled checks if internet is disabled.
func (s *ClusterScope) IsInternetDisabled() bool {
	return slices.Contains(s.GetSpec().Disable, infrastructurev1beta2.DisableInternet)
}

// IsLBDisabled checks if loadbalancer is disabled.
func (s *ClusterScope) IsLBDisabled() bool {
	return slices.Contains(s.GetSpec().Disable, infrastructurev1beta2.DisableLB)
}

// GetInternetServiceName return the name of the net
func (s *ClusterScope) GetInternetServiceName() string {
	return "Internet Service for " + s.OscCluster.Name
}

var ErrNoNatFound = errors.New("natService not found")

// GetNatServices return the natServices of the cluster
func (s *ClusterScope) GetNatServices() []infrastructurev1beta2.OscNatService {
	if s.IsInternetDisabled() {
		return nil
	}
	switch {
	case s.OscCluster.Spec.UseExisting.Net:
		return nil
	case len(s.OscCluster.Spec.NatServices) > 0:
		return s.OscCluster.Spec.NatServices
	case s.OscCluster.Spec.NatService != infrastructurev1beta2.OscNatService{}:
		return []infrastructurev1beta2.OscNatService{s.OscCluster.Spec.NatService}
	default:
		var nss []infrastructurev1beta2.OscNatService
		for _, subnet := range s.GetSubnets() {
			if !s.SubnetHasRole(subnet, infrastructurev1beta2.RoleNat) {
				continue
			}
			nss = append(nss, infrastructurev1beta2.OscNatService{
				SubregionName: s.GetSubnetSubregion(subnet),
				SubnetName:    subnet.Name,
			})
		}
		return nss
	}
}

// GetNatService return the natService of the cluster
func (s *ClusterScope) GetNatService(name string, subregion string) (infrastructurev1beta2.OscNatService, error) {
	nats := s.GetNatServices()
	if name != "" {
		for _, spec := range nats {
			if spec.Name == name {
				return spec, nil
			}
		}
		return infrastructurev1beta2.OscNatService{}, ErrNoNatFound
	}
	if len(nats) == 1 {
		return nats[0], nil
	}
	for _, spec := range nats {
		if spec.SubregionName == "" {
			spec.SubregionName = s.GetDefaultSubregion()
		}
		if spec.SubregionName == subregion || subregion == "" {
			return spec, nil
		}
	}
	return infrastructurev1beta2.OscNatService{}, ErrNoNatFound
}

// GetNatServiceName return the name of a nat service
func (s *ClusterScope) GetNatServiceName(nat infrastructurev1beta2.OscNatService) string {
	if nat.Name != "" {
		return nat.Name
	}
	name := "Nat service for " + s.OscCluster.Name
	if nat.SubregionName != "" {
		name += "/" + nat.SubregionName
	}
	return name
}

// GetNatServiceClientToken return the client token for a nat service
func (s *ClusterScope) GetNatServiceClientToken(nat infrastructurev1beta2.OscNatService) string {
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
func (s *ClusterScope) GetRouteTables() []infrastructurev1beta2.OscRouteTable {
	if len(s.OscCluster.Spec.RouteTables) > 0 {
		return s.OscCluster.Spec.RouteTables
	}
	subnets := s.GetSubnets()
	rtbls := make([]infrastructurev1beta2.OscRouteTable, 0, len(subnets))
	for _, subnet := range subnets {
		rtbl := infrastructurev1beta2.OscRouteTable{
			Name:          s.GetSubnetName(subnet),
			Subregion: s.GetSubnetSubregion(subnet),
		}
		if subnet.Name != "" {
			rtbl.Subnets = []string{subnet.Name}
		}
		if len(subnet.Roles) > 0 {
			rtbl.Role = subnet.Roles[0]
		}
		switch {
		case s.IsInternetDisabled():
		case s.SubnetIsPublic(subnet):
			rtbl.Routes = []infrastructurev1beta2.OscRoute{{Destination: "0.0.0.0/0", TargetType: "gateway"}}
		default:
			rtbl.Routes = []infrastructurev1beta2.OscRoute{{Destination: "0.0.0.0/0", TargetType: "nat"}}
		}
		rtbls = append(rtbls, rtbl)
	}
	return rtbls
}

// HasIPRestriction returns true if AllowFromIps is set.
func (s *ClusterScope) HasIPRestriction() bool {
	return len(s.OscCluster.Spec.AllowFromIPRanges) > 0
}

func (s *ClusterScope) getAdditionalRules(roles ...infrastructurev1beta2.OscRole) []infrastructurev1beta2.OscSecurityGroupRule {
	for _, ar := range s.GetSpec().AdditionalSecurityRules {
		if slices.Equal(roles, ar.Roles) {
			return ar.Rules
		}
	}
	return nil
}

// GetSecurityGroups returns the list of all security groups for the cluster.
func (s *ClusterScope) GetSecurityGroups() []infrastructurev1beta2.OscSecurityGroup {
	if len(s.OscCluster.Spec.SecurityGroups) > 0 {
		return s.getManualSecurityGroups()
	}
	return s.getAutomaticSecurityGroups()
}

func (s *ClusterScope) getManualSecurityGroups() []infrastructurev1beta2.OscSecurityGroup {
	allowedIn := s.OscCluster.Spec.AllowFromIPRanges
	allowedOut := s.OscCluster.Spec.AllowToIPRanges
	if len(allowedIn) == 0 && len(allowedOut) == 0 {
		return s.OscCluster.Spec.SecurityGroups
	}
	sgs := slices.Clone(s.OscCluster.Spec.SecurityGroups)
	for i := range sgs {
		sgs[i].SecurityGroupRules = slices.Clone(sgs[i].SecurityGroupRules)
	}
	if len(allowedIn) > 0 {
		for i := range sgs {
			if slices.Contains(sgs[i].Roles, infrastructurev1beta2.RoleLoadBalancer) {
				sgs[i].SecurityGroupRules = append(sgs[i].SecurityGroupRules,
					infrastructurev1beta2.OscSecurityGroupRule{Flow: "Inbound", IpProtocol: "tcp", FromPortRange: 6443, ToPortRange: 6443, IpRanges: allowedIn},
				)
			}
			if slices.Contains(sgs[i].Roles, infrastructurev1beta2.RoleBastion) {
				sgs[i].SecurityGroupRules = append(sgs[i].SecurityGroupRules,
					infrastructurev1beta2.OscSecurityGroupRule{Flow: "Inbound", IpProtocol: "tcp", FromPortRange: 22, ToPortRange: 22, IpRanges: allowedIn},
				)
			}
		}
	}
	if len(allowedOut) > 0 {
		for i := range sgs {
			if (slices.Contains(sgs[i].Roles, infrastructurev1beta2.RoleWorker) &&
				slices.Contains(sgs[i].Roles, infrastructurev1beta2.RoleControlPlane)) ||
				slices.Contains(sgs[i].Roles, infrastructurev1beta2.RoleBastion) {
				sgs[i].SecurityGroupRules = append(sgs[i].SecurityGroupRules,
					infrastructurev1beta2.OscSecurityGroupRule{Flow: "Outbound", IpProtocol: "-1", FromPortRange: -1, ToPortRange: -1, IpRanges: allowedOut},
				)
			}
		}
	}
	return sgs
}

func (s *ClusterScope) getAutomaticSecurityGroups() []infrastructurev1beta2.OscSecurityGroup {
	var allSN, allSNCP, allSNBastion []string
	for _, sn := range s.GetSubnets() {
		if s.SubnetHasRole(sn, infrastructurev1beta2.RoleBastion) {
			allSNBastion = append(allSNBastion, sn.IpRange)
		}
		if s.SubnetIsPublic(sn) {
			continue
		}
		if s.SubnetHasRole(sn, infrastructurev1beta2.RoleControlPlane) {
			allSNCP = append(allSNCP, sn.IpRange)
		}
		allSN = append(allSN, sn.IpRange)
	}
	allowedIn := s.OscCluster.Spec.AllowFromIPRanges
	if len(allowedIn) == 0 {
		allowedIn = []string{"0.0.0.0/0"}
	}
	lb := infrastructurev1beta2.OscSecurityGroup{
		Name:        s.GetName() + "-lb",
		Description: "LB securityGroup for " + s.GetName(),
		Roles:       []infrastructurev1beta2.OscRole{infrastructurev1beta2.RoleLoadBalancer},
		SecurityGroupRules: []infrastructurev1beta2.OscSecurityGroupRule{
			{Flow: "Inbound", IpProtocol: "tcp", FromPortRange: 6443, ToPortRange: 6443, IpRanges: allowedIn},
			{Flow: "Outbound", IpProtocol: "tcp", FromPortRange: 6443, ToPortRange: 6443, IpRanges: allSNCP},
		},
		Authoritative: true,
	}
	lb.SecurityGroupRules = append(lb.SecurityGroupRules, s.getAdditionalRules(infrastructurev1beta2.RoleLoadBalancer)...)

	worker := infrastructurev1beta2.OscSecurityGroup{
		Name:        s.GetName() + "-worker",
		Description: "Worker securityGroup for " + s.GetName(),
		Roles:       []infrastructurev1beta2.OscRole{infrastructurev1beta2.RoleWorker},
		SecurityGroupRules: []infrastructurev1beta2.OscSecurityGroupRule{
			{Flow: "Inbound", IpProtocol: "tcp", FromPortRange: 443, ToPortRange: 443, IpRanges: allSNCP},    // HTTPS
			{Flow: "Inbound", IpProtocol: "tcp", FromPortRange: 1024, ToPortRange: 65535, IpRanges: allSNCP}, // Applicative ports (services, ...)
			{Flow: "Inbound", IpProtocol: "tcp", FromPortRange: 10250, ToPortRange: 10250, IpRanges: allSN},  // Kubelet
		},
		Authoritative: true,
	}
	worker.SecurityGroupRules = append(worker.SecurityGroupRules, s.getAdditionalRules(infrastructurev1beta2.RoleWorker)...)

	controlplane := infrastructurev1beta2.OscSecurityGroup{
		Name:        s.GetName() + "-controlplane",
		Description: "Controlplane securityGroup for " + s.GetName(),
		Roles:       []infrastructurev1beta2.OscRole{infrastructurev1beta2.RoleControlPlane},
		SecurityGroupRules: []infrastructurev1beta2.OscSecurityGroupRule{
			{Flow: "Inbound", IpProtocol: "tcp", FromPortRange: 6443, ToPortRange: 6443, IpRange: s.GetNet().IpRange}, // API
			{Flow: "Inbound", IpProtocol: "tcp", FromPortRange: 2378, ToPortRange: 2380, IpRanges: allSNCP},           // etcd
			{Flow: "Inbound", IpProtocol: "tcp", FromPortRange: 10250, ToPortRange: 10252, IpRanges: allSNCP},         // Kubelet
		},
		Authoritative: true,
	}
	controlplane.SecurityGroupRules = append(controlplane.SecurityGroupRules, s.getAdditionalRules(infrastructurev1beta2.RoleControlPlane)...)

	allowedOut := s.OscCluster.Spec.AllowToIPRanges
	switch {
	case len(allowedOut) == 0:
		allowedOut = []string{"0.0.0.0/0"}
	case allowedOut[0] == "":
		allowedOut = nil
	}

	node := infrastructurev1beta2.OscSecurityGroup{
		Name:        s.GetName() + "-node",
		Description: "Node securityGroup for " + s.GetName(),
		Roles:       []infrastructurev1beta2.OscRole{infrastructurev1beta2.RoleControlPlane, infrastructurev1beta2.RoleWorker},
		SecurityGroupRules: []infrastructurev1beta2.OscSecurityGroupRule{
			// Calico - see https://docs.tigera.io/calico/latest/getting-started/kubernetes/requirements#network-requirements
			{Flow: "Inbound", IpProtocol: "tcp", FromPortRange: 179, ToPortRange: 179, IpRange: s.GetNet().IpRange},     // BGP
			{Flow: "Inbound", IpProtocol: "udp", FromPortRange: 4789, ToPortRange: 4789, IpRange: s.GetNet().IpRange},   // VXLAN/flannel
			{Flow: "Inbound", IpProtocol: "udp", FromPortRange: 5473, ToPortRange: 5473, IpRange: s.GetNet().IpRange},   // Typha
			{Flow: "Inbound", IpProtocol: "udp", FromPortRange: 8285, ToPortRange: 8285, IpRange: s.GetNet().IpRange},   // Flannel
			{Flow: "Inbound", IpProtocol: "udp", FromPortRange: 51820, ToPortRange: 51821, IpRange: s.GetNet().IpRange}, // Wiregard
			{Flow: "Inbound", IpProtocol: "4", FromPortRange: -1, ToPortRange: -1, IpRange: s.GetNet().IpRange},         // IP-in-IP

			// Cillium - see https://docs.cilium.io/en/stable/operations/system_requirements/#firewall-rules
			{Flow: "Inbound", IpProtocol: "icmp", FromPortRange: 8, ToPortRange: 8, IpRange: s.GetNet().IpRange},        // ICMP
			{Flow: "Inbound", IpProtocol: "tcp", FromPortRange: 4240, ToPortRange: 4240, IpRange: s.GetNet().IpRange},   // Health
			{Flow: "Inbound", IpProtocol: "tcp", FromPortRange: 4244, ToPortRange: 4244, IpRange: s.GetNet().IpRange},   // Hubble
			{Flow: "Inbound", IpProtocol: "udp", FromPortRange: 8472, ToPortRange: 8472, IpRange: s.GetNet().IpRange},   // VXLAN
			{Flow: "Inbound", IpProtocol: "udp", FromPortRange: 51871, ToPortRange: 51871, IpRange: s.GetNet().IpRange}, // Wiregard

			{Flow: "Inbound", IpProtocol: "tcp", FromPortRange: 30000, ToPortRange: 32767, IpRange: s.GetNet().IpRange}, // NodePort
			{Flow: "Outbound", IpProtocol: "-1", FromPortRange: -1, ToPortRange: -1, IpRange: s.GetNet().IpRange},       // internal trafic
		},
		Tag:           "OscK8sMainSG",
		Authoritative: true,
	}
	// Outbound traffic
	if len(allowedOut) > 0 {
		node.SecurityGroupRules = append(node.SecurityGroupRules,
			infrastructurev1beta2.OscSecurityGroupRule{Flow: "Outbound", IpProtocol: "-1", FromPortRange: -1, ToPortRange: -1, IpRanges: allowedOut},
		)
	}
	node.SecurityGroupRules = append(node.SecurityGroupRules, s.getAdditionalRules(infrastructurev1beta2.RoleControlPlane, infrastructurev1beta2.RoleWorker)...)

	if !s.OscCluster.Spec.Bastion.Enable {
		return []infrastructurev1beta2.OscSecurityGroup{lb, worker, controlplane, node}
	}
	node.SecurityGroupRules = append(node.SecurityGroupRules, infrastructurev1beta2.OscSecurityGroupRule{
		Flow: "Inbound", IpProtocol: "tcp", FromPortRange: 22, ToPortRange: 22, IpRanges: allSNBastion,
	})
	bastion := infrastructurev1beta2.OscSecurityGroup{
		Name:        s.GetName() + "-bastion",
		Description: "Bastion securityGroup for " + s.GetName(),
		Roles:       []infrastructurev1beta2.OscRole{infrastructurev1beta2.RoleBastion},
		SecurityGroupRules: []infrastructurev1beta2.OscSecurityGroupRule{
			{Flow: "Inbound", IpProtocol: "tcp", FromPortRange: 22, ToPortRange: 22, IpRanges: allowedIn},
			{Flow: "Outbound", IpProtocol: "tcp", FromPortRange: 22, ToPortRange: 22, IpRange: s.GetNet().IpRange},
		},
		Authoritative: true,
	}
	// Outbound traffic
	if len(allowedOut) > 0 {
		bastion.SecurityGroupRules = append(bastion.SecurityGroupRules,
			infrastructurev1beta2.OscSecurityGroupRule{Flow: "Outbound", IpProtocol: "-1", FromPortRange: -1, ToPortRange: -1, IpRanges: allowedOut},
		)
	}
	bastion.SecurityGroupRules = append(bastion.SecurityGroupRules, s.getAdditionalRules(infrastructurev1beta2.RoleBastion)...)

	return []infrastructurev1beta2.OscSecurityGroup{lb, worker, controlplane, node, bastion}
}

var ErrNoSecurityGroupFound = errors.New("no security group found")

// getSecurityGroupsForNames returns the security group having names.
func (s *ClusterScope) getSecurityGroupsForNames(names []infrastructurev1beta2.OscSecurityGroupElement) ([]infrastructurev1beta2.OscSecurityGroup, error) {
	var sgs []infrastructurev1beta2.OscSecurityGroup
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
func (s *ClusterScope) GetSecurityGroupsFor(names []infrastructurev1beta2.OscSecurityGroupElement, role infrastructurev1beta2.OscRole) ([]infrastructurev1beta2.OscSecurityGroup, error) {
	if len(names) > 0 {
		return s.getSecurityGroupsForNames(names)
	}
	var sgs []infrastructurev1beta2.OscSecurityGroup
	for _, spec := range s.GetSecurityGroups() {
		if spec.HasRole(role) {
			sgs = append(sgs, spec)
		}
	}
	return sgs, nil
}

// GetSecurityGroupName returns the SecurityGroupName attribute value for a security group.
func (s *ClusterScope) GetSecurityGroupName(sg infrastructurev1beta2.OscSecurityGroup) string {
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
func (s *ClusterScope) SetFailureDomain(name string, controlplane bool) {
	idx := slices.IndexFunc(s.OscCluster.Status.FailureDomains, func(fd clusterv1.FailureDomain) bool {
		return fd.Name == name
	})
	switch {
	case idx == -1:
		s.OscCluster.Status.FailureDomains = append(s.OscCluster.Status.FailureDomains, clusterv1.FailureDomain{
			Name:         name,
			ControlPlane: new(controlplane),
		})
	case controlplane && !ptr.From(s.OscCluster.Status.FailureDomains[idx].ControlPlane):
		s.OscCluster.Status.FailureDomains[idx].ControlPlane = new(true)
	}
}

// GetLoadBalancer return the loadbalanacer of the cluster
func (s *ClusterScope) GetLoadBalancer() infrastructurev1beta2.OscLoadBalancer {
	lb := s.OscCluster.Spec.LoadBalancer
	if lb.LoadBalancerName == "" {
		lb.LoadBalancerName = s.GetName() + "-k8s"
	}
	lb.SetDefaultValue()
	return lb
}

// GetIpSubnetRange return IpSubnetRang from the subnet
func (s *ClusterScope) GetIpSubnetRange(name string) string {
	subnets := s.OscCluster.Spec.Subnets
	for _, subnet := range subnets {
		if subnet.Name == name {
			return subnet.IpRange
		}
	}
	return ""
}

// GetSecurityGroupRule return slices of securityGroupRule asscociated with securityGroup Name
func (s *ClusterScope) GetSecurityGroupRule(name string) []infrastructurev1beta2.OscSecurityGroupRule {
	securityGroups := s.OscCluster.Spec.SecurityGroups
	for _, securityGroup := range securityGroups {
		if securityGroup.Name == name {
			return securityGroup.SecurityGroupRules
		}
	}
	return nil
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
func (s *ClusterScope) GetBastion() *infrastructurev1beta2.OscVm {
	if !s.OscCluster.Spec.Bastion.Enable {
		return nil
	}
	bastionSpec := s.OscCluster.Spec.Bastion
	bastionSpec.SetDefaultValue()
	return bastionSpec
}

// GetBastionName return the name of the bastion
func (s *ClusterScope) GetBastionName() string {
	return "Bastion for " + s.GetName()
}

func (s *ClusterScope) GetBastionClientToken() string {
	return "bastion-" + s.GetUID()
}

// GetBastionPrivateIps return the bastion privateIps
func (s *ClusterScope) GetBastionPrivateIps() []infrastructurev1beta2.OscPrivateIpElement {
	return s.GetBastion().PrivateIps
}

// GetBastionSecurityGroups return the bastion securityGroups
func (s *ClusterScope) GetBastionSecurityGroups() []infrastructurev1beta2.OscSecurityGroupElement {
	return s.GetBastion().SecurityGroupNames
}

// GetImage return the image
func (s *ClusterScope) GetImage() *infrastructurev1beta2.OscImage {
	return &s.OscCluster.Spec.Bastion.VM.Image
}

// SetVmState set vmstate
func (s *ClusterScope) SetVmState(v osc.VmState) {
	s.OscCluster.Status.VmState = &v
}

// SetVmState set vmstate
func (s *ClusterScope) GetVmState() *osc.VmState {
	return s.OscCluster.Status.VmState
}

// GetResources returns the resource list from the OscCluster status.
func (s *ClusterScope) GetResources() *infrastructurev1beta2.OscClusterResources {
	return &s.OscCluster.Status.Resources
}

func (s *ClusterScope) getReconcilationRule(reconciler infrastructurev1beta2.Reconciler) infrastructurev1beta2.OscReconciliationRule {
	for _, r := range s.GetSpec().ReconciliationRules {
		if slices.Contains(r.AppliesTo, infrastructurev1beta2.ReconcilerAll) || slices.Contains(r.AppliesTo, reconciler) {
			return r
		}
	}
	switch reconciler {
	case infrastructurev1beta2.ReconcilerSecurityGroup:
		return infrastructurev1beta2.OscReconciliationRule{
			Mode:                 infrastructurev1beta2.ReconciliationModeRandom,
			ReconciliationChance: 10,
		}
	default:
		return infrastructurev1beta2.OscReconciliationRule{
			Mode: infrastructurev1beta2.ReconciliationModeOnChange,
		}
	}
}

var Rand = func() int {
	return rand.IntN(100)
}

// NeedReconciliation returns true if a reconciler needs to run.
func (s *ClusterScope) NeedReconciliation(reconciler infrastructurev1beta2.Reconciler) bool {
	if s.OscCluster.Status.ReconcilerGeneration == nil {
		return true
	}
	if s.OscCluster.Status.ReconcilerGeneration[reconciler] < s.OscCluster.Generation {
		return true
	}
	r := s.getReconcilationRule(reconciler)
	switch r.Mode {
	case infrastructurev1beta2.ReconciliationModeAlways:
		return true
	case infrastructurev1beta2.ReconciliationModeRandom:
		return Rand() < r.ReconciliationChance
	default:
		return false
	}
}

// SetReconciliationGeneration marks a reconciler as having finished its job for a specific cluster generation.
func (s *ClusterScope) SetReconciliationGeneration(reconciler infrastructurev1beta2.Reconciler) {
	if s.OscCluster.Status.ReconcilerGeneration == nil {
		s.OscCluster.Status.ReconcilerGeneration = map[infrastructurev1beta2.Reconciler]int64{}
	}
	s.OscCluster.Status.ReconcilerGeneration[reconciler] = s.OscCluster.Generation
}

// PatchObject keep the cluster configuration and status
func (s *ClusterScope) PatchObject(ctx context.Context) error {
	setConditions := []clusterv1.ConditionType{
		infrastructurev1beta2.NetReadyCondition,
		infrastructurev1beta2.SubnetsReadyCondition,
		infrastructurev1beta2.LoadBalancerReadyCondition,
	}
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
		ctx,
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
	machineByOscMachineName := make(map[string]*clusterv1.Machine)
	if err := s.Client.List(ctx, &machineListRaw, client.InNamespace(s.GetNamespace())); err != nil {
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
	if err := s.Client.List(ctx, &oscMachineListRaw, client.InNamespace(s.GetNamespace())); err != nil {
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
