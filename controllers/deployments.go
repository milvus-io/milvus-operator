package controllers

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/milvus-io/milvus-operator/api/v1alpha1"
)

const (
	MilvusConfigVolumeName   = "milvus-config"
	MilvusConfigMountPath    = "/milvus/configs/milvus.yaml"
	MilvusConfigMountSubPath = "milvus.yaml"
	AccessKey                = "access-key"
	SecretKey                = "secret-key"
	AnnotationCheckSum       = "checksum/config"
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

	if deployment.Spec.Selector == nil {
		deployment.Spec.Selector = new(metav1.LabelSelector)
		deployment.Spec.Selector.MatchLabels = appLabels
	}
	deployment.Spec.Template.Labels = MergeLabels(deployment.Spec.Template.Labels, appLabels)

	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = map[string]string{}
	}
	deployment.Spec.Template.Annotations[AnnotationCheckSum] = component.GetConfCheckSum(mc.Spec)

	// update configmap volume
	volumeIdx := GetVolumeIndex(deployment.Spec.Template.Spec.Volumes, MilvusConfigVolumeName)
	volume := corev1.Volume{
		Name: MilvusConfigVolumeName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: mc.Name,
				},
				DefaultMode: &MilvusConfigMapMode,
			},
		},
	}
	if volumeIdx < 0 {
		deployment.Spec.Template.Spec.Volumes = append(deployment.Spec.Template.Spec.Volumes, volume)
	} else {
		deployment.Spec.Template.Spec.Volumes[volumeIdx] = volume
	}

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
	container.Args = []string{"milvus", "run", component.String()}
	env := component.GetEnv(mc.Spec)
	env = append(env, GetStorageSecretRefEnv(mc.Spec.Dep.Storage.SecretRef)...)
	container.Env = MergeEnvVar(container.Env, env)
	container.Ports = MergeContainerPort(container.Ports, component.GetContainerPorts(mc.Spec))

	milvusVolumeMount := corev1.VolumeMount{
		Name:      MilvusConfigVolumeName,
		ReadOnly:  true,
		MountPath: MilvusConfigMountPath,
		SubPath:   MilvusConfigMountSubPath,
	}
	mountIdx := GetVolumeMountIndex(container.VolumeMounts, milvusVolumeMount.MountPath)
	if mountIdx < 0 {
		container.VolumeMounts = append(container.VolumeMounts, milvusVolumeMount)
	} else {
		container.VolumeMounts[mountIdx] = milvusVolumeMount
	}

	container.ImagePullPolicy = component.GetImagePullPolicy(mc.Spec)
	container.Image = component.GetImage(mc.Spec)
	container.Resources = component.GetResources(mc.Spec)
	container.LivenessProbe = component.GetLivenessProbe()
	container.ReadinessProbe = component.GetReadinessProbe()
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
