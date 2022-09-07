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

package test

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	network "net"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	gomega "github.com/onsi/gomega"
	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/compute"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/net"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/security"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/service"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/storage"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/controllers"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// deployOscInfraCluster will deploy OscInfraCluster (create osccluster object).
func deployOscInfraCluster(ctx context.Context, infraClusterSpec infrastructurev1beta1.OscClusterSpec, name string, namespace string) (client.Object, client.ObjectKey) {
	By("Deploy oscInfraCluster")
	oscInfraCluster := &infrastructurev1beta1.OscCluster{
		Spec: infraClusterSpec,
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	gomega.Expect(k8sClient.Create(ctx, oscInfraCluster)).To(gomega.Succeed())
	oscInfraClusterKey := client.ObjectKey{Namespace: namespace, Name: name}
	return oscInfraCluster, oscInfraClusterKey
}

// deployOscInfraMachine will deploy OscInfraMachine (create oscmachine object).
func deployOscInfraMachine(ctx context.Context, infraMachineSpec infrastructurev1beta1.OscMachineSpec, name string, namespace string) (client.Object, client.ObjectKey) {
	By("Deploy oscInfraMachine")
	oscInfraMachine := &infrastructurev1beta1.OscMachine{
		Spec: infraMachineSpec,
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	gomega.Expect(k8sClient.Create(ctx, oscInfraMachine)).To(gomega.Succeed())
	oscInfraMachineKey := client.ObjectKey{Namespace: namespace, Name: name}
	return oscInfraMachine, oscInfraMachineKey
}

// createCheckDeleteOscCluster will deploy oscInfraCluster (create osccluster object), deploy capoCluster (create cluster object), will validate each OscInfraCluster component is provisioned and then will delelete OscInfraCluster (delete osccluster) and capoCluster (delete cluster).
func createCheckDeleteOscCluster(ctx context.Context, infraClusterSpec infrastructurev1beta1.OscClusterSpec) {
	oscInfraCluster, oscInfraClusterKey := deployOscInfraCluster(ctx, infraClusterSpec, "cluster-api-test", "default")
	capoCluster, capoClusterKey := deployCapoCluster(ctx, "cluster-api-test", "default")
	waitOscInfraClusterToBeReady(ctx, oscInfraClusterKey)
	waitOscClusterToProvision(ctx, capoClusterKey)
	clusterScope, err := getClusterScope(ctx, capoClusterKey, oscInfraClusterKey)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	checkOscNetToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscSubnetToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscInternetServiceToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscNatServiceToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscPublicIPToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscRouteTableToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscRouteToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscSecurityGroupToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscSecurityGroupRuleToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscLoadBalancerToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	By("Delete cluster")
	deleteObj(ctx, oscInfraCluster, oscInfraClusterKey, "oscInfraCluster", "default")
	deleteObj(ctx, capoCluster, capoClusterKey, "capoCluster", "default")
}

// createCheckDeleteOscClusterMachine will deploy oscInfraCluster (create osccluster object), deploy oscInfraMachine (create oscmachine object),  deploy capoCluster (create cluster object), deploy capoMachine (create machine object), will validate each OscInfraCluster component is provisioned and then will delelete OscInfraCluster (delete osccluster) and capoCluster (delete cluster).
func createCheckDeleteOscClusterMachine(ctx context.Context, infraClusterSpec infrastructurev1beta1.OscClusterSpec, infraMachineSpec infrastructurev1beta1.OscMachineSpec) {
	oscInfraCluster, oscInfraClusterKey := deployOscInfraCluster(ctx, infraClusterSpec, "cluster-api-test", "default")
	capoCluster, capoClusterKey := deployCapoCluster(ctx, "cluster-api-test", "default")
	waitOscInfraClusterToBeReady(ctx, oscInfraClusterKey)
	waitOscClusterToProvision(ctx, capoClusterKey)
	clusterScope, err := getClusterScope(ctx, capoClusterKey, oscInfraClusterKey)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	oscInfraMachine, oscInfraMachineKey := deployOscInfraMachine(ctx, infraMachineSpec, "cluster-api-test", "default")
	capoMachine, capoMachineKey := deployCapoMachine(ctx, "cluster-api-test", "default")
	waitOscInfraMachineToBeReady(ctx, oscInfraMachineKey)
	waitOscMachineToProvision(ctx, capoMachineKey)
	machineScope, err := getMachineScope(ctx, capoMachineKey, capoClusterKey, oscInfraMachineKey, oscInfraClusterKey)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	checkOscNetToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscSubnetToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscInternetServiceToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscNatServiceToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscPublicIPToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscRouteTableToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscRouteToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscSecurityGroupToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscSecurityGroupRuleToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscLoadBalancerToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscVolumeToBeProvisioned(ctx, oscInfraMachineKey, clusterScope, machineScope)
	checkOscVMToBeProvisioned(ctx, oscInfraMachineKey, clusterScope, machineScope)
	WaitControlPlaneDnsNameRegister(clusterScope)
	WaitControlPlaneEndpointUp(clusterScope)
	By("Delete machine")
	deleteObj(ctx, oscInfraMachine, oscInfraMachineKey, "oscInfraMachine", "default")
	deletePatchMachineObj(ctx, capoMachine, capoMachineKey, "capoMachine", "default")
	By("Delete cluster")
	deleteObj(ctx, oscInfraCluster, oscInfraClusterKey, "oscInfraCluster", "default")
	deleteObj(ctx, capoCluster, capoClusterKey, "capoCluster", "default")
}

// deleteObj will delete any kubernetes object.
func deleteObj(ctx context.Context, obj client.Object, key client.ObjectKey, kind string, name string) {
	gomega.Expect(k8sClient.Delete(ctx, obj)).To(gomega.Succeed())
	gomega.EventuallyWithOffset(1, func() error {
		fmt.Fprintf(GinkgoWriter, "Wait %s %s to be deleted\n", kind, name)
		return k8sClient.Get(ctx, key, obj)
	}, 5*time.Minute, 3*time.Second).ShouldNot(gomega.Succeed())
}

// deletePatchMachineObj will delete and patch machine object.
func deletePatchMachineObj(ctx context.Context, obj client.Object, key client.ObjectKey, kind string, name string) {
	gomega.Eventually(func() error {
		return k8sClient.Delete(ctx, obj)
	}, 30*time.Second, 10*time.Second).Should(gomega.Succeed())
	fmt.Fprintf(GinkgoWriter, "Delete Machine pending \n")

	time.Sleep(5 * time.Second)
	updated := &clusterv1.Machine{}
	gomega.Expect(k8sClient.Get(ctx, key, updated)).Should(gomega.Succeed())
	fmt.Fprintf(GinkgoWriter, "Get Machine \n")

	updated.ObjectMeta.Finalizers = nil
	gomega.Expect(k8sClient.Update(ctx, updated)).Should(gomega.Succeed())
	fmt.Fprintf(GinkgoWriter, "Patch machine \n")

	gomega.EventuallyWithOffset(1, func() error {
		fmt.Fprintf(GinkgoWriter, "Wait %s %s to be deleted\n", kind, name)
		return k8sClient.Get(ctx, key, obj)
	}, 5*time.Minute, 3*time.Second).ShouldNot(gomega.Succeed())
}

// deployCapoCluster will deploy capoCluster (create cluster object).
func deployCapoCluster(ctx context.Context, name string, namespace string) (client.Object, client.ObjectKey) {
	By("Deploy capoCluster")
	capoCluster := &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"cluster.x-k8s.io/provider": "infrastructure-outscale",
			},
			Finalizers: []string{"cluster.cluster.x-k8s.io"},
		},
		Spec: clusterv1.ClusterSpec{
			InfrastructureRef: &corev1.ObjectReference{
				APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
				Kind:       "OscCluster",
				Name:       name,
				Namespace:  namespace,
			},
		},
	}
	gomega.Expect(k8sClient.Create(ctx, capoCluster)).To(gomega.Succeed())
	capoClusterKey := client.ObjectKey{Namespace: namespace, Name: name}
	return capoCluster, capoClusterKey
}

// GetControlPlaneEndpoint retrieve control plane endpoint.
func GetControlPlaneEndpoint(clusterScope *scope.ClusterScope) string {
	controlPlaneEndpoint := "https://" + clusterScope.GetControlPlaneEndpointHost() + ":" + fmt.Sprint(clusterScope.GetControlPlaneEndpointPort())
	return controlPlaneEndpoint
}

// GetControlPlaneDnsName retrieve control plane dns name.
func GetControlPlaneDnsName(clusterScope *scope.ClusterScope) string {
	controlPlaneDnsName := clusterScope.GetControlPlaneEndpointHost()
	return controlPlaneDnsName
}

// IsControlPlaneDnsNameRegister validate control plane dns name is registered.
func IsControlPlaneDnsNameRegister(controlPlaneDnsName string) (bool, error) {
	ns, err := network.LookupHost(controlPlaneDnsName)
	if err != nil {
		fmt.Fprintf(GinkgoWriter, "Can not resolve yet controlPlane dns name \n")
		return false, err
	}
	fmt.Fprintf(GinkgoWriter, "Can resolve controlPlane dns name %s \n", ns[0])
	return true, nil
}

// WaitControlPlaneDnsNameRegister wait control plane dns name is registered.
func WaitControlPlaneDnsNameRegister(clusterScope *scope.ClusterScope) {
	By("Wait ControlPlaneDnsName be registered")
	gomega.Eventually(func() (bool, error) {
		controlPlaneDnsName := GetControlPlaneDnsName(clusterScope)
		return IsControlPlaneDnsNameRegister(controlPlaneDnsName)
	}, 2*time.Minute, 5*time.Second).Should(gomega.BeTrue())
}

// IsControlPlaneEndpointUp validate that control plane is up and running.
func IsControlPlaneEndpointUp(controlPlaneEndpoint string) (bool, error) {
	transportCfg := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: transportCfg}
	response, err := client.Get(controlPlaneEndpoint)

	if err != nil {
		fmt.Fprintf(GinkgoWriter, "Can not communicate with control plane %s \n", controlPlaneEndpoint)
		return false, err
	}

	defer response.Body.Close()

	data, err := ioutil.ReadAll(response.Body)
	var res map[string]interface{}
	gomega.Expect(json.Unmarshal([]byte(data), &res)).To(gomega.Succeed())
	if err != nil {
		return false, err
	}
	if res["reason"] == "Forbidden" {
		fmt.Fprintf(GinkgoWriter, "Control plane is up because we received %s\n", res["reason"])
		return true, nil
	}
	fmt.Fprintf(GinkgoWriter, "Control plane is not up yet")

	return false, nil
}

// WaitControlPlaneEndpointUp wait that control plane endpoint.
func WaitControlPlaneEndpointUp(clusterScope *scope.ClusterScope) {
	By("Wait ControlPlaneEndpoint be up")
	gomega.Eventually(func() (bool, error) {
		controlPlaneEndpoint := GetControlPlaneEndpoint(clusterScope)
		return IsControlPlaneEndpointUp(controlPlaneEndpoint)
	}, 10*time.Minute, 15*time.Second).Should(gomega.BeTrue())
}

// deployCapoMachine will deploy capoMachine (create machine object).
func deployCapoMachine(ctx context.Context, name string, namespace string) (client.Object, client.ObjectKey) {
	By("Deploy capoMachine")
	capoMachine := &clusterv1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"cluster.x-k8s.io/provider": "infrastructure-outscale",
			},
			Finalizers: []string{"oscmachine.infrastructure.cluster.x-k8s.io"},
		},
		Spec: clusterv1.MachineSpec{
			ClusterName: name,
			InfrastructureRef: corev1.ObjectReference{
				APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
				Kind:       "OscMachine",
				Name:       name,
				Namespace:  namespace,
			},
			Bootstrap: clusterv1.Bootstrap{
				DataSecretName: pointer.String("cluster-api-test"),
			},
		},
	}
	gomega.Expect(k8sClient.Create(ctx, capoMachine)).To(gomega.Succeed())
	capoMachineKey := client.ObjectKey{Namespace: namespace, Name: name}
	return capoMachine, capoMachineKey

}

// waitOscClusterToProvision will wait that capi will set capoCluster in provisionned phase.
func waitOscClusterToProvision(ctx context.Context, capoClusterKey client.ObjectKey) {
	By("Wait capoCluster to be in provisioned phase")
	gomega.Eventually(func() (string, error) {
		capoCluster := &clusterv1.Cluster{}
		gomega.Expect(k8sClient.Get(ctx, capoClusterKey, capoCluster)).To(gomega.Succeed())
		fmt.Fprintf(GinkgoWriter, "capoClusterPhase: %v\n", capoCluster.Status.Phase)
		return capoCluster.Status.Phase, nil
	}, 2*time.Minute, 3*time.Second).Should(gomega.Equal("Provisioned"))
}

// waitOscMachineToProvision will wait that capi will set capoMachine in provisionned phase.
func waitOscMachineToProvision(ctx context.Context, capoMachineKey client.ObjectKey) {
	By("Wait capoMachine to be in provisioned phase")
	gomega.Eventually(func() (string, error) {
		capoMachine := &clusterv1.Machine{}
		gomega.Expect(k8sClient.Get(ctx, capoMachineKey, capoMachine)).To(gomega.Succeed())
		fmt.Fprintf(GinkgoWriter, "capoMachinePhase: %v\n", capoMachine.Status.Phase)
		return capoMachine.Status.Phase, nil
	}, 8*time.Minute, 15*time.Second).Should(gomega.Equal("Provisioned"))

}

// waitOscClusterToProvision will wait OscInfraCluster to be deployed and ready (object osccluster create with ready status).
func waitOscInfraClusterToBeReady(ctx context.Context, oscInfraClusterKey client.ObjectKey) {
	By("Wait OscInfraCluster to be in ready status")
	gomega.EventuallyWithOffset(1, func() bool {
		oscInfraCluster := &infrastructurev1beta1.OscCluster{}
		gomega.Expect(k8sClient.Get(ctx, oscInfraClusterKey, oscInfraCluster)).To(gomega.Succeed())
		fmt.Fprintf(GinkgoWriter, "oscInfraClusterReady: %v\n", oscInfraCluster.Status.Ready)
		return oscInfraCluster.Status.Ready
	}, 2*time.Minute, 3*time.Second).Should(gomega.BeTrue())
}

// waitOscMachineToProvision will wait OscInfraCluster to be deployed and ready (object oscmachine create with ready status).
func waitOscInfraMachineToBeReady(ctx context.Context, oscInfraMachineKey client.ObjectKey) {
	By("Wait OscInfraMachine to be in ready status")
	gomega.EventuallyWithOffset(1, func() bool {
		oscInfraMachine := &infrastructurev1beta1.OscMachine{}
		gomega.Expect(k8sClient.Get(ctx, oscInfraMachineKey, oscInfraMachine)).To(gomega.Succeed())
		fmt.Fprintf(GinkgoWriter, "oscInfraMachineReady: %v\n", oscInfraMachine.Status.Ready)
		return oscInfraMachine.Status.Ready
	}, 8*time.Minute, 15*time.Second).Should(gomega.BeTrue())
}

// checkOscNetToBeProvisioned will validate that OscNet is provisionned.
func checkOscNetToBeProvisioned(ctx context.Context, oscInfraClusterKey client.ObjectKey, clusterScope *scope.ClusterScope) {
	By("Check OscNet is provisioned")
	gomega.Eventually(func() error {
		netsvc := net.NewService(ctx, clusterScope)
		netSpec := clusterScope.GetNet()
		netID := netSpec.ResourceID
		fmt.Fprintf(GinkgoWriter, "Check NetId %s\n", netID)
		net, err := netsvc.GetNet(netID)
		if err != nil {
			return err
		}
		fmt.Fprintf(GinkgoWriter, "Check NetId has been received %s\n", net.GetNetId())
		if netID != net.GetNetId() {
			return fmt.Errorf("Net %s does not exist", netID)
		}
		fmt.Fprintf(GinkgoWriter, "Found OscNet \n")
		return nil
	}, 1*time.Minute, 1*time.Second).Should(gomega.BeNil())
}

// checkOscVMToBeProvisioned will validate that OscVm is provisionned.
func checkOscVMToBeProvisioned(ctx context.Context, oscInfraMachineKey client.ObjectKey, clusterScope *scope.ClusterScope, machineScope *scope.MachineScope) {
	By("Check OscVM is provisioned")
	gomega.Eventually(func() error {
		vmSvc := compute.NewService(ctx, clusterScope)
		vmSpec := machineScope.GetVM()
		vmId := vmSpec.ResourceID
		fmt.Fprintf(GinkgoWriter, "Check VmId %s\n", vmId)
		vm, err := vmSvc.GetVM(vmId)
		if err != nil {
			return err
		}
		fmt.Fprintf(GinkgoWriter, "Check VmId has been received %s\n", vm.GetVmId())
		if vmId != vm.GetVmId() {
			return fmt.Errorf("Vm %s does not exist", vmId)
		}
		fmt.Fprintf(GinkgoWriter, "Found OscVM \n")
		return nil
	}, 1*time.Minute, 1*time.Second).Should(gomega.BeNil())
}

// checkOscSubnetToBeProvisioned will validate that OscSubnet is provisionned.
func checkOscSubnetToBeProvisioned(ctx context.Context, oscInfraClusterKey client.ObjectKey, clusterScope *scope.ClusterScope) {
	By("Check OscSubnet is provisioned")
	gomega.Eventually(func() error {
		netsvc := net.NewService(ctx, clusterScope)
		netSpec := clusterScope.GetNet()
		subnetsSpec := clusterScope.GetSubnet()
		netID := netSpec.ResourceID
		var subnetIDs []string
		subnetIDs, err := netsvc.GetSubnetIDsFromNetIds(netID)
		if err != nil {
			return err
		}
		fmt.Fprintf(GinkgoWriter, "Check SubnetIDs has been received %v \n", subnetIDs)
		for _, subnetSpec := range subnetsSpec {
			subnetID := subnetSpec.ResourceID

			fmt.Fprintf(GinkgoWriter, "Check SubnetID %s\n", subnetID)
			if !controllers.Contains(subnetIDs, subnetID) {
				return fmt.Errorf("Subnet %s does not exist", subnetID)
			}
		}
		fmt.Fprintf(GinkgoWriter, "Found OscSubnet \n")
		return nil
	}, 1*time.Minute, 1*time.Second).Should(gomega.BeNil())
}

// checkOscInternetServiceToBeProvisioned will validate that OscInternetService is provisionned.
func checkOscInternetServiceToBeProvisioned(ctx context.Context, oscInfraClusterKey client.ObjectKey, clusterScope *scope.ClusterScope) {
	By("Check OscInternetService is provisioned")
	gomega.Eventually(func() error {
		netsvc := net.NewService(ctx, clusterScope)
		internetServiceSpec := clusterScope.GetInternetService()
		internetServiceID := internetServiceSpec.ResourceID
		fmt.Fprintf(GinkgoWriter, "Check InternetServiceId %s\n", internetServiceID)
		internetService, err := netsvc.GetInternetService(internetServiceID)
		if err != nil {
			return err
		}
		fmt.Fprintf(GinkgoWriter, "Check InternetServiceId has been received %s\n", internetService.GetInternetServiceId())
		if internetServiceID != internetService.GetInternetServiceId() {
			return fmt.Errorf("InternetService %s does not exist", internetServiceID)
		}
		fmt.Fprintf(GinkgoWriter, "Found OscInternetService \n")
		return nil
	}, 1*time.Minute, 1*time.Second).Should(gomega.BeNil())
}

// checkOscNatServiceToBeProvisioned will validate that OscNatService is provisionned.
func checkOscNatServiceToBeProvisioned(ctx context.Context, oscInfraClusterKey client.ObjectKey, clusterScope *scope.ClusterScope) {
	By("Check OscNatService is provisioned")
	gomega.Eventually(func() error {
		netsvc := net.NewService(ctx, clusterScope)
		natServiceSpec := clusterScope.GetNatService()
		natServiceID := natServiceSpec.ResourceID
		fmt.Fprintf(GinkgoWriter, "Check NatServiceId %s\n", natServiceID)
		natService, err := netsvc.GetNatService(natServiceID)
		if err != nil {
			return err
		}
		fmt.Fprintf(GinkgoWriter, "Check NatServiceId has been received %s\n", natService.GetNatServiceId())
		if natServiceID != natService.GetNatServiceId() {
			return fmt.Errorf("NatService %s does not exist", natServiceID)
		}
		fmt.Fprintf(GinkgoWriter, "Found OscNatService \n")
		return nil
	}, 1*time.Minute, 1*time.Second).Should(gomega.BeNil())
}

// checkOscPublicIPToBeProvisioned will validate that OscPublicIp is provisionned.
func checkOscPublicIPToBeProvisioned(ctx context.Context, oscInfraClusterKey client.ObjectKey, clusterScope *scope.ClusterScope) {
	By("Check OscPublicIP is provisioned")
	gomega.Eventually(func() error {
		securitysvc := security.NewService(ctx, clusterScope)
		var publicIpsSpec []*infrastructurev1beta1.OscPublicIP = clusterScope.GetPublicIP()
		var publicIPId string
		var publicIPIds []string
		for _, publicIPSpec := range publicIpsSpec {
			publicIPId = publicIPSpec.ResourceID
			publicIPIds = append(publicIPIds, publicIPId)
		}
		validPublicIpIds, err := securitysvc.ValidatePublicIPIds(publicIPIds)
		fmt.Fprintf(GinkgoWriter, "Check PublicIpIds has been received %s\n", validPublicIpIds)
		if err != nil {
			return err
		}
		for _, publicIPSpec := range publicIpsSpec {
			publicIPId = publicIPSpec.ResourceID
			fmt.Fprintf(GinkgoWriter, "Check PublicIpId %s\n", publicIPId)
		}
		if !controllers.Contains(validPublicIpIds, publicIPId) {
			return fmt.Errorf("PublicIpId %s does not exist", publicIPId)
		}
		fmt.Fprintf(GinkgoWriter, "Found OscPublicIP \n")
		return nil
	}, 1*time.Minute, 1*time.Second).Should(gomega.BeNil())
}

// checkOscVolumeToBeProvisioned will validate that OscVolume is provisionned.
func checkOscVolumeToBeProvisioned(ctx context.Context, oscInfraMachineKey client.ObjectKey, clusterScope *scope.ClusterScope, machineScope *scope.MachineScope) {
	By("Check OscVolume is provisioned")
	gomega.Eventually(func() error {
		volumeSvc := storage.NewService(ctx, clusterScope)
		var volumesSpec []*infrastructurev1beta1.OscVolume = machineScope.GetVolume()
		var volumeID string
		var volumeIDs []string
		for _, volumeSpec := range volumesSpec {
			volumeID = volumeSpec.ResourceID
			volumeIDs = append(volumeIDs, volumeID)
		}
		validVolumeIds, err := volumeSvc.ValidateVolumeIds(volumeIDs)
		fmt.Fprintf(GinkgoWriter, "Check VolumeIds has been received %s\n", validVolumeIds)
		if err != nil {
			return err
		}
		for _, volumeSpec := range volumesSpec {
			volumeID := volumeSpec.ResourceID
			fmt.Fprintf(GinkgoWriter, "Check VolumeId %s\n", volumeID)
		}
		if !controllers.Contains(validVolumeIds, volumeID) {
			return fmt.Errorf("VolumeId %s does not exist", volumeID)
		}
		fmt.Fprintf(GinkgoWriter, "Found OscVolume \n")
		return nil
	}, 1*time.Minute, 1*time.Second).Should(gomega.BeNil())
}

// checkOscRouteTableToBeProvisioned will validate that OscRouteTable is provisionned.
func checkOscRouteTableToBeProvisioned(ctx context.Context, oscInfraClusterKey client.ObjectKey, clusterScope *scope.ClusterScope) {
	By("Check OscRouteTable is provisioned")
	gomega.Eventually(func() error {
		netSpec := clusterScope.GetNet()
		netID := netSpec.ResourceID
		securitysvc := security.NewService(ctx, clusterScope)
		routeTablesSpec := clusterScope.GetRouteTables()
		routeTableIDs, err := securitysvc.GetRouteTableIdsFromNetIds(netID)
		fmt.Fprintf(GinkgoWriter, "Check RouteTableIds has been received %v \n", routeTableIDs)

		if err != nil {
			return err
		}

		for _, routeTableSpec := range routeTablesSpec {
			routeTableID := routeTableSpec.ResourceID
			fmt.Fprintf(GinkgoWriter, "Check RouteTableId %s\n", routeTableID)
			if !controllers.Contains(routeTableIDs, routeTableID) {
				return fmt.Errorf("RouteTableId %s does not exist", routeTableID)
			}
		}
		fmt.Fprintf(GinkgoWriter, "Found OscRouteTable \n")
		return nil
	}, 1*time.Minute, 1*time.Second).Should(gomega.BeNil())
}

// checkOscRouteToBeProvisioned will validate that OscRoute is provisionned.
func checkOscRouteToBeProvisioned(ctx context.Context, oscInfraClusterKey client.ObjectKey, clusterScope *scope.ClusterScope) {
	By("Check OscRoute is provisioned")
	gomega.Eventually(func() error {
		securitysvc := security.NewService(ctx, clusterScope)
		routeTablesSpec := clusterScope.GetRouteTables()
		natServiceSpec := clusterScope.GetNatService()
		natServiceID := natServiceSpec.ResourceID
		internetServiceSpec := clusterScope.GetInternetService()
		internetServiceID := internetServiceSpec.ResourceID
		var resourceID string
		for _, routeTableSpec := range routeTablesSpec {
			routeTableID := routeTableSpec.ResourceID
			fmt.Fprintf(GinkgoWriter, "Check RouteTableId %s\n", routeTableID)
			routesSpec := clusterScope.GetRoute(routeTableSpec.Name)
			for _, routeSpec := range *routesSpec {
				routeName := routeSpec.Name + clusterScope.GetUID()
				fmt.Fprintf(GinkgoWriter, "Check Route %s exist \n", routeName)
				resourceType := routeSpec.TargetType
				if resourceType == "gateway" {
					resourceID = internetServiceID
				} else {
					resourceID = natServiceID
				}
				fmt.Fprintf(GinkgoWriter, "Check RouteTable %s %s %s\n", routeTableID, resourceID, resourceType)

				routeTableFromRoute, err := securitysvc.GetRouteTableFromRoute(routeTableID, resourceID, resourceType)
				if err != nil {
					return err
				}
				fmt.Fprintf(GinkgoWriter, "Check RouteTableId has been received %s\n", routeTableFromRoute.GetRouteTableId())
				if routeTableID != routeTableFromRoute.GetRouteTableId() {
					return fmt.Errorf("Route %s does not exist", routeName)
				}
			}
		}
		fmt.Fprintf(GinkgoWriter, "Found OscRoute \n")
		return nil
	}, 1*time.Minute, 1*time.Second).Should(gomega.BeNil())
}

// checkOscSecurityGroupToBeProvisioned will validate that OscSecurityGroup is provisionned.
func checkOscSecurityGroupToBeProvisioned(ctx context.Context, oscInfraClusterKey client.ObjectKey, clusterScope *scope.ClusterScope) {
	By("Check OscSecurityGroup is provisioned")
	gomega.Eventually(func() error {
		netSpec := clusterScope.GetNet()
		netID := netSpec.ResourceID
		securitysvc := security.NewService(ctx, clusterScope)
		securityGroupsSpec := clusterScope.GetSecurityGroups()
		securityGroupIDs, err := securitysvc.GetSecurityGroupIdsFromNetIds(netID)
		fmt.Fprintf(GinkgoWriter, "Check SecurityGroupIds received %v \n", securityGroupIDs)
		if err != nil {
			return err
		}
		for _, securityGroupSpec := range securityGroupsSpec {
			securityGroupID := securityGroupSpec.ResourceID
			fmt.Fprintf(GinkgoWriter, "Check SecurityGroupId %s\n", securityGroupID)
			if !controllers.Contains(securityGroupIDs, securityGroupID) {
				return fmt.Errorf("SecurityGroupId %s does not exist", securityGroupID)
			}
		}
		fmt.Fprintf(GinkgoWriter, "Found OscSecurityGroup \n")
		return nil

	}, 1*time.Minute, 1*time.Second).Should(gomega.BeNil())
}

// checkOscSecurityGroupRuleToBeProvisioned will validate that OscSecurityGroupRule is provisionned.
func checkOscSecurityGroupRuleToBeProvisioned(ctx context.Context, oscInfraClusterKey client.ObjectKey, clusterScope *scope.ClusterScope) {
	By("Check OscSecurityGroupRule is provisioned")
	gomega.Eventually(func() error {
		securitysvc := security.NewService(ctx, clusterScope)
		securityGroupsSpec := clusterScope.GetSecurityGroups()
		for _, securityGroupSpec := range securityGroupsSpec {
			securityGroupID := securityGroupSpec.ResourceID
			fmt.Fprintf(GinkgoWriter, "Check SecurityGroupId %s\n", securityGroupID)
			securityGroupRulesSpec := clusterScope.GetSecurityGroupRule(securityGroupSpec.Name)
			for _, securityGroupRuleSpec := range *securityGroupRulesSpec {
				securityGroupRuleName := securityGroupRuleSpec.Name + "-" + clusterScope.GetUID()
				fmt.Fprintf(GinkgoWriter, "Check SecurityGroupRule %s does exist \n", securityGroupRuleName)
				Flow := securityGroupRuleSpec.Flow
				IPProtocol := securityGroupRuleSpec.IPProtocol
				IPRange := securityGroupRuleSpec.IPRange
				FromPortRange := securityGroupRuleSpec.FromPortRange
				ToPortRange := securityGroupRuleSpec.ToPortRange
				securityGroupFromSecurityGroupRule, err := securitysvc.GetSecurityGroupFromSecurityGroupRule(securityGroupID, Flow, IPProtocol, IPRange, FromPortRange, ToPortRange)
				if err != nil {
					return err
				}
				fmt.Fprintf(GinkgoWriter, "Check SecurityGroupId received %s\n", securityGroupFromSecurityGroupRule.GetSecurityGroupId())
				if securityGroupID != securityGroupFromSecurityGroupRule.GetSecurityGroupId() {
					return fmt.Errorf("SecurityGroupRule %s does not exist", securityGroupRuleName)
				}

			}
		}
		fmt.Fprintf(GinkgoWriter, "Found OscSecurityGroupRule \n")
		return nil
	}, 1*time.Minute, 1*time.Second).Should(gomega.BeNil())
}

// checkOscLoadBalancerToBeProvisioned will validate that OscLoadBalancer is provisionned.
func checkOscLoadBalancerToBeProvisioned(ctx context.Context, oscInfraClusterKey client.ObjectKey, clusterScope *scope.ClusterScope) {
	By("Check OscLoadBalancer is provisioned")
	gomega.Eventually(func() error {
		servicesvc := service.NewService(ctx, clusterScope)
		loadBalancerSpec := clusterScope.GetLoadBalancer()
		loadbalancer, err := servicesvc.GetLoadBalancer(loadBalancerSpec)
		loadBalancerName := loadBalancerSpec.LoadBalancerName
		fmt.Fprintf(GinkgoWriter, "Check loadBalancer %s\n", loadBalancerName)
		if err != nil {
			return err
		}
		if loadbalancer == nil {
			return fmt.Errorf("LoadBalancer %s does not exist", loadBalancerName)
		}
		fmt.Fprintf(GinkgoWriter, "Check loadBalancer has been received %s\n", *loadbalancer.LoadBalancerName)
		fmt.Fprintf(GinkgoWriter, "Found OscLoadBalancer \n")
		return nil
	}, 1*time.Minute, 1*time.Second).Should(gomega.BeNil())
}

// getClusterScope will setup clusterscope use for our functional test.
func getClusterScope(ctx context.Context, capoClusterKey client.ObjectKey, oscInfraClusterKey client.ObjectKey) (clusterScope *scope.ClusterScope, err error) {
	By("Get ClusterScope")
	capoCluster := &clusterv1.Cluster{}
	gomega.Expect(k8sClient.Get(ctx, capoClusterKey, capoCluster)).To(gomega.Succeed())
	oscInfraCluster := &infrastructurev1beta1.OscCluster{}
	gomega.Expect(k8sClient.Get(ctx, oscInfraClusterKey, oscInfraCluster)).To(gomega.Succeed())
	clusterScope, err = scope.NewClusterScope(scope.ClusterScopeParams{
		Client:     k8sClient,
		Cluster:    capoCluster,
		OscCluster: oscInfraCluster,
	})
	return clusterScope, err
}

// getMachineScope will setup machinescope use for our functional test.
func getMachineScope(ctx context.Context, capoMachineKey client.ObjectKey, capoClusterKey client.ObjectKey, oscInfraMachineKey client.ObjectKey, oscInfraClusterKey client.ObjectKey) (machineScope *scope.MachineScope, err error) {
	By("Get MachineScope")
	capoCluster := &clusterv1.Cluster{}
	gomega.Expect(k8sClient.Get(ctx, capoClusterKey, capoCluster)).To(gomega.Succeed())
	capoMachine := &clusterv1.Machine{}
	gomega.Expect(k8sClient.Get(ctx, capoMachineKey, capoMachine)).To(gomega.Succeed())
	oscInfraCluster := &infrastructurev1beta1.OscCluster{}
	gomega.Expect(k8sClient.Get(ctx, oscInfraClusterKey, oscInfraCluster)).To(gomega.Succeed())
	oscInfraMachine := &infrastructurev1beta1.OscMachine{}
	gomega.Expect(k8sClient.Get(ctx, oscInfraMachineKey, oscInfraMachine)).To(gomega.Succeed())
	machineScope, err = scope.NewMachineScope(scope.MachineScopeParams{
		Client:     k8sClient,
		Cluster:    capoCluster,
		Machine:    capoMachine,
		OscCluster: oscInfraCluster,
		OscMachine: oscInfraMachine,
	})
	return machineScope, err
}

var _ = Describe("Outscale Cluster Reconciler", func() {
	BeforeEach(func() {})
	AfterEach(func() {})
	Context("Reconcile an Outscale cluster", func() {
		It("should create a simple cluster", func() {
			ctx := context.Background()
			infraClusterSpec := infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "cluster-api-net",
						IPRange: "10.0.0.0/16",
					},
					Subnets: []*infrastructurev1beta1.OscSubnet{
						{
							Name:          "cluster-api-subnet",
							IPSubnetRange: "10.0.0.0/24",
						},
					},
					InternetService: infrastructurev1beta1.OscInternetService{
						Name: "cluster-api-internetservice",
					},
					NatService: infrastructurev1beta1.OscNatService{
						Name:         "cluster-api-natservice",
						PublicIPName: "cluster-api-publicip",
						SubnetName:   "cluster-api-subnet",
					},
					RouteTables: []*infrastructurev1beta1.OscRouteTable{
						{
							Name:       "cluster-api-routetable",
							SubnetName: "cluster-api-subnet",
							Routes: []infrastructurev1beta1.OscRoute{
								{
									Name:        "cluster-api-routes",
									TargetName:  "cluster-api-internetservice",
									TargetType:  "gateway",
									Destination: "0.0.0.0/0",
								},
							},
						},
					},
					PublicIPS: []*infrastructurev1beta1.OscPublicIP{
						{
							Name: "cluster-api-publicip",
						},
					},
					SecurityGroups: []*infrastructurev1beta1.OscSecurityGroup{
						{
							Name:        "cluster-api-securitygroups",
							Description: "securitygroup",
							SecurityGroupRules: []infrastructurev1beta1.OscSecurityGroupRule{
								{
									Name:          "cluster-api-securitygrouprule",
									Flow:          "Inbound",
									IPProtocol:    "tcp",
									IPRange:       "0.0.0.0/0",
									FromPortRange: 6443,
									ToPortRange:   6443,
								},
							},
						},
					},
					LoadBalancer: infrastructurev1beta1.OscLoadBalancer{
						LoadBalancerName:  "OscSdkExample-8",
						LoadBalancerType:  "internet-facing",
						SubnetName:        "cluster-api-subnet",
						SecurityGroupName: "cluster-api-securitygroups",
					},
				},
			}
			createCheckDeleteOscCluster(ctx, infraClusterSpec)
		})
		It("should create a simple cluster with multi subnet, routeTable, securityGroup", func() {
			ctx := context.Background()
			infraClusterSpec := infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "cluster-api-net",
						IPRange: "10.0.0.0/16",
					},
					Subnets: []*infrastructurev1beta1.OscSubnet{
						{
							Name:          "cluster-api-subnet",
							IPSubnetRange: "10.0.0.0/24",
						},
						{
							Name:          "cluster-api-sub",
							IPSubnetRange: "10.0.1.0/24",
						},
					},
					InternetService: infrastructurev1beta1.OscInternetService{
						Name: "cluster-api-internetservice",
					},
					NatService: infrastructurev1beta1.OscNatService{
						Name:         "cluster-api-natservice",
						PublicIPName: "cluster-api-publicip",
						SubnetName:   "cluster-api-subnet",
					},
					RouteTables: []*infrastructurev1beta1.OscRouteTable{
						{
							Name:       "cluster-api-routetable",
							SubnetName: "cluster-api-subnet",
							Routes: []infrastructurev1beta1.OscRoute{
								{
									Name:        "cluster-api-routes",
									TargetName:  "cluster-api-internetservice",
									TargetType:  "gateway",
									Destination: "0.0.0.0/0",
								},
							},
						},
						{
							Name:       "cluster-api-rt",
							SubnetName: "cluster-api-sub",
							Routes: []infrastructurev1beta1.OscRoute{
								{
									Name:        "cluster-api-r",
									TargetName:  "cluster-api-natservice",
									TargetType:  "nat",
									Destination: "0.0.0.0/0",
								},
							},
						},
					},
					PublicIPS: []*infrastructurev1beta1.OscPublicIP{
						{
							Name: "cluster-api-publicip",
						},
					},
					SecurityGroups: []*infrastructurev1beta1.OscSecurityGroup{
						{
							Name:        "cluster-api-securitygroups",
							Description: "securitygroup",
							SecurityGroupRules: []infrastructurev1beta1.OscSecurityGroupRule{
								{
									Name:          "cluster-api-securitygrouprule",
									Flow:          "Inbound",
									IPProtocol:    "tcp",
									IPRange:       "0.0.0.0/0",
									FromPortRange: 6443,
									ToPortRange:   6443,
								},
								{
									Name:          "cluster-api-securitygrouprule-http",
									Flow:          "Inbound",
									IPProtocol:    "tcp",
									IPRange:       "0.0.0.0/0",
									FromPortRange: 80,
									ToPortRange:   80,
								},
							},
						},
					},
					LoadBalancer: infrastructurev1beta1.OscLoadBalancer{
						LoadBalancerName:  "OscSdkExample-10",
						LoadBalancerType:  "internet-facing",
						SubnetName:        "cluster-api-subnet",
						SecurityGroupName: "cluster-api-securitygroups",
					},
				},
			}
			createCheckDeleteOscCluster(ctx, infraClusterSpec)

		})
		It("should create a simple cluster with default values", func() {
			ctx := context.Background()
			infraClusterSpec := infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name: "cluster-api-net",
					},
				},
			}
			createCheckDeleteOscCluster(ctx, infraClusterSpec)

		})
		It("Should create cluster with machine", func() {
			ctx := context.Background()
			infraClusterSpec := infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    "cluster-api-net",
						IPRange: "10.0.0.0/24",
					},
					Subnets: []*infrastructurev1beta1.OscSubnet{
						{
							Name:          "cluster-api-subnet-kcp",
							IPSubnetRange: "10.0.0.32/28",
						},
						{
							Name:          "cluster-api-subnet-kw",
							IPSubnetRange: "10.0.0.128/26",
						},
						{
							Name:          "cluster-api-subnet-public",
							IPSubnetRange: "10.0.0.8/29",
						},
						{
							Name:          "cluster-api-subnet-nat",
							IPSubnetRange: "10.0.0.0/29",
						},
					},
					InternetService: infrastructurev1beta1.OscInternetService{
						Name: "cluster-api-internetservice",
					},
					NatService: infrastructurev1beta1.OscNatService{
						Name:         "cluster-api-natservice",
						PublicIPName: "cluster-api-publicip-nat",
						SubnetName:   "cluster-api-subnet-nat",
					},
					RouteTables: []*infrastructurev1beta1.OscRouteTable{
						{
							Name:       "cluster-api-routable-kw",
							SubnetName: "cluster-api-subnet-kw",
							Routes: []infrastructurev1beta1.OscRoute{
								{
									Name:        "cluster-api-routes-kw",
									TargetName:  "cluster-api-natservice",
									TargetType:  "nat",
									Destination: "0.0.0.0/0",
								},
							},
						},
						{
							Name:       "cluster-api-routetable-kcp",
							SubnetName: "cluster-api-subnet-kcp",
							Routes: []infrastructurev1beta1.OscRoute{
								{
									Name:        "cluster-api-routes-kcp",
									TargetName:  "cluster-api-natservice",
									TargetType:  "nat",
									Destination: "0.0.0.0/0",
								},
							},
						},
						{
							Name:       "cluster-api-routetable-nat",
							SubnetName: "cluster-api-subnet-nat",
							Routes: []infrastructurev1beta1.OscRoute{
								{
									Name:        "cluster-api-routes-nat",
									TargetName:  "cluster-api-internetservice",
									TargetType:  "gateway",
									Destination: "0.0.0.0/0",
								},
							},
						},
						{
							Name:       "cluster-api-routetable-public",
							SubnetName: "cluster-api-subnet-public",
							Routes: []infrastructurev1beta1.OscRoute{
								{
									Name:        "cluster-api-routes-public",
									TargetName:  "cluster-api-internetservice",
									TargetType:  "gateway",
									Destination: "0.0.0.0/0",
								},
							},
						},
					},
					PublicIPS: []*infrastructurev1beta1.OscPublicIP{
						{
							Name: "cluster-api-publicip-nat",
						},
					},
					SecurityGroups: []*infrastructurev1beta1.OscSecurityGroup{
						{
							Name:        "cluster-api-securitygroups-kw",
							Description: "Security Group with cluster-api",
							SecurityGroupRules: []infrastructurev1beta1.OscSecurityGroupRule{
								{
									Name:          "cluster-api-securitygrouprule-api-kubelet-kw",
									Flow:          "Inbound",
									IPProtocol:    "tcp",
									IPRange:       "10.0.0.128/26",
									FromPortRange: 10250,
									ToPortRange:   10250,
								},
								{
									Name:          "cluster-api-securitygrouprule-api-kubelet-kcp",
									Flow:          "Inbound",
									IPProtocol:    "tcp",
									IPRange:       "10.0.0.32/28",
									FromPortRange: 10250,
									ToPortRange:   10250,
								},
								{
									Name:          "cluster-api-securitygrouprule-nodeip-kw",
									Flow:          "Inbound",
									IPProtocol:    "tcp",
									IPRange:       "10.0.0.128/26",
									FromPortRange: 30000,
									ToPortRange:   32767,
								},
								{
									Name:          "cluster-api-securitygrouprule-nodeip-kcp",
									Flow:          "Inbound",
									IPProtocol:    "tcp",
									IPRange:       "10.0.0.32/28",
									FromPortRange: 30000,
									ToPortRange:   32767,
								},
							},
						},
						{
							Name:        "cluster-api-securitygroups-kcp",
							Description: "Security Group with cluster-api",
							SecurityGroupRules: []infrastructurev1beta1.OscSecurityGroupRule{
								{
									Name:          "cluster-api-securitygrouprule-api-kw",
									Flow:          "Inbound",
									IPProtocol:    "tcp",
									IPRange:       "10.0.0.128/26",
									FromPortRange: 6443,
									ToPortRange:   6443,
								},
								{
									Name:          "cluster-api-securitygrouprule-api-kcp",
									Flow:          "Inbound",
									IPProtocol:    "tcp",
									IPRange:       "10.0.0.32/28",
									FromPortRange: 6443,
									ToPortRange:   6443,
								},
								{
									Name:          "cluster-api-securitygrouprule-etcd",
									Flow:          "Inbound",
									IPProtocol:    "tcp",
									IPRange:       "10.0.0.32/28",
									FromPortRange: 2378,
									ToPortRange:   2379,
								},
								{
									Name:          "cluster-api-securitygrouprule-kubelet-kcp",
									Flow:          "Inbound",
									IPProtocol:    "tcp",
									IPRange:       "10.0.0.32/28",
									FromPortRange: 10250,
									ToPortRange:   10252,
								},
							},
						},
						{
							Name:        "cluster-api-securitygroup-lb",
							Description: "Security Group with cluster-api",
							SecurityGroupRules: []infrastructurev1beta1.OscSecurityGroupRule{
								{
									Name:          "cluster-api-securitygrouprule-lb",
									Flow:          "Inbound",
									IPProtocol:    "tcp",
									IPRange:       "0.0.0.0/0",
									FromPortRange: 6443,
									ToPortRange:   6443,
								},
							},
						},
					},
					LoadBalancer: infrastructurev1beta1.OscLoadBalancer{
						LoadBalancerName:  "osc-k8s-machine",
						LoadBalancerType:  "internet-facing",
						SubnetName:        "cluster-api-subnet-public",
						SecurityGroupName: "cluster-api-securitygroup-lb",
					},
				},
			}

			infraMachineSpec := infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					Volumes: []*infrastructurev1beta1.OscVolume{
						{
							Name:          "cluster-api-volume-kcp",
							Iops:          1000,
							Size:          30,
							VolumeType:    "io1",
							SubregionName: "eu-west-2a",
						},
					},
					VM: infrastructurev1beta1.OscVM{
						Name:             "cluster-api-vm-kcp",
						ImageID:          "ami-6a871c21",
						Role:             "controlplane",
						VolumeName:       "cluster-api-volume-kcp",
						DeviceName:       "/dev/xvdb",
						KeypairName:      "cluster-api",
						SubregionName:    "eu-west-2a",
						SubnetName:       "cluster-api-subnet-kcp",
						LoadBalancerName: "osc-k8s-machine",
						VMType:           "tinav4.c2r4p2",
						SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
							{
								Name: "cluster-api-securitygroups-kcp",
							},
						},
						PrivateIPS: []infrastructurev1beta1.OscPrivateIPElement{
							{
								Name:      "cluster-api-privateip-kcp",
								PrivateIP: "10.0.0.38",
							},
						},
					},
				},
			}
			createCheckDeleteOscClusterMachine(ctx, infraClusterSpec, infraMachineSpec)

		})

	})
})
