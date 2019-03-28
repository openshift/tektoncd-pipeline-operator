# Tektoncd-operator

## Dev env

### Prerequisites

1. operator-sdk: https://github.com/operator-framework/operator-sdk

2. minikube: https://kubernetes.io/docs/tasks/tools/install-minikube/

### Install Minikube

**create minikube instance**

```
minikube start -p mk-tekton \
 --cpus=4 --memory=8192 --kubernetes-version=v1.12.0 \
 --extra-config=apiserver.enable-admission-plugins="LimitRanger,NamespaceExists,NamespaceLifecycle,ResourceQuota,ServiceAccount,DefaultStorageClass,MutatingAdmissionWebhook"  \
 --extra-config=apiserver.service-node-port-range=80-32767
```

**set docker env**

```
eval $(minikube docker-env -p mk-tekton)
```

### Install OLM

**clone OLM repository (into go path)**

```
git clone git@github.com:operator-framework/operator-lifecycle-manager.git \
          $GOPATH/github.com/operator-framework/
```

```
kubectl apply -f $GOPATH/github.com/operator-framework/operator-lifecycle-manager/deploy/upstream/quickstart
```

### Deploy tekton-operator

#### On minikube for testing

1. Apply operator crd

   `kubectl apply deploy/crds/*_crd.yaml`

1. Apply operator namespace, operatorgroup, serviceaccount, role, and rolebinding

  `kubectl -n tekton-pipelines apply -f deploy/ `

1. apply the `olm-catalog`

  ` kubectl -n tekton-pipelines apply -f deploy/olm-catalog/tektoncd-operator/0.0.1/`

1. once an instance of the cr is created the tekton-operator will launch a pod
  `kubectl apply deploy/crds/*_cr.yaml`

### Deploy pipeline using OLM

1. install minikube [see above](#install-minikube)
1. install olm [see above](#install-olm)
1. Add new catalog source **localOperators**

    `kubectl apply -f https://raw.githubusercontent.com/nikhil-thomas/operator-registry/pipeline-operator/deploy/operator-catalogsource.0.0.1.yaml`

    Once the CatalogSource has been applied, you should find it
    under `Catatog > Operator Management`  of the [web console]

1. Subscribe to `Tektoncd Operator`
    1. Open [web console]
    1. Select [`tekton-pipelines` namespace](http://localhost:9000/status/ns/tekton-pipelines)
    1. Select [`Catalog > Operator Management`](http://localhost:9000/operatormanagement/ns/tekton-pipelines)
    1. Scroll down to `Tektoncd Operator` under `localoperators`

        **NOTE:** it will take few minutes to appear after applying the `catalogsource`

    1. Click `Create Subscription` button
        1. ensure `namespace` in yaml is `tekton-pipelines` e.g.
            <details>
              <summary> sample subscription </summary>

              ```yaml
                apiVersion: operators.coreos.com/v1alpha1
                kind: Subscription
                metadata:
                  generateName: tektoncd-subscription
                  namespace: tekton-pipelines
                spec:
                  source: localoperators
                  sourceNamespace: tekton-pipelines
                  name: tektoncd
                  startingCSV: tektoncd-operator.v0.0.1
                  channel: alpha
              ```
            </details>
        1. Click `Create` button at the bottom

  1. Verify operator is installed successfully
      1. Select `Catalog > Installed operators`
      1. look for `Status` `InstallSucceeded`

1. Install Tektoncd-Pipeline by creating an `install` CR
    1. Select `Catalog > Developer Catalog`, you should find `TektonCD-Pipeline Install`
    1. Click on it and it should show the Operator Details Panel
    1. click on `Create` which show an example as below
          <details>
              <summary> example </summary>
              ```yaml

                apiVersion: tekton.dev/v1alpha1
                kind: Install
                metadata:
                  name: example
                  namespace: tekton-pipelines
                spec: {}

              ```
          </details>
    1. Verify that the pipeline is installed
        1. ensure pipeline pods are running
        1. ensure pipeline crds exist e.g. `kubectl get crds | grep tekton` should show
          ```
          clustertasks.tekton.dev
          installs.tekton.dev
          pipelineresources.tekton.dev
          pipelineruns.tekton.dev
          pipelines.tekton.dev
          taskruns.tekton.dev
          tasks.tekton.dev
          ```

[web console]: http://localhost:9000
