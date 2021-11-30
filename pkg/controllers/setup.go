package controllers

import (
	"context"

	milvusv1alpha1 "github.com/milvus-io/milvus-operator/api/v1alpha1"
	"helm.sh/helm/v3/pkg/cli"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func SetupControllers(ctx context.Context, mgr manager.Manager, enableHook bool) error {
	logger := ctrl.Log.WithName("controller")

	conf := mgr.GetConfig()
	settings := cli.New()
	settings.KubeAPIServer = conf.Host
	settings.MaxHistory = 2
	settings.KubeToken = conf.BearerToken
	getter := settings.RESTClientGetter()
	config := getter.(*genericclioptions.ConfigFlags)
	insecure := true
	config.Insecure = &insecure

	// should be run after mgr started to make sure the client is ready
	clusterStatusSyncer := NewMilvusClusterStatusSyncer(ctx, mgr.GetClient(), logger.WithName("status-syncer"))

	clusterController := &MilvusClusterReconciler{
		Client:       mgr.GetClient(),
		Scheme:       mgr.GetScheme(),
		logger:       logger.WithName("milvus-cluster"),
		helmSettings: settings,
		statusSyncer: clusterStatusSyncer,
	}

	if err := clusterController.SetupWithManager(mgr); err != nil {
		logger.Error(err, "unable to setup milvus cluster controller with manager", "controller", "MilvusCluster")
		return err
	}

	// should be run after mgr started to make sure the client is ready
	statusSyncer := NewMilvusStatusSyncer(ctx, mgr.GetClient(), logger.WithName("status-syncer"))

	controller := &MilvusReconciler{
		Client:       mgr.GetClient(),
		Scheme:       mgr.GetScheme(),
		logger:       logger.WithName("milvus"),
		helmSettings: settings,
		statusSyncer: statusSyncer,
	}
	if err := controller.SetupWithManager(mgr); err != nil {
		logger.Error(err, "unable to setup milvus controller with manager", "controller", "MilvusCluster")
		return err
	}

	if enableHook {
		if err := (&milvusv1alpha1.MilvusCluster{}).SetupWebhookWithManager(mgr); err != nil {
			logger.Error(err, "unable to create webhook", "webhook", "MilvusCluster")
			return err
		}
		if err := (&milvusv1alpha1.Milvus{}).SetupWebhookWithManager(mgr); err != nil {
			logger.Error(err, "unable to create webhook", "webhook", "Milvus")
			return err
		}
	}

	return nil
}
