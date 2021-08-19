package controllers

import (
	"context"
	"fmt"

	"github.com/milvus-io/milvus-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	MilvusConfigVolumeName   = "milvus-config"
	MilvusConfigMountPath    = "/milvus/configs/milvus.yaml"
	MilvusConfigMountSubPath = "milvus.yaml"
	S3AccessKey              = "accesskey"
	S3SecretKey              = "secretkey"
)

var (
	MilvusConfigMapMode int32 = 420
)

func GetS3SecretRefEnv(secretRef string) []corev1.EnvVar {
	env := []corev1.EnvVar{}
	env = append(env, corev1.EnvVar{
		Name: "MINIO_ACCESS_KEY",
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: secretRef,
				},
				Key: S3AccessKey,
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
				Key: S3SecretKey,
			},
		},
	})
	return env
}

func (r *MilvusClusterReconciler) updateDeployment(
	mc v1alpha1.MilvusCluster, deployment *appsv1.Deployment, component MilvusComponent,
) error {
	appLabels := NewAppLabels(mc.Name, component)

	deployment.Labels = MergeLabels(deployment.Labels, appLabels)
	if err := ctrl.SetControllerReference(&mc, deployment, r.Scheme); err != nil {
		return err
	}

	var replicas int32 = 1
	switch component {
	case Proxy:
		replicas = *mc.Spec.Proxy.Replicas
	case QueryNode:
		replicas = *mc.Spec.QueryNode.Replicas
	case DataNode:
		replicas = *mc.Spec.DataNode.Replicas
	case IndexNode:
		replicas = *mc.Spec.IndexNode.Replicas
	case RootCoord:
		replicas = *mc.Spec.RootCoord.Replicas
	case QueryCoord:
		replicas = *mc.Spec.QueryCoord.Replicas
	case DataCoord:
		replicas = *mc.Spec.DataCoord.Replicas
	case IndexCoord:
		replicas = *mc.Spec.IndexCoord.Replicas
	}
	deployment.Spec.Replicas = &replicas

	if deployment.Spec.Selector == nil {
		deployment.Spec.Selector = new(metav1.LabelSelector)
		deployment.Spec.Selector.MatchLabels = appLabels
	}
	deployment.Spec.Template.Labels = MergeLabels(deployment.Spec.Template.Labels, appLabels)

	// update configmap volume
	volumeIdx := GetVolumeIndex(deployment.Spec.Template.Spec.Volumes, MilvusConfigVolumeName)
	volume := corev1.Volume{
		Name: MilvusConfigVolumeName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: GetConfigMapInstanceName(mc.Name),
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
	env := mc.Spec.Env
	env = append(env, GetS3SecretRefEnv(mc.Spec.S3.SecretRef)...)
	container.Env = MergeEnvVar(container.Env, env)
	container.Ports = MergeContainerPort(container.Ports, component.GetContainerPorts())

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

	if mc.Spec.ImagePullPolicy != nil {
		container.ImagePullPolicy = *mc.Spec.ImagePullPolicy
	}

	container.Image = mc.Spec.Image

	return nil
}

func (r *MilvusClusterReconciler) ReconcileComponentDeployment(
	ctx context.Context, mc *v1alpha1.MilvusCluster, component MilvusComponent,
) error {
	namespacedName := types.NamespacedName{
		Namespace: mc.Namespace,
		Name:      component.GetDeploymentInstanceName(mc.Name),
	}
	old := &appsv1.Deployment{}
	err := r.Get(ctx, namespacedName, old)
	if errors.IsNotFound(err) {
		new := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      namespacedName.Name,
				Namespace: namespacedName.Namespace,
			},
		}
		if err := r.updateDeployment(*mc, new, component); err != nil {
			return err
		}

		return r.Create(ctx, new)
	} else if err != nil {
		return err
	}

	cur := old.DeepCopy()
	if err := r.updateDeployment(*mc, cur, component); err != nil {
		return err
	}

	if IsEqual(old, cur) {
		r.logger.Info("Equal")
		return nil
	}

	return r.Update(ctx, cur)
}

func (r *MilvusClusterReconciler) ReconcileDeployments(ctx context.Context, mc *v1alpha1.MilvusCluster) error {
	g, gtx := NewGroup(ctx)
	for _, component := range MilvusComponents {
		g.Go(func(ctx context.Context, mc *v1alpha1.MilvusCluster, component MilvusComponent) func() error {
			return func() error {
				return r.ReconcileComponentDeployment(ctx, mc, component)
			}
		}(gtx, mc, component))
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("reconcile milvus deployments: %w", err)
	}

	return nil
}
