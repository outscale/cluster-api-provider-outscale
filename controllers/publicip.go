package controllers

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"

	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services"
	"github.com/outscale/osc-sdk-go/v2"
	ctrl "sigs.k8s.io/controller-runtime"
)

var ErrEmptyPool = errors.New("no available IP in pool")

type IPAllocatorInterface interface {
	AllocateIP(ctx context.Context, key, name, pool string, clusterScope *scope.ClusterScope) (id, ip string, err error)
	RetrackIP(ctx context.Context, key, publicIp string, clusterScope *scope.ClusterScope) error
}

type IPAllocator struct {
	Cloud       services.Servicer
	getPublicIP func(key string) (id string, found bool)
	setPublicIP func(key, id string)
}

func (a *IPAllocator) AllocateIP(ctx context.Context, key, name, pool string, clusterScope *scope.ClusterScope) (id, ip string, err error) {
	log := ctrl.LoggerFrom(ctx)
	if id, ok := a.getPublicIP(key); ok {
		pip, err := a.Cloud.PublicIp(ctx, *clusterScope).GetPublicIp(ctx, id)
		switch {
		case err != nil:
			return "", "", fmt.Errorf("allocate ip: %w", err)
		case pip == nil:
		case pip.LinkPublicIpId != nil:
			// we might have had concurrent allocation, and it looks like the other one won.
			// we need to reallocate
			log.V(3).Info("Known IP is already linked, reallocating", "publicIpId", id)
		default:
			return id, pip.GetPublicIp(), nil
		}
	}
	var (
		pip *osc.PublicIp
	)
	if pool != "" {
		pip, err = a.allocateFromPool(ctx, pool, clusterScope)
	} else {
		pip, err = a.allocate(ctx, name, clusterScope)
	}
	if err != nil {
		return "", "", fmt.Errorf("allocate ip: %w", err)
	}
	a.setPublicIP(key, pip.GetPublicIpId())
	return pip.GetPublicIpId(), pip.GetPublicIp(), nil
}

func (a *IPAllocator) allocate(ctx context.Context, name string, clusterScope *scope.ClusterScope) (*osc.PublicIp, error) {
	log := ctrl.LoggerFrom(ctx)
	log.V(3).Info("Allocating publicIp")
	pip, err := a.Cloud.PublicIp(ctx, *clusterScope).CreatePublicIp(ctx, name, clusterScope.GetUID())
	if err != nil {
		return nil, err
	}
	log.V(2).Info("Allocated publicIp", "publicIpId", pip.GetPublicIpId(), "publicIp", pip.GetPublicIp())
	return pip, nil
}

// TODO: two concurrent reconciliation might get the same IP
// it is expected that a VM creation will fail, trigger a new loop, and the next loop
// will reallocate, even if
func (a *IPAllocator) allocateFromPool(ctx context.Context, pool string, clusterScope *scope.ClusterScope) (*osc.PublicIp, error) {
	log := ctrl.LoggerFrom(ctx)
	log.V(4).Info("Fetching publicIps from pool", "pool", pool)
	pips, err := a.Cloud.PublicIp(ctx, *clusterScope).ListPublicIpsFromPool(ctx, pool)
	if err != nil {
		return nil, fmt.Errorf("from pool: %w", err)
	}
	if len(pips) == 0 {
		return nil, ErrEmptyPool
	}
	// randomly fetch from the list, to limit the chance of allocating
	// the same IP to two concurrent requests
	off := rand.IntN(len(pips)) //nolint:gosec
	for i := range pips {
		pip := pips[(off+i)%len(pips)]
		if pip.LinkPublicIpId == nil {
			log.V(3).Info("Found publicIp in pool", "publicIpId", pip.GetPublicIpId(), "publicIp", pip.GetPublicIp())
			return &pip, nil
		}
	}
	return nil, ErrEmptyPool
}

// RetrackIP retracks
func (a *IPAllocator) RetrackIP(ctx context.Context, key, publicIp string, clusterScope *scope.ClusterScope) error {
	if publicIp == "" {
		return nil
	}
	if ipid, found := a.getPublicIP(key); found && ipid != "" {
		return nil
	}
	ip, err := a.Cloud.PublicIp(ctx, *clusterScope).GetPublicIpByIp(ctx, publicIp)
	if err != nil {
		return fmt.Errorf("retrack ip: %w", err)
	}
	if ip != nil {
		a.setPublicIP(key, ip.GetPublicIpId())
	}
	return nil
}
