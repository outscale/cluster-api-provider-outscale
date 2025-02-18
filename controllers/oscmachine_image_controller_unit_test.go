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
	"testing"

	"github.com/golang/mock/gomock"
	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/compute/mock_compute"
	osc "github.com/outscale/osc-sdk-go/v2"
	"github.com/stretchr/testify/require"
)

var (
	defaultImageClusterInitialize = infrastructurev1beta1.OscClusterSpec{
		Network: infrastructurev1beta1.OscNetwork{
			Net: infrastructurev1beta1.OscNet{
				Name:    "test-net",
				IpRange: "10.0.0.0/16",
			},
		},
	}
	defaultImageInitialize = infrastructurev1beta1.OscMachineSpec{
		Node: infrastructurev1beta1.OscNode{
			Image: infrastructurev1beta1.OscImage{
				Name: "test-image",
			},
		},
	}
)

// SetupWithImageMock set imageMock with clusterScope and osccluster
func SetupWithImageMock(t *testing.T, name string, spec infrastructurev1beta1.OscClusterSpec, machineSpec infrastructurev1beta1.OscMachineSpec) (clusterScope *scope.ClusterScope, machineScope *scope.MachineScope, ctx context.Context, mockOscImageInterface *mock_compute.MockOscImageInterface) {
	clusterScope, machineScope = SetupMachine(t, name, spec, machineSpec)
	mockCtrl := gomock.NewController(t)
	mockOscImageInterface = mock_compute.NewMockOscImageInterface(mockCtrl)
	ctx = context.Background()
	return clusterScope, machineScope, ctx, mockOscImageInterface
}

// TestGetImageResourceId has several tests to cover the code of the function getImageResourceId
func TestGetImageResourceId(t *testing.T) {
	imageTestCases := []struct {
		name                     string
		spec                     infrastructurev1beta1.OscClusterSpec
		machineSpec              infrastructurev1beta1.OscMachineSpec
		expImageFound            bool
		expGetImageResourceIdErr error
	}{
		{
			name:                     "get ImageId",
			spec:                     defaultImageClusterInitialize,
			machineSpec:              defaultImageInitialize,
			expImageFound:            true,
			expGetImageResourceIdErr: nil,
		},
		{
			name:                     "failed to get ImageId",
			spec:                     defaultImageClusterInitialize,
			machineSpec:              defaultImageInitialize,
			expImageFound:            false,
			expGetImageResourceIdErr: errors.New("test-image does not exist"),
		},
	}
	for _, itc := range imageTestCases {
		t.Run(itc.name, func(t *testing.T) {
			_, machineScope := SetupMachine(t, itc.name, itc.spec, itc.machineSpec)
			imageName := itc.machineSpec.Node.Image.Name
			imageId := "omi-" + imageName
			imageRef := machineScope.GetImageRef()
			imageRef.ResourceMap = make(map[string]string)
			if itc.expImageFound {
				imageRef.ResourceMap[imageName] = imageId
			}
			imageResourceId, err := getImageResourceId(imageName, machineScope)
			if itc.expGetImageResourceIdErr != nil {
				require.EqualError(t, err, itc.expGetImageResourceIdErr.Error(), "GetImageResourceId() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("find imageResourceId %s", imageResourceId)
		})
	}
}

// TestCheckImageFormatParameters has several tests to cover the code of the function checkImageFormatParameters
func TestCheckImageFormatParameters(t *testing.T) {
	imageTestCases := []struct {
		name                             string
		clusterSpec                      infrastructurev1beta1.OscClusterSpec
		machineSpec                      infrastructurev1beta1.OscMachineSpec
		expCheckImageFormatParametersErr error
	}{
		{
			name:        "check Image format",
			clusterSpec: defaultImageClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Image: infrastructurev1beta1.OscImage{
						Name: "test-Image",
					},
				},
			},
			expCheckImageFormatParametersErr: nil,
		},
		{
			name:        "Check work without spec (with default values)",
			clusterSpec: defaultImageClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{},
			},
			expCheckImageFormatParametersErr: nil,
		},
		{
			name:        "Check Bad name image",
			clusterSpec: defaultImageClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Image: infrastructurev1beta1.OscImage{
						Name: "!test-image@Name",
					},
				},
			},
			expCheckImageFormatParametersErr: errors.New("Invalid Image Name"),
		},
	}
	for _, itc := range imageTestCases {
		t.Run(itc.name, func(t *testing.T) {
			_, machineScope := SetupMachine(t, itc.name, itc.clusterSpec, itc.machineSpec)
			imageName, err := checkImageFormatParameters(machineScope)
			if itc.expCheckImageFormatParametersErr != nil {
				require.EqualError(t, err, itc.expCheckImageFormatParametersErr.Error(), "checkImageFormatParameters() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("find imageName %s\n", imageName)
		})
	}
}

// TestReconcileImageGet has several tests to cover the code of the function reconcileImage
func TestReconcileImageGet(t *testing.T) {
	imageTestCases := []struct {
		name                 string
		spec                 infrastructurev1beta1.OscClusterSpec
		machineSpec          infrastructurev1beta1.OscMachineSpec
		expImageFound        bool
		expImageNameFound    bool
		expImageErr          bool
		expGetImageIdErr     error
		expGetImageNameErr   error
		expGetImageErr       error
		expReconcileImageErr error
	}{
		{
			name: "check image exist",
			spec: defaultImageClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Image: infrastructurev1beta1.OscImage{
						Name: "test-image",
					},
				},
			},
			expImageFound:        true,
			expImageNameFound:    true,
			expReconcileImageErr: nil,
			expGetImageIdErr:     nil,
			expGetImageNameErr:   nil,
			expGetImageErr:       nil,
			expImageErr:          false,
		},
		{
			name: "reconcile image",
			spec: defaultImageClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Image: infrastructurev1beta1.OscImage{
						Name:       "test-image",
						ResourceId: "test-image-uid",
					},
					Vm: infrastructurev1beta1.OscVm{
						Name:    "test-vm",
						ImageId: "omi-image",
					},
				},
			},
			expImageFound:        true,
			expImageNameFound:    true,
			expReconcileImageErr: nil,
			expGetImageIdErr:     nil,
			expGetImageNameErr:   nil,
			expGetImageErr:       nil,
			expImageErr:          false,
		},
		{
			name: "failed to get image",
			spec: defaultImageClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Image: infrastructurev1beta1.OscImage{
						Name:       "test-image",
						ResourceId: "test-image-uid",
					},
					Vm: infrastructurev1beta1.OscVm{
						Name:    "test-vm",
						ImageId: "omi-image",
					},
				},
			},
			expImageFound:        true,
			expImageNameFound:    true,
			expGetImageIdErr:     nil,
			expGetImageNameErr:   nil,
			expGetImageErr:       errors.New("GetImage generic error"),
			expReconcileImageErr: errors.New("cannot get image: GetImage generic error"),
			expImageErr:          true,
		},
		{
			name: "find no image",
			spec: defaultImageClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Image: infrastructurev1beta1.OscImage{
						Name:       "test-image",
						ResourceId: "test-image-uid",
					},
					Vm: infrastructurev1beta1.OscVm{
						Name:    "test-vm",
						ImageId: "omi-image",
					},
				},
			},
			expImageFound:        true,
			expImageNameFound:    true,
			expGetImageIdErr:     nil,
			expGetImageNameErr:   nil,
			expGetImageErr:       nil,
			expReconcileImageErr: nil,
			expImageErr:          true,
		},
		{
			name: "failed to get imageName",
			spec: defaultImageClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						ImageId: "omi-image",
					},
				},
			},
			expImageFound:        false,
			expImageNameFound:    false,
			expGetImageIdErr:     nil,
			expGetImageNameErr:   errors.New("GetImageName generic error"),
			expGetImageErr:       nil,
			expReconcileImageErr: errors.New("cannot get image: GetImageName generic error"),
			expImageErr:          false,
		},
		{
			name: "failed to get image",
			spec: defaultImageClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						ImageId: "omi-image",
					},
				},
			},
			expImageFound:        true,
			expImageNameFound:    false,
			expGetImageIdErr:     nil,
			expGetImageNameErr:   nil,
			expGetImageErr:       errors.New("GetImage generic error"),
			expReconcileImageErr: errors.New("cannot get image: GetImage generic error"),
			expImageErr:          false,
		},
		{
			name: "failed to get ImageId",
			spec: defaultImageClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Image: infrastructurev1beta1.OscImage{
						Name: "test-image",
					},
					Vm: infrastructurev1beta1.OscVm{
						Name: "test-vm",
					},
				},
			},
			expImageFound:        false,
			expImageNameFound:    true,
			expImageErr:          false,
			expGetImageIdErr:     errors.New("GetImageId generic error"),
			expGetImageNameErr:   nil,
			expGetImageErr:       nil,
			expReconcileImageErr: errors.New("cannot get image: GetImageId generic error"),
		},
	}
	for _, itc := range imageTestCases {
		t.Run(itc.name, func(t *testing.T) {
			_, machineScope, ctx, mockOscimageInterface := SetupWithImageMock(t, itc.name, itc.spec, itc.machineSpec)
			imageName := itc.machineSpec.Node.Image.Name
			imageId := itc.machineSpec.Node.Vm.ImageId
			imageSpec := machineScope.GetImage()
			imageRef := machineScope.GetImageRef()
			image := osc.ReadImagesResponse{
				Images: &[]osc.Image{
					{
						ImageId: &imageId,
					},
				},
			}
			if len(imageRef.ResourceMap) == 0 {
				imageRef.ResourceMap = make(map[string]string)
			}
			imageSpec.ResourceId = imageId

			if itc.expImageNameFound {
				mockOscimageInterface.
					EXPECT().
					GetImageId(gomock.Any(), gomock.Eq(imageName)).
					Return(imageId, itc.expGetImageIdErr)
			} else {
				mockOscimageInterface.
					EXPECT().
					GetImageName(gomock.Any(), gomock.Eq(imageId)).
					Return(imageName, itc.expGetImageNameErr)
			}
			if itc.expImageFound {
				if itc.expImageErr {
					mockOscimageInterface.
						EXPECT().
						GetImage(gomock.Any(), gomock.Eq(imageId)).
						Return(nil, itc.expGetImageErr)
				} else {
					mockOscimageInterface.
						EXPECT().
						GetImage(gomock.Any(), gomock.Eq(imageId)).
						Return(&(*image.Images)[0], itc.expGetImageErr)
				}
			}

			reconcileImage, err := reconcileImage(ctx, machineScope, mockOscimageInterface)
			if itc.expReconcileImageErr != nil {
				require.EqualError(t, err, itc.expReconcileImageErr.Error(), "reconcileImage() should return the same error")
			} else {
				require.NoError(t, err)
			}
			t.Logf("find reconcileKeyPair %v\n", reconcileImage)
		})
	}
}
