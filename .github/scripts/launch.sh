#!/bin/bash

set -e
root=$(cd "$(dirname "$0")/../.." && pwd)
# get_date return date
function get_date {
    backup_dir=$(date +'%Y-%m-%d')
    echo "$backup_dir"
}

function usage {
    echo "./$(basename "$0") -u image_url -l launch_script -f --> show usage"    
}

# check_gh set default values.
function check_gh {
    if [ -z "${IMAGE_URL}" ]; then
        echo "IMAGE_URL is unset";
        echo "Set default Values";
        IMAGE_URL="https://raw.githubusercontent.com/kubernetes-sigs/image-builder/master/images/capi/packer/outscale/ci/nightly"
    else
        echo "IMAGE_URL is set to '$IMAGE_URL'"
    fi
    IMAGE_TARGET=("centos-7" "ubuntu-2004") 
    if [ -z "${GIT_FILE_PATH}" ]; then
        echo "GIT_FILE_PATH is unset"; 
        echo "Set Default Values";
        GIT_FILE_PATH=docs/src/topics/omi.md
    else
        echo "README is set to '$GIT_FILE_PATH'"
    fi

    if [ -z "${GIT_CURRENT_BRANCH}" ]; then
        echo "GIT_CURRENT_BRANCH is unset";
        echo "Set default Values";
        GIT_CURRENT_BRANCH=main
    else
        echo "GIT_CURRENT_BRANCH is set to '$GIT_CURRENT_BRANCH'"
    fi

    if [ -z "${GH_ORG_NAME}" ]; then
        echo "GH_ORG_NAME is unset";
        echo "Set default Values";
        GH_ORG_NAME="outscale-dev"
    else
        echo "GH_ORG_NAME is set to '$GH_ORG_NAME'"
    fi
    
    if [ -z "${GH_REPO_NAME}" ]; then
        echo "GH_REPO_NAME is unset":
	echo "Set default Values";
	GH_REPO_NAME="cluster-api-provider-outscale"
    else
	echo "GH_REPO_NAME is set to '$GH_REPO_NAME'"
    fi
    if [ -z "${GIT_USERNAME}" ]; then
        echo "GIT_USERNAME is unset";
        echo "Set default Values";
        GIT_USERNAME="Outscale Bot"
    else
        echo "GIT_USERNAME is set to '$GIT_USERNAME'"
    fi
    if [ -z "${GIT_USEREMAIL}" ]; then
        echo "GIT_USEREMAIL is unset";
	echo "Set default Values";
	GIT_USEREMAIL="opensource+bot@outscale.com"
    else 
        echo "GIT_USEREMAIL is set to '$GIT_USEREMAIL'"
    fi
  }

# get_file return image file version
function get_file {
    if curl --head --silent -i --fail "$IMAGE_URL/$1" 2>/dev/null;
    then
        echo "$IMAGE_URL/$1 exist"
        curl "$IMAGE_URL/$1" -o "$1"
    else
        echo "$IMAGE_URL/$1 does not exist"
    fi
}

# get_version return kubernetes version.
function get_version {
    if [ -f "$1" ];
    then
        version=$(jq .kubernetes_semver "$1")
        echo "$version"
        rm -f "$1"
    else
        echo "the file $1 does not exist"
    fi
}

# get_name return version name.
function get_name {
  declare -A hashmap
  hashmap["centos-7"]="7.7"
  hashmap["ubuntu-2004"]="2004"
  date=$(get_date)
  echo "$date"
  for os_target in "${IMAGE_TARGET[@]}"
  do
    for version in {21..25}
    do
      get_file overwrite-1-"${version}".json
      k8s_version=$(get_version overwrite-1-"${version}".json | tr -d '"')
      image_version="${os_target}-${hashmap[$os_target]}-${k8s_version}-${date}"
      echo "${image_version}"
      if [[ "$os_target" == "centos-7" ]]
      then
        sed -i "/^centos:/a - ${image_version}" "$root"/$GIT_FILE_PATH
      elif [[ "$os_target" == "ubuntu-2004" ]]
      then
        sed -i "/^ubuntu:/a - ${image_version}"  "$root"/$GIT_FILE_PATH
      else
        echo "os version not found"
      fi
    done
  done
  "$root"/.github/scripts/git_command.sh -o "${GH_ORG_NAME}" -r "${GH_REPO_NAME}" -u "${GIT_CURRENT_BRANCH}" -n "${GIT_USERNAME}" -e "${GIT_USEREMAIL}" -t add_"${k8s_version}" -b add-k8s-"${k8s_version}" -m Update_kubernetes_OMI_"${k8s_version}" -c add_"${k8s_version}" -p "$GIT_FILE_PATH" -g "$root"/bin/gh
}


optstring=":u:t:c:o:r:n:e:"
while getopts ${optstring} arg; do
  case ${arg} in
    u)
      IMAGE_URL=${OPTARG}
      ;;
    f)
      GIT_FILE_PATH=${OPTARG}
      ;;
    c) 
      GIT_CURRENT_BRANCH=${OPTARG}
      ;;
    o)
      GH_ORG_NAME=${OPTARG}
      ;;
    r)
      GH_REPO_NAME=${OPTARG}
      ;;
    n)
      GIT_USERNAME=${OPTARG}
      ;;
    e)
      GIT_USEREMAIL=${OPTARG}
      ;;
    *)
      echo "showing usage!"
      usage
      ;;
  esac
done


check_gh  
get_name
