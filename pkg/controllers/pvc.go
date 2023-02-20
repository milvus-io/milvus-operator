package controllers

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/milvus-io/milvus-operator/apis/milvus.io/v1beta1"
	"github.com/pkg/errors"
)

func getPVCNameByInstName(instName string) string {
	return instName + "-data"
}

func (r *MilvusReconciler) ReconcilePVCs(ctx context.Context, mil v1beta1.Milvus) error {
	persistence := mil.Spec.Dep.RocksMQ.Persistence
	needPVC := persistence.Enabled && len(persistence.PersistentVolumeClaim.ExistingClaim) < 1
	if !needPVC {
		return nil
	}
	// if needPVC
	namespacedName := NamespacedName(mil.Namespace, getPVCNameByInstName(mil.Name))
	return r.syncUpdatePVC(ctx, namespacedName, mil)
}

func (r *MilvusReconciler) syncUpdatePVC(ctx context.Context, namespacedName types.NamespacedName, m v1beta1.Milvus) error {
	pvc := &corev1.PersistentVolumeClaim{}
	pvc.Name = getPVCNameByInstName(m.Name)
	pvc.Namespace = m.Namespace

	_, err := ctrl.CreateOrUpdate(ctx, r.Client, pvc, func() error {
		r.syncPVC(ctx, m.Spec.Dep.RocksMQ.Persistence.PersistentVolumeClaim, pvc)
		if m.Spec.Dep.RocksMQ.Persistence.PVCDeletion {
			return ctrl.SetControllerReference(&m, pvc, r.Scheme)
		}
		// else
		pvc.OwnerReferences = nil
		return nil
	})
	return errors.Wrap(err, "failed to update data pvc")
}

func (r *MilvusReconciler) syncPVC(ctx context.Context, milvusPVC v1beta1.PersistentVolumeClaim, pvc *corev1.PersistentVolumeClaim) {
	pvc.Labels = milvusPVC.Labels
	pvc.Annotations = milvusPVC.Annotations

	currentSc := pvc.Spec.StorageClassName
	currentVn := pvc.Spec.VolumeName
	pvc.Spec = milvusPVC.GetSpec()
	if len(pvc.Spec.AccessModes) < 1 {
		pvc.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{
			corev1.ReadWriteOnce,
		}
	}
	if pvc.Spec.Resources.Requests == nil {
		pvc.Spec.Resources.Requests = corev1.ResourceList{}
	}
	if pvc.Spec.Resources.Requests.Storage().IsZero() {
		q := resource.Quantity{}
		q.Set(defaultPVCSize)
		pvc.Spec.Resources.Requests[corev1.ResourceStorage] = q
	}

	if currentVn != "" {
		// emtpy volume name will be set by pvc controller
		pvc.Spec.VolumeName = currentVn
	}
	if milvusPVC.GetSpec().StorageClassName == nil {
		// nil storage class name will be set by pvc controller
		pvc.Spec.StorageClassName = currentSc
	}

}

// 5Gi
var defaultPVCSize int64 = 5 * 1024 * 1024 * 1024
