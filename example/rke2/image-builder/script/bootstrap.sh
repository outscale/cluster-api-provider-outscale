#!/bin/bash
# Copyright 2022 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
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
