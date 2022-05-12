package controllers

import (
	"testing"

	"github.com/milvus-io/milvus-operator/apis/milvus.io/v1alpha1"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestMilvusComponent_IsCoord(t *testing.T) {
	assert.False(t, QueryNode.IsCoord())
	assert.True(t, QueryCoord.IsCoord())
}

func TestMilvusComponent_IsNode(t *testing.T) {
	assert.False(t, QueryCoord.IsNode())
	assert.True(t, QueryNode.IsNode())
}

func TestMergeComponentSpec(t *testing.T) {
	t.Run("merge image", func(t *testing.T) {
		src := ComponentSpec{}
		dst := ComponentSpec{}
		dst.Image = "a"
		merged := MergeComponentSpec(src, dst).Image
		assert.Equal(t, "a", merged)
		src.Image = "b"
		merged = MergeComponentSpec(src, dst).Image
		assert.Equal(t, "b", merged)
	})

	t.Run("merge imagePullPolicy", func(t *testing.T) {
		src := ComponentSpec{}
		dst := ComponentSpec{}
		merged := MergeComponentSpec(src, dst).ImagePullPolicy
		assert.Equal(t, corev1.PullIfNotPresent, *merged)

		always := corev1.PullAlways
		dst.ImagePullPolicy = &always
		merged = MergeComponentSpec(src, dst).ImagePullPolicy
		assert.Equal(t, always, *merged)

		never := corev1.PullNever
		src.ImagePullPolicy = &never
		merged = MergeComponentSpec(src, dst).ImagePullPolicy
		assert.Equal(t, never, *merged)
	})

	t.Run("merge env", func(t *testing.T) {
		src := ComponentSpec{}
		dst := ComponentSpec{}
		src.Env = []corev1.EnvVar{
			{Name: "a"},
		}
		dst.Env = []corev1.EnvVar{
			{Name: "b"},
		}
		merged := MergeComponentSpec(src, dst).Env
		assert.Equal(t, 3, len(merged))
		assert.Equal(t, "CACHE_SIZE", merged[2].Name)
	})

	t.Run("merge imagePullSecret", func(t *testing.T) {
		src := ComponentSpec{}
		dst := ComponentSpec{}
		dst.ImagePullSecrets = []corev1.LocalObjectReference{
			{Name: "a"},
		}
		merged := MergeComponentSpec(src, dst).ImagePullSecrets
		assert.Equal(t, 1, len(merged))
		assert.Equal(t, "a", merged[0].Name)

		src.ImagePullSecrets = []corev1.LocalObjectReference{
			{Name: "b"},
		}
		merged = MergeComponentSpec(src, dst).ImagePullSecrets
		assert.Equal(t, 1, len(merged))
		assert.Equal(t, "b", merged[0].Name)
	})

	t.Run("merge tolerations", func(t *testing.T) {
		src := ComponentSpec{}
		dst := ComponentSpec{}
		dst.Tolerations = []corev1.Toleration{
			{Key: "a"},
		}
		merged := MergeComponentSpec(src, dst).Tolerations
		assert.Equal(t, 1, len(merged))
		assert.Equal(t, "a", merged[0].Key)

		src.Tolerations = []corev1.Toleration{
			{Key: "b"},
		}
		merged = MergeComponentSpec(src, dst).Tolerations
		assert.Equal(t, 1, len(merged))
		assert.Equal(t, "b", merged[0].Key)
	})

	t.Run("merge nodeSelector", func(t *testing.T) {
		src := ComponentSpec{}
		dst := ComponentSpec{}
		dst.NodeSelector = map[string]string{
			"a": "b",
		}
		merged := MergeComponentSpec(src, dst).NodeSelector
		assert.Equal(t, 1, len(merged))
		assert.Equal(t, "b", merged["a"])

		src.NodeSelector = map[string]string{
			"a": "c",
		}
		merged = MergeComponentSpec(src, dst).NodeSelector
		assert.Equal(t, 1, len(merged))
		assert.Equal(t, "c", merged["a"])
	})

	t.Run("merge resources", func(t *testing.T) {
		src := ComponentSpec{}
		dst := ComponentSpec{}
		dst.Resources = &corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				"cpu": resource.MustParse("1"),
			},
			Requests: corev1.ResourceList{
				"cpu": resource.MustParse("1"),
			},
		}
		merged := MergeComponentSpec(src, dst).Resources
		assert.Equal(t, dst.Resources, merged)

		src.Resources = &corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				"a": resource.MustParse("2"),
			},
			Requests: corev1.ResourceList{
				"b": resource.MustParse("2"),
			},
		}
		merged = MergeComponentSpec(src, dst).Resources
		assert.Equal(t, src.Resources, merged)
	})

	t.Run("merge affinity", func(t *testing.T) {
		src := ComponentSpec{}
		dst := ComponentSpec{}
		dst.Affinity = &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{},
		}
		merged := MergeComponentSpec(src, dst).Affinity
		assert.Equal(t, dst.Affinity, merged)
		src.Affinity = &corev1.Affinity{
			PodAntiAffinity: &corev1.PodAntiAffinity{},
		}
		merged = MergeComponentSpec(src, dst).Affinity
		assert.Equal(t, src.Affinity, merged)
	})
}

func TestMilvusComponent_GetReplicas(t *testing.T) {
	// has global, use global
	spec := v1alpha1.MilvusClusterSpec{}
	com := QueryNode
	replica := int32(1)
	spec.Com.QueryNode.Component.Replicas = &replica
	assert.Equal(t, &replica, com.GetReplicas(spec))
}

func TestMilvusComponent_String(t *testing.T) {
	com := QueryNode
	assert.Equal(t, com.Name, com.String())
}

func TestMilvusComponent_GetDeploymentInstanceName(t *testing.T) {
	com := QueryNode
	assert.Equal(t, "inst1-milvus-querynode", com.GetDeploymentInstanceName("inst1"))
}

func TestMilvusComponent_GetSerfviceInstanceName(t *testing.T) {
	assert.Equal(t, "inst1-milvus", GetServiceInstanceName("inst1"))
}

func TestMilvusComponent_GetContainerName(t *testing.T) {
	com := QueryNode
	assert.Equal(t, com.Name, com.GetContainerName())
}

func TestMilvusComponent_GetContainerPorts(t *testing.T) {
	com := QueryNode
	spec := v1alpha1.MilvusClusterSpec{}
	spec.Com.QueryNode.Component.Port = 8080
	ports := com.GetContainerPorts(spec)
	assert.Equal(t, 2, len(ports))
	assert.Equal(t, spec.Com.QueryNode.Component.Port, ports[0].ContainerPort)
	assert.Equal(t, int32(MetricPort), ports[1].ContainerPort)
}

func TestMilvusComponent_GetServiceType(t *testing.T) {
	com := QueryNode
	spec := v1alpha1.MilvusClusterSpec{}
	assert.Equal(t, corev1.ServiceTypeClusterIP, com.GetServiceType(spec))

	com = Proxy
	spec.Com.Proxy.ServiceType = corev1.ServiceTypeNodePort
	assert.Equal(t, corev1.ServiceTypeNodePort, com.GetServiceType(spec))
}

func TestMilvusComponent_GetServicePorts(t *testing.T) {
	com := QueryNode
	spec := v1alpha1.MilvusClusterSpec{}
	ports := com.GetServicePorts(spec)
	assert.Equal(t, 1, len(ports))
	assert.Equal(t, int32(MetricPort), ports[0].Port)

	com = QueryCoord
	spec.Com.QueryCoord.Component.Port = 8080
	ports = com.GetServicePorts(spec)
	assert.Equal(t, 2, len(ports))
	assert.Equal(t, spec.Com.QueryCoord.Component.Port, ports[0].Port)
	assert.Equal(t, int32(MetricPort), ports[1].Port)
}

func TestMilvusComponent_GetComponentPort(t *testing.T) {
	com := QueryNode
	spec := v1alpha1.MilvusClusterSpec{}
	assert.Equal(t, com.DefaultPort, com.GetComponentPort(spec))

	spec.Com.QueryNode.Component.Port = 8080
	assert.Equal(t, spec.Com.QueryNode.Component.Port, com.GetComponentPort(spec))
}

func TestMilvusComponent_GetComponentSpec(t *testing.T) {
	spec := v1alpha1.MilvusClusterSpec{}
	spec.Com.QueryNode.Component.ComponentSpec.Image = "a"
	com := QueryNode
	assert.Equal(t, "a", com.GetComponentSpec(spec).Image)
}

func TestMilvusComponent_GetConfCheckSum(t *testing.T) {
	spec := v1alpha1.MilvusClusterSpec{}
	checksum1 := GetConfCheckSum(spec)

	spec.Conf.Data = map[string]interface{}{
		"k1": "v1",
		"k2": "v2",
		"k3": "v3",
	}
	checksum2 := GetConfCheckSum(spec)
	assert.NotEqual(t, checksum1, checksum2)

	spec.Conf.Data = map[string]interface{}{
		"k3": "v3",
		"k2": "v2",
		"k1": "v1",
	}
	checksum3 := GetConfCheckSum(spec)
	assert.Equal(t, checksum2, checksum3)

	spec.Dep.Kafka.BrokerList = []string{"ep1"}
	spec.Dep.Storage.Endpoint = "ep"
	checksum4 := GetConfCheckSum(spec)
	assert.NotEqual(t, checksum1, checksum4)
}

func TestMilvusComponent_GetMilvusConfCheckSumt(t *testing.T) {
	spec := v1alpha1.MilvusSpec{}
	spec.Conf.Data = map[string]interface{}{
		"k1": "v1",
		"k2": "v2",
		"k3": "v3",
	}
	checksum1 := GetMilvusConfCheckSum(spec)

	spec.Dep.Etcd.Endpoints = []string{"ep1"}
	spec.Dep.Storage.Endpoint = "ep"
	checksum2 := GetMilvusConfCheckSum(spec)
	assert.NotEqual(t, checksum1, checksum2)

	spec.Conf.Data = map[string]interface{}{
		"k3": "v3",
		"k2": "v2",
		"k1": "v1",
	}
	checksum3 := GetMilvusConfCheckSum(spec)
	assert.Equal(t, checksum2, checksum3)
}

func TestMilvusComponent_GetLivenessProbe_GetReadinessProbe(t *testing.T) {
	lProbe := GetLivenessProbe()
	assert.Equal(t, "/healthz", lProbe.HTTPGet.Path)
	assert.Equal(t, intstr.FromInt(MetricPort), lProbe.HTTPGet.Port)

	rProbe := GetReadinessProbe()
	assert.Equal(t, lProbe, rProbe)
}

func TestMilvusComponent_GetDeploymentStrategy(t *testing.T) {
	com := QueryNode
	strategy := com.GetDeploymentStrategy()
	assert.Equal(t, appsv1.RollingUpdateDeploymentStrategyType, strategy.Type)
	assert.Equal(t, intstr.FromInt(0), *strategy.RollingUpdate.MaxUnavailable)
	assert.Equal(t, intstr.FromInt(1), *strategy.RollingUpdate.MaxSurge)

	com = DataCoord
	assert.Equal(t, appsv1.RecreateDeploymentStrategyType, com.GetDeploymentStrategy().Type)
}

func TestMilvusComponent_SetStatusReplica(t *testing.T) {
	com := QueryNode
	status := v1alpha1.MilvusReplicas{}
	com.SetStatusReplicas(&status, 1)
	assert.Equal(t, 1, status.QueryNode)
}

func TestGetInstanceName_GetInstance(t *testing.T) {
	assert.Equal(t, "a", MilvusStandalone.GetDeploymentInstanceName("a"))
	assert.Equal(t, "a-milvus-proxy", Proxy.GetDeploymentInstanceName("a"))
}
