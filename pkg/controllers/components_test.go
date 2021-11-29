package controllers

import (
	"testing"

	"github.com/milvus-io/milvus-operator/api/v1alpha1"
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

func TestMilvusComponent_GetEnv(t *testing.T) {
	spec := v1alpha1.MilvusClusterSpec{}
	spec.Com.ComponentSpec.Env = []corev1.EnvVar{
		{Name: "a"},
	}
	spec.Com.QueryNode.Component.ComponentSpec.Env = []corev1.EnvVar{
		{Name: "b"},
	}
	com := QueryNode
	merged := com.GetEnv(spec)
	assert.Equal(t, 3, len(merged))
	assert.Equal(t, "CACHE_SIZE", merged[2].Name)
}

func TestMilvusComponent_GetImagePullSecrets(t *testing.T) {
	// if empty, use global
	globalSecrets := []corev1.LocalObjectReference{
		{Name: "a"},
	}
	spec := v1alpha1.MilvusClusterSpec{}
	spec.Com.ComponentSpec.ImagePullSecrets = globalSecrets
	com := QueryNode
	secrets := com.GetImagePullSecrets(spec)
	assert.Equal(t, 1, len(secrets))
	assert.Equal(t, "a", secrets[0].Name)

	// if not empty, use specific component's
	specificSecrets := []corev1.LocalObjectReference{
		{Name: "b"},
	}
	spec.Com.QueryNode.Component.ComponentSpec.ImagePullSecrets = specificSecrets
	com = QueryNode
	secrets = com.GetImagePullSecrets(spec)
	assert.Equal(t, 1, len(secrets))
	assert.Equal(t, "b", secrets[0].Name)
}

func TestMilvusComponent_GetImagePullPolicy(t *testing.T) {
	// not set in both global & specific, use default

	spec := v1alpha1.MilvusClusterSpec{}
	com := QueryNode
	policy := com.GetImagePullPolicy(spec)
	assert.Equal(t, corev1.PullIfNotPresent, policy)

	// has only global, use global
	globalPullPolicy := corev1.PullAlways
	spec.Com.ComponentSpec.ImagePullPolicy = &globalPullPolicy
	com = QueryNode
	policy = com.GetImagePullPolicy(spec)
	assert.Equal(t, globalPullPolicy, policy)

	// has both, use specific
	specificPullPolicy := corev1.PullNever
	spec.Com.QueryNode.Component.ComponentSpec.ImagePullPolicy = &specificPullPolicy
	com = QueryNode
	policy = com.GetImagePullPolicy(spec)
	assert.Equal(t, specificPullPolicy, policy)
}

func TestMilvusComponent_GetTolerations(t *testing.T) {
	// only global, use global
	globalTolerations := []corev1.Toleration{
		{Key: "a"},
	}
	spec := v1alpha1.MilvusClusterSpec{
		Com: v1alpha1.MilvusComponents{
			ComponentSpec: v1alpha1.ComponentSpec{
				Tolerations: globalTolerations,
			},
		},
	}
	com := QueryNode
	assert.Equal(t, globalTolerations, com.GetTolerations(spec))

	// has specific, use specific
	specificTolerations := []corev1.Toleration{
		{Key: "b"},
	}
	spec.Com.QueryNode.Component.ComponentSpec.Tolerations = specificTolerations
	assert.Equal(t, specificTolerations, com.GetTolerations(spec))
}

func TestMilvusComponent_GetNodeSelector(t *testing.T) {
	// only global, use global
	globalNodeSelector := map[string]string{
		"a": "b",
	}
	spec := v1alpha1.MilvusClusterSpec{}
	spec.Com.ComponentSpec.NodeSelector = globalNodeSelector
	com := QueryNode
	assert.Equal(t, globalNodeSelector, com.GetNodeSelector(spec))

	// has specific, use specific
	specificNodeSelector := map[string]string{
		"b": "c",
	}
	spec.Com.QueryNode.Component.ComponentSpec.NodeSelector = specificNodeSelector
	assert.Equal(t, specificNodeSelector, com.GetNodeSelector(spec))
}

func TestMilvusComponent_GetResources(t *testing.T) {
	// not set, return empty
	spec := v1alpha1.MilvusClusterSpec{}
	com := QueryNode
	assert.Equal(t, corev1.ResourceRequirements{}, com.GetResources(spec))

	// only global, use global
	globalResources := corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			"cpu": resource.MustParse("100m"),
		},
	}
	spec.Com.ComponentSpec.Resources = &globalResources
	assert.Equal(t, globalResources, com.GetResources(spec))

	// has specific, use specific
	specificResources := corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			"cpu": resource.MustParse("200m"),
		},
	}
	spec.Com.QueryNode.Component.ComponentSpec.Resources = &specificResources
	assert.Equal(t, specificResources, com.GetResources(spec))
}

func TestMilvusComponent_GetImage(t *testing.T) {
	// has global, use global
	spec := v1alpha1.MilvusClusterSpec{}
	spec.Com.ComponentSpec.Image = "a"
	com := QueryNode
	assert.Equal(t, "a", com.GetImage(spec))

	// has specific, use specific
	spec.Com.QueryNode.Component.ComponentSpec.Image = "b"
	assert.Equal(t, "b", com.GetImage(spec))
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

func TestMilvusComponent_GetServiceInstanceName(t *testing.T) {
	com := QueryNode
	assert.Equal(t, "inst1-milvus-querynode", com.GetServiceInstanceName("inst1"))

	com = Proxy
	assert.Equal(t, "inst1-milvus", com.GetServiceInstanceName("inst1"))
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
	com := QueryNode
	spec := v1alpha1.MilvusClusterSpec{}
	checksum1 := com.GetConfCheckSum(spec)

	spec.Conf.Data = map[string]interface{}{
		"k1": "v1",
		"k2": "v2",
		"k3": "v3",
	}
	checksum2 := com.GetConfCheckSum(spec)
	assert.NotEqual(t, checksum1, checksum2)

	spec.Conf.Data = map[string]interface{}{
		"k3": "v3",
		"k2": "v2",
		"k1": "v1",
	}
	checksum3 := com.GetConfCheckSum(spec)
	assert.Equal(t, checksum2, checksum3)
}

func TestMilvusComponent_GetLivenessProbe_GetReadinessProbe(t *testing.T) {
	com := QueryNode
	lProbe := com.GetLivenessProbe()
	assert.Equal(t, "/healthz", lProbe.HTTPGet.Path)
	assert.Equal(t, intstr.FromInt(MetricPort), lProbe.HTTPGet.Port)

	rProbe := com.GetReadinessProbe()
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
