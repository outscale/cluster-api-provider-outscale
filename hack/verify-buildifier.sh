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
TMP_DIR="./lint"
OUT="${TMP_DIR}/out.log"
goarch="$(go env GOARCH)"
goos="$(go env GOOS)"
check_buildifier_installed() {
	if ! [ -x "$(command -v "${BIN_ROOT}/buildifier")" ]; then
		echo 'buildifier is not found installing'
		echo 'Installing buildifier' && install_buildifier
	fi
}

verify_buildifier_version() {

	local buildifier_version
	buildifier_version="$(${BIN_ROOT}/buildifier --version | grep -Eo "([0-9]{1,}\.)+[0-9]{1,}" )"
	if [[ "${MINIMUM_BUILDIFIER_VERSION}" != "${buildifier_version}" ]]; then
		cat <<EOF
Detected buildifier version: ${buildifier_version}
Install ${MINIMUM_BUILDIFIER_VERSION} of buildifier
EOF

		echo 'Installing buildifier' && install_buildifier
	else
		cat <<EOF
Detected buildifier version: ${buildifier_version}.
buildifier ${MINIMUM_BUILDIFIER_VERSION} is already installed.
EOF
	fi
}

install_buildifier() {
	if [[ "${OSTYPE}" == "linux"* || "${OSTYPE}" == "darwin22"* ]]; then
		if ! [ -d "${BIN_ROOT}" ]; then
			mkdir -p "${BIN_ROOT}"
		fi
                curl -sLo "${BIN_ROOT}/buildifier" "https://github.com/bazelbuild/buildtools/releases/download/${MINIMUM_BUILDIFIER_VERSION}/buildifier-${goos}-${goarch}"
                chmod +x "${BIN_ROOT}/buildifier"
	else
		set +x
		echo "The installer does not work for your platform: $OSTYPE"
		exit 1
	fi
}

cleanup() {
  ret=0
  if [[ -s "${OUT}" ]]; then
    echo "Found errors:"
    cat "${OUT}"
    ret=1
  else
    echo "All shell files are passing lint"
  fi
  echo "Cleaning up..."
  rm -rf "${TMP_DIR}"
  exit ${ret}
}
trap cleanup EXIT

run_buildifier(){
                if ! [ -d "${TMP_DIR}" ]; then
                        mkdir -p "${TMP_DIR}"
		fi
		"${BIN_ROOT}/buildifier" -mode="${MODE}" Tiltfile >> "${OUT}" 2>&1
}
check_buildifier_installed "$@"
verify_buildifier_version "$@"
run_buildifier
