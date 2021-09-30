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
	"fmt"

	"github.com/milvus-io/milvus-operator/pkg/config"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var milvusclusterlog = logf.Log.WithName("milvuscluster-resource")

func (r *MilvusCluster) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-milvus-io-v1alpha1-milvuscluster,mutating=true,failurePolicy=fail,sideEffects=None,groups=milvus.io,resources=milvusclusters,verbs=create;update,versions=v1alpha1,name=mmilvuscluster.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &MilvusCluster{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *MilvusCluster) Default() {
	milvusclusterlog.Info("default", "name", r.Name)

	if r.Spec.Com.Image == "" {
		r.Spec.Com.Image = config.DefaultMilvusImage
	}

	replicas := int32(1)
	if r.Spec.Com.Proxy.Replicas == nil {
		r.Spec.Com.Proxy.Replicas = &replicas
	}
	if r.Spec.Com.RootCoord.Replicas == nil {
		r.Spec.Com.RootCoord.Replicas = &replicas
	}
	if r.Spec.Com.DataCoord.Replicas == nil {
		r.Spec.Com.DataCoord.Replicas = &replicas
	}
	if r.Spec.Com.IndexCoord.Replicas == nil {
		r.Spec.Com.IndexCoord.Replicas = &replicas
	}
	if r.Spec.Com.QueryCoord.Replicas == nil {
		r.Spec.Com.QueryCoord.Replicas = &replicas
	}
	if r.Spec.Com.DataNode.Replicas == nil {
		r.Spec.Com.DataNode.Replicas = &replicas
	}
	if r.Spec.Com.IndexNode.Replicas == nil {
		r.Spec.Com.IndexNode.Replicas = &replicas
	}
	if r.Spec.Com.QueryNode.Replicas == nil {
		r.Spec.Com.QueryNode.Replicas = &replicas
	}

	// set in cluster etcd endpoints
	if !r.Spec.Dep.Etcd.External && r.Spec.Dep.Etcd.InCluster == nil {
		r.Spec.Dep.Etcd.InCluster = &InClusterEtcd{
			Values: Values{Data: map[string]interface{}{}},
		}
	}

	// set in cluster pulsar endpoint
	if !r.Spec.Dep.Pulsar.External && r.Spec.Dep.Pulsar.InCluster == nil {
		r.Spec.Dep.Pulsar.InCluster = &InClusterPulsar{
			Values: Values{Data: map[string]interface{}{}},
		}
	}

	// set in cluster storage
	if !r.Spec.Dep.Storage.External && r.Spec.Dep.Storage.InCluster == nil {
		r.Spec.Dep.Storage.InCluster = &InClusterStorage{
			Values: Values{Data: map[string]interface{}{}},
		}
		r.Spec.Dep.Storage.SecretRef = r.Name + "-minio"
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-milvus-io-v1alpha1-milvuscluster,mutating=false,failurePolicy=fail,sideEffects=None,groups=milvus.io,resources=milvusclusters,verbs=create;update,versions=v1alpha1,name=vmilvuscluster.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &MilvusCluster{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *MilvusCluster) ValidateCreate() error {
	milvusclusterlog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	var allErrs field.ErrorList

	if errs := r.validateExternal(); len(errs) > 0 {
		allErrs = append(allErrs, errs...)
	}

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(schema.GroupKind{Group: GroupVersion.Group, Kind: "MilvusCluster"}, r.Name, allErrs)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *MilvusCluster) ValidateUpdate(old runtime.Object) error {
	milvusclusterlog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	_, ok := old.(*MilvusCluster)
	if !ok {
		return errors.Errorf("failed type assertion on kind: %s", old.GetObjectKind().GroupVersionKind().String())
	}

	var allErrs field.ErrorList
	if errs := r.validateExternal(); len(errs) > 0 {
		allErrs = append(allErrs, errs...)
	}

	if len(allErrs) == 0 {
		return nil
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *MilvusCluster) ValidateDelete() error {
	milvusclusterlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}

func (r *MilvusCluster) validateExternal() field.ErrorList {
	var allErrs field.ErrorList
	fp := field.NewPath("spec").Child("dependencies")

	if r.Spec.Dep.Etcd.External && len(r.Spec.Dep.Etcd.Endpoints) == 0 {
		allErrs = append(allErrs, required(fp.Child("etcd").Child("endpoints")))
	}

	if r.Spec.Dep.Storage.External && len(r.Spec.Dep.Storage.Endpoint) == 0 {
		allErrs = append(allErrs, required(fp.Child("storage").Child("endpoint")))
	}

	if r.Spec.Dep.Pulsar.External && len(r.Spec.Dep.Pulsar.Endpoint) == 0 {
		allErrs = append(allErrs, required(fp.Child("pulsar").Child("endpoint")))
	}

	return allErrs
}

func required(mainPath *field.Path) *field.Error {
	return field.Required(mainPath, fmt.Sprintf("%s should be configured", mainPath.String()))
}

func invalid(mainPath *field.Path, value interface{}, details string) *field.Error {
	return field.Invalid(mainPath, value, details)
}

func forbidden(mainPath fmt.Stringer, conflictPath *field.Path) *field.Error {
	return field.Forbidden(conflictPath, fmt.Sprintf("conflicts: %s should not be configured as %s has been configured already", conflictPath.String(), mainPath.String()))
}
