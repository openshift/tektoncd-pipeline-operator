#!/bin/sh

cd test/resources
kubectl apply -f addon1.yaml
cat ../../deploy/service_account.yaml divider.txt ../../deploy/role.yaml divider.txt ../../deploy/role_binding.yaml divider.txt ../../deploy/operator.yaml mount.txt > namespaced.yaml
cd -
operator-sdk test local ./test/e2e --namespace default --namespaced-manifest test/resources/namespaced.yaml  --global-manifest test/resources/operator_v1alpha1_config_cr.yaml  --image $1
cd test/resources
rm namespaced.yaml
kubectl delete -f addon1.yaml
cd -
