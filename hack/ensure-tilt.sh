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

check_tilt_installed() {
	if ! [ -x "$(command -v tilt)" ]; then
		echo 'tilt not found, installing'
		install_tilt
	fi
}

verify_tilt_version() {

	local tilt_version
	tilt_version="$(tilt version | grep -Eo "([0-9]{1,}\.)+[0-9]{1,}")"
	if [[ "${MINIMUM_TILT_VERSION}" != $(echo -e "${MINIMUM_TILT_VERSION}\n${tilt_version}" | sort -s -t. -k 1,1n -k 2,2n -k 3,3n | head -n1) ]]; then
		cat <<EOF
Detected tilt version: v${tilt_version}.
Install v${MINIMUM_TILT_VERSION} of Tilt
EOF

		echo 'Installing Tilt' && install_tilt
	else
		cat <<EOF
Detected tilt version: v${tilt_version}.
Tilt v${MINIMUM_TILT_VERSION} is already install.
EOF
	fi
}

install_tilt() {
	if [[ "${OSTYPE}" == "linux"* || "${OSTYPE}" == "darwin22"* ]]; then
		curl -fsSL "https://github.com/tilt-dev/tilt/releases/download/v$MINIMUM_TILT_VERSION/tilt.$MINIMUM_TILT_VERSION.linux.x86_64.tar.gz" | tar -xzv tilt
		copy_binary
	else
		set +x
		echo "The installer does not work for your platform: $OSTYPE"
		exit 1
	fi
}

function copy_binary() {
	echo "Copy binary in /usr/local/bin which is protected as sudo"
	sudo mv tilt /usr/local/bin/tilt
	chmod +x /usr/local/bin/tilt
	echo "Installation Finished"
}

check_tilt_installed "$@"
verify_tilt_version "$@"
