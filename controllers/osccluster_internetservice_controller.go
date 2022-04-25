package controllers

import (
	"context"
	"fmt"

	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/net"
	tag "github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag"
	osc "github.com/outscale/osc-sdk-go/v2"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// getInternetServiceResourceId return the InternetServiceId from the resourceMap base on resourceName (tag name + cluster object uid)
func getInternetServiceResourceId(resourceName string, clusterScope *scope.ClusterScope) (string, error) {
	internetServiceRef := clusterScope.GetInternetServiceRef()
	if internetServiceId, ok := internetServiceRef.ResourceMap[resourceName]; ok {
		return internetServiceId, nil
	} else {
		return "", fmt.Errorf("%s is not exist", resourceName)
	}
}

// checkInternetServiceFormatParameters check InternetService parameters format
func checkInternetServiceFormatParameters(clusterScope *scope.ClusterScope) (string, error) {
	clusterScope.Info("Check Internet Service parameters")
	internetServiceSpec := clusterScope.GetInternetService()
	internetServiceSpec.SetDefaultValue()
	internetServiceName := internetServiceSpec.Name + "-" + clusterScope.GetUID()
	internetServiceTagName, err := tag.ValidateTagNameValue(internetServiceName)
	if err != nil {
		return internetServiceTagName, err
	}
	return "", nil
}

// ReconcileInternetService reconcile the InternetService of the cluster.
func reconcileInternetService(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	netsvc := net.NewService(ctx, clusterScope)

	clusterScope.Info("Create InternetGateway")
	internetServiceSpec := clusterScope.GetInternetService()
	internetServiceRef := clusterScope.GetInternetServiceRef()
	internetServiceName := internetServiceSpec.Name + "-" + clusterScope.GetUID()
	var internetService *osc.InternetService
	netSpec := clusterScope.GetNet()
	netSpec.SetDefaultValue()
	netName := netSpec.Name + "-" + clusterScope.GetUID()
	netId, err := getNetResourceId(netName, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}
	if len(internetServiceRef.ResourceMap) == 0 {
		internetServiceRef.ResourceMap = make(map[string]string)
	}
	if internetServiceSpec.ResourceId != "" {
		internetServiceRef.ResourceMap[internetServiceName] = internetServiceSpec.ResourceId
		internetServiceId := internetServiceSpec.ResourceId
		clusterScope.Info("### Get internetServiceId ###", "internetservice", internetServiceRef.ResourceMap)
		internetService, err = netsvc.GetInternetService(internetServiceId)
		if err != nil {
			return reconcile.Result{}, err
		}
	}
	if internetService == nil || internetServiceSpec.ResourceId == "" {
		internetService, err := netsvc.CreateInternetService(internetServiceName)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not create internetservice for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
		}
		err = netsvc.LinkInternetService(*internetService.InternetServiceId, netId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not link internetService with net for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
		}
		clusterScope.Info("### Get internetService ###", "internetservice", internetService)
		internetServiceRef.ResourceMap[internetServiceName] = internetService.GetInternetServiceId()
		internetServiceSpec.ResourceId = internetService.GetInternetServiceId()

	}
	return reconcile.Result{}, nil
}

// reconcileDeleteInternetService reconcile the destruction of the InternetService of the cluster.
func reconcileDeleteInternetService(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	osccluster := clusterScope.OscCluster
	netsvc := net.NewService(ctx, clusterScope)

	clusterScope.Info("Delete internetService")

	internetServiceSpec := clusterScope.GetInternetService()

	netSpec := clusterScope.GetNet()
	netSpec.SetDefaultValue()
	netName := netSpec.Name + "-" + clusterScope.GetUID()

	netId, err := getNetResourceId(netName, clusterScope)
	if err != nil {
		return reconcile.Result{}, err
	}

	internetServiceId := internetServiceSpec.ResourceId
	internetservice, err := netsvc.GetInternetService(internetServiceId)
	if err != nil {
		return reconcile.Result{}, err
	}
	if internetservice == nil {
		controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
		return reconcile.Result{}, nil
	}
	err = netsvc.UnlinkInternetService(internetServiceId, netId)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("%w Can not unlink internetService and net for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
	}
	err = netsvc.DeleteInternetService(internetServiceId)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("%w Can not delete internetService for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
	}
	return reconcile.Result{}, nil
}
