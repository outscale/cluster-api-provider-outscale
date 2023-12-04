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

check_controller_gen_installed() {
	if ! [ -x "$(command -v "${BIN_ROOT}/controller-gen")" ]; then
		echo 'controller-gen is not found installing'
		echo 'Installing controller-gen' && install_controller_gen
	fi
}

verify_controller_gen_version() {

	local controller_gen_version
	controller_gen_version="$(${BIN_ROOT}/controller-gen --version | grep -Eo "([0-9]{1,}\.)+[0-9]{1,}" | head -1)"
	if [[ "${MINIMUM_CONTROLLER_GEN_VERSION}" != "${controller_gen_version}" ]]; then
		cat <<EOF
Detected controller-gen version: ${controller_gen_version}
Install ${MINIMUM_CONTROLLER_GEN_VERSION} of controller-gen
EOF

		echo 'Installing controller-gen' && install_controller_gen
	else
		cat <<EOF
Detected controller-gen version: ${controller_gen_version}.
controller-gen ${MINIMUM_CONTROLLER_GEN_VERSION} is already installed.
EOF
	fi
}

install_controller_gen() {
	if [[ "${OSTYPE}" == "linux"* || "${OSTYPE}" == "darwin22"* ]]; then
		if ! [ -d "${BIN_ROOT}" ]; then
			mkdir -p "${BIN_ROOT}"
		fi
		go install -v sigs.k8s.io/controller-tools/cmd/controller-gen@master
		cp "$GOPATH/bin/controller-gen" "${BIN_ROOT}/controller-gen"
	else
		set +x
		echo "The installer does not work for your platform: $OSTYPE"
		exit 1
	fi
}

check_controller_gen_installed "$@"
verify_controller_gen_version "$@"
