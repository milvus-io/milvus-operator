package v1alpha1

import (
	"testing"

	"github.com/milvus-io/milvus-operator/pkg/config"
	"github.com/stretchr/testify/assert"
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
