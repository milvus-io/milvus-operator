package controllers

import (
	"context"
	"fmt"
	"os"

	"github.com/milvus-io/milvus-operator/api/v1alpha1"
	"github.com/milvus-io/milvus-operator/pkg/helm"
	"helm.sh/helm/v3/pkg/action"
)

const (
	EtcdChart   = "config/assets/charts/etcd"
	MinioChart  = "config/assets/charts/minio"
	PulsarChart = "config/assets/charts/pulsar"
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

func (r *MilvusClusterReconciler) ReconcileHelm(ctx context.Context, request helm.ChartRequest) error {
	cfg, err := r.NewHelmCfg(request.Namespace)
	if err != nil {
		return err
	}

	exist, err := helm.ReleaseExist(cfg, request.ReleaseName)
	if err != nil {
		return err
	}

	if !exist {
		if request.Chart == PulsarChart {
			request.Values["initialize"] = true
		}
		return helm.Install(cfg, request)
	}

	//todo update
	return nil
}

func (r *MilvusClusterReconciler) ReconcileEtcd(ctx context.Context, mc v1alpha1.MilvusCluster) error {
	if mc.Spec.Etcd.External != nil {
		return nil
	}

	request := helm.ChartRequest{
		ReleaseName: mc.Name + "-etcd",
		Namespace:   mc.Namespace,
		Chart:       EtcdChart,
		Values:      mc.Spec.Etcd.InCluster.Values.Data,
	}

	return r.ReconcileHelm(ctx, request)
}

func (r *MilvusClusterReconciler) ReconcilePulsar(ctx context.Context, mc v1alpha1.MilvusCluster) error {
	if mc.Spec.Pulsar.External != nil {
		return nil
	}

	request := helm.ChartRequest{
		ReleaseName: mc.Name + "-pulsar",
		Namespace:   mc.Namespace,
		Chart:       PulsarChart,
		Values:      mc.Spec.Pulsar.InCluster.Values.Data,
	}

	return r.ReconcileHelm(ctx, request)
}

func (r *MilvusClusterReconciler) ReconcileMinio(ctx context.Context, mc v1alpha1.MilvusCluster) error {
	if mc.Spec.Storage.External != nil {
		return nil
	}

	request := helm.ChartRequest{
		ReleaseName: mc.Name + "-minio",
		Namespace:   mc.Namespace,
		Chart:       MinioChart,
		Values:      mc.Spec.Storage.InCluster.Values.Data,
	}

	return r.ReconcileHelm(ctx, request)
}
