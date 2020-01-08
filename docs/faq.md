# FAQ

### 1. OpenShift-Pipelines version and OpenShift-Pipelines-Operator version

OpenShift-Pipelines and OpenShift-Pipelines-Operator are versioned
separately. The OpenShift-Pipelines-Operator is published through OperatorHub on OpenShift 4.X.
OpenShift-Pipelines is the primary payload of the operator.

At present (jan-07-2020) we provide two subscription channels for this operator.

- dev-preview: latest stable release
- canary: the next release candidate

### 2. Operator Description page on OperatorHub show v0.9.2, Installed Operator version is shown as v0.8.2, and the payload version (or the image tag of openshift-pipeline) is v0.8.0. What does this mean?

1. The latest version available for this operator is v0.9.2. This 
implies latest version available through canary channel is v0.9.2
 (the stable version available on dev-preview may be an older version)

1. If the installed operator version is v0.8.2 (a version older than the
version shown in the operator description), it indicates that the operator 
was installed (subscribed) through dev-preview channel and the latest stable 
version available on the dev-preview channel is v0.8.2

1. Payload (openshift-pipeline) version is shown as v0.8.0: This means the 
OpenShift-Pipelines-Operator (v0.8.2) installed the payload OpenShift-Pipelines v0.8.0.
In other words the primary payload OpenShift-Pipelines v0.8.0 is shipped with 
OpenShift-Pipelines-Operator v0.8.2

### 3. After installing OpenShift-Pipelines-Operator TaskRuns, PipelineRuns created are never executed. And pipeline controller looks like the process just hung and no logs are printed.

If your cluster is on GCP, the reason might be this issue: [OpenShift + Pipeline on GCP is broken](https://github.com/tektoncd/pipeline/issues/1742)

else contact us on: 
- downstream: [https://coreos.slack.com](https://coreos.slack.com), #tektoncd-pipeline
- upstream: [https://tektoncd.slack.com](https://tektoncd.slack.com), #openshift-pipelines, #pipeline-dev, #operator, #cli ...

### 4. Resource are not being deleted after uninstalling the OpenShift-Pipelines-Operator.

Make sure that you have done a clean Uninstall of OpenShift-Pipelines Operator.

We have to delete the instance of the CRD `config.operator.tekton.dev` (resource-name: cluster) before
uninstalling the operator. This necessary because the resource is auto-created during operator install,
but not auto-uninstalled during operator uninstall. This is expected Operator-Lifecycle-Manager (OLM) behavior (at present).

Please follow the steps from this article: [Right way to Uninstall OpenShift-Pipelines fromOpenShift 4.x](https://medium.com/@nikhilthomas1/right-way-to-uninstall-openshift-pipelines-fromopenshift-4-x-fb2a7b7c492c)
