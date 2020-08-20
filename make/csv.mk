QUAY_USERNAME ?=
QUAY_PASSWORD ?=

OPERATOR_NAME ?= openshift-pipelines-operator
OPERATOR_VERSION ?= 0.5.0

QYAPP_NAMESPACE ?= openshift-pipeline
QYAPP_REPOSITORY ?= openshift-pipelines-operators

OPERATOR_CI_IMAGE ?= registry.svc.ci.openshift.org/${OPENSHIFT_BUILD_NAMESPACE}/stable:tektoncd-pipeline-operator
OPERATOR_IMAGE ?= quay.io/$(QYAPP_NAMESPACE)/$(OPERATOR_NAME)

INSTALL_DIR ?= deploy/install

.PHONY: gen-tag
gen-tag:
	$(eval export TAG := $(shell date +%s))

.PHONY: tag-image
tag-image: gen-tag
	docker tag $(OPERATOR_CI_IMAGE) $(OPERATOR_IMAGE):$(OPERATOR_VERSION)-$(TAG)


.PHONY: gen-csv
gen-csv: courier
	$(eval OPERATOR_MANIFESTS := /tmp/artifacts/openshift-pipelines-operator)
	$(eval CREATION_TIMESTAMP := $(shell date --date="@$(TAG)" '+%Y-%m-%d %H:%M:%S'))
	./out/venv3/bin/operator-courier --verbose flatten manifests/ $(OPERATOR_MANIFESTS)
	cp -vf deploy/crds/*_crd.yaml $(OPERATOR_MANIFESTS)
	@sed -i -e 's,REPLACE_NAME,$(OPERATOR_NAME),g' $(OPERATOR_MANIFESTS)/openshift-pipelines-operator.v$(OPERATOR_VERSION).clusterserviceversion-v$(OPERATOR_VERSION).yaml
	@sed -i -e 's,REPLACE_VERSION,$(OPERATOR_VERSION),g' $(OPERATOR_MANIFESTS)/openshift-pipelines-operator.v$(OPERATOR_VERSION).clusterserviceversion-v$(OPERATOR_VERSION).yaml
	@sed -i -e 's,REPLACE_IMAGE,$(OPERATOR_IMAGE):$(OPERATOR_VERSION)-$(TAG),g' $(OPERATOR_MANIFESTS)/openshift-pipelines-operator.v$(OPERATOR_VERSION).clusterserviceversion-v$(OPERATOR_VERSION).yaml
	@sed -i -e 's,REPLACE_CREATED_AT,$(CREATION_TIMESTAMP),' $(OPERATOR_MANIFESTS)/openshift-pipelines-operator.v$(OPERATOR_VERSION).clusterserviceversion-v$(OPERATOR_VERSION).yaml
	@sed -i -e 's,REPLACE_NAME,$(OPERATOR_NAME),g' $(OPERATOR_MANIFESTS)/openshift-pipelines-operator.package.yaml
	@sed -i -e 's,REPLACE_VERSION,$(OPERATOR_VERSION),g' $(OPERATOR_MANIFESTS)/openshift-pipelines-operator.package.yaml
	@sed -i -e 's,REPLACE_PACKAGE,$(QYAPP_REPOSITORY),' $(OPERATOR_MANIFESTS)/openshift-pipelines-operator.package.yaml
	./out/venv3/bin/operator-courier --verbose verify --ui_validate_io $(OPERATOR_MANIFESTS)
	cp $(INSTALL_DIR)/operator-source.yaml /tmp/artifacts
	@sed -i -e 's,REPLACE_NAMESPACE,$(QYAPP_NAMESPACE),g' /tmp/artifacts/operator-source.yaml
	@sed -i -e 's,REPLACE_REPOSITORY,$(QYAPP_REPOSITORY),g' /tmp/artifacts/operator-source.yaml

.PHONY: push-quay-app
push-quay-app: gen-csv
	$(eval QUAY_API_TOKEN := $(shell curl -sH "Content-Type: application/json" -XPOST https://quay.io/cnr/api/v1/users/login -d '{"user":{"username":"'${QUAY_USERNAME}'","password":"'${QUAY_PASSWORD}'"}}' | jq -r '.token'))
	@./out/venv3/bin/operator-courier push $(OPERATOR_MANIFESTS) $(QYAPP_NAMESPACE) $(QYAPP_REPOSITORY) $(OPERATOR_VERSION)-$(TAG) "$(QUAY_API_TOKEN)"

.PHONY: gen-operator-source
gen-operator-source: push-quay-app
	cp $(INSTALL_DIR)/operator-source.yaml /tmp/artifacts
	@sed -i -e 's,REPLACE_NAMESPACE,$(QYAPP_NAMESPACE),g' /tmp/artifacts/operator-source.yaml
	@sed -i -e 's,REPLACE_REPOSITORY,$(QYAPP_REPOSITORY),g' /tmp/artifacts/operator-source.yaml

.PHONY: test-csv
#test-csv: gen-csv
test-csv:
	@echo "skip this check based on old operator format"
	@echo "reuse this target for tests on new format"

test-script:
	openshift/bundle-tests.sh
