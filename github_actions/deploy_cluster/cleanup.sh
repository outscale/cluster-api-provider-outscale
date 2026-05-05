#!/bin/bash
set -ex
DEPLOY_CLUSTER=$1
OSC_CLUSTER_NAME=$3

if
# do not wait if stuck, frieza will purge everything
KUBECONFIG=management.kubeconfig kubectl delete ns $OSC_CLUSTER_NAME --timeout 3m || /bin/true

make cleanup-test-e2e
