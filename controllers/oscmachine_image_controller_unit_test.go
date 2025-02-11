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

	infrastructurev1beta1 "github.com/outscale/cluster-api-provider-outscale/api/v1beta1"
	"github.com/outscale/cluster-api-provider-outscale/cloud/scope"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/compute/mock_compute"
	osc "github.com/outscale/osc-sdk-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
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
			expCheckImageFormatParametersErr: errors.New("invalid image name"),
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
		expImageByName       bool
		expImageErr          bool
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
			expImageFound:  true,
			expImageByName: true,
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
			expImageFound:  true,
			expImageByName: true,
		},
		{
			name: "reconcile image by name and account_id",
			spec: defaultImageClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Image: infrastructurev1beta1.OscImage{
						Name:      "test-image",
						AccountId: "0123",
					},
					Vm: infrastructurev1beta1.OscVm{
						Name: "test-vm",
					},
				},
			},
			expImageFound:  true,
			expImageByName: true,
		},
		{
			name: "error getting image",
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
			expImageByName:       true,
			expGetImageErr:       errors.New("GetImage generic error"),
			expReconcileImageErr: errors.New("cannot get image: GetImage generic error"),
			expImageErr:          true,
		},
		{
			name: "no image was found",
			spec: defaultImageClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Vm: infrastructurev1beta1.OscVm{
						ImageId: "omi-image",
					},
				},
			},
			expReconcileImageErr: errors.New("no image found"),
		},
		{
			name: "failed to get image by name",
			spec: defaultImageClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Image: infrastructurev1beta1.OscImage{
						Name:      "test-image",
						AccountId: "0123",
					},
					Vm: infrastructurev1beta1.OscVm{
						Name: "test-vm",
					},
				},
			},
			expImageFound:        true,
			expImageByName:       true,
			expGetImageErr:       errors.New("GetImageId generic error"),
			expReconcileImageErr: errors.New("cannot get image: GetImageId generic error"),
		},
	}
	for _, itc := range imageTestCases {
		t.Run(itc.name, func(t *testing.T) {
			_, machineScope, ctx, mockOscimageInterface := SetupWithImageMock(t, itc.name, itc.spec, itc.machineSpec)
			imageName := itc.machineSpec.Node.Image.Name
			imageAccount := itc.machineSpec.Node.Image.AccountId
			imageId := itc.machineSpec.Node.Vm.ImageId
			imageSpec := machineScope.GetImage()
			imageRef := machineScope.GetImageRef()
			if len(imageRef.ResourceMap) == 0 {
				imageRef.ResourceMap = make(map[string]string)
			}
			imageSpec.ResourceId = imageId

			var image *osc.Image
			if itc.expImageFound {
				image = &osc.Image{
					ImageId: &imageId,
				}
			}
			if itc.expImageByName {
				mockOscimageInterface.
					EXPECT().
					GetImageByName(gomock.Any(), gomock.Eq(imageName), gomock.Eq(imageAccount)).
					Return(image, itc.expGetImageErr)
			} else {
				mockOscimageInterface.
					EXPECT().
					GetImage(gomock.Any(), gomock.Eq(imageId)).
					Return(image, itc.expGetImageErr)
			}

			reconcileImage, err := reconcileImage(ctx, machineScope, mockOscimageInterface)
			if itc.expReconcileImageErr != nil {
				require.EqualError(t, err, itc.expReconcileImageErr.Error(), "reconcileImage() should return the right error")
			} else {
				require.NoError(t, err)
			}
			assert.Zero(t, reconcileImage)
		})
	}
}
