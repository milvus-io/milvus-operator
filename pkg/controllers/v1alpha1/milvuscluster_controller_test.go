package controllers

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/milvus-io/milvus-operator/apis/milvus.io/v1alpha1"
	milvusv1alpha1 "github.com/milvus-io/milvus-operator/apis/milvus.io/v1alpha1"
	"github.com/milvus-io/milvus-operator/pkg/config"
	"github.com/milvus-io/milvus-operator/pkg/util"
)

var _ = Describe("MilvusCluster controller", func() {
	const (
		MCName      = "test"
		MCNamespace = "default"

		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("Creating Default MilvusCluster", func() {
		It("It should has default spec", func() {
			By("By creating a new Milvuscluster")
			ctx := context.Background()
			mc := &milvusv1alpha1.MilvusCluster{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "milvus.io/v1alpha1",
					Kind:       "MilvusCluster",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      MCName,
					Namespace: MCNamespace,
				},
			}

			mc.Default()
			Expect(k8sClient.Create(ctx, mc)).Should(Succeed())

			mcLookupKey := types.NamespacedName{Name: MCName, Namespace: MCNamespace}
			createdMC := &milvusv1alpha1.MilvusCluster{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, mcLookupKey, createdMC)
				return err == nil
			}, time.Second*10, interval).Should(BeTrue())
			// Let's make sure our Schedule string value was properly converted/handled.
			Expect(*createdMC.Spec.Com.DataNode.Replicas).Should(Equal(int32(1)))

			/* ss := &appsv1.StatefulSetList{}
			Eventually(func() bool {
				return nil == k8sClient.List(ctx, ss)
			}, time.Second*10, interval).Should(BeTrue()) */
			//Expect(len(ss.Items) > 0).Should(BeTrue())

		})
	})

})

func TestClusterReconciler_ReconcileFinalizer(t *testing.T) {
	config.Init(util.GetGitRepoRootDir())

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	r := newClusterReconcilerForTest(ctrl)
	r.statusSyncer = &MilvusClusterStatusSyncer{}
	// syncer need not to run in this test
	r.statusSyncer.Once.Do(func() {})
	mockClient := r.Client.(*MockK8sClient)

	// case create
	m := v1alpha1.MilvusCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "mc",
		},
	}
	m.Default()

	ctx := context.Background()

	mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).
		Do(func(ctx, key, obj interface{}) {
			o := obj.(*v1alpha1.MilvusCluster)
			*o = m
		}).
		Return(nil)

	mockClient.EXPECT().Update(gomock.Any(), gomock.Any()).Do(
		func(ctx, obj interface{}, opts ...interface{}) {
			u := obj.(*v1alpha1.MilvusCluster)
			// finalizer should be added
			assert.Equal(t, u.Finalizers, []string{MCFinalizerName})
		},
	).Return(nil)

	m.Finalizers = []string{MCFinalizerName}
	_, err := r.Reconcile(ctx, reconcile.Request{})
	assert.NoError(t, err)

	// case delete remove finalizer
	m.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Now()}
	mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).
		Do(func(ctx, key, obj interface{}) {
			o := obj.(*v1alpha1.MilvusCluster)
			*o = m
		}).
		Return(nil)

	mockClient.EXPECT().Update(gomock.Any(), gomock.Any()).Do(
		func(ctx, obj interface{}, opts ...interface{}) {
			// finalizer should be removed
			u := obj.(*v1alpha1.MilvusCluster)
			assert.Equal(t, u.Finalizers, []string{})
		},
	).Return(nil)

	_, err = r.Reconcile(ctx, reconcile.Request{})
	assert.NoError(t, err)
}
