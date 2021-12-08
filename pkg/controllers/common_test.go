package controllers

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/milvus-io/milvus-operator/apis/milvus.io/v1alpha1"
	"github.com/milvus-io/milvus-operator/pkg/config"
	"github.com/milvus-io/milvus-operator/pkg/util"
	"helm.sh/helm/v3/pkg/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlRuntime "sigs.k8s.io/controller-runtime"
)

type milvusTestEnv struct {
	MockClient *MockK8sClient
	Ctrl       *gomock.Controller
	Reconciler *MilvusReconciler
	Inst       v1alpha1.Milvus
	ctx        context.Context
}

func (m *milvusTestEnv) tearDown() {
	m.Ctrl.Finish()
}

func newMilvusTestEnv(t *testing.T) *milvusTestEnv {
	config.Init(util.GetGitRepoRootDir())

	ctrl := gomock.NewController(t)
	reconciler := newMilvusReconcilerForTest(ctrl)
	mockClient := reconciler.Client.(*MockK8sClient)

	inst := v1alpha1.Milvus{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "n",
		},
	}
	inst.Spec.Dep.Etcd.InCluster = new(v1alpha1.InClusterConfig)
	inst.Spec.Dep.Storage.InCluster = new(v1alpha1.InClusterConfig)
	return &milvusTestEnv{
		MockClient: mockClient,
		Ctrl:       ctrl,
		Reconciler: reconciler,
		Inst:       inst,
		ctx:        context.Background(),
	}
}

type clusterTestEnv struct {
	MockClient *MockK8sClient
	Ctrl       *gomock.Controller
	Reconciler *MilvusClusterReconciler
	Inst       v1alpha1.MilvusCluster
	ctx        context.Context
}

func (m *clusterTestEnv) tearDown() {
	m.Ctrl.Finish()
}

func newClusterTestEnv(t *testing.T) *clusterTestEnv {
	config.Init(util.GetGitRepoRootDir())

	ctrl := gomock.NewController(t)
	reconciler := newClusterReconcilerForTest(ctrl)
	mockClient := reconciler.Client.(*MockK8sClient)

	inst := v1alpha1.MilvusCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "mc",
		},
	}
	inst.Spec.Dep.Etcd.InCluster = new(v1alpha1.InClusterConfig)
	inst.Spec.Dep.Storage.InCluster = new(v1alpha1.InClusterConfig)
	inst.Spec.Dep.Pulsar.InCluster = new(v1alpha1.InClusterConfig)
	return &clusterTestEnv{
		MockClient: mockClient,
		Ctrl:       ctrl,
		Reconciler: reconciler,
		Inst:       inst,
		ctx:        context.Background(),
	}
}

func newClusterReconcilerForTest(ctrl *gomock.Controller) *MilvusClusterReconciler {
	mockClient := NewMockK8sClient(ctrl)

	logger := ctrlRuntime.Log.WithName("test")
	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)
	helmSetting := cli.New()
	helm := NewLocalHelmReconciler(helmSetting, logger)
	r := MilvusClusterReconciler{
		Client:         mockClient,
		logger:         logger,
		Scheme:         scheme,
		helmReconciler: helm,
	}
	return &r
}

func newMilvusReconcilerForTest(ctrl *gomock.Controller) *MilvusReconciler {
	mockClient := NewMockK8sClient(ctrl)

	logger := ctrlRuntime.Log.WithName("test")
	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)
	helmSetting := cli.New()
	helm := NewLocalHelmReconciler(helmSetting, logger)
	r := MilvusReconciler{
		Client:         mockClient,
		logger:         logger,
		Scheme:         scheme,
		helmReconciler: helm,
	}
	return &r
}
