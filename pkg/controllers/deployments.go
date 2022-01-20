package controllers

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/milvus-io/milvus-operator/apis/milvus.io/v1alpha1"
)

const (
	MilvusConfigVolumeName       = "milvus-config"
	MilvusOriginalConfigPath     = "/milvus/configs/milvus.yaml"
	MilvusUserConfigMountPath    = "/milvus/configs/user.yaml"
	MilvusUserConfigMountSubPath = "user.yaml"
	AccessKey                    = "accesskey"
	SecretKey                    = "secretkey"
	AnnotationCheckSum           = "checksum/config"

	ToolsVolumeName = "tools"
	ToolsMountPath  = "/milvus/tools"
	RunScriptPath   = ToolsMountPath + "/run.sh"
	MergeToolPath   = ToolsMountPath + "/merge"
)

var (
	MilvusConfigMapMode int32 = 420
)

func GetStorageSecretRefEnv(secretRef string) []corev1.EnvVar {
	env := []corev1.EnvVar{}
	env = append(env, corev1.EnvVar{
		Name: "MINIO_ACCESS_KEY",
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: secretRef,
				},
				Key: AccessKey,
			},
		},
	})
	env = append(env, corev1.EnvVar{
		Name: "MINIO_SECRET_KEY",
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: secretRef,
				},
				Key: SecretKey,
			},
		},
	})
	return env
}

func (r *MilvusClusterReconciler) updateDeployment(
	mc v1alpha1.MilvusCluster, deployment *appsv1.Deployment, component MilvusComponent,
) error {
	appLabels := NewComponentAppLabels(mc.Name, component.String())

	deployment.Labels = MergeLabels(deployment.Labels, appLabels)
	if err := ctrl.SetControllerReference(&mc, deployment, r.Scheme); err != nil {
		return err
	}

	deployment.Spec.Replicas = component.GetReplicas(mc.Spec)
	deployment.Spec.Strategy = component.GetDeploymentStrategy()

	if deployment.Spec.Selector == nil {
		deployment.Spec.Selector = new(metav1.LabelSelector)
		deployment.Spec.Selector.MatchLabels = appLabels
	}

	deployment.Spec.Template.Spec.InitContainers = []corev1.Container{
		getInitContainer(),
	}
	deployment.Spec.Template.Labels = MergeLabels(deployment.Spec.Template.Labels, appLabels)

	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = map[string]string{}
	}
	deployment.Spec.Template.Annotations[AnnotationCheckSum] = GetConfCheckSum(mc.Spec)

	// update configmap volume
	volumes := &deployment.Spec.Template.Spec.Volumes
	addVolume(volumes, configVolumeByName(mc.Name))
	addVolume(volumes, toolVolume)

	// update component container
	containerIdx := GetContainerIndex(deployment.Spec.Template.Spec.Containers, component.GetContainerName())
	if containerIdx < 0 {
		deployment.Spec.Template.Spec.Containers = append(
			deployment.Spec.Template.Spec.Containers,
			corev1.Container{Name: component.GetContainerName()},
		)
		containerIdx = len(deployment.Spec.Template.Spec.Containers) - 1
	}
	container := &deployment.Spec.Template.Spec.Containers[containerIdx]
	container.Args = []string{RunScriptPath, "milvus", "run", component.String()}
	env := component.GetEnv(mc.Spec)
	env = append(env, GetStorageSecretRefEnv(mc.Spec.Dep.Storage.SecretRef)...)
	container.Env = MergeEnvVar(container.Env, env)
	container.Ports = MergeContainerPort(container.Ports, component.GetContainerPorts(mc.Spec))

	addVolumeMount(&container.VolumeMounts, configVolumeMount)
	addVolumeMount(&container.VolumeMounts, toolVolumeMount)

	container.ImagePullPolicy = component.GetImagePullPolicy(mc.Spec)
	container.Image = component.GetImage(mc.Spec)
	container.Resources = component.GetResources(mc.Spec)
	container.LivenessProbe = GetLivenessProbe()
	container.ReadinessProbe = GetReadinessProbe()
	deployment.Spec.Template.Spec.ImagePullSecrets = component.GetImagePullSecrets(mc.Spec)

	return nil
}

func (r *MilvusClusterReconciler) ReconcileComponentDeployment(
	ctx context.Context, mc v1alpha1.MilvusCluster, component MilvusComponent,
) error {
	namespacedName := NamespacedName(mc.Namespace, component.GetDeploymentInstanceName(mc.Name))
	old := &appsv1.Deployment{}
	err := r.Get(ctx, namespacedName, old)
	if errors.IsNotFound(err) {
		new := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      namespacedName.Name,
				Namespace: namespacedName.Namespace,
			},
		}
		if err := r.updateDeployment(mc, new, component); err != nil {
			return err
		}

		r.logger.Info("Create Deployment", "name", new.Name, "namespace", new.Namespace)
		return r.Create(ctx, new)
	} else if err != nil {
		return err
	}

	cur := old.DeepCopy()
	if err := r.updateDeployment(mc, cur, component); err != nil {
		return err
	}

	if IsEqual(old, cur) {
		//r.logger.Info("Equal", "cur", cur.Name)
		return nil
	}

	/* if config.IsDebug() {
		diff, err := diffObject(old, cur)
		if err == nil {
			r.logger.Info("Deployment diff", "name", cur.Name, "namespace", cur.Namespace, "diff", string(diff))
		}
	} */

	r.logger.Info("Update Deployment", "name", cur.Name, "namespace", cur.Namespace)
	return r.Update(ctx, cur)
}

func (r *MilvusClusterReconciler) ReconcileDeployments(ctx context.Context, mc v1alpha1.MilvusCluster) error {
	g, gtx := NewGroup(ctx)
	for _, component := range MilvusComponents {
		g.Go(WarppedReconcileComponentFunc(r.ReconcileComponentDeployment, gtx, mc, component))
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("reconcile milvus deployments: %w", err)
	}

	return nil
}

func (r *MilvusReconciler) ReconcileDeployments(ctx context.Context, mil v1alpha1.Milvus) error {
	namespacedName := NamespacedName(mil.Namespace, mil.Name)
	old := &appsv1.Deployment{}
	err := r.Get(ctx, namespacedName, old)
	if errors.IsNotFound(err) {
		new := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      namespacedName.Name,
				Namespace: namespacedName.Namespace,
			},
		}
		if err := r.updateDeployment(mil, new); err != nil {
			return err
		}

		r.logger.Info("Create Deployment", "name", new.Name, "namespace", new.Namespace)
		return r.Create(ctx, new)
	} else if err != nil {
		return err
	}

	cur := old.DeepCopy()
	if err := r.updateDeployment(mil, cur); err != nil {
		return err
	}

	if IsEqual(old, cur) {
		return nil
	}

	r.logger.Info("Update Deployment", "name", cur.Name, "namespace", cur.Namespace)
	return r.Update(ctx, cur)
}

func (r *MilvusReconciler) updateDeployment(
	mc v1alpha1.Milvus, deployment *appsv1.Deployment,
) error {
	appLabels := NewComponentAppLabels(mc.Name, MilvusName)

	deployment.Labels = MergeLabels(deployment.Labels, appLabels)
	if err := ctrl.SetControllerReference(&mc, deployment, r.Scheme); err != nil {
		return err
	}

	deployment.Spec.Replicas = int32Ptr(1)
	deployment.Spec.Strategy = appsv1.DeploymentStrategy{
		Type: appsv1.RecreateDeploymentStrategyType,
	}

	if deployment.Spec.Selector == nil {
		deployment.Spec.Selector = new(metav1.LabelSelector)
		deployment.Spec.Selector.MatchLabels = appLabels
	}
	deployment.Spec.Template.Spec.InitContainers = []corev1.Container{
		getInitContainer(),
	}
	deployment.Spec.Template.Labels = MergeLabels(deployment.Spec.Template.Labels, appLabels)

	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = map[string]string{}
	}
	deployment.Spec.Template.Annotations[AnnotationCheckSum] = GetMilvusConfCheckSum(mc.Spec)

	// update configmap volume
	volumes := &deployment.Spec.Template.Spec.Volumes
	addVolume(volumes, configVolumeByName(mc.Name))
	addVolume(volumes, toolVolume)

	// update component container
	containerIdx := GetContainerIndex(deployment.Spec.Template.Spec.Containers, MilvusName)
	if containerIdx < 0 {
		deployment.Spec.Template.Spec.Containers = append(
			deployment.Spec.Template.Spec.Containers,
			corev1.Container{Name: MilvusName},
		)
		containerIdx = len(deployment.Spec.Template.Spec.Containers) - 1
	}
	container := &deployment.Spec.Template.Spec.Containers[containerIdx]
	container.Args = []string{RunScriptPath, "milvus", "run", "standalone"}
	env := mc.Spec.Env
	env = append(env, GetStorageSecretRefEnv(mc.Spec.Dep.Storage.SecretRef)...)
	container.Env = MergeEnvVar(container.Env, env)
	container.Ports = MergeContainerPort(container.Ports, []corev1.ContainerPort{
		{
			Name:          MilvusName,
			ContainerPort: MilvusPort,
			Protocol:      corev1.ProtocolTCP,
		},
		{
			Name:          MetricPortName,
			ContainerPort: MetricPort,
			Protocol:      corev1.ProtocolTCP,
		},
	})

	addVolumeMount(&container.VolumeMounts, configVolumeMount)
	addVolumeMount(&container.VolumeMounts, toolVolumeMount)

	if mc.Spec.ImagePullPolicy != nil {
		container.ImagePullPolicy = *mc.Spec.ImagePullPolicy
	}
	container.Image = mc.Spec.Image
	if mc.Spec.Resources != nil {
		container.Resources = *mc.Spec.Resources
	}
	container.LivenessProbe = GetLivenessProbe()
	container.ReadinessProbe = GetReadinessProbe()
	deployment.Spec.Template.Spec.ImagePullSecrets = mc.Spec.ImagePullSecrets

	return nil
}

func addVolume(volumes *[]corev1.Volume, volume corev1.Volume) {
	volumeIdx := GetVolumeIndex(*volumes, volume.Name)
	if volumeIdx < 0 {
		*volumes = append(*volumes, volume)
	} else {
		(*volumes)[volumeIdx] = volume
	}
}

func addVolumeMount(volumeMounts *[]corev1.VolumeMount, volumeMount corev1.VolumeMount) {
	volumeMountIdx := GetVolumeMountIndex(*volumeMounts, volumeMount.MountPath)
	if volumeMountIdx < 0 {
		*volumeMounts = append(*volumeMounts, volumeMount)
	} else {
		(*volumeMounts)[volumeMountIdx] = volumeMount
	}
}

func getInitContainer() corev1.Container {
	imageInfo := globalCommonInfo.OperatorImageInfo
	copyToolsCommand := []string{"/cp", "/run.sh,/merge", RunScriptPath + "," + MergeToolPath}
	return corev1.Container{
		Name:            "config",
		Image:           imageInfo.Image,
		ImagePullPolicy: imageInfo.ImagePullPolicy,
		Command:         copyToolsCommand,
		VolumeMounts: []corev1.VolumeMount{
			toolVolumeMount,
		},
	}
}

var (
	toolVolume = corev1.Volume{
		Name: ToolsVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
	toolVolumeMount = corev1.VolumeMount{
		Name:      ToolsVolumeName,
		MountPath: ToolsMountPath,
	}

	configVolumeMount = corev1.VolumeMount{
		Name:      MilvusConfigVolumeName,
		ReadOnly:  true,
		MountPath: MilvusUserConfigMountPath,
		SubPath:   MilvusUserConfigMountSubPath,
	}
)

func configVolumeByName(name string) corev1.Volume {
	return corev1.Volume{
		Name: MilvusConfigVolumeName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: name,
				},
				DefaultMode: &MilvusConfigMapMode,
			},
		},
	}
}
