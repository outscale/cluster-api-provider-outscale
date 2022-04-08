package cloud

import (
	"sigs.k8s.io/cluster-api/util/conditions"
)

// ClusterObject is a Osc cluster object.
type ClusterObject interface {
	conditions.Setter
}
