#!/bin/bash
set -ex
export RUNNER_NAME=$1
export OKS_ACCESS_KEY=$2
export OKS_SECRET_KEY=$3
export OKS_REGION=$4

export OSC_ACCESS_KEY=$5
export OSC_SECRET_KEY=$6
export OSC_REGION=$7

export OSC_CLUSTER_NAME=$8
export OSC_IMAGE_NAME=$9

cluster_name=`echo $RUNNER_NAME|tr '[:upper:]' '[:lower:]'|sed -r 's/-[a-z0-9]+$//'|cut -c1-40|sed -r 's/[^a-z0-9-]+/-/g'`
# OKS
/.venv/bin/oks-cli profile add --profile-name "default" --access-key $OKS_ACCESS_KEY --secret-key $OKS_SECRET_KEY --region $OKS_REGION
/.venv/bin/oks-cli project login --project-name github-runner
kubeconfig=`/.venv/bin/oks-cli cluster kubeconfig --cluster-name $cluster_name --print-path`
export KUBECONFIG=$kubeconfig

# clusterctl
export OSC_VM_TYPE=tinav6.c2r8p2
export OSC_SUBREGION_NAME=${OSC_REGION}a
export OSC_KEYPAIR_NAME=cluster-api
export OSC_IOPS=1000
export OSC_VOLUME_SIZE=30
export OSC_VOLUME_TYPE=gp2
export KUBERNETES_VERSION=`echo $OSC_IMAGE_NAME|sed 's/.*\(v1[.0-9]*\).*/\1/'`
clusterctl generate cluster $OSC_CLUSTER_NAME --kubernetes-version $KUBERNETES_VERSION --control-plane-machine-count=1 --worker-machine-count=1 -n $OSC_CLUSTER_NAME -i outscale > clusterapi.yaml

kubectl create ns $OSC_CLUSTER_NAME
kubectl apply -f clusterapi.yaml

kubectl wait --namespace $OSC_CLUSTER_NAME --timeout=600s \
  --for=jsonpath='{.status.Initialized}' kubeadmcontrolplane.controlplane.cluster.x-k8s.io/$OSC_CLUSTER_NAME-control-plane

kubeconfig=`clusterctl get kubeconfig $OSC_CLUSTER_NAME --namespace $OSC_CLUSTER_NAME > $OSC_CLUSTER_NAME.kubeconfig`
echo "KUBECONFIG=$kubeconfig" >> $GITHUB_OUTPUT
