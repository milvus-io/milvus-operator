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

func (r *MilvusClusterReconciler) updatePodMonitor(
	mc v1alpha1.MilvusCluster, podmonitor *monitoringv1.PodMonitor) error {

	appLabels := NewAppLabels(mc.Name)
	podmonitor.Labels = MergeLabels(podmonitor.Labels, appLabels)
	if err := ctrl.SetControllerReference(&mc, podmonitor, r.Scheme); err != nil {
		r.logger.Error(err, "PodMonitor SetControllerReference error", "name", mc.Name, "namespace", mc.Namespace)
		return err
	}

	podmonitor.Spec.PodMetricsEndpoints = []monitoringv1.PodMetricsEndpoint{
		{
			HonorLabels: true,
			Interval:    "60s",
			Path:        MetricPath,
			Port:        MetricPortName,
		},
	}
	podmonitor.Spec.NamespaceSelector = monitoringv1.NamespaceSelector{
		MatchNames: []string{mc.Namespace},
	}
	podmonitor.Spec.Selector.MatchLabels = appLabels
	podmonitor.Spec.PodTargetLabels = []string{
		AppLabelInstance, AppLabelName, AppLabelComponent,
	}

	return nil
}

func (r *MilvusClusterReconciler) ReconcilePodMonitor(ctx context.Context, mc v1alpha1.MilvusCluster) error {
	namespacedName := NamespacedName(mc.Namespace, mc.Name)
	old := &monitoringv1.PodMonitor{}
	err := r.Get(ctx, namespacedName, old)
	if meta.IsNoMatchError(err) {
		r.logger.Info("podmonitor kind no matchs, maybe is not installed")
		return nil
	}

	if errors.IsNotFound(err) {
		new := &monitoringv1.PodMonitor{
			ObjectMeta: metav1.ObjectMeta{
				Name:      namespacedName.Name,
				Namespace: namespacedName.Namespace,
			},
		}

		if err := r.updatePodMonitor(mc, new); err != nil {
			return err
		}

		r.logger.Info("Create PodMonitor", "name", new.Name, "namespace", new.Namespace)
		return r.Create(ctx, new)
	}

	if err != nil {
		return err
	}

	cur := old.DeepCopy()
	if err := r.updatePodMonitor(mc, cur); err != nil {
		return err
	}

	if IsEqual(old, cur) {
		//r.logger.Info("Equal", "cur", cur.Name)
		return nil
	}

	r.logger.Info("Update PodMonitor", "name", cur.Name, "namespace", cur.Namespace)
	return r.Update(ctx, cur)
}
