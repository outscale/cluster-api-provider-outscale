#!/bin/bash
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
	*)
		echo "usage: $0 [-v] [-r]" >&2
		exit 1
		;;
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
