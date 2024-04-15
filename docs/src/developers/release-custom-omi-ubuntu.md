
# Kubernetes Custom Omi Generation

## OMI

Select omi you want to use. (we only test and verify with ubuntu omi)

## Clone

Please clone projet image-builder in $HOME

```bash
git clone https://github.com/kubernetes-sigs/image-builder.git
```

## New Omi


Please create **$HOME/image-builder/images/capi/packer/outscale/ubuntu-2204.json** and replace **UBUNTU_OMI** with the name for you omi and remove **$HOME/image-builder/images/capi/packer/outscale/ubuntu-2004.json**


```json
{
  "build_name": "ubuntu-2204",
  "distribution": "ubuntu",
  "distribution_release": "ubuntu",
  "distribution_version": "2204",
  "image_name": "UBUNTU_OMI"
}
```

## Makefile

Replace in Makefile (**$HOME/image-builder/images/capi/Makefile**) osc-ubuntu-2004 by osc-ubuntu-2204.

## Select the version

The kubernetes packages [repository][repository] change.

You can also override other values from [kubernetes.json][kubernetes.json].

### Before k8s 1.26 

Please set the version you want (Replace 1.22.1 with the kubernetes version you want) in **$HOME/image-builder/images/capi/overwrite-k8s.json**

```json
{
  "build_timestamp": "nightly",
  "kubernetes_deb_gpg_key": "https://packages.cloud.google.com/apt/doc/apt-key.gpg",
  "kubernetes_deb_repo": "\"https://apt.kubernetes.io/ kubernetes-xenial\"",
  "kubernetes_deb_version": "1.22.1-00",
"kubernetes_rpm_gpg_key": "\"https://packages.cloud.google.com/yum/doc/yum-key.gpg https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg\"",
  "kubernetes_rpm_repo": "https://packages.cloud.google.com/yum/repos/kubernetes-el7-{{user `kubernetes_rpm_repo_arch`}}",
  "kubernetes_rpm_version": "1.22.1",
  "kubernetes_semver": "v1.22.1-0",
  "kubernetes_series": "v1.22"
}
```

### After k8s 1.26

Please set the version you want (Replace 1.22.1 with the kubernetes version you want) in **$HOME/image-builder/images/capi/overwrite-k8s.json**

```json
{
  "build_timestamp": "nightly",
  "kubernetes_deb_version": "1.22.1-1.1",
  "kubernetes_rpm_version": "1.22.1",
  "kubernetes_semver": "v1.22.1",
  "kubernetes_series": "v1.22"
}
```

## Download dependencies

```bash
        cd $HOME/image-builder/images/capi
        make deps-osc
```

## Build image

Add packer group, and curent user to packer group

```bash
        sudo groupadd -r packer && sudo useradd -m -s /bin/bash -r -g packer packer
```


Set permision for capi:

```bash
        cp -rf $HOME/image-builder/images/capi /tmp
        sudo chown -R packer:packer /tmp/capi
        sudo chmod -R 777 /tmp/capi

```

Execute packer:

```bash
        sudo runuser -l packer -c "export LANG=C.UTF-8; export LC_ALL=C.UTF-8; export PACKER_LOG=1; export PATH=$HOME/.local/bin/:/tmp/capi/.local/bin:$PATH; export OSC_ACCESS_KEY=${OSC_ACCESS_KEY}; export OSC_SECRET_KEY=${OSC_SECRET_KEY}; export OSC_REGION=${OSC_REGION}; export OSC_ACCOUNT_ID=${OSC_ACCOUNT_ID}; cd /tmp/capi; PACKER_VAR_FILES=overwrite-k8s.json make build-osc-all"
```



<!-- References -->
[repository]: https://kubernetes.io/docs/tasks/administer-cluster/kubeadm/change-package-repository/
[kubernetes.json]: https://github.com/kubernetes-sigs/image-builder/blob/main/images/capi/packer/config/kubernetes.json
