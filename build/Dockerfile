FROM registry.access.redhat.com/ubi8-dev-preview/ubi-minimal:latest

ENV OPERATOR=/usr/local/bin/openshift-pipelines-operator \
    USER_UID=1001 \
    USER_NAME=openshift-pipelines-operator

# install operator binary
COPY build/_output/bin/tektoncd-pipeline-operator ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

COPY deploy/resources /deploy/resources

# UBI8 image missing uuidgen
RUN cat /proc/sys/kernel/random/uuid > /deploy/uuid

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
