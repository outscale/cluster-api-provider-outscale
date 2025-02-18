package services

import (
	"context"

	"github.com/outscale/cluster-api-provider-outscale/cloud"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/compute"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/net"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/security"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/service"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/storage"
	tag "github.com/outscale/cluster-api-provider-outscale/cloud/tag"
)

type Servicer interface {
	OscClient() *cloud.OscClient

	Net(ctx context.Context, scope scope.ClusterScope) net.OscNetInterface
	Subnet(ctx context.Context, scope scope.ClusterScope) net.OscSubnetInterface
	SecurityGroup(ctx context.Context, scope scope.ClusterScope) security.OscSecurityGroupInterface

	InternetService(ctx context.Context, scope scope.ClusterScope) net.OscInternetServiceInterface
	RouteTable(ctx context.Context, scope scope.ClusterScope) security.OscRouteTableInterface
	NatService(ctx context.Context, scope scope.ClusterScope) net.OscNatServiceInterface
	PublicIp(ctx context.Context, scope scope.ClusterScope) security.OscPublicIpInterface
	LoadBalancer(ctx context.Context, scope scope.ClusterScope) service.OscLoadBalancerInterface

	Volume(ctx context.Context, scope scope.ClusterScope) storage.OscVolumeInterface
	VM(ctx context.Context, scope scope.ClusterScope) compute.OscVmInterface
	Image(ctx context.Context, scope scope.ClusterScope) compute.OscImageInterface
	KeyPair(ctx context.Context, scope scope.ClusterScope) security.OscKeyPairInterface

	Tag(ctx context.Context, scope scope.ClusterScope) tag.OscTagInterface
}

type Services struct {
	oscClient *cloud.OscClient
}

func NewServices() (Services, error) {
	client, err := cloud.NewOscClient()
	if err != nil {
		return Services{}, err
	}

	return Services{oscClient: client}, nil
}

func (cs Services) OscClient() *cloud.OscClient {
	return cs.oscClient
}

// getNetSvc retrieve netSvc
func (Services) Net(ctx context.Context, scope scope.ClusterScope) net.OscNetInterface {
	return net.NewService(ctx, &scope)
}

// getSubnetSvc retrieve subnetSvc
func (Services) Subnet(ctx context.Context, scope scope.ClusterScope) net.OscSubnetInterface {
	return net.NewService(ctx, &scope)
}

// getInternetServiceSvc retrieve internetServiceSvc
func (Services) InternetService(ctx context.Context, scope scope.ClusterScope) net.OscInternetServiceInterface {
	return net.NewService(ctx, &scope)
}

// getRouteTableSvc retrieve routeTableSvc
func (Services) RouteTable(ctx context.Context, scope scope.ClusterScope) security.OscRouteTableInterface {
	return security.NewService(ctx, &scope)
}

// getSecurityGroupSvc retrieve securityGroupSvc
func (Services) SecurityGroup(ctx context.Context, scope scope.ClusterScope) security.OscSecurityGroupInterface {
	return security.NewService(ctx, &scope)
}

// getNatServiceSvc retrieve natServiceSvc
func (Services) NatService(ctx context.Context, scope scope.ClusterScope) net.OscNatServiceInterface {
	return net.NewService(ctx, &scope)
}

// getVolumeSvc retrieve volumeSvc
func (Services) Volume(ctx context.Context, scope scope.ClusterScope) storage.OscVolumeInterface {
	return storage.NewService(ctx, &scope)
}

// getVmSvc retrieve vmSvc
func (Services) VM(ctx context.Context, scope scope.ClusterScope) compute.OscVmInterface {
	return compute.NewService(ctx, &scope)
}

// getImageSvc retrieve imageSvc
func (Services) Image(ctx context.Context, scope scope.ClusterScope) compute.OscImageInterface {
	return compute.NewService(ctx, &scope)
}

// getPublicIpSvc retrieve publicIpSvc
func (Services) PublicIp(ctx context.Context, scope scope.ClusterScope) security.OscPublicIpInterface {
	return security.NewService(ctx, &scope)
}

// getLoadBalancerSvc retrieve loadBalancerSvc
func (Services) LoadBalancer(ctx context.Context, scope scope.ClusterScope) service.OscLoadBalancerInterface {
	return service.NewService(ctx, &scope)
}

// getKeyPairSvc retrieve keypairSvc
func (Services) KeyPair(ctx context.Context, scope scope.ClusterScope) security.OscKeyPairInterface {
	return security.NewService(ctx, &scope)
}

// getTagSvc retrieve tagSvc
func (Services) Tag(ctx context.Context, scope scope.ClusterScope) tag.OscTagInterface {
	return tag.NewService(ctx, &scope)
}

var _ Servicer = Services{}
