# Securing cluster access

## Restricting access to a cluster

By default, the bastion and Kubernetes API load-balancer security groups accept trafic from the internet (source range: `0.0.0.0/0`). You might want to restrict access to known IP ranges by setting `allowFromIPRanges`.

```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: OscCluster
metadata:
  name: test-cluster-api
  namespace: test-cluster-api
spec:
  network:
    [...]
    allowFromIPRanges:
    - 192.0.2.15/32
    - 203.0.113.0/24
```

> Note: the NAT used by the nodes for their outbound trafic are dynamically added to the allowed sources in the loadbalancer security group.

## Restricting outbound traffic

By default, outbound trafic from bastion and nodes is not restricted.

You can restrict outbound traffic by setting `allowToIPRanges`.

```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: OscCluster
metadata:
  name: test-cluster-api
  namespace: test-cluster-api
spec:
  network:
    [...]
    allowToIPRanges:
    - 192.0.2.15/32
    - 203.0.113.0/24
```

This replaces the defaul outbound rule in the node security group and the bastion security group (if configured) by:

```yaml
  - flow: Outbound
    ipProtocol: "-1"
    fromPortRange: -1
    toPortRange: -1
    ipRanges:
    - {allowedToRanges}
```

If you need finer control, you can disable the Outbound rule by setting an empty entry and add your own outbound rules to `additionalSecurityRules`:

```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1beta1
kind: OscCluster
metadata:
  name: test-cluster-api
  namespace: test-cluster-api
spec:
  network:
    [...]
    allowToIPRanges:
    - 
    additionalSecurityRules:
    - roles:
      - controlplane
      - worker
      rules:
      - flow: Outbound
        ipProtocol: "udp"
        fromPortRange: 53
        toPortRange: 53
        ipRanges:
        - 203.0.113.0/24
```

> Note: Internal traffic within the cluster VPC is always allowed by an additional outbound rule.

```yaml
  - flow: Outbound
    ipProtocol: "-1"
    fromPortRange: -1
    toPortRange: -1
    ipRanges:
    - {cluster VPC range}
```

## clusterctl setup

A clusterctl template may be used to build a cluster with IP restriction on the Kubernetes API:

```bash
export OSC_IOPS=<osc-iops>
export OSC_VOLUME_SIZE=<osc-volume-size>
export OSC_VOLUME_TYPE=<osc-volume-type>
export OSC_KEYPAIR_NAME=<osc-keypairname>
export OSC_REGION=<osc-region>
export OSC_VM_TYPE=<osc-vm-type>
export OSC_IMAGE_NAME=<osc-image-name>
export OSC_ALLOW_FROM=<IP range allowed to access the Kubernetes API or 0.0.0.0/0 for no restriction>

clusterctl generate cluster <cluster-name> --kubernetes-version <kubernetes-version> --control-plane-machine-count=<control-plane-machine-count> --worker-machine-count=<worker-machine-count> --flavor=secure > getstarted.yaml
```

or, when deploying a multi-az cluster:

```bash
export OSC_IOPS=<osc-iops>
export OSC_VOLUME_SIZE=<osc-volume-size>
export OSC_VOLUME_TYPE=<osc-volume-type>
export OSC_KEYPAIR_NAME=<osc-keypairname>
export OSC_REGION=<osc-region>
export OSC_VM_TYPE=<osc-vm-type>
export OSC_IMAGE_NAME=<osc-image-name>
export OSC_ALLOW_FROM=<IP range allowed to access the Kubernetes API or 0.0.0.0/0 for no restriction>

clusterctl generate cluster <cluster-name> --kubernetes-version <kubernetes-version> --control-plane-machine-count=<control-plane-machine-count> --worker-machine-count=<worker-machine-count> --flavor=multiaz-secure > getstarted.yaml
```
