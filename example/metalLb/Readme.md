# Kubernetes Cluster Deployment with MetalLB

## Prerequisites
Before starting, ensure the following are in place:

1. Infrastructure
Management Cluster:
- A management cluster is required to use Cluster API for provisioning and managing the workload cluster. This cluster can be deployed locally (e.g., using kind or minikube) or [rke.](https://github.com/outscale/osc-k8s-rke-cluster)

2. Tools
- kubectl
- Cluster API
- Cluster-api outscale provider 

3. MetalLB
Ensure Layer 2 (L2) connectivity between your cluster nodes to support MetalLB's ARP-based IP advertising.
IP range configured for MetalLB should not overlap with any existing subnet or DHCP ranges in your environment.

## This documentation provides a step-by-step guide to:

- Deploy a Kubernetes cluster with Cluster API.
- Install and configure MetalLB.
- Test the setup with a LoadBalancer service.
- Verify the assigned IP from the MetalLB IP pool.

### MetalLB Integration in the Control Plane

The MetalLB installation is fully automated through the postKubeadmCommands in the control plane configuration.
The IP pool (10.0.1.240-10.0.1.250) and Layer 2 advertisement configuration are pre-created as a file and applied during the node initialization.
After deployment, verify the metallb-system namespace and pods, then test by deploying a LoadBalancer service.

```bash
kubectl apply -f example/metalLb/service.yaml
```

```bash
kubectl get pods -n metallb-system
```

```bash
NAME                          READY   STATUS    RESTARTS   AGE
controller-7bcd9b5f47-l9r96   1/1     Running   0          104s
speaker-5dvs2                 1/1     Running   0          104s
speaker-nnwdg                 1/1     Running   0          104s
speaker-rvkmp                 1/1     Running   0          104s
```

### Deploy a Test Service
```bash
kubectl apply -f service.yaml
```

```bash
kubectl get svc nginx-service
NAME            TYPE           CLUSTER-IP      EXTERNAL-IP   PORT(S)        AGE
nginx-service   LoadBalancer   10.43.209.200   10.0.1.244    80:30509/TCP   6m45s
```

#### Test the Service
Access the service using the external IP:
```bash
curl http://10.0.1.244
```
