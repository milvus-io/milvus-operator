package controllers

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/milvus-io/milvus-operator/api/v1alpha1"
	"github.com/milvus-io/milvus-operator/pkg/config"
	"github.com/milvus-io/milvus-operator/pkg/util"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestClusterReconciler_ReconcileDeployments_CreateIfNotFound(t *testing.T) {
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
	mockClient.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&appsv1.Deployment{})).
		Return(k8sErrors.NewNotFound(schema.GroupResource{}, "")).
		Times(len(MilvusComponents))
	mockClient.EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&appsv1.Deployment{})).
		Return(nil).
		Times(len(MilvusComponents))

	err := r.ReconcileDeployments(ctx, mc)
	assert.NoError(t, err)
}

func TestClusterReconciler_ReconcileDeployments_UpdateIfExisted(t *testing.T) {
	config.Init(util.GetGitRepoRootDir())

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	r := newClusterReconcilerForTest(ctrl)
	mockClient := r.Client.(*MockK8sClient)

	m := v1alpha1.MilvusCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "mc",
		},
	}

	ctx := context.Background()

	// call client.Update if changed
	mockClient.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&appsv1.Deployment{})).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, obj client.Object) error {
			cm := obj.(*appsv1.Deployment)
			cm.Namespace = "ns"
			cm.Name = "mc"
			return nil
		}).Times(len(MilvusComponents))
	mockClient.EXPECT().
		Update(gomock.Any(), gomock.Any()).Return(nil).
		Times(len(MilvusComponents))

	err := r.ReconcileDeployments(ctx, m)
	assert.NoError(t, err)

	// not call client.Update if configmap not changed
	mockClient.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&appsv1.Deployment{})).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, obj client.Object) error {
			cm := obj.(*appsv1.Deployment)
			cm.Namespace = "ns"
			cm.Name = "mc-xxx"
			switch key.Name {
			case "mc-milvus-proxy":
				r.updateDeployment(m, cm, Proxy)
			case "mc-milvus-rootcoord":
				r.updateDeployment(m, cm, RootCoord)
			case "mc-milvus-datacoord":
				r.updateDeployment(m, cm, DataCoord)
			case "mc-milvus-querycoord":
				r.updateDeployment(m, cm, QueryCoord)
			case "mc-milvus-indexcoord":
				r.updateDeployment(m, cm, IndexCoord)
			case "mc-milvus-datanode":
				r.updateDeployment(m, cm, DataNode)
			case "mc-milvus-querynode":
				r.updateDeployment(m, cm, QueryNode)
			case "mc-milvus-indexnode":
				r.updateDeployment(m, cm, IndexNode)
			}
			return nil
		}).Times(len(MilvusComponents))

	err = r.ReconcileDeployments(ctx, m)
	assert.NoError(t, err)
}

func TestReconciler_ReconcileDeployments_CreateIfNotFound(t *testing.T) {
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
			Get(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&appsv1.Deployment{})).
			Return(k8sErrors.NewNotFound(schema.GroupResource{}, "")).
			Times(1),
		mockClient.EXPECT().
			Create(gomock.Any(), gomock.AssignableToTypeOf(&appsv1.Deployment{})).
			Return(nil).
			Times(1),
	)

	err := r.ReconcileDeployments(ctx, m)
	assert.NoError(t, err)
}

func TestMilvusReconciler_ReconcileDeployments_UpdateIfExisted(t *testing.T) {
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
			Get(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&appsv1.Deployment{})).
			DoAndReturn(func(ctx context.Context, key client.ObjectKey, obj client.Object) error {
				cm := obj.(*appsv1.Deployment)
				cm.Namespace = "ns"
				cm.Name = "cm1"
				return nil
			}),
		mockClient.EXPECT().
			Update(gomock.Any(), gomock.Any()).Return(nil),
	)

	err := r.ReconcileDeployments(ctx, m)
	assert.NoError(t, err)

	// not call client.Update if configmap not changed
	gomock.InOrder(
		mockClient.EXPECT().
			Get(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(&appsv1.Deployment{})).
			DoAndReturn(func(ctx context.Context, key client.ObjectKey, obj client.Object) error {
				cm := obj.(*appsv1.Deployment)
				cm.Namespace = "ns"
				cm.Name = "cm1"
				r.updateDeployment(m, cm)
				return nil
			}),
	)
	err = r.ReconcileDeployments(ctx, m)
	assert.NoError(t, err)
}
