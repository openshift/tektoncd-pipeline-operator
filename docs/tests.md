# Running E2E Tests

1. Ensure **operator-sdk** version: **v0.8.0** is installed

1. if namespace **openshift-pipelines-operator** exists, delete it (to make sure the namespace is clean)

```
oc delete namespace openshift-pipelines-operator
```

1. create namespace **openshift-pipelines-operator**

```
oc create namespace openshift-pipelines-operator
```

1. run the test using `operator-sdk test`  command locally (without buid)

```
operator-sdk test local --up-local ./test/e2e \
  --namespace openshift-pipelines-operator \
  --verbose --debug
```

## Reference

* [Running tests](https://github.com/operator-framework/operator-sdk/blob/master/doc/test-framework/writing-e2e-tests.md#running-the-tests)
* [Installing Operator-SDK](https://github.com/operator-framework/operator-sdk#quick-start)
