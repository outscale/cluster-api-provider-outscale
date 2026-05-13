#!/bin/bash
set -ex
RUNNER_NAME=$1
OSC_CLUSTER_NAME=$4

cluster_name=`echo $RUNNER_NAME|tr '[:upper:]' '[:lower:]'|sed -r 's/-[a-z0-9]+$//'|cut -c1-40|sed -r 's/[^a-z0-9-]+/-/g'`

# do not wait if stuck, frieza will purge everything
octl kube kubectl --cluster $cluster_name --profile oks -- delete ns $OSC_CLUSTER_NAME --timeout 3m || /bin/true
