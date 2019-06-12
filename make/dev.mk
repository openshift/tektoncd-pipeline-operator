ifndef DEV_MK
DEV_MK:=# Prevent repeated "-include".

include ./make/verbose.mk
include ./make/git.mk

DOCKER_REPO?=quay.io/openshift-pipeline
IMAGE_NAME?=tektoncd-pipeline-operator
REGISTRY_URI=quay.io

TEKTONCD_PIPELINE_OPERATOR_IMAGE?=quay.io/openshift-pipeline/tektoncd-pipeline-operator
TIMESTAMP:=$(shell date +%s)
TAG?=$(GIT_COMMIT_ID_SHORT)-$(TIMESTAMP)
OPENSHIFT_VERSION?=4

# to watch all namespaces, keep namespace empty
APP_NAMESPACE ?= ""
LOCAL_TEST_NAMESPACE ?= "local-test"

.PHONY: local
## Run Operator locally
local: deploy-rbac build deploy-crd
	$(Q)-oc new-project $(LOCAL_TEST_NAMESPACE)
	$(Q)operator-sdk up local --namespace=$(APP_NAMESPACE)

.PHONY: deploy-rbac
## Setup service account and deploy RBAC
deploy-rbac:
	$(Q)-oc login -u system:admin
	$(Q)-oc create -f deploy/service_account.yaml
	$(Q)-oc create -f deploy/role.yaml
	$(Q)-oc create -f deploy/role_binding.yaml

.PHONY: deploy-crd
## Deploy CRD
deploy-crd:
	$(Q)-oc apply -f deploy/crds/openshift-pipelines-operator-tekton-v1alpha1_install_crd.yaml
	$(Q)-oc apply -f deploy/crds/openshift_v1alpha1_install_crd.yaml

.PHONY: deploy-operator
## Deploy Operator
deploy-operator: deploy-crd
	$(Q)oc create -f deploy/operator.yaml

.PHONY: deploy-clean
## Deploy a CR as test
deploy-clean:
	$(Q)-oc delete project $(LOCAL_TEST_NAMESPACE)

.PHONY: build-operator-image
## Build and create the operator container image
build-operator-image: ./vendor
	operator-sdk build $(TEKTONCD_PIPELINE_OPERATOR_IMAGE):$(TAG)

.PHONY: deploy-operator-only
deploy-operator-only:
	@echo "Creating Deployment for Operator"
	@cat minishift/operator.yaml | sed s/\:dev/:$(TAG)/ | oc create -f -

.PHONY: clean-all
clean-all:  clean-operator clean-resources

.PHONY: clean-operator
clean-operator:
	@echo "Deleting Deployment for Operator"
	@cat minishift/operator.yaml | sed s/\:dev/:$(TAG)/ | oc delete -f - || true

.PHONY: clean-resources
clean-resources:
	@echo "Deleting sub resources..."
	@echo "Deleting ClusterRoleBinding"
	@oc delete -f ./deploy/role_binding.yaml || true
	@echo "Deleting ClusterRole"
	@oc delete -f ./deploy/role.yaml || true
	@echo "Deleting Service Account"
	@oc delete -f ./deploy/service_account.yaml || true
	@echo "Deleting Custom Resource Definitions..."
	@oc delete -f ./deploy/crds/openshift-pipelines-operator-tekton-v1alpha1_install_crd.yaml || true
	@oc delete -f ./deploy/crds/openshift_v1alpha1_install_crd.yaml || true

.PHONY: deploy-operator
deploy-operator: build build-operator-image deploy-operator-only

.PHONY: minikube-start
minikube-start:
	minikube start --cpus 4 --memory 8GB \
	--extra-config=apiserver.enable-admission-plugins="LimitRanger,NamespaceExists,NamespaceLifecycle,ResourceQuota,ServiceAccount,DefaultStorageClass,MutatingAdmissionWebhook" \
	--extra-config=apiserver.service-node-port-range=80-32767

.PHONY: deploy-all
deploy-all: clean-resources tektoncd-pipeline-operator

endif
