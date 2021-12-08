package controllers

import (
	"context"
	"fmt"
	"os"
	"reflect"

	"github.com/go-logr/logr"
	"github.com/milvus-io/milvus-operator/apis/milvus.io/v1alpha1"
	"github.com/milvus-io/milvus-operator/pkg/helm"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
)

//go:generate mockgen -package=controllers -source=dependencies.go -destination=dependencies_mock.go HelmReconciler

const (
	EtcdChart   = "config/assets/charts/etcd"
	MinioChart  = "config/assets/charts/minio"
	PulsarChart = "config/assets/charts/pulsar"
)

// HelmReconciler reconciles Helm releases
type HelmReconciler interface {
	NewHelmCfg(namespace string) *action.Configuration
	Reconcile(ctx context.Context, request helm.ChartRequest) error
}

// LocalHelmReconciler implements HelmReconciler at local
type LocalHelmReconciler struct {
	helmSettings *cli.EnvSettings
	logger       logr.Logger
}

func NewLocalHelmReconciler(helmSettings *cli.EnvSettings, logger logr.Logger) HelmReconciler {
	return &LocalHelmReconciler{
		helmSettings: helmSettings,
		logger:       logger,
	}
}

func (l LocalHelmReconciler) NewHelmCfg(namespace string) *action.Configuration {
	cfg := new(action.Configuration)
	helmLogger := func(format string, v ...interface{}) {
		l.logger.Info(fmt.Sprintf(format, v...))
	}

	// cfg.Init will never return err, only panic if bad driver
	cfg.Init(l.helmSettings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), helmLogger)

	return cfg
}

// ReconcileHelm reconciles Helm releases
func (l LocalHelmReconciler) Reconcile(ctx context.Context, request helm.ChartRequest) error {
	cfg := l.NewHelmCfg(request.Namespace)

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

	vals, err := helm.GetValues(cfg, request.ReleaseName)
	if err != nil {
		return err
	}

	status, err := helm.GetStatus(cfg, request.ReleaseName)
	if err != nil {
		return err
	}

	if request.Chart == PulsarChart {
		delete(vals, "initialize")
	}

	if reflect.DeepEqual(vals, request.Values) && !helm.NeedUpdate(status) {
		return nil
	}

	return helm.Update(cfg, request)
}

func (r *MilvusClusterReconciler) ReconcileEtcd(ctx context.Context, mc v1alpha1.MilvusCluster) error {
	if mc.Spec.Dep.Etcd.External {
		return nil
	}

	request := helm.ChartRequest{
		ReleaseName: mc.Name + "-etcd",
		Namespace:   mc.Namespace,
		Chart:       EtcdChart,
		Values:      mc.Spec.Dep.Etcd.InCluster.Values.Data,
	}

	return r.helmReconciler.Reconcile(ctx, request)
}

func (r *MilvusClusterReconciler) ReconcilePulsar(ctx context.Context, mc v1alpha1.MilvusCluster) error {
	if mc.Spec.Dep.Pulsar.External {
		return nil
	}

	request := helm.ChartRequest{
		ReleaseName: mc.Name + "-pulsar",
		Namespace:   mc.Namespace,
		Chart:       PulsarChart,
		Values:      mc.Spec.Dep.Pulsar.InCluster.Values.Data,
	}

	return r.helmReconciler.Reconcile(ctx, request)
}

func (r *MilvusClusterReconciler) ReconcileMinio(ctx context.Context, mc v1alpha1.MilvusCluster) error {
	if mc.Spec.Dep.Storage.External {
		return nil
	}

	request := helm.ChartRequest{
		ReleaseName: mc.Name + "-minio",
		Namespace:   mc.Namespace,
		Chart:       MinioChart,
		Values:      mc.Spec.Dep.Storage.InCluster.Values.Data,
	}

	return r.helmReconciler.Reconcile(ctx, request)
}

func (r *MilvusReconciler) ReconcileEtcd(ctx context.Context, mil v1alpha1.Milvus) error {
	if mil.Spec.Dep.Etcd.External {
		return nil
	}

	request := helm.ChartRequest{
		ReleaseName: mil.Name + "-etcd",
		Namespace:   mil.Namespace,
		Chart:       EtcdChart,
		Values:      mil.Spec.Dep.Etcd.InCluster.Values.Data,
	}

	return r.helmReconciler.Reconcile(ctx, request)
}

func (r *MilvusReconciler) ReconcileMinio(ctx context.Context, mil v1alpha1.Milvus) error {
	if mil.Spec.Dep.Storage.External {
		return nil
	}

	request := helm.ChartRequest{
		ReleaseName: mil.Name + "-minio",
		Namespace:   mil.Namespace,
		Chart:       MinioChart,
		Values:      mil.Spec.Dep.Storage.InCluster.Values.Data,
	}

	return r.helmReconciler.Reconcile(ctx, request)
}
