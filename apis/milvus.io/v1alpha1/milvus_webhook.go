/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	v1beta1 "github.com/milvus-io/milvus-operator/apis/milvus.io/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

func (r *Milvus) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// ConvertTo converts this CronJob to the Hub version (v1).
func (r *Milvus) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1beta1.Milvus)
	dst.Spec.Mode = v1beta1.MilvusModeStandalone
	dst.Spec.Conf = r.Spec.Conf
	dst.Spec.Dep = r.Spec.Dep
	dst.Spec.Com.DisableMetric = r.Spec.DisableMetric

	dst.Spec.Com.Standalone = v1beta1.MilvusStandalone{
		Component: v1beta1.Component{
			ComponentSpec: r.Spec.ComponentSpec,
			Replicas:      r.Spec.Replicas,
			Port:          19530,
		},
		ServiceType:        r.Spec.ServiceType,
		ServiceLabels:      r.Spec.ServiceLabels,
		ServiceAnnotations: r.Spec.ServiceAnnotations,
		Ingress:            r.Spec.Ingress,
	}
	return nil
}

// ConvertFrom converts from the Hub version to this version.
func (r *Milvus) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1beta1.Milvus)
	r.Spec.Conf = src.Spec.Conf
	r.Spec.Dep = src.Spec.Dep
	r.Spec.DisableMetric = src.Spec.Com.DisableMetric
	r.Spec.ComponentSpec = src.Spec.Com.Standalone.ComponentSpec

	r.Spec.Replicas = src.Spec.Com.Standalone.Replicas
	r.Spec.ServiceType = src.Spec.Com.Standalone.ServiceType
	r.Spec.ServiceLabels = src.Spec.Com.Standalone.ServiceLabels
	r.Spec.ServiceAnnotations = src.Spec.Com.Standalone.ServiceAnnotations
	r.Spec.Ingress = src.Spec.Com.Standalone.Ingress

	return nil
}
