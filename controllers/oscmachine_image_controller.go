/*
Copyright 2022 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"

	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/compute"
	osc "github.com/outscale/osc-sdk-go/v2"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// checkImageFormatParameters check keypair format
func checkImageFormatParameters(machineScope *scope.MachineScope) (string, error) {
	machineScope.Info("Check Image parameters")
	var err error
	var imageSpec *infrastructurev1beta1.OscImage
	nodeSpec := machineScope.GetNode()
	if nodeSpec.Image.Name == "" {
		return "", nil
	} else {
		imageSpec = machineScope.GetImage()
	}
	imageName := imageSpec.Name
	imageName, err = infrastructurev1beta1.ValidateImageName(imageName)
	if err != nil {
		return imageName, err
	}
	return "", nil
}

// getImageResourceId return the iamgeName from the resourceMap base on resourceName (tag name + cluster uid)
func getImageResourceId(resourceName string, machineScope *scope.MachineScope) (string, error) {
	imageRef := machineScope.GetImageRef()
	if imageId, ok := imageRef.ResourceMap[resourceName]; ok {
		return imageId, nil
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
	imageName := imageSpec.Name
	imageId := machineScope.GetVm().ImageId
	var image *osc.Image
	var err error

	if len(imageRef.ResourceMap) == 0 {
		imageRef.ResourceMap = make(map[string]string)
	}

	if imageName != "" {
		machineScope.Info("########### Image Names exist", "imageName", imageName)
		if imageId, err = imageSvc.GetImageId(imageName); err != nil {
			return reconcile.Result{}, err
		}
	} else {
		machineScope.Info("########### Image Name is empty and we wiqqll try to get it from Id#####", "imageId", imageId)
		if imageName, err = imageSvc.GetImageName(imageId); err != nil {
			return reconcile.Result{}, err
		}
	}
	if imageSpec.ResourceId != "" {
		imageRef.ResourceMap[imageName] = imageSpec.ResourceId
	}

	if image, err = imageSvc.GetImage(imageId); err != nil {
		return reconcile.Result{}, err
	}
	if image == nil || imageSpec.ResourceId == "" {
		machineScope.Info("########### Image is nil")

		return reconcile.Result{}, err
	} else {
		imageRef.ResourceMap[imageName] = imageSpec.ResourceId
	}
	return reconcile.Result{}, nil
}
