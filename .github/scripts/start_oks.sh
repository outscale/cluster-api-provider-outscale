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

oks-cli profile add --profile-name "default" --access-key $OSC_ACCESS_KEY --secret-key $OSC_SECRET_KEY --region eu-west-2
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

# NGinx ingress
kubectl apply -f https://raw.githubusercontent.com/nginxinc/kubernetes-ingress/v3.6.2/deploy/crds.yaml

helm upgrade --install nginx-ingress oci://ghcr.io/nginxinc/charts/nginx-ingress --version 1.3.2 \
--set controller.kind="daemonset" \
--set controller.nodeSelector.node-pool=github-runner \
--set controller.service.type="LoadBalancer" \
--set controller.ingressClass.name="internal-nginx" \
--set controller.ingressClass.create="true" \
--set controller.service.annotations."service\.beta\.kubernetes\.io/osc-load-balancer-name"=harbor-ingress  \
--set controller.service.annotations."service\.beta\.kubernetes\.io/osc-load-balancer-target-node-pools"=github-runner \
--set "controller.config.use-proxy-protocol=true" \
--set "controller.config.use-forwarded-headers=true" \
--set "controller.config.compute-full-forwarded-for=true" \
--set "controller.config.enable-real-ip=true"
ret=$?
if [ $ret -ne 0 ]; then
    exit $ret
fi

# cert manager
helm upgrade --install cert-manager oci://registry-1.docker.io/bitnamicharts/cert-manager --set installCRDs=true

kubectl apply -f - << EOF
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  selfSigned: {}
---
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: letsencrypt-ca
  namespace: default
spec:
  ca:
    secretName: letsencrypt-ca
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: letsencrypt-ca
  namespace: default
spec:
  isCA: true
  commonName: osm-system
  secretName: letsencrypt-ca
  issuerRef:
    name: letsencrypt-prod
    kind: ClusterIssuer
    group: cert-manager.io
EOF

harbor_host=`kubectl get ingress harbor-ingress -o jsonpath="{.status.loadBalancer.ingress[0].hostname}"`
if [ -z "$harbor_host" ]; then
    # Harbor
    helm repo add harbor https://helm.goharbor.io
    helm upgrade --install harbor harbor/harbor --set "expose.type=ingress" --set "expose.ingress.className=internal-nginx" \
        --set "expose.ingress.annotations.service\.beta\.kubernetes\.io/osc-load-balancer-source-ranges=$PUBLIC_IP/32" \
        --set "expose.tls.enabled=false" \
        --set "persistence.enabled=false"
    ret=$?
    if [ $ret -ne 0 ]; then
        exit $ret
    fi
    wait "harbor" "oks-cli cluster kubectl get ingress harbor-ingress --cluster-name $cluster_name" ".lbu.outscale.com"
    harbor_host=`kubectl get ingress harbor-ingress -o jsonpath="{.status.loadBalancer.ingress[0].hostname}"`

    helm upgrade --install harbor harbor/harbor --set "expose.type=ingress" --set "expose.ingress.className=internal-nginx" \
        --set "expose.ingress.annotations.service\.beta\.kubernetes\.io/osc-load-balancer-source-ranges=$PUBLIC_IP/32" \
        --set "expose.ingress.annotations.cert-manager\.io/issuer=letsencrypt-prod" \
        --set "expose.ingress.hosts.core=$harbor_host" \
        --set "expose.tls.enabled=true" --set "expose.tls.auto.commonName=$harbor_host" \
        --set "expose.tls.certSource=secret" --set "expose.tls.secret.secretName=letsencrypt-ca" \
        --set "persistence.enabled=false"
    ret=$?
    if [ $ret -ne 0 ]; then
        exit $ret
    fi
fi

echo "cluster_name=$cluster_name" >> $GITHUB_OUTPUT
echo "kubeconfig=$kubeconfig" >> $GITHUB_OUTPUT
echo "harbor_host=$harbor_host" >> $GITHUB_OUTPUT