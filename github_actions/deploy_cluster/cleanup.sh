#!/bin/bash
set -ex
export RUNNER_NAME=$1
export OKS_ACCESS_KEY=$2
export OKS_SECRET_KEY=$3
export OKS_REGION=$4

export OSC_CLUSTER_NAME=$8

cluster_name=`echo $RUNNER_NAME|tr '[:upper:]' '[:lower:]'|sed -r 's/-[a-z0-9]+$//'|cut -c1-40|sed -r 's/[^a-z0-9-]+/-/g'`

/.venv/bin/oks-cli profile add --profile-name "default" --access-key $OKS_ACCESS_KEY --secret-key $OKS_SECRET_KEY --region $OKS_REGION
/.venv/bin/oks-cli project login --project-name github-runner
kubeconfig=`/.venv/bin/oks-cli cluster kubeconfig --cluster-name $cluster_name --print-path`
export KUBECONFIG=$kubeconfig

kubectl delete ns $OSC_CLUSTER_NAME