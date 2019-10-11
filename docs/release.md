# steps

1. Sync latest master
1. new branch
1. Sync the downstream pipeline repo
1. Checkout release branch
1. Copy the release yaml from the pipelines repo to operator
   deploy/resources/<version>
1. modify operator config_controller.go to update the <version>
1. test the operator using `up local`
1. build image

  ```
  operator-sdk build quay.io/openshift-pipeline/openshift-pipelines-operator:v0.7.0

  ```

1. update `deploy/operator.yaml` image:

1. generate csv

  ```
  operator-sdk olm-catalog gen-csv \
    --csv-channel dev-preview \
    --csv-version 0.7.0 \
    --from-version 0.5.2 \
    --operator-name  openshift-pipelines-operator \
    --update-crds
  ```

You might  need  to edit the package.yaml to remove any duplicate channels

e.g.
  ```
  channels:
  - currentCSV: openshift-pipelines-operator.v0.5.0
    name: dev-preview
  - currentCSV: openshift-pipelines-operator.v0.7.0
    name: dev-preview
  ```

will need to be corrected to:

```
channels:
- currentCSV: openshift-pipelines-operator.v0.7.0
  name: dev-preview
```

See existing pacakge in community operators for reference


1. flatten  to generate bundle
```
operator-courier flatten \
  deploy/olm-catalog/openshift-pipelines-operator \
  tmp/openshift-pipelines-operator
```
1. verify that
```
operator-courier verify --ui_validate_io \
  tmp/openshift-pipelines-operator
```

1. operator-courier use that to push the app bundle

NOTE:

You can obtain quay token by running `./scripts/get-quay-token` in
operator-courier repo. see [Push to quay.io](https://github.com/operator-framework/community-operators/blob/master/docs/testing-operators.md#push-to-quayio)

```
export OPERATOR_DIR=tmp/openshift-pipelines-operator
export QUAY_NAMESPACE=<QUAY_USER>
export PACKAGE_NAME=openshift-pipelines-operator
export PACKAGE_VERSION=0.7.0
export TOKEN="basic <your-token here>"
```

```
 operator-courier --verbose push  \
    $OPERATOR_DIR $QUAY_NAMESPACE \
    $PACKAGE_NAME $PACKAGE_VERSION  \
    "$TOKEN"
```
**NOTE** : special characters in password created issues when courier tried to
push the app bundle.

1. Ensure that the application in quay is public


1. Create an operator source for the app bundle .e.g

```
apiVersion: operators.coreos.com/v1
kind: OperatorSource
metadata:
  name: <QUAY_USER>-operators
  namespace: openshift-marketplace
spec:
  type: appregistry
  endpoint: https://quay.io/cnr
  registryNamespace: <QUAY_USER>
  displayName: "<QUAY_USER> Operators"
  publisher: "<Your Name>"
```
see: [Testing deployment on OpenShift](https://github.com/operator-framework/community-operators/blob/master/docs/testing-operators.md#testing-operator-deployment-on-openshift)

Validate operator source by
```
oc get operatorsource <QUAY_USER>-operators -n openshift-marketplace -o yaml
oc get catalogsources <QUAY_USER>-operators -n openshift-marketplace -o yaml

```

Should see "Success: True" or something like that


1. Create a subscription to install operator in `openshift-operators` ns
```

apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: <QUAY_USER>-pipelines-subsription
  namespace: openshift-operators
spec:
  channel: dev-preview
  name: openshift-pipelines-operator
  source: <QUAY_USER>-operators
  sourceNamespace: openshift-marketplace

```

1. Run scorecard against the generated CSV

```
operator-sdk scorecard \
  --olm-deployed \
  --csv-path deploy/olm-catalog/openshift-pipelines-operator/0.7.0/openshift-pipelines-operator.v0.7.0.clusterserviceversion.yaml \
  --namespace openshift-operators \
  --cr-manifest ./deploy/crds/operator_v1alpha1_config_cr.yaml \
  --crds-dir ./deploy/crds/

```

see: [testing with scorecard](https://github.com/operator-framework/community-operators/blob/master/docs/testing-operators.md#testing-with-scorecard)

1. Publish to community operators
```
cp tmp/openshift-pipelines-operator/* \
  <community-git-repo>/community-operators/openshift-pipelines-operator
```

1. Submit a PR: e.g: https://github.com/operator-framework/community-operators/pull/756

see [Publishing your operator](https://github.com/operator-framework/community-operators/blob/master/docs/contributing.md#package-your-operator)
