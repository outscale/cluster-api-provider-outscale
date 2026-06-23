package v1beta1

import (
	infrastructurev1beta2 "github.com/outscale/cluster-api-provider-outscale/api/v1beta2"
	"github.com/samber/lo"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

func (src *OscClusterSpec) ConvertTo(dst *infrastructurev1beta2.OscClusterSpec) error {
	dst.ControlPlaneEndpoint = src.ControlPlaneEndpoint
	dst.Credentials = infrastructurev1beta2.OscCredentials(src.Credentials)
	srcNet := src.Network
	dst.Network = infrastructurev1beta2.OscNetwork{
		UseExisting: infrastructurev1beta2.OscReuse(srcNet.UseExisting),
		Disable: lo.Map(srcNet.Disable, func(src OscDisable, _ int) infrastructurev1beta2.OscDisable {
			return infrastructurev1beta2.OscDisable(src)
		}),
		LoadBalancer: infrastructurev1beta2.OscLoadBalancer{
			LoadBalancerName:  srcNet.LoadBalancer.LoadBalancerName,
			LoadBalancerType:  srcNet.LoadBalancer.LoadBalancerType,
			SubnetName:        srcNet.LoadBalancer.SubnetName,
			SecurityGroupName: srcNet.LoadBalancer.SecurityGroupName,
			Listener:          infrastructurev1beta2.OscLoadBalancerListener(srcNet.LoadBalancer.Listener),
			HealthCheck:       infrastructurev1beta2.OscLoadBalancerHealthCheck(srcNet.LoadBalancer.HealthCheck),
		},
		Net: infrastructurev1beta2.OscNet{
			Name:    srcNet.Net.Name,
			IpRange: srcNet.Net.IpRange,
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
			}
		}),
		InternetService: infrastructurev1beta2.OscInternetService{
			Name: srcNet.InternetService.Name,
		},
		NatService: infrastructurev1beta2.OscNatService{
			Name:          srcNet.NatService.Name,
			SubnetName:    srcNet.NatService.SubnetName,
			SubregionName: srcNet.NatService.SubregionName,
		},
		NatServices: lo.Map(srcNet.NatServices, func(src OscNatService, _ int) infrastructurev1beta2.OscNatService {
			return infrastructurev1beta2.OscNatService{
				Name:          src.Name,
				SubnetName:    src.SubnetName,
				SubregionName: src.SubregionName,
			}
		}),
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
				Name:        src.Name,
				Description: src.Description,
				SecurityGroupRules: lo.Map(src.SecurityGroupRules, func(src OscSecurityGroupRule, _ int) infrastructurev1beta2.OscSecurityGroupRule {
					return infrastructurev1beta2.OscSecurityGroupRule(src)
				}),
			}
		}),
		AdditionalSecurityRules: lo.Map(srcNet.AdditionalSecurityRules, func(src OscAdditionalSecurityRules, _ int) infrastructurev1beta2.OscAdditionalSecurityRules {
			return infrastructurev1beta2.OscAdditionalSecurityRules{
				Roles: lo.Map(src.Roles, func(src OscRole, _ int) infrastructurev1beta2.OscRole {
					return infrastructurev1beta2.OscRole(src)
				}),
				Rules: lo.Map(src.Rules, func(src OscSecurityGroupRule, _ int) infrastructurev1beta2.OscSecurityGroupRule {
					return infrastructurev1beta2.OscSecurityGroupRule(src)
				}),
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
	dst.ControlPlaneEndpoint = src.ControlPlaneEndpoint
	dst.Credentials = OscCredentials(src.Credentials)
	srcNet := src.Network
	dst.Network = OscNetwork{
		UseExisting: OscReuse(srcNet.UseExisting),
		Disable: lo.Map(srcNet.Disable, func(src infrastructurev1beta2.OscDisable, _ int) OscDisable {
			return OscDisable(src)
		}),
		LoadBalancer: OscLoadBalancer{
			LoadBalancerName:  srcNet.LoadBalancer.LoadBalancerName,
			LoadBalancerType:  srcNet.LoadBalancer.LoadBalancerType,
			SubnetName:        srcNet.LoadBalancer.SubnetName,
			SecurityGroupName: srcNet.LoadBalancer.SecurityGroupName,
			Listener:          OscLoadBalancerListener(srcNet.LoadBalancer.Listener),
			HealthCheck:       OscLoadBalancerHealthCheck(srcNet.LoadBalancer.HealthCheck),
		},
		Net: OscNet{
			Name:    srcNet.Net.Name,
			IpRange: srcNet.Net.IpRange,
		},
		NetPeering: OscNetPeering{
			Enable:                srcNet.NetPeering.Enable,
			ManagementCredentials: OscCredentials(srcNet.NetPeering.ManagementCredentials),
			ManagementAccountID:   srcNet.NetPeering.ManagementAccountID,
			ManagementNetID:       srcNet.NetPeering.ManagementNetID,
			ManagementSubnetID:    srcNet.NetPeering.ManagementSubnetID,
		},
		NetAccessPoints: lo.Map(srcNet.NetAccessPoints, func(src infrastructurev1beta2.OscNetAccessPointService, _ int) OscNetAccessPointService {
			return OscNetAccessPointService(src)
		}),
		ControlPlaneSubnets: srcNet.ControlPlaneSubnets,
		Subnets: lo.Map(srcNet.Subnets, func(src infrastructurev1beta2.OscSubnet, _ int) OscSubnet {
			return OscSubnet{
				Name: src.Name,
				Roles: lo.Map(src.Roles, func(src infrastructurev1beta2.OscRole, _ int) OscRole {
					return OscRole(src)
				}),
				IpSubnetRange: src.IpSubnetRange,
				SubregionName: src.SubregionName,
			}
		}),
		InternetService: OscInternetService{
			Name: srcNet.InternetService.Name,
		},
		NatService: OscNatService{
			Name:          srcNet.NatService.Name,
			SubnetName:    srcNet.NatService.SubnetName,
			SubregionName: srcNet.NatService.SubregionName,
		},
		NatServices: lo.Map(srcNet.NatServices, func(src infrastructurev1beta2.OscNatService, _ int) OscNatService {
			return OscNatService{
				Name:          src.Name,
				SubnetName:    src.SubnetName,
				SubregionName: src.SubregionName,
			}
		}),
		NatPublicIpPool: srcNet.NatPublicIpPool,
		RouteTables: lo.Map(srcNet.RouteTables, func(src infrastructurev1beta2.OscRouteTable, _ int) OscRouteTable {
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
		SecurityGroups: lo.Map(srcNet.SecurityGroups, func(src infrastructurev1beta2.OscSecurityGroup, _ int) OscSecurityGroup {
			return OscSecurityGroup{
				Name:        src.Name,
				Description: src.Description,
				SecurityGroupRules: lo.Map(src.SecurityGroupRules, func(src infrastructurev1beta2.OscSecurityGroupRule, _ int) OscSecurityGroupRule {
					return OscSecurityGroupRule(src)
				}),
			}
		}),
		AdditionalSecurityRules: lo.Map(srcNet.AdditionalSecurityRules, func(src infrastructurev1beta2.OscAdditionalSecurityRules, _ int) OscAdditionalSecurityRules {
			return OscAdditionalSecurityRules{
				Roles: lo.Map(src.Roles, func(src infrastructurev1beta2.OscRole, _ int) OscRole {
					return OscRole(src)
				}),
				Rules: lo.Map(src.Rules, func(src infrastructurev1beta2.OscSecurityGroupRule, _ int) OscSecurityGroupRule {
					return OscSecurityGroupRule(src)
				}),
			}
		}),
		Bastion: OscBastion{
			Name:           srcNet.Bastion.Name,
			ImageId:        srcNet.Bastion.ImageId,
			ImageName:      srcNet.Bastion.ImageName,
			ImageAccountId: srcNet.Bastion.ImageAccountId,
			KeypairName:    srcNet.Bastion.KeypairName,
			VmType:         srcNet.Bastion.VmType,
			SubnetName:     srcNet.Bastion.SubnetName,
			RootDisk:       OscRootDisk(srcNet.Bastion.RootDisk),
			PublicIpId:     srcNet.Bastion.PublicIpId,
			SecurityGroupNames: lo.Map(srcNet.Bastion.SecurityGroupNames, func(src infrastructurev1beta2.OscSecurityGroupElement, _ int) OscSecurityGroupElement {
				return OscSecurityGroupElement(src)
			}),
			Enable: srcNet.Bastion.Enable,
		},
		SubregionName:     srcNet.SubregionName,
		Subregions:        srcNet.Subregions,
		AllowFromIPRanges: srcNet.AllowFromIPRanges, // The list of IP ranges (in CIDR notation) the nodes can talk to ("0.0.0.0/0" if not set).
		AllowToIPRanges:   srcNet.AllowToIPRanges,
		ReconciliationRules: lo.Map(srcNet.ReconciliationRules, func(src infrastructurev1beta2.OscReconciliationRule, _ int) OscReconciliationRule {
			return OscReconciliationRule{
				AppliesTo: lo.Map(src.AppliesTo, func(src infrastructurev1beta2.Reconciler, _ int) Reconciler {
					return Reconciler(src)
				}),
				Mode:                 ReconciliationMode(src.Mode),
				ReconciliationChance: src.ReconciliationChance,
			}
		}),
	}
	return nil
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
	return src.Spec.ConvertTo(&dst.Spec)
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
	return dst.Spec.ConvertFrom(&src.Spec)
}

var _ conversion.Convertible = (*OscCluster)(nil)
