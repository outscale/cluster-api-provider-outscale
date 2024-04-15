# Troubleshooting Guide

Common issues that you might see.

### Missing credentials

Please set your credentials
```bash
 kubectl logs -f  cluster-api-provider-outscale-controller-manager-9f8dd7d8bqncnb  -n cluster-api-provider-outscale-system  
```

You will get:
```
1.6630978127864842e+09	ERROR	controller.oscmachine	Reconciler error	{"reconciler group": "infrastructure.cluster.x-k8s.io", "reconciler kind": "OscMachine", "name": "capo-quickstart-md-0-tpjgs", "namespace": "default", "error": "environment variable OSC_ACCESS_KEY is required failed to create Osc Client"}
```


### Override Limit 

Please check you have enough core, ram, instance quota.

Otherwise you will get:
```bash
controller.osccluster	Reconciler error	{"reconciler group": "infrastructure.cluster.x-k8s.io", "reconciler kind": "OscCluster", "name": "cluster-api-test", "namespace": "default", "error": "400 Bad Request Can not create net for Osccluster default/cluster-api-test"}
sigs.k8s.io/controller-runtime/pkg/internal/controller.(*Controller).processNextWorkItem
	/go/pkg/mod/sigs.k8s.io/controller-runtime@v0.11.2/pkg/internal/controller/controller.go:266
sigs.k8s.io/controller-runtime/pkg/internal/controller.(*Controller).Start.func2.2
	/go/pkg/mod/sigs.k8s.io/controller-runtime@v0.11.2/pkg/internal/controller/controller.go:227
```


### Node are not ready

Node are not ready because they need cni to be ready.


### Not running Node
If your vm is never in running phase and but still in provisonned phase, please look at the cloud init log of your vm.


### Trouble with e2etest path
You should clean a previous installation before launching e2etest:
```
make uninstall-clusterapi
kubectl get crd -A | grep x-k8s.
```

You should delete all cluster-api's CRD.
If deletion is blocked, you can use Finalizers to delete crds by patching them one by one
```
kubectl patch --namespace=my-namespace my-object my-object--name --patch='{"metadata":{"finalizers":null}}' --type=merge

```
### Clean Stack
If your vm did not reach running state, you can use:
```bash
ClusterToClean=my-cluster-name make testclean
```
to clean you stack

If there is some cluster-api k8s object (such as oscMachineTemplate) still remaning after running the cleaning script, please do:
```
kubectl delete oscmachinetemplate --all -A
kubectl patch --namespace=my-namespace oscmachinetemplate my-object-name --patch='{"metadata":{"finalizers":null}}' --type=merge
```


### Be able to use cillium as a cni

With **ubuntu 22.04**, cillium is not compatibled with hotplug.

Log console of the vm:
```
Jan 16 11:17:35 ip-10-0-0-36 kubelet[644]: I0116 11:17:35.535025     644 log.go:198] http: superfluous response.WriteHeader call from k8s.io/kubernetes/vendor/github.com/emicklei/go-restful.(*Response).WriteHeader (response.go:220)
Jan 16 11:17:41 ip-10-0-0-36 cloud-init-hotplugd[2589]: 2024-01-16 11:17:41,273 - hotplug_hook.py[ERROR]: Received fatal exception handling hotplug!
Jan 16 11:17:41 ip-10-0-0-36 cloud-init-hotplugd[2589]: Traceback (most recent call last):
Jan 16 11:17:41 ip-10-0-0-36 cloud-init-hotplugd[2589]:   File "/usr/lib/python3/dist-packages/cloudinit/cmd/devel/hotplug_hook.py", line 277, in handle_args
Jan 16 11:17:41 ip-10-0-0-36 cloud-init-hotplugd[2589]:     handle_hotplug(
Jan 16 11:17:41 ip-10-0-0-36 cloud-init-hotplugd[2589]:   File "/usr/lib/python3/dist-packages/cloudinit/cmd/devel/hotplug_hook.py", line 235, in handle_hotplug
Jan 16 11:17:41 ip-10-0-0-36 cloud-init-hotplugd[2589]:     raise last_exception
Jan 16 11:17:41 ip-10-0-0-36 cloud-init-hotplugd[2589]:   File "/usr/lib/python3/dist-packages/cloudinit/cmd/devel/hotplug_hook.py", line 224, in handle_hotplug
Jan 16 11:17:41 ip-10-0-0-36 cloud-init-hotplugd[2589]:     event_handler.detect_hotplugged_device()
Jan 16 11:17:41 ip-10-0-0-36 cloud-init-hotplugd[2589]:   File "/usr/lib/python3/dist-packages/cloudinit/cmd/devel/hotplug_hook.py", line 104, in detect_hotplugged_device
Jan 16 11:17:41 ip-10-0-0-36 cloud-init-hotplugd[2589]:     raise RuntimeError(
Jan 16 11:17:41 ip-10-0-0-36 cloud-init-hotplugd[2589]: RuntimeError: Failed to detect aa:17:dc:e4:6a:8d in updated metadata
Jan 16 11:17:41 ip-10-0-0-36 cloud-init[2589]: [CLOUDINIT]2024-01-16 11:17:41,273 - hotplug_hook.py[ERROR]: Received fatal exception handling hotplug!
                                               Traceback (most recent call last):
                                                 File "/usr/lib/python3/dist-packages/cloudinit/cmd/devel/hotplug_hook.py", line 277, in handle_args
                                                   handle_hotplug(
                                                 File "/usr/lib/python3/dist-packages/cloudinit/cmd/devel/hotplug_hook.py", line 235, in handle_hotplug
                                                   raise last_exception
                                                 File "/usr/lib/python3/dist-packages/cloudinit/cmd/devel/hotplug_hook.py", line 224, in handle_hotplug
                                                   event_handler.detect_hotplugged_device()
                                                 File "/usr/lib/python3/dist-packages/cloudinit/cmd/devel/hotplug_hook.py", line 104, in detect_hotplugged_device
                                                   raise RuntimeError(
                                               RuntimeError: Failed to detect aa:17:dc:e4:6a:8d in updated metadata
Jan 16 11:17:41 ip-10-0-0-36 cloud-init[2589]: [CLOUDINIT]2024-01-16 11:17:41,274 - handlers.py[DEBUG]: finish: hotplug-hook: FAIL: Handle reconfiguration on hotplug events.
Jan 16 11:17:41 ip-10-0-0-36 cloud-init[2589]: [CLOUDINIT]2024-01-16 11:17:41,274 - util.py[DEBUG]: Reading from /proc/uptime (quiet=False)
Jan 16 11:17:41 ip-10-0-0-36 cloud-init[2589]: [CLOUDINIT]2024-01-16 11:17:41,274 - util.py[DEBUG]: Read 14 bytes from /proc/uptime
Jan 16 11:17:41 ip-10-0-0-36 cloud-init[2589]: [CLOUDINIT]2024-01-16 11:17:41,274 - util.py[DEBUG]: cloud-init mode 'hotplug-hook' took 76.643 seconds (76.64)
Jan 16 11:17:41 ip-10-0-0-36 cloud-init-hotplugd[2589]: Traceback (most recent call last):
Jan 16 11:17:41 ip-10-0-0-36 cloud-init-hotplugd[2589]:   File "/usr/bin/cloud-init", line 11, in <module>
Jan 16 11:17:41 ip-10-0-0-36 cloud-init-hotplugd[2589]:     load_entry_point('cloud-init==22.2', 'console_scripts', 'cloud-init')()
Jan 16 11:17:41 ip-10-0-0-36 cloud-init-hotplugd[2589]:   File "/usr/lib/python3/dist-packages/cloudinit/cmd/main.py", line 1088, in main
Jan 16 11:17:41 ip-10-0-0-36 cloud-init-hotplugd[2589]:     retval = util.log_time(
Jan 16 11:17:41 ip-10-0-0-36 cloud-init-hotplugd[2589]:   File "/usr/lib/python3/dist-packages/cloudinit/util.py", line 2621, in log_time
Jan 16 11:17:41 ip-10-0-0-36 cloud-init-hotplugd[2589]:     ret = func(*args, **kwargs)
Jan 16 11:17:41 ip-10-0-0-36 cloud-init-hotplugd[2589]:   File "/usr/lib/python3/dist-packages/cloudinit/cmd/devel/hotplug_hook.py", line 277, in handle_args
Jan 16 11:17:41 ip-10-0-0-36 cloud-init-hotplugd[2589]:     handle_hotplug(
Jan 16 11:17:41 ip-10-0-0-36 cloud-init-hotplugd[2589]:   File "/usr/lib/python3/dist-packages/cloudinit/cmd/devel/hotplug_hook.py", line 235, in handle_hotplug
Jan 16 11:17:41 ip-10-0-0-36 cloud-init-hotplugd[2589]:     raise last_exception
Jan 16 11:17:41 ip-10-0-0-36 cloud-init-hotplugd[2589]:   File "/usr/lib/python3/dist-packages/cloudinit/cmd/devel/hotplug_hook.py", line 224, in handle_hotplug
Jan 16 11:17:41 ip-10-0-0-36 cloud-init-hotplugd[2589]:     event_handler.detect_hotplugged_device()
Jan 16 11:17:41 ip-10-0-0-36 cloud-init-hotplugd[2589]:   File "/usr/lib/python3/dist-packages/cloudinit/cmd/devel/hotplug_hook.py", line 104, in detect_hotplugged_device
Jan 16 11:17:41 ip-10-0-0-36 cloud-init-hotplugd[2589]:     raise RuntimeError(
Jan 16 11:17:41 ip-10-0-0-36 cloud-init-hotplugd[2589]: RuntimeError: Failed to detect aa:17:dc:e4:6a:8d in updated metadata
Jan 16 11:17:41 ip-10-0-0-36 systemd[1]: cloud-init-hotplugd.service: Main process exited, code=exited, status=1/FAILURE
Jan 16 11:17:41 ip-10-0-0-36 systemd[1]: cloud-init-hotplugd.service: Failed with result 'exit-code'.
Jan 16 11:17:41 ip-10-0-0-36 audit[1]: SERVICE_STOP pid=1 uid=0 auid=4294967295 ses=4294967295 subj=unconfined msg='unit=cloud-init-hotplugd comm="systemd" exe="/usr/lib/systemd/systemd" hostname=? addr=? terminal=? res=failed'
Jan 16 11:17:41 ip-10-0-0-36 audit[1]: SERVICE_START pid=1 uid=0 auid=4294967295 ses=4294967295 subj=unconfined msg='unit=cloud-init-hotplugd comm="systemd" exe="/usr/lib/systemd/systemd" hostname=? addr=? terminal=? res=success'
Jan 16 11:17:41 ip-10-0-0-36 systemd[1]: Started cloud-init hotplug hook daemon.
Jan 16 11:17:41 ip-10-0-0-36 cloud-init-hotplugd[2606]: args=--subsystem=net handle --devpath=/devices/virtual/net/cilium_vxlan --udevaction=add
Jan 16 11:17:41 ip-10-0-0-36 bash[2606]: [CLOUDINIT]2024-01-16 11:17:41,659 - hotplug_hook.py[DEBUG]: hotplug-hook called with the following arguments: {hotplug_action: handle, subsystem: net, udevaction: add, devpath: /devices/virtual/net/cilium_vxlan}
Jan 16 11:17:41 ip-10-0-0-36 bash[2606]: [CLOUDINIT]2024-01-16 11:17:41,659 - handlers.py[DEBUG]: start: hotplug-hook: Handle reconfiguration on hotplug events. 
```

In order to be able to deploy cillium with k8s nodes based on **ubuntu 22.04**, you have to deactivate hotplug.

Remove hotplug from array when
/etc/cloud/cloud.cfg.d/06_hotplug.cfg:

```yaml
updates:
  network:
    when: ["boot"] 
```

You can reboot your node to run again cloudinit.

Or you can run again cloudinit without reboot.

#### Clean existing config
```bash
sudo cloud-init clean --logs
```

#### Detect local data source
```bash
sudo cloud-init init --local
```


#### Detect any datasources whic require network up
```bash
sudo cloud-init init
```

#### Run all cloud_config_modules
```bash
sudo cloud-init modules --mode=config
```

#### Run all cloud_final_modules
```bash
sudo cloud-init modules --mode=final
```