package v1beta1

import (
	"errors"
	"slices"
	"strings"

	infrastructurev1beta2 "github.com/outscale/cluster-api-provider-outscale/api/v1beta2"
	"github.com/samber/lo"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

func ConvertRulesTo(src []OscSecurityGroupRule) []infrastructurev1beta2.OscSecurityGroupRule {
	return lo.Map(src, func(src OscSecurityGroupRule, _ int) infrastructurev1beta2.OscSecurityGroupRule {
		return infrastructurev1beta2.OscSecurityGroupRule{
			Name:     src.Name,
			Flow:     ConvertFlowTo(src.Flow),
			Ports:    []infrastructurev1beta2.Port{infrastructurev1beta2.BuildPort(src.IpProtocol, src.FromPortRange, src.ToPortRange)},
			IpRanges: src.GetIpRanges(),
		}
	})
}

func ConvertRulesFrom(srcs []infrastructurev1beta2.OscSecurityGroupRule) ([]OscSecurityGroupRule, error) {
	dst := make([]OscSecurityGroupRule, 0, len(srcs))
	for _, src := range srcs {
		for _, p := range src.Ports {
			proto, fromPort, toPort, err := p.Parse()
			if err != nil {
				return nil, err
			}
			dst = append(dst, OscSecurityGroupRule{
				Name:          src.Name,
				Flow:          ConvertFlowFrom(src.Flow),
				IpProtocol:    proto,
				FromPortRange: int32(fromPort), //nolint
				ToPortRange:   int32(toPort),   //nolint
				IpRanges:      src.IpRanges,
			})
		}
	}
	return dst, nil
}

func ConvertFlowTo(src string) infrastructurev1beta2.Flow {
	if strings.ToLower(src) == "outbound" {
		return infrastructurev1beta2.FlowOutbound
	}
	return infrastructurev1beta2.FlowInbound
}

func ConvertFlowFrom(src infrastructurev1beta2.Flow) string {
	return string(src)
}

func (src *OscClusterSpec) ConvertTo(dst *infrastructurev1beta2.OscClusterSpec) error {
	if len(src.Network.NatServices) > lo.CountBy(src.Network.Subnets, func(sn OscSubnet) bool {
		return slices.Contains(sn.Roles, RoleNat)
	}) {
		return errors.New(`there are fewer subnets with a "nat" role than NAT services`)
	}
	srcNet := src.Network
	*dst = infrastructurev1beta2.OscClusterSpec{
		ControlPlaneEndpoint: src.ControlPlaneEndpoint,

		Credentials: infrastructurev1beta2.OscCredentials(src.Credentials),
		UseExisting: infrastructurev1beta2.OscReuse(srcNet.UseExisting),
		Disable: infrastructurev1beta2.OscDisable{
			Internet:     lo.Contains(srcNet.Disable, DisableInternet),
			Loadbalancer: lo.Contains(srcNet.Disable, DisableLB),
		},
		LoadBalancer: infrastructurev1beta2.OscLoadBalancer{
			LoadBalancerName:  srcNet.LoadBalancer.LoadBalancerName,
			LoadBalancerType:  srcNet.LoadBalancer.LoadBalancerType,
			SubnetName:        srcNet.LoadBalancer.SubnetName,
			SecurityGroupName: srcNet.LoadBalancer.SecurityGroupName,
			Listener:          infrastructurev1beta2.OscLoadBalancerListener(srcNet.LoadBalancer.Listener),
			HealthCheck:       infrastructurev1beta2.OscLoadBalancerHealthCheck(srcNet.LoadBalancer.HealthCheck),
		},
		Net: infrastructurev1beta2.OscNet{
			Name:       srcNet.Net.Name,
			IpRange:    srcNet.Net.IpRange,
			ResourceId: srcNet.Net.ResourceId,
		},
		NetPeering: infrastructurev1beta2.OscNetPeering{
			Enable:                srcNet.NetPeering.Enable,
			ManagementCredentials: infrastructurev1beta2.OscCredentials(srcNet.NetPeering.ManagementCredentials),
			ManagementAccountID:   srcNet.NetPeering.ManagementAccountID,
			ManagementNetID:       srcNet.NetPeering.ManagementNetID,
			ManagementSubnetID:    srcNet.NetPeering.ManagementSubnetID,
		},
		NetAccessPoints: lo.Map(srcNet.NetAccessPoints, func(src OscNetAccessPointService, _ int) infrastructurev1beta2.OscNetAccessPointService {
			return infrastructurev1beta2.OscNetAccessPointService(src)
		}),
		ControlPlaneSubnets: srcNet.ControlPlaneSubnets,
		Subnets: lo.Map(srcNet.Subnets, func(src OscSubnet, _ int) infrastructurev1beta2.OscSubnet {
			return infrastructurev1beta2.OscSubnet{
				Name: src.Name,
				Roles: lo.Map(src.Roles, func(src OscRole, _ int) infrastructurev1beta2.OscRole {
					return infrastructurev1beta2.OscRole(src)
				}),
				IpSubnetRange: src.IpSubnetRange,
				SubregionName: src.SubregionName,
				ResourceId:    src.ResourceId,
			}
		}),
		InternetService: infrastructurev1beta2.OscInternetService{
			Name: srcNet.InternetService.Name,
		},
		NatPublicIpPool: srcNet.NatPublicIpPool,
		RouteTables: lo.Map(srcNet.RouteTables, func(src OscRouteTable, _ int) infrastructurev1beta2.OscRouteTable {
			return infrastructurev1beta2.OscRouteTable{
				Name:          src.Name,
				Subnets:       src.Subnets,
				Role:          infrastructurev1beta2.OscRole(src.Role),
				SubregionName: src.SubregionName,
				Routes: lo.Map(src.Routes, func(src OscRoute, _ int) infrastructurev1beta2.OscRoute {
					return infrastructurev1beta2.OscRoute(src)
				}),
			}
		}),
		SecurityGroups: lo.Map(srcNet.SecurityGroups, func(src OscSecurityGroup, _ int) infrastructurev1beta2.OscSecurityGroup {
			return infrastructurev1beta2.OscSecurityGroup{
				Name:               src.Name,
				Description:        src.Description,
				SecurityGroupRules: ConvertRulesTo(src.SecurityGroupRules),
				ResourceId:         src.ResourceId,
				Roles:              lo.Map(src.Roles, func(src OscRole, _ int) infrastructurev1beta2.OscRole { return infrastructurev1beta2.OscRole(src) }),
				Tag:                src.Tag,
				Authoritative:      src.Authoritative,
			}
		}),
		AdditionalSecurityRules: lo.Map(srcNet.AdditionalSecurityRules, func(src OscAdditionalSecurityRules, _ int) infrastructurev1beta2.OscAdditionalSecurityRules {
			return infrastructurev1beta2.OscAdditionalSecurityRules{
				Roles: lo.Map(src.Roles, func(src OscRole, _ int) infrastructurev1beta2.OscRole {
					return infrastructurev1beta2.OscRole(src)
				}),
				Rules: ConvertRulesTo(src.Rules),
			}
		}),
		Bastion: infrastructurev1beta2.OscBastion{
			Name:           srcNet.Bastion.Name,
			ImageId:        srcNet.Bastion.ImageId,
			ImageName:      srcNet.Bastion.ImageName,
			ImageAccountId: srcNet.Bastion.ImageAccountId,
			KeypairName:    srcNet.Bastion.KeypairName,
			VmType:         srcNet.Bastion.VmType,
			SubnetName:     srcNet.Bastion.SubnetName,
			RootDisk:       infrastructurev1beta2.OscRootDisk(srcNet.Bastion.RootDisk),
			PublicIpId:     srcNet.Bastion.PublicIpId,
			SecurityGroupNames: lo.Map(srcNet.Bastion.SecurityGroupNames, func(src OscSecurityGroupElement, _ int) infrastructurev1beta2.OscSecurityGroupElement {
				return infrastructurev1beta2.OscSecurityGroupElement(src)
			}),
			Enable: srcNet.Bastion.Enable,
		},
		SubregionName:     srcNet.SubregionName,
		Subregions:        srcNet.Subregions,
		AllowFromIPRanges: srcNet.AllowFromIPRanges, // The list of IP ranges (in CIDR notation) the nodes can talk to ("0.0.0.0/0" if not set).
		AllowToIPRanges:   srcNet.AllowToIPRanges,
		ReconciliationRules: lo.Map(srcNet.ReconciliationRules, func(src OscReconciliationRule, _ int) infrastructurev1beta2.OscReconciliationRule {
			return infrastructurev1beta2.OscReconciliationRule{
				AppliesTo: lo.Map(src.AppliesTo, func(src Reconciler, _ int) infrastructurev1beta2.Reconciler {
					return infrastructurev1beta2.Reconciler(src)
				}),
				Mode:                 infrastructurev1beta2.ReconciliationMode(src.Mode),
				ReconciliationChance: src.ReconciliationChance,
			}
		}),
	}
	return nil
}

func (dst *OscClusterSpec) ConvertFrom(src *infrastructurev1beta2.OscClusterSpec) error {
	var err error
	dst.ControlPlaneEndpoint = src.ControlPlaneEndpoint
	dst.Credentials = OscCredentials(src.Credentials)
	dst.Network = OscNetwork{
		UseExisting: OscReuse(src.UseExisting),
		LoadBalancer: OscLoadBalancer{
			LoadBalancerName:  src.LoadBalancer.LoadBalancerName,
			LoadBalancerType:  src.LoadBalancer.LoadBalancerType,
			SubnetName:        src.LoadBalancer.SubnetName,
			SecurityGroupName: src.LoadBalancer.SecurityGroupName,
			Listener:          OscLoadBalancerListener(src.LoadBalancer.Listener),
			HealthCheck:       OscLoadBalancerHealthCheck(src.LoadBalancer.HealthCheck),
		},
		Net: OscNet{
			Name:       src.Net.Name,
			IpRange:    src.Net.IpRange,
			ResourceId: src.Net.ResourceId,
		},
		NetPeering: OscNetPeering{
			Enable:                src.NetPeering.Enable,
			ManagementCredentials: OscCredentials(src.NetPeering.ManagementCredentials),
			ManagementAccountID:   src.NetPeering.ManagementAccountID,
			ManagementNetID:       src.NetPeering.ManagementNetID,
			ManagementSubnetID:    src.NetPeering.ManagementSubnetID,
		},
		NetAccessPoints: lo.Map(src.NetAccessPoints, func(src infrastructurev1beta2.OscNetAccessPointService, _ int) OscNetAccessPointService {
			return OscNetAccessPointService(src)
		}),
		ControlPlaneSubnets: src.ControlPlaneSubnets,
		Subnets: lo.Map(src.Subnets, func(src infrastructurev1beta2.OscSubnet, _ int) OscSubnet {
			return OscSubnet{
				Name: src.Name,
				Roles: lo.Map(src.Roles, func(src infrastructurev1beta2.OscRole, _ int) OscRole {
					return OscRole(src)
				}),
				IpSubnetRange: src.IpSubnetRange,
				SubregionName: src.SubregionName,
				ResourceId:    src.ResourceId,
			}
		}),
		InternetService: OscInternetService{
			Name: src.InternetService.Name,
		},
		NatServices: lo.FilterMap(src.Subnets, func(src infrastructurev1beta2.OscSubnet, _ int) (OscNatService, bool) {
			if !slices.Contains(src.Roles, infrastructurev1beta2.RoleNat) {
				return OscNatService{}, false
			}
			return OscNatService{
				SubnetName:    src.Name,
				SubregionName: src.SubregionName,
			}, true
		}),
		NatPublicIpPool: src.NatPublicIpPool,
		RouteTables: lo.Map(src.RouteTables, func(src infrastructurev1beta2.OscRouteTable, _ int) OscRouteTable {
			return OscRouteTable{
				Name:          src.Name,
				Subnets:       src.Subnets,
				Role:          OscRole(src.Role),
				SubregionName: src.SubregionName,
				Routes: lo.Map(src.Routes, func(src infrastructurev1beta2.OscRoute, _ int) OscRoute {
					return OscRoute(src)
				}),
			}
		}),
		SecurityGroups: lo.Map(src.SecurityGroups, func(src infrastructurev1beta2.OscSecurityGroup, _ int) OscSecurityGroup {
			rules, rerr := ConvertRulesFrom(src.SecurityGroupRules)
			if rerr != nil {
				err = rerr
			}
			return OscSecurityGroup{
				Name:               src.Name,
				Description:        src.Description,
				SecurityGroupRules: rules,
				ResourceId:         src.ResourceId,
				Roles:              lo.Map(src.Roles, func(src infrastructurev1beta2.OscRole, _ int) OscRole { return OscRole(src) }),
				Tag:                src.Tag,
				Authoritative:      src.Authoritative,
			}
		}),
		AdditionalSecurityRules: lo.Map(src.AdditionalSecurityRules, func(src infrastructurev1beta2.OscAdditionalSecurityRules, _ int) OscAdditionalSecurityRules {
			rules, rerr := ConvertRulesFrom(src.Rules)
			if rerr != nil {
				err = rerr
			}
			return OscAdditionalSecurityRules{
				Roles: lo.Map(src.Roles, func(src infrastructurev1beta2.OscRole, _ int) OscRole {
					return OscRole(src)
				}),
				Rules: rules,
			}
		}),
		Bastion: OscBastion{
			Name:           src.Bastion.Name,
			ImageId:        src.Bastion.ImageId,
			ImageName:      src.Bastion.ImageName,
			ImageAccountId: src.Bastion.ImageAccountId,
			KeypairName:    src.Bastion.KeypairName,
			VmType:         src.Bastion.VmType,
			SubnetName:     src.Bastion.SubnetName,
			RootDisk:       OscRootDisk(src.Bastion.RootDisk),
			PublicIpId:     src.Bastion.PublicIpId,
			SecurityGroupNames: lo.Map(src.Bastion.SecurityGroupNames, func(src infrastructurev1beta2.OscSecurityGroupElement, _ int) OscSecurityGroupElement {
				return OscSecurityGroupElement(src)
			}),
			Enable: src.Bastion.Enable,
		},
		SubregionName:     src.SubregionName,
		Subregions:        src.Subregions,
		AllowFromIPRanges: src.AllowFromIPRanges, // The list of IP ranges (in CIDR notation) the nodes can talk to ("0.0.0.0/0" if not set).
		AllowToIPRanges:   src.AllowToIPRanges,
		ReconciliationRules: lo.Map(src.ReconciliationRules, func(src infrastructurev1beta2.OscReconciliationRule, _ int) OscReconciliationRule {
			return OscReconciliationRule{
				AppliesTo: lo.Map(src.AppliesTo, func(src infrastructurev1beta2.Reconciler, _ int) Reconciler {
					return Reconciler(src)
				}),
				Mode:                 ReconciliationMode(src.Mode),
				ReconciliationChance: src.ReconciliationChance,
			}
		}),
	}
	if src.Disable.Internet {
		dst.Network.Disable = append(dst.Network.Disable, DisableInternet)
	}
	if src.Disable.Loadbalancer {
		dst.Network.Disable = append(dst.Network.Disable, DisableLB)
	}
	return err
}

func (src *OscCluster) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*infrastructurev1beta2.OscCluster)
	dst.ObjectMeta = src.ObjectMeta
	dst.Status = infrastructurev1beta2.OscClusterStatus{
		Ready:     src.Status.Ready,
		Resources: infrastructurev1beta2.OscClusterResources(src.Status.Resources),
		ReconcilerGeneration: lo.MapEntries(src.Status.ReconcilerGeneration, func(k Reconciler, v int64) (infrastructurev1beta2.Reconciler, int64) {
			return infrastructurev1beta2.Reconciler(k), v
		}),
		FailureDomains: src.Status.FailureDomains,
		Conditions:     src.Status.Conditions,
		VmState:        src.Status.VmState,
	}
	if err := src.Spec.ConvertTo(&dst.Spec); err != nil {
		return err
	}
	restored := &infrastructurev1beta2.OscCluster{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	dst.Spec.Keypair = restored.Spec.Keypair
	return nil
}

func (dst *OscCluster) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*infrastructurev1beta2.OscCluster)
	dst.ObjectMeta = src.ObjectMeta
	dst.Status = OscClusterStatus{
		Ready:     src.Status.Ready,
		Resources: OscClusterResources(src.Status.Resources),
		ReconcilerGeneration: lo.MapEntries(src.Status.ReconcilerGeneration, func(k infrastructurev1beta2.Reconciler, v int64) (Reconciler, int64) {
			return Reconciler(k), v
		}),
		FailureDomains: src.Status.FailureDomains,
		Conditions:     src.Status.Conditions,
		VmState:        src.Status.VmState,
	}
	if err := dst.Spec.ConvertFrom(&src.Spec); err != nil {
		return err
	}
	return utilconversion.MarshalData(src, dst)
}

var _ conversion.Convertible = (*OscCluster)(nil)

func (src *OscClusterList) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*infrastructurev1beta2.OscClusterList)
	dst.Items = make([]infrastructurev1beta2.OscCluster, len(src.Items))
	for i := range src.Items {
		if err := src.Items[i].ConvertTo(&dst.Items[i]); err != nil {
			return err
		}
	}
	return nil
}

func (dst *OscClusterList) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*infrastructurev1beta2.OscClusterList)
	dst.Items = make([]OscCluster, len(src.Items))
	for i := range src.Items {
		if err := dst.Items[i].ConvertFrom(&src.Items[i]); err != nil {
			return err
		}
	}
	return nil
}

var _ conversion.Convertible = (*OscClusterList)(nil)
