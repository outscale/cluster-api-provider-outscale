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
If you need finer control, you can disable the Outbound rule by setting an empty entry and add your own outbound rules to `additionalsecurityRules`:

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
    additionalsecurityRules:
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
