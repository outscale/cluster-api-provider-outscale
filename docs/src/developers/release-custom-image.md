
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

Please set the version you want (Replace 1.22.1 with the version you want in kubernetes) in **$HOME/image-builder/images/capi/overwrite-k8s.json**

```json
{
  "build_timestamp": "nightly",
  "kubernetes_deb_version": "1.22.1-1.1",
  "kubernetes_rpm_version": "1.22.1,
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

First install packer

And then:

```bash
        export PATH=$HOME/.local/bin:$HOME/image-builder/images/capi/.local/bin:$PATH
        sudo groupadd -r packer && sudo useradd -m -s /bin/bash -r -g packer packer
        cp -rf $HOME/image-builder/images/capi /tmp
        sudo chown -R packer:packer /tmp/capi
        sudo chmod -R 777 /tmp/capi
        sudo runuser -l packer -c "export LANG=C.UTF-8; export LC_ALL=C.UTF-8; export PACKER_LOG=1; export PATH=~packer/.local/bin/:/tmp/capi/.local/bin:$PATH; export OSC_ACCESS_KEY=${OSC_ACCESS_KEY}; export OSC_SECRET_KEY=${OSC_SECRET_KEY}; export OSC_REGION=${OSC_REGION}; export OSC_ACCOUNT_ID=${OSC_ACCOUNT_ID}; cd /tmp/capi; PACKER_VAR_FILES=overwrite-k8s.json make build-osc-all
```



