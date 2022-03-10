package controllers

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/milvus-io/milvus-operator/apis/milvus.io/v1alpha1"
	"github.com/pkg/errors"
)

func getPVCNameByInstName(instName string) string {
	return instName + "-data"
}

func (r *MilvusReconciler) ReconcilePVCs(ctx context.Context, mil v1alpha1.Milvus) error {
	needPVC := mil.Spec.Persistence.Enabled && len(mil.Spec.Persistence.PersistentVolumeClaim.ExistingClaim) < 1
	if !needPVC {
		return nil
	}
	// if needPVC
	namespacedName := NamespacedName(mil.Namespace, getPVCNameByInstName(mil.Name))
	return r.syncUpdatePVC(ctx, namespacedName, mil.Spec.Persistence.PersistentVolumeClaim)
}

func (r *MilvusReconciler) syncUpdatePVC(ctx context.Context, namespacedName types.NamespacedName, milvusPVC v1alpha1.PersistentVolumeClaim) error {
	old := &corev1.PersistentVolumeClaim{}
	err := r.Client.Get(ctx, namespacedName, old)

	if kerrors.IsNotFound(err) {
		new := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      namespacedName.Name,
				Namespace: namespacedName.Namespace,
			},
		}
		r.syncPVC(ctx, milvusPVC, new)
		r.logger.Info("Create PersistentVolumeClaim", "name", new.Name, "namespace", new.Namespace)
		return r.Create(ctx, new)
	}
	if err != nil {
		return errors.Wrap(err, "failed to get data pvc")
	}
	new := old.DeepCopy()

	r.syncPVC(ctx, milvusPVC, new)
	// volume name set by pvc controller
	new.Spec.VolumeName = old.Spec.VolumeName
	if milvusPVC.Spec.StorageClassName == nil {
		// if nil, default storage class name set by pvc controller
		new.Spec.StorageClassName = old.Spec.StorageClassName
	}

	if IsEqual(old, new) {
		return nil
	}
	r.logger.Info("Update PVC", "name", new.Name, "namespace", new.Namespace)
	err = r.Client.Update(ctx, new)
	return errors.Wrap(err, "failed to update data pvc")
}

func (r *MilvusReconciler) syncPVC(ctx context.Context, milvusPVC v1alpha1.PersistentVolumeClaim, pvc *corev1.PersistentVolumeClaim) {
	pvc.Labels = milvusPVC.Labels
	pvc.Annotations = milvusPVC.Annotations
	pvc.Spec = milvusPVC.Spec
}
