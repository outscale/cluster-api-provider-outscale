#!/bin/bash
set -ex
export OKS_CLUSTER_NAME=$1
export OKS_ACCESS_KEY=$2
export OKS_SECRET_KEY=$3
export OKS_REGION=$4

export OSC_CLUSTER_NAME=$8

/.venv/bin/oks-cli profile add --profile-name "default" --access-key $OKS_ACCESS_KEY --secret-key $OKS_SECRET_KEY --region $OKS_REGION
/.venv/bin/oks-cli project login --project-name github-runner
kubeconfig=`/.venv/bin/oks-cli cluster kubeconfig --cluster-name $OKS_CLUSTER_NAME --print-path`
export KUBECONFIG=$kubeconfig

kubectl delete ns $OSC_CLUSTER_NAME