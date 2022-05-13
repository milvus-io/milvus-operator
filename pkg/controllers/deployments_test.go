package controllers

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/milvus-io/milvus-operator/apis/milvus.io/v1beta1"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestClusterReconciler_ReconcileDeployments_CreateIfNotFound(t *testing.T) {
	env := newTestEnv(t)
	defer env.checkMocks()
	r := env.Reconciler
	mockClient := env.MockClient
	ctx := env.ctx
	mc := env.Inst
	mc.Spec.Mode = v1beta1.MilvusModeCluster
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

func TestClusterReconciler_ReconcileDeployments_Existed(t *testing.T) {
	env := newTestEnv(t)
	defer env.checkMocks()
	r := env.Reconciler
	mockClient := env.MockClient
	ctx := env.ctx
	m := env.Inst
	m.Spec.Mode = v1beta1.MilvusModeCluster
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
