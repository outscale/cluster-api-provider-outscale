package v1beta2_test

import (
	"testing"

	infrastructurev1beta2 "github.com/outscale/cluster-api-provider-outscale/api/v1beta2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPort_Parse(t *testing.T) {
	tts := []struct {
		port     infrastructurev1beta2.Port
		proto    string
		from, to int
		err      bool
	}{
		{port: "tcp", proto: "tcp", from: -1, to: -1},
		{port: "tcp/1", proto: "tcp", from: 1, to: 1},
		{port: "tcp/1-2", proto: "tcp", from: 1, to: 2},
		{port: "-1", proto: "-1", from: -1, to: -1},
		{port: "4", proto: "4", from: -1, to: -1},
		{port: "udp/1-65535", proto: "udp", from: 1, to: 65535},
		{port: "icmp/65534-65535", proto: "icmp", from: 65534, to: 65535},
		{port: "icmp/65534-65535 # comment", proto: "icmp", from: 65534, to: 65535},
		{port: "icmp/65534-65535# comment", proto: "icmp", from: 65534, to: 65535},
	}
	for _, tt := range tts {
		proto, from, to, err := tt.port.Parse()
		require.NoError(t, err)
		assert.Equal(t, tt.proto, proto)
		assert.Equal(t, tt.from, from)
		assert.Equal(t, tt.to, to)
	}
}
