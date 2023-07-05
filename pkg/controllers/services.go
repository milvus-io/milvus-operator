package controllers

import (
	"context"

	pkgerr "github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/milvus-io/milvus-operator/apis/milvus.io/v1beta1"
)

func (r *MilvusReconciler) updateService(
	mc v1beta1.Milvus, service *corev1.Service, component MilvusComponent,
) error {
	appLabels := NewComponentAppLabels(mc.Name, component.GetName())
	service.Labels = MergeLabels(service.Labels, appLabels)
	if err := SetControllerReference(&mc, service, r.Scheme); err != nil {
		return err
	}
	service.Spec.Ports = MergeServicePort(service.Spec.Ports, component.GetServicePorts(mc.Spec))
	service.Spec.Selector = appLabels
	service.Spec.Type = component.GetServiceType(mc.Spec)

	if component.IsNode() || component.IsCoord() {
		service.Spec.ClusterIP = "None"
	}

	if mc.Spec.Mode == v1beta1.MilvusModeCluster {
		service.Labels = MergeLabels(service.Labels, mc.Spec.Com.Proxy.ServiceLabels)
		service.Annotations = MergeLabels(service.Annotations, mc.Spec.Com.Proxy.ServiceAnnotations)
	} else {
		service.Labels = MergeLabels(service.Labels, mc.Spec.Com.Standalone.ServiceLabels)
		service.Annotations = MergeLabels(service.Annotations, mc.Spec.Com.Standalone.ServiceAnnotations)

	}

	return nil
}

func (r *MilvusReconciler) ReconcileComponentService(
	ctx context.Context, mc v1beta1.Milvus, component MilvusComponent,
) error {

	var namespacedName types.NamespacedName

	if component.IsNode() || component.IsCoord() {
		namespacedName = NamespacedName(mc.Namespace, component.GetHeadlessServiceInstanceName(mc.Name))
	} else {
		namespacedName = NamespacedName(mc.Namespace, GetServiceInstanceName(mc.Name))
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
		if err := r.updateService(mc, new, component); err != nil {
			return err
		}

		r.logger.Info("Create Service", "name", new.Name, "namespace", new.Namespace)
		return r.Create(ctx, new)
	} else if err != nil {
		return err
	}

	cur := old.DeepCopy()
	if err := r.updateService(mc, cur, component); err != nil {
		return err
	}

	if IsEqual(old, cur) {
		return nil
	}

	/* if config.IsDebug() {
		diff, err := diffObject(old, cur)
		if err == nil {
			r.logger.Info("Service diff", "name", cur.Name, "namespace", cur.Namespace, "diff", string(diff))
		}
	} */

	r.logger.Info("Update Service", "name", cur.Name, "namespace", cur.Namespace)
	return r.Update(ctx, cur)
}

func (r *MilvusReconciler) ReconcileServices(ctx context.Context, mc v1beta1.Milvus) error {
	var err error
	var components []MilvusComponent

	if mc.Spec.Mode == v1beta1.MilvusModeCluster {
		components = append(components, Proxy, RootCoord, DataCoord, QueryCoord, IndexCoord, DataNode, QueryNode, IndexNode)
	} else {
		components = append(components, MilvusStandalone)
	}

	for _, component := range components {
		err = r.ReconcileComponentService(ctx, mc, component)
		if err != nil {
			r.logger.Error(err, "ReconcileComponentService", "component", component.GetName())
			break
		}
	}

	return pkgerr.Wrap(err, "reconcile milvus services")
}
