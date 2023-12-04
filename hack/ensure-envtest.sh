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
check_envtest_installed() {
	if ! [ -x "$(command -v "${BIN_ROOT}/envtest")" ]; then
		echo 'envtest is not found installing'
		echo 'Installing envtest' && install_envtest
	fi
}

verify_envtest_version() {

	local envtest_version
	envtest_version="$(${BIN_ROOT}/envtest list | grep -Eo "([0-9]{1,}\.)+[0-9]{1,}" | head -1)"
	if [[ "${MINIMUM_ENVTEST_VERSION}" != "${envtest_version}" ]]; then
		cat <<EOF
Detected envtest install kubernetes version: v${envtest_version}
Install envtest with kubernetes version v${MINIMUM_ENVTEST_VERSION}
EOF

		echo 'Installing envtest' && install_envtest
	else
		cat <<EOF
Detected envtest install kubernetes version: v${envtest_version}.
envtest with kubernetes version v${MINIMUM_ENVTEST_VERSION} is already installed.
EOF
	fi
}

install_envtest() {
	if [[ "${OSTYPE}" == "linux"* || "${OSTYPE}" == "darwin22"* ]]; then
		if ! [ -d "${BIN_ROOT}" ]; then
			mkdir -p "${BIN_ROOT}"
		fi
		go install -v sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
		cp "$GOPATH/bin/setup-envtest" "${BIN_ROOT}/envtest"
	else
		set +x
		echo "The installer does not work for your platform: $OSTYPE"
		exit 1
	fi
}

check_envtest_installed "$@"
verify_envtest_version "$@"
