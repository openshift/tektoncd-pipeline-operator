#!/usr/bin/env bash
#
# Detect which version of pipeline should be installed
# First it tries nightly
# If that doesn't work it tries previous releases (until the MAX_SHIFT variable)
# If not it exit 1
# It can take the argument --only-stable-release to not do nightly but only detect the pipeline version

# set max shift to 0, so that when a version is explicitly specified that version is fetched
# modify this in future if a workflow based on latest version and recent (shifted) versions is needed
MAX_SHIFT=1
NIGHTLY_RELEASE="https://raw.githubusercontent.com/openshift/tektoncd-pipeline/release-next/openshift/release/tektoncd-pipeline-nightly.yaml"
STABLE_RELEASE_URL='https://raw.githubusercontent.com/openshift/tektoncd-pipeline/release-${version}/openshift/release/tektoncd-pipeline-${version}.yaml'
PAYLOAD_PIPELINE_VERSION="release-next"

function get_version {
    local shift=${1} # 0 is latest, increase is the version before etc...
    local version=$(curl -s https://api.github.com/repos/tektoncd/pipeline/releases | python -c "import sys, json;x=json.load(sys.stdin);print(x[${shift}]['tag_name'])")
    PAYLOAD_PIPELINE_VERSION=${version}
    echo $(eval echo ${STABLE_RELEASE_URL})
}

function tryurl {
    curl -s -o /dev/null -f ${1} || return 1
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

[[ -d ${1}/pipelines ]] || mkdir -p ${1}/pipelines
curl -Ls ${URL} -o ${1}/pipelines/release.yaml
