package v1beta1_test

import (
	"testing"

	fuzz "github.com/google/gofuzz"
	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	infrastructurev1beta2 "github.com/outscale/cluster-api-provider-outscale/api/v1beta2"
	"k8s.io/apimachinery/pkg/api/apitesting/fuzzer"
	runtimeserializer "k8s.io/apimachinery/pkg/runtime/serializer"
	conversionutil "sigs.k8s.io/cluster-api/util/conversion"
)

func TestOscMachineConversion(t *testing.T) {
	conversionutil.FuzzTestFunc(conversionutil.FuzzTestFuncInput{
		Hub:         &infrastructurev1beta2.OscMachine{},
		Spoke:       &infrastructurev1beta1.OscMachine{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{OscMachineFuzzFunc},
	})(t)
}

func OscMachineFuzzFunc(_ runtimeserializer.CodecFactory) []any {
	return []any{
		spokeSkipOscMachineStatus,
		spokeSkipOscMachineUnused,
		spokeHardCodeMachineResources,
		hubHardCodeMachineResources,
		spokeSkipVolumeUnused,
		spokeSkipKeypair,

		hubSkipOscMachineUnused,
		hubSingleRootVolume,
		hubProviderIDScheme,
	}
}

func spokeSkipOscMachineStatus(in *infrastructurev1beta1.OscNodeResource, c fuzz.Continue) {}

func spokeHardCodeMachineResources(in *infrastructurev1beta1.OscMachineResources, c fuzz.Continue) {
	*in = infrastructurev1beta1.OscMachineResources{
		Vm: map[string]string{
			"default": "i-foo",
		},
		FGPU: map[string]string{
			"default": "fgpu-foo",
		},
		Image: map[string]string{
			"default": "ami-foo",
		},
		Volumes: map[string]string{
			"/dev/xvdb": "vol-foo",
		},
		PublicIPs: map[string]string{
			"default": "ipalloc-foo",
		},
	}
}

func hubHardCodeMachineResources(in *infrastructurev1beta2.OscMachineResources, c fuzz.Continue) {
	*in = infrastructurev1beta2.OscMachineResources{
		FGPU: map[string]string{
			"default": "fgpu-foo",
		},
		Image: map[string]string{
			"default": "ami-foo",
		},
		Volumes: map[string]string{
			"/dev/xvdb": "vol-foo",
		},
		PublicIPs: map[string]string{
			"default": "ipalloc-foo",
		},
	}
}

func spokeSkipOscMachineUnused(in *infrastructurev1beta1.OscMachineSpec, c fuzz.Continue) {
	c.FuzzNoCustom(in)

	in.Node.ClusterName = ""
	in.Node.Image.ResourceId = ""
	in.Node.Vm.VolumeName = ""
	in.Node.Vm.VolumeDeviceName = ""
	in.Node.Vm.DeviceName = ""
	in.Node.Vm.LoadBalancerName = ""
	in.Node.Vm.PublicIpName = ""
	in.Node.Vm.PrivateIps = nil
	in.Node.Vm.ResourceId = ""
	in.Node.Vm.ClusterName = ""
	in.Node.Vm.Replica = 0
	in.Node.Vm.SubnetName = ""

	switch {
	case len(in.Node.Vm.SubregionNames) == 0:
	case len(in.Node.Vm.SubregionNames) == 1:
		in.Node.Vm.SubregionName = in.Node.Vm.SubregionNames[0]
		in.Node.Vm.SubregionNames = nil
	default:
		in.Node.Vm.SubregionName = ""
	}

	in.ProviderID = new("osc:///eu-west-2/i-foo")
}

func hubSkipOscMachineUnused(in *infrastructurev1beta2.OscMachineSpec, c fuzz.Continue) {
	c.FuzzNoCustom(in)

	in.Vm.PrivateIps = nil
	in.Vm.ResourceId = ""

	in.ProviderID = new("aws:///eu-west-2/i-foo")
}

func hubSingleRootVolume(in *infrastructurev1beta2.OscMachineSpec, c fuzz.Continue) {
	if len(in.Volumes) > 0 {
		for i := range in.Volumes {
			in.Volumes[i].Root = i == 0
			if i == 0 {
				in.Volumes[i].Device = ""
				in.Volumes[i].Description = ""
				in.Volumes[i].FromSnapshot = ""
			}
		}
	} else {
		in.Volumes = []infrastructurev1beta2.OscVolume{{
			Root: true,
			Size: 10,
			Type: infrastructurev1beta2.DefaultRootDiskType,
		}}
	}
}

func hubProviderIDScheme(in *infrastructurev1beta2.ProviderIDScheme, c fuzz.Continue) {
	switch c.Intn(2) {
	case 0:
		*in = infrastructurev1beta2.SchemeAWS
	default:
		*in = infrastructurev1beta2.SchemeOutscale
	}
}

func spokeSkipVolumeUnused(in *infrastructurev1beta1.OscVolume, c fuzz.Continue) {
	c.FuzzNoCustom(in)
	in.ResourceId = ""
	in.SubregionName = ""
}

func spokeSkipKeypair(in *infrastructurev1beta1.OscKeypair, c fuzz.Continue) {}
