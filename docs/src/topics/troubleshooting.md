# Troubleshooting Guide

This guide provides useful information for troubleshooting CAPOSC.

## Viewing logs

To view the Outscale provider (CAPOSC) logs:
```shell
kubectl -n cluster-api-provider-outscale-system deploy/cluster-api-provider-outscale-controller-manager -f
```

To view the cluster-api (CAPI) logs:
```shell
kubectl logs -n capi-system deploy/capi-controller-manager -f
```

## Node issues

### Node stays in the provisioned phase

You can check the console of the VM to inspect cloud-init logs using [octl](https://github.com/outscale/octl):
```shell
octl iaas vm readconsole <vm id>
```

### Node are not ready

Note that both a CCM and a CNI are required for a node to become ready. Make sure they are correctly deployed and running.
