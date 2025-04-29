
# Getting started with clusterctl

## Prerequisites

- Install [kubectl][kubectl]
- Outscale account with [an Access Key and a Secret Key][Outscale Access Key and Secret Key]
- A management Kubernetes cluster:
    - You can use a Vm with [kubeadm][kubeadm] or [Minikube][Minikube]. 
    - You can use a container with [kind][kind]. 
    - You can use a rke cluster with [osc-rke][osc-rke].

## Installing clusterctl

On a Linux/amd64 platform:
```bash
curl -L https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.9.6/clusterctl-linux-amd64 -o clusterctl
sudo install -o root -g root -m 0755 clusterctl /usr/local/bin/clusterctl
```

> For other platforms, please refer to the [Cluster API Quickstart][cluster-api].

Check which version is installed:
```bash
clusterctl version

clusterctl version: &version.Info{Major:"1", Minor:"8", GitVersion:"v1.8.1", GitCommit:"02769254e95db17afbd6ec4036aacbd294d9424c", GitTreeState:"clean", BuildDate:"2024-08-14T05:53:36Z", GoVersion:"go1.22.5", Compiler:"gc", Platform:"linux/amd64"}
```

## Configuring clusterctl

You can enable [ClusterResourceSet][ClusterResourceSet] with 
```bash
export EXP_CLUSTER_RESOURCE_SET=true
```

You can enable [ClusterClass][ClusterClass] with
```bash
export CLUSTER_TOPOLOGY=true
```

Please create `$HOME/.cluster-api/clusterctl.yaml`:
```yaml
providers:
- name: outscale
  type: InfrastructureProvider
  url: https://github.com/outscale/cluster-api-provider-outscale/releases/latest/infrastructure-components.yaml
```

Install credentials for your workload cluster Outscale account:
```bash
export OSC_ACCESS_KEY=<your-access-key>
export OSC_SECRET_KEY=<your-secret-access-key>
export OSC_REGION=<your-region>

kubectl create namespace cluster-api-provider-outscale-system
kubectl create secret generic cluster-api-provider-outscale --from-literal=access_key=$OSC_ACCESS_KEY --from-literal=secret_key=$OSC_SECRET_KEY --from-literal=region=$OSC_REGION  -n cluster-api-provider-outscale-system
```

Install Cluster API controllers:
```bash
clusterctl init --infrastructure outscale
```

## Generating a default workload cluster configuration

Create a keypair.

```bash
export OSC_IOPS=<osc-iops>
export OSC_VOLUME_SIZE=<osc-volume-size>
export OSC_VOLUME_TYPE=<osc-volume-type>
export OSC_KEYPAIR_NAME=<osc-keypairname>
export OSC_SUBREGION_NAME=<osc-subregion>
export OSC_VM_TYPE=<osc-vm-type>
export OSC_IMAGE_NAME=<osc-image-name>

clusterctl generate cluster <cluster-name> --kubernetes-version <kubernetes-version>   --control-plane-machine-count=<control-plane-machine-count>   --worker-machine-count=<worker-machine-count> > getstarted.yaml
```

You can edit the generated YAML file to customize the configuration to your needs.

Then apply:
```
kubectl apply -f getstarted.yaml
```

## Installing CNI & CCM

In order for nodes to be ready, a CNI and CCM must be installed.

You can use [ClusterResourceSet][ClusterResourceSet] with label **clustername** + **crs-cni** and label **clustername** + **crs-ccm** where clustername is the name of your cluster.

To install CNI you can use a [helm][helm] chart or a ClusterResourceSet.

* [calico][calico]
* [cillium][cillium]
* [canal][canal]

To install CCM, you can use a [helm][helm] chart or a ClusterResourceSet.

* [cloud-provider-outscale][cloud-provider-outscale]

## Checking status

You can check the status of your new workload cluster:
```bash
root@cidev-admin v1beta1]# kubectl get cluster-api  -A
NAMESPACE   NAME                                                                       CLUSTER       AGE
default     kubeadmconfig.bootstrap.cluster.x-k8s.io/cluster-api-control-plane-lzj65   cluster-api   95m
default     kubeadmconfig.bootstrap.cluster.x-k8s.io/cluster-api-md-0-zgx4w            cluster-api   95m

NAMESPACE   NAME                                                                AGE
default     kubeadmconfigtemplate.bootstrap.cluster.x-k8s.io/cluster-api-md-0   95m

NAMESPACE   NAME                                                      CLUSTER       REPLICAS   READY   AVAILABLE   AGE   VERSION
default     machineset.cluster.x-k8s.io/cluster-api-md-0-7568fb659d   cluster-api   1                              95m   v1.22.11

NAMESPACE   NAME                                                  CLUSTER       REPLICAS   READY   UPDATED   UNAVAILABLE   PHASE       AGE   VERSION
default     machinedeployment.cluster.x-k8s.io/cluster-api-md-0   cluster-api   1                  1         1             ScalingUp   95m   v1.22.11

NAMESPACE   NAME                                                         CLUSTER       NODENAME                                  PROVIDERID         PHASE      AGE   VERSION
default     machine.cluster.x-k8s.io/cluster-api-control-plane-4q2s8     cluster-api   ip-10-0-0-45.eu-west-2.compute.internal   aws:///eu-west-2a/i-3b629324    Running    95m   v1.22.11
default     machine.cluster.x-k8s.io/cluster-api-md-0-7568fb659d-hnkfw   cluster-api   ip-10-0-0-144.eu-west-2.compute.internal   aws:///eu-west-2a/i-add154be   Running    95m   v1.22.11

NAMESPACE   NAME                                   PHASE         AGE   VERSION
default     cluster.cluster.x-k8s.io/cluster-api   Provisioned   95m   

NAMESPACE                           NAME                                                         AGE   TYPE                   PROVIDER      VERSION
capi-kubeadm-bootstrap-system       provider.clusterctl.cluster.x-k8s.io/bootstrap-kubeadm       46h   BootstrapProvider      kubeadm       v1.2.1
capi-kubeadm-control-plane-system   provider.clusterctl.cluster.x-k8s.io/control-plane-kubeadm   46h   ControlPlaneProvider   kubeadm       v1.2.1
capi-system                         provider.clusterctl.cluster.x-k8s.io/cluster-api             46h   CoreProvider           cluster-api   v1.2.1

NAMESPACE   NAME                                                                          CLUSTER       INITIALIZED   API SERVER AVAILABLE   REPLICAS   READY   UPDATED   UNAVAILABLE   AGE   VERSION
default     kubeadmcontrolplane.controlplane.cluster.x-k8s.io/cluster-api-control-plane   cluster-api                                        1                  1         1             95m   v1.22.11

NAMESPACE   NAME                                                                           AGE
default     oscmachinetemplate.infrastructure.cluster.x-k8s.io/cluster-api-control-plane   95m
default     oscmachinetemplate.infrastructure.cluster.x-k8s.io/cluster-api-md-0            95m

```
## Connecting to a workload cluster

You can get a kubeconfig to connect to your workload cluster with
[clusterctl get kubeconfig][kubeconfig].

## Deleting a cluster

To delete your cluster:
```bash
kubectl delete -f getstarted.yaml
```

## Deleting Cluster API

To delete cluster-api:
```bash
clusterctl delete --all
```

<!-- References -->
[canal]: https://projectcalico.docs.tigera.io/getting-started/kubernetes/flannel/flannel
[cillium]: https://docs.cilium.io/en/stable/installation/k8s-toc/
[calico]: https://projectcalico.docs.tigera.io/getting-started/kubernetes/helm
[kubeconfig]: https://cluster-api.sigs.k8s.io/clusterctl/commands/get-kubeconfig.html
[cloud-provider-outscale]: https://github.com/outscale-dev/cloud-provider-osc/blob/OSC-MIGRATION/deploy/README.md
[kubectl]: https://kubernetes.io/docs/tasks/tools/install-kubectl/
[helm]: https://helm.sh/docs/intro/install/
[ClusterResourceSet]: https://cluster-api.sigs.k8s.io/tasks/experimental-features/cluster-resource-set.html
[ClusterClass]: https://cluster-api.sigs.k8s.io/tasks/experimental-features/cluster-class/
[kind]: https://github.com/kubernetes-sigs/kind#installation-and-usage
[kubeadm]: https://kubernetes.io/fr/docs/setup/production-environment/tools/kubeadm/install-kubeadm/
[Outscale Access Key and Secret Key]: https://wiki.outscale.net/display/EN/Creating+an+Access+Key
[osc-rke]: https://github.com/outscale-dev/osc-k8s-rke-cluster
[Minikube]: https://kubernetes.io/docs/tasks/tools/install-minikube/
[openlens]: https://github.com/MuhammedKalkan/OpenLens
[lens]: https://github.com/lensapp/lens
[cluster-api]: https://cluster-api.sigs.k8s.io/user/quick-start.html
[cluster-api-config]: https://cluster-api.sigs.k8s.io/clusterctl/configuration.html 
[omi]: ./omi.md
[golang]: https://medium.com/@sherlock297/install-and-set-the-environment-variable-path-for-go-in-kali-linux-446d0f16a338
