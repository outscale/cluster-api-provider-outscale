package v1beta1_test

import (
	"testing"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	infrastructurev1beta2 "github.com/outscale/cluster-api-provider-outscale/api/v1beta2"
	"k8s.io/apimachinery/pkg/api/apitesting/fuzzer"
	runtimeserializer "k8s.io/apimachinery/pkg/runtime/serializer"
	conversionutil "sigs.k8s.io/cluster-api/util/conversion"
)

func TestOscMachineTemplateConversion(t *testing.T) {
	conversionutil.FuzzTestFunc(conversionutil.FuzzTestFuncInput{
		Hub:         &infrastructurev1beta2.OscMachineTemplate{},
		Spoke:       &infrastructurev1beta1.OscMachineTemplate{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{OscMachineTemplateFuzzFunc},
	})(t)
}

func OscMachineTemplateFuzzFunc(_ runtimeserializer.CodecFactory) []any {
	return []any{
		spokeSkipOscMachineUnused,
		hubSkipOscMachineUnused,
		spokeSkipVolumeUnused,
		spokeSkipKeypair,
		hubProviderIDScheme,
	}
}
