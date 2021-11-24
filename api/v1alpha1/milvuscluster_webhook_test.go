package v1alpha1

import (
	"testing"

	"github.com/milvus-io/milvus-operator/pkg/config"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMilvusCluster_Default_NotExternalOK(t *testing.T) {
	replica1 := int32(1)
	defaultComponent := Component{
		Replicas: &replica1,
	}
	defaultInClusterConfig := &InClusterConfig{
		DeletionPolicy: DeletionPolicyRetain,
		Values: Values{
			Data: map[string]interface{}{},
		},
	}

	var crName = "mc"

	var defaultSpec = MilvusClusterSpec{
		Dep: MilvusDependencies{
			Etcd: MilvusEtcd{
				Endpoints: []string{},
				InCluster: defaultInClusterConfig,
			},
			Pulsar: MilvusPulsar{
				InCluster: defaultInClusterConfig,
			},
			Storage: MilvusStorage{
				Type:      "MinIO",
				SecretRef: crName + "-minio",
				InCluster: defaultInClusterConfig,
			},
		},
		Com: MilvusComponents{
			ComponentSpec: ComponentSpec{
				Image: config.DefaultMilvusImage,
			},
			Proxy: MilvusProxy{
				Component: defaultComponent,
			},
			RootCoord: MilvusRootCoord{
				Component: defaultComponent,
			},
			DataCoord: MilvusDataCoord{
				Component: defaultComponent,
			},
			IndexCoord: MilvusIndexCoord{
				Component: defaultComponent,
			},
			QueryCoord: MilvusQueryCoord{
				Component: defaultComponent,
			},
			DataNode: MilvusDataNode{
				Component: defaultComponent,
			},
			IndexNode: MilvusIndexNode{
				Component: defaultComponent,
			},
			QueryNode: MilvusQueryNode{
				Component: defaultComponent,
			},
		},
		Conf: Values{
			Data: map[string]interface{}{},
		},
	}

	mc := MilvusCluster{ObjectMeta: metav1.ObjectMeta{Name: crName}}
	mc.Default()
	assert.Equal(t, defaultSpec, mc.Spec)
}

func TestMilvusCluster_Default_ExternalDepOK(t *testing.T) {
	var crName = "mc"

	var defaultSpec = MilvusClusterSpec{
		Dep: MilvusDependencies{
			Etcd: MilvusEtcd{
				External: true,
			},
			Pulsar: MilvusPulsar{
				External: true,
			},
			Storage: MilvusStorage{
				External: true,
				Type:     "MinIO",
			},
		},
	}

	mc := MilvusCluster{
		ObjectMeta: metav1.ObjectMeta{Name: crName},
		Spec: MilvusClusterSpec{
			Dep: MilvusDependencies{
				Etcd: MilvusEtcd{
					External: true,
				},
				Pulsar: MilvusPulsar{
					External: true,
				},
				Storage: MilvusStorage{
					External: true,
				},
			},
		},
	}
	mc.Default()
	assert.Equal(t, defaultSpec.Dep, mc.Spec.Dep)
}

func TestMilvusCluster_Default_DeleteUnSetableOK(t *testing.T) {
	var crName = "mc"

	var conf = Values{
		Data: map[string]interface{}{
			"minio": map[string]interface{}{},
		},
	}

	mc := MilvusCluster{
		ObjectMeta: metav1.ObjectMeta{Name: crName},
		Spec: MilvusClusterSpec{
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

func TestMilvusCluster_ValidateCreate_NoError(t *testing.T) {
	mc := MilvusCluster{}
	err := mc.ValidateCreate()
	assert.NoError(t, err)
}

func TestMilvusCluster_ValidateCreate_Invalid1(t *testing.T) {
	mc := MilvusCluster{
		Spec: MilvusClusterSpec{
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

func TestMilvusCluster_ValidateCreate_Invalid3(t *testing.T) {
	mc := MilvusCluster{
		Spec: MilvusClusterSpec{
			Dep: MilvusDependencies{
				Etcd: MilvusEtcd{
					External: true,
				},
				Storage: MilvusStorage{
					External: true,
				},
				Pulsar: MilvusPulsar{
					External: true,
				},
			},
		},
	}
	err := mc.ValidateCreate()
	assert.Error(t, err)
}

func TestMilvusCluster_ValidateUpdate_NoError(t *testing.T) {
	mc := MilvusCluster{}
	err := mc.ValidateUpdate(&mc)
	assert.NoError(t, err)
}

func TestMilvusCluster_ValidateUpdate_Invalid(t *testing.T) {
	new := MilvusCluster{
		Spec: MilvusClusterSpec{
			Dep: MilvusDependencies{
				Etcd: MilvusEtcd{
					External: true,
				},
			},
		},
	}
	old := MilvusCluster{}
	err := new.ValidateUpdate(&old)
	assert.Error(t, err)
}

func TestMilvusCluster_ValidateUpdate_KindAssertionFailed(t *testing.T) {
	new := MilvusCluster{}
	old := appsv1.Deployment{}
	err := new.ValidateUpdate(&old)
	assert.Error(t, err)
}
