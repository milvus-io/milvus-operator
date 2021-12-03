package controllers

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/milvus-io/milvus-operator/api/v1alpha1"
	"github.com/milvus-io/milvus-operator/pkg/config"
	"github.com/milvus-io/milvus-operator/pkg/util"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestClusterReconciler_ReconcileServices_CreateIfNotExist(t *testing.T) {
	config.Init(util.GetGitRepoRootDir())

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	r := newClusterReconcilerForTest(ctrl)
	mockClient := r.Client.(*MockK8sClient)
	ctx := context.Background()

	m := v1alpha1.MilvusCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "mc",
		},
	}
	m.Default()

	mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(k8sErrors.NewNotFound(schema.GroupResource{}, "mockErr")).Times(5)

	mockClient.EXPECT().Create(gomock.Any(), gomock.Any()).Times(5)

	err := r.ReconcileServices(ctx, m)
	assert.NoError(t, err)
}

func TestClusterReconciler_ReconcileServices_UpdateIfExisted(t *testing.T) {
	config.Init(util.GetGitRepoRootDir())

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	r := newClusterReconcilerForTest(ctrl)
	mockClient := r.Client.(*MockK8sClient)
	ctx := context.Background()

	m := v1alpha1.MilvusCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "mc",
		},
	}
	m.Default()

	mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).
		Do(func(ctx, key, obj interface{}) error {
			s := obj.(*corev1.Service)
			s.Namespace = "ns"
			s.Name = "cm1"
			return nil
		}).Times(5)

	mockClient.EXPECT().Update(gomock.Any(), gomock.Any()).Times(5)

	err := r.ReconcileServices(ctx, m)
	assert.NoError(t, err)
}

func TestMilvusReconciler_ReconcileServices_CreateIfNotExist(t *testing.T) {
	config.Init(util.GetGitRepoRootDir())

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	r := newMilvusReconcilerForTest(ctrl)
	mockClient := r.Client.(*MockK8sClient)
	ctx := context.Background()

	m := v1alpha1.Milvus{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "mc",
		},
	}
	m.Default()

	mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(k8sErrors.NewNotFound(schema.GroupResource{}, "mockErr")).Times(1)

	mockClient.EXPECT().Create(gomock.Any(), gomock.Any()).Times(1)

	err := r.ReconcileServices(ctx, m)
	assert.NoError(t, err)
}

func TestMilvusReconciler_ReconcileServices_UpdateIfExisted(t *testing.T) {
	config.Init(util.GetGitRepoRootDir())

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	r := newMilvusReconcilerForTest(ctrl)
	mockClient := r.Client.(*MockK8sClient)
	ctx := context.Background()

	m := v1alpha1.Milvus{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "mc",
		},
	}
	m.Default()

	mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).
		Do(func(ctx, key, obj interface{}) error {
			s := obj.(*corev1.Service)
			s.Namespace = "ns"
			s.Name = "cm1"
			return nil
		}).Times(1)

	mockClient.EXPECT().Update(gomock.Any(), gomock.Any()).Times(1)

	err := r.ReconcileServices(ctx, m)
	assert.NoError(t, err)
}
