package controllers

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/milvus-io/milvus-operator/apis/milvus.io/v1alpha1"
	"github.com/milvus-io/milvus-operator/pkg/helm"
	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestLocalHelmReconciler_ReconcilePanic(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	settings := cli.New()
	logger := ctrl.Log.WithName("test")

	ctx := context.TODO()
	request := helm.ChartRequest{}
	rec := NewLocalHelmReconciler(settings, logger)

	// bad driver failed
	os.Setenv("HELM_DRIVER", "bad")
	defer func() {
		os.Unsetenv("HELM_DRIVER")
		if r := recover(); r == nil {
			t.Error("should panic with bad driver")
		}
	}()
	err := rec.Reconcile(ctx, request)
	assert.Error(t, err)
}

func TestLocalHelmReconciler_Reconcile(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockHelm := helm.NewMockClient(mockCtrl)
	helm.SetDefaultClient(mockHelm)

	settings := cli.New()
	logger := ctrl.Log.WithName("test")

	ctx := context.TODO()
	request := helm.ChartRequest{}
	rec := NewLocalHelmReconciler(settings, logger)
	errTest := errors.New("test")

	// ReleaseExist failed
	mockHelm.EXPECT().
		ReleaseExist(gomock.Any(), gomock.Any()).
		Return(false, errTest)
	err := rec.Reconcile(ctx, request)
	assert.Error(t, err)

	// not existed, install pulsar
	request.Chart = PulsarChart
	request.Values = make(map[string]interface{})
	mockHelm.EXPECT().
		ReleaseExist(gomock.Any(), gomock.Any()).
		Return(false, nil)
	mockHelm.EXPECT().
		Install(gomock.Any(), gomock.Any()).DoAndReturn(
		func(cfg *action.Configuration, request helm.ChartRequest) error {
			assert.True(t, request.Values["initialize"].(bool))
			return nil
		})
	err = rec.Reconcile(ctx, request)
	assert.NoError(t, err)

	// existed, get values failed
	request.Values = make(map[string]interface{})
	mockHelm.EXPECT().
		ReleaseExist(gomock.Any(), gomock.Any()).
		Return(true, nil)
	mockHelm.EXPECT().GetValues(gomock.Any(), gomock.Any()).Return(nil, errTest)
	err = rec.Reconcile(ctx, request)
	assert.Error(t, err)

	// existed, get status failed
	request.Values = make(map[string]interface{})
	mockHelm.EXPECT().
		ReleaseExist(gomock.Any(), gomock.Any()).
		Return(true, nil)
	mockHelm.EXPECT().GetValues(gomock.Any(), gomock.Any())
	mockHelm.EXPECT().GetStatus(gomock.Any(), gomock.Any()).Return(release.StatusUnknown, errTest)
	err = rec.Reconcile(ctx, request)
	assert.Error(t, err)

	// existed, not need udpate
	request.Values = make(map[string]interface{})
	mockHelm.EXPECT().
		ReleaseExist(gomock.Any(), gomock.Any()).
		Return(true, nil)
	mockHelm.EXPECT().GetValues(gomock.Any(), gomock.Any()).Return(map[string]interface{}{}, nil)
	mockHelm.EXPECT().GetStatus(gomock.Any(), gomock.Any()).Return(release.StatusDeployed, nil)
	err = rec.Reconcile(ctx, request)
	assert.NoError(t, err)

	// existed, pulsar update
	request.Chart = PulsarChart
	request.Values["val2"] = true
	mockHelm.EXPECT().
		ReleaseExist(gomock.Any(), gomock.Any()).
		Return(true, nil)
	mockHelm.EXPECT().GetValues(gomock.Any(), gomock.Any()).Return(map[string]interface{}{"initialize": true}, nil)
	mockHelm.EXPECT().GetStatus(gomock.Any(), gomock.Any()).Return(release.StatusDeployed, nil)
	mockHelm.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
		func(cfg *action.Configuration, request helm.ChartRequest) error {
			_, has := request.Values["initialize"]
			assert.False(t, has)
			assert.True(t, request.Values["val2"].(bool))
			return nil
		})
	err = rec.Reconcile(ctx, request)
	assert.NoError(t, err)
}

func TestMilvusReconciler_ReconcileDeps(t *testing.T) {
	env := newMilvusTestEnv(t)
	defer env.tearDown()
	r := env.Reconciler
	ctx := env.ctx
	m := env.Inst
	mockHelm := NewMockHelmReconciler(env.Ctrl)
	r.helmReconciler = mockHelm
	icc := new(v1alpha1.InClusterConfig)

	m.Spec.Dep.Etcd.InCluster = icc

	// internal reconcile helm
	mockHelm.EXPECT().Reconcile(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, request helm.ChartRequest) error {
			assert.Equal(t, request.Chart, EtcdChart)
			return nil
		})
	assert.NoError(t, r.ReconcileEtcd(ctx, m))

	// external ignored
	m.Spec.Dep.Etcd.External = true
	assert.NoError(t, r.ReconcileEtcd(ctx, m))

	m.Spec.Dep.Storage.InCluster = icc
	// internal reconcile helm
	mockHelm.EXPECT().Reconcile(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, request helm.ChartRequest) error {
			assert.Equal(t, request.Chart, MinioChart)
			return nil
		})
	assert.NoError(t, r.ReconcileMinio(ctx, m))

	// external ignored
	m.Spec.Dep.Storage.External = true
	assert.NoError(t, r.ReconcileMinio(ctx, m))
}

func TestClusterReconciler_ReconcileDeps(t *testing.T) {
	env := newClusterTestEnv(t)
	defer env.tearDown()
	r := env.Reconciler
	ctx := env.ctx
	m := env.Inst
	mockHelm := NewMockHelmReconciler(env.Ctrl)
	r.helmReconciler = mockHelm
	icc := new(v1alpha1.InClusterConfig)

	m.Spec.Dep.Etcd.InCluster = icc

	// internal reconcile helm
	mockHelm.EXPECT().Reconcile(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, request helm.ChartRequest) error {
			assert.Equal(t, request.Chart, EtcdChart)
			return nil
		})
	assert.NoError(t, r.ReconcileEtcd(ctx, m))

	// external ignored
	m.Spec.Dep.Etcd.External = true
	assert.NoError(t, r.ReconcileEtcd(ctx, m))

	m.Spec.Dep.Storage.InCluster = icc
	// internal reconcile helm
	mockHelm.EXPECT().Reconcile(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, request helm.ChartRequest) error {
			assert.Equal(t, request.Chart, MinioChart)
			return nil
		})
	assert.NoError(t, r.ReconcileMinio(ctx, m))

	// external ignored
	m.Spec.Dep.Storage.External = true
	assert.NoError(t, r.ReconcileMinio(ctx, m))

	m.Spec.Dep.Pulsar.InCluster = icc
	// internal reconcile helm
	mockHelm.EXPECT().Reconcile(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, request helm.ChartRequest) error {
			assert.Equal(t, request.Chart, PulsarChart)
			return nil
		})
	assert.NoError(t, r.ReconcilePulsar(ctx, m))

	// external ignored
	m.Spec.Dep.Pulsar.External = true
	assert.NoError(t, r.ReconcilePulsar(ctx, m))
}
