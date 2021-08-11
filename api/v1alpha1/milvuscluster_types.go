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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MilvusClusterSpec defines the desired state of MilvusCluster
type MilvusClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Optional
	Etcd *MiluvsEtcd `json:"etcd"`

	// +kubebuilder:validation:Optional
	S3 *MilvusS3 `json:"s3"`

	// +kubebuilder:validation:Optional
	Pulsar *MilvusPulsar `json:"pulsar"`

	// LogLevel defines Log level in Milvus. You can configure this parameter as debug, info, warn, error, panic, or fatal
	// +kubebuilder:validation:Enum={"debug", "info", "warn", "error", "panic", "fatal"}
	// +kubebuilder:default="info"
	LogLevel string `json:"logLevel"`

	QueryNode *QueryNode `json:"queryNode"`

	DataNode *DataNode `json:"dataNode"`
}

// MilvusClusterStatus defines the observed state of MilvusCluster
type MilvusClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
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
