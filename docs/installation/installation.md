# Install with deployment manifest

The installation guide documented here help you deploy Milvus operator stack with a deployment manifest, which is the recommended way.

## Prerequisites
1. `Kubernetes` cluster (v1.19+) is running.
2. `cert-manager` (v1.3+) is [installed](https://cert-manager.io/docs/installation/kubernetes/).
3. `kubectl` with a proper version(v1.19+) is [installed](https://kubernetes.io/docs/tasks/tools/).
4. `git` (optional) is [installed](https://git-scm.com/book/en/v2/Getting-Started-Installing-Git).

### Kind
For local development purpose, check [Kind installation](./kind-installation.md).

### cert-manager
Milvus Operator using cert manager for provisioning the certificates for the webhook server. It can be installed as follows:
```shell
$ kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.5.3/cert-manager.yaml
```
Make sure the cert-manager is successfully deployed in the Kubenetes cluster.
```shell
$ kubectl get pods -n cert-manager
NAME                                      READY   STATUS    RESTARTS   AGE
cert-manager-848f547974-gccz8             1/1     Running   0          70s
cert-manager-cainjector-54f4cc6b5-dpj84   1/1     Running   0          70s
cert-manager-webhook-7c9588c76-tqncn      1/1     Running   0          70s
```

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

>NOTES: Here we use the deployment manifest in the `main` branch as an example, for deploying the released versions, you can get the deployment manifest in the GitHub release page or find it in the corresponding code branch such as `release-0.1.0`.

Check the installed operators:

```shell
kubectl get pods -n milvus-operator
```

Output:
```log
NAME                                                  READY   STATUS    RESTARTS   AGE
milvus-operator-controller-manager-698fc7dc8d-8f52d   1/1     Running   0          65s
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

## What's next

If the Milvus operator is successfully installed, you can follow the guide shown to deploy your Milvus cluster to your Kubernetes and try it.