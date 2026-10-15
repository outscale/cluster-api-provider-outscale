package v1beta1

import (
	"errors"
	"strings"

	infrastructurev1beta2 "github.com/outscale/cluster-api-provider-outscale/api/v1beta2"
	"github.com/outscale/osc-sdk-go/v3/pkg/osc"
	"github.com/samber/lo"
	utilconversion "sigs.k8s.io/cluster-api/util/conversion"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

func ConvertRulesTo(src []OscSecurityGroupRule) []infrastructurev1beta2.OscSecurityGroupRule {
	return lo.Map(src, func(src OscSecurityGroupRule, _ int) infrastructurev1beta2.OscSecurityGroupRule {
		return infrastructurev1beta2.OscSecurityGroupRule{
			Description: src.Name,
			Flow:        ConvertFlowTo(src.Flow),
			Ports:       []infrastructurev1beta2.Port{infrastructurev1beta2.BuildPort(src.IpProtocol, src.FromPortRange, src.ToPortRange)},
			IpRanges:    src.GetIpRanges(),
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
				Name:          src.Description,
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
		return sn.HasRole(RoleNat)
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
			Name: srcNet.LoadBalancer.LoadBalancerName,
			Type: infrastructurev1beta2.OscLoadBalancerType(srcNet.LoadBalancer.LoadBalancerType),
			Listener: infrastructurev1beta2.OscLoadBalancerListener{
				BackendPort:          srcNet.LoadBalancer.Listener.BackendPort,
				BackendProtocol:      infrastructurev1beta2.OscLoadBalancerProtocol(srcNet.LoadBalancer.Listener.BackendProtocol),
				LoadBalancerPort:     srcNet.LoadBalancer.Listener.LoadBalancerPort,
				LoadBalancerProtocol: infrastructurev1beta2.OscLoadBalancerProtocol(srcNet.LoadBalancer.Listener.LoadBalancerProtocol),
			},
			HealthCheck: infrastructurev1beta2.OscLoadBalancerHealthCheck{
				CheckInterval:      srcNet.LoadBalancer.HealthCheck.CheckInterval,
				HealthyThreshold:   srcNet.LoadBalancer.HealthCheck.HealthyThreshold,
				Port:               srcNet.LoadBalancer.HealthCheck.Port,
				Protocol:           infrastructurev1beta2.OscLoadBalancerProtocol(srcNet.LoadBalancer.HealthCheck.Protocol),
				Timeout:            srcNet.LoadBalancer.HealthCheck.Timeout,
				UnhealthyThreshold: srcNet.LoadBalancer.HealthCheck.UnhealthyThreshold,
			},
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
		InternetService: infrastructurev1beta2.OscInternetService{
			Description: srcNet.InternetService.Name,
		},
		NatPublicIpPool: srcNet.NatPublicIpPool,
		RouteTables: lo.Map(srcNet.RouteTables, func(src OscRouteTable, _ int) infrastructurev1beta2.OscRouteTable {
			return infrastructurev1beta2.OscRouteTable{
				Description: src.Name,
				Role:        infrastructurev1beta2.OscRole(src.Role),
				Subregion:   infrastructurev1beta2.OscSubRegion(src.SubregionName),
				Routes: lo.Map(src.Routes, func(src OscRoute, _ int) infrastructurev1beta2.OscRoute {
					return infrastructurev1beta2.OscRoute{
						Description: src.Name,
						TargetName:  src.TargetName,
						TargetType:  src.TargetType,
						Destination: src.Destination,
					}
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
			RootDisk: infrastructurev1beta2.OscVolume{
				Iops: srcNet.Bastion.RootDisk.RootDiskIops,
				Size: srcNet.Bastion.RootDisk.RootDiskSize,
				Type: infrastructurev1beta2.OscVolumeType(srcNet.Bastion.RootDisk.RootDiskType),
			},
			PublicIpId: srcNet.Bastion.PublicIpId,
			SecurityGroupNames: lo.Map(srcNet.Bastion.SecurityGroupNames, func(src OscSecurityGroupElement, _ int) infrastructurev1beta2.OscSecurityGroupElement {
				return infrastructurev1beta2.OscSecurityGroupElement(src)
			}),
			Enable: srcNet.Bastion.Enable,
		},
		Subregions:        lo.Map(srcNet.Subregions, func(s string, _ int) infrastructurev1beta2.OscSubRegion { return infrastructurev1beta2.OscSubRegion(s) }),
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
	if !srcNet.Net.IsZero() || len(srcNet.Subnets) > 0 {
		dst.Nets = []infrastructurev1beta2.OscNet{{
			Description: srcNet.Net.Name,
			IpRange:     srcNet.Net.IpRange,
			ResourceId:  srcNet.Net.ResourceId,
			Subnets: lo.Map(srcNet.Subnets, func(src OscSubnet, _ int) infrastructurev1beta2.OscSubnet {
				return infrastructurev1beta2.OscSubnet{
					Description: src.Name,
					Roles: lo.Map(src.Roles, func(src OscRole, _ int) infrastructurev1beta2.OscRole {
						return infrastructurev1beta2.OscRole(src)
					}),
					IpRange:    src.IpSubnetRange,
					Subregion:  infrastructurev1beta2.OscSubRegion(src.SubregionName),
					ResourceId: src.ResourceId,
				}
			}),
		}}
	}
	if len(srcNet.Subregions) == 0 && srcNet.SubregionName != "" {
		dst.Subregions = []infrastructurev1beta2.OscSubRegion{infrastructurev1beta2.OscSubRegion(srcNet.SubregionName)}
	}
	return nil
}

func (dst *OscClusterSpec) ConvertFrom(src *infrastructurev1beta2.OscClusterSpec) error {
	if len(src.Nets) > 1 {
		return errors.New("clusters with 2+ nets cannot be converted to v1beta1")
	}
	var err error
	var srcNet infrastructurev1beta2.OscNet
	if len(src.Nets) > 0 {
		srcNet = src.Nets[0]
	}
	dst.ControlPlaneEndpoint = src.ControlPlaneEndpoint
	dst.Credentials = OscCredentials(src.Credentials)
	dst.Network = OscNetwork{
		UseExisting: OscReuse(src.UseExisting),
		LoadBalancer: OscLoadBalancer{
			LoadBalancerName: src.LoadBalancer.Name,
			LoadBalancerType: string(src.LoadBalancer.Type),
			Listener: OscLoadBalancerListener{
				BackendPort:          src.LoadBalancer.Listener.BackendPort,
				BackendProtocol:      string(src.LoadBalancer.Listener.BackendProtocol),
				LoadBalancerPort:     src.LoadBalancer.Listener.LoadBalancerPort,
				LoadBalancerProtocol: string(src.LoadBalancer.Listener.LoadBalancerProtocol),
			},
			HealthCheck: OscLoadBalancerHealthCheck{
				CheckInterval:      src.LoadBalancer.HealthCheck.CheckInterval,
				HealthyThreshold:   src.LoadBalancer.HealthCheck.HealthyThreshold,
				Port:               src.LoadBalancer.HealthCheck.Port,
				Protocol:           string(src.LoadBalancer.HealthCheck.Protocol),
				Timeout:            src.LoadBalancer.HealthCheck.Timeout,
				UnhealthyThreshold: src.LoadBalancer.HealthCheck.UnhealthyThreshold,
			},
		},
		Net: OscNet{
			Name:       srcNet.Description,
			IpRange:    srcNet.IpRange,
			ResourceId: srcNet.ResourceId,
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
		Subnets: lo.Map(srcNet.Subnets, func(src infrastructurev1beta2.OscSubnet, _ int) OscSubnet {
			return OscSubnet{
				Name: src.Description,
				Roles: lo.Map(src.Roles, func(src infrastructurev1beta2.OscRole, _ int) OscRole {
					return OscRole(src)
				}),
				IpSubnetRange: src.IpRange,
				SubregionName: string(src.Subregion),
				ResourceId:    src.ResourceId,
			}
		}),
		InternetService: OscInternetService{
			Name: src.InternetService.Description,
		},
		NatServices: lo.FilterMap(srcNet.Subnets, func(src infrastructurev1beta2.OscSubnet, _ int) (OscNatService, bool) {
			if !src.HasRole(infrastructurev1beta2.RoleNat) {
				return OscNatService{}, false
			}
			return OscNatService{
				SubregionName: string(src.Subregion),
			}, true
		}),
		NatPublicIpPool: src.NatPublicIpPool,
		RouteTables: lo.Map(src.RouteTables, func(src infrastructurev1beta2.OscRouteTable, _ int) OscRouteTable {
			return OscRouteTable{
				Name:          src.Description,
				Role:          OscRole(src.Role),
				SubregionName: string(src.Subregion),
				Routes: lo.Map(src.Routes, func(src infrastructurev1beta2.OscRoute, _ int) OscRoute {
					return OscRoute{
						Name:        src.Description,
						TargetName:  src.TargetName,
						TargetType:  src.TargetType,
						Destination: src.Destination,
					}
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
			RootDisk: OscRootDisk{
				RootDiskIops: src.Bastion.RootDisk.Iops,
				RootDiskSize: src.Bastion.RootDisk.Size,
				RootDiskType: osc.VolumeType(src.Bastion.RootDisk.Type),
			},
			PublicIpId: src.Bastion.PublicIpId,
			SecurityGroupNames: lo.Map(src.Bastion.SecurityGroupNames, func(src infrastructurev1beta2.OscSecurityGroupElement, _ int) OscSecurityGroupElement {
				return OscSecurityGroupElement(src)
			}),
			Enable: src.Bastion.Enable,
		},
		Subregions:        lo.Map(src.Subregions, func(s infrastructurev1beta2.OscSubRegion, _ int) string { return string(s) }),
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

func (src *OscCluster) ConvertStatusTo(dst *infrastructurev1beta2.OscCluster) error {
	dst.Status = infrastructurev1beta2.OscClusterStatus{
		Ready: src.Status.Ready,
		ReconcilerGeneration: lo.MapEntries(src.Status.ReconcilerGeneration, func(k Reconciler, v int64) (infrastructurev1beta2.Reconciler, int64) {
			return infrastructurev1beta2.Reconciler(k), v
		}),
		FailureDomains: src.Status.FailureDomains,
		Conditions:     src.Status.Conditions,
		VmState:        src.Status.VmState,
	}
	ipRange := src.Spec.Network.Net.IpRange
	if ipRange == "" {
		ipRange = DefaultNet.IpRange
	}
	srcRsrcs := src.Status.Resources
	dst.Status.LegacyResources = new(infrastructurev1beta2.OscClusterResources(srcRsrcs))
	dstRsrcs := &dst.Status.Resources
	dstRsrcs.Bastion = srcRsrcs.Bastion
	dstRsrcs.NetPeering = srcRsrcs.NetPeering
	dstRsrcs.Subnet = srcRsrcs.Subnet
	dstRsrcs.NetAccessPoint = srcRsrcs.NetAccessPoint
	dstRsrcs.SecurityGroup = srcRsrcs.SecurityGroup

	if srcRsrcs.Net != nil && srcRsrcs.Net[DefaultResourceKey] != "" {
		dstRsrcs.Net = map[string]string{
			ipRange: srcRsrcs.Net[DefaultResourceKey],
		}
	}
	if srcRsrcs.InternetService != nil && srcRsrcs.InternetService[DefaultResourceKey] != "" {
		dstRsrcs.InternetService = map[string]string{
			ipRange: srcRsrcs.InternetService[DefaultResourceKey],
		}
	}
	mapNATServiceKey := map[string]string{}
	if srcRsrcs.NatService != nil {
		dstRsrcs.NatService = make(map[string]string, len(srcRsrcs.NatService))
		// Note: natservices are retracked only if the subnet are defined in the spec and not autogenerated
		for k, v := range srcRsrcs.NatService {
			// src have a <subregion>-<cluster ID> key
			// dst have a <subnet id> key
			sn, found := lo.Find(src.Spec.Network.Subnets, func(sn OscSubnet) bool {
				return sn.HasRole(RoleNat) && strings.HasPrefix(k, string(sn.SubregionName))
			})
			if found {
				nv := srcRsrcs.Subnet[sn.IpSubnetRange]
				mapNATServiceKey[k] = nv
				dstRsrcs.NatService[nv] = v
			}
		}
	}
	if srcRsrcs.PublicIPs != nil {
		dstRsrcs.PublicIPs = make(map[string]string, len(srcRsrcs.PublicIPs))
		for k, v := range srcRsrcs.PublicIPs {
			if nk, found := mapNATServiceKey[k]; found {
				dstRsrcs.PublicIPs[nk+"/nat"] = v
			}
		}
	}
	return nil
}

func (dst *OscCluster) ConvertStatusFrom(src *infrastructurev1beta2.OscCluster) error {
	dst.Status = OscClusterStatus{
		Ready: src.Status.Ready,
		ReconcilerGeneration: lo.MapEntries(src.Status.ReconcilerGeneration, func(k infrastructurev1beta2.Reconciler, v int64) (Reconciler, int64) {
			return Reconciler(k), v
		}),
		FailureDomains: src.Status.FailureDomains,
		Conditions:     src.Status.Conditions,
		VmState:        src.Status.VmState,
	}

	if src.Status.LegacyResources != nil {
		dst.Status.Resources = OscClusterResources(*src.Status.LegacyResources)
	}
	return nil
}

func (src *OscCluster) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*infrastructurev1beta2.OscCluster)
	dst.ObjectMeta = src.ObjectMeta
	if err := src.ConvertStatusTo(dst); err != nil {
		return err
	}
	if err := src.Spec.ConvertTo(&dst.Spec); err != nil {
		return err
	}
	restored := &infrastructurev1beta2.OscCluster{}
	if ok, err := utilconversion.UnmarshalData(src, restored); err != nil || !ok {
		return err
	}
	dst.Spec.Keypair = restored.Spec.Keypair
	dst.Status.Resources = restored.Status.Resources
	return nil
}

func (dst *OscCluster) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*infrastructurev1beta2.OscCluster)
	dst.ObjectMeta = src.ObjectMeta
	if err := dst.ConvertStatusFrom(src); err != nil {
		return err
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
