package controllers

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("MilvusCluster utils", func() {
	const (
		namespace = "default"
		name      = "test"
	)

	Context("NamespacedName", func() {
		It("It should has name and namespace", func() {
			key := NamespacedName(namespace, name)
			Expect(key.Namespace).Should(Equal(namespace))
			Expect(key.Name).Should(Equal(name))
		})
	})
})
