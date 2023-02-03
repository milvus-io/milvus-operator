package controllers

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/milvus-io/milvus-operator/apis/milvus.io/v1beta1"
	milvusv1beta1 "github.com/milvus-io/milvus-operator/apis/milvus.io/v1beta1"
	"github.com/milvus-io/milvus-operator/pkg/config"
	"github.com/milvus-io/milvus-operator/pkg/helm"
	"github.com/milvus-io/milvus-operator/pkg/util"
)

var _ = Describe("Milvus controller", func() {
	const (
		MCName      = "test"
		MCNamespace = "default"

		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("Creating Default Milvus", func() {
		It("It should has default spec", func() {
			By("By creating a new Milvus")
			ctx := context.Background()
			mc := &milvusv1beta1.Milvus{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "milvus.io/v1beta1",
					Kind:       "Milvus",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      MCName,
					Namespace: MCNamespace,
				},
			}

			mc.Default()
			Expect(k8sClient.Create(ctx, mc)).Should(Succeed())

			mcLookupKey := types.NamespacedName{Name: MCName, Namespace: MCNamespace}
			createdMC := &milvusv1beta1.Milvus{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, mcLookupKey, createdMC)
				return err == nil
			}, time.Second*10, interval).Should(BeTrue())
			// Let's make sure our Schedule string value was properly converted/handled.
			Expect(*createdMC.Spec.Com.Standalone.Replicas).Should(Equal(int32(1)))

			/* ss := &appsv1.StatefulSetList{}
			Eventually(func() bool {
				return nil == k8sClient.List(ctx, ss)
			}, time.Second*10, interval).Should(BeTrue()) */
			//Expect(len(ss.Items) > 0).Should(BeTrue())

		})
	})

})

func TestClusterReconciler(t *testing.T) {
	config.Init(util.GetGitRepoRootDir())

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	r := newMilvusReconcilerForTest(ctrl)
	mockSyncer := NewMockMilvusStatusSyncerInterface(ctrl)
	r.statusSyncer = mockSyncer
	// syncer need not to run in this test
	mockSyncer.EXPECT().RunIfNot().AnyTimes()
	globalCommonInfo.once.Do(func() {})

	mockClient := r.Client.(*MockK8sClient)

	m := v1beta1.Milvus{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "mc",
		},
	}

	ctx := context.Background()

	t.Run("create", func(t *testing.T) {
		defer ctrl.Finish()

		mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).
			Do(func(ctx, key, obj interface{}) {
				o := obj.(*v1beta1.Milvus)
				*o = m
			}).
			Return(nil)

		mockClient.EXPECT().Update(gomock.Any(), gomock.Any()).Do(
			func(ctx, obj interface{}, opts ...interface{}) {
				u := obj.(*v1beta1.Milvus)
				// finalizer should be added
				assert.Equal(t, u.Finalizers, []string{MilvusFinalizerName})
			},
		).Return(errors.Errorf("mock"))

		m.Finalizers = []string{MilvusFinalizerName}
		_, err := r.Reconcile(ctx, reconcile.Request{})
		assert.Error(t, err)
	})

	t.Run("case delete remove finalizer", func(t *testing.T) {
		defer ctrl.Finish()

		m.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Now()}
		mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).
			Do(func(ctx, key, obj interface{}) {
				o := obj.(*v1beta1.Milvus)
				*o = m
			}).
			Return(nil)

		mockClient.EXPECT().Status().Return(mockClient)
		mockClient.EXPECT().Update(gomock.Any(), gomock.Any()).Times(1)

		mockClient.EXPECT().Update(gomock.Any(), gomock.Any()).Do(
			func(ctx, obj interface{}, opts ...interface{}) {
				// finalizer should be removed
				u := obj.(*v1beta1.Milvus)
				assert.Equal(t, u.Finalizers, []string{})
			},
		).Return(nil)

		_, err := r.Reconcile(ctx, reconcile.Request{})
		assert.NoError(t, err)
	})
}

func TestMilvusReconciler_ReconcileLegacyValues(t *testing.T) {
	config.Init(util.GetGitRepoRootDir())

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testEnv := newTestEnv(t)
	r := testEnv.Reconciler
	helm.SetDefaultClient(&helm.LocalClient{})
	template := v1beta1.Milvus{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "mc",
		},
		Status: v1beta1.MilvusStatus{
			Status: v1beta1.StatusPending,
		},
	}
	ctx := context.Background()

	t.Run("not legacy, no need to sync", func(t *testing.T) {
		defer ctrl.Finish()
		old := template.DeepCopy()
		old.Status.Status = ""
		old.Default()
		obj := old.DeepCopy()
		err := r.ReconcileLegacyValues(ctx, old, obj)
		assert.NoError(t, err)
	})

	t.Run("legacy, update", func(t *testing.T) {
		defer ctrl.Finish()
		old := template.DeepCopy()
		old.Default()
		testEnv.MockClient.EXPECT().Update(gomock.Any(), gomock.Any())
		obj := old.DeepCopy()
		err := r.ReconcileLegacyValues(ctx, old, obj)
		assert.NoError(t, err)
	})

	t.Run("standalone all internal ok", func(t *testing.T) {
		obj := template.DeepCopy()
		obj.Default()
		assert.Equal(t, true, obj.LegacyNeedSyncValues())
		err := r.syncLegacyValues(ctx, obj)
		assert.NoError(t, err)
		assert.Equal(t, false, obj.LegacyNeedSyncValues())
	})

	t.Run("cluster with pulsar all internal ok", func(t *testing.T) {
		obj := template.DeepCopy()
		obj.Spec.Mode = v1beta1.MilvusModeCluster

		obj.Default()
		assert.Equal(t, true, obj.LegacyNeedSyncValues())
		err := r.syncLegacyValues(ctx, obj)
		assert.NoError(t, err)
		assert.Equal(t, false, obj.LegacyNeedSyncValues())
	})
	t.Run("cluster with kafka all internal ok", func(t *testing.T) {
		obj := template.DeepCopy()
		obj.Spec.Dep.MsgStreamType = v1beta1.MsgStreamTypeKafka
		obj.Default()
		assert.Equal(t, true, obj.LegacyNeedSyncValues())
		err := r.syncLegacyValues(ctx, obj)
		assert.NoError(t, err)
		assert.Equal(t, false, obj.LegacyNeedSyncValues())
	})
}
