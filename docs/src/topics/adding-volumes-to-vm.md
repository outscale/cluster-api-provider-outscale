# Adding volumes to nodes

By default, nodes use a single root volume (/dev/sda1). Additional volumes can be added to VM.

> Volumes are created unformatted. You will need to format the newly created volumes during cloud-init.

In your oscmachinetemplate definition, add the list of volumes required:

```
[...]
  node:
    vm: [...]
    volumes:
    - name: data
      device: /dev/sdb
      iops: 500
      size: 50
      volumeType: io1
    - name: logs
      device: /dev/sdc
      iops: 500
      size: 10
      volumeType: io1
[...]
```