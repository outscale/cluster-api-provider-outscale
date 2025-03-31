package controllers

import (
	"context"
	"errors"
	"fmt"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
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

func (t *ResourceTracker) _getInternetServiceOrId(ctx context.Context, clusterScope *scope.ClusterScope) (*osc.InternetService, string, error) {
	rsrc := clusterScope.GetResources()
	id := getResource(defaultResource, rsrc.InternetService)
	if id != "" {
		return nil, id, nil
	}
	netId, err := t.getNetId(ctx, clusterScope)
	if err != nil {
		return nil, "", fmt.Errorf("get net for internet service: %w", err)
	}
	is, err := t.Cloud.InternetService(ctx, *clusterScope).GetInternetServiceForNet(ctx, netId)
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

func (t *ResourceTracker) getInternetService(ctx context.Context, clusterScope *scope.ClusterScope) (*osc.InternetService, error) {
	is, id, err := t._getInternetServiceOrId(ctx, clusterScope)
	switch {
	case err != nil:
		return nil, err
	case is != nil:
		return is, nil
	}
	is, err = t.Cloud.InternetService(ctx, *clusterScope).GetInternetService(ctx, id)
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
func (t *ResourceTracker) getInternetServiceId(ctx context.Context, clusterScope *scope.ClusterScope) (string, error) {
	_, id, err := t._getInternetServiceOrId(ctx, clusterScope)
	return id, err
}

func (t *ResourceTracker) setInternetServiceId(clusterScope *scope.ClusterScope, id string) {
	rsrc := clusterScope.GetResources()
	if rsrc.InternetService == nil {
		rsrc.InternetService = map[string]string{}
	}
	rsrc.InternetService[defaultResource] = id
}

func (t *ResourceTracker) _getSubnetOrId(ctx context.Context, subnet infrastructurev1beta1.OscSubnet, clusterScope *scope.ClusterScope) (*osc.Subnet, string, error) {
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
	sn, err := t.Cloud.Subnet(ctx, *clusterScope).GetSubnetFromNet(ctx, netId, subnet.IpSubnetRange)
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

func (t *ResourceTracker) getSubnet(ctx context.Context, subnet infrastructurev1beta1.OscSubnet, clusterScope *scope.ClusterScope) (*osc.Subnet, error) {
	sn, id, err := t._getSubnetOrId(ctx, subnet, clusterScope)
	switch {
	case err != nil:
		return nil, err
	case sn != nil:
		return sn, nil
	}
	sn, err = t.Cloud.Subnet(ctx, *clusterScope).GetSubnet(ctx, id)
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
func (t *ResourceTracker) getSubnetId(ctx context.Context, subnet infrastructurev1beta1.OscSubnet, clusterScope *scope.ClusterScope) (string, error) {
	_, id, err := t._getSubnetOrId(ctx, subnet, clusterScope)
	return id, err
}

func (t *ResourceTracker) setSubnetId(clusterScope *scope.ClusterScope, subnet infrastructurev1beta1.OscSubnet, id string) {
	rsrc := clusterScope.GetResources()
	if rsrc.Subnet == nil {
		rsrc.Subnet = map[string]string{}
	}
	rsrc.Subnet[subnet.IpSubnetRange] = id
}

func (t *ResourceTracker) _getNatServiceOrId(ctx context.Context, nat infrastructurev1beta1.OscNatService, clusterScope *scope.ClusterScope) (*osc.NatService, string, error) {
	rsrc := clusterScope.GetResources()

	clientToken := clusterScope.GetNatServiceClientToken(nat)
	id := getResource(clientToken, rsrc.NatService)
	if id != "" {
		return nil, id, nil
	}
	ns, err := t.Cloud.NatService(ctx, *clusterScope).GetNatServiceFromClientToken(ctx, clientToken)
	switch {
	case err != nil:
		return nil, "", fmt.Errorf("get nat service from client token: %w", err)
	case ns != nil:
		t.setNatServiceId(clusterScope, nat, ns.GetNatServiceId())
		return ns, ns.GetNatServiceId(), nil
	}
	tag, err := t.Cloud.Tag(ctx, *clusterScope).ReadTag(ctx, tag.NatResourceType, tag.NameKey, clientToken)
	switch {
	case err != nil:
		return nil, "", fmt.Errorf("get nat service from tag: %w", err)
	case ns == nil:
		return nil, "", fmt.Errorf("get nat service: %w", ErrNoResourceFound)
	default:
		t.setNatServiceId(clusterScope, nat, tag.GetResourceId())
		return nil, tag.GetResourceId(), nil
	}
}

func (t *ResourceTracker) getNatService(ctx context.Context, nat infrastructurev1beta1.OscNatService, clusterScope *scope.ClusterScope) (*osc.NatService, error) {
	ns, id, err := t._getNatServiceOrId(ctx, nat, clusterScope)
	switch {
	case err != nil:
		return nil, err
	case ns != nil:
		return ns, nil
	}
	ns, err = t.Cloud.NatService(ctx, *clusterScope).GetNatService(ctx, id)
	switch {
	case err != nil:
		return nil, err
	case ns == nil:
		return nil, fmt.Errorf("get nat service %s: %w", id, ErrMissingResource)
	default:
		return ns, nil
	}
}

func (t *ResourceTracker) getNatServiceId(ctx context.Context, nat infrastructurev1beta1.OscNatService, clusterScope *scope.ClusterScope) (string, error) {
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

func (t *ResourceTracker) setNatServiceId(clusterScope *scope.ClusterScope, nat infrastructurev1beta1.OscNatService, id string) {
	rsrc := clusterScope.GetResources()
	if rsrc.NatService == nil {
		rsrc.NatService = map[string]string{}
	}
	rsrc.NatService[clusterScope.GetNatServiceClientToken(nat)] = id
}

func (t *ResourceTracker) allocateIP(ctx context.Context, name string, clusterScope *scope.ClusterScope) (string, error) {
	rsrc := clusterScope.GetResources()
	id := getResource(name, rsrc.PublicIPs)
	if id != "" {
		return id, nil
	}
	ip, err := t.Cloud.PublicIp(ctx, *clusterScope).CreatePublicIp(ctx, name, clusterScope.GetUID())
	if err != nil {
		return "", fmt.Errorf("allocate ip: %w", err)
	}
	t.trackIP(clusterScope, name, ip.GetPublicIpId())
	return ip.GetPublicIpId(), nil
}

func (t *ResourceTracker) trackIP(clusterScope *scope.ClusterScope, name, id string) {
	rsrc := clusterScope.GetResources()
	if rsrc.PublicIPs == nil {
		rsrc.PublicIPs = map[string]string{}
	}
	rsrc.PublicIPs[name] = id
}

func (t *ResourceTracker) untrackIP(clusterScope *scope.ClusterScope, name string) {
	rsrc := clusterScope.GetResources()
	if rsrc.PublicIPs == nil {
		return
	}
	delete(rsrc.PublicIPs, name)
}

func (t *ResourceTracker) getPublicIps(clusterScope *scope.ClusterScope) map[string]string {
	rsrc := clusterScope.GetResources()
	return rsrc.PublicIPs
}

func (t *ResourceTracker) getBastion(ctx context.Context, clusterScope *scope.ClusterScope) (*osc.Vm, error) {
	vm, id, err := t._getBastionOrId(ctx, clusterScope)
	switch {
	case err != nil:
		return nil, err
	case vm != nil:
		return vm, nil
	}
	vm, err = t.Cloud.VM(ctx, *clusterScope).GetVm(ctx, id)
	switch {
	case err != nil:
		return nil, err
	case vm == nil:
		return nil, fmt.Errorf("get bastion %s: %w", id, ErrMissingResource)
	default:
		return vm, nil
	}
}

func (t *ResourceTracker) getBastionId(ctx context.Context, clusterScope *scope.ClusterScope) (string, error) {
	vm, id, err := t._getBastionOrId(ctx, clusterScope)
	switch {
	case err != nil:
		return "", err
	case vm != nil:
		return vm.GetVmId(), nil
	default:
		return id, nil
	}
}

// getNetId returns the id for the cluster network, a wrapped ErrNoResourceFound error otherwise.
func (t *ResourceTracker) _getBastionOrId(ctx context.Context, clusterScope *scope.ClusterScope) (*osc.Vm, string, error) {
	id := clusterScope.GetBastion().ResourceId
	if id != "" {
		return nil, id, nil
	}

	clientToken := clusterScope.GetBastionClientToken()
	rsrc := clusterScope.GetResources()
	id = getResource(clientToken, rsrc.Bastion)
	if id != "" {
		return nil, id, nil
	}
	vm, err := t.Cloud.VM(ctx, *clusterScope).GetVmFromClientToken(ctx, clientToken)
	switch {
	case err != nil:
		return nil, "", fmt.Errorf("get bastion from client token: %w", err)
	case vm != nil:
		t.setBastionId(clusterScope, vm.GetVmId())
		return vm, vm.GetVmId(), nil
	}
	// Search by name (retrocompatibility)
	tg, err := t.Cloud.Tag(ctx, *clusterScope).ReadTag(ctx, tag.NetResourceType, tag.NameKey, clusterScope.GetNetName())
	if err != nil {
		return nil, "", fmt.Errorf("get net: %w", err)
	}
	if tg.GetResourceId() != "" {
		t.setNetId(clusterScope, tg.GetResourceId())
		return nil, tg.GetResourceId(), nil
	}
	return nil, "", fmt.Errorf("get net: %w", ErrNoResourceFound)
}

func (t *ResourceTracker) setBastionId(clusterScope *scope.ClusterScope, id string) {
	rsrc := clusterScope.GetResources()
	if rsrc.Bastion == nil {
		rsrc.Bastion = map[string]string{}
	}
	rsrc.Bastion[defaultResource] = id
}
func (t *ResourceTracker) _getSecurityGroupOrId(ctx context.Context, sg infrastructurev1beta1.OscSecurityGroup, clusterScope *scope.ClusterScope) (*osc.SecurityGroup, string, error) {
	rsrc := clusterScope.GetResources()

	name := clusterScope.GetSecurityGroupName(sg)
	id := getResource(name, rsrc.SecurityGroup)
	if id != "" {
		return nil, id, nil
	}
	ns, err := t.Cloud.SecurityGroup(ctx, *clusterScope).GetSecurityGroupFromName(ctx, name)
	switch {
	case err != nil:
		return nil, "", fmt.Errorf("get securityGroup from securityGroupName: %w", err)
	case ns != nil:
		t.setSecurityGroupId(clusterScope, sg, ns.GetSecurityGroupId())
		return ns, ns.GetSecurityGroupId(), nil
	}
	tag, err := t.Cloud.Tag(ctx, *clusterScope).ReadTag(ctx, tag.SecurityGroupResourceType, tag.NameKey, name)
	switch {
	case err != nil:
		return nil, "", fmt.Errorf("get securityGroup from tag: %w", err)
	case ns == nil:
		return nil, "", fmt.Errorf("get securityGroup: %w", ErrNoResourceFound)
	default:
		t.setSecurityGroupId(clusterScope, sg, tag.GetResourceId())
		return nil, tag.GetResourceId(), nil
	}
}

func (t *ResourceTracker) getSecurityGroup(ctx context.Context, sg infrastructurev1beta1.OscSecurityGroup, clusterScope *scope.ClusterScope) (*osc.SecurityGroup, error) {
	ns, id, err := t._getSecurityGroupOrId(ctx, sg, clusterScope)
	switch {
	case err != nil:
		return nil, err
	case ns != nil:
		return ns, nil
	}
	ns, err = t.Cloud.SecurityGroup(ctx, *clusterScope).GetSecurityGroup(ctx, id)
	switch {
	case err != nil:
		return nil, err
	case ns == nil:
		return nil, fmt.Errorf("get securityGroup %s: %w", id, ErrMissingResource)
	default:
		return ns, nil
	}
}

func (t *ResourceTracker) getSecurityGroupId(ctx context.Context, sg infrastructurev1beta1.OscSecurityGroup, clusterScope *scope.ClusterScope) (string, error) {
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

func (t *ResourceTracker) setSecurityGroupId(clusterScope *scope.ClusterScope, sg infrastructurev1beta1.OscSecurityGroup, id string) {
	rsrc := clusterScope.GetResources()
	if rsrc.SecurityGroup == nil {
		rsrc.SecurityGroup = map[string]string{}
	}
	rsrc.SecurityGroup[clusterScope.GetSecurityGroupName(sg)] = id
}
