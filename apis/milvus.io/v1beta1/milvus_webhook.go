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

package v1beta1

import (
	"fmt"

	"github.com/milvus-io/milvus-operator/pkg/config"
	"github.com/milvus-io/milvus-operator/pkg/util"
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

//+kubebuilder:webhook:path=/mutate-milvus-io-v1beta1-milvus,mutating=true,failurePolicy=fail,sideEffects=None,groups=milvus.io,resources=milvuses,verbs=create;update,versions=v1beta1,name=mmilvus.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Milvus{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Milvus) Default() {
	milvuslog.Info("default", "name", r.Name)

	if r.Namespace == "" {
		r.Namespace = "default"
	}

	if r.Spec.Mode == "" {
		r.Spec.Mode = MilvusModeStandalone
	}

	if r.Spec.Dep.Storage.Type == "" {
		r.Spec.Dep.Storage.Type = "MinIO"
	}

	if r.Spec.Conf.Data == nil {
		r.Spec.Conf.Data = map[string]interface{}{}
	} else {
		deleteUnsettableConf(r.Spec.Conf.Data)
	}

	if r.Spec.Com.Image == "" {
		r.Spec.Com.Image = config.DefaultMilvusImage
	}

	// default components
	if r.Spec.Mode == MilvusModeCluster {
		if r.Spec.Com.Proxy == nil {
			r.Spec.Com.Proxy = &MilvusProxy{}
		}
		if r.Spec.Com.MixCoord == nil {
			r.Spec.Com.RootCoord = &MilvusRootCoord{}
			r.Spec.Com.IndexCoord = &MilvusIndexCoord{}
			r.Spec.Com.DataCoord = &MilvusDataCoord{}
			r.Spec.Com.QueryCoord = &MilvusQueryCoord{}
		}
		if r.Spec.Com.DataNode == nil {
			r.Spec.Com.DataNode = &MilvusDataNode{}
		}
		if r.Spec.Com.IndexNode == nil {
			r.Spec.Com.IndexNode = &MilvusIndexNode{}
		}
		if r.Spec.Com.QueryNode == nil {
			r.Spec.Com.QueryNode = &MilvusQueryNode{}
		}
	} else {
		// standalone
		if r.Spec.Com.Standalone == nil {
			r.Spec.Com.Standalone = &MilvusStandalone{}
		}
	}

	// defaultReplicas
	defaultReplicas := int32(1)
	if r.Spec.Mode == MilvusModeCluster {

		if r.Spec.Com.MixCoord != nil {
			if r.Spec.Com.MixCoord.Replicas == nil {
				r.Spec.Com.MixCoord.Replicas = &defaultReplicas
			}
		} else {
			if r.Spec.Com.Proxy.Replicas == nil {
				r.Spec.Com.Proxy.Replicas = &defaultReplicas
			}
			if r.Spec.Com.RootCoord.Replicas == nil {
				r.Spec.Com.RootCoord.Replicas = &defaultReplicas
			}
			if r.Spec.Com.DataCoord.Replicas == nil {
				r.Spec.Com.DataCoord.Replicas = &defaultReplicas
			}
			if r.Spec.Com.IndexCoord.Replicas == nil {
				r.Spec.Com.IndexCoord.Replicas = &defaultReplicas
			}
			if r.Spec.Com.QueryCoord.Replicas == nil {
				r.Spec.Com.QueryCoord.Replicas = &defaultReplicas
			}
			if r.Spec.Com.DataNode.Replicas == nil {
				r.Spec.Com.DataNode.Replicas = &defaultReplicas
			}
			if r.Spec.Com.IndexNode.Replicas == nil {
				r.Spec.Com.IndexNode.Replicas = &defaultReplicas
			}
			if r.Spec.Com.QueryNode.Replicas == nil {
				r.Spec.Com.QueryNode.Replicas = &defaultReplicas
			}
		}
	} else {
		if r.Spec.Com.Standalone.Replicas == nil {
			r.Spec.Com.Standalone.Replicas = &defaultReplicas
		}
	}

	// etcd
	if !r.Spec.Dep.Etcd.External {
		r.Spec.Dep.Etcd.Endpoints = []string{fmt.Sprintf("%s-etcd.%s:2379", r.Name, r.Namespace)}
		if r.Spec.Dep.Etcd.InCluster == nil {
			r.Spec.Dep.Etcd.InCluster = &InClusterConfig{}
		}
		if r.Spec.Dep.Etcd.InCluster.Values.Data == nil {
			r.Spec.Dep.Etcd.InCluster.Values.Data = map[string]interface{}{}
		}
		if r.Spec.Dep.Etcd.InCluster.DeletionPolicy == "" {
			r.Spec.Dep.Etcd.InCluster.DeletionPolicy = DeletionPolicyRetain
		}
		if r.Spec.Dep.Etcd.Endpoints == nil {
			r.Spec.Dep.Etcd.Endpoints = []string{}
		}
	}

	// mq
	if r.Spec.Dep.MsgStreamType == "" {
		switch r.Spec.Mode {
		case MilvusModeStandalone:
			r.Spec.Dep.MsgStreamType = MsgStreamTypeRocksMQ
		case MilvusModeCluster:
			r.Spec.Dep.MsgStreamType = MsgStreamTypePulsar
		}
	}
	switch r.Spec.Dep.MsgStreamType {
	case MsgStreamTypeKafka:
		if !r.Spec.Dep.Kafka.External {
			r.Spec.Dep.Kafka.BrokerList = []string{fmt.Sprintf("%s-kafka.%s:9092", r.Name, r.Namespace)}
			if r.Spec.Dep.Kafka.InCluster == nil {
				r.Spec.Dep.Kafka.InCluster = &InClusterConfig{}
			}
			if r.Spec.Dep.Kafka.InCluster.Values.Data == nil {
				r.Spec.Dep.Kafka.InCluster.Values.Data = map[string]interface{}{}
			}
			if r.Spec.Dep.Kafka.InCluster.DeletionPolicy == "" {
				r.Spec.Dep.Kafka.InCluster.DeletionPolicy = DeletionPolicyRetain
			}
		}
	case MsgStreamTypePulsar:
		if !r.Spec.Dep.Pulsar.External {
			r.Spec.Dep.Pulsar.Endpoint = fmt.Sprintf("%s-pulsar-proxy.%s:6650", r.Name, r.Namespace)
			if r.Spec.Dep.Pulsar.InCluster == nil {
				r.Spec.Dep.Pulsar.InCluster = &InClusterConfig{}
			}
			if r.Spec.Dep.Pulsar.InCluster.Values.Data == nil {
				r.Spec.Dep.Pulsar.InCluster.Values.Data = map[string]interface{}{}
			}
			if r.Spec.Dep.Pulsar.InCluster.DeletionPolicy == "" {
				r.Spec.Dep.Pulsar.InCluster.DeletionPolicy = DeletionPolicyRetain
			}
		}
	case MsgStreamTypeRocksMQ:
		// do nothing
	}

	// storage
	if !r.Spec.Dep.Storage.External {
		r.Spec.Dep.Storage.Endpoint = fmt.Sprintf("%s-minio.%s:9000", r.Name, r.Namespace)
		if r.Spec.Dep.Storage.InCluster == nil {
			r.Spec.Dep.Storage.InCluster = &InClusterConfig{}
		}
		if r.Spec.Dep.Storage.InCluster.Values.Data == nil {
			r.Spec.Dep.Storage.InCluster.Values.Data = map[string]interface{}{}
		}
		if r.Spec.Dep.Storage.InCluster.DeletionPolicy == "" {
			r.Spec.Dep.Storage.InCluster.DeletionPolicy = DeletionPolicyRetain
		}
		r.Spec.Dep.Storage.SecretRef = r.Name + "-minio"
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-milvus-io-v1beta1-milvus,mutating=false,failurePolicy=fail,sideEffects=None,groups=milvus.io,resources=milvuses,verbs=create;update,versions=v1beta1,name=vmilvus.kb.io,admissionReviewVersions=v1

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

func (r *Milvus) validateExternal() field.ErrorList {
	var allErrs field.ErrorList
	fp := field.NewPath("spec").Child("dependencies")

	if r.Spec.Dep.Etcd.External && len(r.Spec.Dep.Etcd.Endpoints) == 0 {
		allErrs = append(allErrs, required(fp.Child("etcd").Child("endpoints")))
	}

	if r.Spec.Dep.Storage.External && len(r.Spec.Dep.Storage.Endpoint) == 0 {
		allErrs = append(allErrs, required(fp.Child("storage").Child("endpoint")))
	}

	switch r.Spec.Dep.MsgStreamType {
	case MsgStreamTypeKafka:
		if r.Spec.Dep.Kafka.External && len(r.Spec.Dep.Kafka.BrokerList) == 0 {
			allErrs = append(allErrs, required(fp.Child("kafka").Child("brokerList")))
		}
	case MsgStreamTypePulsar:
		if r.Spec.Dep.Pulsar.External && len(r.Spec.Dep.Pulsar.Endpoint) == 0 {
			allErrs = append(allErrs, required(fp.Child("pulsar").Child("endpoint")))
		}
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

func deleteUnsettableConf(conf map[string]interface{}) {
	util.DeleteValue(conf, "minio", "address")
	util.DeleteValue(conf, "minio", "port")
	util.DeleteValue(conf, "pulsar", "address")
	util.DeleteValue(conf, "pulsar", "port")
	util.DeleteValue(conf, "etcd", "endpoints")

	for _, t := range MilvusComponentTypes {
		util.DeleteValue(conf, t.String(), "port")
	}
	for _, t := range MilvusCoordTypes {
		util.DeleteValue(conf, t.String(), "address")
	}
}
