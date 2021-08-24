package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
)

type Component struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=1
	Replicas *int32 `json:"replicas,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=65535
	Port int32 `json:"port,omitempty"`

	// +kubebuilder:validation:Optional
	GRPC *ConfigGRPC `json:"grpc,omitempty"`
}

type ConfigGRPC struct {
	// +kubebuilder:validation:Optional
	ServerMaxRecvSize int32 `json:"serverMaxRecvSize,omitempty"`

	// +kubebuilder:validation:Optional
	ServerMaxSendSize int32 `json:"serverMaxSendSize,omitempty"`

	// +kubebuilder:validation:Optional
	ClientMaxRecvSize int32 `json:"clientMaxRecvSize,omitempty"`

	// +kubebuilder:validation:Optional
	ClientMaxSendSize int32 `json:"clientMaxSendSize,omitempty"`
}

type Node struct {
	Component `json:",inline"`
}

type Coordinator struct {
	Component `json:",inline"`
}

type QueryNode struct {
	Node `json:",inline"`

	// Minimum time before the newly inserted data can be searched. Unit: ms
	// +kubebuilder:validation:Optional
	GracefulTime int32 `json:"gracefulTime,omitempty"`
}

type DataNode struct {
	Node `json:",inline"`

	// Maximum row count of a segment buffered in memory
	// +kubebuilder:validation:Optional
	InsertBufSize int64 `json:"insertBufSize,omitempty"`
}

type IndexNode struct {
	Node `json:",inline"`
}

type Proxy struct {
	Node `json:",inline"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum={"ClusterIP", "NodePort", "LoadBalancer"}
	// +kubebuilder:default="ClusterIP"
	ServiceType corev1.ServiceType `json:"serviceType,omitempty"`
}

type RootCoordinator struct {
	Coordinator `json:",inline"`
}

type DataCoordinator struct {
	Coordinator `json:",inline"`
}

type QueryCoordinator struct {
	Coordinator `json:",inline"`
}

type IndexCoordinator struct {
	Coordinator `json:",inline"`
}
