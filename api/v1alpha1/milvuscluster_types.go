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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MilvusClusterSpec defines the desired state of MilvusCluster
type MilvusClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Optional
	Image string `json:"image,omitempty"`

	// +kubebuilder:validation:Optional
	ImagePullPolicy *corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// +kubebuilder:validation:Optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// +kubebuilder:validation:Optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// +kubebuilder:validation:Optional
	EnvFrom []corev1.EnvFromSource `json:"envFrom,omitempty"`

	// +kubebuilder:validation:Optional
	Etcd *MiluvsEtcd `json:"etcd,omitempty"`

	// +kubebuilder:validation:Optional
	Storage *MilvusStorage `json:"storage,omitempty"`

	// +kubebuilder:validation:Optional
	Pulsar *MilvusPulsar `json:"pulsar,omitempty"`

	// Proxy defines the desired state of Proxy
	// +kubebuilder:validation:Optional
	Proxy *Proxy `json:"proxy,omitempty"`

	// RootCoord defines the desired state of RootCoord
	// +kubebuilder:validation:Optional
	RootCoord *RootCoordinator `json:"rootCoord,omitempty"`

	// +kubebuilder:validation:Optional
	DataCoord *DataCoordinator `json:"dataCoord,omitempty"`

	// +kubebuilder:validation:Optional
	QueryCoord *QueryCoordinator `json:"queryCoord,omitempty"`

	// +kubebuilder:validation:Optional
	IndexCoord *IndexCoordinator `json:"indexCoord,omitempty"`

	// +kubebuilder:validation:Optional
	QueryNode *QueryNode `json:"queryNode,omitempty"`

	// +kubebuilder:validation:Optional
	DataNode *DataNode `json:"dataNode,omitempty"`

	// +kubebuilder:validation:Optional
	IndexNode *IndexNode `json:"indexNode,omitempty"`

	// LogLevel defines Log level in Milvus. You can configure this parameter as debug, info, warn, error, panic, or fatal
	// +kubebuilder:validation:Enum={"debug", "info", "warn", "error", "panic", "fatal"}
	// +kubebuilder:default="info"
	LogLevel string `json:"logLevel"`
}

// MiluvsClusterConditionType is a valid value for MiluvsClusterConditionType.Type.
type MiluvsClusterConditionType string

// MilvusStatus is a type for milvus status.
type MilvusStatus string

const (
	// StatusCreating is the status of creating.
	StatusCreating MilvusStatus = "Creating"
	// StatusHealthy is the status of healthy.
	StatusHealthy MilvusStatus = "Healthy"
	// StatusUnHealthy is the status of unhealthy.
	StatusUnHealthy MilvusStatus = "Unhealthy"

	// EtcdReady means the Etcd is ready.
	EtcdReady MiluvsClusterConditionType = "EtcdReady"
	// StorageReady means the Storage is ready.
	StorageReady MiluvsClusterConditionType = "StorageReady"
	// PulsarReady means the Storage is ready.
	PulsarReady MiluvsClusterConditionType = "PulsarReady"
	// ServiceReady means the Service of Milvus is ready.
	ServiceReady MiluvsClusterConditionType = "ServiceReady"

	// ReasonEndpointsHealthy means the
	ReasonEndpointsHealthy string = "EndpointsHealty"
)

// MilvusClusterStatus defines the observed state of MilvusCluster
type MilvusClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Status indicates the overall status of the Milvus
	// Status can be "Creating", "Healthy" and "Unhealthy"
	// +kubebuilder:default="Creating"
	Status MilvusStatus `json:"status"`

	// Conditions of each components
	Conditions []MilvusClusterCondition `json:"conditions,omitempty"`

	// Status of each etcd endpoint
	EtcdStatus []MilvusEtcdStatus `json:"etcdStatus,omitempty"`

	// Status of each storage endpoint
	StorageStatus []MilvusStorageStatus `json:"storageStatus,omitempty"`
}

// MilvusEtcdStatus contains a list of all etcd endpoints status
type MilvusEtcdStatus struct {
	Endpoint string `json:"endpoint"`
	Healthy  bool   `json:"healthy"`
	Error    string `json:"error,omitempty"`
}

// MilvusStorageStatus contains a list of all storage endpoints status
type MilvusStorageStatus struct {
	Endpoint string `json:"endpoint"`
	Status   string `json:"status"`
	Uptime   string `json:"uptime,omitempty"`
	Error    string `json:"error,omitempty"`
}

// MilvusClusterCondition contains details for the current condition of this pod.
type MilvusClusterCondition struct {
	// Type is the type of the condition.
	Type MiluvsClusterConditionType `json:"type"`
	// Status is the status of the condition.
	// Can be True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// Last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime *metav1.Time `json:"lastTransitionTime,omitempty"`
	// Unique, one-word, CamelCase reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`
	// Human-readable message indicating details about last transition.
	// +optional
	Message string `json:"message,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// MilvusCluster is the Schema for the milvusclusters API
type MilvusCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MilvusClusterSpec   `json:"spec,omitempty"`
	Status MilvusClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MilvusClusterList contains a list of MilvusCluster
type MilvusClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MilvusCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MilvusCluster{}, &MilvusClusterList{})
}
