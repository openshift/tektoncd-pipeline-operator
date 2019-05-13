# Running e2e tests

make sure **operator-sdk** version: **v0.7.0**
is installed

create Namespace **dev-openshift-pipelines-operator**

then run
```
$ operator-sdk test local ./test/e2e --namespace dev-openshift-pipelines-operator --debug
```

## Reference

### Running tests: https://github.com/operator-framework/operator-sdk/blob/master/doc/test-framework/writing-e2e-tests.md#running-the-tests

### Installing Operator-SDK: https://github.com/operator-framework/operator-sdk#quick-start
