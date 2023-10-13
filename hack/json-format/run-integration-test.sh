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

set -o nounset
set -o pipefail

ROOT=$(pwd)

# check_json_format_installed checks that json_format is installed.
function check_json_format_installed() {
	kubernetes_output_json="$check_working_binary_directory/json_format"
	
	if [[ -f "$kubernetes_output_json" ]]; then
		echo "json-format is found"
	else
		echo "json-format is not found"
		exit 1
	fi
}


# check_file_exist checks that json output exist.
function check_file_exist() {
	kubernetes_output_json="$ROOT/$check_output" 
	if [[ -f "$kubernetes_output_json" ]]; then
		echo "kubernetes_output_json is found"
	else
		echo "kubernetes_output_json is not found"
		exit 1
	fi
}

# check_content checks that every json parameters of output json file.
function check_content() {
	kubernetes_deb_version=$(jq .kubernetes_deb_version "$ROOT"/"$check_output")
	echo "$kubernetes_deb_version"
	kubernetes_rpm_version=$(jq .kubernetes_rpm_version "$ROOT"/"$check_output")
	echo "$kubernetes_rpm_version"
	kubernetes_semver=$(jq .kubernetes_semver "$ROOT"/"$check_output")
	kubernetes_series=$(jq .kubernetes_series "$ROOT"/"$check_output")
	if [[ $kubernetes_deb_version == '"1.22.1-1.1"' ]]; then
		echo "Find kubernetes_deb_version"
	else
		echo "Can not find kubernetes_deb_version"
		exit 1
	fi
	if [[ $kubernetes_rpm_version == '"1.22.1"' ]]; then	
		echo "Find kubernetes_rpm_version"
	else	
		echo "Can not find kubernetes_rpm_version"
		exit 1
	fi
	if [[ $kubernetes_semver == '"v1.22.1"' ]]; then	
		echo "Find kubernetes_semver"
	else	
		echo "Can not find kubernetes_semver"
		exit 1
	fi
	if [[ $kubernetes_series == '"v1.22"' ]]; then	
		echo "Find kubernetes_series"
	else	
		echo "Can not find kubernetes_series"
		exit 1
	fi	
}

# run_json_format set default values and run json_format
function run_json_format() {
	check_format=${format:-}
	if [ -z "${check_format}" ]; then
		echo "format is unset";
		echo "Set default values"
		check_format="params"
	else
		echo "format is set to '$check_format'"
	fi
	check_kubernetes_series=${kubernetes_series:-}
	if [ -z "${check_kubernetes_series}" ]; then
		echo "kubenetes_series is unset";
		echo "Set default values"
		check_kubernetes_series="v1.22"
	else		
		echo "kubernetes_series is set to '$check_kubernetes_series'"
	fi

	check_kubernetes_semver=${kubernetes_semver:-}
	if [ -z "${check_kubernetes_semver}" ]; then	
		echo "kubernetes_semver is unset";
		echo "Set default values"
		check_kubernetes_semver="v1.22.1"
	else		
		echo "kubernetes_semver is set to '$check_kubernetes_semver'"
	fi
	check_input=${input:-}
	if [ -z "${check_input}" ]; then	
		echo "input is unset";
		echo "Set default values"
		check_input="kubernetes.json"
	else		
		echo "input is set to '$check_input'"
	fi
	check_output=${output:-}
	if [ -z "${check_output}" ]; then
		echo "output is unset";
		echo "Set default values"
		check_output="overwrite-1.22.json"
	else		
		echo "output is set to '$check_output'"
	fi
	check_working_binary_directory=${working_binary_directory:-}
	if [ -z "${check_working_binary_directory}" ]; then
		echo "working_directory is unset";
		echo "Set default values"
		check_working_binary_directory=$ROOT/target/x86_64-unknown-linux-musl/release 
	else
		echo "working_binary_directory is set to '$check_working_binary_directory'"
	fi
	"$check_working_binary_directory"/json_format  --format $check_format --kubernetes-series $check_kubernetes_series --kubernetes-semver $check_kubernetes_semver --input $check_input --output $check_output
	echo "$check_working_binary_directory/json_format --format $check_format --kubernetes-series $check_kubernetes_series --kubernetes-semver $check_kubernetes_semver --input $check_input --output $check_output"
}

# usage  show the usage of the script with parameters.
function usage {
    echo "./run-integration-test.sh -f format -k kubernetes_series -r kubernetes_semver -i input -o output --> shows usage"
}

optstring=":k:r:f:i:o:w:"
while getopts ${optstring} arg; do
  case ${arg} in
    f)
	  format=${OPTARG}
	  ;;
	k)
	  kubernetes_series=${OPTARG}
	  ;;
	r)
	  kubernetes_semver=${OPTARG}
	  ;;
	i)
	  input=${OPTARG}
	  ;;
	o)
	  output=${OPTARG}
	  ;;
	w)
	  working_binary_directory=${OPTARG}
	  ;;
	*)
	  echo "showing usage!"
	  usage;;
	esac
done

run_json_format
check_json_format_installed
check_file_exist
check_content
