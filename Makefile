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
include ./opo-makefile.mk
include ./opo-makefile-new-bundle-format.mk

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

ifeq ($(shell uname -m),x86_64)
        ARCH := amd64
else ifeq ($(shell uname -m),ppc64le)
        ARCH := ppc64le
endif

./out/operator: ./vendor $(shell find . -path ./vendor -prune -o -name '*.go' -print)
	#$(Q)operator-sdk generate k8s
	$(Q)go version
	$(Q)CGO_ENABLED=0 GOARCH=${ARCH} GOOS=linux \
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

ifeq ($(uname -m),x86_64)
        ARCH=amd64
else ifeq ($(uname -m),ppc64le)
        ARCH=ppc64le
endif
	$(Q)rm -rf build/_output/bin
	$(eval IMAGE_TAG := quay.io/rhpipeline/openshift-pipelines-operator:test)
	$(Q) GOARCH=${ARCH} operator-sdk build \
	--go-build-args "-o build/_output/bin/openshift-pipelines-operator" \
	$(IMAGE_TAG)
