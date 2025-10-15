#!/bin/bash
set -ex
RUNNER_NAME=$1
OKS_AKSK=$2
OSC_CLUSTER_NAME=$4

OKS_ACCESS_KEY=`echo $OKS_AKSK|cut -d% -f 1`
OKS_SECRET_KEY=`echo $OKS_AKSK|cut -d% -f 2`
OKS_REGION=`echo $OKS_AKSK|cut -d% -f 3`

cluster_name=`echo $RUNNER_NAME|tr '[:upper:]' '[:lower:]'|sed -r 's/-[a-z0-9]+$//'|cut -c1-40|sed -r 's/[^a-z0-9-]+/-/g'`

/.venv/bin/oks-cli profile add --profile-name "default" --access-key $OKS_ACCESS_KEY --secret-key $OKS_SECRET_KEY --region $OKS_REGION
/.venv/bin/oks-cli project login --project-name github-runner
kubeconfig=`/.venv/bin/oks-cli cluster kubeconfig --cluster-name $cluster_name --print-path`
export KUBECONFIG=$kubeconfig

# do not wait if stuck, frieza will purge everything
# as freeza cannot delete netpeerings, be careful
kubectl delete ns $OSC_CLUSTER_NAME --timeout 10m || /bin/true