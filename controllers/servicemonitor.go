package controllers

import (
	"context"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/milvus-io/milvus-operator/api/v1alpha1"
)

func (r *MilvusClusterReconciler) updateServiceMonitor(
	mc v1alpha1.MilvusCluster, servicemonitor *monitoringv1.ServiceMonitor) error {

	appLabels := NewAppLabels(mc.Name)
	servicemonitor.Labels = MergeLabels(servicemonitor.Labels, appLabels)
	if err := ctrl.SetControllerReference(&mc, servicemonitor, r.Scheme); err != nil {
		r.logger.Error(err, "servicemonitor SetControllerReference error")
		return err
	}

	servicemonitor.Spec.Endpoints = []monitoringv1.Endpoint{
		{
			HonorLabels: true,
			Interval:    "60s",
			Path:        MetricPath,
			Port:        MetricPortName,
		},
	}
	servicemonitor.Spec.NamespaceSelector = monitoringv1.NamespaceSelector{
		MatchNames: []string{mc.Namespace},
	}
	servicemonitor.Spec.Selector.MatchLabels = appLabels
	servicemonitor.Spec.TargetLabels = []string{
		AppLabelInstance, AppLabelName, AppLabelComponent,
	}

	return nil
}

func (r *MilvusClusterReconciler) ReconcileServiceMonitor(ctx context.Context, mc v1alpha1.MilvusCluster) error {
	namespacedName := NamespacedName(mc.Namespace, mc.Name)
	old := &monitoringv1.ServiceMonitor{}
	err := r.Get(ctx, namespacedName, old)
	if meta.IsNoMatchError(err) {
		r.logger.Info("servicemonitor kind no matchs, maybe is not installed")
		return nil
	}

	if errors.IsNotFound(err) {
		new := &monitoringv1.ServiceMonitor{
			ObjectMeta: metav1.ObjectMeta{
				Name:      namespacedName.Name,
				Namespace: namespacedName.Namespace,
			},
		}

		if err := r.updateServiceMonitor(mc, new); err != nil {
			return err
		}

		r.logger.Info("Create ServiceMonitor", "name", new.Name, "namespace", new.Namespace)
		return r.Create(ctx, new)
	}

	if err != nil {
		return err
	}

	cur := old.DeepCopy()
	if err := r.updateServiceMonitor(mc, cur); err != nil {
		return err
	}

	if IsEqual(old, cur) {
		//r.logger.Info("Equal", "cur", cur.Name)
		return nil
	}

	r.logger.Info("Update ServiceMonitor", "name", cur.Name, "namespace", cur.Namespace)
	return r.Update(ctx, cur)
}
