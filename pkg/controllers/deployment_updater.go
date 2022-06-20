package controllers

import (
	"github.com/milvus-io/milvus-operator/apis/milvus.io/v1beta1"
	pkgErrs "github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

type deploymentUpdater interface {
	GetIntanceName() string
	GetComponentName() string
	GetPortName() string
	GetControllerRef() metav1.Object
	GetScheme() *runtime.Scheme
	GetReplicas() *int32
	GetDeploymentStrategy() appsv1.DeploymentStrategy
	GetConfCheckSum() string
	GetMergedComponentSpec() ComponentSpec
	GetArgs() []string
	GetSecretRef() string
	GetPersistenceConfig() *v1beta1.Persistence
	GetMilvus() *v1beta1.Milvus
}

func updateDeployment(deployment *appsv1.Deployment, updater deploymentUpdater) error {
	appLabels := NewComponentAppLabels(updater.GetIntanceName(), updater.GetComponentName())
	deployment.Labels = MergeLabels(deployment.Labels, appLabels)
	if err := ctrl.SetControllerReference(updater.GetControllerRef(), deployment, updater.GetScheme()); err != nil {
		return pkgErrs.Wrap(err, "set controller reference")
	}

	deployment.Spec.Replicas = updater.GetReplicas()
	deployment.Spec.Strategy = updater.GetDeploymentStrategy()
	if deployment.Spec.Selector == nil {
		deployment.Spec.Selector = new(metav1.LabelSelector)
		deployment.Spec.Selector.MatchLabels = appLabels
	}
	template := &deployment.Spec.Template
	configContainerIdx := GetContainerIndex(template.Spec.InitContainers, configContainerName)
	spec := updater.GetMilvus().Spec
	if configContainerIdx < 0 {
		if len(template.Spec.InitContainers) < 1 {
			template.Spec.InitContainers = []corev1.Container{}
		}
		template.Spec.InitContainers = append(template.Spec.InitContainers, getInitContainer(spec.Com.ToolImage))
	} else if spec.Com.UpdateToolImage {
		template.Spec.InitContainers[configContainerIdx] = getInitContainer(spec.Com.ToolImage)
	}
	if template.Labels == nil {
		template.Labels = map[string]string{}
	}
	template.Labels = MergeLabels(template.Labels, appLabels)

	if template.Annotations == nil {
		template.Annotations = map[string]string{}
	}
	template.Annotations[AnnotationCheckSum] = updater.GetConfCheckSum()

	// update configmap volume
	volumes := &template.Spec.Volumes
	addVolume(volumes, configVolumeByName(updater.GetIntanceName()))
	addVolume(volumes, toolVolume)
	if persistence := updater.GetPersistenceConfig(); persistence != nil && persistence.Enabled {
		if len(persistence.PersistentVolumeClaim.ExistingClaim) > 0 {
			addVolume(volumes, persisentVolumeByName(persistence.PersistentVolumeClaim.ExistingClaim))
		} else {
			addVolume(volumes, persisentVolumeByName(getPVCNameByInstName(updater.GetIntanceName())))
		}
	}

	mergedComSpec := updater.GetMergedComponentSpec()
	template.Spec.Affinity = mergedComSpec.Affinity
	template.Spec.Tolerations = mergedComSpec.Tolerations
	template.Spec.NodeSelector = mergedComSpec.NodeSelector
	template.Spec.ImagePullSecrets = mergedComSpec.ImagePullSecrets
	template.Spec.ServiceAccountName = mergedComSpec.ServiceAccountName
	// update component container
	containerIdx := GetContainerIndex(template.Spec.Containers, updater.GetComponentName())
	if containerIdx < 0 {
		template.Spec.Containers = append(
			template.Spec.Containers,
			corev1.Container{Name: updater.GetComponentName()},
		)
		containerIdx = len(template.Spec.Containers) - 1
	}
	container := &template.Spec.Containers[containerIdx]
	container.Args = updater.GetArgs()
	env := mergedComSpec.Env
	env = append(env, GetStorageSecretRefEnv(updater.GetSecretRef())...)
	container.Env = MergeEnvVar(container.Env, env)
	metricPort := corev1.ContainerPort{
		Name:          MetricPortName,
		ContainerPort: MetricPort,
		Protocol:      corev1.ProtocolTCP,
	}
	componentName := updater.GetComponentName()
	if componentName == ProxyName || componentName == StandaloneName {
		container.Ports = []corev1.ContainerPort{
			{
				Name:          updater.GetPortName(),
				ContainerPort: MilvusPort,
				Protocol:      corev1.ProtocolTCP,
			},
			metricPort,
		}
	} else {
		container.Ports = []corev1.ContainerPort{metricPort}
	}

	addVolumeMount(&container.VolumeMounts, configVolumeMount)
	addVolumeMount(&container.VolumeMounts, toolVolumeMount)
	if persistence := updater.GetPersistenceConfig(); persistence != nil && persistence.Enabled {
		addVolumeMount(&container.VolumeMounts, persistentVolumeMount(*persistence))
	}

	container.ImagePullPolicy = *mergedComSpec.ImagePullPolicy
	container.Image = mergedComSpec.Image
	container.Resources = *mergedComSpec.Resources
	container.LivenessProbe = GetLivenessProbe()
	container.ReadinessProbe = GetReadinessProbe()

	return nil
}

// milvusDeploymentUpdater implements deploymentUpdater for milvus
type milvusDeploymentUpdater struct {
	v1beta1.Milvus
	scheme    *runtime.Scheme
	component MilvusComponent
}

func newMilvusDeploymentUpdater(m v1beta1.Milvus, scheme *runtime.Scheme, component MilvusComponent) *milvusDeploymentUpdater {
	return &milvusDeploymentUpdater{
		Milvus:    m,
		scheme:    scheme,
		component: component,
	}
}

func (m milvusDeploymentUpdater) GetPersistenceConfig() *v1beta1.Persistence {
	if m.Milvus.Spec.Dep.RocksMQ.Persistence.Enabled {
		return &m.Milvus.Spec.Dep.RocksMQ.Persistence
	}
	return nil
}

func (m milvusDeploymentUpdater) GetIntanceName() string {
	return m.Name
}
func (m milvusDeploymentUpdater) GetComponentName() string {
	return m.component.GetName()
}

func (m milvusDeploymentUpdater) GetPortName() string {
	return m.component.GetPortName()
}

func (m milvusDeploymentUpdater) GetControllerRef() metav1.Object {
	return &m.Milvus
}
func (m milvusDeploymentUpdater) GetScheme() *runtime.Scheme {
	return m.scheme
}
func (m milvusDeploymentUpdater) GetReplicas() *int32 {
	return m.component.GetReplicas(m.Spec)
}
func (m milvusDeploymentUpdater) GetDeploymentStrategy() appsv1.DeploymentStrategy {
	return m.component.GetDeploymentStrategy()
}
func (m milvusDeploymentUpdater) GetConfCheckSum() string {
	return GetConfCheckSum(m.Spec)
}
func (m milvusDeploymentUpdater) GetMergedComponentSpec() ComponentSpec {
	return MergeComponentSpec(
		m.component.GetComponentSpec(m.Spec),
		m.Spec.Com.ComponentSpec,
	)
}
func (m milvusDeploymentUpdater) GetArgs() []string {
	return append([]string{RunScriptPath, "milvus", "run"}, m.component.GetRunCommands()...)
}
func (m milvusDeploymentUpdater) GetSecretRef() string {
	return m.Spec.Dep.Storage.SecretRef
}

func (m milvusDeploymentUpdater) GetMilvus() *v1beta1.Milvus {
	return &m.Milvus
}
