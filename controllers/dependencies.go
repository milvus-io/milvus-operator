package controllers

import (
	"context"
	"fmt"
	"os"

	"github.com/milvus-io/milvus-operator/api/v1alpha1"
	"github.com/milvus-io/milvus-operator/pkg/helm"
	"helm.sh/helm/v3/pkg/action"
)

func (r *MilvusClusterReconciler) NewHelmCfg(namespace string) (*action.Configuration, error) {
	cfg := new(action.Configuration)
	helmLogger := func(format string, v ...interface{}) {
		r.logger.WithName("helm").Info(fmt.Sprintf(format, v...))
	}
	if err := cfg.Init(r.helmSettings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), helmLogger); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (r *MilvusClusterReconciler) HelmUpgradePulsar(ctx context.Context, mc *v1alpha1.MilvusCluster) error {
	cfg, err := r.NewHelmCfg(mc.Namespace)
	if err != nil {
		return err
	}

	releaseName := mc.Name + "-pulsar"
	exist, err := helm.ReleaseExist(cfg, releaseName)
	if err != nil {
		return err
	}

	request := helm.ChartRequest{
		ReleaseName: mc.Name + "-pulsar",
		Namespace:   mc.Namespace,
		Chart:       "config/assets/charts/pulsar",
		Values:      mc.Spec.Pulsar.InCluster.Values.Data,
	}

	if !exist {
		request.Values["initialize"] = true
		return helm.Install(cfg, request)
	}

	return nil
	//return helm.Update(cfg, request)
}

func (r *MilvusClusterReconciler) HelmUpgradeMinio(ctx context.Context, mc *v1alpha1.MilvusCluster) error {
	cfg, err := r.NewHelmCfg(mc.Namespace)
	if err != nil {
		return err
	}

	return helm.Upgrade(cfg, helm.ChartRequest{
		ReleaseName: mc.Name + "-minio",
		Namespace:   mc.Namespace,
		Chart:       "config/assets/charts/minio",
		Values:      mc.Spec.Storage.InCluster.Values.Data,
	})
}

func (r *MilvusClusterReconciler) HelmUpgradeEtcd(ctx context.Context, mc *v1alpha1.MilvusCluster) error {
	cfg, err := r.NewHelmCfg(mc.Namespace)
	if err != nil {
		return err
	}

	return helm.Upgrade(cfg, helm.ChartRequest{
		ReleaseName: mc.Name + "-etcd",
		Namespace:   mc.Namespace,
		Chart:       "config/assets/charts/etcd",
		Values:      mc.Spec.Etcd.InCluster.Values.Data,
	})
}

func (r *MilvusClusterReconciler) ReconcileDependencies(ctx context.Context, mc *v1alpha1.MilvusCluster) error {
	if mc.Spec.Etcd.InCluster != nil {
		if err := r.HelmUpgradeEtcd(ctx, mc); err != nil {
			return err
		}
	}

	if mc.Spec.Pulsar.InCluster != nil {
		if err := r.HelmUpgradePulsar(ctx, mc); err != nil {
			return err
		}
	}

	if mc.Spec.Storage.InCluster != nil {
		if err := r.HelmUpgradeMinio(ctx, mc); err != nil {
			return err
		}
	}

	checker := newStatusChecker(ctx, r.Client, mc)

	g, _ := NewGroup(ctx)
	g.Go(checker.checkEtcdStatus)
	g.Go(checker.checkStorageStatus)
	g.Go(checker.checkPulsarStatus)

	if err := g.Wait(); err != nil {
		return fmt.Errorf("reconcile milvus dependencies: %w", err)
	}

	return nil
}
