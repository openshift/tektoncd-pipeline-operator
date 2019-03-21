# Tektoncd-operator

## Dev env

### Prerequisites

1. operator-sdk: https://github.com/operator-framework/operator-sdk

2. minikube: https://kubernetes.io/docs/tasks/tools/install-minikube/

### Setup Minikube

**create minikube instance**

```
minikube start -p mk-tekton --extra-config=apiserver.enable-admission-plugins="LimitRanger,NamespaceExists,NamespaceLifecycle,ResourceQuota,ServiceAccount,DefaultStorageClass,MutatingAdmissionWebhook" --extra-config=apiserver.service-node-port-range=80-32767 --cpus=2 --memory=6144 --kubernetes-version=v1.12.0
```

**set docker env**

```
eval $(minikube docker-env -p mk-tekton)
```

### Install OLM

**clone OLM repository (into go path)**

```
> cd $GOPATH/github.com/operator-framework
```
```
$GOPATH/github.com/operator-framework> git clone git@github.com:operator-framework/operator-lifecycle-manager.git
```
```
> cd $GOPATH/github.com/operator-framework/operator-lifecycle-manager
```

**set NO_MINIKUBE environment variable:** to prevent minkube from starting a new virtual machine (value doesn't matter, just ensure that it is non-empty)
```
> export NO_MINIKUBE=skip
```

**check cluster config**
```
> minikube ip -p mk-tekton
```
```
kubectl cluster-config
```
the IP's should match

**install OLM**
from operator-lifecycle-manager root run:
```
make run-local
```

### Deploy tekton-operator


