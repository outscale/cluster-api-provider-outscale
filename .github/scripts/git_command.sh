#!/bin/bash

set -e
root=$(cd "$(dirname "$0")/../.." && pwd)
function usage {
        echo "./$(basename "$0") -g GH_PATH -o GH_ORG_NAME -r GH_REPO_NAME -n GIT_USERNAME -t GIT_TITLE -e GIT_USEREMAIL -p GIT_FILE_PATH -m GIT_COMMIT_MESSAGE -b GIT_BRANCH -c GIT_CONTENT_BODY -u GIT_USE_BRANCH --> shows usage"
}

# github_login allow to log to github.

function github_login {
    cat <<< "${SECRET_GITHUB_TOKEN}" | $GH auth login --with-token
}

# github_add configure and add git file.

function github_add {
    git config user.name "$GIT_USERNAME"
    git config user.email "$GIT_USEREMAIL"
    for f in $root/$GIT_FILE_PATH; do
      git add "$f" || true
    done
}

# github_commit commit git file if needed.

function github_commit {
   echo "$GIT_FILE_PATH"
   if git status | grep -q ".*$GIT_FILE_PATH.*" 
   then
       git commit -m "$GIT_COMMIT_MESSAGE"
   else
       echo "No changes to commit"
   fi
}

# github_pr create pr if needed.

function github_pr {
   git branch
   echo "$GIT_TITLE"
   if gh pr list | grep -q "$GIT_TITLE" 
   then
     echo "Pr Already exist"
   else
     git checkout "$GIT_USE_BRANCH"
     result=$(curl -s -X POST -H "Authorization: token $SECRET_GITHUB_TOKEN" -d "{\"head\":\"outscale-vbr:$GIT_BRANCH\",\"base\":\"main\",\"title\":\"$GIT_TITLE\",\"body\":\"$GIT_CONTENT_BODY\"}" "https://api.github.com/repos/outscale-dev/$GH_REPO_NAME/pulls")
     errors=$(echo "$result" | jq .errors)
     if [ "$errors" != "null" ]; then
       echo "$errors"
       echo "errors while creating pull request"    
     fi 
   fi
}

# github_push push git branch.

function github_push {
   git push "https://${SECRET_GITHUB_TOKEN}@github.com/${GH_ORG_NAME}/${GH_REPO_NAME}.git"
}

# github_checkout checkout branch.

function github_checkout {
   branch=$(git symbolic-ref HEAD | sed -e 's,.*/\(.*\),\1,')
   if [ "$branch" != "$GIT_BRANCH" ]; then
   	git checkout -b "$GIT_BRANCH"
   else
        echo "'$GIT_BRANCH' already exist"
        git branch -D "$GIT_BRANCH"
        git checkout -b "$GIT_BRANCH"
   fi
   
}

# check_gh set default values.

function check_gh {
	if [ -z "${GH}" ]; then
            echo "GH is unset";
            echo "Set Default Values";
            GH=gh
        else
            echo "GH is set to '$GH'";

        fi
	if [ -z "${GH_ORG_NAME}" ]; then
            echo "GH_ORG_NAME is unset";
            echo "Set Default Values";
            GH_ORG_NAME=outscale-dev         
	else
	    echo "GH_ORG_NAME is set to '$GH_ORG_NAME'"
	fi
	if [ -z "${GH_REPO_NAME}" ]; then
	    echo "GH_REPO_NAME is unset";
            echo "Set Default Values";
	    GH_REPO_NAME=cluster-api-provider-outscale
        else
            echo "GH_REPO_NAME is set to '$GH_REPO_NAME'"
        fi
	if [ -z "${GIT_USERNAME}" ]; then
            echo "GIT_USERNAME is unset";
            echo "Set Default Values";
	    GIT_USERNAME="Outscale Bot"
	else
            echo "GIT_USERNAME is set to '$GIT_USERNAME'"
	fi
       	if [ -z "${GIT_USEREMAIL}" ]; then
            echo "GIT_USEREMAIL is unset";
	    echo "Set Default Values";
	    GIT_USEREMAIL="opensource+bot@outscale.com" 
        else
            echo "GIT_USEREMAIL is set to '$GIT_USEREMAIL'"
	fi
	if [ -z "${GIT_FILE_PATH}" ]; then
	   echo "GIT_FILE_PATH is unset";
	   echo "Set Default Values";
	   GIT_FILE_PATH="README.md"
        else
           echo "GIT_FILE_PATH is set to '$GIT_FILE_PATH'"
        fi 
	if [ -z "${GIT_COMMIT_MESSAGE}" ]; then
            echo "GIT_COMMIT_MESSAGE is unset";
	    echo "Set default Values";
            GIT_COMMIT_MESSAGE="Update kubernetes OMI"
        else
            echo "GIT_COMMIT_MESSAGE is set to '$GIT_COMMIT_MESSAGE'"
        fi      
        if [ -z "${GIT_BRANCH}" ]; then
            echo "GIT_BRANCH is unset";
            echo "Set default Values";
            GIT_BRANCH="branch"
        else
            echo "GIT_BRANCH is set to '$GIT_BRANCH'"
        fi 
        echo "${GIT_TITLE}"
        if [ -z "${GIT_TITLE}" ]; then
            echo "GIT_TITLE is unset";
            echo "Set Default Values";
            GIT_TITLE="My Title"
        else
            echo "GIT_TITLE is set to '$GIT_TITLE'"
        fi 
        if [ -z "${GIT_CONTENT_BODY}" ]; then
            echo "GIT_CONTENT_BODY is unset";
            echo "Set Default values";
            GIT_CONTENT_BODY="Pr to validate"
        else
            echo "GIT_CONTENT_BODY is set to '$GIT_CONTENT_BODY'"
        fi
        if [ -z "${GIT_USE_BRANCH}" ]; then
            echo "GIT_USE_BRANCH is unset";
            echo "Set Default values";
            GIT_USE_BRANCH=main
        else
            echo "GIT_USE_BRANCH is set to '$GIT_USE_BRANCH'"
        fi
}
        	
 
optstring=":g:o:r:n:e:t:p:m:b:c:u:"
while getopts ${optstring} arg; do
  case ${arg} in
    g)
      GH=${OPTARG}
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
    t)
      GIT_TITLE=${OPTARG}
      ;;
    e)
      GIT_USEREMAIL=${OPTARG}
      ;;
    p)
      GIT_FILE_PATH=${OPTARG}
      ;;
    m)
      GIT_COMMIT_MESSAGE=${OPTARG}
      ;;
    b)
      GIT_BRANCH=${OPTARG}
      ;;
    c)
      GIT_CONTENT_BODY=${OPTARG}
      ;;
    u)
      GIT_USE_BRANCH=${OPTARG}
      ;;
    *)
      echo "showing usage!"
      usage
      ;;
  esac
done

check_gh
github_login
github_checkout
github_add
github_commit
github_push
github_pr
