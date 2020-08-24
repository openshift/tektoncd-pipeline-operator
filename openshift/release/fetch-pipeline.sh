#!/usr/bin/env bash
#
# Detect which version of pipeline should be installed
# First it tries nightly
# If that doesn't work it tries previous releases (until the MAX_SHIFT variable)
# If not it exit 1
# It can take the argument --only-stable-release to not do nightly but only detect the pipeline version

# set max shift to 0, so that when a version is explicitly specified that version is fetched
# modify this in future if a workflow based on latest version and recent (shifted) versions is needed
set -eu

CURL_OPTIONS="-s" # -s for quiet, -v if you want debug

MAX_SHIFT=1
NIGHTLY_RELEASE="https://raw.githubusercontent.com/openshift/tektoncd-pipeline/release-next/openshift/release/tektoncd-pipeline-nightly.yaml"
STABLE_RELEASE_URL='https://raw.githubusercontent.com/openshift/tektoncd-pipeline/${version}/openshift/release/tektoncd-pipeline-${version}.yaml'
PAYLOAD_PIPELINE_VERSION="release-next"

TMPFILE=$(mktemp /tmp/.mm.XXXXXX)
clean() { rm -f ${TMPFILE}; }
trap clean EXIT

function get_version {
    local shift=${1} # 0 is latest, increase is the version before etc...
    curl -f ${CURL_OPTIONS} -o ${TMPFILE} https://api.github.com/repos/tektoncd/pipeline/releases
    local version=$(python -c "from pkg_resources import parse_version;import json;jeez=json.load(open('${TMPFILE}'));print(sorted([x['tag_name'] for x in jeez], key=parse_version, reverse=True)[${shift}])")
    PAYLOAD_PIPELINE_VERSION=${version}
    echo $(eval echo ${STABLE_RELEASE_URL})
}

function tryurl {
    curl --fail-early ${CURL_OPTIONS} -o /dev/null -f ${1} || return 1
}

function geturl() {

    for (( i = 0; i < 10; i++ )); do
      if tryurl ${NIGHTLY_RELEASE};then
          echo ${NIGHTLY_RELEASE}
          return 0
      fi
      sleep 30s
    done

    for shifted in `seq 0 ${MAX_SHIFT}`;do
        versionyaml=$(get_version ${shifted})
        if tryurl ${versionyaml};then
            echo ${versionyaml}
            return 0
        fi
    done
    echo \n"No working Pipeline payload url found"\n
    exit 1
}

URL=$(geturl)
echo Pipeline Payload URL: ${URL}

# setting this a default so set -u is not failing
arg=${1:-"/tmp"}

[[ -d ${arg}/pipelines ]] || mkdir -p ${arg}/pipelines
curl -Ls ${URL} -o ${arg}/pipelines/00-release.yaml
