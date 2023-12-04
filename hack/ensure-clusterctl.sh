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

BIN_ROOT="./bin"
check_clusterctl_installed() {
  if ! [ -x "$(command -v ${BIN_ROOT}/clusterctl)" ]; then
    echo 'clusterctl not found, installing'
    install_clusterctl
  fi
}

verify_clusterctl_version() {

  local clusterctl_version
  clusterctl_version="$(${BIN_ROOT}/clusterctl version -o short | sed 's/v//')"
  if [[ "${MINIMUM_CLUSTERCTL_VERSION}" != $(echo -e "${MINIMUM_CLUSTERCTL_VERSION}\n${clusterctl_version}" | sort -s -t. -k 1,1n -k 2,2n -k 3,3n | head -n1) ]]; then
    cat <<EOF
Detected clusterctl version: v${clusterctl_version}.
Install v${MINIMUM_CLUSTERCTL_VERSION} of clusterctl.
EOF
    
    echo 'Installing Clusterctl' && install_clusterctl
  else
    cat <<EOF
Detected clusterctl version: v${clusterctl_version}.
Clusterctl v${MINIMUM_CLUSTERCTL_VERSION} is already installed.
EOF
  fi
}

install_clusterctl() {
    if [[ "${OSTYPE}" == "linux"* || "${OSTYPE}" == "darwin22"* ]]; then
      curl -sLo "clusterctl" "https://github.com/kubernetes-sigs/cluster-api/releases/download/v${MINIMUM_CLUSTERCTL_VERSION}/clusterctl-linux-amd64"
      copy_binary
    else
      set +x
      echo "The installer does not work for your platform: $OSTYPE"
      exit 1
    fi

}

function copy_binary() {
      echo "Copy binary in ./bin"
      if ! [ -d "${BIN_ROOT}" ]; then
        mkdir -p "${BIN_ROOT}"
      fi
      sudo mv clusterctl  ${BIN_ROOT}/clusterctl
      chmod +x "${BIN_ROOT}/clusterctl"
      echo "Installation Finished"
}
check_clusterctl_installed "$@"
verify_clusterctl_version "$@"
