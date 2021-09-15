package controllers

import (
	"context"
	"fmt"
	"os"

	"helm.sh/helm/v3/pkg/action"

	"github.com/go-logr/logr"
	"github.com/milvus-io/milvus-operator/api/v1alpha1"
	"github.com/milvus-io/milvus-operator/pkg/helm"
)

func HelmLogWarpper(l logr.Logger) action.DebugLog {
	return func(format string, v ...interface{}) {
		l.WithName("helm").Info(fmt.Sprintf(format, v...))
	}
}

func (r *MilvusClusterReconciler) HelmUpgradeEtcd(ctx context.Context, mc *v1alpha1.MilvusCluster) error {
	cfg := new(action.Configuration)
	if err := cfg.Init(
		r.helmSettings.RESTClientGetter(),
		mc.Namespace, os.Getenv("HELM_DRIVER"),
		HelmLogWarpper(r.logger)); err != nil {
		return err
	}

	return helm.Upgrade(cfg, helm.ChartRequest{
		ReleaseName: mc.Name,
		Namespace:   mc.Namespace,
		Chart:       "config/assets/charts/etcd-6.8.0.tgz",
	})
}

func (r *MilvusClusterReconciler) ReconcileDependencies(ctx context.Context, mc *v1alpha1.MilvusCluster) error {
	if err := r.HelmUpgradeEtcd(ctx, mc); err != nil {
		return err
	}

	checker := newStatusChecker(ctx, r.Client, mc)

	g, _ := NewGroup(ctx)
	g.Go(checker.checkEtcdStatus)
	g.Go(checker.checkStorageStatus)

	if err := g.Wait(); err != nil {
		return fmt.Errorf("reconcile milvus dependencies: %w", err)
	}

	return nil
}
