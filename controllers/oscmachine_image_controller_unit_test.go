package controllers

import (
	"context"
	"fmt"

	"testing"

	osc "github.com/outscale/osc-sdk-go/v2"
	"github.com/stretchr/testify/assert"

	"github.com/golang/mock/gomock"
	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/compute/mock_compute"
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
	defaultImageClusterReconcile = infrastructurev1beta1.OscClusterSpec{
		Network: infrastructurev1beta1.OscNetwork{
			Net: infrastructurev1beta1.OscNet{
				Name:       "test-net",
				IpRange:    "10.0.0.0/16",
				ResourceId: "vpc-test-net-uid",
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

	defaultImageReconcile = infrastructurev1beta1.OscMachineSpec{
		Node: infrastructurev1beta1.OscNode{
			Image: infrastructurev1beta1.OscImage{
				Name:       "test-image",
				ResourceId: "test-image-uid",
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
			expGetImageResourceIdErr: fmt.Errorf("test-image does not exist"),
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
			if err != nil {
				assert.Equal(t, itc.expGetImageResourceIdErr.Error(), err.Error(), "GetImageResourceId() should return the same error")
			} else {
				assert.Nil(t, itc.expGetImageResourceIdErr)
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
			expCheckImageFormatParametersErr: fmt.Errorf("Invalid Image Name"),
		},
	}
	for _, itc := range imageTestCases {
		t.Run(itc.name, func(t *testing.T) {
			_, machineScope := SetupMachine(t, itc.name, itc.clusterSpec, itc.machineSpec)
			imageName, err := checkImageFormatParameters(machineScope)
			if err != nil {
				assert.Equal(t, itc.expCheckImageFormatParametersErr, err, "checkImageFormatParameters() should return the same error")
			} else {
				assert.Nil(t, itc.expCheckImageFormatParametersErr)
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
			expReconcileImageErr: nil,
		},
		{
			name: "failed to get Image",
			spec: defaultImageClusterInitialize,
			machineSpec: infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{},
			},
			expImageFound:        false,
			expReconcileImageErr: fmt.Errorf("GetImage generic error"),
		},
	}
	for _, itc := range imageTestCases {
		t.Run(itc.name, func(t *testing.T) {
			_, machineScope, ctx, mockOscimageInterface := SetupWithImageMock(t, itc.name, itc.spec, itc.machineSpec)
			imageSpec := itc.machineSpec.Node.Image
			imageName := itc.machineSpec.Node.Image.Name
			imageId := "omi-" + imageName
			imageRef := machineScope.GetImageRef()
			imageRef.ResourceMap = make(map[string]string)
			imageRef.ResourceMap[imageName] = imageId
			image := osc.ReadImagesResponse{
				Images: &[]osc.Image{
					{
						ImageId: &imageId,
					},
				},
			}
			imageSpec.ResourceId = imageName

			if itc.expImageFound {
				mockOscimageInterface.
					EXPECT().
					GetImageId(gomock.Eq(imageName)).
					Return(imageId, itc.expReconcileImageErr)
				mockOscimageInterface.
					EXPECT().
					GetImage(gomock.Eq(imageId)).
					Return(&(*image.Images)[0], itc.expReconcileImageErr)

			} else {
				mockOscimageInterface.
					EXPECT().
					GetImageId(gomock.Eq(imageName)).
					Return("", itc.expReconcileImageErr)
			}

			reconcileImage, err := reconcileImage(ctx, machineScope, mockOscimageInterface)
			if err != nil {
				assert.Equal(t, itc.expReconcileImageErr.Error(), err.Error(), "reconcileImage() should return the same error")
			} else {
				assert.Nil(t, itc.expReconcileImageErr)
			}
			t.Logf("find reconcileKeyPair %v\n", reconcileImage)
		})
	}
}
