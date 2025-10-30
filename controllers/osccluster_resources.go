/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package controllers

import (
	"context"
	"fmt"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services"
	"github.com/outscale/cluster-api-provider-outscale/cloud/tag"
	osc "github.com/outscale/osc-sdk-go/v2"
)

const bastionIPResourceKey = "bastion"

type ClusterResourceTracker struct {
	Cloud services.Servicer
}

func (t *ClusterResourceTracker) getNet(ctx context.Context, clusterScope *scope.ClusterScope) (*osc.Net, error) {
	id, err := t.getNetId(ctx, clusterScope)
	if err != nil {
		return nil, err
	}
	n, err := t.Cloud.Net(clusterScope.Tenant).GetNet(ctx, id)
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
func (t *ClusterResourceTracker) getNetId(ctx context.Context, clusterScope *scope.ClusterScope) (string, error) {
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
	tg, err := t.Cloud.Tag(clusterScope.Tenant).ReadOwnedByTag(ctx, tag.NetResourceType, clusterScope.GetUID())
	if err != nil {
		return "", fmt.Errorf("get net: %w", err)
	}
	if tg.GetResourceId() != "" {
		t.setNetId(clusterScope, tg.GetResourceId())
		return tg.GetResourceId(), nil
	}
	// Search by name (retrocompatibility)
	if clusterScope.GetNet().Name != "" {
		nameValue := clusterScope.GetNet().Name + "-" + clusterScope.GetUID()
		tg, err = t.Cloud.Tag(clusterScope.Tenant).ReadTag(ctx, tag.NetResourceType, tag.NameKey, nameValue)
		if err != nil {
			return "", fmt.Errorf("get net: %w", err)
		}
		if tg.GetResourceId() != "" {
			t.setNetId(clusterScope, tg.GetResourceId())
			return tg.GetResourceId(), nil
		}
	}
	return "", fmt.Errorf("get net: %w", ErrNoResourceFound)
}

func (t *ClusterResourceTracker) setNetId(clusterScope *scope.ClusterScope, id string) {
	rsrc := clusterScope.GetResources()
	if rsrc.Net == nil {
		rsrc.Net = map[string]string{}
	}
	rsrc.Net[defaultResource] = id
}

func (t *ClusterResourceTracker) getNetPeering(ctx context.Context, clusterScope *scope.ClusterScope) (*osc.NetPeering, error) {
	id, err := t.getNetPeeringId(ctx, clusterScope)
	if err != nil {
		return nil, err
	}
	n, err := t.Cloud.NetPeering(clusterScope.Tenant).GetNetPeering(ctx, id)
	switch {
	case err != nil:
		return nil, err
	case n == nil:
		return nil, fmt.Errorf("get net peering %s: %w", id, ErrMissingResource)
	default:
		return n, nil
	}
}

// getNetPeeringId returns the id for the netpeering, a wrapped ErrNoResourceFound error otherwise.
func (t *ClusterResourceTracker) getNetPeeringId(ctx context.Context, clusterScope *scope.ClusterScope) (string, error) {
	rsrc := clusterScope.GetResources()
	id := getResource(defaultResource, rsrc.NetPeering)
	if id != "" {
		return id, nil
	}
	// Search by OscK8sClusterID/(uid): owned tag
	tg, err := t.Cloud.Tag(clusterScope.Tenant).ReadOwnedByTag(ctx, tag.NetPeeringResourceType, clusterScope.GetUID())
	if err != nil {
		return "", fmt.Errorf("get net peering: %w", err)
	}
	if tg.GetResourceId() != "" {
		t.setNetPeeringId(clusterScope, tg.GetResourceId())
		return tg.GetResourceId(), nil
	}
	return "", fmt.Errorf("get net peering: %w", ErrNoResourceFound)
}

func (t *ClusterResourceTracker) setNetPeeringId(clusterScope *scope.ClusterScope, id string) {
	rsrc := clusterScope.GetResources()
	if rsrc.NetPeering == nil {
		rsrc.NetPeering = map[string]string{}
	}
	rsrc.NetPeering[defaultResource] = id
}

func (t *ClusterResourceTracker) _getInternetServiceOrId(ctx context.Context, clusterScope *scope.ClusterScope) (*osc.InternetService, string, error) {
	rsrc := clusterScope.GetResources()
	id := getResource(defaultResource, rsrc.InternetService)
	if id != "" {
		return nil, id, nil
	}
	netId, err := t.getNetId(ctx, clusterScope)
	if err != nil {
		return nil, "", fmt.Errorf("get net for internet service: %w", err)
	}
	is, err := t.Cloud.InternetService(clusterScope.Tenant).GetInternetServiceForNet(ctx, netId)
	switch {
	case err != nil:
		return nil, "", fmt.Errorf("get internet service for net: %w", err)
	case is == nil:
		return nil, "", fmt.Errorf("get internet service: %w", ErrNoResourceFound)
	default:
		t.setInternetServiceId(clusterScope, is.GetInternetServiceId())
		return is, is.GetInternetServiceId(), nil
	}
}

func (t *ClusterResourceTracker) getInternetService(ctx context.Context, clusterScope *scope.ClusterScope) (*osc.InternetService, error) {
	is, id, err := t._getInternetServiceOrId(ctx, clusterScope)
	switch {
	case err != nil:
		return nil, err
	case is != nil:
		return is, nil
	}
	is, err = t.Cloud.InternetService(clusterScope.Tenant).GetInternetService(ctx, id)
	switch {
	case err != nil:
		return nil, err
	case is == nil:
		return nil, fmt.Errorf("get internet service %s: %w", id, ErrMissingResource)
	default:
		return is, nil
	}
}

// getNetId returns the id for the cluster network, a wrapped ErrNoResourceFound error otherwise.
func (t *ClusterResourceTracker) getInternetServiceId(ctx context.Context, clusterScope *scope.ClusterScope) (string, error) {
	_, id, err := t._getInternetServiceOrId(ctx, clusterScope)
	return id, err
}

func (t *ClusterResourceTracker) setInternetServiceId(clusterScope *scope.ClusterScope, id string) {
	rsrc := clusterScope.GetResources()
	if rsrc.InternetService == nil {
		rsrc.InternetService = map[string]string{}
	}
	rsrc.InternetService[defaultResource] = id
}

func (t *ClusterResourceTracker) _getNetAccessPointOrId(ctx context.Context, service infrastructurev1beta1.OscNetAccessPointService, clusterScope *scope.ClusterScope) (*osc.NetAccessPoint, string, error) {
	rsrc := clusterScope.GetResources()
	id := getResource(string(service), rsrc.NetAccessPoint)
	if id != "" {
		return nil, id, nil
	}
	netId, err := t.getNetId(ctx, clusterScope)
	if err != nil {
		return nil, "", fmt.Errorf("get net for net access point: %w", err)
	}
	nap, err := t.Cloud.NetAccessPoint(clusterScope.Tenant).GetNetAccessPointFor(ctx, netId, clusterScope.Tenant.Region(), string(service))
	switch {
	case err != nil:
		return nil, "", fmt.Errorf("get net access point for net: %w", err)
	case nap == nil:
		return nil, "", fmt.Errorf("get net access point: %w", ErrNoResourceFound)
	default:
		t.setNetAccessPointId(clusterScope, service, nap.GetNetAccessPointId())
		return nap, nap.GetNetAccessPointId(), nil
	}
}

func (t *ClusterResourceTracker) getNetAccessPoint(ctx context.Context, service infrastructurev1beta1.OscNetAccessPointService, clusterScope *scope.ClusterScope) (*osc.NetAccessPoint, error) {
	nap, id, err := t._getNetAccessPointOrId(ctx, service, clusterScope)
	switch {
	case err != nil:
		return nil, err
	case nap != nil:
		return nap, nil
	}
	nap, err = t.Cloud.NetAccessPoint(clusterScope.Tenant).GetNetAccessPoint(ctx, id)
	switch {
	case err != nil:
		return nil, err
	case nap == nil:
		return nil, fmt.Errorf("get net access point %s: %w", id, ErrMissingResource)
	default:
		return nap, nil
	}
}

func (t *ClusterResourceTracker) setNetAccessPointId(clusterScope *scope.ClusterScope, service infrastructurev1beta1.OscNetAccessPointService, id string) {
	rsrc := clusterScope.GetResources()
	if rsrc.NetAccessPoint == nil {
		rsrc.NetAccessPoint = map[string]string{}
	}
	rsrc.NetAccessPoint[string(service)] = id
}

func (t *ClusterResourceTracker) _getSubnetOrId(ctx context.Context, subnet infrastructurev1beta1.OscSubnet, clusterScope *scope.ClusterScope) (*osc.Subnet, string, error) {
	id := subnet.ResourceId
	if id != "" {
		return nil, id, nil
	}

	rsrc := clusterScope.GetResources()
	id = getResource(subnet.IpSubnetRange, rsrc.Subnet)
	if id != "" {
		return nil, id, nil
	}
	netId, err := t.getNetId(ctx, clusterScope)
	if err != nil {
		return nil, "", fmt.Errorf("get net for subnet: %w", err)
	}
	sn, err := t.Cloud.Subnet(clusterScope.Tenant).GetSubnetFromNet(ctx, netId, subnet.IpSubnetRange)
	switch {
	case err != nil:
		return nil, "", fmt.Errorf("get subnet from net: %w", err)
	case sn == nil:
		return nil, "", fmt.Errorf("get subnet: %w", ErrNoResourceFound)
	default:
		t.setSubnetId(clusterScope, subnet, sn.GetSubnetId())
		return sn, sn.GetSubnetId(), nil
	}
}

func (t *ClusterResourceTracker) getSubnet(ctx context.Context, subnet infrastructurev1beta1.OscSubnet, clusterScope *scope.ClusterScope) (*osc.Subnet, error) {
	sn, id, err := t._getSubnetOrId(ctx, subnet, clusterScope)
	switch {
	case err != nil:
		return nil, err
	case sn != nil:
		return sn, nil
	}
	sn, err = t.Cloud.Subnet(clusterScope.Tenant).GetSubnet(ctx, id)
	switch {
	case err != nil:
		return nil, err
	case sn == nil:
		return nil, fmt.Errorf("get subnet %s: %w", id, ErrMissingResource)
	default:
		return sn, nil
	}
}

// getNetId returns the id for the cluster network, a wrapped ErrNoResourceFound error otherwise.
func (t *ClusterResourceTracker) getSubnetId(ctx context.Context, subnet infrastructurev1beta1.OscSubnet, clusterScope *scope.ClusterScope) (string, error) {
	_, id, err := t._getSubnetOrId(ctx, subnet, clusterScope)
	return id, err
}

func (t *ClusterResourceTracker) setSubnetId(clusterScope *scope.ClusterScope, subnet infrastructurev1beta1.OscSubnet, id string) {
	rsrc := clusterScope.GetResources()
	if rsrc.Subnet == nil {
		rsrc.Subnet = map[string]string{}
	}
	rsrc.Subnet[subnet.IpSubnetRange] = id
}

func (t *ClusterResourceTracker) _getNatServiceOrId(ctx context.Context, nat infrastructurev1beta1.OscNatService, clusterScope *scope.ClusterScope) (*osc.NatService, string, error) {
	rsrc := clusterScope.GetResources()

	clientToken := clusterScope.GetNatServiceClientToken(nat)
	id := getResource(clientToken, rsrc.NatService)
	if id != "" {
		return nil, id, nil
	}
	ns, err := t.Cloud.NatService(clusterScope.Tenant).GetNatServiceFromClientToken(ctx, clientToken)
	switch {
	case err != nil:
		return nil, "", fmt.Errorf("get nat service from client token: %w", err)
	case ns != nil:
		t.setNatServiceId(clusterScope, nat, ns.GetNatServiceId())
		return ns, ns.GetNatServiceId(), nil
	}
	if nat.Name != "" {
		nameValue := nat.Name + "-" + clusterScope.GetUID()
		tag, err := t.Cloud.Tag(clusterScope.Tenant).ReadTag(ctx, tag.NatResourceType, tag.NameKey, nameValue)
		switch {
		case err != nil:
			return nil, "", fmt.Errorf("get nat service from tag: %w", err)
		case tag == nil:
		default:
			t.setNatServiceId(clusterScope, nat, tag.GetResourceId())
			return nil, tag.GetResourceId(), nil
		}
	}
	return nil, "", fmt.Errorf("get nat service: %w", ErrNoResourceFound)
}

func (t *ClusterResourceTracker) getNatService(ctx context.Context, nat infrastructurev1beta1.OscNatService, clusterScope *scope.ClusterScope) (*osc.NatService, error) {
	ns, id, err := t._getNatServiceOrId(ctx, nat, clusterScope)
	switch {
	case err != nil:
		return nil, err
	case ns != nil:
		// update IP tracking, in case status was reset
		if len(ns.GetPublicIps()) > 0 {
			t.trackIP(clusterScope, clusterScope.GetNatServiceClientToken(nat), ns.GetPublicIps()[0].GetPublicIpId())
		}
		return ns, nil
	}
	ns, err = t.Cloud.NatService(clusterScope.Tenant).GetNatService(ctx, id)
	switch {
	case err != nil:
		return nil, err
	case ns == nil:
		return nil, fmt.Errorf("get nat service %s: %w", id, ErrMissingResource)
	default:
		// update IP tracking, in case status was reset
		if len(ns.GetPublicIps()) > 0 {
			t.trackIP(clusterScope, clusterScope.GetNatServiceClientToken(nat), ns.GetPublicIps()[0].GetPublicIpId())
		}
		return ns, nil
	}
}

func (t *ClusterResourceTracker) getNatServiceId(ctx context.Context, nat infrastructurev1beta1.OscNatService, clusterScope *scope.ClusterScope) (string, error) {
	ns, id, err := t._getNatServiceOrId(ctx, nat, clusterScope)
	switch {
	case err != nil:
		return "", err
	case ns != nil:
		return ns.GetNatServiceId(), nil
	default:
		return id, nil
	}
}

func (t *ClusterResourceTracker) setNatServiceId(clusterScope *scope.ClusterScope, nat infrastructurev1beta1.OscNatService, id string) {
	rsrc := clusterScope.GetResources()
	if rsrc.NatService == nil {
		rsrc.NatService = map[string]string{}
	}
	rsrc.NatService[clusterScope.GetNatServiceClientToken(nat)] = id
}

func (t *ClusterResourceTracker) getPublicIps(clusterScope *scope.ClusterScope) map[string]string {
	rsrc := clusterScope.GetResources()
	return rsrc.PublicIPs
}

func (t *ClusterResourceTracker) getBastion(ctx context.Context, clusterScope *scope.ClusterScope) (*osc.Vm, error) {
	vm, id, err := t._getBastionOrId(ctx, clusterScope)
	switch {
	case err != nil:
		return nil, err
	case vm != nil:
		err := t.IPAllocator(clusterScope).RetrackIP(ctx, bastionIPResourceKey, vm.GetPublicIp(), clusterScope)
		return vm, err
	}
	vm, err = t.Cloud.VM(clusterScope.Tenant).GetVm(ctx, id)
	switch {
	case err != nil:
		return nil, err
	case vm == nil:
		return nil, fmt.Errorf("get bastion %s: %w", id, ErrMissingResource)
	default:
		err := t.IPAllocator(clusterScope).RetrackIP(ctx, bastionIPResourceKey, vm.GetPublicIp(), clusterScope)
		return vm, err
	}
}

// getNetId returns the id for the cluster network, a wrapped ErrNoResourceFound error otherwise.
func (t *ClusterResourceTracker) _getBastionOrId(ctx context.Context, clusterScope *scope.ClusterScope) (*osc.Vm, string, error) {
	id := clusterScope.GetBastion().ResourceId
	if id != "" {
		return nil, id, nil
	}

	rsrc := clusterScope.GetResources()
	id = getResource(defaultResource, rsrc.Bastion)
	if id != "" {
		return nil, id, nil
	}
	clientToken := clusterScope.GetBastionClientToken()
	vm, err := t.Cloud.VM(clusterScope.Tenant).GetVmFromClientToken(ctx, clientToken)
	switch {
	case err != nil:
		return nil, "", fmt.Errorf("get bastion from client token: %w", err)
	case vm != nil:
		t.setBastionId(clusterScope, vm.GetVmId())
		return vm, vm.GetVmId(), nil
	}
	// Search by name (retrocompatibility)
	if clusterScope.GetBastion().Name != "" {
		nameValue := clusterScope.GetBastionName() + "-" + clusterScope.GetUID()
		tg, err := t.Cloud.Tag(clusterScope.Tenant).ReadTag(ctx, tag.VmResourceType, tag.NameKey, nameValue)
		if err != nil {
			return nil, "", fmt.Errorf("get bastion: %w", err)
		}
		if tg.GetResourceId() != "" {
			t.setBastionId(clusterScope, tg.GetResourceId())
			return nil, tg.GetResourceId(), nil
		}
	}
	return nil, "", fmt.Errorf("get bastion: %w", ErrNoResourceFound)
}

func (t *ClusterResourceTracker) setBastionId(clusterScope *scope.ClusterScope, id string) {
	rsrc := clusterScope.GetResources()
	if rsrc.Bastion == nil {
		rsrc.Bastion = map[string]string{}
	}
	rsrc.Bastion[defaultResource] = id
}
func (t *ClusterResourceTracker) _getSecurityGroupOrId(ctx context.Context, sg infrastructurev1beta1.OscSecurityGroup, clusterScope *scope.ClusterScope) (*osc.SecurityGroup, string, error) {
	if sg.ResourceId != "" {
		return nil, sg.ResourceId, nil
	}

	rsrc := clusterScope.GetResources()
	name := clusterScope.GetSecurityGroupName(sg)
	id := getResource(name, rsrc.SecurityGroup)
	if id != "" {
		return nil, id, nil
	}
	ns, err := t.Cloud.SecurityGroup(clusterScope.Tenant).GetSecurityGroupFromName(ctx, name)
	switch {
	case err != nil:
		return nil, "", fmt.Errorf("get securityGroup from securityGroupName: %w", err)
	case ns == nil:
		return nil, "", fmt.Errorf("get securityGroup: %w", ErrNoResourceFound)
	default:
		t.setSecurityGroupId(clusterScope, sg, ns.GetSecurityGroupId())
		return ns, ns.GetSecurityGroupId(), nil
	}
}

func (t *ClusterResourceTracker) getSecurityGroup(ctx context.Context, sg infrastructurev1beta1.OscSecurityGroup, clusterScope *scope.ClusterScope) (*osc.SecurityGroup, error) {
	ns, id, err := t._getSecurityGroupOrId(ctx, sg, clusterScope)
	switch {
	case err != nil:
		return nil, err
	case ns != nil:
		return ns, nil
	}
	ns, err = t.Cloud.SecurityGroup(clusterScope.Tenant).GetSecurityGroup(ctx, id)
	switch {
	case err != nil:
		return nil, err
	case ns == nil:
		return nil, fmt.Errorf("get securityGroup %s: %w", id, ErrMissingResource)
	default:
		return ns, nil
	}
}

func (t *ClusterResourceTracker) getSecurityGroupId(ctx context.Context, sg infrastructurev1beta1.OscSecurityGroup, clusterScope *scope.ClusterScope) (string, error) {
	ns, id, err := t._getSecurityGroupOrId(ctx, sg, clusterScope)
	switch {
	case err != nil:
		return "", err
	case ns != nil:
		return ns.GetSecurityGroupId(), nil
	default:
		return id, nil
	}
}

func (t *ClusterResourceTracker) setSecurityGroupId(clusterScope *scope.ClusterScope, sg infrastructurev1beta1.OscSecurityGroup, id string) {
	rsrc := clusterScope.GetResources()
	if rsrc.SecurityGroup == nil {
		rsrc.SecurityGroup = map[string]string{}
	}
	rsrc.SecurityGroup[clusterScope.GetSecurityGroupName(sg)] = id
}

func (t *ClusterResourceTracker) IPAllocator(clusterScope *scope.ClusterScope) IPAllocatorInterface {
	return &IPAllocator{
		Cloud: t.Cloud,
		getPublicIP: func(key string) (id string, found bool) {
			rsrc := clusterScope.GetResources()
			if rsrc.PublicIPs == nil {
				return "", false
			}
			ip := rsrc.PublicIPs[key]
			return ip, ip != ""
		},
		setPublicIP: func(key, id string) {
			rsrc := clusterScope.GetResources()
			if rsrc.PublicIPs == nil {
				rsrc.PublicIPs = map[string]string{}
			}
			rsrc.PublicIPs[key] = id
		},
	}
}

func (t *ClusterResourceTracker) trackIP(clusterScope *scope.ClusterScope, key, id string) {
	rsrc := clusterScope.GetResources()
	if rsrc.PublicIPs == nil {
		rsrc.PublicIPs = map[string]string{}
	}
	rsrc.PublicIPs[key] = id
}

func (t *ClusterResourceTracker) untrackIP(clusterScope *scope.ClusterScope, name string) {
	rsrc := clusterScope.GetResources()
	if rsrc.PublicIPs == nil {
		return
	}
	delete(rsrc.PublicIPs, name)
}
