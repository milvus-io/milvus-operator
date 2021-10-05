package controllers

import (
	"context"
	"fmt"

	"github.com/milvus-io/milvus-operator/api/v1alpha1"
	"github.com/milvus-io/milvus-operator/pkg/helm"
	"github.com/milvus-io/milvus-operator/pkg/util"
	"github.com/pkg/errors"
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

func (r *MilvusClusterReconciler) doFinalize(ctx context.Context, mc v1alpha1.MilvusCluster) error {
	releaseNames := []string{}

	if mc.Spec.Dep.Etcd.InCluster.DeletionPolicy == v1alpha1.DeletionPolicyDelete {
		releaseNames = append(releaseNames, mc.Name+"-etcd")
	}
	if mc.Spec.Dep.Pulsar.InCluster.DeletionPolicy == v1alpha1.DeletionPolicyDelete {
		releaseNames = append(releaseNames, mc.Name+"-pulsar")
	}
	if mc.Spec.Dep.Storage.InCluster.DeletionPolicy == v1alpha1.DeletionPolicyDelete {
		releaseNames = append(releaseNames, mc.Name+"-minio")
	}

	if len(releaseNames) > 0 {
		cfg, err := r.NewHelmCfg(mc.Namespace)
		if err != nil {
			return err
		}

		errs := []error{}
		for _, releaseName := range releaseNames {
			if err := helm.Uninstall(cfg, releaseName); err != nil {
				errs = append(errs, err)
			}
		}

		if len(errs) > 0 {
			return errors.Errorf(util.JoinErrors(errs))
		}
	}

	return nil
}
