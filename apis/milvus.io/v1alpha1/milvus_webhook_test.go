package v1alpha1

import (
	"testing"

	"github.com/milvus-io/milvus-operator/pkg/config"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func TestMilvus_Default_NotExternalOK(t *testing.T) {
	defaultInClusterConfig := InClusterConfig{
		DeletionPolicy: DeletionPolicyRetain,
		Values: Values{
			Data: map[string]interface{}{},
		},
	}

	var crName = "m"
	etcdIC := defaultInClusterConfig
	etcdIC.Values.Data = map[string]interface{}{
		"replicaCount": 1,
	}
	storageIC := defaultInClusterConfig
	storageIC.Values.Data = map[string]interface{}{
		"mode": "standalone",
	}

	var defaultSpec = MilvusSpec{
		Dep: MilvusDependencies{
			Etcd: MilvusEtcd{
				Endpoints: []string{},
				InCluster: &etcdIC,
			},
			Storage: MilvusStorage{
				Type:      "MinIO",
				SecretRef: crName + "-minio",
				InCluster: &storageIC,
			},
		},
		ComponentSpec: ComponentSpec{
			Image: config.DefaultMilvusImage,
		},
		Conf: Values{
			Data: map[string]interface{}{},
		},
	}

	mc := Milvus{ObjectMeta: metav1.ObjectMeta{Name: crName}}
	mc.Default()
	assert.Equal(t, defaultSpec, mc.Spec)
}

func TestMilvus_Default_DeleteUnSetableOK(t *testing.T) {
	var crName = "mc"

	var conf = Values{
		Data: map[string]interface{}{
			"minio": map[string]interface{}{},
		},
	}

	mc := Milvus{
		ObjectMeta: metav1.ObjectMeta{Name: crName},
		Spec: MilvusSpec{
			Conf: Values{
				Data: map[string]interface{}{
					"minio": map[string]interface{}{
						"address": "myHost",
					},
				},
			},
		},
	}
	mc.Default()
	assert.Equal(t, conf, mc.Spec.Conf)
}

func TestMilvus_ValidateCreate_NoError(t *testing.T) {
	mc := Milvus{}
	err := mc.ValidateCreate()
	assert.NoError(t, err)
}

func TestMilvus_ValidateCreate_Invalid1(t *testing.T) {
	mc := Milvus{
		Spec: MilvusSpec{
			Dep: MilvusDependencies{
				Etcd: MilvusEtcd{
					External: true,
				},
			},
		},
	}
	err := mc.ValidateCreate()
	assert.Error(t, err)
}

func TestMilvus_ValidateCreate_Invalid3(t *testing.T) {
	mc := Milvus{
		Spec: MilvusSpec{
			Dep: MilvusDependencies{
				Etcd: MilvusEtcd{
					External: true,
				},
				Storage: MilvusStorage{
					External: true,
				},
			},
		},
	}
	err := mc.ValidateCreate()
	assert.Error(t, err)
}

func TestMilvus_ValidateUpdate_NoError(t *testing.T) {
	mc := Milvus{}
	err := mc.ValidateUpdate(&mc)
	assert.NoError(t, err)
}

func TestMilvus_ValidateUpdate_Invalid(t *testing.T) {
	new := Milvus{
		Spec: MilvusSpec{
			Dep: MilvusDependencies{
				Etcd: MilvusEtcd{
					External: true,
				},
			},
		},
	}
	old := Milvus{}
	err := new.ValidateUpdate(&old)
	assert.Error(t, err)
}

func TestMilvus_ValidateUpdate_KindAssertionFailed(t *testing.T) {
	new := Milvus{}
	old := appsv1.Deployment{}
	err := new.ValidateUpdate(&old)
	assert.Error(t, err)
}
