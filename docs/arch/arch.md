# How it works

## Overall design

The diagram shown below describes the overall design of the Milvus operator functionalities.

![overall design](../images/arch.jpg)

A CR `MilvusCluster` is introduced to hold the whole component stack of the deploying Milvus cluster. The Milvus cluster owns all of the Milvus components itself as well as the possible related dependent services including 'etcd', 'minio' and 'pulsar'. The `MilvusCluster` controller takes charge of the reconciling process to make all the Milvus components and the related dependent service can be correctly created, updated and even deleted.

