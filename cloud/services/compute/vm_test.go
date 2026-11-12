/*
SPDX-FileCopyrightText: 2026 Outscale SAS <opensource@outscale.com>

SPDX-License-Identifier: BSD-3-Clause
*/
package compute_test

import (
	"os"
	"testing"
	"uuid"

	"github.com/outscale/cluster-api-provider-outscale/cloud/services/compute"
	"github.com/outscale/cluster-api-provider-outscale/cloud/services/tag"
	"github.com/outscale/cluster-api-provider-outscale/cloud/tenant"
	"github.com/outscale/goutils/sdk/ptr"
	"github.com/outscale/osc-sdk-go/v3/pkg/osc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestCheckVM(t *testing.T) {
	ctx := ctrl.LoggerInto(t.Context(), klog.Background())

	tenant, err := tenant.Default()
	require.NoError(t, err)
	client := tenant.Client()

	region := os.Getenv("OSC_REGION")

	svc := compute.NewService(tenant, tag.NewService(tenant))

	img, err := svc.GetImageByName(ctx, os.Getenv("IMG_NAME"), compute.OutscaleOpenSourceAccounts[region])
	require.NoError(t, err)
	require.NotNil(t, img)

	id := uuid.New().String()
	createResp, err := client.CreateVms(ctx, osc.CreateVmsRequest{
		ImageId: img.ImageId,
		VmType:  new(os.Getenv("VM_TYPE")),
		Placement: &osc.Placement{
			SubregionName: region + "a",
		},
	})
	require.NoError(t, err)
	require.Len(t, ptr.From(createResp.Vms), 1)
	vm := new(ptr.From(createResp.Vms)[0])

	defer func() {
		_, _ = client.DeleteVms(ctx, osc.DeleteVmsRequest{
			VmIds: []string{vm.VmId},
		})
	}()

	err = svc.CheckVM(ctx, vm, id, *vm.PrivateDnsName)
	require.NoError(t, err)

	readResp, err := client.ReadVms(ctx, osc.ReadVmsRequest{
		Filters: &osc.FiltersVm{
			VmIds: &[]string{vm.VmId},
		},
	})
	require.NoError(t, err)
	require.Len(t, ptr.From(readResp.Vms), 1)
	vm = new(ptr.From(readResp.Vms)[0])
	assert.Len(t, vm.Tags, 2)
	assert.Equal(t, osc.ShutdownBehaviorConfigurationHostActionRestart, *vm.ShutdownBehaviorConfiguration.HostAction)

	err = svc.CheckVM(ctx, vm, id, *vm.PrivateDnsName)
	require.NoError(t, err)
}
