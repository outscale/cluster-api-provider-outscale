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
check_mockgen_installed() {
	if ! [ -x "$(command -v "${BIN_ROOT}/mockgen")" ]; then
		echo 'mockgen is not found installing'
		echo 'Installing mockgen' && install_mockgen
	fi
}

verify_mockgen_version() {

	local mockgen_version
	mockgen_version="$(${BIN_ROOT}/mockgen --version | grep -Eo "([0-9]{1,}\.)+[0-9]{1,}" | head -1)"
	if [[ "${MINIMUM_MOCKGEN_VERSION}" != "${mockgen_version}" ]]; then
		cat <<EOF
Detected mockgen version: ${mockgen_version}
Install ${MINIMUM_MOCKGEN_VERSION} of mockgen
EOF

		echo 'Installing mockgen' && install_mockgen
	else
		cat <<EOF
Detected mockgen version: ${mockgen_version}.
mockgen ${MINIMUM_MOCKGEN_VERSION} is already installed.
EOF
	fi
}

install_mockgen() {
	if [[ "${OSTYPE}" == "linux"* || "${OSTYPE}" == "darwin22"* ]]; then
		if ! [ -d "${BIN_ROOT}" ]; then
			mkdir -p "${BIN_ROOT}"
		fi
		go install -v "github.com/golang/mock/mockgen@v${MINIMUM_MOCKGEN_VERSION}"
		cp "$GOPATH/bin/mockgen" "${BIN_ROOT}/mockgen"
	else
		set +x
		echo "The installer does not work for your platform: $OSTYPE"
		exit 1
	fi
}

check_mockgen_installed "$@"
verify_mockgen_version "$@"
