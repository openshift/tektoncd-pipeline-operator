## Building Operator Bundle

1. Sync latest master
1. new branch
1. Sync the downstream pipeline repo
1. Checkout release branch  
    ```
    git checkout -b release-v0.9.0
    ```
1. Copy the release yaml from the pipelines repo to operator
   deploy/resources/<openshift-pipelines version>
1. update operator version pkg/flag/flag.go (if the payload version has changed)  
    ```
    make opo-set-pipeline-payload-version PAYLOAD_VERSION=v0.9.2
    ```
1. add latest clustertasks
    ```
    make opo-cluster-tasks CATALOG_VERSION=release-v0.9 PAYLOAD_VERSION=v0.9.2 CATALOG_VERSION_SUFFIX=v0.9.0
    ```
1. add latest triggers, tasksnippets, pipeline samples etc (if there is an updated version available)
1. test the operator using `up local`
    ```
    make opo-test-e2e-up-local
    ```
1. Build operator image and test operator deployment
    build image, push image and update deployment manifest  
    (Todo: move this part to openshift-ci)  
    **make sure the version number is set carefully**  
    **using versions of already published operator will overwrite the published image**
    ```
    make opo-build-push-update VERSION=<n.n.n>
    ```
    make sure the new image has been updated in deploy/operator.yaml `image: `
    test operator deployment
    ```
    make opo-test-e2e
    ```
    
1. make CSV (make sure that the project base directory name is `openshift-pipelines-operator`)

    - `VERSION`: version of current release
    - `FROM_VERSION`: previous CSV version from which CSV metadata should be copied
    - `CHANNEL`: targeted channel
      ```
      make opo-new-csv VERSION=0.9.0 FROM_VERSION=0.8.2 CHANNEL=canary
      ```
  
    You might  need  to edit the package.yaml to remove any duplicate channels. 
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
    and
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
    make opo-push-quay-app VERSION=<n.n.n> TOKEN=$TOKEN QUAY_NAMESPACE=nikhilthomas
    ```
    **NOTE** : special characters in password created issues when courier tried to
    push the app bundle.

1. Ensure that the application in quay is public (Applications>Settings>Make Public)


1. Create an operator source for the app bundle .e.g

```
oc apply -f - <<EOF
apiVersion: operators.coreos.com/v1
kind: OperatorSource
metadata:
  name: <QUAY_USERNAME>-operators
  namespace: openshift-marketplace
spec:
  type: appregistry
  endpoint: https://quay.io/cnr
  registryNamespace: <QUAY_USERNAME>
  displayName: "<QUAY_USERNAME> Operators"
  publisher: "<Your Name>"
EOF
```

see: [Testing deployment on OpenShift](https://github.com/operator-framework/community-operators/blob/master/docs/testing-operators.md#testing-operator-deployment-on-openshift)

Validate operator source by

```
oc get operatorsource <QUAY_USERNAME>-operators -n openshift-marketplace -o yaml
oc get catalogsources <QUAY_USERNAME>-operators -n openshift-marketplace -o yaml

```

Should see "Success: True" or something like that


1. Create a subscription to install operator in `openshift-operators` ns

```
oc apply -f - <<EOF
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: <QUAY_USER>-pipelines-subsription
  namespace: openshift-operators
spec:
  channel: <channel-to-be-tested>
  name: openshift-pipelines-operator
  source: <QUAY_USER>-operators
  sourceNamespace: openshift-marketplace
EOF
```

1. Run scorecard against the generated CSV

```
make opo-test-scorecard VERSION=0.9.0
```

see: [testing with scorecard](https://github.com/operator-framework/community-operators/blob/master/docs/testing-operators.md#testing-with-scorecard)

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
