# Install with helm
The installation guide documented here help you deploy Milvus operator stack with helm, which is the recommended way.

## Prerequisites
1. `Kubernetes` cluster (v1.19+) is running.
2. `Helm` is [installed](https://helm.sh/).
2. `cert-manager`(optional) (v1.0+) is [installed](https://cert-manager.io/docs/installation/kubernetes/).

## Installation

For quick start, install with one line command:

```shell
helm install milvus-operator \
  -n milvus-operator --create-namespace \
  https://github.com/milvus-io/milvus-operator/releases/download/v0.7.4/milvus-operator-0.7.4.tgz
```

If you already have `cert-manager` v1.0+ installed which is not in its default configuration, you may encounter some error with the check of cert-manager installation. you can install with special options to disable the check:

```
helm install milvus-operator \
  -n milvus-operator --create-namespace \
  https://github.com/milvus-io/milvus-operator/releases/download/v0.7.4/milvus-operator-0.7.4.tgz \
  --set checker.disableCertManagerCheck=true
```

## Check installation & do update

use helm commands to check installation:

```shell
# list installations
helm -n milvus-operator list
# get values configuratins
helm -n milvus-operator get values milvus-operator
```

use helm commands to upgrade earlier milvus-operator to current version:

```shell
helm upgrade -n milvus-operator milvus-operator --reuse-values \
  https://github.com/milvus-io/milvus-operator/releases/download/v0.7.4/milvus-operator-0.7.4.tgz
```

## Delete operator
Delete the milvus operator stack by helm

```shell
helm uninstall milvus-operator -n milvus-operator
```

# Install with deployment manifest

If you don't want to use helm you can also install with kubectl and raw manifests.

## Prerequisites
1. `Kubernetes` cluster (v1.19+) is running.
2. `cert-manager`(optional) (v1.0+) is [installed](https://cert-manager.io/docs/installation/kubernetes/) in cert-manager namespace with default config.
3. `kubectl` with a proper version(v1.19+) is [installed](https://kubernetes.io/docs/tasks/tools/).
4. `git` (optional) is [installed](https://git-scm.com/book/en/v2/Getting-Started-Installing-Git).

## Installation
Directly apply the deployment manifest to your Kubernetes cluster:
```shell
kubectl apply -f https://raw.githubusercontent.com/milvus-io/milvus-operator/main/deploy/manifests/deployment.yaml
```

Or install the milvus operator stack by using makefile
```shell
git clone https://github.com/milvus-io/milvus-operator.git
# Checkout to the specified branch or the specified tag.
# To branch: git checkout <branch-name> e.g.: git checkout release-0.1.0

make deploy
``` 

>NOTES: Here we use the deployment manifest in the `main` branch as an example, for deploying the released versions, you can get the deployment manifest in the GitHub release page or find it in the corresponding code branch such as `v0.1.0`.

Check the installed operators:

```shell
kubectl get pods -n milvus-operator
```

Output:
```log
NAME                                                  READY   STATUS    RESTARTS   AGE
milvus-operator-698fc7dc8d-8f52d   1/1     Running   0          65s
```

## Delete operator
Delete the milvus operator stack by the deployment manifest:

```shell
kubectl delete -f https://raw.githubusercontent.com/milvus-io/milvus-operator/main/deploy/manifests/deployment.yaml
```

Or delete the milvus operator stack by using makefile:

```shell
make undeploy
```


# What's next

If the Milvus operator is successfully installed, you can follow the guide shown to deploy your Milvus cluster to your Kubernetes and try it. The examples can be found in https://github.com/milvus-io/milvus-operator/tree/main/config/samples

quick start with `kubectl apply -f https://raw.githubusercontent.com/milvus-io/milvus-operator/main/config/samples/milvus_minimum.yaml`

# Install Kind for Development
For local development purpose, check [Kind installation](./kind-installation.md).
