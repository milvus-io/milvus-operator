package controllers

import (
	"context"
	"github.com/milvus-io/milvus-operator/apis/milvus.io/v1alpha1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

var ingressClass = "nginx"
var pathTypePrefix = networkingv1.PathTypePrefix
var host = "milvus.proxy.com"


func (r *MilvusReconciler) ReconcileIngress(ctx context.Context, mil v1alpha1.Milvus) error {

	namespacedName := NamespacedName(mil.Namespace, mil.Name)
	old := &networkingv1.Ingress{}
	err := r.Get(ctx, namespacedName, old)
	if errors.IsNotFound(err) {
		new := &networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      namespacedName.Name,
				Namespace: namespacedName.Namespace,
				Annotations: map[string]string{
					"nginx.ingress.kubernetes.io/backend-protocol": "GRPC",
				},
			},
		}
		if err := r.updateIngress(mil, new,namespacedName.Name); err != nil {
			return err
		}

		r.logger.Info("Create Ingress", "name", new.Name, "namespace", new.Namespace)
		return r.Create(ctx, new)
	} else if err != nil {
		return err
	}

	cur := old.DeepCopy()
	if err := r.updateIngress(mil, cur,namespacedName.Name); err != nil {
		return err
	}

	if IsEqual(old, cur) {
		return nil
	}

	r.logger.Info("Update Ingress", "name", cur.Name, "namespace", cur.Namespace)
	return r.Update(ctx, cur)
}

func (r *MilvusReconciler) updateIngress(
	mc v1alpha1.Milvus, ingress *networkingv1.Ingress, name string,
) error {

	if err := ctrl.SetControllerReference(&mc, ingress, r.Scheme); err != nil {
		return err
	}

	ingress.Spec.IngressClassName = &ingressClass
	ingress.Spec.TLS =  []networkingv1.IngressTLS{
		{
			SecretName: "milvus-secret",
			Hosts: []string{host},
		},

	}
	ingress.Spec.Rules = []networkingv1.IngressRule{
		{
			Host: host,
			IngressRuleValue: networkingv1.IngressRuleValue{
				HTTP: &networkingv1.HTTPIngressRuleValue{
					Paths: []networkingv1.HTTPIngressPath{
						{
							Path:     "/",
							PathType: &pathTypePrefix,
							Backend: networkingv1.IngressBackend{
								Service: &networkingv1.IngressServiceBackend{
									Name: name,
									Port: networkingv1.ServiceBackendPort{
										Number: 19530,
									},
								},
							},
						},
					},
				},
			},
		},
	}


	return nil
}
