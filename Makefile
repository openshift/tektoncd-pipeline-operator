export GO111MODULE:=on
export GOCACHE=/tmp/.cache/go-build

include ./make/verbose.mk
.DEFAULT_GOAL := help
include ./make/help.mk
include ./make/out.mk
include ./make/find-tools.mk
include ./make/go.mk
include ./make/git.mk
include ./make/dev.mk
include ./make/format.mk
include ./make/lint.mk
include ./make/test.mk
include ./make/docker.mk
include ./make/csv.mk

BUILD_OUTPUT_DIR ?= ./out/
BUILD_OUTPUT_FILE ?= operator
VENDOR_BUILD ?= 

.PHONY: build
## Build the operator
build: ./out/operator

.PHONY: build-all
## Build the operator and compress all manifests
build-all: ./out/operator ./out/build/bin manifests

.PHONY: clean
clean:
	$(Q)-rm -rf ${V_FLAG} ./out
	$(Q)-rm -rf ${V_FLAG} ./vendor
	$(Q)-rm -rf ${V_FLAG} ./tmp
	$(Q)go clean ${X_FLAG} ./...

.PHONY: ./vendor
./vendor: go.mod go.sum
	$(Q)go mod vendor

./out/operator: ./vendor $(shell find . -path ./vendor -prune -o -name '*.go' -print)
	#$(Q)operator-sdk generate k8s
	$(Q)go version
	$(Q)CGO_ENABLED=0 GOARCH=amd64 GOOS=linux \
		go build ${V_FLAG} ${VENDOR_BUILD} -o ${BUILD_OUTPUT_DIR}${BUILD_OUTPUT_FILE} \
		./cmd/manager

./out/build/bin:
	$(Q)mkdir -p ./out/build
	$(Q)cp -r build/bin ./out/build/bin

manifests:
	$(Q)cp -r deploy/olm-catalog manifests && \
	tar -zcf manifests.tar.gz manifests && \
	rm -rf manifests

# TODO: Disable for now for CI to go over
upgrade-build: #TODO: reenable it

.PHONY: osdk-image
osdk-image:
	$(Q)rm -rf build/_output/bin
	$(eval IMAGE_TAG := quay.io/rhpipeline/openshift-pipelines-operator:test)
	$(Q)operator-sdk build \
	--go-build-args "-o build/_output/bin/openshift-pipelines-operator" \
	$(IMAGE_TAG)

##########------------------------------------------------------------##########
##########- Operator Release------------------------------------------##########
##########------------------------------------------------------------##########

.PHONY: opo-test-clean
opo-test-clean:
	-oc delete -f deploy/
	-oc delete -f deploy/crds/

.PHONY: opo-up-local
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
	operator-sdk build quay.io/openshift-pipeline/openshift-pipelines-operator:v${VERSION}

.PHONY: opo-image-push
opo-image-push: opo-image
ifndef VERSION
	@echo VERSION not set
	@exit 1
endif
	docker push quay.io/openshift-pipeline/openshift-pipelines-operator:v${VERSION}

.PHONY: opo-build-push-update
opo-build-push-update: opo-image-push
ifndef VERSION
	@echo VERSION not set
	@exit 1
endif
	sed -i 's/image:.*/image: quay.io\/openshift-pipeline\/openshift-pipelines-operator:'v${VERSION}'/' deploy/operator.yaml

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
      --csv-channel dev-preview \
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
ifndef QUAY_NAMESPACE
	@echo QUAY_NAMESPACE not set
	@exit 1
endif
ifndef TOKEN
	@echo TOKEN not set
	@exit 1
endif
	operator-courier --verbose push  \
		./deploy/olm-catalog/openshift-pipelines-operator \
		${QUAY_NAMESPACE} \
		openshift-pipelines-operator \
		${VERSION}  \
		"${TOKEN}"

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
