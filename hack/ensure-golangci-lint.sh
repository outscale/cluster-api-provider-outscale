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

check_golangci_lint_installed() {
	if ! [ -x "$(command -v "${BIN_ROOT}/golangci-lint")" ]; then
		echo 'golangci-lint is not found installing'
		echo 'Installing golangci-lint' && install_golangci_lint
	fi
}

verify_golangci_lint_version() {

	local golangci_lint_version
	golangci_lint_version="$(${BIN_ROOT}/golangci-lint version | grep -Eo "([0-9]{1,}\.)+[0-9]{1,}" )"
	if [[ "${MINIMUM_GOLANGCI_LINT_VERSION}" != "${golangci_lint_version}" ]]; then
		cat <<EOF
Detected golangci_lint version: ${golangci_lint_version}
Install ${MINIMUM_GOLANGCI_LINT_VERSION} of golangci_lint
EOF

		echo 'Installing golangci_lint' && install_golangci_lint
	else
		cat <<EOF
Detected golangci_lint version: ${golangci_lint_version}.
golint ${MINIMUM_GOLANGCI_LINT_VERSION} is already installed.
EOF
	fi
}

install_golangci_lint() {
	if [[ "${OSTYPE}" == "linux"* || "${OSTYPE}" == "darwin22"* ]]; then
		if ! [ -d "${BIN_ROOT}" ]; then
			mkdir -p "${BIN_ROOT}"
		fi
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "$GOPATH/bin" "v${MINIMUM_GOLANGCI_LINT_VERSION}"
		cp "$GOPATH/bin/golangci-lint" ${BIN_ROOT}/golangci-lint
	else
		set +x
		echo "The installer does not work for your platform: $OSTYPE"
		exit 1
	fi
}

check_golangci_lint_installed "$@"
verify_golangci_lint_version "$@"
