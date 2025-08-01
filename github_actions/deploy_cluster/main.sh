#!/bin/bash
set -ex
RUNNER_NAME=$1
OKS_AKSK=$2
OSC_AKSK=$3
export OSC_CLUSTER_NAME=$4
OSC_IMAGE_NAME_ACCOUNT_ID=$5
export OSC_VM_TYPE=$6
CCM=$7
CERT_MANAGER=$8
KUSTOMIZE_PATH=$9

OKS_ACCESS_KEY=`echo $OKS_AKSK|cut -d% -f 1`
OKS_SECRET_KEY=`echo $OKS_AKSK|cut -d% -f 2`
OKS_REGION=`echo $OKS_AKSK|cut -d% -f 3`

export OSC_ACCESS_KEY=`echo $OSC_AKSK|cut -d% -f 1`
export OSC_SECRET_KEY=`echo $OSC_AKSK|cut -d% -f 2`
export OSC_REGION=`echo $OSC_AKSK|cut -d% -f 3`

export OSC_IMAGE_NAME=`echo $OSC_IMAGE_NAME_ACCOUNT_ID|cut -d% -f 1`
export OSC_IMAGE_ACCOUNT_ID=`echo $OSC_IMAGE_NAME_ACCOUNT_ID|cut -d% -f 2`

if [ -z "$OSC_IMAGE_ACCOUNT_ID" ]; then
  export OSC_IMAGE_ACCOUNT_ID=`curl -X POST https://api.$OSC_REGION.outscale.com/api/v1/ReadAccounts \
    --user $OSC_ACCESS_KEY:$OSC_SECRET_KEY --aws-sigv4 'osc' \
    --header 'Content-Type: application/json' \
    --data '{}'|jq -r '.Accounts[0].AccountId'`
fi

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
ip=`curl https://api.ipify.org`
export OSC_ALLOW_FROM="$ip/32"
mip=`kubectl run --image curlimages/curl:8.14.1 getip --restart=Never -ti --rm -q -- curl https://api.ipify.org`
export OSC_ALLOW_FROM_CAPI="$mip/32"
clusterctl generate cluster $OSC_CLUSTER_NAME --kubernetes-version $KUBERNETES_VERSION --control-plane-machine-count=1 --worker-machine-count=1 -n $OSC_CLUSTER_NAME -i outscale --flavor secure > clusterapi.yaml

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
  cnt=`kubectl get nodes | tee /dev/stderr | wc -l`
  if [[ $cnt -ge 3 ]]; then
    break
  fi
  sleep 30
done

if [ "$CCM" = "true" ]; then
  echo "installing CCM"
  kubectl create secret generic osc-secret --from-literal=key_id=$OSC_ACCESS_KEY --from-literal=access_key=$OSC_SECRET_KEY --from-literal=aws_default_region=$OSC_REGION \
    --from-literal=aws_availability_zones=MY_AWS_AVAILABILITY_ZONES --from-literal=osc_account_id=MY_OSC_ACCOUNT_ID --from-literal=osc_account_iam=MY_OSC_ACCOUNT_IAM --from-literal=osc_user_id=MY_OSC_USER_ID --from-literal=osc_arn=MY_OSC_ARN \
    -n kube-system
  helm install --wait k8s-osc-ccm oci://registry-1.docker.io/outscalehelm/osc-cloud-controller-manager --set oscSecretName=osc-secret
  echo "waiting for nodes"
  kubectl wait --for=condition=Ready nodes --all --timeout=900s
  kubectl get nodes -o wide
  echo "restarting pods"
  kubectl get pods --all-namespaces -o custom-columns=NAMESPACE:.metadata.namespace,NAME:.metadata.name,HOSTNETWORK:.spec.hostNetwork --no-headers=true | grep '<none>' | awk '{print "-n "$1" "$2}' | xargs -L 1 -r kubectl delete pod
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