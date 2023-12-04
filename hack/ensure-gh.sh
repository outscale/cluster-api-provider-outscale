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

goarch="$(go env GOARCH)"
goos="$(go env GOOS)"
if [ "$goos" != "linux" ]; then
	echo "OS '$OSTYPE' not supported. Aborting." >&2
	exit 1
fi

check_gh_installed() {
	if ! [ -x "$(command -v "${BIN_ROOT}/gh")" ]; then
		echo 'gh is not found installing'
		echo 'Installing gh' && install_gh
	fi
}

verify_gh_version() {

	local gh_version
	gh_version="$(${BIN_ROOT}/gh version | grep -Eo "([0-9]{1,}\.)+[0-9]{1,}" | head -1)"
	gh_version="$(echo "${gh_version}" | xargs)"
	if [[ "${MINIMUM_GH_VERSION}" != "${gh_version}" ]]; then
		cat <<EOF
Detected gh version: ${gh_version}
Install ${MINIMUM_GH_VERSION} of gh
EOF

		echo 'Installing gh' && install_gh
	else
		cat <<EOF
Detected gh version: ${gh_version}
Gh ${MINIMUM_GH_VERSION} is already installed.
EOF
	fi
}

install_gh() {
	if [[ "${OSTYPE}" == "linux"* || "${OSTYPE}" == "darwin22"* ]]; then
		if ! [ -d "${BIN_ROOT}" ]; then
			mkdir -p "${BIN_ROOT}"
		fi
		curl -sLo "${BIN_ROOT}/gh_${MINIMUM_GH_VERSION}_${goos}_${goarch}.tar.gz" "https://github.com/cli/cli/releases/download/v${MINIMUM_GH_VERSION}/gh_${MINIMUM_GH_VERSION}_${goos}_${goarch}.tar.gz"
		tar -zvxf "${BIN_ROOT}/gh_${MINIMUM_GH_VERSION}_${goos}_${goarch}.tar.gz" -C "${BIN_ROOT}/"
		mv "${BIN_ROOT}/gh_${MINIMUM_GH_VERSION}_${goos}_${goarch}/bin/gh" "${BIN_ROOT}/gh"
		chmod +x "${BIN_ROOT}/gh"
		rm "${BIN_ROOT}/gh_${MINIMUM_GH_VERSION}_${goos}_${goarch}.tar.gz"
		rm -rf "${BIN_ROOT}/gh_${MINIMUM_GH_VERSION}_${goos}_${goarch}"
	else
		set +x
		echo "The installer does not work for your platform: $OSTYPE"
		exit 1
	fi
}

check_gh_installed "$@"
verify_gh_version "$@"
