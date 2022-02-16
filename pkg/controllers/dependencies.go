package controllers

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"

	"github.com/go-logr/logr"
	"github.com/milvus-io/milvus-operator/apis/milvus.io/v1alpha1"
	"github.com/milvus-io/milvus-operator/pkg/helm"
	"github.com/milvus-io/milvus-operator/pkg/util"
	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/yaml"
)

//go:generate mockgen -package=controllers -source=dependencies.go -destination=dependencies_mock.go HelmReconciler

const (
	EtcdChart   = "config/assets/charts/etcd"
	MinioChart  = "config/assets/charts/minio"
	PulsarChart = "config/assets/charts/pulsar"

	Etcd   = "etcd"
	Minio  = "minio"
	Pulsar = "pulsar"
)

var (
	// DefaultValuesPath is the path to the default values file
	// variable in test, const in runtime
	DefaultValuesPath = "config/assets/charts/values.yaml"
)

// HelmReconciler reconciles Helm releases
type HelmReconciler interface {
	NewHelmCfg(namespace string) *action.Configuration
	Reconcile(ctx context.Context, request helm.ChartRequest) error
}

type Chart = string
type Values = map[string]interface{}

// LocalHelmReconciler implements HelmReconciler at local
type LocalHelmReconciler struct {
	helmSettings       *cli.EnvSettings
	logger             logr.Logger
	chartDefaultValues map[Chart]Values
}

func readValuesFromFile(file string) (Values, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read file %s", file)
	}
	ret := Values{}
	err = yaml.Unmarshal(data, &ret)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal file %s", file)
	}
	return ret, nil
}

func MustNewLocalHelmReconciler(helmSettings *cli.EnvSettings, logger logr.Logger) *LocalHelmReconciler {
	values, err := readValuesFromFile(DefaultValuesPath)
	if err != nil {
		err = errors.Wrapf(err, "failed to read default helm chart values from [%s]", DefaultValuesPath)
		panic(err)
	}
	return &LocalHelmReconciler{
		helmSettings: helmSettings,
		logger:       logger,
		chartDefaultValues: map[Chart]Values{
			Etcd:   values[Etcd].(Values),
			Minio:  values[Minio].(Values),
			Pulsar: values[Pulsar].(Values),
		},
	}
}

func (l LocalHelmReconciler) NewHelmCfg(namespace string) *action.Configuration {
	cfg := new(action.Configuration)
	helmLogger := func(format string, v ...interface{}) {
		l.logger.Info(fmt.Sprintf(format, v...))
	}

	// cfg.Init will never return err, only panic if bad driver
	cfg.Init(
		getRESTClientGetterWithNamespace(l.helmSettings, namespace),
		namespace,
		os.Getenv("HELM_DRIVER"),
		helmLogger,
	)

	return cfg
}

func getRESTClientGetterWithNamespace(env *cli.EnvSettings, namespace string) genericclioptions.RESTClientGetter {
	return &genericclioptions.ConfigFlags{
		Namespace:        &namespace,
		Context:          &env.KubeContext,
		BearerToken:      &env.KubeToken,
		APIServer:        &env.KubeAPIServer,
		CAFile:           &env.KubeCaFile,
		KubeConfig:       &env.KubeConfig,
		Impersonate:      &env.KubeAsUser,
		ImpersonateGroup: &env.KubeAsGroups,
		Insecure:         &configFlagInsecure,
	}
}

func getChartNameByPath(chart string) string {
	switch chart {
	case EtcdChart:
		return Etcd
	case MinioChart:
		return Minio
	case PulsarChart:
		return Pulsar
	default:
		return ""
	}
}

func (l LocalHelmReconciler) MergeWithDefaultValues(chartPath string, values Values) Values {
	chart := getChartNameByPath(chartPath)
	ret := Values{}
	util.MergeValues(ret, l.chartDefaultValues[chart])
	util.MergeValues(ret, values)
	return ret
}

// ReconcileHelm reconciles Helm releases
func (l LocalHelmReconciler) Reconcile(ctx context.Context, request helm.ChartRequest) error {
	cfg := l.NewHelmCfg(request.Namespace)
	l.logger.Info("debug helm reconcile", "namespace", request.Namespace, "release", request.ReleaseName)

	request.Values = l.MergeWithDefaultValues(request.Chart, request.Values)

	exist, err := helm.ReleaseExist(cfg, request.ReleaseName)
	if err != nil {
		return err
	}

	if !exist {
		if request.Chart == PulsarChart {
			request.Values["initialize"] = true
		}
		l.logger.Info("helm install values", "values", request.Values)
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

	deepEqual := reflect.DeepEqual(vals, request.Values)
	needUpdate := helm.NeedUpdate(status)
	if deepEqual && !needUpdate {
		return nil
	}
	l.logger.Info("update helm", "namespace", request.Namespace, "release", request.ReleaseName, "needUpdate", needUpdate, "deepEqual", deepEqual)
	if !deepEqual {
		l.logger.Info("update helm values", "old", vals, "new", request.Values)
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
