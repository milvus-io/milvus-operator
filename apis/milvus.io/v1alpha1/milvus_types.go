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
	networkv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MilvusSpec defines the desired state of Milvus
type MilvusSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Optional
	ComponentSpec `json:",inline"`

	// +kubebuilder:validation:Optional
	Replicas *int32 `json:"replicas,omitempty"`

	// +kubebuilder:validation:Optional
	Persistence Persistence `json:"persistence,omitempty"`

	// +kubebuilder:validation:Optional
	Ingress *MilvusIngress `json:"ingress,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum={"ClusterIP", "NodePort", "LoadBalancer"}
	// +kubebuilder:default="ClusterIP"
	ServiceType corev1.ServiceType `json:"serviceType,omitempty"`

	// +kubebuilder:validation:Optional
	ServiceLabels map[string]string `json:"serviceLabels,omitempty"`

	// +kubebuilder:validation:Optional
	ServiceAnnotations map[string]string `json:"serviceAnnotations,omitempty"`

	// +kubebuilder:validation:Optional
	Dep MilvusDependencies `json:"dependencies,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Conf Values `json:"config,omitempty"`
}

// MilvusStatus defines the observed state of Milvus
type MilvusStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Status indicates the overall status of the Milvus
	// Status can be "Creating", "Healthy" and "Unhealthy"
	// +kubebuilder:default:="Creating"
	Status MilvusHealthStatus `json:"status"`

	// Conditions of each components
	Conditions []MilvusCondition `json:"conditions,omitempty"`

	// Endpoint of milvus cluster
	Endpoint string `json:"endpoint,omitempty"`

	IngressStatus networkv1.IngressStatus `json:"ingress,omitempty"`

	// +kubebuilder:validation:Optional
	Replicas MilvusReplicas `json:"replicas,omitempty"`
}

// MilvusReplicas is the replicas of milvus components
type MilvusReplicas struct {
	//+kubebuilder:validation:Optional
	Proxy int `json:"proxy,omitempty"`
	//+kubebuilder:validation:Optional
	RootCoord int `json:"rootCoord,omitempty"`
	//+kubebuilder:validation:Optional
	DataCoord int `json:"dataCoord,omitempty"`
	//+kubebuilder:validation:Optional
	IndexCoord int `json:"indexCoord,omitempty"`
	//+kubebuilder:validation:Optional
	QueryCoord int `json:"queryCoord,omitempty"`
	//+kubebuilder:validation:Optional
	DataNode int `json:"dataNode,omitempty"`
	//+kubebuilder:validation:Optional
	IndexNode int `json:"indexNode,omitempty"`
	//+kubebuilder:validation:Optional
	QueryNode int `json:"queryNode,omitempty"`
	//+kubebuilder:validation:Optional
	Standalone int `json:"standalone,omitempty"`
}

// MilvusIngress defines the ingress of MilvusCluster
// TODO: add docs
type MilvusIngress struct {
	// +kubebuilder:validation:Optional
	Labels map[string]string `json:"labels,omitempty"`

	// +kubebuilder:validation:Optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// +kubebuilder:validation:Optional
	IngressClassName *string `json:"ingressClassName,omitempty"`

	// +kubebuilder:validation:Optional
	Hosts []string `json:"hosts,omitempty"`

	// TLSSecretRefs is a map of TLS secret to hosts
	// +kubebuilder:validation:Optional
	TLSSecretRefs map[string][]string `json:"tlsSecretRefs,omitempty"`
}

// MiluvsConditionType is a valid value for MiluvsConditionType.Type.
type MiluvsConditionType string

// MilvusHealthStatus is a type for milvus status.
type MilvusHealthStatus string

const (
	// StatusCreating is the status of creating.
	StatusCreating MilvusHealthStatus = "Creating"
	// StatusHealthy is the status of healthy.
	StatusHealthy MilvusHealthStatus = "Healthy"
	// StatusUnHealthy is the status of unhealthy.
	StatusUnHealthy MilvusHealthStatus = "Unhealthy"

	// EtcdReady means the Etcd is ready.
	EtcdReady MiluvsConditionType = "EtcdReady"
	// StorageReady means the Storage is ready.
	StorageReady MiluvsConditionType = "StorageReady"
	// MsgStreamReady means the MsgStream is ready.
	MsgStreamReady MiluvsConditionType = "MsgStreamReady"
	// MilvusReady means all components of Milvus are ready.
	MilvusReady MiluvsConditionType = "MilvusReady"

	// ReasonEndpointsHealthy means the endpoint is healthy
	ReasonEndpointsHealthy string = "EndpointsHealthy"
	// ReasonMilvusHealthy means milvus cluster is healthy
	ReasonMilvusHealthy string = "ReasonMilvusHealthy"
	// ReasonMilvusClusterHealthy means milvus cluster is healthy
	ReasonMilvusClusterHealthy string = "MilvusClusterHealthy"
	// ReasonMilvusClusterNotHealthy means at least one of milvus component is not healthy
	ReasonMilvusComponentNotHealthy string = "MilvusComponentNotHealthy"

	ReasonEtcdReady          = "EtcdReady"
	ReasonEtcdNotReady       = "EtcdNotReady"
	ReasonS3Ready            = "S3StorageAssumeReady"
	ReasonStorageReady       = "StorageReady"
	ReasonStorageNotReady    = "StorageNotReady"
	ReasonMsgStreamReady     = "MsgStreamReady"
	ReasonMsgStreamNotReady  = "MsgStreamReady"
	ReasonSecretNotExist     = "SecretNotExist"
	ReasonSecretErr          = "SecretError"
	ReasonSecretDecodeErr    = "SecretDecodeError"
	ReasonClientErr          = "ClientError"
	ReasonDependencyNotReady = "DependencyNotReady"
)

// MilvusCondition contains details for the current condition of this milvus/milvus cluster instance
type MilvusCondition struct {
	// Type is the type of the condition.
	Type MiluvsConditionType `json:"type"`
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

// +genclient
// +genclient:noStatus
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=milvuses,singular=milvus,shortName=mi
// Milvus is the Schema for the milvus API
type Milvus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MilvusSpec   `json:"spec,omitempty"`
	Status MilvusStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true
// MilvusList contains a list of Milvus
type MilvusList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Milvus `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Milvus{}, &MilvusList{})
}
