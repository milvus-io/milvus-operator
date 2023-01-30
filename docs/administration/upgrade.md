
# Upgrade Milvus Cluster with Milvus Operator

This topic describes how to ugrade your Milvus with Milvus Operator.

## Upgrade By Change Milvus Image

Usually you can simply update your Milvus' image to upgrade to a new version.

#### Example

For example, if you're using Milvus v2.1.0 you can upgrade to Milvus v2.1.4 in this way.

```yaml
apiVersion: milvus.io/v1beta1
kind: Milvus
metadata:
name: my-release
labels:
app: milvus
spec:
  # Omit other fields ...
  image: milvusdb/milvus:v2.1.4
```

> If you want to upgrade to Milvus v2.2.0, changing the image will lost your built index. you'll have to rebuild the index after upgrading. Or you can migrate your metadata before upgrade. Check the next section for more details.

## Upgrading Milvus 2.1.x to Milvus 2.2.x
The metadata structure of Milvus 2.2.x is different from that of Milvus 2.1.x. Therefore, you need to migrate the metadata of Milvus 2.1.x to Milvus 2.2.x. The following steps describe how to upgrade Milvus 2.1.4 to Milvus 2.2.0.

#### 1. Upgrade you Milvus Operator to v0.7.4

Run the following command to upgrade the version of your Milvus Operator to v0.7.4

```
helm repo add milvus-operator https://milvus-io.github.io/milvus-operator/
helm repo update milvus-operator
helm -n milvus-operator upgrade milvus-operator milvus-operator/milvus-operator
```


#### 2. Create a `milvusupgrade.yaml` file for metadata migration

Create a metadata migration file. The following is an example. You need to specify the `name`, `sourceVersion`, and `targetVersion` in the configuration file. The following example sets the `name` to `my-release-upgrade`, `sourceVersion` to `v2.1.4`, and `targetVersion` to `v2.2.0`. This means that your Milvus cluster will be upgraded from v2.1.4 to v2.2.0.

```yaml
apiVersion: milvus.io/v1beta1
kind: MilvusUpgrade
metadata:
  name: my-release-upgrade
spec:
  milvus:
    namespace: default
    name: my-release
  sourceVersion: "v2.1.4"
  targetVersion: "v2.2.0"
  # below are some omit default values:
  # targetImage: "milvusdb/milvus:v2.2.0"
  # toolImage: "milvusdb/meta-migration:v2.2.0"
  # operation: upgrade
  # rollbackIfFailed: true
  # backupPVC: ""
  # maxRetry: 3
```



#### 3. Apply the new configuration

Run the following command to start upgrading your Milvus cluster.

```
$ kubectl apply -f milvusupgrade.yaml
```


#### 4. Check the status of metadata migration

Run the following command to check the status of your metadata migration.

```shell
kubectl describe milvusupgrade my-release-upgrade
```

The status of `Succeeded` in the output means that the metadata migration is successful. 

If failed, the milvus-operator will automatically rollback the changes, and restore the old version of Milvus.

You can also run `kubectl get pod` to check all the pods After the metadata migration is successful.

## 5. Cleanup backups created during the upgrade

During the upgrade, the milvus-operator will create a backup of the metadata. You can delete the backup after the upgrade is successful.

```
$ kubectl delete -f milvusupgrade.yaml
```
