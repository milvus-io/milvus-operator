package controllers

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/yaml"

	"github.com/milvus-io/milvus-operator/api/v1alpha1"
	"github.com/milvus-io/milvus-operator/pkg/config"
	"github.com/milvus-io/milvus-operator/pkg/util"
)

const (
	MilvusConfigYaml = "milvus.yaml"
)

func (r *MilvusClusterReconciler) getMinioAccessInfo(ctx context.Context, mc v1alpha1.MilvusCluster) (string, string) {
	secret := &corev1.Secret{}
	key := types.NamespacedName{Namespace: mc.Namespace, Name: mc.Spec.Dep.Storage.SecretRef}
	if err := r.Get(ctx, key, secret); err != nil {
		// TODO @shaoyue: handle error, or not get if no secret set
		r.logger.Error(err, "get minio secret error")
		return "", ""
	}

	return string(secret.Data[AccessKey]), string(secret.Data[SecretKey])

}

func (r *MilvusClusterReconciler) updateConfigMap(ctx context.Context, mc v1alpha1.MilvusCluster, configmap *corev1.ConfigMap) error {
	confYaml, err := util.GetTemplatedValues(config.GetMilvusClusterConfigTemplate(), mc)
	if err != nil {
		return err
	}

	conf := map[string]interface{}{}
	if err := yaml.Unmarshal(confYaml, &conf); err != nil {
		r.logger.Error(err, "yaml Unmarshal conf error")
		return err
	}

	key, secret := r.getMinioAccessInfo(ctx, mc)
	util.SetValue(conf, key, "minio", "accessKeyID")
	util.SetValue(conf, secret, "minio", "secretAccessKey")

	util.MergeValues(conf, mc.Spec.Conf.Data)
	util.SetStringSlice(conf, mc.Spec.Dep.Etcd.Endpoints, "etcd", "endpoints")

	host, port := util.GetHostPort(mc.Spec.Dep.Storage.Endpoint)
	util.SetValue(conf, host, "minio", "address")
	util.SetValue(conf, int64(port), "minio", "port")

	host, port = util.GetHostPort(mc.Spec.Dep.Pulsar.Endpoint)
	util.SetValue(conf, host, "pulsar", "address")
	util.SetValue(conf, int64(port), "pulsar", "port")

	milvusYaml, err := yaml.Marshal(conf)
	if err != nil {
		r.logger.Error(err, "yaml Marshal conf error")
		return err
	}

	configmap.Labels = MergeLabels(configmap.Labels, NewAppLabels(mc.Name))

	if err := ctrl.SetControllerReference(&mc, configmap, r.Scheme); err != nil {
		r.logger.Error(err, "configmap SetControllerReference error")
		return err
	}

	if configmap.Data == nil {
		configmap.Data = make(map[string]string)
	}
	configmap.Data[MilvusConfigYaml] = string(milvusYaml)

	return nil
}

//
func (r *MilvusClusterReconciler) ReconcileConfigMaps(ctx context.Context, mc v1alpha1.MilvusCluster) error {
	namespacedName := NamespacedName(mc.Namespace, mc.Name)
	old := &corev1.ConfigMap{}
	err := r.Get(ctx, namespacedName, old)
	if errors.IsNotFound(err) {
		new := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      mc.Name,
				Namespace: mc.Namespace,
			},
		}
		if err = r.updateConfigMap(ctx, mc, new); err != nil {
			return err
		}

		r.logger.Info("Create Configmap", "name", new.Name, "namespace", new.Namespace)
		return r.Create(ctx, new)
	} else if err != nil {
		return err
	}

	cur := old.DeepCopy()
	if err := r.updateConfigMap(ctx, mc, cur); err != nil {
		return err
	}

	if IsEqual(old, cur) {
		return nil
	}

	r.logger.Info("Update Configmap", "name", cur.Name, "namespace", cur.Namespace)
	return r.Update(ctx, cur)
}

func (r *MilvusReconciler) ReconcileConfigMaps(ctx context.Context, mil v1alpha1.Milvus) error {
	namespacedName := NamespacedName(mil.Namespace, mil.Name)
	old := &corev1.ConfigMap{}
	err := r.Get(ctx, namespacedName, old)
	if errors.IsNotFound(err) {
		new := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      mil.Name,
				Namespace: mil.Namespace,
			},
		}
		if err = r.updateConfigMap(ctx, mil, new); err != nil {
			return err
		}

		r.logger.Info("Create Configmap", "name", new.Name, "namespace", new.Namespace)
		return r.Create(ctx, new)
	} else if err != nil {
		return err
	}

	cur := old.DeepCopy()
	if err := r.updateConfigMap(ctx, mil, cur); err != nil {
		return err
	}

	if IsEqual(old, cur) {
		return nil
	}

	r.logger.Info("Update Configmap", "name", cur.Name, "namespace", cur.Namespace)
	return r.Update(ctx, cur)
}

func (r *MilvusReconciler) updateConfigMap(ctx context.Context, mil v1alpha1.Milvus, configmap *corev1.ConfigMap) error {
	confYaml, err := util.GetTemplatedValues(config.GetMilvusConfigTemplate(), mil)
	if err != nil {
		return err
	}

	conf := map[string]interface{}{}
	if err := yaml.Unmarshal(confYaml, &conf); err != nil {
		r.logger.Error(err, "yaml Unmarshal conf error")
		return err
	}

	key, secret := r.getMinioAccessInfo(ctx, mil)
	util.SetValue(conf, key, "minio", "accessKeyID")
	util.SetValue(conf, secret, "minio", "secretAccessKey")

	util.MergeValues(conf, mil.Spec.Conf.Data)
	util.SetStringSlice(conf, mil.Spec.Dep.Etcd.Endpoints, "etcd", "endpoints")

	host, port := util.GetHostPort(mil.Spec.Dep.Storage.Endpoint)
	util.SetValue(conf, host, "minio", "address")
	util.SetValue(conf, int64(port), "minio", "port")

	milvusYaml, err := yaml.Marshal(conf)
	if err != nil {
		r.logger.Error(err, "yaml Marshal conf error")
		return err
	}

	configmap.Labels = MergeLabels(configmap.Labels, NewAppLabels(mil.Name))

	if err := ctrl.SetControllerReference(&mil, configmap, r.Scheme); err != nil {
		r.logger.Error(err, "configmap SetControllerReference error")
		return err
	}

	if configmap.Data == nil {
		configmap.Data = make(map[string]string)
	}
	configmap.Data[MilvusConfigYaml] = string(milvusYaml)

	return nil
}

func (r *MilvusReconciler) getMinioAccessInfo(ctx context.Context, mil v1alpha1.Milvus) (string, string) {
	secret := &corev1.Secret{}
	key := types.NamespacedName{Namespace: mil.Namespace, Name: mil.Spec.Dep.Storage.SecretRef}
	if err := r.Get(ctx, key, secret); err != nil {
		// TODO @shaoyue: handle error, or not get if no secret set
		r.logger.Error(err, "get minio secret error")
		return "", ""
	}

	return string(secret.Data[AccessKey]), string(secret.Data[SecretKey])

}
