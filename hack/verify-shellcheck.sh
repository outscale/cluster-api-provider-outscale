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
disabled=(
  1090
  2230
  1091
)

BIN_ROOT="./bin"
TMP_DIR="./lint"
OUT="${TMP_DIR}/out.log"
goos="$(go env GOOS)"
check_shellcheck_installed() {
	if ! [ -x "$(command -v "${BIN_ROOT}/shellcheck")" ]; then
		echo 'shellcheck is not found installing'
		echo 'Installing shellcheck' && install_shellcheck
	fi
}

verify_shellcheck_version() {

	local shellcheck_version
	shellcheck_version="$(${BIN_ROOT}/shellcheck --version | grep -Eo "([0-9]{1,}\.)+[0-9]{1,}" )"
	if [[ "${MINIMUM_SHELLCHECK_VERSION}" != "${shellcheck_version}" ]]; then
		cat <<EOF
Detected shellcheck version: ${shellcheck_version}
Install ${MINIMUM_SHELLCHECK_VERSION} of shellcheck
EOF

		echo 'Installing shellcheck' && install_shellcheck
	else
		cat <<EOF
Detected shellcheck version: ${shellcheck_version}.
shellcheck ${MINIMUM_SHELLCHECK_VERSION} is already installed.
EOF
	fi
}

install_shellcheck() {
	if [[ "${OSTYPE}" == "linux"* || "${OSTYPE}" == "darwin"* ]]; then
		if ! [ -d "${BIN_ROOT}" ]; then
			mkdir -p "${BIN_ROOT}"
		fi
                curl -sLo "${BIN_ROOT}/shellcheck.x86_64.tar.xz" "https://github.com/koalaman/shellcheck/releases/download/v${MINIMUM_SHELLCHECK_VERSION}/shellcheck-v${MINIMUM_SHELLCHECK_VERSION}.${goos}.x86_64.tar.xz"
                tar xf  "${BIN_ROOT}/shellcheck.x86_64.tar.xz" -C "${BIN_ROOT}"
                chmod +x "${BIN_ROOT}/shellcheck-v${MINIMUM_SHELLCHECK_VERSION}/shellcheck"
		mv "${BIN_ROOT}/shellcheck-v${MINIMUM_SHELLCHECK_VERSION}/shellcheck" "${BIN_ROOT}"
                rm -rf "${BIN_ROOT}/shellcheck.x86_64.tar.xz"
		rm -rf "${BIN_ROOT}/shellcheck-v${MINIMUM_SHELLCHECK_VERSION}/"
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

join_by() {
  local IFS="$1";
  shift;
  echo "$*";
}

run_shellcheck(){
                if ! [ -d "${TMP_DIR}" ]; then
                        mkdir -p "${TMP_DIR}"
                fi
		SHELLCHECK_DISABLED="$(join_by , "${disabled[@]}")"
		readonly SHELLCHECK_DISABLED
		ROOT_PATH=$(git rev-parse --show-toplevel)
		IGNORE_FILES=$(find "${ROOT_PATH}" -name "*.sh" | grep "tilt_modules|vendor")
		echo "Ignoring shellcheck on ${IGNORE_FILES}"
		FILES=$(find "${ROOT_PATH}" -name "*.sh" -not -path "./tilt_modules/*" -not -path "./vendor/*")
		while read -r file; do
		   "${ROOT_PATH}/bin/shellcheck" "--exclude=${SHELLCHECK_DISABLED}" "--color=auto" -x "$file" >> "${OUT}" 2>&1
		done <<< "$FILES"
}

check_shellcheck_installed "$@"
verify_shellcheck_version "$@"
run_shellcheck
