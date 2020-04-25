#!/usr/bin/env bash

TEST_NAMESPACE=$1

if [ -z $TEST_NAMESPACE ]; then
  echo TEST_NAMESPACE is not set
  exit 1
fi

operator-sdk test local ./test/e2e \
  --image ${IMAGE_FORMAT//\$\{component\}/tektoncd-pipeline-operator} \
	--namespace ${TEST_NAMESPACE} \
	--go-test-flags "-v -timeout=15m" \
	--debug
