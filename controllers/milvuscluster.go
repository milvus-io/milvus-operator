package controllers

import (
	"context"
	"fmt"

	"github.com/milvus-io/milvus-operator/api/v1alpha1"
)

func (r *MilvusClusterReconciler) ReconcileMilvusComponent(ctx context.Context, mc *v1alpha1.MilvusCluster) error {
	if err := r.ReconcileConfigMaps(ctx, *mc); err != nil {
		return err
	}

	g, gtx := NewGroup(ctx)
	g.Go(func(ctx context.Context, mc v1alpha1.MilvusCluster) func() error {
		return func() error {
			return r.ReconcileDeployments(ctx, mc)
		}
	}(gtx, *mc))

	g.Go(func(ctx context.Context, mc v1alpha1.MilvusCluster) func() error {
		return func() error {
			return r.ReconcileServices(ctx, mc)
		}
	}(gtx, *mc))

	if err := g.Wait(); err != nil {
		return fmt.Errorf("reconcile milvus components: %w", err)
	}

	checker := newStatusChecker(ctx, r.Client, mc)
	if err := checker.checkMilvusClusterStatus(); err != nil {
		return err
	}

	return nil
}
