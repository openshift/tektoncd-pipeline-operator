#!/bin/sh

cd test/resources
kubectl apply -f addon1.yaml
cat ../../deploy/service_account.yaml divider.txt ../../deploy/role.yaml divider.txt ../../deploy/role_binding.yaml divider.txt ../../deploy/operator.yaml mount.txt > namespaced.yaml
cd -
operator-sdk test local ./test/e2e --namespace openshift-pipelines-operator --namespaced-manifest test/resources/namespaced.yaml --image $1
cd test/resources
rm namespaced.yaml
kubectl delete -f addon1.yaml
cd -
