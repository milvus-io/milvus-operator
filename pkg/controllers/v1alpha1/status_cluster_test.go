package controllers

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/milvus-io/milvus-operator/apis/milvus.io/v1alpha1"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func TestClusterStatusSyncer_syncUnhealthy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCli := NewMockK8sClient(ctrl)
	ctx := context.Background()
	logger := logf.Log.WithName("test")
	s := NewMilvusClusterStatusSyncer(ctx, mockCli, logger)

	mockRunner := NewMockGroupRunner(ctrl)
	defaultGroupRunner = mockRunner

	// list failed err
	mockCli.EXPECT().List(gomock.Any(), gomock.Any()).Return(errors.New("test"))
	err := s.syncUnhealthy()
	assert.Error(t, err)

	// status not set, healthy, not run
	mockCli.EXPECT().List(gomock.Any(), gomock.Any()).
		Do(func(ctx context.Context, list *v1alpha1.MilvusClusterList, opts ...client.ListOption) {
			list.Items = []v1alpha1.MilvusCluster{
				{},
				{},
			}
			list.Items[1].Status.Status = v1alpha1.StatusHealthy
		})
	mockRunner.EXPECT().RunDiffArgs(gomock.Any(), gomock.Any(), gomock.Len(0))
	s.syncUnhealthy()

	// status unhealthy, run
	mockCli.EXPECT().List(gomock.Any(), gomock.Any()).
		Do(func(ctx context.Context, list *v1alpha1.MilvusClusterList, opts ...client.ListOption) {
			list.Items = []v1alpha1.MilvusCluster{
				{},
				{},
				{},
			}
			list.Items[1].Status.Status = v1alpha1.StatusUnHealthy
			list.Items[2].Status.Status = v1alpha1.StatusUnHealthy
		})
	mockRunner.EXPECT().RunDiffArgs(gomock.Any(), gomock.Any(), gomock.Len(2))
	s.syncUnhealthy()
}

func TestClusterStatusSyncer_syncHealthy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCli := NewMockK8sClient(ctrl)
	ctx := context.Background()
	logger := logf.Log.WithName("test")
	s := NewMilvusClusterStatusSyncer(ctx, mockCli, logger)

	mockRunner := NewMockGroupRunner(ctrl)
	defaultGroupRunner = mockRunner

	// list failed err
	mockCli.EXPECT().List(gomock.Any(), gomock.Any()).Return(errors.New("test"))
	err := s.syncHealthy()
	assert.Error(t, err)

	// status not set, unhealthy, not run
	mockCli.EXPECT().List(gomock.Any(), gomock.Any()).
		Do(func(ctx context.Context, list *v1alpha1.MilvusClusterList, opts ...client.ListOption) {
			list.Items = []v1alpha1.MilvusCluster{
				{},
				{},
			}
			list.Items[1].Status.Status = v1alpha1.StatusUnHealthy
		})
	mockRunner.EXPECT().RunDiffArgs(gomock.Any(), gomock.Any(), gomock.Len(0))
	s.syncHealthy()

	// status unhealthy, run
	mockCli.EXPECT().List(gomock.Any(), gomock.Any()).
		Do(func(ctx context.Context, list *v1alpha1.MilvusClusterList, opts ...client.ListOption) {
			list.Items = []v1alpha1.MilvusCluster{
				{},
				{},
				{},
			}
			list.Items[1].Status.Status = v1alpha1.StatusHealthy
			list.Items[2].Status.Status = v1alpha1.StatusHealthy
		})
	mockRunner.EXPECT().RunDiffArgs(gomock.Any(), gomock.Any(), gomock.Len(2))
	s.syncHealthy()
}

func TestClusterStatusSyncer_UpdateStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCli := NewMockK8sClient(ctrl)
	ctx := context.Background()
	logger := logf.Log.WithName("test")
	m := &v1alpha1.MilvusCluster{}
	s := NewMilvusClusterStatusSyncer(ctx, mockCli, logger)

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

	m.Status.Status = v1alpha1.StatusCreating
	err = s.UpdateStatus(ctx, m)
	assert.Error(t, err)

	t.Run("update ingress status failed", func(t *testing.T) {
		defer ctrl.Finish()
		mockRunner.EXPECT().RunWithResult(gomock.Len(3), gomock.Any(), gomock.Any()).
			Return([]Result{
				{Data: v1alpha1.MilvusCondition{}},
			})
		mockCli.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("test"))
		m.Status.Status = v1alpha1.StatusCreating
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
				{Data: v1alpha1.MilvusCondition{}},
			})
		mockCli.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		mockReplicaUpdater.EXPECT().UpdateReplicas(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("test"))
		m.Status.Status = v1alpha1.StatusCreating
		err = s.UpdateStatus(ctx, m)
		assert.Error(t, err)
	})

	t.Run("update status success", func(t *testing.T) {
		defer ctrl.Finish()
		mockReplicaUpdater.EXPECT().UpdateReplicas(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		mockRunner.EXPECT().RunWithResult(gomock.Len(3), gomock.Any(), gomock.Any()).
			Return([]Result{
				{Data: v1alpha1.MilvusCondition{}},
			})
		mockCli.EXPECT().Status().Return(mockCli)
		mockCli.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		mockCli.EXPECT().Update(gomock.Any(), gomock.Any())
		m.Status.Status = v1alpha1.StatusCreating
		err = s.UpdateStatus(ctx, m)
		assert.NoError(t, err)
	})
}

func TestClusterStatusSyncer_UpdateReplicas(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCli := NewMockK8sClient(ctrl)
	ctx := context.Background()
	m := &v1alpha1.MilvusCluster{}
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

func TestMilvusStatusSyncer_GetMinioCondition_S3Ready(t *testing.T) {
	m := v1alpha1.Milvus{}
	m.Spec.Dep.Storage.Type = v1alpha1.StorageTypeS3
	ret, err := new(MilvusStatusSyncer).GetMinioCondition(context.TODO(), m)
	assert.NoError(t, err)
	assert.Equal(t, S3ReadyCondition, ret)
}

// mockEndpointCheckCache is for test
type mockEndpointCheckCache struct {
	isUpToDate bool
	condition  *v1alpha1.MilvusCondition
}

func (m *mockEndpointCheckCache) Get(endpoint []string) (condition *v1alpha1.MilvusCondition, isUpToDate bool) {
	return m.condition, m.isUpToDate
}

func (m *mockEndpointCheckCache) Set(endpoints []string, condition *v1alpha1.MilvusCondition) {
	m.condition = condition
}

func mockConditionGetter() v1alpha1.MilvusCondition {
	return v1alpha1.MilvusCondition{Reason: "update"}
}

func TestGetCondition(t *testing.T) {
	bak := endpointCheckCache
	defer func() { endpointCheckCache = bak }()

	t.Run("use cache", func(t *testing.T) {
		condition := v1alpha1.MilvusCondition{Reason: "test"}
		endpointCheckCache = &mockEndpointCheckCache{condition: &condition, isUpToDate: true}
		ret := GetCondition(mockConditionGetter, []string{})
		assert.Equal(t, condition, ret)
	})
	t.Run("not use cache", func(t *testing.T) {
		endpointCheckCache = &mockEndpointCheckCache{condition: nil, isUpToDate: false}
		ret := GetCondition(mockConditionGetter, []string{})
		assert.Equal(t, v1alpha1.MilvusCondition{Reason: "update"}, ret)
	})
}

func TestWrapGetter(t *testing.T) {
	var getter func() v1alpha1.MilvusCondition
	getter = wrapPulsarConditonGetter(nil, nil, v1alpha1.MilvusPulsar{})
	assert.NotNil(t, getter)
	getter = wrapEtcdConditionGetter(nil, []string{})
	assert.NotNil(t, getter)
	getter = wrapMinioConditionGetter(nil, nil, nil, StorageConditionInfo{})
	assert.NotNil(t, getter)
}
