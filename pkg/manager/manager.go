package manager

import (
	"time"

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	milvusiov1alpha1 "github.com/milvus-io/milvus-operator/apis/milvus.io/v1alpha1"
	milvusiov1beta1 "github.com/milvus-io/milvus-operator/apis/milvus.io/v1beta1"
	"github.com/milvus-io/milvus-operator/pkg/config"
)

var (
	scheme = runtime.NewScheme()
	mgrLog = ctrl.Log.WithName("manager")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(milvusiov1alpha1.AddToScheme(scheme))
	utilruntime.Must(milvusiov1beta1.AddToScheme(scheme))
	utilruntime.Must(monitoringv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func NewManager(metricsAddr, probeAddr string, enableLeaderElection bool) (ctrl.Manager, error) {
	syncPeriod := time.Second * time.Duration(config.SyncIntervalSec)
	ctrlOptions := ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "71808ec5.milvus.io",
		SyncPeriod:             &syncPeriod,
	}

	conf := ctrl.GetConfigOrDie()
	mgr, err := ctrl.NewManager(conf, ctrlOptions)
	if err != nil {
		mgrLog.Error(err, "unable to start manager")
		return nil, err
	}

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		mgrLog.Error(err, "unable to set up health check")
		return nil, err
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		mgrLog.Error(err, "unable to set up ready check")
		return nil, err
	}

	return mgr, nil
}
