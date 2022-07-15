package storage

import (
	"context"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
)

type Service struct {
	scope *scope.ClusterScope
	ctx   context.Context
}

func NewService(ctx context.Context, scope *scope.ClusterScope) *Service {
	return &Service{
		scope: scope,
		ctx:   ctx,
	}
}
