package controllers

import (
	"context"
	"errors"
	"testing"

	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/golang/mock/gomock"
	"github.com/milvus-io/milvus-operator/api/v1alpha1"
	"github.com/milvus-io/milvus-operator/pkg/config"
	"github.com/milvus-io/milvus-operator/pkg/util"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlRuntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func newClusterReconcilerForTest(ctrl *gomock.Controller) *MilvusClusterReconciler {
	mockClient := NewMockK8sClient(ctrl)

	logger := ctrlRuntime.Log.WithName("test")
	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)
	r := MilvusClusterReconciler{
		Client: mockClient,
		logger: logger,
		Scheme: scheme,
	}
	return &r
}

func TestReconcileConfigMaps_CreateIfNotfound(t *testing.T) {
	config.Init(util.GetGitRepoRootDir())

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	r := newClusterReconcilerForTest(ctrl)
	mockClient := r.Client.(*MockK8sClient)

	mc := v1alpha1.MilvusCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "n",
		},
	}

	ctx := context.Background()

	// all ok
	gomock.InOrder(
		mockClient.EXPECT().
			Get(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&corev1.ConfigMap{})).
			Return(k8sErrors.NewNotFound(schema.GroupResource{}, "")),
		// get secret of minio
		mockClient.EXPECT().
			Get(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&corev1.Secret{})).
			Return(k8sErrors.NewNotFound(schema.GroupResource{}, "mockErr")),
		mockClient.EXPECT().
			Create(gomock.Any(), gomock.Any()).Return(nil),
	)
	err := r.ReconcileConfigMaps(ctx, mc)
	assert.NoError(t, err)

	// get failed
	mockClient.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&corev1.ConfigMap{})).
		Return(errors.New("some network issue"))
	err = r.ReconcileConfigMaps(ctx, mc)
	assert.Error(t, err)

	// get failed
	mockClient.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&corev1.ConfigMap{})).
		Return(errors.New("some network issue"))
	err = r.ReconcileConfigMaps(ctx, mc)
	assert.Error(t, err)
}

func TestReconcileConfigMaps_UpdateIfExisted_OK(t *testing.T) {
	config.Init(util.GetGitRepoRootDir())

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	r := newClusterReconcilerForTest(ctrl)
	mockClient := r.Client.(*MockK8sClient)

	mc := v1alpha1.MilvusCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "mc",
		},
	}

	ctx := context.Background()

	// call client.Update if changed configmap
	gomock.InOrder(
		mockClient.EXPECT().
			Get(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&corev1.ConfigMap{})).
			DoAndReturn(func(ctx context.Context, key client.ObjectKey, obj client.Object) error {
				cm := obj.(*corev1.ConfigMap)
				cm.Namespace = "ns"
				cm.Name = "cm1"
				return nil
			}),
		// get secret of minio
		mockClient.EXPECT().
			Get(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&corev1.Secret{})).
			Return(k8sErrors.NewNotFound(schema.GroupResource{}, "mockErr")),
		mockClient.EXPECT().
			Update(gomock.Any(), gomock.Any()).Return(nil),
	)

	err := r.ReconcileConfigMaps(ctx, mc)
	assert.NoError(t, err)

	// not call client.Update if configmap not changed
	gomock.InOrder(
		mockClient.EXPECT().
			Get(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&corev1.ConfigMap{})).
			DoAndReturn(func(ctx context.Context, key client.ObjectKey, obj client.Object) error {
				cm := obj.(*corev1.ConfigMap)
				cm.Namespace = "ns"
				cm.Name = "cm1"
				r.updateConfigMap(ctx, mc, cm)
				return nil
			}),
		// get secret of minio
		mockClient.EXPECT().
			Get(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&corev1.Secret{})).
			Return(k8sErrors.NewNotFound(schema.GroupResource{}, "mockErr")).Times(2),
	)
	err = r.ReconcileConfigMaps(ctx, mc)
	assert.NoError(t, err)
}

// ---------------- Test Milvus Reconciler ----------------
func newMilvusReconcilerForTest(ctrl *gomock.Controller) *MilvusReconciler {
	mockClient := NewMockK8sClient(ctrl)

	logger := ctrlRuntime.Log.WithName("test")
	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)
	r := MilvusReconciler{
		Client: mockClient,
		logger: logger,
		Scheme: scheme,
	}
	return &r
}

func TestMilvusReconciler_ReconcileConfigMaps_CreateIfNotFound(t *testing.T) {
	config.Init(util.GetGitRepoRootDir())

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	r := newMilvusReconcilerForTest(ctrl)
	mockClient := r.Client.(*MockK8sClient)

	m := v1alpha1.Milvus{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "n",
		},
	}

	ctx := context.Background()

	// all ok
	gomock.InOrder(
		mockClient.EXPECT().
			Get(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&corev1.ConfigMap{})).
			Return(k8sErrors.NewNotFound(schema.GroupResource{}, "")),
		// get secret of minio
		mockClient.EXPECT().
			Get(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&corev1.Secret{})).
			Return(k8sErrors.NewNotFound(schema.GroupResource{}, "mockErr")),
		mockClient.EXPECT().
			Create(gomock.Any(), gomock.Any()).Return(nil),
	)
	err := r.ReconcileConfigMaps(ctx, m)
	assert.NoError(t, err)

	// get failed
	mockClient.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&corev1.ConfigMap{})).
		Return(errors.New("some network issue"))
	err = r.ReconcileConfigMaps(ctx, m)
	assert.Error(t, err)

	// get failed
	mockClient.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&corev1.ConfigMap{})).
		Return(errors.New("some network issue"))
	err = r.ReconcileConfigMaps(ctx, m)
	assert.Error(t, err)
}

func TestMilvusReconciler_ReconcileConfigMaps_UpdateIfExisted(t *testing.T) {
	config.Init(util.GetGitRepoRootDir())

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	r := newMilvusReconcilerForTest(ctrl)
	mockClient := r.Client.(*MockK8sClient)

	m := v1alpha1.Milvus{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "mc",
		},
	}

	ctx := context.Background()

	// call client.Update if changed configmap
	gomock.InOrder(
		mockClient.EXPECT().
			Get(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&corev1.ConfigMap{})).
			DoAndReturn(func(ctx context.Context, key client.ObjectKey, obj client.Object) error {
				cm := obj.(*corev1.ConfigMap)
				cm.Namespace = "ns"
				cm.Name = "cm1"
				return nil
			}),
		// get secret of minio
		mockClient.EXPECT().
			Get(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&corev1.Secret{})).
			Return(k8sErrors.NewNotFound(schema.GroupResource{}, "mockErr")),
		mockClient.EXPECT().
			Update(gomock.Any(), gomock.Any()).Return(nil),
	)

	err := r.ReconcileConfigMaps(ctx, m)
	assert.NoError(t, err)

	// not call client.Update if configmap not changed
	gomock.InOrder(
		mockClient.EXPECT().
			Get(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&corev1.ConfigMap{})).
			DoAndReturn(func(ctx context.Context, key client.ObjectKey, obj client.Object) error {
				cm := obj.(*corev1.ConfigMap)
				cm.Namespace = "ns"
				cm.Name = "cm1"
				r.updateConfigMap(ctx, m, cm)
				return nil
			}),
		// get secret of minio
		mockClient.EXPECT().
			Get(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&corev1.Secret{})).
			Return(k8sErrors.NewNotFound(schema.GroupResource{}, "mockErr")).Times(2),
	)
	err = r.ReconcileConfigMaps(ctx, m)
	assert.NoError(t, err)
}
