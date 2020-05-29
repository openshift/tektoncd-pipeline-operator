# Tektoncd-operator

## FAQ
[click here](/docs/faq.md)

# Configurations

### Override Images:
Following environment variables allows overriding images of pipelines, trigger and 
task addons components

1) `IMAGE_PIPELINES_<DEPLOYMENT-IMAGE-NAME>` e.g. `IMAGE_PIPELINES_WEBHOOK`
It allows overriding pipelines or triggers deployment images. Note, `*_PIPELINE_*` 
will override images in pipelines manifest only. Same way `IMAGE__TRIGGERS_*`
2) `IMAGE_PIPELINES__ARG_<DEPLOYMENT-IMAGE-ARG-NAME>` e.g. `IMAGE_PIPELINES_ARG_NOP`
It allows overriding pipelines or triggers deployment images of containers `args`. 
Note, `*_PIPELINE_ARG_*` will override images in pipelines manifest only. Same way `IMAGE__TRIGGERS_ARG_*`  
3) `IMAGE_ADDONS_<STEP-NAME>` e.g. `IMAGE_ADDONS_PUSH` 
It allows overriding `ClusterTask` addons steps images.
4) `IMAGE_ADDONS_PARAM_<NAME>` e.g. `IMAGE_ADDONS_PARAM_BUILDER` 
It allows overriding `ClusterTask` addons params images. the `_PARAM_` will replace the
value of `Task.Spec.Param`.

`Note: IF IMAGE NAME, IMAGE ARGUMENT, STEP NAME AND PARAMETER NAME HAS "-" IN IT, THEN PLEASE SUBSTITUTE IT BY "_"`

`Note: BE CAUTIOUS WHILE SUBSTITUTING THE IMAGES. FOR INSTANCE, IF DEPLOYMENT HAS MORE THAN ONE CONTAINER AND 
CONTAINER HAS SAME NAME AS DEFINED IN THE OVERRIDE IMAGE CONFIGURATION, THEN THIS COULD RESULT IN UNWATED IMAGE SUBSTITUTATION` 



## Dev env

### Checkout your fork

The Go tools require that you clone the repository to the
`src/github.com/openshift/tektoncd-pipeline-operator` directory in your
[`GOPATH`](https://github.com/golang/go/wiki/SettingGOPATH).

To check out this repository:

1. Create your own
   [fork of this repo](https://help.github.com/articles/fork-a-repo/)
1. Clone it to your machine:

```shell
mkdir -p ${GOPATH}/src/github.com/openshift
cd ${GOPATH}/src/github.com/openshift
git clone git@github.com:${YOUR_GITHUB_USERNAME}/tektoncd-pipeline-operator.git
cd tektoncd-pipeline-operator
git remote add upstream git@github.com:tektoncd/tektoncd-pipeline-operator.git
git remote set-url --push upstream no_push
```

### Prerequisites
You must install these tools:

1. [`go`](https://golang.org/doc/install): The language Tektoncd-pipeline-operator is
   built in
1. [`git`](https://help.github.com/articles/set-up-git/): For source control
1. [`kubectl`](https://kubernetes.io/docs/tasks/tools/install-kubectl/): For
   interacting with your kube cluster
1. operator-sdk: https://github.com/operator-framework/operator-sdk
1. minikube: https://kubernetes.io/docs/tasks/tools/install-minikube/

### Install Minikube

**Create minikube instance**

```
minikube start -p mk-tekton \
 --cpus=4 --memory=8192 --kubernetes-version=v1.12.0 \
 --extra-config=apiserver.enable-admission-plugins="LimitRanger,NamespaceExists,NamespaceLifecycle,ResourceQuota,ServiceAccount,DefaultStorageClass,MutatingAdmissionWebhook"  \
 --extra-config=apiserver.service-node-port-range=80-32767
```

**Set the shell environment up for the container runtime**

```
eval $(minikube docker-env -p mk-tekton)
```

### Development build

1. Change directory to '${GOPATH}/src/github.com/openshift/tektoncd-pipeline-operator'
```
cd ${GOPATH}/src/github.com/openshift/tektoncd-pipeline-operator
```
2. Build go and the container image
```
make osdk-image IMAGE_TAG=${YOUR_REGISTRY}/openshift-pipelines-operator:${YOUR_IMAGE_TAG}
```
3. Push the container image
```
docker push ${YOUR_REGISTRY}/openshift-pipelines-operator:${YOUR_IMAGE_TAG}
```
4. Edit the 'image' value in deploy/operator.yaml to match to your image

#### [Running tests](docs/tests.md)

### Install OLM

**Clone OLM repository (into go path)**

```
git clone git@github.com:operator-framework/operator-lifecycle-manager.git \
          $GOPATH/src/github.com/operator-framework/
```

**Install OLM**

Ensure minikube is installed and docker env is set [see above](#install-minikube)

```
cd $GOPATH/src/github.com/operator-framework/operator-lifecycle-manager
```
```
GO111MODULE=on NO_MINIKUBE=true make run-local
```
**NOTE:** NO_MINIKUBE=true: we don't want to start a new minikube instance while installing OLM

**Launch web console**

Open a new terminal

```
cd $GOPATH/src/github.com/operator-framework/operator-lifecycle-manager
```

```
./scripts/run_console_local.sh
```

### Deploy openshift-pipelines-operator on minikube for testing

1. Change directory to `${GOPATH}/src/github.com/openshift/tektoncd-pipeline-operator`

1. Create `openshift-operators` namespace

   `kubectl create namespace openshift-operators`

1. Apply operator crd

   `kubectl apply -f deploy/crds/*_crd.yaml`

1. Deploy the operator

    `kubectl apply -f deploy/ -n openshift-operators`

1. Install pipeline by creating an `Install` CR

    `kubectl apply -f deploy/crds/*_cr.yaml`

### Deploy openshift-pipelines-operator using CatalogSource on OLM

1. Install minikube [see above](#install-minikube)
1. Install olm [see above](#install-olm)
1. Create `openshift-operators` namespace

   `kubectl create namespace openshift-operators`

2. Generate local catalog source

    `NAMESPACE=operators ./scripts/olm_catalog.sh > olm/openshift-pipelines-operator.resources.yaml`

1. Add local catalog source

    `kubectl apply -f  olm/openshift-pipelines-operator.resources.yaml`

    Once the CatalogSource has been applied, you should find it
    under `Catalog > Operator Management`  of the [web console]

1. Subscribe to `Openshift Pipelines Operator`
    1. Open [web console]
    1. Select [`openshift-operators` namespace](http://0.0.0.0:9000/status/ns/openshift-operators)
    1. Select [`Catalog > Operator Management`](http://0.0.0.0:9000/operatormanagement/ns/openshift-operators)
    1. Select [`Catalog > Operator Management > Operator Catalogs`](http://0.0.0.0:9000/operatormanagement/ns/openshift-operators/catalogsources)
    1. Scroll down to `Openshift Pipelines Operator` under `Openshift Pipelines Operator Registry`

        **NOTE:** it will take a few minutes to appear after applying the `catalogsource`

    1. Click `Create Subscription` button
        1. ensure `namespace` in yaml is `openshift-operator` e.g.
            <details>
              <summary> sample subscription </summary>

              ```yaml
                apiVersion: operators.coreos.com/v1alpha1
                kind: Subscription
                metadata:
                  generateName: openshift-pipelines-operator-
                  namespace: openshift-operators
                spec:
                  source: openshift-pipelines-operator-registry
                  sourceNamespace: openshift-operators
                  name: openshift-pipelines-operator
                  startingCSV: openshift-pipelines-operator.v0.3.1
                  channel: alpha
              ```
            </details>
        1. Click `Create` button at the bottom

  1. Verify operator is installed successfully
      1. Select `Catalog > Installed operators`
      1. Look for `Status` `InstallSucceeded`

1. Install Tektoncd-Pipeline by creating an `install` CR
    1. Select `Catalog > Developer Catalog`, you should find `Openshift Pipelines Install`
    1. Click on it and it should show the Operator Details Panel
    1. Click on `Create` which show an example as below
        <details>
        <summary> example </summary>

        ```yaml
            apiVersion: tekton.dev/v1alpha1
            kind: Install
            metadata:
            name: pipelines-install
            namespace: openshift-pipelines-operator
            spec: {}
        ```
        </details>

        **NOTE:** This will install Openshift Pipeline resources in `Tekton-Pipelines` Namespace
    1. Verify that the pipeline is installed
        1. Ensure pipeline pods are running

           `kubectl get all -n tekton-pipelines`

        1. Ensure pipeline crds exist

           `kubectl get crds | grep tekton`

           should show

           ```shell
           clustertasks.tekton.dev
           installs.tekton.dev
           pipelineresources.tekton.dev
           pipelineruns.tekton.dev
           pipelines.tekton.dev
           taskruns.tekton.dev
           tasks.tekton.dev
           ```
    **NOTE:** Now TektonCD Pipelines can be created and run

### End to End workflow

This section explains how to test changes to the operator by executing the entire end-to-end workflow of edit, test, build, package, etc... 

It assumes you have already followed install [minikube](#install-minikube) and [OLM](#install-olm).

#### Generate new image, CSV

1. Make changes to the operator
1. Test operator locally with `operator-sdk up local`
1. Build operator image `operator-sdk build <imagename:tag>`
1. Update image reference in `deploy/operator.yaml`
1. Update image reference in CSV `deploy/olm-catalog/openshift-pipelines-operator/0.3.1/openshift-pipelines-operator.v0.3.1.clusterserviceversion.yaml`

#### Update Local CatalogSource

1. 1. Build local catalog source **localOperators**

    `NAMESPACE=operators ./scripts/olm_catalog.sh > olm/openshift-pipelines-operator.resources.yaml`

[web console]: http://localhost:9000
