/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package controllers

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"sync"

	infrastructurev1beta2 "github.com/outscale/cluster-api-provider-outscale/api/v1beta2"
	"github.com/outscale/goutils/sdk/ptr"
	"github.com/samber/lo"
	"k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var RandIntN = rand.IntN

type MultiAZAllocator struct {
	client client.Client

	mu          sync.Mutex
	deployments map[types.NamespacedName][]types.NamespacedName
	azs         map[types.NamespacedName]infrastructurev1beta2.OscSubRegion
	nets        map[types.NamespacedName]string
}

func NewMultiAZAllocator(c client.Client) *MultiAZAllocator {
	return &MultiAZAllocator{
		client:      c,
		deployments: map[types.NamespacedName][]types.NamespacedName{},
		azs:         map[types.NamespacedName]infrastructurev1beta2.OscSubRegion{},
		nets:        map[types.NamespacedName]string{},
	}
}

func (a *MultiAZAllocator) deployment(m *infrastructurev1beta2.OscMachine) (types.NamespacedName, error) {
	md := m.GetLabels()[clusterv1.MachineDeploymentNameLabel]
	if md == "" {
		return types.NamespacedName{}, errors.New("no MachineDeployment label found")
	}
	key := types.NamespacedName{
		Namespace: m.GetNamespace(),
		Name:      md,
	}
	return key, nil
}

func (a *MultiAZAllocator) name(m *infrastructurev1beta2.OscMachine) types.NamespacedName {
	return types.NamespacedName{
		Namespace: m.GetNamespace(),
		Name:      m.GetName(),
	}
}

func (a *MultiAZAllocator) AllocateAZ(ctx context.Context, m *infrastructurev1beta2.OscMachine, mode infrastructurev1beta2.AllocationMode, role infrastructurev1beta2.OscRole, nets []infrastructurev1beta2.OscNet, azs []infrastructurev1beta2.OscSubRegion) (infrastructurev1beta2.OscNet, infrastructurev1beta2.OscSubnet, infrastructurev1beta2.OscSubRegion, error) {
	az, net, err := a.checkExisting(ctx, m)
	if err != nil {
		return infrastructurev1beta2.OscNet{}, infrastructurev1beta2.OscSubnet{}, "", fmt.Errorf("check AZ/net: %w", err)
	}
	// AZ allocation
	switch {
	case az != "":
	case len(azs) == 0:
		err = errors.New("allocate AZ: no subregions configured")
	case len(azs) == 1:
		log.FromContext(ctx).V(5).Info("Single AZ")
		az = azs[0]
	case mode == infrastructurev1beta2.AllocationModeRandom:
		log.FromContext(ctx).V(5).Info("Random AZ")
		az = a.allocateRandomAZ(ctx, m, azs)
	default:
		log.FromContext(ctx).V(5).Info("Least node AZ")
		az, err = a.allocateLeastNodeAZ(ctx, m, azs)
	}
	if err != nil {
		return infrastructurev1beta2.OscNet{}, infrastructurev1beta2.OscSubnet{}, "", fmt.Errorf("allocate AZ: %w", err)
	}
	log.FromContext(ctx).V(5).Info("Assigned subregion", "subregion", az)

	// Net allocation
	possibles := lo.FilterMap(nets, func(n infrastructurev1beta2.OscNet, i int) (string, bool) {
		if slices.ContainsFunc(n.Subnets, func(sn infrastructurev1beta2.OscSubnet) bool {
			return sn.Subregion == az && sn.HasRole(role)
		}) {
			return n.IpRange, true
		}
		return "", false
	})
	switch {
	case net != "":
	case len(possibles) == 0:
		return infrastructurev1beta2.OscNet{}, infrastructurev1beta2.OscSubnet{}, "", errors.New("allocate net: no net found")
	case len(possibles) == 1:
		log.FromContext(ctx).V(5).Info("Single net")
		net = possibles[0]
	case mode == infrastructurev1beta2.AllocationModeRandom:
		log.FromContext(ctx).V(5).Info("Random net")
		net = a.allocateRandomNet(ctx, m, possibles)
	default:
		log.FromContext(ctx).V(5).Info("Least node net")
		net, err = a.allocateLeastNodeNet(ctx, m, possibles)
	}
	if err != nil {
		return infrastructurev1beta2.OscNet{}, infrastructurev1beta2.OscSubnet{}, "", fmt.Errorf("allocate net: %w", err)
	}
	log.FromContext(ctx).V(5).Info("Assigned net", "net", net)
	n, nfound := lo.Find(nets, func(n infrastructurev1beta2.OscNet) bool {
		return n.IpRange == net
	})

	// Subnet selection
	sn, snfound := lo.Find(n.Subnets, func(sn infrastructurev1beta2.OscSubnet) bool {
		return sn.Subregion == az && sn.HasRole(role)
	})
	if !nfound || !snfound {
		return infrastructurev1beta2.OscNet{}, infrastructurev1beta2.OscSubnet{}, "", errors.New("no subnet found")
	}
	log.FromContext(ctx).V(5).Info("Assigning machine to subnet", "subnet", sn)
	return n, sn, az, nil
}

func (a *MultiAZAllocator) checkExisting(ctx context.Context, m *infrastructurev1beta2.OscMachine) (infrastructurev1beta2.OscSubRegion, string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	name := a.name(m)
	az, azfound := a.azs[name]
	net, netfound := a.nets[name]
	if !azfound || !netfound {
		err := a.refreshWorkers(ctx, m.GetNamespace())
		if err != nil {
			return "", "", err
		}
		az, _ = a.azs[name]
		net, _ = a.nets[name]
	}
	if az != "" || net != "" {
		log.FromContext(ctx).V(3).Info("Found assigned subregion/net", "subregion", az, "net", net)
	}
	return az, net, nil
}

func (a *MultiAZAllocator) allocateRandomAZ(ctx context.Context, m *infrastructurev1beta2.OscMachine, azs []infrastructurev1beta2.OscSubRegion) infrastructurev1beta2.OscSubRegion {
	return a.allocateRandom(ctx, m, azs)
}

func (a *MultiAZAllocator) allocateLeastNodeAZ(ctx context.Context, m *infrastructurev1beta2.OscMachine, azs []infrastructurev1beta2.OscSubRegion) (infrastructurev1beta2.OscSubRegion, error) {
	az, allocated, err := a.allocatLeast(ctx, m, azs, a.azs)
	if err != nil {
		return "", fmt.Errorf("allocate AZ: %w", err)
	}
	a.azs = allocated
	return az, nil
}

func (a *MultiAZAllocator) allocateRandomNet(ctx context.Context, m *infrastructurev1beta2.OscMachine, nets []string) string {
	return a.allocateRandom(ctx, m, nets)
}

func (a *MultiAZAllocator) allocateLeastNodeNet(ctx context.Context, m *infrastructurev1beta2.OscMachine, nets []string) (string, error) {
	net, allocated, err := a.allocatLeast(ctx, m, nets, a.nets)
	if err != nil {
		return "", fmt.Errorf("allocate net: %w", err)
	}
	a.nets = allocated
	return net, nil
}

func (a *MultiAZAllocator) refreshWorkers(ctx context.Context, ns string) error {
	var ms infrastructurev1beta2.OscMachineList
	err := a.client.List(ctx, &ms, client.InNamespace(ns))
	if err != nil {
		return err
	}

	// truncate all clusters in cache
	for _, m := range ms.Items {
		deploy, err := a.deployment(&m)
		if err != nil {
			continue
		}
		if len(a.deployments[deploy]) > 0 {
			a.deployments[deploy] = a.deployments[deploy][:0]
		}
	}

	// refill cache
	for _, m := range ms.Items {
		deploy, err := a.deployment(&m)
		if err != nil {
			continue
		}
		if m.Spec.Role == infrastructurev1beta2.RoleControlPlane {
			continue
		}
		name := a.name(&m)
		a.deployments[deploy] = append(a.deployments[deploy], name)
		a.azs[name] = infrastructurev1beta2.OscSubRegion(ptr.From(m.Status.FailureDomain))
		if m.Annotations != nil {
			a.nets[name] = m.Annotations[infrastructurev1beta2.NetRangeAnnotation]
		}
	}
	return nil
}

func (a *MultiAZAllocator) allocateRandom[T comparable](ctx context.Context, m *infrastructurev1beta2.OscMachine, choices []T) T {
	choice := choices[RandIntN(len(choices))]
	log.FromContext(ctx).V(3).Info("Assigning machine to subregion", "subregion", choice)
	return choice
}

func (a *MultiAZAllocator) allocatLeast[T comparable](ctx context.Context, m *infrastructurev1beta2.OscMachine, choices []T, allocated map[types.NamespacedName]T) (T, map[types.NamespacedName]T, error) {
	var zero T
	logger := log.FromContext(ctx)
	deploy, err := a.deployment(m)
	if err != nil {
		return zero, nil, err
	}
	perChoice := lo.Associate(choices, func(c T) (T, int) { return c, 0 })
	for _, name := range a.deployments[deploy] {
		if allocated[name] != zero {
			perChoice[allocated[name]]++
		}
	}
	for _, name := range a.deployments[deploy] {
		if allocated[name] == zero {
			logger.V(5).Info(fmt.Sprintf("Subregion counts: %v", perChoice), "MachineDeployment", deploy)

			min := math.MaxInt
			var az T
			for k, v := range perChoice {
				if v < min {
					az = k
					min = v
				}
			}

			allocated[name] = az
			perChoice[az]++
			logger.V(3).Info("Assigning machine to subregion", "machine", name.Name, "subregion", az)
		}
		if m.GetName() == name.Name {
			return allocated[name], allocated, nil
		}
	}
	return zero, nil, errors.New("machine was not found")
}
