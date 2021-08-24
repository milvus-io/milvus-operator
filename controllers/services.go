package controllers

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/milvus-io/milvus-operator/api/v1alpha1"
)

func (r *MilvusClusterReconciler) updateService(
	mc v1alpha1.MilvusCluster, service *corev1.Service, component MilvusComponent,
) error {
	appLabels := NewAppLabels(mc.Name, component)
	service.Labels = MergeLabels(service.Labels, appLabels)
	if err := ctrl.SetControllerReference(&mc, service, r.Scheme); err != nil {
		return err
	}

	service.Spec.Ports = MergeServicePort(service.Spec.Ports, component.GetServicePorts(mc.Spec))
	service.Spec.Selector = appLabels
	service.Spec.Type = component.GetServiceType(mc.Spec)

	return nil
}

func (r *MilvusClusterReconciler) ReconcileComponentService(
	ctx context.Context, mc *v1alpha1.MilvusCluster, component MilvusComponent,
) error {
	namespacedName := types.NamespacedName{
		Namespace: mc.Namespace,
		Name:      component.GetServiceInstanceName(mc.Name),
	}
	old := &corev1.Service{}
	err := r.Get(ctx, namespacedName, old)
	if errors.IsNotFound(err) {
		new := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      namespacedName.Name,
				Namespace: namespacedName.Namespace,
			},
		}
		if err := r.updateService(*mc, new, component); err != nil {
			return err
		}

		r.logger.Info("Create Service", "name", new.Name, "namespace", new.Namespace)
		return r.Create(ctx, new)
	} else if err != nil {
		return err
	}

	cur := old.DeepCopy()
	if err := r.updateService(*mc, cur, component); err != nil {
		return err
	}

	if IsEqual(old, cur) {
		//r.logger.Info("Equal", "cur", cur.Name)
		return nil
	}

	r.logger.Info("Update Service", "name", cur.Name, "namespace", cur.Namespace)
	return r.Update(ctx, cur)
}

func (r *MilvusClusterReconciler) ReconcileServices(ctx context.Context, mc *v1alpha1.MilvusCluster) error {
	g, gtx := NewGroup(ctx)
	for _, component := range MilvusComponents {
		g.Go(func(ctx context.Context, mc *v1alpha1.MilvusCluster, component MilvusComponent) func() error {
			return func() error {
				return r.ReconcileComponentService(ctx, mc, component)
			}
		}(gtx, mc, component))
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("reconcile milvus services: %w", err)
	}

	return nil
}
