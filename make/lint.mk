ifndef LINT_MK
LINT_MK:=# Prevent repeated "-include".

GOLANGCI_LINT_BIN=./out/golangci-lint

include ./make/verbose.mk
include ./make/go.mk

# NOTE: Disabling courier check by default until we move to a directory
# nomenclature as what courier is expecting
.PHONY: lint
## Runs linters on Go code files and YAML files
##  gen-csv is added here to test gen-csv target
lint: lint-go-code lint-yaml

YAML_FILES := $(shell find . -path ./vendor -prune -o -type f -regex ".*y[a]ml" -print)
.PHONY: lint-yaml
## runs yamllint on all yaml files
lint-yaml: ./vendor ${YAML_FILES}
	$(Q)yamllint -c .yamllint $(YAML_FILES)

.PHONY: lint-go-code
## Checks the code with golangci-lint
lint-go-code: ./vendor $(GOLANGCI_LINT_BIN)
	# This is required for OpenShift CI enviroment
	# Ref: https://github.com/openshift/release/pull/3438#issuecomment-482053250
	$(Q)XDG_CACHE_HOME=$(shell pwd)/out/cache \
	GOCACHE=$(shell pwd)/out/gocache \
	./out/golangci-lint ${V_FLAG} run --deadline=30m

$(GOLANGCI_LINT_BIN):
		$(Q) GO111MODULE=on \
		GOBIN=$(shell pwd)/out \
		go get -u github.com/golangci/golangci-lint/cmd/golangci-lint

	# re-enable this when the fix for the issue is released
	#$(Q)curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ./out v1.16.0

.PHONY: courier
## Validate manifests using operator-courier
courier:
	python3 -m venv ./out/venv3
	./out/venv3/bin/pip install --upgrade setuptools
	./out/venv3/bin/pip install --upgrade pip
	./out/venv3/bin/pip install operator-courier==2.1.2
	# enable operator-courier tests later
	# flatten command is throwing error. suppress it for now
	# @-./out/venv3/bin/operator-courier flatten ./manifests/tektoncd-pipelines ./out/manifests-flat
	#(Q)./out/venv3/bin/operator-courier verify ./out/manifests-flat

endif
