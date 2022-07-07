package controllers

import (
	"context"
	"errors"
	"testing"

	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestReconcileConfigMaps_CreateIfNotfound(t *testing.T) {
	env := newTestEnv(t)
	defer env.checkMocks()
	r := env.Reconciler
	mockClient := env.MockClient
	ctx := env.ctx
	mc := env.Inst

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

func TestReconcileConfigMaps_Existed(t *testing.T) {
	env := newTestEnv(t)
	defer env.checkMocks()
	r := env.Reconciler
	mockClient := env.MockClient
	ctx := env.ctx
	mc := env.Inst

	t.Run("call client.Update if changed configmap", func(t *testing.T) {
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
	})

	err := r.ReconcileConfigMaps(ctx, mc)
	assert.NoError(t, err)

	t.Run("not call client.Update if configmap not changed", func(t *testing.T) {
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
	})

	t.Run("iam no update", func(t *testing.T) {
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
	})
}
