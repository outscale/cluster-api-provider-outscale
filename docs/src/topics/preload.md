# Preloading images

Preloading images lets you add container images to the local containerd cache at boot.

It enables:
* faster startup times (no need to download images from the internet),
* building clusters without internet access.

## Creating a snapshot

You need a seed cluster with access to the internet and all apps required on the target cluster installed.

The seed cluster must have CSI installed, with snapshotting enabled.

You can use the [following example](k8s-image-preloader-example) based on [k8s-image-preloader](8s-image-preloader) to create the snapshot.

You may want to adapt the spec if you need more than 2GB of storage for your images or if you want to change the `deletionPolicy` of the `VolumeSnapshotClass`.

[k8s-image-preloader](8s-image-preloader):
* mounts a `PersistentVolumeClaim` (PVC),
* searches for images in the local containerd cache,
* searches `Pods` and `CronJobs`,
* pulls images from the remote registry (even if already present in the cache, as containerd may prune some layers from cached images),
* exports the images to the PVC,
* creates a `VolumeSnapshot` from the PVC.

You can get the handle (IaaS snapshot ID) by querying the `VolumeSnapshotContent` status:
```bash
kubectl get vsc -o custom-columns=NAME:.metadata.name,SNAPSHOT:.spec.volumeSnapshotRef.name,NAMESPACE:.spec.volumeSnapshotRef.namespace,HANDLE:.status.snapshotHandle
```

## Importing images

You will need to mount a volume based on the previously generated snapshot on target nodes:

In `OscMachineTemplate` resources:
```yaml
        vm:
          [...]
        volumes:
        - device: /dev/xvdb
          fromSnapshot: snap-xxx
```

Then import all images during cloud-init:

In `KubeadmConfigTemplate`/`KubeadmControlPlane` resources:
```yaml
  spec:
    joinConfiguration:
      [...]
    mounts:
      - - xvdb
        - /preload
        - ext4
        - auto,exec,ro
    preKubeadmCommands:
      - /preload/restore.sh
```

## Notes

k8s-image-preloader only works with containerd, and uses the `ctr` tool installed on the node, but a similar process can be built for CRI-O using [skopeo](skopeo).

<!-- References -->
[k8s-image-preloader-example]: https://raw.githubusercontent.com/outscale/k8s-image-preloader/refs/heads/main/examples/example.yaml
[skopeo]: https://github.com/containers/skopeo
[k8s-image-preloader]: https://github.com/outscale/k8s-image-preloader
