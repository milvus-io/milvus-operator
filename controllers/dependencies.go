package controllers

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/milvus-io/milvus-operator/api/v1alpha1"
	"github.com/milvus-io/milvus-operator/pkg/config"
)

const (
	TplEtcdSS  = "etcd.yaml.tmpl"
	TplEtcdSvc = "etcd-svc.yaml.tmpl"
	TplEtcdHead
)

func (r *MilvusClusterReconciler) ReconcileEtcdStatefulset(ctx context.Context, mc *v1alpha1.MilvusCluster) error {
	new := &appsv1.StatefulSet{}
	json, err := RenderTemplate(config.GetTemplate(TplEtcdSS), mc)
	if err != nil {
		return err
	}
	if err := new.Unmarshal(json); err != nil {
		return err
	}

	old := &appsv1.StatefulSet{}
	if err := r.Get(ctx, NamespacedName(new.Namespace, new.Name), old); err != nil {
		return err
	}
	if errors.IsNotFound(err) {
		r.logger.Info("Create Etcd StatefulSet", "name", new.Name, "namespace", new.Namespace)
		return r.Create(ctx, new)
	}

	return nil
}

func (r *MilvusClusterReconciler) InitInClusterEtcd(ctx context.Context, mc *v1alpha1.MilvusCluster) error {
	if err := r.ReconcileEtcdStatefulset(ctx, mc); err != nil {
		return err
	}
	return nil
}

func (r *MilvusClusterReconciler) ReconcileDependencies(ctx context.Context, mc *v1alpha1.MilvusCluster) error {
	checker := newStatusChecker(ctx, r.Client, mc)

	g, _ := NewGroup(ctx)
	g.Go(checker.checkEtcdStatus)
	g.Go(checker.checkStorageStatus)

	if err := g.Wait(); err != nil {
		return fmt.Errorf("reconcile milvus dependencies: %w", err)
	}

	return nil
}
