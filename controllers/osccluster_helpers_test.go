package controllers_test

import "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"

func patchMoveCluster() patchOSCClusterFunc {
	return func(m *v1beta1.OscCluster) {
		m.Status = v1beta1.OscClusterStatus{}
	}
}
