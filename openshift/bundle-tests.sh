#!/usr/bin/env bash

# usage:
#    create a breakpoint by adding
#    ```
#    breakPoint <breakPointName>
#    ```
#
#    to resume (run in pod `e2e`, container `test`)
#    ```
#    touch <breakPointName>
#    ```
function breakPoint() {
  waitFileName=${1:-waitFile}
  while [[ ! -f ${waitFileName} ]]; do
    sleep 10;
    echo \*\* --------------------------------------- \*\*
    echo \*\* breakPoint                              \*\*;
    echo \*\* run \`touch ${waitFileName}\` to resume \*\*
  done
}

breakPoint bundle_tests_start
echo hello
breakPoint bundle_tests_mid
echo world
breakPoint bundle_tests_end
echo '!!!!'
