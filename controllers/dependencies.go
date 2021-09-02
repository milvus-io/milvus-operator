package controllers

import (
	"context"
	"fmt"

	"github.com/milvus-io/milvus-operator/api/v1alpha1"
)

func (r *MilvusClusterReconciler) ReconcileDependencies(ctx context.Context, mc *v1alpha1.MilvusCluster) error {
	checker := newStatusChecker(ctx, r.Client, mc)

	g, _ := NewGroup(ctx)
	g.Go(checker.checkEtcdStatus)
	g.Go(checker.checkStorageStatus)

	if err := g.Wait(); err != nil {
		return fmt.Errorf("reconcile milvus dependencies: %w", err)
	}

	return nil
}
