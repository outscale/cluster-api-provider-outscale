#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

check_clusterctl_installed() {
  if ! [ -x "$(command -v clusterctl)" ]; then
    echo 'clusterctl not found, installing'
    install_clusterctl
  fi
}


verify_clusterctl_version() {

  local clusterctl_version
  clusterctl_version="$(clusterctl version -o short | sed 's/v//')"
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
    if [[ "${OSTYPE}" == "linux"* ]]; then
      curl -sLo "clusterctl" "https://github.com/kubernetes-sigs/cluster-api/releases/download/v${MINIMUM_CLUSTERCTL_VERSION}/clusterctl-linux-amd64"
      copy_binary
    else
      set +x
      echo "The installer does not work for your platform: $OSTYPE"
      exit 1
    fi

}

function copy_binary() {
      echo "Copy binary in /usr/local/bin which is protected as sudo"
      sudo mv clusterctl /usr/local/bin/clusterctl
      chmod +x "/usr/local/bin/clusterctl"
      echo "Installation Finished"
}

check_clusterctl_installed "$@"
verify_clusterctl_version "$@"
