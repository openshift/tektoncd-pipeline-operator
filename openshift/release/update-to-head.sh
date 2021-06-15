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

# get pipeline manifest
${PROJECT_ROOT}/openshift/release/fetch-pipeline.sh ${PAYLOAD_ROOT}

# update flag value to release-next
sed -i 's/^[[:space:]]*TektonVersion.*/TektonVersion = "'${VERSION}'"/' ${PROJECT_ROOT}/pkg/flag/flag.go
go fmt ${PROJECT_ROOT}/pkg/flag/flag.go

# commit changes to release-next
git add deploy/resources
git add pkg/flag/flag.go
git commit -m ":Add nightly or last release pipeline manifest"
git push -f openshift release-next

# commit changes to release-next-ci
git checkout release-next -B release-next-ci
date > ci
git add ci
git commit -m ":robot: Triggering CI on branch 'release-next' after synching to openshift/master"
git push -f openshift release-next-ci

# Trigger CI by raising a PR
if hash hub 2>/dev/null; then
   # Test if there is already a sync PR in 
   COUNT=$(hub api -H "Accept: application/vnd.github.v3+json" repos/openshift/${REPO_NAME}/pulls --flat \
    | grep -c ":robot: Triggering CI on branch 'release-next' after synching to upstream/[master|main]") || true
   if [ "$COUNT" = "0" ]; then
      hub pull-request --no-edit -l "kind/sync-fork-to-upstream" -b openshift/${REPO_NAME}:release-next -h openshift/${REPO_NAME}:release-next-ci
   fi
else
   echo "hub (https://github.com/github/hub) is not installed, so you'll need to create a PR manually."
fi
