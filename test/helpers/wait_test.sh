#!/bin/sh


function wait_for_CR_until_ready(){
  local timeout="$1"; shift
  local obj="$1"; shift

  echo "Waiting for $obj to be ready; timeout: $timeout"

  local waited=0
  while [[ $waited -lt $timeout ]]; do

    local status=$(kubectl get $obj -o json -n tektoncd | jq -r .status.conditions[0].status)

    case "$status" in
      True) return 0 ;;
      False) return  1 ;;

      *)
        waited=$(( $waited + 2 ))
        echo "   ... [$waited] status is $status "
        sleep 2
        ;;
    esac
  done

  # timeout is an error
  return 1
}

function success() {
  echo "**************************************"
  echo "***        E2E TESTS PASSED        ***"
  echo "**************************************"
  exit 0
}





echo Waiting for resources to be ready
echo ---------------------------------
wait_for_CR_until_ready 600 taskrun/test-template-volume  || exit 1
echo ---------------------------------

success

