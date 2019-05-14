# Running E2E Tests

1. Ensure **operator-sdk** version: **v0.7.0** is installed

1. create namespace **openshift-pipelines-operator**
```
oc create namespace openshift-pipelines-operator
```

1. run the test using `operator-sdk test`  command
```
operator-sdk test local ./test/e2e \
  --namespace openshift-pipelines-operator \
  --debug
```

## Reference

* [Running tests](https://github.com/operator-framework/operator-sdk/blob/master/doc/test-framework/writing-e2e-tests.md#running-the-tests)
* [Installing Operator-SDK](https://github.com/operator-framework/operator-sdk#quick-start)
