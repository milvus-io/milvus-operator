# Create a kind cluster

> The Kind cluster is only for development/testing purpose.

## Prerequisites

* [Docker](https://docs.docker.com/engine/install/) installed (Version: v19.03.12+)
* [kind](https://kind.sigs.k8s.io/docs/user/quick-start/#installation) installed (Version: v0.8.1+).
* [kubectl](https://kubernetes.io/docs/tasks/tools/) installed (Version: v1.19+)

## Prepare kind config
Use the following command to create the kind configuration used for creating a kind cluster with multiple worker nodes:
```shell
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
- role: worker
- role: worker
- role: worker
```

## Create the kind cluster

Execute command:

```shell script
kind create cluster --name myk8s --config kind.yaml
```

## Check the cluster info
Check cluster info by

```shell script
kubectl cluster-info
```

Output:

```log
Kubernetes master is running at https://127.0.0.1:39821
KubeDNS is running at https://127.0.0.1:39821/api/v1/namespaces/kube-system/services/kube-dns:dns/proxy

To further debug and diagnose cluster problems, use 'kubectl cluster-info dump'.
```