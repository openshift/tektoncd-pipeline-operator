# Local Testing using Operator New Bundle Format

## Build Operator Controller Image

eg:
```shell script
VERSION=0.15.2-1 QUAY_NAMESPACE=openshift-pipeline make opo-ctrl-image
```

## Test Operator

```shell script
make opo-test-e2e
```

## Update CSV file

eg:
```shell script
CHANNELS=canary DEFAULT_CHANNEL=canary ./scripts/update-bundle.sh
```

Then update the following fields in the csv file

1. set `metadata.annotations.containerImage` to <new operator-controller image built above step
2. set `metadata.name: openshift-pipelines-operator.<version of operator>`
3. set `spec.install.spec.deployments.` controller image to the image built in the above step
4. set `spec.version` to `<version of operator>`

note: this step will be improved by handling these edits using a script

## Build Operator Bundle Image

eg:
```shell script
VERSION=0.15.2-1 QUAY_NAMESPACE=openshift-pipeline make opo-bundle-image
```

## Build/Update Index Image

eg:
```shell script
VERSION=0.15.2-1 QUAY_NAMESPACE=openshift-pipeline make opo-index-image
```

## Add CatalogSource to OCP-4.x Cluster

```shell script
oc apply -f samples/catalog-source.yaml
```

## Install Operator

The CatalogSource will be listed in OperatorHub on the OCP-4.x cluster.
Note: Namespace should be `openshift-marketplace` as that is where the CatalogSource gets created.
Note: Clusterwide CatalogSource isn't working as expected. Once that starts working the operator listing should be
visible in AllNamespaces.

Filter by Provider `OpenShift Pipelines`

Follow usual OperatorHub workflow to install the operator.
