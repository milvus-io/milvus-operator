package controllers

import (
	"context"
	"fmt"

	"github.com/milvus-io/milvus-operator/apis/milvus.io/v1alpha1"
	"github.com/milvus-io/milvus-operator/pkg/helm"
	"github.com/milvus-io/milvus-operator/pkg/util"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func IsSetDefaultDone(mc *v1alpha1.Milvus) bool {
	return mc.Status.Status != ""
}

func (r *MilvusReconciler) Finalize(ctx context.Context, mil v1alpha1.Milvus) error {
	deletingReleases := map[string]bool{}

	if !mil.Spec.Dep.Etcd.External && mil.Spec.Dep.Etcd.InCluster.DeletionPolicy == v1alpha1.DeletionPolicyDelete {
		deletingReleases[mil.Name+"-etcd"] = mil.Spec.Dep.Etcd.InCluster.PVCDeletion
	}
	if !mil.Spec.Dep.Storage.External && mil.Spec.Dep.Storage.InCluster.DeletionPolicy == v1alpha1.DeletionPolicyDelete {
		deletingReleases[mil.Name+"-minio"] = mil.Spec.Dep.Storage.InCluster.PVCDeletion
	}

	if len(deletingReleases) > 0 {
		cfg := r.helmReconciler.NewHelmCfg(mil.Namespace)

		errs := []error{}
		for releaseName, deletePVC := range deletingReleases {
			if err := helm.Uninstall(cfg, releaseName); err != nil {
				errs = append(errs, err)
				continue
			}

			if deletePVC {
				pvcList := &corev1.PersistentVolumeClaimList{}
				if err := r.List(ctx, pvcList, &client.ListOptions{
					Namespace: mil.Namespace,
					LabelSelector: labels.SelectorFromSet(map[string]string{
						AppLabelInstance: releaseName,
					}),
				}); err != nil {
					errs = append(errs, err)
					continue
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

func (r *MilvusReconciler) SetDefault(ctx context.Context, mc *v1alpha1.Milvus) error {
	if !mc.Spec.Dep.Etcd.External && len(mc.Spec.Dep.Etcd.Endpoints) == 0 {
		mc.Spec.Dep.Etcd.Endpoints = []string{fmt.Sprintf("%s-etcd.%s:2379", mc.Name, mc.Namespace)}
	}
	if !mc.Spec.Dep.Storage.External && len(mc.Spec.Dep.Storage.Endpoint) == 0 {
		mc.Spec.Dep.Storage.Endpoint = fmt.Sprintf("%s-minio.%s:9000", mc.Name, mc.Namespace)
	}
	return nil
}

// SetDefaultStatus update status if default not set; return true if updated, return false if not, return err if update failed
func (r *MilvusReconciler) SetDefaultStatus(ctx context.Context, mc *v1alpha1.Milvus) (bool, error) {
	if mc.Status.Status == "" {
		mc.Status.Status = v1alpha1.StatusCreating
		err := r.Client.Status().Update(ctx, mc)
		if err != nil {
			return false, errors.Wrapf(err, "set mc default status[%s/%s] failed", mc.Namespace, mc.Name)
		}
		return true, nil
	}
	return false, nil
}

func (r *MilvusReconciler) ReconcileAll(ctx context.Context, mil v1alpha1.Milvus) error {
	milvusReconcilers := []Func{
		r.ReconcileEtcd,
		r.ReconcileMinio,
		r.ReconcileMilvus,
	}
	err := defaultGroupRunner.Run(milvusReconcilers, ctx, mil)
	return errors.Wrap(err, "reconcile milvus")
}

func (r *MilvusReconciler) ReconcileMilvus(ctx context.Context, mil v1alpha1.Milvus) error {
	if !IsDependencyReady(mil.Status.Conditions, false) {
		return nil
	}

	if err := r.ReconcileConfigMaps(ctx, mil); err != nil {
		return errors.Wrap(err, "configmap")
	}
	milvusComsReconcilers := []Func{
		r.ReconcileDeployments,
		r.ReconcileServices,
		r.ReconcilePodMonitor,
	}
	err := defaultGroupRunner.Run(milvusComsReconcilers, ctx, mil)
	return errors.Wrap(err, "reconcile components")
}
