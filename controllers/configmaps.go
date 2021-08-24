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

func SetGRPCConfig(grpc *milvus.MilvusConfigGRPC, mc *v1alpha1.ConfigGRPC) {
	if mc != nil {
		grpc.ClientMaxRecvSize = mc.ClientMaxRecvSize
		grpc.ClientMaxSendSize = mc.ClientMaxSendSize
		grpc.ServerMaxRecvSize = mc.ServerMaxRecvSize
		grpc.ServerMaxSendSize = mc.ServerMaxSendSize
	}
}

func GetMilvusConfigFrom(mc v1alpha1.MilvusCluster) milvus.MilvusConfig {
	conf := milvus.MilvusConfig{
		Etcd: milvus.MilvusConfigEtcd{
			Endpoints:               mc.Spec.Etcd.Endpoints,
			RootPath:                mc.Spec.Etcd.RootPath,
			KVSubPath:               mc.Spec.Etcd.KVSubPath,
			MetaSubPath:             mc.Spec.Etcd.MetaSubPath,
			SegmentBinlogSubPath:    mc.Spec.Etcd.SegmentBinlogSubPath,
			CollectionBinlogSubPath: mc.Spec.Etcd.CollectionBinlogSubPath,
			FlushStreamPosSubPath:   mc.Spec.Etcd.FlushStreamPosSubPath,
			StatsStreamPosSubPath:   mc.Spec.Etcd.StatsStreamPosSubPath,
		},
		Minio:  milvus.NewMinioConfig(mc.Spec.S3.Endpoint, mc.Spec.S3.Bucket, !mc.Spec.S3.Insecure),
		Pulsar: milvus.NewPulsarConfig(mc.Spec.Pulsar.Endpoint, mc.Spec.Pulsar.MaxMessageSize),
		Log: milvus.MilvusConfigLog{
			Level: mc.Spec.LogLevel,
		},
	}
	conf.RootCoord.Address = fmt.Sprintf("%s-milvus-rootcoord", mc.Name)
	conf.RootCoord.Port = mc.Spec.RootCoord.Port
	SetGRPCConfig(&conf.RootCoord.GRPC, mc.Spec.RootCoord.GRPC)

	conf.DataCoord.Address = fmt.Sprintf("%s-milvus-datacoord", mc.Name)
	conf.DataCoord.Port = mc.Spec.DataCoord.Port
	SetGRPCConfig(&conf.DataCoord.GRPC, mc.Spec.DataCoord.GRPC)

	conf.QueryCoord.Address = fmt.Sprintf("%s-milvus-querycoord", mc.Name)
	conf.QueryCoord.Port = mc.Spec.QueryCoord.Port
	SetGRPCConfig(&conf.QueryCoord.GRPC, mc.Spec.QueryCoord.GRPC)

	conf.IndexCoord.Address = fmt.Sprintf("%s-milvus-indexcoord", mc.Name)
	conf.IndexCoord.Port = mc.Spec.IndexCoord.Port
	SetGRPCConfig(&conf.IndexCoord.GRPC, mc.Spec.IndexCoord.GRPC)

	conf.DataNode.Port = mc.Spec.DataNode.Port
	SetGRPCConfig(&conf.DataNode.GRPC, mc.Spec.DataNode.GRPC)
	conf.DataNode.InsertBufSize = mc.Spec.DataNode.InsertBufSize

	conf.QueryNode.Port = mc.Spec.QueryNode.Port
	SetGRPCConfig(&conf.QueryNode.GRPC, mc.Spec.QueryNode.GRPC)
	conf.QueryNode.GracefulTime = mc.Spec.QueryNode.GracefulTime

	conf.IndexNode.Port = mc.Spec.IndexNode.Port
	SetGRPCConfig(&conf.IndexNode.GRPC, mc.Spec.IndexNode.GRPC)

	conf.Proxy.Port = mc.Spec.Proxy.Port
	SetGRPCConfig(&conf.Proxy.GRPC, mc.Spec.Proxy.GRPC)

	return conf
}

func (r *MilvusClusterReconciler) updateConfigMap(mc v1alpha1.MilvusCluster, configmap *corev1.ConfigMap) error {
	configuration := GetMilvusConfigFrom(mc)

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

		r.logger.Info("Create Configmap", "name", new.Name, "namespace", new.Namespace)
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

	r.logger.Info("Update Configmap", "name", cur.Name, "namespace", cur.Namespace)
	return r.Update(ctx, cur)
}
