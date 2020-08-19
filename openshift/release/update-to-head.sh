#!/usr/bin/env bash

# Synchs the release-next branch to master and then triggers CI
# Usage: update-to-head.sh

set -e
BRANCH_NAME=release-next
VERSION=release-next

PROJECT_ROOT=$(git rev-parse --show-toplevel)
REPO_NAME=`basename ${PROJECT_ROOT}`
PAYLOAD_ROOT=${PROJECT_ROOT}/deploy/resources

# Reset release-next to openshift/master.
# as there is no upstream repository for this yet.
# after moving openshift-pipelines-operator development to https://github.com/openshift/tektoncd-operator
# the release-next branch should be synced from upstream/master (upstream=tektoncd/operator)
git fetch openshift master
git checkout openshift/master -B ${BRANCH_NAME}

#create payload dir (path where pipeline, triggers, addons/clustertasks are copied)
PAYLOAD_PATH=${PAYLOAD_ROOT}/${VERSION}
[[ -d ${PAYLOAD_PATH} ]] && rm -rf ${PAYLOAD_PATH}
mkdir -p ${PAYLOAD_PATH}

#get pipeline manifest
${PROJECT_ROOT}/openshift/release/fetch-pipeline.sh ${PAYLOAD_PATH}

sed -i 's/^[[:space:]]*TektonVersion.*/TektonVersion = "'${VERSION}'"/' ${PROJECT_ROOT}/pkg/flag/flag.go
go fmt ${PROJECT_ROOT}/pkg/flag/flag.go

# copy rest of the payload from the previous release
# TODO get triggers from nightly or latest release
# TODO run scripts/update-tasks.sh to get cluster tasks
#get triggers manifest
#get cluster task manifest
#get consoleSample
# this is a hack, filter out all versions with single digit minor versions to get the sort working as expected
LATEST_RELEASE=$(ls ${PAYLOAD_ROOT} | sort | grep -v 'v0\.[[:digit:]]\.' | tail -n 1)
for d in $(ls ${PAYLOAD_ROOT}/${LATEST_RELEASE}); do
  echo $d
  if [[ "${d}" = "pipelines" ]]; then
    cp  ${PAYLOAD_ROOT}/${LATEST_RELEASE}/${d}/01-clusterrole.yaml ${PAYLOAD_PATH}/${d}/
    cp  ${PAYLOAD_ROOT}/${LATEST_RELEASE}/${d}/02-rolebinding.yaml ${PAYLOAD_PATH}/${d}/
    continue
  fi
  cp -r ${PAYLOAD_ROOT}/${LATEST_RELEASE}/${d} ${PAYLOAD_PATH}/${d}
done

git add deploy/resources
git add  pkg/flag/flag.go
git commit -m ":Add payload: pipelines,clustertasks,triggers,consolesampleyamls"
git push -f openshift release-next

# Trigger CI
git checkout release-next -B release-next-ci
date > ci
git add ci
git commit -m ":robot: Triggering CI on branch 'release-next' after synching to openshift/master"
git push -f openshift release-next-ci

if hash hub 2>/dev/null; then
   hub pull-request --no-edit -l "kind/sync-fork-to-upstream" -b openshift/${REPO_NAME}:release-next -h openshift/${REPO_NAME}:release-next-ci
else
   echo "hub (https://github.com/github/hub) is not installed, so you'll need to create a PR manually."
fi
