ifndef DOCKER_MK
DOCKER_MK:=# Prevent repeated "-include".

# If running in Jenkins we don't allow for interactively running the container
DOCKER_RUN_INTERACTIVE_SWITCH = -i
ifneq ($(BUILD_TAG),)
	DOCKER_RUN_INTERACTIVE_SWITCH =
endif

.PHONY: docker-image-deploy
## Build the docker image that can be deployed (only contains bare operator)
docker-image-deploy: Dockerfile
	$(Q)docker build ${Q_FLAG} \
		--build-arg GO_PACKAGE_PATH=${GO_PACKAGE_PATH} \
		--build-arg VERBOSE=${VERBOSE} \
		--target deploy \
		. \
		-t ${GO_PACKAGE_ORG_NAME}/${GO_PACKAGE_REPO_NAME}:${GIT_COMMIT_ID}
	$(Q)docker tag ${GO_PACKAGE_ORG_NAME}/${GO_PACKAGE_REPO_NAME}:${GIT_COMMIT_ID} tektoncd-pipeline-operator-deploy

DOCKER_BUILD_TOOLS_CONTAINER := build-tools

.PHONY: docker-image-build-tools
## Build the docker image that has all the build-tools installed
docker-image-build-tools: Dockerfile
	$(Q)docker build ${Q_FLAG} \
	--build-arg GO_PACKAGE_PATH=${GO_PACKAGE_PATH} \
	--build-arg VERBOSE=${VERBOSE} \
	--target build-tools \
	. \
	-t ${GO_PACKAGE_ORG_NAME}/${GO_PACKAGE_REPO_NAME}-build-tools:${GIT_COMMIT_ID}

DOCKER_BUILD_TOOLS_CONTAINER_NAME=${GO_PACKAGE_REPO_NAME}-build-tools

.PHONY: docker-start
## Starts the docker build container in the background (detached mode).
## After calling this command you can invoke all the make targets from the
## normal Makefile (e.g. build or test) inside the build container
## by prefixing them with "docker-". For example to execute "make build"
## inside the build container, just run "make docker-build".
## To remove the container when no longer needed, call "make docker-rm".
## NOTE: If a container already exists, it will be removed before starting
## a new one.
docker-start: docker-image-build-tools docker-rm
	$(Q)docker run \
		--detach=true \
		-t \
		$(DOCKER_RUN_INTERACTIVE_SWITCH) \
		--name="$(DOCKER_BUILD_TOOLS_CONTAINER_NAME)" \
		-v $(shell pwd):/tmp/go/src/${GO_PACKAGE_PATH}:Z \
		-u $(shell id -u ${USER}):$(shell id -g ${USER}) \
		${GO_PACKAGE_ORG_NAME}/${GO_PACKAGE_REPO_NAME}-build-tools:${GIT_COMMIT_ID}
	$(info Docker container "$(DOCKER_BUILD_TOOLS_CONTAINER_NAME)" created. Continue with "make docker-build")

.PHONY: docker-rm
## Removes the docker build container, if any (see "make docker-start").
docker-rm:
	$(info removing any "$(DOCKER_BUILD_TOOLS_CONTAINER_NAME)" container (if exists))
	$(Q)-docker rm -f "$(DOCKER_BUILD_TOOLS_CONTAINER_NAME)"

.PHONY: check-build-tools-container-is-running
# Runs a check to see if the build tools container is running and issues and
# error if it isn't. You can make this target a dependency to all targets that
# need the docker build tools container to be running.
check-build-tools-container-is-running: 
	$(eval makecommand:=$(subst docker-,,$@))
ifeq ($(strip $(shell docker ps -qa --filter "name=$(DOCKER_BUILD_TOOLS_CONTAINER_NAME)" 2>/dev/null)),)
	$(error No container name "$(DOCKER_BUILD_TOOLS_CONTAINER_NAME)" exists. Consider running "make docker-start" first)
endif

.PHONY: docker-%
## This is a wildcard target to let you call any make target from the normal
## makefile but it will run inside the docker build tools container that you've
## started with docker-start. This target will only get executed if there's no
## specialized form available. For example if you call "make docker-start" not
## this target gets executed but the "docker-start" target.
docker-%: check-build-tools-container-is-running
	$(eval makecommand:=$(subst docker-,,$@))
	$(Q)docker exec \
		-t \
		$(DOCKER_RUN_INTERACTIVE_SWITCH) \
		"$(DOCKER_BUILD_TOOLS_CONTAINER_NAME)" \
		bash -ec 'make VERBOSE=${VERBOSE} $(makecommand)'

endif
