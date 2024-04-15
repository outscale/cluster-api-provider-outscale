# Deploy cluster with rke2 botstrapper
This folder contains a example of rke2 config and image-builder to build new omi with rk2 for cluster-api

## Requirements
 * [Kubernetes](https://www.packer.io/downloads)
 * [cluster-api](https://github.com/kubernetes-sigs/cluster-api)
 * [cluster-api-provider-rke2](https://github.com/rancher-sandbox/cluster-api-provider-rke2)

## Change the characteristic cluster and node

You can change the characteristic of the cluster (SecurityGroup, Subnet, RouteTables, .. ) and node (Image, vmType, ...) in osc-rke2-clusterctl.yaml

## Create your rk2 cluster
```bash
kubectl apply -f osc-rke2-clusterctl.yml
```

## Check Machine are provisionned

Please check machines are provisionned

```bash
kubectl get machine -A
NAMESPACE   NAME                              CLUSTER       NODENAME    PROVIDERID                               PHASE      AGE   VERSION
default     rke2-osc-control-plane-jr7b2      rke2-osc                  aws:///cloudgouv-eu-west-1a/i-e2dd281a   Provisionned    52m   v1.27.9
default     rke2-osc-md-0-55qrn-jg4l7         rke2-osc                  aws:///cloudgouv-eu-west-1a/i-e31f76d5   Provisionned    52m   v1.27.9+rke2r1
```

Please check with cockpit2 that you cluster vm has cloud-init finish without any trouble


## Get kubeconfig

Please retrieve kubeconfig
```bash
clusterctl get kubeconfig <clustername> > <clustername>.config
export KUBECONFIG=${PWD}/<clustername>.config
```

## Set ccm credentials

Replace only MY_AWS_ACCESS_KEY_ID with your outscale access key, MY_AWS_SECRET_ACCESS_KEY with your outscale secret key and MY_AWS_DEFAULT_REGION with your outscale region.

```bash
kubectl apply -f ccm-rke2.yaml
```

### Return to the management cluster
Change KUBECONFIG environment variable to use your management cluster

```bash
kubectl get machine -A
NAMESPACE   NAME                              CLUSTER       NODENAME    PROVIDERID                               PHASE      AGE   VERSION
default     rke2-osc-control-plane-jr7b2      rke2-osc      ip-10-0-4-8.cloudgouv-eu-west-1.compute.internal     aws:///cloudgouv-eu-west-1a/i-e2dd281a   Running    52m   v1.27.9
default     rke2-osc-md-0-55qrn-jg4l7         rke2-osc      ip-10-0-3-77.cloudgouv-eu-west-1.compute.internal    aws:///cloudgouv-eu-west-1a/i-e31f76d5   Running    52m   v1.27.9+rke2r1
```

