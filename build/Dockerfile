FROM registry.access.redhat.com/ubi8-dev-preview/ubi-minimal:latest

ENV OPERATOR=/usr/local/bin/openshift-pipelines-operator \
    USER_UID=1001 \
    USER_NAME=openshift-pipelines-operator

# install operator binary
COPY build/_output/bin/openshift-pipelines-operator ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

COPY deploy/resources /deploy/resources
COPY deploy/post-install /deploy/post-install

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
