package controllers

import (
	"context"
	"fmt"

	"github.com/milvus-io/milvus-operator/api/v1alpha1"
	"github.com/milvus-io/milvus-operator/pkg/milvus"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	TemplateMilvus   = "milvus.yaml.tmpl"
	MilvusConfigYaml = "milvus.yaml"
	AccessKeyID      = "accessKeyID"
	SecretAccessKey  = "secretAccessKey"
)

func (r *MilvusClusterReconciler) updateConfigMap(mc *v1alpha1.MilvusCluster, configmap *corev1.ConfigMap) error {
	configuration := milvus.MilvusConfig{
		Etcd: milvus.MilvusConfigEtcd{
			Endpoints: mc.Spec.Etcd.Endpoints,
			RootPath:  mc.Spec.Etcd.RootPath,
		},
		Minio:  milvus.NewMinioConfig(mc.Spec.S3.Endpoint, mc.Spec.S3.Bucket, mc.Spec.S3.Insecure),
		Pulsar: milvus.NewPulsarConfig(mc.Spec.Pulsar.Endpoint),
		Log: milvus.MilvusConfigLog{
			Level: mc.Spec.LogLevel,
		},
	}
	configuration.RootCoord.Address = fmt.Sprintf("%s-milvus-rootcoord", mc.Name)
	configuration.DataCoord.Address = fmt.Sprintf("%s-milvus-datacoord", mc.Name)
	configuration.QueryCoord.Address = fmt.Sprintf("%s-milvus-querycoord", mc.Name)
	configuration.IndexCoord.Address = fmt.Sprintf("%s-milvus-indexcoord", mc.Name)

	yaml, err := configuration.GetTemplatedConfig(r.config.GetTemplate(TemplateMilvus))
	if err != nil {
		return err
	}
	if err := ctrl.SetControllerReference(mc, configmap, r.Scheme); err != nil {
		return err
	}
	configmap.Data[MilvusConfigYaml] = yaml

	return nil
}

//
func (r *MilvusClusterReconciler) reconcilerConfigMap(ctx context.Context, mc *v1alpha1.MilvusCluster) error {
	namespacedName := types.NamespacedName{Namespace: mc.Namespace, Name: mc.Name}
	cur := &corev1.ConfigMap{}
	if err := r.Get(ctx, namespacedName, cur); err != nil {
		if errors.IsNotFound(err) {
			new := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      mc.Name,
					Namespace: mc.Namespace,
				},
			}

			if err = r.updateConfigMap(mc, new); err != nil {
				return err
			}

			return r.Create(ctx, new)
		}
		return err
	}

	if err := r.updateConfigMap(mc, cur); err != nil {
		return err
	}

	return r.Update(ctx, cur)
}

// PodRunningAndReady returns whether a pod is running and each container has
// passed it's ready state.
func PodRunningAndReady(pod corev1.Pod) (bool, error) {
	switch pod.Status.Phase {
	case corev1.PodFailed, corev1.PodSucceeded:
		return false, fmt.Errorf("pod completed")
	case corev1.PodRunning:
		for _, cond := range pod.Status.Conditions {
			if cond.Type != corev1.PodReady {
				continue
			}
			return cond.Status == corev1.ConditionTrue, nil
		}
		return false, fmt.Errorf("pod ready condition not found")
	}
	return false, nil
}
