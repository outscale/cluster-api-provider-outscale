package service

import (
	"context"

	"github.com/outscale-vbr/cluster-api-provider-outscale.git/cloud/scope"
)

// Service is a collection of interfaces
type Service struct {
	scope *scope.ClusterScope
	ctx   context.Context
}

// NewService return a service which is based on outscale api client
func NewService(ctx context.Context, scope *scope.ClusterScope) *Service {
	return &Service{
		scope: scope,
		ctx:   ctx,
	}
}
