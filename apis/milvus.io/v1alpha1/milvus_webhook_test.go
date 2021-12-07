package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMilvus_Default(t *testing.T) {
	r := Milvus{}
	r.Default()
	assert.Equal(t, "MinIO", r.Spec.Dep.Storage.Type)
	assert.NotNil(t, r.Spec.Conf.Data)
	assert.Equal(t, 1, r.Spec.Dep.Etcd.InCluster.Values.Data["replicaCount"])
	assert.Equal(t, "standalone", r.Spec.Dep.Storage.InCluster.Values.Data["mode"])
	assert.Equal(t, "-minio", r.Spec.Dep.Storage.SecretRef)
	assert.Equal(t, DeletionPolicyRetain, r.Spec.Dep.Etcd.InCluster.DeletionPolicy)
	assert.Equal(t, DeletionPolicyRetain, r.Spec.Dep.Storage.InCluster.DeletionPolicy)
}
