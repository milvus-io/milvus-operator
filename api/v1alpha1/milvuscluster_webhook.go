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
	"k8s.io/apimachinery/pkg/runtime"
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

	if r.Spec.Image == "" {
		r.Spec.Image = config.DefaultMilvusImage
	}

	if r.Spec.RootCoord == nil {
		r.Spec.RootCoord = &RootCoordinator{}
	}

	if r.Spec.DataCoord == nil {
		r.Spec.DataCoord = &DataCoordinator{}
	}

	if r.Spec.QueryCoord == nil {
		r.Spec.QueryCoord = &QueryCoordinator{}
	}

	if r.Spec.IndexCoord == nil {
		r.Spec.IndexCoord = &IndexCoordinator{}
	}

	if r.Spec.DataNode == nil {
		r.Spec.DataNode = &DataNode{}
	}

	if r.Spec.QueryNode == nil {
		r.Spec.QueryNode = &QueryNode{}
	}

	if r.Spec.IndexNode == nil {
		r.Spec.IndexNode = &IndexNode{}
	}

	if r.Spec.Proxy == nil {
		r.Spec.Proxy = &Proxy{}
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-milvus-io-v1alpha1-milvuscluster,mutating=false,failurePolicy=fail,sideEffects=None,groups=milvus.io,resources=milvusclusters,verbs=create;update;delete,versions=v1alpha1,name=vmilvuscluster.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &MilvusCluster{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *MilvusCluster) ValidateCreate() error {
	milvusclusterlog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *MilvusCluster) ValidateUpdate(old runtime.Object) error {
	milvusclusterlog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *MilvusCluster) ValidateDelete() error {
	milvusclusterlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
