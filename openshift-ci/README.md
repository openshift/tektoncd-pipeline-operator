# TektonCD Pipeline Operator Continuous Integration

TektonCD Pipeline operator uses [OpenShift CI][openshift-ci] for continuous
integration.  The Openshift CI is built using
[CI-Operator][ci-operator].  The TektonCD Pipeline Operator specific
configuration is located here: http://bit.ly/304kpIo

As part of the continuous integration, there are several
jobs configured to run against pull requests in GitHub.  The CI jobs
are triggered whenever there is a new pull request from the team
members.  It is also triggered when there is a new commit in the
current pull request.  If a non-member creates the pull request, there
should be a comment with text as `/ok-to-test` from any member to run
the CI job.

Here is a high-level schematic diagram of the CI pipeline for an
end-to-end test:

```
+--------+     +--------+     +--------+
|        |     |        |     |        |
|  root  +---->+  src   +---->+  bin   |
|        |     |        |     |        |
+--------+     +--------+     +---+----+
                                   |
    ,------------------------------'
    |
    v
+---+----+     +--------+
|        |     |        |
| images +---->+ tests  |
|        |     |        |
+--------+     +--------+
```

For lint and unit test, the schematic diagram is as follows:

```
+--------+     +--------+     +----------------+
|        |     |        |     |                |
|  root  +---->+  src   +---->+ lint/unit test |
|        |     |        |     |                |
+--------+     +--------+     +----------------+
```


All the steps mentioned below are taking place inside a temporary work
namespace.  When you click on the job details from your pull request,
you can see the name of the work namespace in the dashboard UI.  The
name starts with `ci-op-`.  The images created goes into this
temporary work namespace.  At the end of every image build, the [work
namespace name has set as an environment variable][namespace] called
`OPENSHIFT_BUILD_NAMESPACE`.  This environment variable is used to
refer the image URLs inside the configuration files.

## root

As part of the CI pipeline, the first step is to create the `root`
image.  In fact, `root` is a tag created for the pipeline image.  This
image contains all the tools including but not limited to Go compiler,
git, kubectl, oc, and Operator SDK.

The `root` image tag is created using this Dockerfile:
`openshift-ci/Dockerfile.tools`.

## src

This step clones the pull request branch with the latest changes.  The
cloning is taking place inside a container created from the `root`
image.  As a result of this step, an image named `src` is going to be
created.

In the CI configuration, there is a declaration like this:

```
canonical_go_repository: github.com/openshift/tektoncd-pipeline-operator
```

The above line ensures the cloned source code goes into the specified
path: `$GOPATH/src/<canonical_go_repository>`.

Note: If your pull request depends on any package installed through
`yum install` inside the `root` image, those changes should be sent
through a different PR and merged first.  As you can see from the
above diagrams, the `src` image gets built after the `root` image,
whereas the pull request branch gets merged in the `src` image.

## bin

This step runs the `build` Makefile target.  This step is taking place
inside a container created from the `src` image created in the
previous step.

The `make build` produces an operator binary image available under
`./out` directory.  Later, this binary is copied to
`tektoncd-pipeline-operator` (see below).

As a result of this step, a container image named `bin` is going to be
created.

## images

Three container images are built as part of this job.  Before this
step, a couple of base images are getting tagged from existing
published images.  The first one is a CentOS 7 image referred to as
`os` in the CI configuration.  The other one is the operator registry
image which contains all the binaries to run a gRPC based registry.
The operator image registry is referred to as `operator-registry` in
the CI configuration.

### tektoncd-pipeline-operator

Thee CentOS 7 image (`os`) is going to use as the base image for
creating `tektoncd-pipeline-operator` image.  The operator binary available
inside `bin` container image is copied over here.  The Dockerfile used
is available here: `openshift-ci/Dockerfile.deploy`

The image produced can be pulled from here:
`registry.svc.ci.openshift.org/${OPENSHIFT_BUILD_NAMESPACE}/stable:tektoncd-pipeline-operator`

### operator-registry-base

The `operator-registry` is going to use as the base image for creating
`operator-registry-base` image.  It is an intermediate image used to
propagate the value of `OPENSHIFT_BUILD_NAMESPACE` environment
variable to `tektoncd-pipeline-operator-registry` image build.  This
intermediate image is going to use as the base image for
`tektoncd-pipeline-operator-registry`.  When the original "operator-registry"
image was getting created, a different value for
`OPENSHIFT_BUILD_NAMESPACE` environment variable has set.

The Dockerfile used is available here:
`openshift-ci/Dockerfile.registry.intermediate` As you can see, the
Dockerfile file has only one line with a `FROM` statement, which is
going to be replaced with `operator-registry` image during image
build.

### tektoncd-pipeline-operator-registry

The `tektoncd-pipeline-operator-registry` image uses `operator-registry-base`
as the base image.

The Dockerfile used is available here:
`openshift-ci/Dockerfile.registry.build`

The image produced can be pulled from here:
`registry.svc.ci.openshift.org/${OPENSHIFT_BUILD_NAMESPACE}/stable:tektoncd-pipeline-operator-registry`

## tests

### lint

The lint runs the [GolangCI Lint][golangci], [YAML Lint][yaml-lint],
and [Operator Courier][operator-courier].  GolangCI is a Go program,
whereas the other two are Python based.  So, Python 3 is a
prerequisite to run lint.

The GolangCI Lint program runs multiple Go lint tools against the
repository.  GolangCI Lint runs lint concurrently and completes
execution in a few seconds.  However, there is one caveat; it requires
lots of memory.  The memory limit has been increased to 6GB to
accommodate the requirement.  As of now, there is no configuration
provided to run GolangCI Lint.

The YAML Lint tools validate all the YAML configuration files.  It
excludes the `vendor` directory while running.  There is a
configuration file at the top-level directory of the source code:
`.yamllint`.

The Operator Courier checks for the validity of Operator Lifecycle
Manager (OLM) manifests.  That includes Cluster Service Version (CSV)
files and CRDs.

### test

The `test` target runs the unit tests.  Some of the tests make use of
mock objects. The unit tests don't require a dedicated OpenShift
cluster, unlike end-to-end tests.

### e2e

The `e2e` run an end-to-end test against an operator running inside
the CI cluster pod but connected to a freshly created temporary
Openshift 4 cluster.  It makes use of the `--up-local` option provided
by the Operator SDK testing tool.  It runs `test-e2e` Makefile target.

### e2e-ci

The `e2e-ci` target runs the end-to-end test against an operator
running inside the freshly created OpenShift 4 cluster.  All the
resources are getting created through scripts invoked from
`test-e2e-ci` Makefile target.  This Makefile target is designed to
run exclusively on the CI environment.

### e2e-olm-ci

The e2e-ci runs the end-to-end test against operator running inside
the freshly created OpenShift 4 cluster.  All the resources are
getting created through Operator Lifecycle Manager (OLM).  This
Makefile target is designed to run exclusively on the CI environment.

**Note:** The difference between `e2e-ci` and `e2e-olm-ci` targets is
how the Kubernetes resources getting created. The resources include
operator deployment, service account, role, and role binding. The
`e2e-ci` target uses a Makefile script (few `oc apply` commands) to
create the resources whereas `e2e-olm-ci` uses Operator Lifecycle
Manager (OLM) to create resources.

[openshift-ci]: https://github.com/openshift/release
[ci-operator]: https://github.com/openshift/release/tree/master/ci-operator
[golangci]: https://github.com/golangci/golangci-lint
[yaml-lint]: https://github.com/adrienverge/yamllint
[operator-courier]: https://github.com/operator-framework/operator-courier
[namespace]: https://docs.okd.io/latest/dev_guide/builds/build_output.html#output-image-environment-variables
