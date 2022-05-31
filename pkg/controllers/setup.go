package controllers

import (
	"context"
	"sync"

	"github.com/milvus-io/milvus-operator/pkg/config"
	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/cli"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	milvusv1alpha1 "github.com/milvus-io/milvus-operator/apis/milvus.io/v1alpha1"
	milvusv1beta1 "github.com/milvus-io/milvus-operator/apis/milvus.io/v1beta1"
	milvuscluster "github.com/milvus-io/milvus-operator/pkg/controllers/milvuscluster"
)

var configFlagInsecure = true

func SetupControllers(ctx context.Context, mgr manager.Manager, enableHook bool) error {
	logger := ctrl.Log.WithName("controller")

	conf := mgr.GetConfig()
	settings := cli.New()
	settings.KubeAPIServer = conf.Host
	settings.MaxHistory = 2
	settings.KubeToken = conf.BearerToken
	getter := settings.RESTClientGetter()
	config := getter.(*genericclioptions.ConfigFlags)
	config.Insecure = &configFlagInsecure
	helmReconciler := MustNewLocalHelmReconciler(settings, logger.WithName("helm"))

	// should be run after mgr started to make sure the client is ready
	statusSyncer := NewMilvusStatusSyncer(ctx, mgr.GetClient(), logger.WithName("status-syncer"))

	milvusController := &MilvusReconciler{
		Client:         mgr.GetClient(),
		Scheme:         mgr.GetScheme(),
		logger:         logger.WithName("milvus"),
		helmReconciler: helmReconciler,
		statusSyncer:   statusSyncer,
	}

	if err := milvusController.SetupWithManager(mgr); err != nil {
		logger.Error(err, "unable to setup milvus controller with manager", "controller", "Milvus")
		return err
	}

	mcController := milvuscluster.NewMilvusClusterReconciler(
		mgr.GetClient(),
		mgr.GetScheme(),
		logger.WithName("milvuscluster"),
	)

	if err := mcController.SetupWithManager(mgr); err != nil {
		logger.Error(err, "unable to setup milvuscluster controller with manager", "controller", "MilvusCluster")
		return err
	}

	if enableHook {
		if err := (&milvusv1beta1.Milvus{}).SetupWebhookWithManager(mgr); err != nil {
			logger.Error(err, "unable to create webhook", "webhook", "Milvus")
			return err
		}
		if err := (&milvusv1alpha1.Milvus{}).SetupWebhookWithManager(mgr); err != nil {
			logger.Error(err, "unable to create webhook", "webhook", "Milvus")
			return err
		}
	}

	return nil
}

// CommonInfo should be init when before time reconcile
type CommonInfo struct {
	OperatorImageInfo ImageInfo
	once              sync.Once
}

// global common info singleton
var globalCommonInfo CommonInfo

func (c *CommonInfo) InitIfNot(cli client.Client) {
	c.once.Do(func() {
		imageInfo, err := getOperatorImageInfo(cli)
		if err != nil {
			logf.Log.WithName("CommonInfo").Error(err, "get operator image info fail, use default")
			imageInfo = &DefaultOperatorImageInfo
		}
		c.OperatorImageInfo = *imageInfo
	})
}

// ImageInfo for image pulling
type ImageInfo struct {
	Image           string
	ImagePullPolicy corev1.PullPolicy
}

var (
	DefaultOperatorImageInfo = ImageInfo{
		Image:           "milvusdb/milvus-operator:main-latest",
		ImagePullPolicy: corev1.PullAlways,
	}
	ToolImage = ""
)

func getOperatorImageInfo(cli client.Client) (*ImageInfo, error) {
	if ToolImage != "" {
		return &ImageInfo{
			Image:           ToolImage,
			ImagePullPolicy: corev1.PullIfNotPresent,
		}, nil
	}
	deploy := appsv1.Deployment{}
	err := cli.Get(context.TODO(), NamespacedName(config.OperatorNamespace, config.OperatorName), &deploy)
	if err != nil {
		return nil, errors.Wrapf(err, "get operator deployment[%s/%s] fail", config.OperatorNamespace, config.OperatorName)
	}
	if len(deploy.Spec.Template.Spec.Containers) < 1 {
		return nil, errors.New("operator deployment has no container")
	}
	container := deploy.Spec.Template.Spec.Containers[0]
	return &ImageInfo{
		Image:           container.Image,
		ImagePullPolicy: container.ImagePullPolicy,
	}, nil
}
