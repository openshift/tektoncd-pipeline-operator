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


.PHONY: build
## Build the operator
build: ./out/operator ./out/build/bin manifests

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
	$(Q)CGO_ENABLED=0 GOARCH=amd64 GOOS=linux \
		go build ${V_FLAG} -o ./out/operator \
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
