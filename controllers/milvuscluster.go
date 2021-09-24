package controllers

import (
	"context"
	"fmt"

	"github.com/milvus-io/milvus-operator/api/v1alpha1"
)

func (r *MilvusClusterReconciler) SetDefault(ctx context.Context, mc *v1alpha1.MilvusCluster) error {
	if mc.Status.Status == "" {
		mc.Status.Status = v1alpha1.StatusCreating
	}

	return nil
}

func (r *MilvusClusterReconciler) ReconcileAll(ctx context.Context, mc v1alpha1.MilvusCluster) error {
	g, gtx := NewGroup(ctx)
	g.Go(WarppedReconcileFunc(r.ReconcileEtcd, gtx, mc))
	g.Go(WarppedReconcileFunc(r.ReconcilePulsar, gtx, mc))
	g.Go(WarppedReconcileFunc(r.ReconcileMinio, gtx, mc))
	g.Go(WarppedReconcileFunc(r.ReconcileMilvus, gtx, mc))

	if err := g.Wait(); err != nil {
		return fmt.Errorf("reconcile milvus: %w", err)
	}

	return nil
}

func (r *MilvusClusterReconciler) ReconcileMilvus(ctx context.Context, mc v1alpha1.MilvusCluster) error {
	if !IsDependencyReady(mc.Status) {
		return nil
	}

	if err := r.ReconcileConfigMaps(ctx, mc); err != nil {
		return err
	}

	g, gtx := NewGroup(ctx)
	g.Go(WarppedReconcileFunc(r.ReconcileDeployments, gtx, mc))
	g.Go(WarppedReconcileFunc(r.ReconcileServices, gtx, mc))
	if err := g.Wait(); err != nil {
		return fmt.Errorf("reconcile milvus components: %w", err)
	}

	return nil
}
