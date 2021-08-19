package controllers

import (
	"context"
	"fmt"

	"github.com/milvus-io/milvus-operator/api/v1alpha1"
	"github.com/milvus-io/milvus-operator/pkg/config"
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
)

func GetConfigMapInstanceName(instance string) string {
	return instance
}

func (r *MilvusClusterReconciler) updateConfigMap(mc v1alpha1.MilvusCluster, configmap *corev1.ConfigMap) error {
	configuration := milvus.MilvusConfig{
		Etcd: milvus.MilvusConfigEtcd{
			Endpoints: mc.Spec.Etcd.Endpoints,
			RootPath:  mc.Spec.Etcd.RootPath,
		},
		Minio:  milvus.NewMinioConfig(mc.Spec.S3.Endpoint, mc.Spec.S3.Bucket, !mc.Spec.S3.Insecure),
		Pulsar: milvus.NewPulsarConfig(mc.Spec.Pulsar.Endpoint),
		Log: milvus.MilvusConfigLog{
			Level: mc.Spec.LogLevel,
		},
	}
	configuration.RootCoord.Address = fmt.Sprintf("%s-milvus-rootcoord", mc.Name)
	configuration.DataCoord.Address = fmt.Sprintf("%s-milvus-datacoord", mc.Name)
	configuration.QueryCoord.Address = fmt.Sprintf("%s-milvus-querycoord", mc.Name)
	configuration.IndexCoord.Address = fmt.Sprintf("%s-milvus-indexcoord", mc.Name)

	yaml, err := configuration.GetTemplatedConfig(config.GetTemplate(TemplateMilvus))
	if err != nil {
		return err
	}
	if err := ctrl.SetControllerReference(&mc, configmap, r.Scheme); err != nil {
		return err
	}

	if configmap.Data == nil {
		configmap.Data = make(map[string]string)
	}
	configmap.Data[MilvusConfigYaml] = yaml

	return nil
}

//
func (r *MilvusClusterReconciler) ReconcileConfigMaps(ctx context.Context, mc *v1alpha1.MilvusCluster) error {
	namespacedName := types.NamespacedName{Namespace: mc.Namespace, Name: mc.Name}
	old := &corev1.ConfigMap{}
	err := r.Get(ctx, namespacedName, old)
	if errors.IsNotFound(err) {
		new := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      GetConfigMapInstanceName(mc.Name),
				Namespace: mc.Namespace,
			},
		}
		if err = r.updateConfigMap(*mc, new); err != nil {
			return err
		}

		return r.Create(ctx, new)
	} else if err != nil {
		return err
	}

	cur := old.DeepCopy()
	if err := r.updateConfigMap(*mc, cur); err != nil {
		return err
	}

	if IsEqual(old, cur) {
		return nil
	}

	return r.Update(ctx, cur)
}
