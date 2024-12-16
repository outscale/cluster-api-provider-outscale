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
	"os"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/compute"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/net"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/security"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/service"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/storage"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/controllers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// deployOscInfraCluster will deploy OscInfraCluster (create osccluster object)
func deployOscInfraCluster(ctx context.Context, infraClusterSpec infrastructurev1beta1.OscClusterSpec, name string, namespace string) (client.Object, client.ObjectKey) {
	By("Deploy oscInfraCluster")
	oscInfraCluster := &infrastructurev1beta1.OscCluster{
		Spec: infraClusterSpec,
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	Expect(k8sClient.Create(ctx, oscInfraCluster)).To(Succeed())
	oscInfraClusterKey := client.ObjectKey{Namespace: namespace, Name: name}
	return oscInfraCluster, oscInfraClusterKey
}

// deployOscInfraMachine will deploy OscInfraMachine (create oscmachine object)
func deployOscInfraMachine(ctx context.Context, infraMachineSpec infrastructurev1beta1.OscMachineSpec, name string, namespace string) (client.Object, client.ObjectKey) {
	By("Deploy oscInfraMachine")
	oscInfraMachine := &infrastructurev1beta1.OscMachine{
		Spec: infraMachineSpec,
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	Expect(k8sClient.Create(ctx, oscInfraMachine)).To(Succeed())
	oscInfraMachineKey := client.ObjectKey{Namespace: namespace, Name: name}
	return oscInfraMachine, oscInfraMachineKey
}

// createCheckDeleteOscCluster will deploy oscInfraCluster (create osccluster object), deploy capoCluster (create cluster object), will validate each OscInfraCluster component is provisioned and then will delelete OscInfraCluster (delete osccluster) and capoCluster (delete cluster)
func createCheckDeleteOscCluster(ctx context.Context, infraClusterSpec infrastructurev1beta1.OscClusterSpec) {
	uid := uuid.New().String()[:8]
	clusterName := fmt.Sprintf("cluster-api-test-%s", uid)
	oscInfraCluster, oscInfraClusterKey := deployOscInfraCluster(ctx, infraClusterSpec, clusterName, "default")
	capoCluster, capoClusterKey := deployCapoCluster(ctx, clusterName, "default")
	waitOscInfraClusterToBeReady(ctx, oscInfraClusterKey)
	waitOscClusterToProvision(ctx, capoClusterKey)
	clusterScope, err := getClusterScope(ctx, capoClusterKey, oscInfraClusterKey)
	Expect(err).ShouldNot(HaveOccurred())
	checkOscNetToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscSubnetToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscInternetServiceToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscNatServiceToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscPublicIpToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscRouteTableToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscRouteToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscSecurityGroupToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscSecurityGroupRuleToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscLoadBalancerToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	By("Delete cluster")
	deleteObj(ctx, oscInfraCluster, oscInfraClusterKey, "oscInfraCluster", "default")
	deleteObj(ctx, capoCluster, capoClusterKey, "capoCluster", "default")
}

// createCheckDeleteOscClusterMachine will deploy oscInfraCluster (create osccluster object), deploy oscInfraMachine (create oscmachine object),  deploy capoCluster (create cluster object), deploy capoMachine (create machine object), will validate each OscInfraCluster component is provisioned and then will delelete OscInfraCluster (delete osccluster) and capoCluster (delete cluster)
func createCheckDeleteOscClusterMachine(ctx context.Context, infraClusterSpec infrastructurev1beta1.OscClusterSpec, infraMachineSpec infrastructurev1beta1.OscMachineSpec) {
	oscInfraCluster, oscInfraClusterKey := deployOscInfraCluster(ctx, infraClusterSpec, "cluster-api-test", "default")
	capoCluster, capoClusterKey := deployCapoCluster(ctx, "cluster-api-test", "default")
	waitOscInfraClusterToBeReady(ctx, oscInfraClusterKey)
	waitOscClusterToProvision(ctx, capoClusterKey)
	clusterScope, err := getClusterScope(ctx, capoClusterKey, oscInfraClusterKey)
	Expect(err).ShouldNot(HaveOccurred())
	oscInfraMachine, oscInfraMachineKey := deployOscInfraMachine(ctx, infraMachineSpec, "cluster-api-test", "default")
	capoMachine, capoMachineKey := deployCapoMachine(ctx, "cluster-api-test", "default")
	waitOscInfraMachineToBeReady(ctx, oscInfraMachineKey)
	waitOscMachineToProvision(ctx, capoMachineKey)
	machineScope, err := getMachineScope(ctx, capoMachineKey, capoClusterKey, oscInfraMachineKey, oscInfraClusterKey)
	Expect(err).ShouldNot(HaveOccurred())
	checkOscNetToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscSubnetToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscInternetServiceToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscNatServiceToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscPublicIpToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscRouteTableToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscRouteToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscSecurityGroupToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscSecurityGroupRuleToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscLoadBalancerToBeProvisioned(ctx, oscInfraClusterKey, clusterScope)
	checkOscVmToBeProvisioned(ctx, oscInfraMachineKey, clusterScope, machineScope)
	WaitControlPlaneDnsNameRegister(clusterScope)
	WaitControlPlaneEndpointUp(clusterScope)
	By("Delete machine")
	deleteObj(ctx, oscInfraMachine, oscInfraMachineKey, "oscInfraMachine", "default")
	deletePatchMachineObj(ctx, capoMachine, capoMachineKey, "capoMachine", "default")
	By("Delete cluster")
	deleteObj(ctx, oscInfraCluster, oscInfraClusterKey, "oscInfraCluster", "default")
	deleteObj(ctx, capoCluster, capoClusterKey, "capoCluster", "default")
}

// deleteObj will delete any kubernetes object
func deleteObj(ctx context.Context, obj client.Object, key client.ObjectKey, kind string, name string) {
	Expect(k8sClient.Delete(ctx, obj)).To(Succeed())
	EventuallyWithOffset(1, func() error {
		fmt.Fprintf(GinkgoWriter, "Wait %s %s to be deleted\n", kind, name)
		return k8sClient.Get(ctx, key, obj)
	}, 5*time.Minute, 3*time.Second).ShouldNot(Succeed())
}

// deletePatchMachineObj will delete and patch machine object
func deletePatchMachineObj(ctx context.Context, obj client.Object, key client.ObjectKey, kind string, name string) {
	Eventually(func() error {
		return k8sClient.Delete(ctx, obj)
	}, 5*time.Minute, 3*time.Second).Should(Succeed())
	fmt.Fprintf(GinkgoWriter, "Delete Machine pending \n")

	time.Sleep(5 * time.Second)
	updated := &clusterv1.Machine{}
	Expect(k8sClient.Get(ctx, key, updated)).Should(Succeed())
	fmt.Fprintf(GinkgoWriter, "Get Machine \n")

	updated.ObjectMeta.Finalizers = nil
	Expect(k8sClient.Update(ctx, updated)).Should(Succeed())
	fmt.Fprintf(GinkgoWriter, "Patch machine \n")

	EventuallyWithOffset(1, func() error {
		fmt.Fprintf(GinkgoWriter, "Wait %s %s to be deleted\n", kind, name)
		return k8sClient.Get(ctx, key, obj)
	}, 5*time.Minute, 3*time.Second).ShouldNot(Succeed())
}

// deployCapoCluster will deploy capoCluster (create cluster object)
func deployCapoCluster(ctx context.Context, name string, namespace string) (client.Object, client.ObjectKey) {
	By("Deploy capoCluster")
	capoCluster := &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"cluster.x-k8s.io/provider": "infrastructure-outscale",
				"cni":                       "cluster-api-test-crs-cni",
				"ccm":                       "cluster-api-test-crs-ccm",
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
	Expect(k8sClient.Create(ctx, capoCluster)).To(Succeed())
	capoClusterKey := client.ObjectKey{Namespace: namespace, Name: name}
	return capoCluster, capoClusterKey
}

// GetControlPlaneEndpoint retrieve control plane endpoint
func GetControlPlaneEndpoint(clusterScope *scope.ClusterScope) string {
	controlPlaneEndpoint := "https://" + clusterScope.GetControlPlaneEndpointHost() + ":" + fmt.Sprint(clusterScope.GetControlPlaneEndpointPort())
	return controlPlaneEndpoint
}

// GetControlPlaneDnsName retrieve control plane dns name
func GetControlPlaneDnsName(clusterScope *scope.ClusterScope) string {
	controlPlaneDnsName := clusterScope.GetControlPlaneEndpointHost()
	return controlPlaneDnsName
}

// IsControlPlaneDnsNameRegister validate control plane dns name is registered
func IsControlPlaneDnsNameRegister(controlPlaneDnsName string) (bool, error) {
	ns, err := network.LookupHost(controlPlaneDnsName)
	if err != nil {
		fmt.Fprintf(GinkgoWriter, "Can not resolve yet controlPlane dns name \n")
		return false, err
	}
	fmt.Fprintf(GinkgoWriter, "Can resolve controlPlane dns name %s \n", ns[0])
	return true, nil
}

// WaitControlPlaneDnsNameRegister wait control plane dns name is registered
func WaitControlPlaneDnsNameRegister(clusterScope *scope.ClusterScope) {
	By("Wait ControlPlaneDnsName be registered")
	Eventually(func() (bool, error) {
		controlPlaneDnsName := GetControlPlaneDnsName(clusterScope)
		return IsControlPlaneDnsNameRegister(controlPlaneDnsName)
	}, 5*time.Minute, 20*time.Second).Should(BeTrue())
}

// IsControlPlaneEndpointUp validate that control plane is up and running
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
	json.Unmarshal([]byte(data), &res)
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

// WaitControlPlaneEndpointUp wait that control plane endpoint
func WaitControlPlaneEndpointUp(clusterScope *scope.ClusterScope) {
	By("Wait ControlPlaneEndpoint be up")
	Eventually(func() (bool, error) {
		controlPlaneEndpoint := GetControlPlaneEndpoint(clusterScope)
		return IsControlPlaneEndpointUp(controlPlaneEndpoint)
	}, 10*time.Minute, 15*time.Second).Should(BeTrue())
}

// deployCapoMachine will deploy capoMachine (create machine object)
func deployCapoMachine(ctx context.Context, name string, namespace string) (client.Object, client.ObjectKey) {
	By("Deploy capoMachine")
	capoMachine := &clusterv1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"cluster.x-k8s.io/provider":      "infrastructure-outscale",
				"cluster.x-k8s.io/control-plane": "",
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
	Expect(k8sClient.Create(ctx, capoMachine)).To(Succeed())
	capoMachineKey := client.ObjectKey{Namespace: namespace, Name: name}
	return capoMachine, capoMachineKey

}

// waitOscClusterToProvision will wait that capi will set capoCluster in provisionned phase
func waitOscClusterToProvision(ctx context.Context, capoClusterKey client.ObjectKey) {
	By("Wait capoCluster to be in provisioned phase")
	Eventually(func() (string, error) {
		capoCluster := &clusterv1.Cluster{}
		k8sClient.Get(ctx, capoClusterKey, capoCluster)
		fmt.Fprintf(GinkgoWriter, "capoClusterPhase: %v\n", capoCluster.Status.Phase)
		return capoCluster.Status.Phase, nil
	}, 5*time.Minute, 3*time.Second).Should(Equal("Provisioned"))
}

// waitOscMachineToProvision will wait that capi will set capoMachine in provisionned phase
func waitOscMachineToProvision(ctx context.Context, capoMachineKey client.ObjectKey) {
	By("Wait capoMachine to be in provisioned phase")
	Eventually(func() (string, error) {
		capoMachine := &clusterv1.Machine{}
		k8sClient.Get(ctx, capoMachineKey, capoMachine)
		fmt.Fprintf(GinkgoWriter, "capoMachinePhase: %v\n", capoMachine.Status.Phase)
		return capoMachine.Status.Phase, nil
	}, 8*time.Minute, 15*time.Second).Should(Equal("Provisioned"))

}

// waitOscClusterToProvision will wait OscInfraCluster to be deployed and ready (object osccluster create with ready status)
func waitOscInfraClusterToBeReady(ctx context.Context, oscInfraClusterKey client.ObjectKey) {
	By("Wait OscInfraCluster to be in ready status")
	EventuallyWithOffset(1, func() bool {
		oscInfraCluster := &infrastructurev1beta1.OscCluster{}
		k8sClient.Get(ctx, oscInfraClusterKey, oscInfraCluster)
		fmt.Fprintf(GinkgoWriter, "oscInfraClusterReady: %v\n", oscInfraCluster.Status.Ready)
		return oscInfraCluster.Status.Ready
	}, 5*time.Minute, 3*time.Second).Should(BeTrue())
}

// waitOscMachineToProvision will wait OscInfraCluster to be deployed and ready (object oscmachine create with ready status)
func waitOscInfraMachineToBeReady(ctx context.Context, oscInfraMachineKey client.ObjectKey) {
	By("Wait OscInfraMachine to be in ready status")
	EventuallyWithOffset(1, func() bool {
		oscInfraMachine := &infrastructurev1beta1.OscMachine{}
		k8sClient.Get(ctx, oscInfraMachineKey, oscInfraMachine)
		fmt.Fprintf(GinkgoWriter, "oscInfraMachineReady: %v\n", oscInfraMachine.Status.Ready)
		return oscInfraMachine.Status.Ready
	}, 8*time.Minute, 15*time.Second).Should(BeTrue())
}

// checkOscNetToBeProvisioned will validate that OscNet is provisionned
func checkOscNetToBeProvisioned(ctx context.Context, oscInfraClusterKey client.ObjectKey, clusterScope *scope.ClusterScope) {
	By("Check OscNet is provisioned")
	Eventually(func() error {
		netsvc := net.NewService(ctx, clusterScope)
		netSpec := clusterScope.GetNet()
		netId := netSpec.ResourceId
		fmt.Fprintf(GinkgoWriter, "Check NetId %s\n", netId)
		net, err := netsvc.GetNet(netId)
		if err != nil {
			return err
		}
		fmt.Fprintf(GinkgoWriter, "Check NetId has been received %s\n", net.GetNetId())
		if netId != net.GetNetId() {
			return fmt.Errorf("Net %s does not exist", netId)
		}
		fmt.Fprintf(GinkgoWriter, "Found OscNet \n")
		return nil
	}, 5*time.Minute, 1*time.Second).Should(BeNil())
}

// checkOscVmToBeProvisioned will validate that OscVm is provisionned
func checkOscVmToBeProvisioned(ctx context.Context, oscInfraMachineKey client.ObjectKey, clusterScope *scope.ClusterScope, machineScope *scope.MachineScope) {
	By("Check OscVm is provisioned")
	Eventually(func() error {
		vmSvc := compute.NewService(ctx, clusterScope)
		vmSpec := machineScope.GetVm()
		vmId := vmSpec.ResourceId
		fmt.Fprintf(GinkgoWriter, "Check VmId %s\n", vmId)
		vm, err := vmSvc.GetVm(vmId)
		if err != nil {
			return err
		}
		fmt.Fprintf(GinkgoWriter, "Check VmId has been received %s\n", vm.GetVmId())
		if vmId != vm.GetVmId() {
			return fmt.Errorf("Vm %s does not exist", vmId)
		}
		fmt.Fprintf(GinkgoWriter, "Found OscVm \n")
		return nil
	}, 5*time.Minute, 1*time.Second).Should(BeNil())
}

// checkOscSubnetToBeProvisioned will validate that OscSubnet is provisionned
func checkOscSubnetToBeProvisioned(ctx context.Context, oscInfraClusterKey client.ObjectKey, clusterScope *scope.ClusterScope) {
	By("Check OscSubnet is provisioned")
	Eventually(func() error {
		netsvc := net.NewService(ctx, clusterScope)
		netSpec := clusterScope.GetNet()
		subnetsSpec := clusterScope.GetSubnet()
		netId := netSpec.ResourceId
		var subnetIds []string
		subnetIds, err := netsvc.GetSubnetIdsFromNetIds(netId)
		if err != nil {
			return err
		}
		fmt.Fprintf(GinkgoWriter, "Check SubnetIds has been received %v \n", subnetIds)
		for _, subnetSpec := range subnetsSpec {
			subnetId := subnetSpec.ResourceId

			fmt.Fprintf(GinkgoWriter, "Check SubnetId %s\n", subnetId)
			if !controllers.Contains(subnetIds, subnetId) {
				return fmt.Errorf("Subnet %s does not exist", subnetId)
			}
		}
		fmt.Fprintf(GinkgoWriter, "Found OscSubnet \n")
		return nil
	}, 5*time.Minute, 1*time.Second).Should(BeNil())
}

// checkOscInternetServiceToBeProvisioned will validate that OscInternetService is provisionned
func checkOscInternetServiceToBeProvisioned(ctx context.Context, oscInfraClusterKey client.ObjectKey, clusterScope *scope.ClusterScope) {
	By("Check OscInternetService is provisioned")
	Eventually(func() error {
		netsvc := net.NewService(ctx, clusterScope)
		internetServiceSpec := clusterScope.GetInternetService()
		internetServiceId := internetServiceSpec.ResourceId
		fmt.Fprintf(GinkgoWriter, "Check InternetServiceId %s\n", internetServiceId)
		internetService, err := netsvc.GetInternetService(internetServiceId)
		if err != nil {
			return err
		}
		fmt.Fprintf(GinkgoWriter, "Check InternetServiceId has been received %s\n", internetService.GetInternetServiceId())
		if internetServiceId != internetService.GetInternetServiceId() {
			return fmt.Errorf("InternetService %s does not exist", internetServiceId)
		}
		fmt.Fprintf(GinkgoWriter, "Found OscInternetService \n")
		return nil
	}, 5*time.Minute, 1*time.Second).Should(BeNil())
}

// checkOscNatServiceToBeProvisioned will validate that OscNatService is provisionned
func checkOscNatServiceToBeProvisioned(ctx context.Context, oscInfraClusterKey client.ObjectKey, clusterScope *scope.ClusterScope) {
	By("Check OscNatService is provisioned")
	Eventually(func() error {
		netsvc := net.NewService(ctx, clusterScope)
		natServiceSpec := clusterScope.GetNatService()
		natServiceId := natServiceSpec.ResourceId
		fmt.Fprintf(GinkgoWriter, "Check NatServiceId %s\n", natServiceId)
		natService, err := netsvc.GetNatService(natServiceId)
		if err != nil {
			return err
		}
		fmt.Fprintf(GinkgoWriter, "Check NatServiceId has been received %s\n", natService.GetNatServiceId())
		if natServiceId != natService.GetNatServiceId() {
			return fmt.Errorf("NatService %s does not exist", natServiceId)
		}
		fmt.Fprintf(GinkgoWriter, "Found OscNatService \n")
		return nil
	}, 5*time.Minute, 1*time.Second).Should(BeNil())
}

// checkOscPublicIpToBeProvisioned will validate that OscPublicIp is provisionned
func checkOscPublicIpToBeProvisioned(ctx context.Context, oscInfraClusterKey client.ObjectKey, clusterScope *scope.ClusterScope) {
	By("Check OscPublicIp is provisioned")
	Eventually(func() error {
		securitysvc := security.NewService(ctx, clusterScope)
		var publicIpsSpec []*infrastructurev1beta1.OscPublicIp
		publicIpsSpec = clusterScope.GetPublicIp()
		var publicIpId string
		var publicIpIds []string
		for _, publicIpSpec := range publicIpsSpec {
			publicIpId = publicIpSpec.ResourceId
			publicIpIds = append(publicIpIds, publicIpId)
		}
		validPublicIpIds, err := securitysvc.ValidatePublicIpIds(publicIpIds)
		fmt.Fprintf(GinkgoWriter, "Check PublicIpIds has been received %s\n", validPublicIpIds)
		if err != nil {
			return err
		}
		for _, publicIpSpec := range publicIpsSpec {
			publicIpId = publicIpSpec.ResourceId
			fmt.Fprintf(GinkgoWriter, "Check PublicIpId %s\n", publicIpId)
		}
		if !controllers.Contains(validPublicIpIds, publicIpId) {
			return fmt.Errorf("PublicIpId %s does not exist", publicIpId)
		}
		fmt.Fprintf(GinkgoWriter, "Found OscPublicIp \n")
		return nil
	}, 5*time.Minute, 1*time.Second).Should(BeNil())
}

// checkOscVolumeToBeProvisioned will validate that OscVolume is provisionned
func checkOscVolumeToBeProvisioned(ctx context.Context, oscInfraMachineKey client.ObjectKey, clusterScope *scope.ClusterScope, machineScope *scope.MachineScope) {
	By("Check OscVolume is provisioned")
	Eventually(func() error {
		volumeSvc := storage.NewService(ctx, clusterScope)
		var volumesSpec []*infrastructurev1beta1.OscVolume
		volumesSpec = machineScope.GetVolume()
		var volumeId string
		var volumeIds []string
		for _, volumeSpec := range volumesSpec {
			volumeId = volumeSpec.ResourceId
			volumeIds = append(volumeIds, volumeId)
		}
		validVolumeIds, err := volumeSvc.ValidateVolumeIds(volumeIds)
		fmt.Fprintf(GinkgoWriter, "Check VolumeIds has been received %s\n", validVolumeIds)
		if err != nil {
			return err
		}
		for _, volumeSpec := range volumesSpec {
			volumeId := volumeSpec.ResourceId
			fmt.Fprintf(GinkgoWriter, "Check VolumeId %s\n", volumeId)
		}
		if !controllers.Contains(validVolumeIds, volumeId) {
			return fmt.Errorf("VolumeId %s does not exist", volumeId)
		}
		fmt.Fprintf(GinkgoWriter, "Found OscVolume \n")
		return nil
	}, 5*time.Minute, 1*time.Second).Should(BeNil())
}

// checkOscRouteTableToBeProvisioned will validate that OscRouteTable is provisionned
func checkOscRouteTableToBeProvisioned(ctx context.Context, oscInfraClusterKey client.ObjectKey, clusterScope *scope.ClusterScope) {
	By("Check OscRouteTable is provisioned")
	Eventually(func() error {
		netSpec := clusterScope.GetNet()
		netId := netSpec.ResourceId
		securitysvc := security.NewService(ctx, clusterScope)
		routeTablesSpec := clusterScope.GetRouteTables()
		routeTableIds, err := securitysvc.GetRouteTableIdsFromNetIds(netId)
		fmt.Fprintf(GinkgoWriter, "Check RouteTableIds has been received %v \n", routeTableIds)

		if err != nil {
			return err
		}

		for _, routeTableSpec := range routeTablesSpec {
			routeTableId := routeTableSpec.ResourceId
			fmt.Fprintf(GinkgoWriter, "Check RouteTableId %s\n", routeTableId)
			if !controllers.Contains(routeTableIds, routeTableId) {
				return fmt.Errorf("RouteTableId %s does not exist", routeTableId)
			}
		}
		fmt.Fprintf(GinkgoWriter, "Found OscRouteTable \n")
		return nil
	}, 5*time.Minute, 1*time.Second).Should(BeNil())
}

// checkOscRouteToBeProvisioned will validate that OscRoute is provisionned
func checkOscRouteToBeProvisioned(ctx context.Context, oscInfraClusterKey client.ObjectKey, clusterScope *scope.ClusterScope) {
	By("Check OscRoute is provisioned")
	Eventually(func() error {
		securitysvc := security.NewService(ctx, clusterScope)
		routeTablesSpec := clusterScope.GetRouteTables()
		natServiceSpec := clusterScope.GetNatService()
		natServiceId := natServiceSpec.ResourceId
		internetServiceSpec := clusterScope.GetInternetService()
		internetServiceId := internetServiceSpec.ResourceId
		var resourceId string
		for _, routeTableSpec := range routeTablesSpec {
			routeTableId := routeTableSpec.ResourceId
			fmt.Fprintf(GinkgoWriter, "Check RouteTableId %s\n", routeTableId)
			routesSpec := clusterScope.GetRoute(routeTableSpec.Name)
			for _, routeSpec := range *routesSpec {
				routeName := routeSpec.Name + clusterScope.GetUID()
				fmt.Fprintf(GinkgoWriter, "Check Route %s exist \n", routeName)
				resourceType := routeSpec.TargetType
				if resourceType == "gateway" {
					resourceId = internetServiceId
				} else {
					resourceId = natServiceId
				}
				fmt.Fprintf(GinkgoWriter, "Check RouteTable %s %s %s\n", routeTableId, resourceId, resourceType)

				routeTableFromRoute, err := securitysvc.GetRouteTableFromRoute(routeTableId, resourceId, resourceType)
				if err != nil {
					return err
				}
				fmt.Fprintf(GinkgoWriter, "Check RouteTableId has been received %s\n", routeTableFromRoute.GetRouteTableId())
				if routeTableId != routeTableFromRoute.GetRouteTableId() {
					return fmt.Errorf("Route %s does not exist", routeName)
				}
			}
		}
		fmt.Fprintf(GinkgoWriter, "Found OscRoute \n")
		return nil
	}, 5*time.Minute, 1*time.Second).Should(BeNil())
}

// checkOscSecurityGroupToBeProvisioned will validate that OscSecurityGroup is provisionned
func checkOscSecurityGroupToBeProvisioned(ctx context.Context, oscInfraClusterKey client.ObjectKey, clusterScope *scope.ClusterScope) {
	By("Check OscSecurityGroup is provisioned")
	Eventually(func() error {
		netSpec := clusterScope.GetNet()
		netId := netSpec.ResourceId
		securitysvc := security.NewService(ctx, clusterScope)
		securityGroupsSpec := clusterScope.GetSecurityGroups()
		securityGroupIds, err := securitysvc.GetSecurityGroupIdsFromNetIds(netId)
		fmt.Fprintf(GinkgoWriter, "Check SecurityGroupIds received %v \n", securityGroupIds)
		if err != nil {
			return err
		}
		for _, securityGroupSpec := range securityGroupsSpec {
			securityGroupId := securityGroupSpec.ResourceId
			fmt.Fprintf(GinkgoWriter, "Check SecurityGroupId %s\n", securityGroupId)
			if !controllers.Contains(securityGroupIds, securityGroupId) {
				return fmt.Errorf("SecurityGroupId %s does not exist", securityGroupId)
			}
		}
		fmt.Fprintf(GinkgoWriter, "Found OscSecurityGroup \n")
		return nil

	}, 5*time.Minute, 1*time.Second).Should(BeNil())
}

// checkOscSecurityGroupRuleToBeProvisioned will validate that OscSecurityGroupRule is provisionned
func checkOscSecurityGroupRuleToBeProvisioned(ctx context.Context, oscInfraClusterKey client.ObjectKey, clusterScope *scope.ClusterScope) {
	By("Check OscSecurityGroupRule is provisioned")
	Eventually(func() error {
		securitysvc := security.NewService(ctx, clusterScope)
		securityGroupsSpec := clusterScope.GetSecurityGroups()
		for _, securityGroupSpec := range securityGroupsSpec {
			securityGroupId := securityGroupSpec.ResourceId
			fmt.Fprintf(GinkgoWriter, "Check SecurityGroupId %s\n", securityGroupId)
			securityGroupRulesSpec := clusterScope.GetSecurityGroupRule(securityGroupSpec.Name)
			for _, securityGroupRuleSpec := range *securityGroupRulesSpec {
				securityGroupRuleName := securityGroupRuleSpec.Name + "-" + clusterScope.GetUID()
				fmt.Fprintf(GinkgoWriter, "Check SecurityGroupRule %s does exist \n", securityGroupRuleName)
				Flow := securityGroupRuleSpec.Flow
				IpProtocol := securityGroupRuleSpec.IpProtocol
				IpRange := securityGroupRuleSpec.IpRange
				FromPortRange := securityGroupRuleSpec.FromPortRange
				ToPortRange := securityGroupRuleSpec.ToPortRange
				securityGroupFromSecurityGroupRule, err := securitysvc.GetSecurityGroupFromSecurityGroupRule(securityGroupId, Flow, IpProtocol, IpRange, "", FromPortRange, ToPortRange)
				if err != nil {
					return err
				}
				fmt.Fprintf(GinkgoWriter, "Check SecurityGroupId received %s\n", securityGroupFromSecurityGroupRule.GetSecurityGroupId())
				if securityGroupId != securityGroupFromSecurityGroupRule.GetSecurityGroupId() {
					return fmt.Errorf("SecurityGroupRule %s does not exist", securityGroupRuleName)
				}

			}
		}
		fmt.Fprintf(GinkgoWriter, "Found OscSecurityGroupRule \n")
		return nil
	}, 5*time.Minute, 1*time.Second).Should(BeNil())
}

// checkOscLoadBalancerToBeProvisioned will validate that OscLoadBalancer is provisionned
func checkOscLoadBalancerToBeProvisioned(ctx context.Context, oscInfraClusterKey client.ObjectKey, clusterScope *scope.ClusterScope) {
	By("Check OscLoadBalancer is provisioned")
	Eventually(func() error {
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
	}, 5*time.Minute, 1*time.Second).Should(BeNil())
}

// getClusterScope will setup clusterscope use for our functional test
func getClusterScope(ctx context.Context, capoClusterKey client.ObjectKey, oscInfraClusterKey client.ObjectKey) (clusterScope *scope.ClusterScope, err error) {
	By("Get ClusterScope")
	capoCluster := &clusterv1.Cluster{}
	k8sClient.Get(ctx, capoClusterKey, capoCluster)
	oscInfraCluster := &infrastructurev1beta1.OscCluster{}
	k8sClient.Get(ctx, oscInfraClusterKey, oscInfraCluster)
	clusterScope, err = scope.NewClusterScope(scope.ClusterScopeParams{
		Client:     k8sClient,
		Cluster:    capoCluster,
		OscCluster: oscInfraCluster,
	})
	return clusterScope, err
}

// getMachineScope will setup machinescope use for our functional test
func getMachineScope(ctx context.Context, capoMachineKey client.ObjectKey, capoClusterKey client.ObjectKey, oscInfraMachineKey client.ObjectKey, oscInfraClusterKey client.ObjectKey) (machineScope *scope.MachineScope, err error) {
	By("Get MachineScope")
	capoCluster := &clusterv1.Cluster{}
	k8sClient.Get(ctx, capoClusterKey, capoCluster)
	capoMachine := &clusterv1.Machine{}
	k8sClient.Get(ctx, capoMachineKey, capoMachine)
	oscInfraCluster := &infrastructurev1beta1.OscCluster{}
	k8sClient.Get(ctx, oscInfraClusterKey, oscInfraCluster)
	oscInfraMachine := &infrastructurev1beta1.OscMachine{}
	k8sClient.Get(ctx, oscInfraMachineKey, oscInfraMachine)
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
			uid := uuid.New().String()[:2]

			osc_region, ok := os.LookupEnv("OSC_REGION")
			if !ok {
				osc_region = "eu-west-2"
			}
			osc_subregion, ok := os.LookupEnv("OSC_SUBREGION_NAME")
			if !ok {
				osc_subregion = osc_region + "a"
			}

			infraClusterSpec := infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    fmt.Sprintf("cluster-api-net-%s", uid),
						IpRange: "10.0.0.0/16",
					},
					Subnets: []*infrastructurev1beta1.OscSubnet{
						{
							Name:          fmt.Sprintf("cluster-api-subnet-%s", uid),
							IpSubnetRange: "10.0.0.0/24",
							SubregionName: osc_subregion,
						},
					},
					InternetService: infrastructurev1beta1.OscInternetService{
						Name: fmt.Sprintf("cluster-api-internetservice-%s", uid),
					},
					NatService: infrastructurev1beta1.OscNatService{
						Name:         fmt.Sprintf("cluster-api-natservice-%s", uid),
						PublicIpName: fmt.Sprintf("cluster-api-publicip-%s", uid),
						SubnetName:   fmt.Sprintf("cluster-api-subnet-%s", uid),
					},
					RouteTables: []*infrastructurev1beta1.OscRouteTable{
						{
							Name: fmt.Sprintf("cluster-api-routetable-%s", uid),
							Subnets: []string{
								fmt.Sprintf("cluster-api-subnet-%s", uid),
							},
							Routes: []infrastructurev1beta1.OscRoute{
								{
									Name:        fmt.Sprintf("cluster-api-route-%s", uid),
									TargetName:  fmt.Sprintf("cluster-api-internetservice-%s", uid),
									TargetType:  "gateway",
									Destination: "0.0.0.0/0",
								},
							},
						},
					},
					PublicIps: []*infrastructurev1beta1.OscPublicIp{
						{
							Name: fmt.Sprintf("cluster-api-publicip-%s", uid),
						},
					},
					SecurityGroups: []*infrastructurev1beta1.OscSecurityGroup{
						{
							Name:        fmt.Sprintf("cluster-api-securitygroup-%s", uid),
							Description: "Security group for cluster API",
							SecurityGroupRules: []infrastructurev1beta1.OscSecurityGroupRule{
								{
									Name:          fmt.Sprintf("cluster-api-securitygrouprule-%s", uid),
									Flow:          "Inbound",
									IpProtocol:    "tcp",
									IpRange:       "0.0.0.0/0",
									FromPortRange: 6443,
									ToPortRange:   6443,
								},
							},
						},
					},
					LoadBalancer: infrastructurev1beta1.OscLoadBalancer{
						LoadBalancerName:  fmt.Sprintf("cluster-api-loadbalancer-%s", uid),
						LoadBalancerType:  "internet-facing",
						SubnetName:        fmt.Sprintf("cluster-api-subnet-%s", uid),
						SecurityGroupName: fmt.Sprintf("cluster-api-securitygroup-%s", uid),
					},
				},
			}

			createCheckDeleteOscCluster(ctx, infraClusterSpec)
		})
		It("should create a simple cluster with multi subnet, routeTable, securityGroup", func() {
			ctx := context.Background()
			uid := uuid.New().String()[:2]
			osc_region, ok := os.LookupEnv("OSC_REGION")
			if !ok {
				osc_region = "eu-west-2"
			}
			osc_subregion, ok := os.LookupEnv("OSC_SUBREGION_NAME")
			if !ok {
				osc_subregion = osc_region + "a"
			}
			infraClusterSpec := infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    fmt.Sprintf("cluster-api-net-%s", uid),
						IpRange: "10.0.0.0/16",
					},
					Subnets: []*infrastructurev1beta1.OscSubnet{
						{
							Name:          fmt.Sprintf("cluster-api-subnet-%s", uid),
							IpSubnetRange: "10.0.0.0/24",
							SubregionName: osc_subregion,
						},
						{
							Name:          fmt.Sprintf("cluster-api-sub-%s", uid),
							IpSubnetRange: "10.0.1.0/24",
							SubregionName: osc_subregion,
						},
					},
					InternetService: infrastructurev1beta1.OscInternetService{
						Name: fmt.Sprintf("cluster-api-internetservice-%s", uid),
					},
					NatService: infrastructurev1beta1.OscNatService{
						Name:         fmt.Sprintf("cluster-api-natservice-%s", uid),
						PublicIpName: fmt.Sprintf("cluster-api-publicip-%s", uid),
						SubnetName:   fmt.Sprintf("cluster-api-subnet-%s", uid),
					},
					RouteTables: []*infrastructurev1beta1.OscRouteTable{
						{
							Name: fmt.Sprintf("cluster-api-routetable-%s", uid),
							Subnets: []string{
								"cluster-api-subnet",
							},
							Routes: []infrastructurev1beta1.OscRoute{
								{
									Name:        fmt.Sprintf("cluster-api-routes-%s", uid),
									TargetName:  fmt.Sprintf("cluster-api-internetservice-%s", uid),
									TargetType:  "gateway",
									Destination: "0.0.0.0/0",
								},
							},
						},
						{
							Name: fmt.Sprintf("cluster-api-rt-%s", uid),
							Subnets: []string{
								fmt.Sprintf("cluster-api-sub-%s", uid),
							},
							Routes: []infrastructurev1beta1.OscRoute{
								{
									Name:        fmt.Sprintf("cluster-api-r-%s", uid),
									TargetName:  fmt.Sprintf("cluster-api-natservice-%s", uid),
									TargetType:  "nat",
									Destination: "0.0.0.0/0",
								},
							},
						},
					},
					PublicIps: []*infrastructurev1beta1.OscPublicIp{
						{
							Name: fmt.Sprintf("cluster-api-publicip-%s", uid),
						},
					},
					SecurityGroups: []*infrastructurev1beta1.OscSecurityGroup{
						{
							Name:                      fmt.Sprintf("cluster-api-securitygroups-%s", uid),
							Description:               "Security group for cluster API",
							DeleteDefaultOutboundRule: false, // Do not delete the default outbound rule
							SecurityGroupRules: []infrastructurev1beta1.OscSecurityGroupRule{
								{
									Name:          fmt.Sprintf("inbound-kube-api-%s", uid),
									Flow:          "Inbound",
									IpProtocol:    "tcp",
									IpRange:       "0.0.0.0/0",
									FromPortRange: 6443,
									ToPortRange:   6443,
								},
								{
									Name:                    fmt.Sprintf("cluster-api-securitygrouprule-http-%s", uid),
									Flow:                    "Inbound",
									IpProtocol:              "tcp",
									IpRange:                 "0.0.0.0/0",
									FromPortRange:           80,
									ToPortRange:             80,
									TargetSecurityGroupName: fmt.Sprintf("cluster-api-securitygroups-%s", uid),
								},
								{
									Name:       fmt.Sprintf("outbound-all-%s", uid),
									Flow:       "Outbound",
									IpProtocol: "-1", // All protocols
									IpRange:    "0.0.0.0/0",
								},
							},
						},
					},
					LoadBalancer: infrastructurev1beta1.OscLoadBalancer{
						LoadBalancerName:  fmt.Sprintf("OscSdkExample-10-%s", uid),
						LoadBalancerType:  "internet-facing",
						SubnetName:        fmt.Sprintf("cluster-api-subnet-%s", uid),
						SecurityGroupName: fmt.Sprintf("cluster-api-securitygroups-%s", uid),
					},
				},
			}
			createCheckDeleteOscCluster(ctx, infraClusterSpec)

		})
		It("should create a simple cluster with default values", func() {
			ctx := context.Background()
			uid := uuid.New().String()[:2]
			osc_region, ok := os.LookupEnv("OSC_REGION")
			if !ok {
				osc_region = "eu-west-2"
			}
			osc_subregion, ok := os.LookupEnv("OSC_SUBREGION_NAME")
			if !ok {
				osc_subregion = osc_region + "a"
			}
			infraClusterSpec := infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					SubregionName: osc_subregion,
					Net: infrastructurev1beta1.OscNet{
						Name: fmt.Sprintf("cluster-api-net-%s", uid),
					},
					LoadBalancer: infrastructurev1beta1.OscLoadBalancer{
						LoadBalancerName: fmt.Sprintf("OscSdkExample-10-%s", uid),
					},
				},
			}
			createCheckDeleteOscCluster(ctx, infraClusterSpec)

		})
		It("Should create cluster with machine", func() {
			ctx := context.Background()
			uid := uuid.New().String()[:2]
			osc_region, ok := os.LookupEnv("OSC_REGION")
			if !ok {
				osc_region = "eu-west-2"
			}
			osc_subregion, ok := os.LookupEnv("OSC_SUBREGION_NAME")
			if !ok {
				osc_subregion = osc_region + "a"
			}
			imageId, ok := os.LookupEnv("IMG_UPGRADE_FROM")
			if !ok {
				imageId = "ami-e1a786f1"
			}
			infraClusterSpec := infrastructurev1beta1.OscClusterSpec{
				Network: infrastructurev1beta1.OscNetwork{
					Net: infrastructurev1beta1.OscNet{
						Name:    fmt.Sprintf("cluster-api-net-%s", uid),
						IpRange: "10.0.0.0/16",
					},
					Subnets: []*infrastructurev1beta1.OscSubnet{
						{
							Name:          fmt.Sprintf("cluster-api-subnet-kcp-%s", uid),
							IpSubnetRange: "10.0.4.0/24",
							SubregionName: osc_subregion,
						},
						{
							Name:          fmt.Sprintf("cluster-api-subnet-kw-%s", uid),
							IpSubnetRange: "10.0.3.0/24",
							SubregionName: osc_subregion,
						},
						{
							Name:          fmt.Sprintf("cluster-api-subnet-public-%s", uid),
							IpSubnetRange: "10.0.2.0/24",
							SubregionName: osc_subregion,
						},
					},
					InternetService: infrastructurev1beta1.OscInternetService{
						Name: fmt.Sprintf("cluster-api-internetservice-%s", uid),
					},
					NatService: infrastructurev1beta1.OscNatService{
						Name:         fmt.Sprintf("cluster-api-natservice-%s", uid),
						PublicIpName: fmt.Sprintf("cluster-api-publicip-nat-%s", uid),
						SubnetName:   fmt.Sprintf("cluster-api-subnet-public-%s", uid),
					},
					RouteTables: []*infrastructurev1beta1.OscRouteTable{
						{
							Name: fmt.Sprintf("cluster-api-routable-kw-%s", uid),
							Subnets: []string{
								fmt.Sprintf("cluster-api-subnet-kw-%s", uid),
							},
							Routes: []infrastructurev1beta1.OscRoute{
								{
									Name:        fmt.Sprintf("cluster-api-routes-kw-%s", uid),
									TargetName:  fmt.Sprintf("cluster-api-natservice-%s", uid),
									TargetType:  "nat",
									Destination: "0.0.0.0/0",
								},
							},
						},
						{
							Name: fmt.Sprintf("cluster-api-routetable-kcp-%s", uid),
							Subnets: []string{
								fmt.Sprintf("cluster-api-subnet-kcp-%s", uid),
							},
							Routes: []infrastructurev1beta1.OscRoute{
								{
									Name:        fmt.Sprintf("cluster-api-routes-kcp-%s", uid),
									TargetName:  fmt.Sprintf("cluster-api-natservice-%s", uid),
									TargetType:  "nat",
									Destination: "0.0.0.0/0",
								},
							},
						},
						{
							Name: fmt.Sprintf("cluster-api-routetable-public-%s", uid),
							Subnets: []string{
								fmt.Sprintf("cluster-api-subnet-public-%s", uid),
							},
							Routes: []infrastructurev1beta1.OscRoute{
								{
									Name:        fmt.Sprintf("cluster-api-routes-public-%s", uid),
									TargetName:  fmt.Sprintf("cluster-api-internetservice-%s", uid),
									TargetType:  "gateway",
									Destination: "0.0.0.0/0",
								},
							},
						},
					},
					PublicIps: []*infrastructurev1beta1.OscPublicIp{
						{
							Name: fmt.Sprintf("cluster-api-publicip-nat-%s", uid),
						},
					},
					SecurityGroups: []*infrastructurev1beta1.OscSecurityGroup{
						{
							Name:                      fmt.Sprintf("cluster-api-securitygroups-kw-%s", uid),
							Description:               "Security Group with cluster-api",
							DeleteDefaultOutboundRule: false,
							SecurityGroupRules: []infrastructurev1beta1.OscSecurityGroupRule{
								{
									Name:          fmt.Sprintf("cluster-api-securitygrouprule-api-kubelet-kw-%s", uid),
									Flow:          "Inbound",
									IpProtocol:    "tcp",
									IpRange:       "10.0.3.0/24",
									FromPortRange: 10250,
									ToPortRange:   10250,
								},
								{
									Name:          fmt.Sprintf("cluster-api-securitygrouprule-api-kubelet-kcp-%s", uid),
									Flow:          "Inbound",
									IpProtocol:    "tcp",
									IpRange:       "10.0.4.0/24",
									FromPortRange: 10250,
									ToPortRange:   10250,
								},
								{
									Name:          fmt.Sprintf("cluster-api-securitygrouprule-nodeip-kw-%s", uid),
									Flow:          "Inbound",
									IpProtocol:    "tcp",
									IpRange:       "10.0.3.0/24",
									FromPortRange: 30000,
									ToPortRange:   32767,
								},
								{
									Name:          fmt.Sprintf("cluster-api-securitygrouprule-nodeip-kcp-%s", uid),
									Flow:          "Inbound",
									IpProtocol:    "tcp",
									IpRange:       "10.0.4.0/24",
									FromPortRange: 30000,
									ToPortRange:   32767,
								},
							},
						},
						{
							Name:                      fmt.Sprintf("cluster-api-securitygroups-kcp-%s", uid),
							Description:               "Security Group with cluster-api",
							DeleteDefaultOutboundRule: false,
							SecurityGroupRules: []infrastructurev1beta1.OscSecurityGroupRule{
								{
									Name:          fmt.Sprintf("cluster-api-securitygrouprule-api-kw-%s", uid),
									Flow:          "Inbound",
									IpProtocol:    "tcp",
									IpRange:       "10.0.3.0/24",
									FromPortRange: 6443,
									ToPortRange:   6443,
								},
								{
									Name:          fmt.Sprintf("cluster-api-securitygrouprule-api-kcp-%s", uid),
									Flow:          "Inbound",
									IpProtocol:    "tcp",
									IpRange:       "10.0.4.0/24",
									FromPortRange: 6443,
									ToPortRange:   6443,
								},
								{
									Name:          fmt.Sprintf("cluster-api-securitygrouprule-etcd-%s", uid),
									Flow:          "Inbound",
									IpProtocol:    "tcp",
									IpRange:       "10.0.3.0/24",
									FromPortRange: 2378,
									ToPortRange:   2379,
								},
								{
									Name:          fmt.Sprintf("cluster-api-securitygrouprule-kubelet-kcp-%s", uid),
									Flow:          "Inbound",
									IpProtocol:    "tcp",
									IpRange:       "10.0.4.0/24",
									FromPortRange: 10250,
									ToPortRange:   10252,
								},
							},
						},
						{
							Name:                      fmt.Sprintf("cluster-api-securitygroup-lb-%s", uid),
							Description:               "Security Group with cluster-api",
							DeleteDefaultOutboundRule: false,
							SecurityGroupRules: []infrastructurev1beta1.OscSecurityGroupRule{
								{
									Name:          fmt.Sprintf("cluster-api-securitygrouprule-lb-%s", uid),
									Flow:          "Inbound",
									IpProtocol:    "tcp",
									IpRange:       "0.0.0.0/0",
									FromPortRange: 6443,
									ToPortRange:   6443,
								},
							},
						},
					},
					LoadBalancer: infrastructurev1beta1.OscLoadBalancer{
						LoadBalancerName:  fmt.Sprintf("osc-k8s-machine-%s", uid),
						LoadBalancerType:  "internet-facing",
						SubnetName:        fmt.Sprintf("cluster-api-subnet-public-%s", uid),
						SecurityGroupName: fmt.Sprintf("cluster-api-securitygroup-lb-%s", uid),
					},
				},
			}
			infraMachineSpec := infrastructurev1beta1.OscMachineSpec{
				Node: infrastructurev1beta1.OscNode{
					KeyPair: infrastructurev1beta1.OscKeypair{
						Name:          fmt.Sprintf("cluster-api-testenv-%s", uid),
						DeleteKeypair: false,
					},
					Vm: infrastructurev1beta1.OscVm{
						Name:          fmt.Sprintf("cluster-api-vm-kcp-%s", uid),
						Role:          "controlplane",
						ImageId:       imageId,
						DeviceName:    "/dev/sda1",
						KeypairName:   fmt.Sprintf("cluster-api-testenv-%s", uid),
						SubregionName: osc_subregion,
						RootDisk: infrastructurev1beta1.OscRootDisk{
							RootDiskSize: 30,
							RootDiskIops: 1500,
							RootDiskType: "gp2",
						},
						SubnetName:       fmt.Sprintf("cluster-api-subnet-kcp-%s", uid),
						LoadBalancerName: fmt.Sprintf("osc-k8s-machine-%s", uid),
						VmType:           "tinav6.c4r8p2",
						SecurityGroupNames: []infrastructurev1beta1.OscSecurityGroupElement{
							{
								Name: fmt.Sprintf("cluster-api-securitygroups-kcp-%s", uid),
							},
						},
						PrivateIps: []infrastructurev1beta1.OscPrivateIpElement{
							{
								Name:      fmt.Sprintf("cluster-api-privateip-kcp-%s", uid),
								PrivateIp: "10.0.4.10",
							},
						},
					},
				},
			}
			createCheckDeleteOscClusterMachine(ctx, infraClusterSpec, infraMachineSpec)
		})
	})
})
