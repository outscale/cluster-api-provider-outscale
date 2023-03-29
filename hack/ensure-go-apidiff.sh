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
check_go_apidiff_installed() {
	if ! [ -x "$(command -v "${BIN_ROOT}/go-apidiff")" ]; then
		echo 'go-apidiff is not found installing'
		echo 'Installing go-apidiff' && install_go_apidiff
	fi
}


install_go_apidiff() {
	if [[ "${OSTYPE}" == "linux"* ]]; then
		if ! [ -d "${BIN_ROOT}" ]; then
			mkdir -p "${BIN_ROOT}"
		fi
		go install -v github.com/joelanford/go-apidiff@${MINIMUM_GO_APIDIFF_VERSION}
		cp "$GOPATH/bin/go-apidiff" "${BIN_ROOT}/go-apidiff"
	else
		set +x
		echo "The installer does not work for your platform: $OSTYPE"
		exit 1
	fi
}

check_go_apidiff_installed "$@"