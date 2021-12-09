package controllers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/milvus-io/milvus-operator/apis/milvus.io/v1alpha1"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func TestNamespacedName(t *testing.T) {
	key := NamespacedName("default", "test")
	assert.Equal(t, "default", key.Namespace)
	assert.Equal(t, "test", key.Name)
}

func TestMergeServicePort(t *testing.T) {
	// empty src
	src := []corev1.ServicePort{}
	dst := []corev1.ServicePort{{}}
	ret := MergeServicePort(src, dst)
	assert.Equal(t, dst, ret)

	// empty dst
	src = []corev1.ServicePort{{}}
	dst = []corev1.ServicePort{}
	ret = MergeServicePort(src, dst)
	assert.Equal(t, src, ret)

	// merge same
	src = []corev1.ServicePort{{}}
	dst = []corev1.ServicePort{{}}
	ret = MergeServicePort(src, dst)
	assert.Len(t, ret, 1)

	// merge same
	src = []corev1.ServicePort{{}}
	dst = []corev1.ServicePort{{}}
	ret = MergeServicePort(src, dst)
	assert.Len(t, ret, 1)

	// merge same port
	src = []corev1.ServicePort{{Name: "a", Port: 8080}}
	dst = []corev1.ServicePort{{Name: "b", Port: 8080}}
	ret = MergeServicePort(src, dst)
	assert.Len(t, ret, 1)

	// merge different
	src = []corev1.ServicePort{{Name: "a", Port: 8081}}
	dst = []corev1.ServicePort{{Name: "b", Port: 8080}}
	ret = MergeServicePort(src, dst)
	assert.Len(t, ret, 2)

	// merge same nodePort
	src = []corev1.ServicePort{{Name: "a", NodePort: 3000, Port: 8081}}
	dst = []corev1.ServicePort{{Name: "b", NodePort: 3000, Port: 8080}}
	ret = MergeServicePort(src, dst)
	assert.Len(t, ret, 1)

	// merge different nodePort
	src = []corev1.ServicePort{{Name: "a", NodePort: 3000, Port: 8081}}
	dst = []corev1.ServicePort{{Name: "b", NodePort: 3001, Port: 8080}}
	ret = MergeServicePort(src, dst)
	assert.Len(t, ret, 2)
}

func TestMergeVolumeMount(t *testing.T) {
	// empty src
	src := []corev1.VolumeMount{}
	dst := []corev1.VolumeMount{{}}
	ret := MergeVolumeMount(src, dst)
	assert.Equal(t, dst, ret)

	// empty dst
	src = []corev1.VolumeMount{{}}
	dst = []corev1.VolumeMount{}
	ret = MergeVolumeMount(src, dst)
	assert.Equal(t, src, ret)

	// merge same
	dst = []corev1.VolumeMount{{}}
	src = []corev1.VolumeMount{{}}
	ret = MergeVolumeMount(src, dst)
	assert.Len(t, ret, 1)

	// merge different
	dst = []corev1.VolumeMount{{MountPath: "/p1"}}
	src = []corev1.VolumeMount{{MountPath: "/p2"}}
	ret = MergeVolumeMount(src, dst)
	assert.Len(t, ret, 2)
}

func TestMergeContainerPort(t *testing.T) {
	// empty src
	src := []corev1.ContainerPort{}
	dst := []corev1.ContainerPort{{}}
	ret := MergeContainerPort(src, dst)
	assert.Equal(t, dst, ret)

	// empty dst
	src = []corev1.ContainerPort{{}}
	dst = []corev1.ContainerPort{}
	ret = MergeContainerPort(src, dst)
	assert.Equal(t, src, ret)

	// merge same
	dst = []corev1.ContainerPort{{Name: "a"}}
	src = []corev1.ContainerPort{{Name: "a"}}
	ret = MergeContainerPort(src, dst)
	assert.Len(t, ret, 1)

	// merge different
	dst = []corev1.ContainerPort{{Name: "a"}}
	src = []corev1.ContainerPort{{Name: "b"}}
	ret = MergeContainerPort(src, dst)
	assert.Len(t, ret, 2)
}

func TestMergeEnvVar(t *testing.T) {
	// empty src
	src := []corev1.EnvVar{}
	dst := []corev1.EnvVar{{}}
	ret := MergeEnvVar(src, dst)
	assert.Equal(t, dst, ret)

	// empty dst
	src = []corev1.EnvVar{{}}
	dst = []corev1.EnvVar{}
	ret = MergeEnvVar(src, dst)
	assert.Equal(t, src, ret)

	// merge same
	dst = []corev1.EnvVar{{Name: "a"}}
	src = []corev1.EnvVar{{Name: "a"}}
	ret = MergeEnvVar(src, dst)
	assert.Len(t, ret, 1)

	// merge different
	dst = []corev1.EnvVar{{Name: "a"}}
	src = []corev1.EnvVar{{Name: "b"}}
	ret = MergeEnvVar(src, dst)
	assert.Len(t, ret, 2)
}

func TestGetContainerIndex(t *testing.T) {
	containers := []corev1.Container{
		{Name: "a"},
		{Name: "b"},
	}
	// found
	assert.Equal(t, 0, GetContainerIndex(containers, "a"))
	// not found
	assert.Equal(t, -1, GetContainerIndex(containers, "z"))
}

func TestGetVolumeIndex(t *testing.T) {
	volumes := []corev1.Volume{
		{Name: "a"},
		{Name: "b"},
	}
	// found
	assert.Equal(t, 0, GetVolumeIndex(volumes, "a"))
	// not found
	assert.Equal(t, -1, GetVolumeIndex(volumes, "z"))
}

func TestGetVolumeMountIndex(t *testing.T) {
	volumeMounts := []corev1.VolumeMount{
		{MountPath: "a"},
		{MountPath: "b"},
	}
	// found
	assert.Equal(t, 0, GetVolumeMountIndex(volumeMounts, "a"))
	// not found
	assert.Equal(t, -1, GetVolumeMountIndex(volumeMounts, "z"))
}

func TestNewComponentAppLabels(t *testing.T) {
	labels := NewComponentAppLabels("a", "b")
	assert.Equal(t, "a", labels[AppLabelInstance])
	assert.Equal(t, "b", labels[AppLabelComponent])
	assert.Equal(t, "milvus", labels[AppLabelName])
}

func TestNewAppLabels(t *testing.T) {
	labels := NewAppLabels("a")
	assert.Equal(t, "a", labels[AppLabelInstance])
	assert.Equal(t, "milvus", labels[AppLabelName])
}

func TestMergeLabels(t *testing.T) {
	labels := MergeLabels(map[string]string{"a": "b"}, map[string]string{"c": "d"})
	assert.Equal(t, "b", labels["a"])
	assert.Equal(t, "d", labels["c"])
}

func TestIsEqual(t *testing.T) {
	assert.True(t, IsEqual(map[string]string{"a": "b"}, map[string]string{"a": "b"}))
}

func TestDeploymentReady(t *testing.T) {
	// Ready
	deployment := appsv1.Deployment{
		Status: appsv1.DeploymentStatus{
			Conditions: []appsv1.DeploymentCondition{
				{
					Type:   appsv1.DeploymentAvailable,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
	assert.True(t, DeploymentReady(deployment))

	// Not Ready
	deployment = appsv1.Deployment{
		Status: appsv1.DeploymentStatus{
			Conditions: []appsv1.DeploymentCondition{
				{
					Type:   appsv1.DeploymentAvailable,
					Status: corev1.ConditionFalse,
				},
			},
		},
	}
	assert.False(t, DeploymentReady(deployment))

	// Has Progressing
	deployment = appsv1.Deployment{
		Status: appsv1.DeploymentStatus{
			Conditions: []appsv1.DeploymentCondition{
				{
					Type:   appsv1.DeploymentProgressing,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
	assert.False(t, DeploymentReady(deployment))

	// Has Progressing With NewReplicaSetAvailable Ignored
	deployment = appsv1.Deployment{
		Status: appsv1.DeploymentStatus{
			Conditions: []appsv1.DeploymentCondition{
				{
					Type:   appsv1.DeploymentProgressing,
					Status: corev1.ConditionTrue,
					Reason: "NewReplicaSetAvailable",
				},
			},
		},
	}
	assert.True(t, DeploymentReady(deployment))

	// Has Falure
	deployment = appsv1.Deployment{
		Status: appsv1.DeploymentStatus{
			Conditions: []appsv1.DeploymentCondition{
				{
					Type:   appsv1.DeploymentReplicaFailure,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
	assert.False(t, DeploymentReady(deployment))
}

func TestPodRunningAndReady(t *testing.T) {
	// pending
	pod := corev1.Pod{
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
		},
	}
	ready, err := PodRunningAndReady(pod)
	assert.NoError(t, err)
	assert.False(t, ready)

	// failed
	pod = corev1.Pod{
		Status: corev1.PodStatus{
			Phase: corev1.PodFailed,
		},
	}
	_, err = PodRunningAndReady(pod)
	assert.Error(t, err)

	// running not ready
	pod = corev1.Pod{
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionFalse,
				},
			},
		},
	}
	ready, err = PodRunningAndReady(pod)
	assert.NoError(t, err)
	assert.False(t, ready)

	// running ready
	pod = corev1.Pod{
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
	ready, err = PodRunningAndReady(pod)
	assert.NoError(t, err)
	assert.True(t, ready)
}

func TestGetConditionStatus(t *testing.T) {
	assert.Equal(t, corev1.ConditionFalse, GetConditionStatus(false))
	assert.Equal(t, corev1.ConditionTrue, GetConditionStatus(true))
}

func TestIsClusterDependencyReady(t *testing.T) {
	// 1 not ready -> not ready
	status := v1alpha1.MilvusClusterStatus{
		Conditions: []v1alpha1.MilvusCondition{
			{
				Type:   v1alpha1.EtcdReady,
				Status: corev1.ConditionTrue,
			},
			{
				Type:   v1alpha1.PulsarReady,
				Status: corev1.ConditionTrue,
			},
			{
				Type:   v1alpha1.StorageReady,
				Status: corev1.ConditionFalse,
			},
		},
	}
	assert.False(t, IsClusterDependencyReady(status))
	// all ready -> ready
	status.Conditions[2].Status = corev1.ConditionTrue
	assert.True(t, IsClusterDependencyReady(status))
}

func TestIsDependencyReady(t *testing.T) {
	// 1 not ready -> not ready
	status := v1alpha1.MilvusStatus{
		Conditions: []v1alpha1.MilvusCondition{
			{
				Type:   v1alpha1.EtcdReady,
				Status: corev1.ConditionTrue,
			},
			{
				Type:   v1alpha1.StorageReady,
				Status: corev1.ConditionFalse,
			},
		},
	}
	assert.False(t, IsDependencyReady(status.Conditions, false))
	// all ready -> ready
	status.Conditions[1].Status = corev1.ConditionTrue
	assert.True(t, IsDependencyReady(status.Conditions, false))
}

func TestUpdateClusterCondition(t *testing.T) {
	// append if not existed
	status := v1alpha1.MilvusClusterStatus{
		Conditions: []v1alpha1.MilvusCondition{
			{
				Type:   v1alpha1.StorageReady,
				Status: corev1.ConditionFalse,
			},
		},
	}
	condition := v1alpha1.MilvusCondition{
		Type:    v1alpha1.PulsarReady,
		Status:  corev1.ConditionFalse,
		Reason:  "NotReady",
		Message: "Pulsar is not ready",
	}
	UpdateClusterCondition(&status, condition)
	assert.Len(t, status.Conditions, 2)
	assert.Equal(t, status.Conditions[1].Status, condition.Status)
	assert.Equal(t, status.Conditions[1].Type, condition.Type)

	// update existed
	condition2 := v1alpha1.MilvusCondition{
		Type:    v1alpha1.PulsarReady,
		Status:  corev1.ConditionTrue,
		Reason:  "Ready",
		Message: "Pulsar is ready",
	}
	UpdateClusterCondition(&status, condition2)
	assert.Len(t, status.Conditions, 2)
	assert.Equal(t, status.Conditions[1].Status, condition2.Status)
	assert.Equal(t, status.Conditions[1].Type, condition2.Type)
}

func TestUpdateCondition(t *testing.T) {
	// append if not existed
	status := v1alpha1.MilvusStatus{
		Conditions: []v1alpha1.MilvusCondition{
			{
				Type:   v1alpha1.StorageReady,
				Status: corev1.ConditionFalse,
			},
		},
	}
	condition := v1alpha1.MilvusCondition{
		Type:    v1alpha1.PulsarReady,
		Status:  corev1.ConditionFalse,
		Reason:  "NotReady",
		Message: "Pulsar is not ready",
	}
	UpdateCondition(&status, condition)
	assert.Len(t, status.Conditions, 2)
	assert.Equal(t, status.Conditions[1].Status, condition.Status)
	assert.Equal(t, status.Conditions[1].Type, condition.Type)

	// update existed
	condition2 := v1alpha1.MilvusCondition{
		Type:    v1alpha1.PulsarReady,
		Status:  corev1.ConditionTrue,
		Reason:  "Ready",
		Message: "Pulsar is ready",
	}
	UpdateCondition(&status, condition2)
	assert.Len(t, status.Conditions, 2)
	assert.Equal(t, status.Conditions[1].Status, condition2.Status)
	assert.Equal(t, status.Conditions[1].Type, condition2.Type)
}

func TestGetMinioSecure(t *testing.T) {
	conf := map[string]interface{}{
		"minio": map[string]interface{}{
			"useSSL": true,
		},
	}
	assert.True(t, GetMinioSecure(conf))

	conf = map[string]interface{}{
		"minio": map[string]interface{}{
			"useSSL": false,
		},
	}
	assert.False(t, GetMinioSecure(conf))

	conf = map[string]interface{}{}
	assert.False(t, GetMinioSecure(conf))
}

func TestDiffObject(t *testing.T) {
	obj1 := &v1alpha1.MilvusCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "obj1",
		},
	}
	obj2 := &v1alpha1.MilvusCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "obj2",
		},
	}
	diff, err := diffObject(obj1, obj2)
	assert.NoError(t, err)
	assert.Equal(t, `{"metadata":{"name":"obj2"}}`, string(diff))
}

func TestInt32Ptr(t *testing.T) {
	assert.Equal(t, int32(1), *int32Ptr(1))
	assert.Equal(t, int32(0), *int32Ptr(0))
}

func TestGetFuncName(t *testing.T) {
	assert.Contains(t, getFuncName(getFuncName), "getFuncName")
}

func TestLoopWithInterval(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	callCnt := 0
	loopFunc := func() error {
		callCnt++
		return errors.New("test")
	}
	logger := logf.Log.WithName("test")
	go LoopWithInterval(ctx, loopFunc, time.Second/10, logger)
	time.Sleep(time.Second)
	assert.Less(t, 9, callCnt)
}
