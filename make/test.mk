ifndef TEST_MK
TEST_MK:=# Prevent repeated "-include".
UNAME_S := $(shell uname -s)

include ./make/verbose.mk
include ./make/out.mk

# Quay App Registry
DEVCONSOLE_APPR_NAMESPACE ?= odcqe
DEVCONSOLE_APPR_REPOSITORY ?= devconsole

export DEPLOYED_NAMESPACE:=

.PHONY: test
## Runs Go package tests and stops when the first one fails
test: ./vendor
	$(Q)go test -vet off ${V_FLAG} $(shell go list ./... | grep -v -E '(/test/e2e|/test/operatorsource)') -failfast

.PHONY: test-coverage
## Runs Go package tests and produces coverage information
test-coverage: ./out/cover.out

.PHONY: test-coverage-html
## Gather (if needed) coverage information and show it in your browser
test-coverage-html: ./vendor ./out/cover.out
	$(Q)go tool cover -html=./out/cover.out

./out/cover.out: ./vendor
	$(Q)go test ${V_FLAG} -race $(shell go list ./... | grep -v -E '(/test/e2e|/test/operatorsource)') -failfast -coverprofile=cover.out -covermode=atomic -outputdir=./out

.PHONY: get-test-namespace
get-test-namespace: ./out/test-namespace
	$(eval TEST_NAMESPACE := $(shell cat ./out/test-namespace))

.PHONY: get-operator-version
get-operator-version:
	$(eval package_yaml := ./manifests/devconsole/devconsole.package.yaml)
	$(eval DEVCONSOLE_OPERATOR_VERSION := $(shell cat $(package_yaml) | grep "currentCSV"| cut -d "." -f2- | cut -d "v" -f2 | tr -d '[:space:]'))

./out/test-namespace:
	@echo -n "test-namespace-$(shell uuidgen | tr '[:upper:]' '[:lower:]')" > ./out/test-namespace

.PHONY: test-e2e
## Runs the e2e tests locally
test-e2e: ./vendor e2e-setup
	$(info Running E2E test: $@)
ifeq ($(OPENSHIFT_VERSION),3)
	$(Q)oc login -u system:admin
endif
	$(Q)operator-sdk test local ./test/e2e --namespace $(TEST_NAMESPACE) --up-local --go-test-flags "-v -timeout=15m"


.PHONY: e2e-setup
e2e-setup: e2e-cleanup 
	$(Q)oc new-project $(TEST_NAMESPACE)

.PHONY: e2e-cleanup
e2e-cleanup: get-test-namespace
	$(Q)-oc delete project $(TEST_NAMESPACE) --timeout=10s --wait

.PHONY: test-olm-integration
## Runs the OLM integration tests without coverage
test-olm-integration: push-operator-image olm-integration-setup get-operator-version
	$(call log-info,"Running OLM integration test: $@")
ifeq ($(OPENSHIFT_VERSION),3)
	$(eval DEPLOYED_NAMESPACE := operators)
	$(Q)oc apply -f https://raw.githubusercontent.com/operator-framework/operator-lifecycle-manager/master/deploy/upstream/quickstart/olm.yaml
endif
	$(Q)docker build -f Dockerfile.registry . -t $(DEVCONSOLE_OPERATOR_REGISTRY_IMAGE):$(DEVCONSOLE_OPERATOR_VERSION)-$(TAG) \
		--build-arg image=$(DEVCONSOLE_OPERATOR_IMAGE):$(TAG) --build-arg version=$(DEVCONSOLE_OPERATOR_VERSION)
	@docker login -u $(QUAY_USERNAME) -p $(QUAY_PASSWORD) $(REGISTRY_URI)
	$(Q)docker push $(DEVCONSOLE_OPERATOR_REGISTRY_IMAGE):$(DEVCONSOLE_OPERATOR_VERSION)-$(TAG)
ifeq ($(OPENSHIFT_VERSION),3)
	$(Q)sed -e "s,REPLACE_IMAGE,$(DEVCONSOLE_OPERATOR_REGISTRY_IMAGE):$(DEVCONSOLE_OPERATOR_VERSION)-$(TAG)," ./test/e2e/catalog_source_OS3.yaml | oc apply -f -
	$(Q)oc apply -f ./test/e2e/subscription_OS3.yaml
endif
ifeq ($(OPENSHIFT_VERSION),4)
	$(eval DEPLOYED_NAMESPACE := openshift-operators)
	$(Q)sed -e "s,REPLACE_IMAGE,$(DEVCONSOLE_OPERATOR_REGISTRY_IMAGE):$(DEVCONSOLE_OPERATOR_VERSION)-$(TAG)," ./test/e2e/catalog_source_OS4.yaml | oc apply -f -
	$(Q)oc apply -f ./test/e2e/subscription_OS4.yaml
endif
	$(Q)operator-sdk test local ./test/e2e/ --no-setup --go-test-flags "-v -timeout=15m"

.PHONY: olm-integration-setup
olm-integration-setup: olm-integration-cleanup
	$(Q)oc new-project $(TEST_NAMESPACE)

.PHONY: push-operator-app-registry
ifdef DO_NOT_PUSH_OPERATOR_IMAGE
push-operator-app-registry: get-operator-version 
else
push-operator-app-registry: get-operator-version push-operator-image
endif
	$(eval OPERATOR_MANIFESTS := tmp/manifests/$(TAG))
	$(Q)operator-courier flatten manifests/devconsole/ $(OPERATOR_MANIFESTS)
	$(Q)cp -vf deploy/crds/* $(OPERATOR_MANIFESTS)
	$(Q)sed -i -e 's,REPLACE_IMAGE,$(DEVCONSOLE_OPERATOR_IMAGE):$(TAG),' $(OPERATOR_MANIFESTS)/tektoncd-pipeline-operator.v$(DEVCONSOLE_OPERATOR_VERSION).clusterserviceversion-v$(DEVCONSOLE_OPERATOR_VERSION).yaml
	$(Q)operator-courier verify $(OPERATOR_MANIFESTS)
	$(eval QUAY_API_TOKEN := $(shell curl -sH "Content-Type: application/json" -XPOST https://quay.io/cnr/api/v1/users/login -d '{"user":{"username":"'${QUAY_USERNAME}'","password":"'${QUAY_PASSWORD}'"}}' | jq -r '.token'))
	$(Q)operator-courier push $(OPERATOR_MANIFESTS) $(DEVCONSOLE_APPR_NAMESPACE) $(DEVCONSOLE_APPR_REPOSITORY) $(DEVCONSOLE_OPERATOR_VERSION)-$(TAG) "$(QUAY_API_TOKEN)"

.PHONY: test-operator-source
test-operator-source: push-operator-app-registry
	$(eval OPSRC_NAME := tektoncd-pipeline-operators-$(TAG))
	$(eval OPSRC_DIR := test/operatorsource)
	$(Q)oc project openshift-marketplace 
	$(Q)sed -e "s,REPLACE_NAMESPACE,$(DEVCONSOLE_APPR_NAMESPACE)," ./$(OPSRC_DIR)/operatorsource.yaml | sed -e "s,REPLACE_OPERATOR_SOURCE_NAME,$(OPSRC_NAME)," | oc apply -f -
	$(Q)sed -e "s,REPLACE_APPR_REPOSITORY,$(DEVCONSOLE_APPR_REPOSITORY)," ./$(OPSRC_DIR)/catalogsourceconfig.yaml | oc apply -f -
	$(Q)sed -e "s,REPLACE_APPR_REPOSITORY,$(DEVCONSOLE_APPR_REPOSITORY)," ./$(OPSRC_DIR)/subscription.yaml | oc apply -f -
	$(Q)./hack/check-crds.sh
	$(Q)OPSRC_NAME=$(OPSRC_NAME) \
	DEVCONSOLE_OPERATOR_VERSION=$(DEVCONSOLE_OPERATOR_VERSION) \
	go test -vet off ${V_FLAG} $(shell go list ./... | grep $(OPSRC_DIR)) -failfast

.PHONY: olm-integration-cleanup
olm-integration-cleanup: get-test-namespace
ifeq ($(OPENSHIFT_VERSION),3)
	$(Q)oc login -u system:admin
	$(Q)-oc delete subscription my-devconsole -n operators
	$(Q)-oc delete catalogsource my-catalog -n olm
endif
ifeq ($(OPENSHIFT_VERSION),4)
	$(Q)-oc delete subscription my-devconsole -n openshift-operators
	$(Q)-oc delete catalogsource my-catalog -n openshift-operator-lifecycle-manager
endif
	$(Q)-oc delete project $(TEST_NAMESPACE)  --wait

.PHONY: test-e2e-olm-ci
test-e2e-olm-ci: ./vendor
	$(Q)sed -e "s,REPLACE_IMAGE,registry.svc.ci.openshift.org/${OPENSHIFT_BUILD_NAMESPACE}/stable:tektoncd-pipeline-operator-registry," ./test/e2e/catalog_source_OS4.yaml | oc apply -f -
	$(Q)oc apply -f ./test/e2e/subscription_OS4.yaml
	$(eval DEPLOYED_NAMESPACE := openshift-operators)
	$(Q)./hack/check-crds.sh
	$(Q)operator-sdk test local ./test/e2e --no-setup --go-test-flags "-v -timeout=15m"

.PHONY: test-e2e-ci
test-e2e-ci: get-test-namespace ./vendor
	$(Q)oc new-project $(TEST_NAMESPACE)
	$(Q)-oc apply -f ./deploy/crds/devconsole_v1alpha1_component_crd.yaml
	$(Q)-oc apply -f ./deploy/crds/devconsole_v1alpha1_gitsource_crd.yaml
	$(Q)-oc apply -f ./deploy/crds/devconsole_v1alpha1_gitsourceanalysis_crd.yaml
	$(Q)-oc apply -f ./deploy/service_account.yaml --namespace $(TEST_NAMESPACE)
	$(Q)-oc apply -f ./deploy/role.yaml --namespace $(TEST_NAMESPACE)
	$(Q)sed -e 's|REPLACE_NAMESPACE|$(TEST_NAMESPACE)|g' ./deploy/test/role_binding_test.yaml | oc apply -f -
	$(Q)sed -e 's|REPLACE_IMAGE|registry.svc.ci.openshift.org/${OPENSHIFT_BUILD_NAMESPACE}/stable:tektoncd-pipeline-operator|g' ./deploy/test/operator_test.yaml  | oc apply -f - --namespace $(TEST_NAMESPACE)
	$(eval DEPLOYED_NAMESPACE := $(TEST_NAMESPACE))
	$(Q)operator-sdk test local ./test/e2e --namespace $(TEST_NAMESPACE) --no-setup --go-test-flags "-v -timeout=15m"

#-------------------------------------------------------------------------------
# e2e test in dev mode
#-------------------------------------------------------------------------------

.PHONY: e2e-cleanup-local
## Create a namespace used in e2e tests
e2e-cleanup-local: get-test-namespace
	$(Q)-oc login -u system:admin
	$(Q)-oc delete -f ./deploy/crds/devconsole_v1alpha1_component_crd.yaml
	$(Q)-oc delete -f ./deploy/service_account.yaml --namespace $(TEST_NAMESPACE)
	$(Q)-oc delete -f ./deploy/role.yaml --namespace $(TEST_NAMESPACE)
	$(Q)-oc delete -f ./deploy/test/role_binding_test.yaml --namespace $(TEST_NAMESPACE)
	$(Q)-oc delete -f ./deploy/test/operator_test.yaml --namespace $(TEST_NAMESPACE)

.PHONY: e2e-setup-local
## Create a namespace used in e2e tests
e2e-setup-local: e2e-cleanup-local
	$(Q)-oc new-project $(TEST_NAMESPACE)

.PHONY: build-image-local
build-image-local: e2e-setup-local
	eval $$(minishift docker-env) && operator-sdk build $(shell minishift openshift registry)/$(TEST_NAMESPACE)/tektoncd-pipeline-operator

.PHONY: test-e2e-local
test-e2e-local: build-image-local
	$(eval DEPLOYED_NAMESPACE := $(TEST_NAMESPACE))
	$(Q)-oc login -u system:admin
	$(Q)-oc project $(TEST_NAMESPACE)
	$(Q)-oc create -f ./deploy/crds/devconsole_v1alpha1_component_crd.yaml
	$(Q)-oc create -f ./deploy/crds/devconsole_v1alpha1_gitsource_crd.yaml
	$(Q)-oc create -f ./deploy/service_account.yaml --namespace $(TEST_NAMESPACE)
	$(Q)-oc create -f ./deploy/role.yaml --namespace $(TEST_NAMESPACE)
ifeq ($(UNAME_S),Darwin	)
	$(Q)sed -i "" 's|REPLACE_NAMESPACE|$(TEST_NAMESPACE)|g' ./deploy/test/role_binding_test.yaml
else
	$(Q)sed -i 's|REPLACE_NAMESPACE|$(TEST_NAMESPACE)|g' ./deploy/test/role_binding_test.yaml
endif
	@-oc create -f ./deploy/test/role_binding_test.yaml --namespace $(TEST_NAMESPACE)
ifeq ($(UNAME_S),Darwin)
	$(Q)sed -i "" 's|REPLACE_IMAGE|172.30.1.1:5000/$(TEST_NAMESPACE)/tektoncd-pipeline-operator:latest|g' ./deploy/test/operator_test.yaml
else
	$(Q)sed -i 's|REPLACE_IMAGE|172.30.1.1:5000/$(TEST_NAMESPACE)/tektoncd-pipeline-operator:latest|g' ./deploy/test/operator_test.yaml
endif
	@eval $$(minishift docker-env) && oc create -f ./deploy/test/operator_test.yaml --namespace $(TEST_NAMESPACE)
ifeq ($(UNAME_S),Darwin)
	$(Q)sed -i "" 's|$(TEST_NAMESPACE)|REPLACE_NAMESPACE|g' ./deploy/test/role_binding_test.yaml
	$(Q)sed -i "" 's|172.30.1.1:5000/$(TEST_NAMESPACE)/tektoncd-pipeline-operator:latest|REPLACE_IMAGE|g' ./deploy/test/operator_test.yaml
else
	$(Q)sed -i 's|$(TEST_NAMESPACE)|REPLACE_NAMESPACE|g' ./deploy/test/role_binding_test.yaml
	$(Q)sed -i 's|172.30.1.1:5000/$(TEST_NAMESPACE)/tektoncd-pipeline-operator:latest|REPLACE_IMAGE|g' ./deploy/test/operator_test.yaml
endif
	$(Q)eval $$(minishift docker-env) && operator-sdk test local ./test/e2e --namespace $(TEST_NAMESPACE) --no-setup --go-test-flags "-v"
endif

.PHONY: test-upgrade-ci
test-upgrade-ci: ./vendor
	$(Q)sed -e "s,REPLACE_IMAGE,registry.svc.ci.openshift.org/${OPENSHIFT_BUILD_NAMESPACE}/stable:tektoncd-pipeline-operator-registry," ./test/e2e/catalog_source_OS4.yaml | oc apply -f -
	$(Q)oc apply -f ./test/e2e/subscription_OS4.yaml
	$(eval DEPLOYED_NAMESPACE := openshift-operators)
	$(Q)./hack/check-crds.sh
	$(Q)operator-sdk test local ./test/e2e --no-setup --go-test-flags "-v -timeout=15m"
	$(Q)oc delete -f ./test/e2e/catalog_source_OS4.yaml --wait
	$(Q)sed -e "s,REPLACE_IMAGE,registry.svc.ci.openshift.org/${OPENSHIFT_BUILD_NAMESPACE}/stable:tektoncd-pipeline-operator-registry-next," ./test/e2e/catalog_source_OS4.yaml | oc apply -f -
	$(Q)./hack/check-crds-upgrade.sh
	$(Q)operator-sdk test local ./test/e2e --no-setup --go-test-flags "-v -timeout=15m"
