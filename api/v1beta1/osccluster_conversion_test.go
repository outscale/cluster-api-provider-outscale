package v1beta1_test

import (
	"strconv"
	"strings"
	"testing"

	fuzz "github.com/google/gofuzz"
	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	infrastructurev1beta2 "github.com/outscale/cluster-api-provider-outscale/api/v1beta2"
	"github.com/samber/lo"
	"k8s.io/apimachinery/pkg/api/apitesting/fuzzer"
	runtimeserializer "k8s.io/apimachinery/pkg/runtime/serializer"
	conversionutil "sigs.k8s.io/cluster-api/util/conversion"
)

func TestOscClusterConversion(t *testing.T) {
	conversionutil.FuzzTestFunc(conversionutil.FuzzTestFuncInput{
		Hub:         &infrastructurev1beta2.OscCluster{},
		Spoke:       &infrastructurev1beta1.OscCluster{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{OscClusterFuzzFunc},
	})(t)
}

func OscClusterFuzzFunc(_ runtimeserializer.CodecFactory) []any {
	return []any{
		spokeSkipOscClusterStatus,
		spokeHardCodeClusterResources,
		hubHardCodeClusterResources,
		spokeSkipOscClusterUnused,
		hubSkipOscClusterUnused,
		spokeSkipOscLoadBalancerUnused,
		spokeSkipOscBastionUnused,
		hubSkipOscBastionUnused,
		hubSkipOscVolumeUnused,
		spokeSkipOscRouteTableUnused,
		spokeSkipOscRouteUnused,
		spokeSetRule,
		hubSetRule,
		hubSetPort,
		hubSetFlow,
		spokeSetDisable,
		spokeSetRole,
		hubSetRole,
	}
}

func spokeSkipOscClusterStatus(in *infrastructurev1beta1.OscNetworkResource, c fuzz.Continue) {}

func spokeHardCodeClusterResources(in *infrastructurev1beta1.OscClusterResources, c fuzz.Continue) {
	*in = infrastructurev1beta1.OscClusterResources{
		Net: map[string]string{
			"default": "vpc-foo",
		},
		Subnet: map[string]string{
			"10.0.2.0/24": "subnet-public",
			"10.0.3.0/24": "subnet-kw",
			"10.0.4.0/24": "subnet-kcp",
		},
		InternetService: map[string]string{
			"default": "igw-foo",
		},
		SecurityGroup: map[string]string{
			"test-cluster-api-controlplane-9e1db9c4-bf0a-4583-8999-203ec002c520": "sg-kcp",
			"test-cluster-api-worker-9e1db9c4-bf0a-4583-8999-203ec002c520":       "sg-kw",
			"test-cluster-api-lb-9e1db9c4-bf0a-4583-8999-203ec002c520":           "sg-lb",
			"test-cluster-api-node-9e1db9c4-bf0a-4583-8999-203ec002c520":         "sg-node",
		},
		NatService: map[string]string{
			"west-2a-9e1db9c4-bf0a-4583-8999-203ec002c520": "nat-foo",
		},
		PublicIPs: map[string]string{
			"west-2a-9e1db9c4-bf0a-4583-8999-203ec002c520": "ipalloc-nat",
		},
	}
}

func hubHardCodeClusterResources(in *infrastructurev1beta2.OscClusterResources, c fuzz.Continue) {
	*in = infrastructurev1beta2.OscClusterResources{
		Net: map[string]string{
			"10.0.0.0/16": "vpc-foo",
		},
		Subnet: map[string]string{
			"10.0.2.0/24": "subnet-public",
			"10.0.3.0/24": "subnet-kw",
			"10.0.4.0/24": "subnet-kcp",
		},
		InternetService: map[string]string{
			"10.0.0.0/16": "igw-foo",
		},
		SecurityGroup: map[string]string{
			"test-cluster-api-controlplane-9e1db9c4-bf0a-4583-8999-203ec002c520": "sg-kcp",
			"test-cluster-api-worker-9e1db9c4-bf0a-4583-8999-203ec002c520":       "sg-kw",
			"test-cluster-api-lb-9e1db9c4-bf0a-4583-8999-203ec002c520":           "sg-lb",
			"test-cluster-api-node-9e1db9c4-bf0a-4583-8999-203ec002c520":         "sg-node",
		},
		NatService: map[string]string{
			"subnet-public": "nat-foo",
		},
		PublicIPs: map[string]string{
			"subnet-public/nat": "ipalloc-nat",
		},
	}
}

func spokeSkipOscClusterUnused(in *infrastructurev1beta1.OscClusterSpec, c fuzz.Continue) {
	c.FuzzNoCustom(in)

	in.Network.ClusterName = ""
	in.Network.LoadBalancer.ClusterName = ""
	in.Network.Net.ClusterName = ""
	in.Network.InternetService.ClusterName = ""
	in.Network.InternetService.ResourceId = ""
	in.Network.ExtraSecurityGroupRule = false
	in.Network.PublicIps = nil
	in.Network.Image = infrastructurev1beta1.OscImage{}
	in.Network.SubregionName = ""
	in.Network.ControlPlaneSubnets = nil

	if in.Network.Net.IsZero() {
		in.Network.Net.Name = ""
	}

	in.Network.NatServices = in.Network.NatServices[:0]
	for _, subnet := range in.Network.Subnets {
		if !subnet.HasRole(infrastructurev1beta1.RoleNat) {
			continue
		}
		in.Network.NatServices = append(in.Network.NatServices, infrastructurev1beta1.OscNatService{
			SubregionName: subnet.SubregionName,
		})
	}
	in.Network.NatService = infrastructurev1beta1.OscNatService{}
}

func hubSkipOscClusterUnused(in *infrastructurev1beta2.OscClusterSpec, c fuzz.Continue) {
	c.FuzzNoCustom(in)

	for i := range in.Nets {
		if in.Nets[i].IsZero() {
			in.Nets[i].Description = ""
		}
	}
	// drop empty nets from list
	in.Nets = lo.Filter(in.Nets, func(n infrastructurev1beta2.OscNet, _ int) bool {
		return !n.IsZero() || len(n.Subnets) > 0
	})
}

func spokeSkipOscLoadBalancerUnused(in *infrastructurev1beta1.OscLoadBalancer, c fuzz.Continue) {
	c.FuzzNoCustom(in)

	in.SubnetName = ""
	in.SecurityGroupName = ""
}

func spokeSkipOscBastionUnused(in *infrastructurev1beta1.OscBastion, c fuzz.Continue) {
	c.FuzzNoCustom(in)

	in.ClusterName = ""
	in.DeviceName = ""
	in.PublicIpName = ""
	in.PrivateIps = nil
	in.SubregionName = ""
	in.ResourceId = ""
	in.SubnetName = ""
}

func hubSkipOscBastionUnused(in *infrastructurev1beta2.OscBastion, c fuzz.Continue) {
	c.FuzzNoCustom(in)

	in.PrivateIps = nil
	in.ResourceId = ""
}

func hubSkipOscVolumeUnused(in *infrastructurev1beta2.OscVolume, c fuzz.Continue) {
	c.FuzzNoCustom(in)

	in.Root = false
	in.Device = ""
	in.Description = ""
	in.FromSnapshot = ""
}

func spokeSkipOscRouteTableUnused(in *infrastructurev1beta1.OscRouteTable, c fuzz.Continue) {
	c.FuzzNoCustom(in)

	in.ResourceId = ""
	in.Subnets = nil
}

func spokeSkipOscRouteUnused(in *infrastructurev1beta1.OscRoute, c fuzz.Continue) {
	c.FuzzNoCustom(in)

	in.ResourceId = ""
}

func spokeSetRule(in *infrastructurev1beta1.OscSecurityGroupRule, c fuzz.Continue) {
	switch c.Intn(2) {
	case 0:
		in.Flow = "Inbound"
	default:
		in.Flow = "Outbound"
	}
	switch c.Intn(2) {
	case 0:
		in.IpProtocol = "-4"
	default:
		in.IpProtocol = "tcp"
	}
	switch c.Intn(3) {
	case 0:
		in.FromPortRange = int32(c.Intn(65534) + 1)
		in.ToPortRange = in.FromPortRange
	case 1:
		in.FromPortRange = int32(c.Intn(65534) + 1)
		in.ToPortRange = int32(c.Intn(65534) + 1)
	default:
		in.FromPortRange = -1
		in.ToPortRange = -1
	}
}

func hubSetRule(in *infrastructurev1beta2.OscSecurityGroupRule, c fuzz.Continue) {
	c.FuzzNoCustom(in)
	if len(in.Ports) == 0 {
		in.Ports = []infrastructurev1beta2.Port{"tcp/8080-8088"}
	}
}

func hubSetPort(in *infrastructurev1beta2.Port, c fuzz.Continue) {
	var p strings.Builder
	switch c.Intn(2) {
	case 0:
		p.WriteString("-4")
	default:
		p.WriteString("tcp")
	}
	switch c.Intn(3) {
	case 0:
		p.WriteString("/")
		p.WriteString(strconv.Itoa(c.Intn(65534) + 1))
	case 1:
		p.WriteString("/")
		p.WriteString(strconv.Itoa(c.Intn(65534) + 1))
		p.WriteString("-")
		p.WriteString(strconv.Itoa(c.Intn(65534) + 1))
	default:
	}
	*in = infrastructurev1beta2.Port(p.String())
}

func hubSetFlow(in *infrastructurev1beta2.Flow, c fuzz.Continue) {
	switch c.Intn(2) {
	case 0:
		*in = infrastructurev1beta2.FlowInbound
	default:
		*in = infrastructurev1beta2.FlowOutbound
	}
}

func spokeSetDisable(in *infrastructurev1beta1.OscDisable, c fuzz.Continue) {
	switch c.Intn(2) {
	case 0:
		*in = infrastructurev1beta1.DisableInternet
	default:
		*in = infrastructurev1beta1.DisableLB
	}
}

func spokeSetRole(in *infrastructurev1beta1.OscRole, c fuzz.Continue) {
	switch c.Intn(4) {
	case 0:
		*in = infrastructurev1beta1.RoleNat
	case 1:
		*in = infrastructurev1beta1.RoleLoadBalancer
	case 2:
		*in = infrastructurev1beta1.RoleControlPlane
	default:
		*in = infrastructurev1beta1.RoleWorker
	}
}

func hubSetRole(in *infrastructurev1beta2.OscRole, c fuzz.Continue) {
	switch c.Intn(4) {
	case 0:
		*in = infrastructurev1beta2.RoleNat
	case 1:
		*in = infrastructurev1beta2.RoleLoadBalancer
	case 2:
		*in = infrastructurev1beta2.RoleControlPlane
	default:
		*in = infrastructurev1beta2.RoleWorker
	}
}
