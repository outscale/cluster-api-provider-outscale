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

check_kubectl_installed() {
	# If kubectl is not available on the path, get it
	if ! [ -x "$(command -v /tmp/kubectl)" ]; then
		echo 'kubectl not found, installing'
		install_kubectl
	fi
}

verify_kubectl_version() {

	local kubectl_version
	kubectl_version="v$(kubectl version --client --short | grep -Eo "([0-9]{1,}\.)+[0-9]{1,}")"
	if [[ "v${MINIMUM_KUBECTL_VERSION}" != "${kubectl_version}" ]]; then
		cat <<EOF
Detected kubectl version: ${kubectl_version}
Install ${MINIMUM_KUBECTL_VERSION} of kubectl.
EOF

		echo 'Installing Kubectl' && install_kubectl
	else
		cat <<EOF
Detected kubectl version: ${kubectl_version}
Kubectl ${MINIMUM_KUBECTL_VERSION} is already installed.
EOF
	fi
}

install_kubectl() {
	if [[ "${OSTYPE}" == "linux"* || "${OSTYPE}" == "darwin22"* ]]; then
		curl -sLo "kubectl" "https://storage.googleapis.com/kubernetes-release/release/v${MINIMUM_KUBECTL_VERSION}/bin/linux/amd64/kubectl"
		copy_binary
	else
		set +x
		echo "The installer does not work for your platform: $OSTYPE"
		exit 1
	fi
}

function copy_binary() {
	echo "Copy binaries in /usr/local/bin which is protected as sudo"
	sudo mv kubectl /usr/local/bin/kubectl
	chmod +x "/usr/local/bin/kubectl"
	echo "Installation Finished"
}
check_kubectl_installed "$@"
verify_kubectl_version "$@"
