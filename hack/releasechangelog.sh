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

while getopts ":g:o:p:t:r:i:" option; do
	case "${option}" in
		g)
			GH=${OPTARG}
			;;
		o)
			GH_ORG_NAME=${OPTARG}
			;;
		t)
			RELEASE_TAG=${OPTARG}
			;;
		p)
			PREVIOUS_RELEASE_TAG=${OPTARG}
			;;
		i)
			IMG_RELEASE=${OPTARG}
			;;
		r)
			GH_REPO_NAME=${OPTARG}
			;;
    *) echo "usage: $0 [-v] [-r]" >&2
       exit 1 ;;
	esac
done

echo "# Release notes for Cluster API Provider Outscale (CAPO) $RELEASE_TAG"
echo "# Changelog since $PREVIOUS_RELEASE_TAG"

if [ "$PREVIOUS_RELEASE_TAG" == "None" ]; then
	$GH api "repos/$GH_ORG_NAME/$GH_REPO_NAME/releases/generate-notes" -F tag_name="$RELEASE_TAG" --jq '.body'
else
	$GH api "repos/$GH_ORG_NAME/$GH_REPO_NAME/releases/generate-notes" -F tag_name="$RELEASE_TAG" -F previous_tag_name="$PREVIOUS_RELEASE_TAG" --jq '.body'
fi

echo "**The release image is**: $IMG_RELEASE"
echo "Thanks to all our contributors!"
