package controllers

import (
	"context"
	"errors"
	"testing"

	"github.com/milvus-io/milvus-operator/apis/milvus.io/v1alpha1"
	"github.com/milvus-io/milvus-operator/pkg/config"
	"github.com/milvus-io/milvus-operator/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestParallelGroupConciler_Milvus(t *testing.T) {
	config.Init(util.GetGitRepoRootDir())
	vals := [3]int{}
	func1 := func(context.Context, v1alpha1.Milvus) error {
		vals[0]++
		return nil
	}
	func2 := func(context.Context, v1alpha1.Milvus) error {
		vals[1]++
		return nil
	}
	func3 := func(context.Context, v1alpha1.Milvus) error {
		vals[2]++
		return errors.New("test")
	}
	funcs := []MilvusReconcileFunc{
		func1, func2, func3,
	}
	err := defaultGroupReconciler.ReconcileMilvus(context.TODO(), funcs, v1alpha1.Milvus{})
	assert.Error(t, err)
	assert.Equal(t, [3]int{1, 1, 1}, vals)
}

func TestParallelGroupConciler_Cluster(t *testing.T) {
	config.Init(util.GetGitRepoRootDir())
	vals := [3]int{}
	func1 := func(context.Context, v1alpha1.MilvusCluster) error {
		vals[0]++
		return nil
	}
	func2 := func(context.Context, v1alpha1.MilvusCluster) error {
		vals[1]++
		return nil
	}
	func3 := func(context.Context, v1alpha1.MilvusCluster) error {
		vals[2]++
		return errors.New("test")
	}
	funcs := []MilvusClusterReconcileFunc{
		func1, func2, func3,
	}
	err := defaultGroupReconciler.ReconcileMilvusCluster(context.TODO(), funcs, v1alpha1.MilvusCluster{})
	assert.Error(t, err)
	assert.Equal(t, [3]int{1, 1, 1}, vals)
}
