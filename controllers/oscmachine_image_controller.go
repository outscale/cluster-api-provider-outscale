package controllers

import (
	"context"
	"fmt"

	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/compute"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag"
	osc "github.com/outscale/osc-sdk-go/v2"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// checkImageFormatParameters check keypair format
func checkImageFormatParameters(machineScope *scope.MachineScope) (string, error) {
	machineScope.Info("Check Image parameters")

	var imageSpec *infrastructurev1beta1.OscImage
	nodeSpec := machineScope.GetNode()
	if nodeSpec.Image.Name == "" {
		nodeSpec.SetImageDefaultValue()
		imageSpec = &nodeSpec.Image
	} else {
		imageSpec = machineScope.GetImage()
	}

	imageName := imageSpec.Name
	imageTagName, err := tag.ValidateTagNameValue(imageName)
	if err != nil {
		return imageTagName, err
	}

	return "", nil
}

// getImageResourceId return the iamgeName from the resourceMap base on resourceName (tag name + cluster uid)
func getImageResourceId(resourceName string, machineScope *scope.MachineScope) (string, error) {
	imageRef := machineScope.GetImageRef()
	if imageName, ok := imageRef.ResourceMap[resourceName]; ok {
		return imageName, nil
	} else {
		return "", fmt.Errorf("%s does not exist", resourceName)
	}
}

// reconcileImage reconcile the image of the machine
func reconcileImage(ctx context.Context, machineScope *scope.MachineScope, imageSvc compute.OscImageInterface) (reconcile.Result, error) {
	machineScope.Info("Create image or add existing one")
	var imageSpec *infrastructurev1beta1.OscImage
	imageSpec = machineScope.GetImage()
	imageRef := machineScope.GetImageRef()
	imageName := imageSpec.Name + "-" + machineScope.GetUID()
	var image *osc.Image
	var err error

	if len(imageRef.ResourceMap) == 0 {
		imageRef.ResourceMap = make(map[string]string)
	}
	if imageSpec.ResourceId != "" {
		imageRef.ResourceMap[imageName] = imageSpec.ResourceId
	}

	if image, err = imageSvc.GetImage(imageName); err != nil {
		return reconcile.Result{}, err
	}
	if image == nil || imageSpec.ResourceId == "" {
		return reconcile.Result{}, err
	} else {
		imageRef.ResourceMap[imageName] = imageSpec.ResourceId
	}
	return reconcile.Result{}, nil
}
