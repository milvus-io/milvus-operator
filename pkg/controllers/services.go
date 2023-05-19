package controllers

import (
	"context"

	pkgerr "github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/milvus-io/milvus-operator/apis/milvus.io/v1beta1"
)

func (r *MilvusReconciler) updateService(
	mc v1beta1.Milvus, service *corev1.Service, component MilvusComponent,
) error {
	serviceLabels := NewAppLabels(mc.Name)
	service.Labels = MergeLabels(service.Labels, serviceLabels)

	// backward compatibility, remove redundant label
	delete(service.Labels, AppLabelComponent)

	if err := SetControllerReference(&mc, service, r.Scheme); err != nil {
		return err
	}
	service.Spec.Ports = MergeServicePort(service.Spec.Ports, component.GetServicePorts(mc.Spec))
	service.Spec.Selector = NewServicePodLabels(mc.Name)
	service.Spec.Type = component.GetServiceType(mc.Spec)

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
	if !component.IsService() {
		return nil
	}

	isChangingMode := mc.IsChangingMode()
	isPodServiceLabelAdded := mc.IsPodServiceLabelAdded()
	if !isPodServiceLabelAdded && !isChangingMode {
		return nil
	}

	if isChangingMode && !isPodServiceLabelAdded {
		err := r.labelServicePods(ctx, mc, component)
		if err != nil {
			return pkgerr.Wrap(err, "label service pods")
		}
	}

	namespacedName := NamespacedName(mc.Namespace, GetServiceInstanceName(mc.Name))
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
	if mc.Spec.Mode == v1beta1.MilvusModeCluster {
		err = r.ReconcileComponentService(ctx, mc, Proxy)
	} else {
		err = r.ReconcileComponentService(ctx, mc, MilvusStandalone)
	}

	return pkgerr.Wrap(err, "reconcile milvus services")
}

func (r *MilvusReconciler) labelServicePods(ctx context.Context, mc v1beta1.Milvus, component MilvusComponent) error {
	pods := &corev1.PodList{}
	opts := &client.ListOptions{
		Namespace: mc.Namespace,
	}
	serviceComponents := []MilvusComponent{MilvusStandalone, Proxy}

	for _, serviceComponent := range serviceComponents {
		opts.LabelSelector = labels.SelectorFromSet(NewComponentAppLabels(
			mc.Name,
			serviceComponent.Name,
		))
		if err := r.List(ctx, pods, opts); err != nil {
			return pkgerr.Wrapf(err, "list [%s] pods", serviceComponent.Name)
		}
		for _, pod := range pods.Items {
			if pod.Labels == nil {
				pod.Labels = map[string]string{}
			}
			if pod.Labels[v1beta1.ServiceLabel] != v1beta1.TrueStr {
				pod.Labels[v1beta1.ServiceLabel] = v1beta1.TrueStr
				if err := r.Update(ctx, &pod); err != nil {
					return pkgerr.Wrapf(err, "label pod %s", pod.Name)
				}
			}
		}
	}

	return nil
}
