
# Prerequisite 
- Install [kubectl][kubectl]

- Outscale account with ak/sk [Outscale Access Key and Secret Key][Outscale Access Key and Secret Key]
- A Kubernetes cluster:
    - You can use a Vm with [kubeadm][kubeadm] or [Minikube][Minikube]. 
    - You can use a container with [kind][kind]. 
    - You can use a rke cluster with [osc-rke][osc-rke].
- Look at cluster-api note ([cluster-api][cluster-api])

# Deploy Cluster Api

## Clone

Please clone the project:
```
git clone https://github.com/outscale-dev/cluster-api-provider-outscale
```

If you use your own cluster for production (with backup, disaster recovery, ...) and expose it:
```
 export KUBECONFIG=<...>
```

Or you can use kind (only for local dev):
Create kind:
```bash
   kind create cluster
```
Check cluster is ready:
```bash
   kubectl cluster-info
```
# Install clusterctl
:warning: In order to install tools (clusterctl, ...) with makefile, you need to have installed golang to download binaries [golang][golang]
You can install clusterctl for linux with:
```bash
  make install-clusterctl
```
Or you can install clusterctl with the clusterctl section ([cluster-api][cluster-api]).
And check version which is already installed:
```bash
./bin/clusterctl version
clusterctl version: &version.Info{Major:"1", Minor:"2", GitVersion:"v1.2.4", GitCommit:"8b5cd363e11b023c2b67a1937a2af680ead9e35c", GitTreeState:"clean", BuildDate:"2022-10-17T13:37:39Z", GoVersion:"go1.18.7", Compiler:"gc", Platform:"linux/amd64"}
```

## Initialize clusterctl
You can enable [clusterresourceset][clusterresourceset] with 
```bash
export EXP_CLUSTER_RESOURCE_SET=true
```

Please create  $HOME/.cluster-api/clusterctl.yaml: 
```
providers:
- name: outscale
  type: InfrastructureProvider
  url: https://github.com/outscale/cluster-api-provider-outscale/releases/latest/infrastructure-components.yaml
```

You can initialize clusterctl with credential with:
```
export OSC_ACCESS_KEY=<your-access-key>
export OSC_SECRET_KEY=<your-secret-access-key>
export OSC_REGION=<your-region>
make credential
./bin/clusterctl init --infrastructure outscale
```

# Create our cluster

## Launch your stack with clusterctl

You can create a keypair before if you want.
You can access nodes shell (with [openlens][openlens], [lens][lens], ...)
You have to set:
```
export OSC_IOPS=<osc-iops>
export OSC_VOLUME_SIZE=<osc-volume-size>
export OSC_VOLUME_TYPE=<osc-volume-type>
export OSC_KEYPAIR_NAME=<osc-keypairname>
export OSC_SUBREGION_NAME=<osc-subregion>
export OSC_VM_TYPE=<osc-vm-type>
export OSC_IMAGE_NAME=<osc-image-name>
```
Then you will generate:
```
./bin/clusterctl generate cluster <cluster-name>   --kubernetes-version <kubernetes-version>   --control-plane-machine-count=<control-plane-machine-count>   --worker-machine-count=<worker-machine-count> > getstarted.yaml
```
**WARNING**: Kubernetes version must match the kubernetes version which is included in image name in [omi][omi]

You can then change to get what you want which is based on doc.

Then apply:
```
kubectl apply -f getstarted.yaml
```

## Add a public ip after bastion is created

You can add a public ip if you set publicIpNameAfterBastion = true after you have already create a cluster with a bastion.

### Get Kubeconfig
You can then get the status:
```
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
## Get kubeconfig

In order to get kubeconfig please use:
[kubeconfig][kubeconfig] 

## Node Ready

In order to have node ready, you must have a CNI and CCM.

You can use [clusterresourceset][clusterresourceset] with label **clustername** + **crs-cni** and label **clustername** + **crs-ccm** where clustername is the name of your cluster.

To install cni you can use helm charts or clusteresourceset.

To install helm,please follow [helm][helm]

A list of cni:
* To install [calico][calico]
* To install [cillium][cillium]
* To install [canal][canal]

To install ccm, you can use helm charts or clusteresourceset.

* [cloud-provider-outscale][cloud-provider-outscale]

# Delete Cluster api

## Delete cluster

To delete our cluster:
```
kubectl delete -f getstarted.yaml
```

To delete cluster-api:
```
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
[clusterresourceset]: https://cluster-api.sigs.k8s.io/tasks/experimental-features/cluster-resource-set.html
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
