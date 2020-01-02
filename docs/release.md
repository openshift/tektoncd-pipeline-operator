# Operator Release

## Important!

1. Repository name
    make sure that this repository is cloned with the base directory name `openshift-pipelines-operator`  
    to avoid complications with operator-framework tooling
    
    * either clone this [repository](https://github.com/openshift/tektoncd-pipeline-operator) as `$GOPATH/github.com/openshift/openshift-pipelines-operator
    * or create a symbolic link `$GOPATH/github.com/openshift/openshift-pipelines-operator to your clone of this [repository](https://github.com/openshift/tektoncd-pipeline-operator)

1. QUAY_NAMESPACE
    * To make an official release use **QUAY_NAMESPACE=openshift-pipeline**
    * For local development/testing use **QUAY_NAMESPACE=\<your quay username\>**
    
    * whereever **MY_QUAY_NAMESPACE** is specified (in OLM testing) always use **MY_QUAY_NAMESPACE=\<your quay username\>** 

## Branching

1. Sync latest openshift/master and create release branch
    ```
    git fetch openshift master
    git checkout openshift/master -B release-v0.9.2
    ```
## Update Payload (Pipelines, Triggers, ClusterTask etc)
    Update the payload(s) if the release is a release following a release in downstream OpenShift-TektonCD-Pipelines.
    
    If the release is an update in operator (addons, controller ...) make sure the payload(s) have the changes (eg: bugfix in a ClusterTask, ConsileYAMLSample ...)

1. fetch openshift-pipelines release.yaml (skip this step if there is no update in openshift-pipelines payload)
    - PIPELINE_VERSION = version of openshift-pipelines downstream release
    ```
    make opo-payload-pipeline PIPELINE_VERSION=0.9.2
    ```

1. add latest clustertasks (if there is a OpenShift-Pipelines Version bump or there updates in cluster tasks)
    ```
    make opo-cluster-tasks CATALOG_VERSION=release-v0.9 PIPELINE_VERSION=v0.9.2 CATALOG_VERSION_SUFFIX=v0.9.0
    ```
1. add latest triggers, tasksnippets, pipeline samples etc (if there is an updated version available)
    - copy yaml manifests into deploy/resources/<payload version>/<item path>

1. test the operator using `up local`
    ```
    make opo-test-e2e-up-local
    ```

## Building Operator Controller Image

1. Build operator image 
    (Todo: move this part to openshift-ci)  
    **make sure the version number is set carefully**  
    **using versions of already published operator will overwrite the published image**
    
    ```
    make opo-image VERSION=<n.n.n> QUAY_NAMESPACE=<Quay namespace>
    ```

1. Push image to quay
**make sure the version number is set carefully**  
    **using versions of already published operator will overwrite the published image**
    ```
    make opo-image-push VERSION=<n.n.n> QUAY_NAMESPACE=<Quay namespace>
    ```

1. update image reference in deploy/operator.yaml (deployment manifest)
    ```
    make opo-deploy-yaml-update  VERSION=<n.n.n> QUAY_NAMESPACE=<Quay namespace>
    ```
1. and test operator deployment
    make sure the new image has been updated in deploy/operator.yaml `image: `
    test operator deployment
    ```
    make opo-test-e2e
    ```

## Making CSV
     
1. make sure that the project base directory name is `openshift-pipelines-operator`.

    - `VERSION`: version of current release
    - `FROM_VERSION`: previous CSV version from which CSV metadata should be copied
    - `CHANNEL`: targeted channel
      ```
      make opo-new-csv VERSION=0.9.0 FROM_VERSION=0.8.2 CHANNEL=canary
      ```
  
    You might  need  to edit the `deploy/olm-catalog/openshift-pipelines-operator/openshift-pipelines-operator.package.yaml` to remove any duplicate channels. 
    Ensure that the currentCSV and channel names are as expected
    
    e.g.
      ```
      channels:
      - currentCSV: openshift-pipelines-operator.v0.8.2
        name: dev-preview
      - currentCSV: openshift-pipelines-operator.v0.8.2
        name: dev-preview
      ```

    will need to be corrected to:

    ```
    channels:
    - currentCSV: openshift-pipelines-operator.v0.8.2
      name: dev-preview
    ```
    and the end result should look something like this
    ```
    channels:
    - currentCSV: openshift-pipelines-operator.v0.9.0
      name: canary
    - currentCSV: openshift-pipelines-operator.v0.8.2
      name: dev-preview
    defaultChannel: dev-preview
    ```
    (depends on the release plan)
    See existing package in community operators for reference
    
    **Note:-** if we are making a release to move an operator version from canary channel to dev-preview, additional steps are needed. Please go through [CSV: `upgrade-path`, `replaces`, `skips`](https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/design/how-to-update-operators.md#subscribing-to-upgrades) before proceeding.

1. verify operator bundle (deploy/olm-catalog/openshift-pipelines-operator directory)
    ```
    make opo-opr-verify
    ```

## Test Operator Bundle on OLM

1. operator-courier use that to push the app bundle

    NOTE:
    
    You can obtain quay token by running `./scripts/get-quay-token` in
    operator-courier repo. see [Push to quay.io](https://github.com/operator-framework/community-operators/blob/master/docs/testing-operators.md#push-to-quayio)

    ```
    make opo-push-quay-app VERSION=<n.n.n> TOKEN=$TOKEN MY_QUAY_NAMESPACE=<your quay username>
    ```
    **NOTE** : special characters in password created issues when courier tried to
    push the app bundle.

1. Ensure that the application in quay (in Applications Tab) is public (Applications>Settings>Make Public)

1. Create an operator source for the app bundle

    ```
    make opo-operator-source MY_QUAY_NAMESPACE=<your quay username>
    ```

see: [Testing deployment on OpenShift](https://github.com/operator-framework/community-operators/blob/master/docs/testing-operators.md#testing-operator-deployment-on-openshift)

1. Validate operator source by

    ```
    oc get operatorsource <MY_QUAY_NAMESPACE>-operators -n openshift-marketplace -o yaml
    oc get catalogsources <MY_QUAY_NAMESPACE>-operators -n openshift-marketplace -o yaml

    ```

    Should see "Success: True" or something like that


1. Create a subscription to install operator in `openshift-operators` ns

    ```
    make opo-subscription MY_QUAY_NAMESPACE=<your quay username> CHANNEL=<channel to be tested> 
    ```

1. Run scorecard against the generated CSV

    ```
    make opo-test-scorecard VERSION=0.9.0
    ```
    
    (**Note:-** the scorecard test is supposed give a percentage score. The spec for the tests varies. Proceed to next step as long as you get any score)

see: [testing with scorecard](https://github.com/operator-framework/community-operators/blob/master/docs/testing-operators.md#testing-with-scorecard)

## Push Release Branch

Push the release branch (eg: release-v0.9.2) to openshift/tektoncd-pipeline-operator.

- **TODO:** (improvements) additional release-<version>-ci flow to run tests using openshift-ci
- **TODO:** (improvements) move create/add CSV on to release-<version>-ci branch, add ci jobs on openshift-ci,  
add image mirroring. Then publish to OperatorHub after release-<version>-ci merges in release-<version> branch

## Publishing Operator to OperatorHub

1. clone community-operator repository https://github.com/operator-framework/community-operators.git

1. checkout openshift-pipelines-operator-0.9.0 branch

    ```
    git checkout -b openshift-pipelines-operator-0.9.0
    ```

1. copy new CSV files and updated package file co community operators repo
    ```
    cp -r <openshift-pipelines-operator-repo>/deploy/olm-catalog/openshift-pipelines-operator/0.9.0 \
      <community-git-repo>/community-operators/openshift-pipelines-operator/
    ```
    and
    ```
    cp <openshift-pipelines-operator-repo>/deploy/olm-catalog/openshift-pipelines-operator/openshift-pipelines-operator.package.yaml \
      <community-git-repo>/community-operators/openshift-pipelines-operator/
    ```
1. check whether the CSV file, CRD(s) and package file has been added
    ```
            new file:   community-operators/openshift-pipelines-operator/0.9.0/openshift-pipelines-operator.v0.9.0.clusterserviceversion.yaml
            new file:   community-operators/openshift-pipelines-operator/0.9.0/operator_v1alpha1_config_crd.yaml
            modified:   community-operators/openshift-pipelines-operator/openshift-pipelines-operator.package.yaml
    ```

1. Make a commit and submit a PR: e.g: https://github.com/operator-framework/community-operators/pull/756

see [Publishing your operator](https://github.com/operator-framework/community-operators/blob/master/docs/contributing.md#package-your-operator)
