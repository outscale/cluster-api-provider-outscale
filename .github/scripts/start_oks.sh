#!/bin/bash

wait() {
    rsrc=$1
    cmd=$2
    max_retry=20
    retry=0
    while [ ${retry} -lt ${max_retry} ]; do
        ready=`$cmd | grep ready`
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
pip install https://outscale:$OKS_CLI_DOWNLOAD_PASSWORD@docs.eu-west-2.oks.outscale.com/oks_cli.zip
ret=$?
if [ $ret -ne 0 ]; then
    exit $ret
fi

oks-cli profile add --profile-name "default" --access-key $OSC_ACCESS_KEY --secret-key $OSC_SECRET_KEY --region eu-west-2
ret=$?
if [ $ret -ne 0 ]; then
    exit $ret
fi
oks-cli project get --project-name github-runner
if [ $? -ne 0 ]; then
    oks-cli project create --project-name github-runner --cidr '10.10.0.0/16'
    ret=$?
    if [ $ret -ne 0 ]; then
        exit $ret
    fi
    sleep 10
fi
wait "project" "oks-cli project get --project-name github-runner"

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
wait "cluster" "oks-cli cluster get --cluster-name $cluster_name"

np=`oks-cli cluster nodepool --cluster-name $cluster_name list|grep github-runner`
if [ -z "$np" ]; then
    oks-cli cluster nodepool --cluster-name $cluster_name create --nodepool-name github-runner --count 1 --type $WORKER_VMTYPE
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

nginx_namespace=ng-`echo $cluster_name|cksum|awk '{print $1}'`
# nginx ingress
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm repo update
helm upgrade --install nginx-ingress ingress-nginx/ingress-nginx \
    --namespace $nginx_namespace --create-namespace
ret=$?
if [ $ret -ne 0 ]; then
    exit $ret
fi

kubectl wait --namespace $nginx_namespace --timeout=120s \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller

# cert manager
helm upgrade --install cert-manager oci://registry-1.docker.io/bitnamicharts/cert-manager \
    --namespace certmanager --create-namespace \
    --set installCRDs=true --version 1.3.19
ret=$?
if [ $ret -ne 0 ]; then
    exit $ret
fi

kubectl wait --namespace certmanager --timeout=120s \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=webhook

kubectl apply -f - << EOF
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: letsencrypt-issuer
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: opensource@outscale.com
    privateKeySecretRef:
      name: letsencrypt-issuer-key
    solvers:
    - http01:
        ingress:
          ingressClassName: nginx
EOF
ret=$?
if [ $ret -ne 0 ]; then
    exit $ret
fi

# Harbor
helm repo add harbor https://helm.goharbor.io

kubectl apply -f - << EOF
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  annotations:
    storageclass.kubernetes.io/is-default-class: "true"
  name: bsu-default
parameters:
  csi.storage.k8s.io/fstype: ext4
  type: standard
provisioner: bsu.csi.outscale.com
reclaimPolicy: Delete
volumeBindingMode: Immediate
EOF

harbor_host=`kubectl get ingress harbor-ingress -o jsonpath="{.status.loadBalancer.ingress[0].hostname}"`
if [ -z "$harbor_host" ]; then
    helm upgrade --install harbor harbor/harbor --set "expose.type=ingress" --set "expose.ingress.className=nginx" \
        --set "expose.tls.enabled=false" \
        --set "harborAdminPassword=$HARBOR_ADMIN_PASSWORD"
    ret=$?
    if [ $ret -ne 0 ]; then
        exit $ret
    fi
    kubectl wait ingress harbor-ingress --for=jsonpath='{.status.loadBalancer.ingress[0].hostname}' --timeout=240s
    harbor_host=`kubectl get ingress harbor-ingress -o jsonpath="{.status.loadBalancer.ingress[0].hostname}"`
fi

if [ -z "$harbor_host" ]; then
    exit 2
fi

helm upgrade --install harbor harbor/harbor --set "expose.type=ingress" --set "expose.ingress.className=nginx" \
    --set "expose.ingress.annotations.cert-manager\.io/issuer=letsencrypt-issuer" \
    --set "expose.ingress.annotations.ingress\.kubernetes\.io/ssl-redirect=\"false\"" \
    --set "expose.ingress.annotations.nginx\.ingress\.kubernetes\.io/ssl-redirect=\"false\"" \
    --set "expose.ingress.hosts.core=$harbor_host" \
    --set "expose.tls.enabled=true" --set "expose.tls.auto.commonName=$harbor_host" \
    --set "expose.tls.certSource=secret" --set "expose.tls.secret.secretName=letsencrypt-harbor-cert" \
    --set "externalURL=https://$harbor_host" \
    --set "harborAdminPassword=$HARBOR_ADMIN_PASSWORD"
ret=$?
if [ $ret -ne 0 ]; then
    exit $ret
fi

kubectl wait --timeout=120s --for=condition=ready certificate letsencrypt-harbor-cert
ret=$?
if [ $ret -ne 0 ]; then
    # sometimes, cert-manager is unable to create the cert, recreating it
    kubectl delete certificate letsencrypt-harbor-cert
    sleep 5
    kubectl wait --timeout=120s --for=condition=ready certificate letsencrypt-harbor-cert
    ret=$?
    if [ $ret -ne 0 ]; then
        exit $ret
    fi
fi

kubectl wait --timeout=240s --for=condition=ready pod \
    --selector=app.kubernetes.io/component=core

curl -fs -u admin:$HARBOR_ADMIN_PASSWORD https://$harbor_host/api/v2.0/projects/outscale || \
   curl -f -XPOST -u admin:$HARBOR_ADMIN_PASSWORD --header "Content-Type: application/json" --data '{"project_name":"outscale", "public":true}' https://$harbor_host/api/v2.0/projects
ret=$?
if [ $ret -ne 0 ]; then
    exit $ret
fi

echo "cluster_name=$cluster_name" >> $GITHUB_OUTPUT
echo "kubeconfig=$kubeconfig" >> $GITHUB_OUTPUT
echo "harbor_host=$harbor_host" >> $GITHUB_OUTPUT