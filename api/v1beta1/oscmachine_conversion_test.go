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
		hubSkipOscMachineUnused,
		spokeSkipVolumeUnused,
		spokeSkipKeypair,
	}
}

func spokeSkipOscMachineStatus(in *infrastructurev1beta1.OscNodeResource, c fuzz.Continue) {}

func spokeSkipOscMachineUnused(in *infrastructurev1beta1.OscNode, c fuzz.Continue) {
	c.FuzzNoCustom(in)

	in.ClusterName = ""
	in.Image.ResourceId = ""
	in.Vm.VolumeName = ""
	in.Vm.VolumeDeviceName = ""
	in.Vm.DeviceName = ""
	in.Vm.LoadBalancerName = ""
	in.Vm.PublicIpName = ""
	in.Vm.PrivateIps = nil
	in.Vm.ResourceId = ""
	in.Vm.ClusterName = ""
	in.Vm.Replica = 0
}

func hubSkipOscMachineUnused(in *infrastructurev1beta2.OscMachineSpec, c fuzz.Continue) {
	c.FuzzNoCustom(in)

	in.Vm.PrivateIps = nil
	in.Vm.ResourceId = ""
}

func spokeSkipVolumeUnused(in *infrastructurev1beta1.OscVolume, c fuzz.Continue) {
	c.FuzzNoCustom(in)
	in.ResourceId = ""
	in.SubregionName = ""
}

func spokeSkipKeypair(in *infrastructurev1beta1.OscKeypair, c fuzz.Continue) {}
