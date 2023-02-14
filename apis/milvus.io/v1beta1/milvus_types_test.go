package v1beta1

import (
	"log"
	"testing"
	"time"

	"github.com/milvus-io/milvus-operator/pkg/util"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestGetMilvusConditionByType(t *testing.T) {
	s := &MilvusStatus{}
	ret := GetMilvusConditionByType(s, MilvusReady)
	assert.Nil(t, ret)

	s.Conditions = []MilvusCondition{
		{
			Type: MilvusReady,
		},
	}
	ret = GetMilvusConditionByType(s, MilvusReady)
	assert.NotNil(t, ret)
}

func Test_SetStoppedAtAnnotation_RemoveStoppedAtAnnotation(t *testing.T) {
	myTimeStr := "2020-01-01T00:00:00Z"
	myTime, _ := time.Parse(time.RFC3339, myTimeStr)
	m := &Milvus{}
	InitLabelAnnotation(m)
	m.SetStoppedAtAnnotation(myTime)
	assert.Equal(t, m.GetAnnotations()[StoppedAtAnnotation], myTimeStr)
	m.RemoveStoppedAtAnnotation()
	_, exists := m.GetAnnotations()[StoppedAtAnnotation]
	assert.False(t, exists)
}

func TestComponentDeployStatus_GetState(t *testing.T) {
	c := &ComponentDeployStatus{
		Generation: 1,
	}
	t.Run("no status, progressing", func(t *testing.T) {

		assert.Equal(t, c.GetState(), DeploymentProgressing)
	})

	c.Generation = 2
	c.Status.ObservedGeneration = 1
	t.Run("new generation not observed, progressing", func(t *testing.T) {
		assert.Equal(t, c.GetState(), DeploymentProgressing)
	})

	c.Status.ObservedGeneration = 2
	t.Run("condition progressing not found, progressing", func(t *testing.T) {

		assert.Equal(t, c.GetState(), DeploymentProgressing)
	})

	t.Run("failed", func(t *testing.T) {
		c.Status.Conditions = []appsv1.DeploymentCondition{
			{
				Type:   appsv1.DeploymentProgressing,
				Status: corev1.ConditionFalse,
			},
		}
		assert.Equal(t, c.GetState(), DeploymentFailed)
	})

	t.Run("complete", func(t *testing.T) {
		c.Status.Conditions = []appsv1.DeploymentCondition{
			{
				Type:   appsv1.DeploymentProgressing,
				Status: corev1.ConditionTrue,
				Reason: NewReplicaSetAvailableReason,
			},
		}
		assert.Equal(t, c.GetState(), DeploymentComplete)
	})

	t.Run("progressing", func(t *testing.T) {
		c.Status.Conditions = []appsv1.DeploymentCondition{
			{
				Type:   appsv1.DeploymentProgressing,
				Status: corev1.ConditionTrue,
				Reason: "test",
			},
		}
		assert.Equal(t, c.GetState(), DeploymentProgressing)
	})
}

func TestMilvusSpec_IsStopping(t *testing.T) {
	m := &Milvus{}
	m.Default()
	t.Run("standalone not stopping", func(t *testing.T) {
		assert.False(t, m.Spec.IsStopping())
	})

	replica0 := int32(0)
	com := &m.Spec.Com
	t.Run("standalone stopping", func(t *testing.T) {
		com.Standalone.Replicas = &replica0
		assert.True(t, m.Spec.IsStopping())
	})

	m.Spec.Mode = MilvusModeCluster
	com.MixCoord = &MilvusMixCoord{}
	m.Default()
	log.Print(m)
	com.Proxy.Replicas = &replica0
	com.IndexNode.Replicas = &replica0
	com.DataNode.Replicas = &replica0
	com.QueryNode.Replicas = &replica0
	t.Run("mixcoord not stopping", func(t *testing.T) {
		assert.False(t, m.Spec.IsStopping())
	})

	com.MixCoord.Replicas = &replica0
	t.Run("mixcoord stopping", func(t *testing.T) {
		assert.True(t, m.Spec.IsStopping())
	})

	com.MixCoord = nil
	m.Default()
	t.Run("cluster not stopping", func(t *testing.T) {
		assert.False(t, m.Spec.IsStopping())
	})

	com.RootCoord.Replicas = &replica0
	com.IndexCoord.Replicas = &replica0
	com.DataCoord.Replicas = &replica0
	com.QueryCoord.Replicas = &replica0
	t.Run("cluster stopping", func(t *testing.T) {
		assert.True(t, m.Spec.IsStopping())
	})
}

func TestGetServiceComponent(t *testing.T) {
	m := Milvus{}
	m.Default()
	assert.Equal(t, &m.Spec.Com.Standalone.ServiceComponent, m.Spec.GetServiceComponent())

	m = Milvus{}
	m.Spec.Mode = MilvusModeCluster
	m.Default()
	assert.Equal(t, &m.Spec.Com.Proxy.ServiceComponent, m.Spec.GetServiceComponent())
}

func TestIsRollingUpdateEnabled(t *testing.T) {
	m := Milvus{}
	m.Default()
	assert.False(t, m.IsRollingUpdateEnabled())

	m.Spec.Com.EnableRollingUpdate = util.BoolPtr(false)
	assert.False(t, m.IsRollingUpdateEnabled())

	m.Spec.Com.EnableRollingUpdate = util.BoolPtr(true)
	assert.True(t, m.IsRollingUpdateEnabled())
}
