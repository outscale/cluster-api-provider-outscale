
# Getting started with clusterctl

## Prerequisites

- [kubectl][kubectl] installed on your computer,
- an Outscale account with [an Access Key and a Secret Key][Outscale Access Key and Secret Key],
- a management Kubernetes cluster, for example an [OKS][OKS] cluster.

## Installing clusterctl

On a Linux/amd64 platform:
```bash
curl -L https://github.com/kubernetes-sigs/cluster-api/releases/download/v1.10.6/clusterctl-linux-amd64 -o clusterctl
sudo install -o root -g root -m 0755 clusterctl /usr/local/bin/clusterctl
```

> For other platforms, please refer to the [Cluster API Quickstart][cluster-api].

Check which version is installed:
```bash
clusterctl version

clusterctl version: &version.Info{Major:"1", Minor:"10", GitVersion:"v1.10.6", GitCommit:"c37851897736ecd5bfcf29791b9ccfef1220e503", GitTreeState:"clean", BuildDate:"2025-09-02T16:04:06Z", GoVersion:"go1.23.12", Compiler:"gc", Platform:"linux/amd64"}
```

## Configuring clusterctl

You can enable [ClusterClass][ClusterClass] with:
```bash
export CLUSTER_TOPOLOGY=true
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

Pick an image. You can start with an [Outscale Open Source image][OutscaleOpenSourceImage].

Generate a cluster configuration.

```bash
export OSC_IOPS=<osc-iops>
export OSC_VOLUME_SIZE=<osc-volume-size>
export OSC_VOLUME_TYPE=<osc-volume-type>
export OSC_KEYPAIR_NAME=<osc-keypairname>
export OSC_SUBREGION_NAME=<osc-subregion>
export OSC_VM_TYPE=<osc-vm-type>
export OSC_IMAGE_NAME=<osc-image-name>

clusterctl generate cluster <cluster-name> --kubernetes-version <kubernetes-version>   --control-plane-machine-count=<control-plane-machine-count> --worker-machine-count=<worker-machine-count> > getstarted.yaml
```

> The Kubernetes version must be the same version as the image you picked. If `OSC_IMAGE_NAME` is `ubuntu-2204-kubernetes-v1.31.12-2025-08-27`, `<kubernetes-version>` is `v1.31.12`.

On Linux, you may compute the Kubernetes version based on the image name using:

```bash
KUBERNETES_VERSION=`echo $OSC_IMAGE_NAME|sed 's/.*\(v1[.0-9]*\).*/\1/'`
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

To install CCM, you can use a [helm][helm] chart or a ClusterResourceSet.

* [cloud-provider-outscale][cloud-provider-outscale]

## Checking status

You can check the status of your new workload cluster:
```bash
kubectl get cluster-api --namespace <namespace of cluster>
```

Or, using clusterctl:
```bash
clusterctl describe cluster <name of cluster> --namespace <namespace of cluster>
```

## Connecting to a workload cluster

You can get a kubeconfig to connect to your workload cluster with
[clusterctl get kubeconfig][kubeconfig].

## Deleting a cluster

To delete your cluster:
```bash
kubectl delete -f getstarted.yaml
```

## Upgrading the Outscale provider

To upgrade the Outscale provider to a new version:
```bash
clusterctl upgrade apply -i outscale:v1.1.1
```

## Deleting Cluster API

To remove all cluster-api components from your cluster:
```bash
clusterctl delete --all
kubectl delete namespace cluster-api-provider-outscale-system
```

<!-- References -->
[cillium]: https://docs.cilium.io/en/stable/installation/k8s-toc/
[calico]: https://projectcalico.docs.tigera.io/getting-started/kubernetes/helm
[kubeconfig]: https://cluster-api.sigs.k8s.io/clusterctl/commands/get-kubeconfig.html
[cloud-provider-outscale]: https://github.com/outscale-dev/cloud-provider-osc/blob/OSC-MIGRATION/deploy/README.md
[kubectl]: https://kubernetes.io/docs/tasks/tools/install-kubectl/
[helm]: https://helm.sh/docs/intro/install/
[ClusterResourceSet]: https://cluster-api.sigs.k8s.io/tasks/experimental-features/cluster-resource-set.html
[ClusterClass]: https://cluster-api.sigs.k8s.io/tasks/experimental-features/cluster-class/
[OKS]: https://docs.outscale.com/en/userguide/About-OKS.html
[Outscale Access Key and Secret Key]: https://wiki.outscale.net/display/EN/Creating+an+Access+Key
[cluster-api]: https://cluster-api.sigs.k8s.io/user/quick-start.html
[OutscaleOpenSourceImage]: https://github.com/outscale/kube-image-workflows/releases