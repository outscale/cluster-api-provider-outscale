package controllers

import (
	"context"
	"errors"
	"fmt"

	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services"
	"github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	osc "github.com/outscale/osc-sdk-go/v2"
)

const (
	defaultResource = "default"
)

var (
	ErrNoResourceFound    = errors.New("not found")
	ErrMissingResource    = errors.New("missing resource")
	ErrNoChangeToResource = errors.New("resource has not changed")
)

type ResourceTracker struct {
	Cloud services.Servicer
}

func getResource(name string, m map[string]string) string {
	if m == nil {
		return ""
	}
	return m[name]
}

func (t *ResourceTracker) getNet(ctx context.Context, clusterScope *scope.ClusterScope) (*osc.Net, error) {
	id, err := t.getNetId(ctx, clusterScope)
	if err != nil {
		return nil, err
	}
	n, err := t.Cloud.Net(ctx, *clusterScope).GetNet(ctx, id)
	switch {
	case err != nil:
		return nil, err
	case n == nil:
		return nil, fmt.Errorf("get net %s: %w", id, ErrMissingResource)
	default:
		return n, nil
	}
}

// getNetId returns the id for the cluster network, a wrapped ErrNoResourceFound error otherwise.
func (t *ResourceTracker) getNetId(ctx context.Context, clusterScope *scope.ClusterScope) (string, error) {
	id := clusterScope.GetNet().ResourceId
	if id != "" {
		return id, nil
	}

	rsrc := clusterScope.GetResources()
	id = getResource(defaultResource, rsrc.Net)
	if id != "" {
		return id, nil
	}
	// Search by OscK8sClusterID/(uid): owned tag
	tg, err := t.Cloud.Tag(ctx, *clusterScope).ReadOwnedByTag(ctx, tag.NetResourceType, clusterScope.GetUID())
	if err != nil {
		return "", fmt.Errorf("get net: %w", err)
	}
	if tg.GetResourceId() != "" {
		t.setNetId(clusterScope, tg.GetResourceId())
		return tg.GetResourceId(), nil
	}
	// Search by name (retrocompatibility)
	tg, err = t.Cloud.Tag(ctx, *clusterScope).ReadTag(ctx, tag.NetResourceType, tag.NameKey, clusterScope.GetNetName())
	if err != nil {
		return "", fmt.Errorf("get net: %w", err)
	}
	if tg.GetResourceId() != "" {
		t.setNetId(clusterScope, tg.GetResourceId())
		return tg.GetResourceId(), nil
	}
	return "", fmt.Errorf("get net: %w", ErrNoResourceFound)
}

func (t *ResourceTracker) setNetId(clusterScope *scope.ClusterScope, id string) {
	rsrc := clusterScope.GetResources()
	if rsrc.Net == nil {
		rsrc.Net = map[string]string{}
	}
	rsrc.Net[defaultResource] = id
}
