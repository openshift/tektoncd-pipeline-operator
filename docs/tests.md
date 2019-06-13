# Running E2E Tests

1. Ensure **operator-sdk** **v0.8.0** is installed.

2. If the namespace **openshift-pipelines-operator** exists, delete it. Do this to make sure the namespace is clean.

```
oc delete namespace openshift-pipelines-operator
```

3. Create the namespace **openshift-pipelines-operator**.

```
oc create namespace openshift-pipelines-operator
```

4. Run the test using `operator-sdk test`  command locally (without buid).

```
operator-sdk test local --up-local ./test/e2e \
  --namespace openshift-pipelines-operator \
  --verbose --debug
```

## Reference

* [Running tests](https://github.com/operator-framework/operator-sdk/blob/master/doc/test-framework/writing-e2e-tests.md#running-the-tests)
* [Installing Operator-SDK](https://github.com/operator-framework/operator-sdk#quick-start)
