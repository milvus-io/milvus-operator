package v1beta1

import (
	"testing"

	"github.com/milvus-io/milvus-operator/pkg/config"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMilvus_Default_NotExternal(t *testing.T) {
	// prepare default
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

	var standaloneDefault = MilvusSpec{
		Mode: MilvusModeStandalone,
		Dep: MilvusDependencies{
			Etcd: MilvusEtcd{
				Endpoints: []string{"mc-etcd.default:2379"},
				InCluster: defaultInClusterConfig,
			},
			MsgStreamType: MsgStreamTypeRocksMQ,
			Storage: MilvusStorage{
				Type:      "MinIO",
				Endpoint:  "mc-minio.default:9000",
				SecretRef: crName + "-minio",
				InCluster: defaultInClusterConfig,
			},
		},
		Com: MilvusComponents{
			ComponentSpec: ComponentSpec{
				Image: config.DefaultMilvusImage,
			},
			Standalone: &MilvusStandalone{
				ServiceComponent: ServiceComponent{
					Component: defaultComponent,
				},
			},
		},
		Conf: Values{
			Data: map[string]interface{}{},
		},
	}

	t.Run("standalone not external ok", func(t *testing.T) {
		mc := Milvus{ObjectMeta: metav1.ObjectMeta{Name: crName}}
		mc.Spec.Mode = MilvusModeStandalone
		mc.Default()
		assert.Equal(t, standaloneDefault, mc.Spec)
	})

	t.Run("standalone already set default ok", func(t *testing.T) {
		mc := Milvus{ObjectMeta: metav1.ObjectMeta{Name: crName}}
		mc.Spec.Mode = MilvusModeStandalone
		mc.Default()
		assert.Equal(t, int32(1), *mc.Spec.Com.Standalone.Replicas)
		newReplica := int32(2)
		mc.Spec.Com.Standalone.Replicas = &newReplica
		mc.Default()
		assert.Equal(t, newReplica, *mc.Spec.Com.Standalone.Replicas)
	})

	clusterDefault := *standaloneDefault.DeepCopy()
	clusterDefault.Mode = MilvusModeCluster
	clusterDefault.Dep.MsgStreamType = MsgStreamTypePulsar
	clusterDefault.Dep.Pulsar = MilvusPulsar{
		Endpoint:  "mc-pulsar-proxy.default:6650",
		InCluster: defaultInClusterConfig,
	}
	clusterDefault.Com = MilvusComponents{
		ComponentSpec: ComponentSpec{
			Image: config.DefaultMilvusImage,
		},
		Proxy: &MilvusProxy{
			ServiceComponent: ServiceComponent{
				Component: defaultComponent,
			},
		},
		RootCoord: &MilvusRootCoord{
			Component: defaultComponent,
		},
		DataCoord: &MilvusDataCoord{
			Component: defaultComponent,
		},
		IndexCoord: &MilvusIndexCoord{
			Component: defaultComponent,
		},
		QueryCoord: &MilvusQueryCoord{
			Component: defaultComponent,
		},
		DataNode: &MilvusDataNode{
			Component: defaultComponent,
		},
		IndexNode: &MilvusIndexNode{
			Component: defaultComponent,
		},
		QueryNode: &MilvusQueryNode{
			Component: defaultComponent,
		},
	}
	t.Run("standalone not external ok", func(t *testing.T) {
		mc := Milvus{ObjectMeta: metav1.ObjectMeta{Name: crName}}
		mc.Spec.Mode = MilvusModeCluster
		mc.Default()
		assert.Equal(t, clusterDefault, mc.Spec)
	})

}

func TestMilvus_Default_ExternalDepOK(t *testing.T) {
	var crName = "mc"

	var defaultSpec = MilvusSpec{
		Dep: MilvusDependencies{
			Etcd: MilvusEtcd{
				External: true,
			},
			MsgStreamType: MsgStreamTypePulsar,
			Pulsar: MilvusPulsar{
				External: true,
			},
			Storage: MilvusStorage{
				External: true,
				Type:     "MinIO",
			},
		},
	}

	mc := Milvus{
		ObjectMeta: metav1.ObjectMeta{Name: crName},
		Spec: MilvusSpec{
			Mode: MilvusModeCluster,
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

func TestMilvus_Default_DeleteUnSetableOK(t *testing.T) {
	var crName = "mc"

	var conf = Values{
		Data: map[string]interface{}{
			"minio": map[string]interface{}{
				"conf": "value",
			},
		},
	}

	mc := Milvus{
		ObjectMeta: metav1.ObjectMeta{Name: crName},
		Spec: MilvusSpec{
			Conf: Values{
				Data: map[string]interface{}{
					"minio": map[string]interface{}{
						"address": "myHost",
						"conf":    "value",
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
				Pulsar: MilvusPulsar{
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

func Test_DefaultLabels_Legacy(t *testing.T) {
	new := Milvus{}
	new.Status.Status = StatusHealthy
	new.DefaultMeta()
	assert.Equal(t, new.Labels[OperatorVersionLabel], LegacyVersion)
}
