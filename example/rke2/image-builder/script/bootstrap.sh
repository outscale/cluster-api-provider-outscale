#!/usr/bin/env bash
set -o errexit
set -o nounset
set -o pipefail

echo "Bootstrap rke2"
apt-get update 
apt-get install -y curl systemd 
mkdir -p /opt/rke2-artifacts
curl -sfL -o /opt/rke2-artifacts/rke2-images.linux-amd64.tar.zst https://github.com/rancher/rke2/releases/download/$1/rke2-images.linux-amd64.tar.zst
curl -sfL -o /opt/rke2-artifacts/rke2.linux-amd64.tar.gz https://github.com/rancher/rke2/releases/download/$1/rke2.linux-amd64.tar.gz
curl -sfL -o /opt/rke2-artifacts/sha256sum-amd64.txt https://github.com/rancher/rke2/releases/download/$1/sha256sum-amd64.txt
curl -sfL -o /opt/install.sh https://get.rke2.io

echo "Deactivate Hotplug, enable cillium"
sudo rm /etc/cloud/cloud.cfg.d/06_hotplug.cfg
sudo mv /tmp/06_hotplug.cfg /etc/cloud/cloud.cfg.d/06_hotplug.cfg 
sudo cloud-init clean --logs
sudo cloud-init init --local
sudo cloud-init init
sudo cloud-init modules --mode=config
sudo cloud-init modules --mode=final
echo "Done"
