package v1beta1_test

import (
	"testing"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	infrastructurev1beta2 "github.com/outscale/cluster-api-provider-outscale/api/v1beta2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOscClusterConversion(t *testing.T) {
	beta2 := &infrastructurev1beta2.OscCluster{
		Spec: infrastructurev1beta2.OscClusterSpec{
			Keypair: &infrastructurev1beta2.OscKeypair{
				Name:              "keypair-name",
				SecretName:        "keypair-secret",
				KeepAfterDeletion: true,
			},
		},
	}
	var beta1 infrastructurev1beta1.OscCluster
	err := beta1.ConvertFrom(beta2)
	require.NoError(t, err)
	var nbeta2 infrastructurev1beta2.OscCluster
	err = beta1.ConvertTo(&nbeta2)
	require.NoError(t, err)
	assert.Equal(t, beta2.Spec.Keypair, nbeta2.Spec.Keypair, "Keypair must not be lost")
}
