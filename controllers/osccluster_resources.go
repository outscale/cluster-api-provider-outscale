/*
SPDX-FileCopyrightText: 2025 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package controllers

import (
	"context"
	"fmt"

	infrastructurev1beta2 "github.com/outscale/cluster-api-provider-outscale/api/v1beta2"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/tag"
	"github.com/outscale/osc-sdk-go/v3/pkg/osc"
)

const bastionIPResourceKey = "bastion"

type ClusterResourceTracker struct {
	Cloud services.Servicer
}

func (t *ClusterResourceTracker) getNet(ctx context.Context, ipRange string, clusterScope *scope.ClusterScope) (*osc.Net, error) {
	id, err := t.getNetId(ctx, ipRange, clusterScope)
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
func (t *ClusterResourceTracker) getNetId(ctx context.Context, ipRange string, clusterScope *scope.ClusterScope) (string, error) {
	for _, n := range clusterScope.GetNets() {
		if n.IpRange == ipRange && n.ResourceId != "" {
			return n.ResourceId, nil
		}
	}

	rsrc := clusterScope.GetResources()
	id := getResource(ipRange, rsrc.Net)
	if id != "" {
		return id, nil
	}
	// Search by OscK8sClusterID/(uid): owned tag
	// FIXME
	tg, err := t.Cloud.Tag(clusterScope.Tenant).ReadOwnedByTag(ctx, tag.NetResourceType, clusterScope.GetUID())
	if err != nil {
		return "", fmt.Errorf("get net: %w", err)
	}
	if tg != nil && tg.ResourceId != "" {
		t.setNetId(clusterScope, ipRange, tg.ResourceId)
		return tg.ResourceId, nil
	}
	return "", fmt.Errorf("get net: %w", ErrNoResourceFound)
}

func (t *ClusterResourceTracker) setNetId(clusterScope *scope.ClusterScope, ipRange, id string) {
	rsrc := clusterScope.GetResources()
	if rsrc.Net == nil {
		rsrc.Net = map[string]string{}
	}
	rsrc.Net[ipRange] = id
}

func (t *ClusterResourceTracker) getNetPeering(ctx context.Context, clusterScope *scope.ClusterScope) (*osc.NetPeering, error) {
	id, err := t.getNetPeeringId(ctx, clusterScope)
	if err != nil {
		return nil, err
	}
	n, err := t.Cloud.Net(clusterScope.Tenant).GetNetPeering(ctx, id)
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
	if tg != nil && tg.ResourceId != "" {
		t.setNetPeeringId(clusterScope, tg.ResourceId)
		return tg.ResourceId, nil
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

func (t *ClusterResourceTracker) _getInternetServiceOrId(ctx context.Context, ipRange string, clusterScope *scope.ClusterScope) (*osc.InternetService, string, error) {
	rsrc := clusterScope.GetResources()
	id := getResource(ipRange, rsrc.InternetService)
	if id != "" {
		return nil, id, nil
	}
	netId, err := t.getNetId(ctx, ipRange, clusterScope)
	if err != nil {
		return nil, "", fmt.Errorf("get net for internet service: %w", err)
	}
	is, err := t.Cloud.Net(clusterScope.Tenant).GetInternetServiceForNet(ctx, netId)
	switch {
	case err != nil:
		return nil, "", fmt.Errorf("get internet service for net: %w", err)
	case is == nil:
		return nil, "", fmt.Errorf("get internet service: %w", ErrNoResourceFound)
	default:
		t.setInternetServiceId(clusterScope, ipRange, is.InternetServiceId)
		return is, is.InternetServiceId, nil
	}
}

func (t *ClusterResourceTracker) getInternetService(ctx context.Context, ipRange string, clusterScope *scope.ClusterScope) (*osc.InternetService, error) {
	is, id, err := t._getInternetServiceOrId(ctx, ipRange, clusterScope)
	switch {
	case err != nil:
		return nil, err
	case is != nil:
		return is, nil
	}
	is, err = t.Cloud.Net(clusterScope.Tenant).GetInternetService(ctx, id)
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
func (t *ClusterResourceTracker) getInternetServiceId(ctx context.Context, ipRange string, clusterScope *scope.ClusterScope) (string, error) {
	_, id, err := t._getInternetServiceOrId(ctx, ipRange, clusterScope)
	return id, err
}

func (t *ClusterResourceTracker) setInternetServiceId(clusterScope *scope.ClusterScope, ipRange, id string) {
	rsrc := clusterScope.GetResources()
	if rsrc.InternetService == nil {
		rsrc.InternetService = map[string]string{}
	}
	rsrc.InternetService[ipRange] = id
}

func _getNetAccessPointKey(ipRange string, service infrastructurev1beta2.OscNetAccessPointService) string {
	return ipRange + "/" + string(service)
}

func (t *ClusterResourceTracker) _getNetAccessPointOrId(ctx context.Context, ipRange string, service infrastructurev1beta2.OscNetAccessPointService, clusterScope *scope.ClusterScope) (*osc.NetAccessPoint, string, error) {
	rsrc := clusterScope.GetResources()
	id := getResource(_getNetAccessPointKey(ipRange, service), rsrc.NetAccessPoint)
	if id != "" {
		return nil, id, nil
	}
	netId, err := t.getNetId(ctx, ipRange, clusterScope)
	if err != nil {
		return nil, "", fmt.Errorf("get net for net access point: %w", err)
	}
	nap, err := t.Cloud.Net(clusterScope.Tenant).GetNetAccessPointFor(ctx, netId, clusterScope.Tenant.Region(), string(service))
	switch {
	case err != nil:
		return nil, "", fmt.Errorf("get net access point for net: %w", err)
	case nap == nil:
		return nil, "", fmt.Errorf("get net access point: %w", ErrNoResourceFound)
	default:
		t.setNetAccessPointId(clusterScope, ipRange, service, nap.NetAccessPointId)
		return nap, nap.NetAccessPointId, nil
	}
}

func (t *ClusterResourceTracker) getNetAccessPoint(ctx context.Context, ipRange string, service infrastructurev1beta2.OscNetAccessPointService, clusterScope *scope.ClusterScope) (*osc.NetAccessPoint, error) {
	nap, id, err := t._getNetAccessPointOrId(ctx, ipRange, service, clusterScope)
	switch {
	case err != nil:
		return nil, err
	case nap != nil:
		return nap, nil
	}
	nap, err = t.Cloud.Net(clusterScope.Tenant).GetNetAccessPoint(ctx, id)
	switch {
	case err != nil:
		return nil, err
	case nap == nil:
		return nil, fmt.Errorf("get net access point %s: %w", id, ErrMissingResource)
	default:
		return nap, nil
	}
}

func (t *ClusterResourceTracker) setNetAccessPointId(clusterScope *scope.ClusterScope, ipRange string, service infrastructurev1beta2.OscNetAccessPointService, id string) {
	rsrc := clusterScope.GetResources()
	if rsrc.NetAccessPoint == nil {
		rsrc.NetAccessPoint = map[string]string{}
	}
	rsrc.NetAccessPoint[_getNetAccessPointKey(ipRange, service)] = id
}

func (t *ClusterResourceTracker) _getSubnetOrId(ctx context.Context, subnet infrastructurev1beta2.OscSubnet, clusterScope *scope.ClusterScope) (*osc.Subnet, string, error) {
	id := subnet.ResourceId
	if id != "" {
		return nil, id, nil
	}
	if subnet.IpRange == "" {
		return nil, "", ErrNoResourceFound
	}

	rsrc := clusterScope.GetResources()
	id = getResource(subnet.IpRange, rsrc.Subnet)
	if id != "" {
		return nil, id, nil
	}
	netSpec, err := clusterScope.GetNetContaining(subnet)
	if err != nil {
		return nil, "", fmt.Errorf("get net for subnet: %w", err)
	}
	netId, err := t.getNetId(ctx, netSpec.IpRange, clusterScope)
	if err != nil {
		return nil, "", fmt.Errorf("get net for subnet: %w", err)
	}
	sn, err := t.Cloud.Net(clusterScope.Tenant).GetSubnetFromNet(ctx, netId, subnet.IpRange)
	switch {
	case err != nil:
		return nil, "", fmt.Errorf("get subnet from net: %w", err)
	case sn == nil:
		return nil, "", fmt.Errorf("get subnet: %w", ErrNoResourceFound)
	default:
		t.setSubnetId(clusterScope, subnet, sn.SubnetId)
		return sn, sn.SubnetId, nil
	}
}

func (t *ClusterResourceTracker) getSubnet(ctx context.Context, subnet infrastructurev1beta2.OscSubnet, clusterScope *scope.ClusterScope) (*osc.Subnet, error) {
	sn, id, err := t._getSubnetOrId(ctx, subnet, clusterScope)
	switch {
	case err != nil:
		return nil, err
	case sn != nil:
		return sn, nil
	}
	sn, err = t.Cloud.Net(clusterScope.Tenant).GetSubnet(ctx, id)
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
func (t *ClusterResourceTracker) getSubnetId(ctx context.Context, subnet infrastructurev1beta2.OscSubnet, clusterScope *scope.ClusterScope) (string, error) {
	_, id, err := t._getSubnetOrId(ctx, subnet, clusterScope)
	return id, err
}

func (t *ClusterResourceTracker) setSubnetId(clusterScope *scope.ClusterScope, subnet infrastructurev1beta2.OscSubnet, id string) {
	rsrc := clusterScope.GetResources()
	if rsrc.Subnet == nil {
		rsrc.Subnet = map[string]string{}
	}
	rsrc.Subnet[subnet.IpRange] = id
}

func (t *ClusterResourceTracker) _getNatServiceOrId(ctx context.Context, subnetId string, clusterScope *scope.ClusterScope) (*osc.NatService, string, error) {
	rsrc := clusterScope.GetResources()

	id := getResource(subnetId, rsrc.NatService)
	if id != "" {
		return nil, id, nil
	}
	ns, err := t.Cloud.Net(clusterScope.Tenant).GetNatServiceForSubnet(ctx, subnetId)
	switch {
	case err != nil:
		return nil, "", fmt.Errorf("get nat service from client token: %w", err)
	case ns != nil:
		t.setNatServiceId(clusterScope, subnetId, ns.NatServiceId)
		return ns, ns.NatServiceId, nil
	}
	return nil, "", fmt.Errorf("get nat service: %w", ErrNoResourceFound)
}

func (t *ClusterResourceTracker) getNatService(ctx context.Context, subnetId string, clusterScope *scope.ClusterScope) (*osc.NatService, error) {
	ns, id, err := t._getNatServiceOrId(ctx, subnetId, clusterScope)
	switch {
	case err != nil:
		return nil, err
	case ns != nil:
		// update IP tracking, in case status was reset
		if len(ns.PublicIps) > 0 {
			t.trackIP(clusterScope, clusterScope.GetNatServiceIPTrackingID(subnetId), ns.PublicIps[0].PublicIpId)
		}
		return ns, nil
	}
	ns, err = t.Cloud.Net(clusterScope.Tenant).GetNatService(ctx, id)
	switch {
	case err != nil:
		return nil, err
	case ns == nil:
		return nil, fmt.Errorf("get nat service %s: %w", id, ErrMissingResource)
	default:
		// update IP tracking, in case status was reset
		if len(ns.PublicIps) > 0 {
			t.trackIP(clusterScope, clusterScope.GetNatServiceIPTrackingID(subnetId), ns.PublicIps[0].PublicIpId)
		}
		return ns, nil
	}
}

func (t *ClusterResourceTracker) getNatServiceId(ctx context.Context, subnetId string, clusterScope *scope.ClusterScope) (string, error) {
	ns, id, err := t._getNatServiceOrId(ctx, subnetId, clusterScope)
	switch {
	case err != nil:
		return "", err
	case ns != nil:
		return ns.NatServiceId, nil
	default:
		return id, nil
	}
}

func (t *ClusterResourceTracker) setNatServiceId(clusterScope *scope.ClusterScope, subnetId, id string) {
	rsrc := clusterScope.GetResources()
	if rsrc.NatService == nil {
		rsrc.NatService = map[string]string{}
	}
	rsrc.NatService[subnetId] = id
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
		if vm.PublicIp != nil {
			err := t.IPAllocator(clusterScope).RetrackIP(ctx, bastionIPResourceKey, *vm.PublicIp, clusterScope)
			return vm, err
		}
		return vm, nil
	}
	vm, err = t.Cloud.Compute(clusterScope.Tenant).GetVm(ctx, id)
	switch {
	case err != nil:
		return nil, err
	case vm == nil:
		return nil, fmt.Errorf("get bastion %s: %w", id, ErrMissingResource)
	default:
		if vm.PublicIp != nil {
			err := t.IPAllocator(clusterScope).RetrackIP(ctx, bastionIPResourceKey, *vm.PublicIp, clusterScope)
			return vm, err
		}
		return vm, nil
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
	vm, err := t.Cloud.Compute(clusterScope.Tenant).GetVmFromClientToken(ctx, clientToken)
	switch {
	case err != nil:
		return nil, "", fmt.Errorf("get bastion from client token: %w", err)
	case vm != nil:
		t.setBastionId(clusterScope, vm.VmId)
		return vm, vm.VmId, nil
	}
	// Search by name (retrocompatibility)
	if clusterScope.GetBastion().Name != "" {
		nameValue := clusterScope.GetBastionName() + "-" + clusterScope.GetUID()
		tg, err := t.Cloud.Tag(clusterScope.Tenant).ReadTag(ctx, tag.VmResourceType, tag.NameKey, nameValue)
		if err != nil {
			return nil, "", fmt.Errorf("get bastion: %w", err)
		}
		if tg != nil && tg.ResourceId != "" {
			t.setBastionId(clusterScope, tg.ResourceId)
			return nil, tg.ResourceId, nil
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

func (t *ClusterResourceTracker) _getSecurityGroupOrId(ctx context.Context, ipRange string, sg infrastructurev1beta2.OscSecurityGroup, clusterScope *scope.ClusterScope) (*osc.SecurityGroup, string, error) {
	if sg.ResourceId != "" {
		return nil, sg.ResourceId, nil
	}

	rsrc := clusterScope.GetResources()
	name := clusterScope.GetSecurityGroupName(sg)
	id := getResource(name, rsrc.SecurityGroup)
	if id != "" {
		return nil, id, nil
	}
	ns, err := t.Cloud.Compute(clusterScope.Tenant).GetSecurityGroupFromName(ctx, name)
	switch {
	case err != nil:
		return nil, "", fmt.Errorf("get securityGroup from securityGroupName: %w", err)
	case ns == nil:
		return nil, "", fmt.Errorf("get securityGroup: %w", ErrNoResourceFound)
	default:
		t.setSecurityGroupId(clusterScope, ipRange, sg, ns.SecurityGroupId)
		return ns, ns.SecurityGroupId, nil
	}
}

func (t *ClusterResourceTracker) getSecurityGroup(ctx context.Context, ipRange string, sg infrastructurev1beta2.OscSecurityGroup, clusterScope *scope.ClusterScope) (*osc.SecurityGroup, error) {
	ns, id, err := t._getSecurityGroupOrId(ctx, ipRange, sg, clusterScope)
	switch {
	case err != nil:
		return nil, err
	case ns != nil:
		return ns, nil
	}
	ns, err = t.Cloud.Compute(clusterScope.Tenant).GetSecurityGroup(ctx, id)
	switch {
	case err != nil:
		return nil, err
	case ns == nil:
		return nil, fmt.Errorf("get securityGroup %s: %w", id, ErrMissingResource)
	default:
		return ns, nil
	}
}

func (t *ClusterResourceTracker) getSecurityGroupId(ctx context.Context, ipRange string, sg infrastructurev1beta2.OscSecurityGroup, clusterScope *scope.ClusterScope) (string, error) {
	ns, id, err := t._getSecurityGroupOrId(ctx, ipRange, sg, clusterScope)
	switch {
	case err != nil:
		return "", err
	case ns != nil:
		return ns.SecurityGroupId, nil
	default:
		return id, nil
	}
}

func (t *ClusterResourceTracker) setSecurityGroupId(clusterScope *scope.ClusterScope, ipRange string, sg infrastructurev1beta2.OscSecurityGroup, id string) {
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
