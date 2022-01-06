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
var milvuslog = logf.Log.WithName("milvus-resource")

func (r *Milvus) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-milvus-io-v1alpha1-milvus,mutating=true,failurePolicy=fail,sideEffects=None,groups=milvus.io,resources=milvus,verbs=create;update,versions=v1alpha1,name=mmilvus.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Milvus{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Milvus) Default() {
	if r.Labels == nil {
		r.Labels = make(map[string]string)
	}
	r.Labels[config.VersionLabel] = config.GetVersion()

	if r.Spec.Dep.Storage.Type == "" {
		r.Spec.Dep.Storage.Type = "MinIO"
	}

	if r.Spec.Conf.Data == nil {
		r.Spec.Conf.Data = map[string]interface{}{}
	} else {
		deleteUnsettableConf(r.Spec.Conf.Data)
	}

	if r.Spec.Image == "" {
		r.Spec.Image = config.DefaultMilvusImage
	}

	// set in cluster etcd endpoints
	if !r.Spec.Dep.Etcd.External {
		if r.Spec.Dep.Etcd.InCluster == nil {
			r.Spec.Dep.Etcd.InCluster = &InClusterConfig{}
		}
		if r.Spec.Dep.Etcd.InCluster.Values.Data == nil {
			r.Spec.Dep.Etcd.InCluster.Values.Data = map[string]interface{}{}
		}
		r.Spec.Dep.Etcd.InCluster.Values.Data["replicaCount"] = 1

		if r.Spec.Dep.Etcd.InCluster.DeletionPolicy == "" {
			r.Spec.Dep.Etcd.InCluster.DeletionPolicy = DeletionPolicyRetain
		}
		if r.Spec.Dep.Etcd.Endpoints == nil {
			r.Spec.Dep.Etcd.Endpoints = []string{}
		}
	}

	// set in cluster storage
	if !r.Spec.Dep.Storage.External {
		if r.Spec.Dep.Storage.InCluster == nil {
			r.Spec.Dep.Storage.InCluster = &InClusterConfig{}
		}
		if r.Spec.Dep.Storage.InCluster.Values.Data == nil {
			r.Spec.Dep.Storage.InCluster.Values.Data = map[string]interface{}{}
		}
		r.Spec.Dep.Storage.InCluster.Values.Data["mode"] = "standalone"

		if r.Spec.Dep.Storage.InCluster.DeletionPolicy == "" {
			r.Spec.Dep.Storage.InCluster.DeletionPolicy = DeletionPolicyRetain
		}
		r.Spec.Dep.Storage.SecretRef = r.Name + "-minio"
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-milvus-io-v1alpha1-milvus,mutating=false,failurePolicy=fail,sideEffects=None,groups=milvus.io,resources=milvus,verbs=create;update,versions=v1alpha1,name=vmilvus.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Milvus{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Milvus) ValidateCreate() error {
	milvuslog.Info("validate create", "name", r.Name)

	var allErrs field.ErrorList

	if errs := r.validateExternal(); len(errs) > 0 {
		allErrs = append(allErrs, errs...)
	}

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(schema.GroupKind{Group: GroupVersion.Group, Kind: "Milvus"}, r.Name, allErrs)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Milvus) ValidateUpdate(old runtime.Object) error {
	milvuslog.Info("validate update", "name", r.Name)

	_, ok := old.(*Milvus)
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

	return apierrors.NewInvalid(schema.GroupKind{Group: GroupVersion.Group, Kind: "Milvus"}, r.Name, allErrs)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Milvus) ValidateDelete() error {
	milvuslog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}

func (r *Milvus) validateInterval() field.ErrorList {
	// TODO:
	return nil
}

func (r *Milvus) validateExternal() field.ErrorList {
	var allErrs field.ErrorList
	fp := field.NewPath("spec").Child("dependencies")

	if r.Spec.Dep.Etcd.External && len(r.Spec.Dep.Etcd.Endpoints) == 0 {
		allErrs = append(allErrs, required(fp.Child("etcd").Child("endpoints")))
	}

	if r.Spec.Dep.Storage.External && len(r.Spec.Dep.Storage.Endpoint) == 0 {
		allErrs = append(allErrs, required(fp.Child("storage").Child("endpoint")))
	}

	return allErrs
}
