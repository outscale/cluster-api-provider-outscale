package scope

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	infrastructurev1beta1 "github.com/outscale-vbr/cluster-api-provider-outscale.git/api/v1beta1"
	"k8s.io/klog/v2/klogr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ClusterScopeParams struct {
	OscClient *OscApiClient
	Client     client.Client
	Logger     logr.Logger
	Cluster    *clusterv1.Cluster
	OscCluster *infrastructurev1beta1.OscCluster
}

func NewClusterScope(params ClusterScopeParams) (*ClusterScope, error) {
	if params.Cluster == nil {
		return nil, errors.New("Cluster is required when creating a ClusterScope")
	}

	if params.OscCluster == nil {
		return nil, errors.New("OscCluster is required when creating a ClusterScope")
	}

	if params.Logger == (logr.Logger{}) {
		params.Logger = klogr.New()
	}

        client, err := newOscApiClient()

        if err != nil {
            return nil, errors.Wrap(err, "failed to create Osc Client")
        }

        if params.OscClient == nil {
            params.OscClient = client
        }

        if params.OscClient.api == nil {
            params.OscClient.api = client.api
        }

        if params.OscClient.auth == nil {
	    params.OscClient.auth = client.auth
        }
        
	helper, err := patch.NewHelper(params.OscCluster, params.Client)
	if err != nil {
		return nil, errors.Wrap(err, "failed to init patch helper")
	}
         
	return &ClusterScope{
		Logger:      params.Logger,
		client:      params.Client,
		Cluster:     params.Cluster,
		OscCluster:  params.OscCluster,
                OscClient:   params.OscClient,
		patchHelper: helper,
	}, nil
}

type ClusterScope struct {
	logr.Logger
	client      client.Client
	patchHelper *patch.Helper
        OscClient  *OscApiClient
	Cluster    *clusterv1.Cluster
	OscCluster *infrastructurev1beta1.OscCluster
}

func (s *ClusterScope) Close() error {
	return s.patchHelper.Patch(context.TODO(), s.OscCluster)
}
