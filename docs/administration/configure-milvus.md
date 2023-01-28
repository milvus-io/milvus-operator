# Configure Milvus with Milvus Operator

This document describes how to set configure Milvus componnets with Milvus Operator.

Before you start, you should have a basic understanding of the Custom Resource (CR) in Kubernetes. If not, please refer to [Kubernetes CRD doc](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/).

## The milvus.yaml

Usually, a Milvus component reads its configuration through a configuration file, which is often refered to as `milvus.yaml`.

If we don't set any specific configuration, the default config file within the your Milvus image will be used. You can find the default configurations for milvus releases in the Milvus GitHub repository.

For example: the latest milvus release v2.2.2's default configuration file can be found at `https://github.com/milvus-io/milvus/blob/master/configs/milvus.yaml`, and the following link describes most configurations fields in detail: `https://milvus.io/docs/system_configuration.md`.

## Change the milvus.yaml

All configurations in the `milvus.yaml` can be changed by setting the corresponding field in the `spec.config` of the Milvus CRD.

For example you deploy a Milvus demo without setting any specific configuration under `spec.config`, like below:

```yaml
apiVersion: milvus.io/v1beta1
kind: Milvus
metadata:
  name: my-release
  labels:
    app: milvus
spec:
  config: {}
# Omit other fields
# ...
```

Then no change will be made to the default configuration file.

To change some configurations, for example we want to change the default log level to `warn`, max log file size to `300MB` and change the default log path to `/var/log/milvus`, we can set the `spec.config` like below:

```yaml
apiVersion: milvus.io/v1beta1
kind: Milvus
metadata:
  name: my-release
  labels:
    app: milvus
spec:
  config:
    # ref: https://milvus.io/docs/configure_log.md
    log:
      level: warn
      file:
        maxSize: 300
        rootPath: /var/log/milvus
```

## Configuration for Milvus dependencies

There're also configuration sections about milvus's dependencies in the `milvus.yaml` (like `minio`, `etcd`, `pulsar`...). Usually you don't need to change these configurations, because the Milvus Operator will set them automatically according to your specifications in `spec.dependencies.<dependency name>` field.

But you can still set these configurations mannually if you want to. It will override the configurations set by Milvus Operator.