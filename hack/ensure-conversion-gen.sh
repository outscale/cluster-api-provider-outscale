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
check_conversion_gen_installed() {
	if ! [ -x "$(command -v "${BIN_ROOT}/conversion-gen")" ]; then
		echo 'conversion-gen is not found installing'
		echo 'Installing conversion-gen' && install_conversion_gen
	else
		echo 'conversion-gen is found'
	fi
}

install_conversion_gen() {
	if [[ "${OSTYPE}" == "linux"* ]]; then
		if ! [ -d "${BIN_ROOT}" ]; then
			mkdir -p "${BIN_ROOT}"
		fi
		go install -v k8s.io/code-generator/cmd/conversion-gen@${MINIMUM_CONVERSION_GEN_VERSION}
		cp "$GOPATH/bin/conversion-gen" "${BIN_ROOT}/conversion-gen"
	else
		set +x
		echo "The installer does not work for your platform: $OSTYPE"
		exit 1
	fi
}
check_conversion_gen_installed "$@"
