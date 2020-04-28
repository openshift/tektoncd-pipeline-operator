##########------------------------------------------------------------##########
##########- Operator Release------------------------------------------##########
##########------------------------------------------------------------##########

STABLE_RELEASE_URL := 'https://raw.githubusercontent.com/openshift/tektoncd-pipeline/release-v${PIPELINE_VERSION}/openshift/release/tektoncd-pipeline-v${PIPELINE_VERSION}.yaml'
PIPELINE_PATH=deploy/resources/v${PIPELINE_VERSION}/pipelines
.PHONY: opo-payload-pipeline
opo-payload-pipeline:
ifndef PIPELINE_VERSION
	@echo PIPELINE_VERSION not set
	@exit 1
endif
	[[ -d "${PIPELINE_PATH}" ]] || mkdir -p ${PIPELINE_PATH}
	curl -s -o ${PIPELINE_PATH}/00-release.yaml ${STABLE_RELEASE_URL}
	sed -i 's/^[[:space:]]*TektonVersion.*/TektonVersion = "'v${PIPELINE_VERSION}'"/' pkg/flag/flag.go
	go fmt pkg/flag/flag.go

TRIGGERS_STABLE_RELEASE_URL='https://raw.githubusercontent.com/openshift/tektoncd-triggers/release-v${TRIGGERS_VERSION}/openshift/release/tektoncd-triggers-v${TRIGGERS_VERSION}.yaml'
TRIGGERS_PATH='deploy/resources/v${PIPELINE_VERSION}/addons/triggers'
.PHONY: opo-payload-triggers
opo-payload-triggers:
ifndef PIPELINE_VERSION
	@echo PIPELINE_VERSION not set
	@exit 1
endif
ifndef TRIGGERS_VERSION
	@echo TRIGGERS_VERSION not set
	@exit 1
endif
	[[ -d "${TRIGGERS_PATH}" ]] || mkdir -p ${TRIGGERS_PATH}
	curl -s -o ${TRIGGERS_PATH}/tektoncd-triggers-v${TRIGGERS_VERSION}.yaml ${TRIGGERS_STABLE_RELEASE_URL}

.PHONY: opo-test-clean
opo-test-clean:
	-oc delete -f deploy/ --ignore-not-found
	-oc delete -f deploy/crds/ --ignore-not-found

.PHONY: opo-set-pipeline-payload-version
opo-set-pipeline-payload-version:
ifndef PAYLOAD_VERSION
	@echo PAYLOAD_VERSION not set
	@exit 1
endif
	sed -i 's/^[[:space:]]*TektonVersion.*/TektonVersion = "'${PAYLOAD_VERSION}'"/' ./pkg/flag/flag.go
	go fmt ./pkg/flag/flag.go

.PHONY: opo-cluster-tasks
opo-cluster-tasks:
ifndef PIPELINE_VERSION
	@echo PIPELINE_VERSION not set
	@echo example: make opo-cluster-tasks CATALOG_VERSION=release-v0.9 PIPELINE_VERSION=v0.9.2 CATALOG_VERSION_SUFFIX=v0.9.0
	@exit 1
endif
ifndef CATALOG_VERSION
	@echo CATALOG_VERSION not set
	@echo example: make opo-cluster-tasks CATALOG_VERSION=release-v0.9 PIPELINE_VERSION=v0.9.2 CATALOG_VERSION_SUFFIX=v0.9.0
	@exit 1
endif
ifndef CATALOG_VERSION_SUFFIX
	@echo CATALOG_VERSION_SUFFIX not set
	@echo example: make opo-cluster-tasks CATALOG_VERSION=release-v0.9 PIPELINE_VERSION=v0.9.2 CATALOG_VERSION_SUFFIX=v0.9.0
	@exit 1
endif
	./scripts/update-tasks.sh ${CATALOG_VERSION} deploy/resources/${PIPELINE_VERSION} ${CATALOG_VERSION_SUFFIX}

.PHONY: opo-test-e2e-up-local
opo-test-e2e-up-local: opo-test-clean
	operator-sdk test local ./test/e2e/ --up-local --namespace openshift-pipelines  --go-test-flags "-v -timeout=10m" --local-operator-flags "--recursive"

.PHONY: opo-test-e2e
opo-test-e2e: opo-test-clean
	operator-sdk test local ./test/e2e/ --namespace openshift-operators  --go-test-flags "-v -timeout=10m" --local-operator-flags "--recursive"

# make targets for release
.PHONY: opo-clean
opo-clean:
	rm -rf build/_output

.PHONY: opo-image
opo-image: opo-clean
ifndef VERSION
	@echo VERSION not set
	@exit 1
endif
ifndef QUAY_NAMESPACE
	@echo QUAY_NAMESPACE not set
	@exit 1
endif
	operator-sdk build quay.io/${QUAY_NAMESPACE}/openshift-pipelines-operator:v${VERSION} --go-build-args "-o build/_output/bin/openshift-pipelines-operator"

.PHONY: opo-image-push
opo-image-push:
ifndef VERSION
	@echo VERSION not set
	@exit 1
endif
ifndef QUAY_NAMESPACE
	@echo QUAY_NAMESPACE not set
	@exit 1
endif
	docker push quay.io/${QUAY_NAMESPACE}/openshift-pipelines-operator:v${VERSION}

.PHONY: opo-operator-yaml-update
opo-operator-yaml-update:
ifndef VERSION
	@echo VERSION not set
	@exit 1
endif
	sed -i 's/image:.*/image: quay.io\/'${QUAY_NAMESPACE}'\/openshift-pipelines-operator:v'${VERSION}'/' deploy/operator.yaml

.PHONY: opo-new-csv
opo-new-csv:
ifndef VERSION
	@echo VERSION not set
	@exit 1
endif
ifndef FROM_VERSION
	@echo FROM_VERSION not set
	@exit 1
endif
ifndef CHANNEL
	@echo CHANNEL not set
	@exit 1
endif
	operator-sdk olm-catalog gen-csv \
      --csv-channel ${CHANNEL} \
      --csv-version ${VERSION} \
      --from-version ${FROM_VERSION} \
      --operator-name  openshift-pipelines-operator \
      --update-crds

.PHONY: opo-opr-verify
opo-opr-verify:
	operator-courier verify \
		--ui_validate_io \
		deploy/olm-catalog/openshift-pipelines-operator

.PHONY: opo-push-quay-app
opo-push-quay-app:
ifndef VERSION
	@echo VERSION not set
	@exit 1
endif
ifndef MY_QUAY_NAMESPACE
	@echo QUAY_NAMESPACE not set
	@exit 1
endif
ifndef TOKEN
	@echo TOKEN not set
	@exit 1
endif
	operator-courier --verbose push  \
		./deploy/olm-catalog/openshift-pipelines-operator \
		${MY_QUAY_NAMESPACE} \
		openshift-pipelines-operator \
		${VERSION}  \
		"${TOKEN}"

.PHONY: opo-olm-clean
opo-olm-clean:
	oc delete operatorsource -n openshift-marketplace ${MY_QUAY_NAMESPACE}-operators --ignore-not-found

export define operatorsource
apiVersion: operators.coreos.com/v1
kind: OperatorSource
metadata:
  name: ${MY_QUAY_NAMESPACE}-operators
  namespace: openshift-marketplace
spec:
  type: appregistry
  endpoint: https://quay.io/cnr
  registryNamespace: ${MY_QUAY_NAMESPACE}
  displayName: "${MY_QUAY_NAMESPACE} Operators"
  publisher: "${MY_QUAY_NAMESPACE}"
endef

.PHONY: opo-operator-source
opo-operator-source: opo-olm-clean
	@echo ::::: operator soruce manifest :::::
	@echo "$$operatorsource"
	@echo ::::::::::::::::::::::::::::::
	@echo "$$operatorsource" | oc apply -f -

export define subscription
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: ${MY_QUAY_NAMESPACE}-pipelines-subscription
  namespace: openshift-operators
spec:
  channel: ${CHANNEL}
  name: openshift-pipelines-operator
  source: ${MY_QUAY_NAMESPACE}-operators
  sourceNamespace: openshift-marketplace
endef

.PHONY: opo-subscription
opo-subscription:
	@echo ::::: subscription soruce manifest :::::
	@echo "$$subscription"
	@echo ::::::::::::::::::::::::::::::
	@echo "$$subscription" | oc apply -f -

.PHONY: opo-test-scorecard
opo-test-scorecard:
ifndef VERSION
	@echo VERSION not set
	@exit 1
endif
	operator-sdk scorecard \
		--olm-deployed \
		--csv-path deploy/olm-catalog/openshift-pipelines-operator/${VERSION}/openshift-pipelines-operator.v${VERSION}.clusterserviceversion.yaml \
		--namespace openshift-operators \
		--cr-manifest ./deploy/crds/operator_v1alpha1_config_cr.yaml \
		--crds-dir .deploy/olm-catalog/openshift-pipelines-operator/${VERSION}
