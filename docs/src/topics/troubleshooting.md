# Troubleshooting Guide

Here are some information useful to troubleshoot CAPOSC.

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

### Node stays in provisioned phase

You can check the console of the VM to check cloud-init logs:
```shell
octl iaas vm readconsole <vm id>
```

### Node are not ready

Be aware that CCM and a CNI is required for a node to be ready. Check that both are correctly deployed and running.
