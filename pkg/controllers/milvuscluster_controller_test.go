package controllers

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	milvusv1alpha1 "github.com/milvus-io/milvus-operator/api/v1alpha1"
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
