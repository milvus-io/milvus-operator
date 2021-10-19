package controllers

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/milvus-io/milvus-operator/api/v1alpha1"
	"github.com/milvus-io/milvus-operator/pkg/helm"
	"github.com/milvus-io/milvus-operator/pkg/util"
	"github.com/pkg/errors"
)

func (r *MilvusClusterReconciler) SetDefault(ctx context.Context, mc *v1alpha1.MilvusCluster) error {
	if mc.Status.Status == "" {
		mc.Status.Status = v1alpha1.StatusCreating
	}

	if !mc.Spec.Dep.Etcd.External && len(mc.Spec.Dep.Etcd.Endpoints) == 0 {
		mc.Spec.Dep.Etcd.Endpoints = []string{fmt.Sprintf("%s-etcd.%s:2379", mc.Name, mc.Namespace)}
	}
	if !mc.Spec.Dep.Pulsar.External && len(mc.Spec.Dep.Pulsar.Endpoint) == 0 {
		mc.Spec.Dep.Pulsar.Endpoint = fmt.Sprintf("%s-pulsar-proxy.%s:6650", mc.Name, mc.Namespace)
	}
	if !mc.Spec.Dep.Storage.External && len(mc.Spec.Dep.Storage.Endpoint) == 0 {
		mc.Spec.Dep.Storage.Endpoint = fmt.Sprintf("%s-minio.%s:9000", mc.Name, mc.Namespace)
	}

	return nil
}

func (r *MilvusClusterReconciler) ReconcileAll(ctx context.Context, mc v1alpha1.MilvusCluster) error {
	g, gtx := NewGroup(ctx)
	g.Go(WarppedReconcileFunc(r.ReconcileEtcd, gtx, mc))
	g.Go(WarppedReconcileFunc(r.ReconcilePulsar, gtx, mc))
	g.Go(WarppedReconcileFunc(r.ReconcileMinio, gtx, mc))
	g.Go(WarppedReconcileFunc(r.ReconcileMilvus, gtx, mc))

	if err := g.Wait(); err != nil {
		return fmt.Errorf("reconcile milvus: %w", err)
	}

	return nil
}

func (r *MilvusClusterReconciler) ReconcileMilvus(ctx context.Context, mc v1alpha1.MilvusCluster) error {
	if !IsDependencyReady(mc.Status) {
		return nil
	}

	if err := r.ReconcileConfigMaps(ctx, mc); err != nil {
		return fmt.Errorf("configmap: %w", err)
	}

	g, gtx := NewGroup(ctx)
	g.Go(WarppedReconcileFunc(r.ReconcileDeployments, gtx, mc))
	g.Go(WarppedReconcileFunc(r.ReconcileServices, gtx, mc))
	g.Go(WarppedReconcileFunc(r.ReconcileServiceMonitor, gtx, mc))
	if err := g.Wait(); err != nil {
		return fmt.Errorf("components: %w", err)
	}

	return nil
}

func (r *MilvusClusterReconciler) Finalize(ctx context.Context, mc v1alpha1.MilvusCluster) error {
	deletingReleases := map[string]bool{}

	if mc.Spec.Dep.Etcd.InCluster.DeletionPolicy == v1alpha1.DeletionPolicyDelete {
		deletingReleases[mc.Name+"-etcd"] = mc.Spec.Dep.Etcd.InCluster.PersistenceKeep
	}
	if mc.Spec.Dep.Pulsar.InCluster.DeletionPolicy == v1alpha1.DeletionPolicyDelete {
		deletingReleases[mc.Name+"-pulsar"] = mc.Spec.Dep.Pulsar.InCluster.PersistenceKeep
	}
	if mc.Spec.Dep.Storage.InCluster.DeletionPolicy == v1alpha1.DeletionPolicyDelete {
		deletingReleases[mc.Name+"-minio"] = mc.Spec.Dep.Storage.InCluster.PersistenceKeep
	}

	if len(deletingReleases) > 0 {
		cfg, err := r.NewHelmCfg(mc.Namespace)
		if err != nil {
			return err
		}

		errs := []error{}
		for releaseName, keepPVC := range deletingReleases {
			if err := helm.Uninstall(cfg, releaseName); err != nil {
				errs = append(errs, err)
				continue
			}

			if !keepPVC {
				pvcList := &corev1.PersistentVolumeClaimList{}
				if err := r.List(ctx, pvcList, &client.ListOptions{
					Namespace: mc.Namespace,
					LabelSelector: labels.SelectorFromSet(map[string]string{
						AppLabelInstance: releaseName,
					}),
				}); err != nil {
					errs = append(errs, err)
				}

				for _, pvc := range pvcList.Items {
					if err := r.Delete(ctx, &pvc); err != nil {
						errs = append(errs, err)
					} else {
						r.logger.Info("pvc deleted", "name", pvc.Name, "namespace", pvc.Namespace)
					}
				}
			}
		}

		if len(errs) > 0 {
			return errors.Errorf(util.JoinErrors(errs))
		}
	}

	return nil
}
