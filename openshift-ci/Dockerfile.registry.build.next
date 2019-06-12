FROM quay.io/openshift/origin-operator-registry:latest

COPY manifests.tar.gz .
RUN tar zxvf manifests.tar.gz
RUN initializer

USER 1001
EXPOSE 50051
CMD ["registry-server", "--termination-log=log.txt"]
