FROM scratch

LABEL operators.operatorframework.io.bundle.mediatype.v1=registry+v1
LABEL operators.operatorframework.io.bundle.manifests.v1=manifests/
LABEL operators.operatorframework.io.bundle.metadata.v1=metadata/
LABEL operators.operatorframework.io.bundle.package.v1=openshift-pipelines-operator-midstr
LABEL operators.operatorframework.io.bundle.channels.v1=canary
LABEL operators.operatorframework.io.bundle.channel.default.v1=canary

COPY bundle/manifests /manifests/
COPY bundle/metadata /metadata/
