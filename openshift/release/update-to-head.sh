#!/usr/bin/env bash

# Synchs the release-next branch to master and then triggers CI
# Usage: update-to-head.sh

set -e
REPO_NAME=`basename $(git rev-parse --show-toplevel)`

# Reset release-next to openshift/master.
# as there is no upstream repository for this yet.
# after moving openshift-pipelines-operator development to https://github.com/openshift/tektoncd-operator
# the release-next branch should be synced from upstream/master (upstream=tektoncd/operator)
git fetch openshift master
git checkout openshift/master -B release-next
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
