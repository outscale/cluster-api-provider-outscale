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

## 