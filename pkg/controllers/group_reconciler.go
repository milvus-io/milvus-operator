package controllers

import (
	"context"

	"github.com/milvus-io/milvus-operator/apis/milvus.io/v1alpha1"
	"github.com/pkg/errors"
)

//go:generate mockgen -source=./group_reconciler.go -destination=./group_reconciler_mock.go -package=controllers

type MilvusReconcileFunc func(context.Context, v1alpha1.Milvus) error
type MilvusClusterReconcileFunc func(context.Context, v1alpha1.MilvusCluster) error

// GroupReconciler does a group of reconcile funcs
type GroupReconciler interface {
	ReconcileMilvus(ctx context.Context, funcs []MilvusReconcileFunc, mil v1alpha1.Milvus) error
	ReconcileMilvusCluster(ctx context.Context, funcs []MilvusClusterReconcileFunc, mil v1alpha1.MilvusCluster) error
}

var defaultGroupReconciler GroupReconciler = new(ParallelGroupConciler)

type ParallelGroupConciler struct {
}

func (ParallelGroupConciler) ReconcileMilvus(ctx context.Context, funcs []MilvusReconcileFunc, m v1alpha1.Milvus) error {
	g, gtx := NewGroup(ctx)
	for _, aFunc := range funcs {
		g.Go(WrappedReconcileMilvus(aFunc, gtx, m))
	}
	err := g.Wait()
	return errors.Wrap(err, "run group reconcile failed")
}

func (ParallelGroupConciler) ReconcileMilvusCluster(ctx context.Context, funcs []MilvusClusterReconcileFunc, m v1alpha1.MilvusCluster) error {
	g, gtx := NewGroup(ctx)
	for _, aFunc := range funcs {
		g.Go(WarppedReconcileFunc(aFunc, gtx, m))
	}
	err := g.Wait()
	return errors.Wrap(err, "run group reconcile failed")
}
