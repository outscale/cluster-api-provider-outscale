package cloud

import (
	"sigs.k8s.io/cluster-api/util/conditions"
)

type ClusterObject interface {
	conditions.Setter
}
