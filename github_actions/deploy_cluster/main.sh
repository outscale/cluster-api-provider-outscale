#!/bin/bash
set -ex
DEPLOY_CLUSTER=$1
OSC_AKSK=$2
export OSC_CLUSTER_NAME=$3
OSC_IMAGE_NAME_ACCOUNT_ID=$4
OSC_VM_TYPE_COUNT=$5
export OSC_VM_TYPE=`echo $OSC_VM_TYPE_COUNT%|cut -d% -f 1`
WORKER_COUNT=`echo $OSC_VM_TYPE_COUNT%|cut -d% -f 2`
if [ -z "$WORKER_COUNT" ]; then
  WORKER_COUNT=1
fi
CCM=$6
CERT_MANAGER=$7
KUSTOMIZE_PATH=$8

export OSC_ACCESS_KEY=`echo $OSC_AKSK|cut -d% -f 1`
export OSC_SECRET_KEY=`echo $OSC_AKSK|cut -d% -f 2`
export OSC_REGION=`echo $OSC_AKSK|cut -d% -f 3`

export OSC_IMAGE_NAME=`echo $OSC_IMAGE_NAME_ACCOUNT_ID|cut -d% -f 1`
export OSC_IMAGE_ACCOUNT_ID=`echo $OSC_IMAGE_NAME_ACCOUNT_ID|cut -d% -f 2`

# F***g DNS
echo "options use-vc single-request attempts:5" >> /etc/resolv.conf

# Kind
export KIND_CLUSTER=cluster-api-e2e
make cleanup-test-e2e || /bin/true
make setup-test-e2e
touch management.kubeconfig
chmod 600 management.kubeconfig
kind get kubeconfig --name=$KIND_CLUSTER > management.kubeconfig
echo "MANAGEMENT_KUBECONFIG=management.kubeconfig" >> $GITHUB_OUTPUT

if [ "$DEPLOY_CLUSTER" == "true"]; then
    # clusterctl init
    kubectl create secret generic osc-secret --from-literal=access_key=$OSC_ACCESS_KEY --from-literal=secret_key=$OSC_SECRET_KEY -n kube-system
    clusterctl init --infrastructure outscale

    export OSC_SUBREGION_NAME=${OSC_REGION}a
    export OSC_KEYPAIR_NAME=cluster-api
    export OSC_IOPS=1000
    export OSC_VOLUME_SIZE=30
    export OSC_VOLUME_TYPE=gp2
    export KUBERNETES_VERSION=`echo $OSC_IMAGE_NAME|sed 's/.*\(v1[.0-9]*\).*/\1/'`
    ip=`curl -s -S --retry 5 --retry-all-errors https://api.ipify.org`
    export OSC_ALLOW_FROM="$ip/32"
    mip=`kubectl run --image curlimages/curl:8.14.1 getip --restart=Never -ti --rm -q -- curl -s -S --retry 5 --retry-all-errors https://api.ipify.org`
    export OSC_ALLOW_FROM_CAPI="$mip/32"
    export OSC_NAT_IP_POOL=caposc
    export OSC_IMAGE_OPENSOURCE=true
    clusterctl generate cluster $OSC_CLUSTER_NAME --kubernetes-version $KUBERNETES_VERSION --control-plane-machine-count=1 --worker-machine-count=$WORKER_COUNT -n $OSC_CLUSTER_NAME --from https://github.com/outscale/cluster-api-provider-outscale/blob/main/templates/cluster-template-secure.yaml > clusterapi.yaml
    # clusterctl generate cluster $OSC_CLUSTER_NAME --kubernetes-version $KUBERNETES_VERSION --control-plane-machine-count=1 --worker-machine-count=$WORKER_COUNT -n $OSC_CLUSTER_NAME -i outscale --flavor secure-opensource > clusterapi.yaml

    kubectl delete ns $OSC_CLUSTER_NAME --ignore-not-found --force
    kubectl create ns $OSC_CLUSTER_NAME

    if [ -n "$KUSTOMIZE_PATH" ]; then
    mv clusterapi.yaml $GITHUB_WORKSPACE/$KUSTOMIZE_PATH
    kubectl apply -k $GITHUB_WORKSPACE/$KUSTOMIZE_PATH
    else
    kubectl apply -f clusterapi.yaml
    fi

    # This can be removed once v1.0.0-alpha.3 is released and it is deployed on the CI.
    # kubectl patch osccluster $OSC_CLUSTER_NAME -n $OSC_CLUSTER_NAME --type='merge' -p \
    #   '{"spec":{"network":{"additionalSecurityRules":[{"roles":["worker"], "rules":[{"flow":"Inbound", "ipRange":"10.0.4.0/24", "fromPortRange":-1, "toPortRange": -1, "ipProtocol": "4"}]}]}}}'

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
    # switch to workload cluster
    export KUBECONFIG=$kubeconfig

    # install calico
    kubectl \
    apply -f https://raw.githubusercontent.com/projectcalico/calico/v3.26.1/manifests/calico.yaml

    echo "waiting for nodes"
    for i in {1..50}; do
    cnt=`kubectl get nodes --no-headers | tee /dev/stderr | wc -l`
    if [[ $cnt -ge 2 ]]; then
        break
    fi
    sleep 30
    done

    if [ "$CCM" = "true" ]; then
    echo "installing CCM v1"
    KUBE_MAJORMINOR=`echo $KUBERNETES_VERSION|cut -d . -f 1,2`
    CCM_VERSION=$KUBE_MAJORMINOR.4-alpha.1
    kubectl create secret generic osc-secret --from-literal=access_key=$OSC_ACCESS_KEY --from-literal=secret_key=$OSC_SECRET_KEY -n kube-system
    helm install --wait k8s-osc-ccm oci://registry-1.docker.io/outscalehelm/osc-cloud-controller-manager --set oscSecretName=osc-secret \
        --set image.tag=$CCM_VERSION
    echo "waiting for nodes"
    kubectl wait --for=condition=Ready nodes --all --timeout=900s
    kubectl get nodes -o wide
    echo "restarting pods"
    kubectl get po -A -o custom-columns=NAMESPACE:.metadata.namespace,NAME:.metadata.name,HOSTNETWORK:.spec.hostNetwork --no-headers=true | grep '<none>' | awk '{print "-n "$1" "$2}' | xargs -L 1 -r kubectl delete pod
    echo "waiting for pods"
    kubectl wait --for=condition=Ready po --all -A --timeout=900s
    kubectl get po -A
    fi

    if [ "$CERT_MANAGER" = "true" ]; then
    echo "installing cert-manager"
    helm repo add jetstack https://charts.jetstack.io --force-update
    helm install \
        cert-manager jetstack/cert-manager \
        --namespace cert-manager \
        --create-namespace \
        --version v1.17.2 \
        --set crds.enabled=true \
        --set prometheus.enabled=false \
        --set webhook.timeoutSeconds=4 \
        --set "webhook.nodeSelector.node-role\.kubernetes\.io/control-plane=" \
        --set "webhook.tolerations[0].key=node-role.kubernetes.io/control-plane" \
        --set "webhook.tolerations[0].operator=Exists" \
        --set "webhook.tolerations[0].effect=NoSchedule"
    fi
fi
