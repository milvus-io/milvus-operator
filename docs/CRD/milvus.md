# Custom Resource Definition
This document guides user to learn the related fields defined in the `Milvus` CRD and then customize their Milvus cluster deployment stack.

*CRD version*: `v1alpha1`

## CRD spec
Describe the spec fields with YAML code snippets and comments. All the parts here share the head YAML code snippet shown below.

``` yaml
apiVersion: milvus.io/v1alpha1
kind: Milvus
metadata:
  name: milvus-sample
  namespace: sample-ns
spec:
  # Global image name for milvus components. It will override the default one. Default is determined by operator version
  image: milvusdb/milvus:latest # Optional

  # Global image pull policy. It will override the the default one.
  imagePullPolicy: IfNotPresent # Optional, default = IfNotPresent

  # Global image pull secrets.
  imagePullSecrets: # Optional
  - name: mySecret
  # Global environment variables
  env: # Optional
  - name: key
    value: value

  # Global nodeSelector.
  # NodeSelector is a selector which must be true for the component to fit on a node.
  # Selector which must match a node's labels for the pod to be scheduled on that node.
  # More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
  nodeSelector: # Optional
    key: value

  # Global tolerations.
  # If specified, the pod's tolerations.
  # More info: https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/
  tolerations: {} # Optional
  
  # Global compute resources required.
  # Compute Resources required by this component.
  # Cannot be updated.
  # More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/
  resources: # Optional
    # Limits describes the maximum amount of compute resources allowed.
    # More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/
    limits: {} # Optional
    # Requests describes the minimum amount of compute resources required.
    # If Requests is omitted for a container, it defaults to Limits if that is explicitly specified,
    # otherwise to an implementation-defined value.
    # More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/
    requests: {} # Optional
  serviceType: LoadBalancer # Optional: config how the service will publish, default as ClusterIP, see https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services-service-types

  dependencies: {} # Optional, will describe below
  config: {} # Optional, will describe below
```

### Dependencies
specifications for milvus's dependencies:
``` yaml
spec:
  # ... Skipped fields
  dependencies: # Optional
    etcd: {} # Optional
    storage: {} # Optional
```

#### Dependency ETCD
The dependency etcd may be specified as external or in-cluster:
``` yaml
spec:
  # ... Skipped fields
  dependencies: # Optional
    etcd: # Optional
      # Whether (=true) to use an existed external etcd as specified in the field endpoints or 
      # (=false) create a new etcd inside the same kubernetes cluster for milvus.
      external: false # Optional default=false
      # The external etcd endpoints if external=true
      endpoints:
      - 192.168.1.1:2379
      # in-Cluster etcd configuration if external=false
      inCluster: 
        # deletionPolicy of etcd when the milvus cluster is deleted
        deletionPolicy: Retain # Optional ("Delete", "Retain") default="Retain"
        # When deletionPolicy="Delete" whether the PersistantVolumeClaim shoud be deleted when the etcd is deleted
        pvcDeletion: false # Optional default=false
        # ... Skipped fields
    # ... Skipped fields
```

The `inCluster.values` field contains etcd's configurable helm values. For example if you want to deploy etcd in its minimun mode:

``` yaml
spec:
  # ... Skipped fields
  dependencies: # Optional
    etcd: # Optional
      # ... Skipped fields
      inCluster:
        # ... Skipped fields
        values: # Optional
          replicaCount: 1
```

A complete fields doc can be found at https://artifacthub.io/packages/helm/bitnami/etcd/6.3.3.


#### Dependency Storage
The dependency storage may be specified as external or in-cluster. When use in-cluster storage, only `MinIO` storage type is supported.
``` yaml
spec:
  # ... Skipped fields
  dependencies: # Optional
    storage: # Optional
      # Whether (=true) to use an existed external storage as specified in the field endpoints or 
      # (=false) create a new storage inside the same kubernetes cluster for milvus.
      external: false # Optional default=false
      type: "MinIO" # Optional ("MinIO", "S3") default:="MinIO"
      # Secret reference of the storage if it has
      secretRef: mySecret # Optional
      # The external storage endpoint if external=true
      endpoint: "storageEndpoint"
      # in-Cluster storage configuration if external=false
      inCluster: 
        # deletionPolicy of storage when the milvus cluster is deleted
        deletionPolicy: Retain # Optional ("Delete", "Retain") default="Retain"
        # When deletionPolicy="Delete" whether the PersistantVolumeClaim shoud be deleted when the storage is deleted
        pvcDeletion: false # Optional default=false
        # ... Skipped fields
    # ... Skipped fields
```

The `inCluster.values` field contains minIO's configurable helm values. For example if you want to deploy minIO in its minimun mode:

``` yaml
spec:
  # ... Skipped fields
  dependencies: # Optional
    storage: # Optional
      # ... Skipped fields
      inCluster:
        # ... Skipped fields
        values: # Optional
          mode: standalone
```

A complete fields doc can be found at https://github.com/milvus-io/milvus-helm/blob/master/charts/minio/values.yaml.


### Config
Config overrides the fields of Milvus Cluster's config file template. 

For example, if you want to change etcd's rootPath and minIO's bucketname:

``` yaml
spec:
  dependencies: {}
  components: {}
  config: # Optional
    etcd:
      rootPath: my-release
    minio:
      bucketName: my-bucket
```

A complete set of config fields can be found at https://github.com/milvus-io/milvus-operator/blob/main/config/assets/templates/milvus-cluster/milvus.yaml.tmpl. 

NOTE! The fields of dependencies' address and port cannot be set in the Milvus Cluster CR.

## Status spec
The status spec of the CR Milvus is described as below:
``` yaml
status:
  # Show the generous status of the Milvus
  # It can be "Creating", "Healthy", "Unhealthy"
  status: "Healthy"
  # Contains details for the current condition of Milvus and its dependency
  conditions: 
    # Condition type
    # It can be "EtcdReady", "StorageReady", "MsgStream", "MilvusReady"
  - type: "MilvusReady" 
    # Status is the status of the condition.
    # Can be True, False, Unknown.
    status: True
    # Last time the condition transitioned from one status to another.
    lastTransitionTime: <time> # Optional
     # Unique, one-word, CamelCase reason for the condition's last transition.
    reason: "reason" # Optional
    # Human-readable message indicating details about last transition.
    message: "message" # Optional
  # The Milvus's endpoint of service
  endpoint: "milvus:19530"
```