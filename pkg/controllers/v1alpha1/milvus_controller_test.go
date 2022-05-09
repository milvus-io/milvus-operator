package controllers

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/milvus-io/milvus-operator/apis/milvus.io/v1alpha1"
	"github.com/milvus-io/milvus-operator/pkg/config"
	"github.com/milvus-io/milvus-operator/pkg/util"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestMilvusReconciler_ReconcileFinalizer(t *testing.T) {
	config.Init(util.GetGitRepoRootDir())

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	r := newMilvusReconcilerForTest(ctrl)
	r.statusSyncer = &MilvusStatusSyncer{}
	// syncer need not to run in this test
	r.statusSyncer.Once.Do(func() {})
	// globalCommonInfo need not to run in this test
	globalCommonInfo.once.Do(func() {})
	mockClient := r.Client.(*MockK8sClient)

	// case create
	m := v1alpha1.Milvus{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "mc",
		},
	}
	m.Default()

	ctx := context.Background()

	mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).
		Do(func(ctx, key, obj interface{}) {
			o := obj.(*v1alpha1.Milvus)
			*o = m
		}).
		Return(nil)

	mockClient.EXPECT().Update(gomock.Any(), gomock.Any()).Do(
		func(ctx, obj interface{}, opts ...interface{}) {
			u := obj.(*v1alpha1.Milvus)
			// finalizer should be added
			assert.Equal(t, u.Finalizers, []string{MilvusFinalizerName})
		},
	).Return(nil)

	m.Finalizers = []string{MilvusFinalizerName}
	_, err := r.Reconcile(ctx, reconcile.Request{})
	assert.NoError(t, err)

	// case delete remove finalizer
	m.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Now()}
	mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).
		Do(func(ctx, key, obj interface{}) {
			o := obj.(*v1alpha1.Milvus)
			*o = m
		}).
		Return(nil)

	mockClient.EXPECT().Update(gomock.Any(), gomock.Any()).Do(
		func(ctx, obj interface{}, opts ...interface{}) {
			u := obj.(*v1alpha1.Milvus)
			// finalizer should be removed
			assert.Equal(t, u.Finalizers, []string{})
		},
	).Return(nil)

	_, err = r.Reconcile(ctx, reconcile.Request{})
	assert.NoError(t, err)
}
