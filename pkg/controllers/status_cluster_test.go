package controllers

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/milvus-io/milvus-operator/apis/milvus.io/v1beta1"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func TestMilvusStatusSyncer_UpdateIngressStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCli := NewMockK8sClient(ctrl)
	ctx := context.Background()
	logger := logf.Log.WithName("test")
	s := NewMilvusStatusSyncer(ctx, mockCli, logger)

	milvus := v1beta1.Milvus{}
	milvus.Default()

	t.Run("no ingress configed", func(t *testing.T) {
		err := s.UpdateIngressStatus(ctx, &milvus)
		assert.NoError(t, err)
	})

	t.Run("get ingress failed", func(t *testing.T) {
		milvus.Spec.Com.Standalone.Ingress = &v1beta1.MilvusIngress{}
		mockCli.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("test"))
		err := s.UpdateIngressStatus(ctx, &milvus)
		assert.Error(t, err)
	})
	t.Run("get ingress not found ok", func(t *testing.T) {
		milvus.Spec.Com.Standalone.Ingress = &v1beta1.MilvusIngress{}
		mockCli.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(kerrors.NewNotFound(schema.GroupResource{}, ""))
		err := s.UpdateIngressStatus(ctx, &milvus)
		assert.NoError(t, err)
	})
	t.Run("get ingress found, status copied", func(t *testing.T) {
		milvus.Spec.Com.Standalone.Ingress = &v1beta1.MilvusIngress{}
		mockCli.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).
			Do(func(_, _ interface{}, obj *networkv1.Ingress) {
				obj.Status.LoadBalancer.Ingress = []corev1.LoadBalancerIngress{
					{
						Hostname: "host1",
					},
				}
			}).Return(nil)
		err := s.UpdateIngressStatus(ctx, &milvus)
		assert.NoError(t, err)
		assert.Equal(t, "host1", milvus.Status.IngressStatus.LoadBalancer.Ingress[0].Hostname)
	})
}

func TestMilvusStatusSyncer_GetDependencyCondition(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCli := NewMockK8sClient(ctrl)
	ctx := context.Background()
	logger := logf.Log.WithName("test")
	s := NewMilvusStatusSyncer(ctx, mockCli, logger)
	milvus := v1beta1.Milvus{}
	milvus.Spec.Conf.Data = map[string]interface{}{}
	t.Run("GetMinioCondition", func(t *testing.T) {
		defer ctrl.Finish()
		mockCli.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("test"))
		ret, err := s.GetMinioCondition(ctx, milvus)
		assert.NoError(t, err)
		assert.Equal(t, corev1.ConditionFalse, ret.Status)
	})
	t.Run("GetEtcdCondition", func(t *testing.T) {
		defer ctrl.Finish()
		ret, err := s.GetEtcdCondition(ctx, milvus)
		assert.NoError(t, err)
		assert.Equal(t, corev1.ConditionFalse, ret.Status)
	})
	t.Run("GetMsgStreamCondition_pulsar", func(t *testing.T) {
		defer ctrl.Finish()
		ret, err := s.GetMsgStreamCondition(ctx, milvus)
		assert.NoError(t, err)
		assert.Equal(t, corev1.ConditionFalse, ret.Status)
	})
	t.Run("GetMsgStreamCondition_kafka", func(t *testing.T) {
		defer ctrl.Finish()
		milvus.Spec.Dep.MsgStreamType = v1beta1.MsgStreamTypeKafka
		ret, err := s.GetMsgStreamCondition(ctx, milvus)
		assert.NoError(t, err)
		assert.Equal(t, corev1.ConditionFalse, ret.Status)
	})
	t.Run("GetMsgStreamCondition_rocksmq", func(t *testing.T) {
		defer ctrl.Finish()
		milvus.Spec.Dep.MsgStreamType = v1beta1.MsgStreamTypeRocksMQ
		ret, err := s.GetMsgStreamCondition(ctx, milvus)
		assert.NoError(t, err)
		assert.Equal(t, corev1.ConditionTrue, ret.Status)
	})
}

func TestStatusSyncer_syncUnhealthy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCli := NewMockK8sClient(ctrl)
	ctx := context.Background()
	logger := logf.Log.WithName("test")
	s := NewMilvusStatusSyncer(ctx, mockCli, logger)

	mockRunner := NewMockGroupRunner(ctrl)
	defaultGroupRunner = mockRunner

	// list failed err
	mockCli.EXPECT().List(gomock.Any(), gomock.Any()).Return(errors.New("test"))
	err := s.syncUnhealthy()
	assert.Error(t, err)

	// status not set, healthy, not run
	mockCli.EXPECT().List(gomock.Any(), gomock.Any()).
		Do(func(ctx context.Context, list *v1beta1.MilvusList, opts ...client.ListOption) {
			list.Items = []v1beta1.Milvus{
				{},
				{},
			}
			list.Items[1].Status.Status = v1beta1.StatusHealthy
		})
	mockRunner.EXPECT().RunDiffArgs(gomock.Any(), gomock.Any(), gomock.Len(0))
	s.syncUnhealthy()

	// status unhealthy, run
	mockCli.EXPECT().List(gomock.Any(), gomock.Any()).
		Do(func(ctx context.Context, list *v1beta1.MilvusList, opts ...client.ListOption) {
			list.Items = []v1beta1.Milvus{
				{},
				{},
				{},
			}
			list.Items[1].Status.Status = v1beta1.StatusUnHealthy
			list.Items[2].Status.Status = v1beta1.StatusUnHealthy
		})
	mockRunner.EXPECT().RunDiffArgs(gomock.Any(), gomock.Any(), gomock.Len(2))
	s.syncUnhealthy()
}

func TestStatusSyncer_syncHealthy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCli := NewMockK8sClient(ctrl)
	ctx := context.Background()
	logger := logf.Log.WithName("test")
	s := NewMilvusStatusSyncer(ctx, mockCli, logger)

	mockRunner := NewMockGroupRunner(ctrl)
	defaultGroupRunner = mockRunner

	// list failed err
	mockCli.EXPECT().List(gomock.Any(), gomock.Any()).Return(errors.New("test"))
	err := s.syncHealthy()
	assert.Error(t, err)

	// status not set, unhealthy, not run
	mockCli.EXPECT().List(gomock.Any(), gomock.Any()).
		Do(func(ctx context.Context, list *v1beta1.MilvusList, opts ...client.ListOption) {
			list.Items = []v1beta1.Milvus{
				{},
				{},
			}
			list.Items[1].Status.Status = v1beta1.StatusUnHealthy
		})
	mockRunner.EXPECT().RunDiffArgs(gomock.Any(), gomock.Any(), gomock.Len(0))
	s.syncHealthy()

	// status unhealthy, run
	mockCli.EXPECT().List(gomock.Any(), gomock.Any()).
		Do(func(ctx context.Context, list *v1beta1.MilvusList, opts ...client.ListOption) {
			list.Items = []v1beta1.Milvus{
				{},
				{},
				{},
			}
			list.Items[1].Status.Status = v1beta1.StatusHealthy
			list.Items[2].Status.Status = v1beta1.StatusHealthy
		})
	mockRunner.EXPECT().RunDiffArgs(gomock.Any(), gomock.Any(), gomock.Len(2))
	s.syncHealthy()
}

func TestStatusSyncer_UpdateStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCli := NewMockK8sClient(ctrl)
	ctx := context.Background()
	logger := logf.Log.WithName("test")
	m := &v1beta1.Milvus{}
	s := NewMilvusStatusSyncer(ctx, mockCli, logger)

	// default status not set
	err := s.UpdateStatus(ctx, m)
	assert.NoError(t, err)

	// get condition failed
	mockRunner := NewMockGroupRunner(ctrl)
	defaultGroupRunner = mockRunner
	mockRunner.EXPECT().RunWithResult(gomock.Len(3), gomock.Any(), gomock.Any()).
		Return([]Result{
			{Err: errors.New("test")},
			{Err: errors.New("test")},
		})

	m.Status.Status = v1beta1.StatusCreating
	err = s.UpdateStatus(ctx, m)
	assert.Error(t, err)

	t.Run("update ingress status failed", func(t *testing.T) {
		defer ctrl.Finish()
		mockRunner.EXPECT().RunWithResult(gomock.Len(3), gomock.Any(), gomock.Any()).
			Return([]Result{
				{Data: v1beta1.MilvusCondition{}},
			})
		mockCli.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("test"))
		m.Status.Status = v1beta1.StatusCreating
		err = s.UpdateStatus(ctx, m)
		assert.Error(t, err)
	})

	mockReplicaUpdater := NewMockreplicaUpdaterInterface(ctrl)
	backup := replicaUpdater
	replicaUpdater = mockReplicaUpdater
	defer func() {
		replicaUpdater = backup
	}()
	t.Run("update ingress status nil, update replica failed", func(t *testing.T) {
		defer ctrl.Finish()
		mockRunner.EXPECT().RunWithResult(gomock.Len(3), gomock.Any(), gomock.Any()).
			Return([]Result{
				{Data: v1beta1.MilvusCondition{}},
			})
		mockCli.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		mockReplicaUpdater.EXPECT().UpdateReplicas(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("test"))
		m.Status.Status = v1beta1.StatusCreating
		err = s.UpdateStatus(ctx, m)
		assert.Error(t, err)
	})

	t.Run("update status success", func(t *testing.T) {
		defer ctrl.Finish()
		mockReplicaUpdater.EXPECT().UpdateReplicas(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		mockRunner.EXPECT().RunWithResult(gomock.Len(3), gomock.Any(), gomock.Any()).
			Return([]Result{
				{Data: v1beta1.MilvusCondition{}},
			})
		mockCli.EXPECT().Status().Return(mockCli)
		mockCli.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		mockCli.EXPECT().Update(gomock.Any(), gomock.Any())
		m.Status.Status = v1beta1.StatusCreating
		err = s.UpdateStatus(ctx, m)
		assert.NoError(t, err)
	})
}

func TestStatusSyncer_UpdateReplicas(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCli := NewMockK8sClient(ctrl)
	ctx := context.Background()
	m := &v1beta1.Milvus{}
	s := new(replicaUpdaterImpl)

	t.Run("all ok", func(t *testing.T) {
		mockCli.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(_, _, deploy interface{}) {
			deploy.(*appsv1.Deployment).Status.Replicas = 2
		}).Return(nil).Times(len(MilvusComponents))
		err := s.UpdateReplicas(ctx, m, mockCli)
		assert.NoError(t, err)
		assert.Equal(t, 2, m.Status.Replicas.Proxy)
		assert.Equal(t, 2, m.Status.Replicas.DataNode)
	})

	t.Run("components not found ok", func(t *testing.T) {
		mockCli.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(kerrors.NewNotFound(schema.GroupResource{}, "")).Times(len(MilvusComponents))
		err := s.UpdateReplicas(ctx, m, mockCli)
		assert.NoError(t, err)
		assert.Equal(t, 0, m.Status.Replicas.Proxy)
		assert.Equal(t, 0, m.Status.Replicas.DataNode)
	})
	t.Run("get deploy err", func(t *testing.T) {
		mockCli.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(kerrors.NewServiceUnavailable("")).Times(1)
		err := s.UpdateReplicas(ctx, m, mockCli)
		assert.Error(t, err)
	})
}

// mockEndpointCheckCache is for test
type mockEndpointCheckCache struct {
	isUpToDate bool
	condition  *v1beta1.MilvusCondition
}

func (m *mockEndpointCheckCache) Get(endpoint []string) (condition *v1beta1.MilvusCondition, isUpToDate bool) {
	return m.condition, m.isUpToDate
}

func (m *mockEndpointCheckCache) Set(endpoints []string, condition *v1beta1.MilvusCondition) {
	m.condition = condition
}

func mockConditionGetter() v1beta1.MilvusCondition {
	return v1beta1.MilvusCondition{Reason: "update"}
}

func TestWrapGetter(t *testing.T) {
	var getter func() v1beta1.MilvusCondition
	getter = wrapPulsarConditonGetter(nil, nil, v1beta1.MilvusPulsar{})
	assert.NotNil(t, getter)
	getter = wrapEtcdConditionGetter(nil, []string{})
	assert.NotNil(t, getter)
	getter = wrapMinioConditionGetter(nil, nil, nil, StorageConditionInfo{})
	assert.NotNil(t, getter)
}

func Test_updateMetrics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockCli := NewMockK8sClient(ctrl)
	ctx := context.Background()
	logger := logf.Log.WithName("test")
	s := NewMilvusStatusSyncer(ctx, mockCli, logger)

	t.Run("list failed", func(t *testing.T) {
		defer ctrl.Finish()
		mockCli.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("test"))
		err := s.updateMetrics()
		assert.Error(t, err)
	})

	t.Run("success with correct count", func(t *testing.T) {
		defer ctrl.Finish()
		mockCli.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).Do(func(_, listType interface{}, _ ...interface{}) {
			list := listType.(*v1beta1.MilvusList)
			milvusHealthy := v1beta1.Milvus{}
			milvusHealthy.Status.Status = v1beta1.StatusHealthy
			milvusUnhealthy := v1beta1.Milvus{}
			milvusUnhealthy.Status.Status = v1beta1.StatusUnHealthy
			milvusCreating := v1beta1.Milvus{}
			milvusCreating.Status.Status = v1beta1.StatusCreating
			milvusDeleting := v1beta1.Milvus{}
			milvusDeleting.Status.Status = v1beta1.StatusDeleting
			list.Items = []v1beta1.Milvus{
				milvusHealthy,
				milvusHealthy,
				milvusUnhealthy,
				milvusCreating,
				milvusDeleting,
			}
		}).Return(nil)
		err := s.updateMetrics()
		assert.NoError(t, err)
		assert.Equal(t, 2, healthyCount)
		assert.Equal(t, 1, unhealthyCount)
		assert.Equal(t, 1, creatingCount)
		assert.Equal(t, 1, deletingCount)
	})
}
