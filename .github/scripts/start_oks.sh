#!/bin/bash

wait() {
    rsrc=$1
    cmd=$2
    waitfor=$3
    max_retry=20
    retry=0
    while [ ${retry} -lt ${max_retry} ]; do
        ready=`$cmd | grep $waitfor`
        if [ -n "$ready" ]; then
            break
        else
        echo "waiting for $rsrc"
            (( retry = retry + 1 ))
            sleep 30
        fi
    done
}

set -x
python3 -m venv .venv
source .venv/bin/activate
pip install https://$OKS_CLI_DOWNLOAD_USER:$OKS_CLI_DOWNLOAD_PASSWORD@oks-docs.prd2-eu-west-2.oks.outscale.com/oks_cli.zip

oks-cli profile add --profile-name "default" --access-key $OSC_ACCESS_KEY --secret-key $OSC_SECRET_KEY --region eu-west-2 --endpoint "https://api.prd2-eu-west-2.oks.outscale.com/api/v2/"
if [ $? -ne 0 ]; then
    exit $?
fi
oks-cli project get --project-name github-runner
if [ $? -ne 0 ]; then
    oks-cli project create --project-name github-runner --cidr '10.10.0.0/16'
    if [ $? -ne 0 ]; then
        exit $?
    fi
    sleep 10
fi
wait "project" "oks-cli project get --project-name github-runner" "ready"

oks-cli project login --project-name github-runner
cluster_name=`echo $CLUSTER_NAME|tr '[:upper:]' '[:lower:]'|sed -r 's/-[a-z0-9]+$//'|cut -c1-40|sed -r 's/[^a-z0-9-]+/-/g'`
echo "creating cluster with admin whitelist for $PUBLIC_IP"
oks-cli cluster get --cluster-name $cluster_name
if [ $? -ne 0 ]; then
    oks-cli cluster create --project-name github-runner --cluster-name $cluster_name --admin $PUBLIC_IP/32 --control-plane cp.mono.master
    ret=$?
    if [ $ret -ne 0 ]; then
        exit $ret
    fi
    sleep 10
fi
wait "cluster" "oks-cli cluster get --cluster-name $cluster_name" "ready"

np=`oks-cli cluster nodepool --cluster-name $cluster_name list|grep github-runner`
if [ -z "$np" ]; then
    oks-cli cluster nodepool --cluster-name $cluster_name create --nodepool-name github-runner --count 2 --type $WORKER_VMTYPE
    ret=$?
    if [ $ret -ne 0 ]; then
        exit $ret
    fi
fi

kubeconfig=`oks-cli cluster kubeconfig --cluster-name $cluster_name --print-path`
ret=$?
if [ $ret -ne 0 ]; then
    exit $ret
fi
export KUBECONFIG=$kubeconfig

helm repo add harbor https://helm.goharbor.io
helm upgrade --install harbor harbor/harbor --set "expose.type=ingress" \
    --set "expose.ingress.annotations.service\.beta\.kubernetes\.io/osc-load-balancer-source-ranges=$PUBLIC_IP/32" \
    --set "expose.ingress.annotations.service\.beta\.kubernetes\.io/osc-load-balancer-name=harbor-caposc-ci" \
    --set "expose.tls.enabled=true,expose.tls.auto.commonName=*.eu-west-2.lbu.outscale.com" \
    --set "persistence.enabled=false"
ret=$?
if [ $ret -ne 0 ]; then
    exit $ret
fi
wait "harbor" "oks-cli cluster kubectl get svc harbor --cluster-name $cluster_name" ".lbu.outscale.com"
harbor_host=`kubectl get svc harbor -o jsonpath="{.status.ingress.ingress[0].hostname}"`

# helm upgrade harbor harbor/harbor --set "expose.type=ingress" \
#     --set "expose.ingress.annotations.service\.beta\.kubernetes\.io/osc-load-balancer-source-ranges=$PUBLIC_IP/32" \
#     --set "expose.ingress.annotations.service\.beta\.kubernetes\.io/osc-load-balancer-name=harbor-caposc-ci" \
#     --set "expose.tls.enabled=true,expose.tls.auto.commonName=$harbor_host" \
#     --set "persistence.enabled=false"
# ret=$?
# if [ $ret -ne 0 ]; then
#     exit $ret
# fi

echo "cluster_name=$cluster_name" >> $GITHUB_OUTPUT
echo "kubeconfig=$kubeconfig" >> $GITHUB_OUTPUT
echo "harbor_host=$harbor_host" >> $GITHUB_OUTPUT