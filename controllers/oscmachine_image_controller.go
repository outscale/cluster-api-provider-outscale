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
	"errors"
	"fmt"

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/compute"
	osc "github.com/outscale/osc-sdk-go/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// checkImageFormatParameters check keypair format
func checkImageFormatParameters(machineScope *scope.MachineScope) (string, error) {
	var err error
	var imageSpec *infrastructurev1beta1.OscImage
	nodeSpec := machineScope.GetNode()
	if nodeSpec.Image.Name == "" {
		return "", nil
	} else {
		imageSpec = machineScope.GetImage()
	}
	imageName := imageSpec.Name
	err = infrastructurev1beta1.ValidateImageName(imageName)
	if err != nil {
		return imageName, err
	}
	return "", nil
}

// reconcileImage reconcile the image of the machine
func reconcileImage(ctx context.Context, machineScope *scope.MachineScope, imageSvc compute.OscImageInterface) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	imageSpec := machineScope.GetImage()

	imageRef := machineScope.GetImageRef()
	if len(imageRef.ResourceMap) > 0 {
		log.V(4).Info("Image found in resource map")
		return reconcile.Result{}, nil
	}

	var image *osc.Image
	var err error
	if imageSpec.Name != "" {
		if imageSpec.AccountId == "" {
			log.V(2).Info("[security] It is recommended to set the image account to control the origin of the image.")
		}
		image, err = imageSvc.GetImageByName(ctx, imageSpec.Name, imageSpec.AccountId)
	} else {
		image, err = imageSvc.GetImage(ctx, machineScope.GetImageId())
	}
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("cannot get image: %w", err)
	}
	if image == nil {
		return reconcile.Result{}, errors.New("no image found")
	}
	imageName, imageId := image.GetImageName(), image.GetImageId()
	log.V(3).Info("Image found", "name", imageName, "id", imageId)

	if imageRef.ResourceMap == nil {
		imageRef.ResourceMap = make(map[string]string)
	}
	// TODO: it might be better to use a constant key, not the image name
	// TODO: check use of imageSpec.ResourceId, as imageId is not set on the vm
	if imageSpec.ResourceId != "" {
		imageRef.ResourceMap[imageName] = imageSpec.ResourceId
	} else {
		machineScope.SetImageId(imageId)
		imageRef.ResourceMap[imageName] = imageId
	}

	return reconcile.Result{}, nil
}
