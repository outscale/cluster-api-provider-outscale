#!/bin/bash
set -ex
RUNNER_NAME=$1
OKS_AKSK=$2
OSC_AKSK=$3
export OSC_CLUSTER_NAME=$4
export OSC_IMAGE_NAME=$5
export OSC_VM_TYPE=$6

OKS_ACCESS_KEY=`echo $OKS_AKSK|cut -d% -f 1`
OKS_SECRET_KEY=`echo $OKS_AKSK|cut -d% -f 2`
OKS_REGION=`echo $OKS_AKSK|cut -d% -f 3`

export OSC_ACCESS_KEY=`echo $OSC_AKSK|cut -d% -f 1`
export OSC_SECRET_KEY=`echo $OSC_AKSK|cut -d% -f 2`
export OSC_REGION=`echo $OSC_AKSK|cut -d% -f 3`

cluster_name=`echo $RUNNER_NAME|tr '[:upper:]' '[:lower:]'|sed -r 's/-[a-z0-9]+$//'|cut -c1-40|sed -r 's/[^a-z0-9-]+/-/g'`
# OKS
/.venv/bin/oks-cli profile add --profile-name "default" --access-key $OKS_ACCESS_KEY --secret-key $OKS_SECRET_KEY --region $OKS_REGION
/.venv/bin/oks-cli project login --project-name github-runner
/.venv/bin/oks-cli cluster list
kubeconfig=`/.venv/bin/oks-cli cluster kubeconfig --cluster-name $cluster_name --print-path`
export KUBECONFIG=$kubeconfig

# clusterctl
export OSC_SUBREGION_NAME=${OSC_REGION}a
export OSC_KEYPAIR_NAME=cluster-api
export OSC_IOPS=1000
export OSC_VOLUME_SIZE=30
export OSC_VOLUME_TYPE=gp2
export KUBERNETES_VERSION=`echo $OSC_IMAGE_NAME|sed 's/.*\(v1[.0-9]*\).*/\1/'`
clusterctl generate cluster $OSC_CLUSTER_NAME --kubernetes-version $KUBERNETES_VERSION --control-plane-machine-count=1 --worker-machine-count=1 -n $OSC_CLUSTER_NAME -i outscale > clusterapi.yaml

kubectl delete ns $OSC_CLUSTER_NAME --ignore-not-found --grace-period 600
kubectl create ns $OSC_CLUSTER_NAME
kubectl apply -f clusterapi.yaml

for i in {1..10}; do
sleep 30
kubectl get --namespace $OSC_CLUSTER_NAME machine.cluster.x-k8s.io
kubectl get --namespace $OSC_CLUSTER_NAME kubeadmcontrolplane.controlplane.cluster.x-k8s.io
done
echo "waiting for control plane"
kubectl wait --namespace $OSC_CLUSTER_NAME --timeout=900s \
  --for=jsonpath='{.status.initialized}' kubeadmcontrolplane.controlplane.cluster.x-k8s.io/$OSC_CLUSTER_NAME-control-plane

# fetch kubeconfig
kubeconfig="$GITHUB_WORKSPACE/$OSC_CLUSTER_NAME.kubeconfig"
touch $kubeconfig
chmod 600 $kubeconfig
clusterctl get kubeconfig $OSC_CLUSTER_NAME --namespace $OSC_CLUSTER_NAME > $kubeconfig
echo "KUBECONFIG=$OSC_CLUSTER_NAME.kubeconfig" >> $GITHUB_OUTPUT

# install calico
kubectl --kubeconfig=$kubeconfig \
  apply -f https://raw.githubusercontent.com/projectcalico/calico/v3.26.1/manifests/calico.yaml

# wait for all pods to be ready
kubectl --kubeconfig=$kubeconfig wait --timeout=900s --for=condition=Ready po --all -A 

kubectl --kubeconfig=$kubeconfig get po -A
