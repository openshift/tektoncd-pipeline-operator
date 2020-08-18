#!/usr/bin/env bash

set -eu

cp -r bundle bundle.bk
opm alpha bundle generate \
    --channels ${CHANNELS} \
    --default ${DEFAULT_CHANNEL} \
    --directory bundle.bk/manifests \
    --output-dir ./bundle \
    --package openshift-pipelines-operator-midstr

rm -rf bundle.bk