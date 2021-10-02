package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
)

type ComponentSpec struct {
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
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// +kubebuilder:validation:Optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
}

type MilvusComponents struct {
	ComponentSpec `json:",inline"`

	// +kubebuilder:validation:Optional
	Proxy Proxy `json:"proxy,omitempty"`

	// +kubebuilder:validation:Optional
	RootCoord RootCoordinator `json:"rootCoord,omitempty"`

	// +kubebuilder:validation:Optional
	IndexCoord IndexCoordinator `json:"indexCoord,omitempty"`

	// +kubebuilder:validation:Optional
	DataCoord DataCoordinator `json:"dataCoord,omitempty"`

	// +kubebuilder:validation:Optional
	QueryCoord QueryCoordinator `json:"queryCoord,omitempty"`

	// +kubebuilder:validation:Optional
	IndexNode IndexNode `json:"indexNode,omitempty"`

	// +kubebuilder:validation:Optional
	DataNode DataNode `json:"dataNode,omitempty"`

	// +kubebuilder:validation:Optional
	QueryNode QueryNode `json:"queryNode,omitempty"`
}

type Component struct {
	ComponentSpec `json:",inline"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=1
	Replicas *int32 `json:"replicas,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=65535
	Port int32 `json:"port,omitempty"`
}

type QueryNode struct {
	Component `json:",inline"`
}

type DataNode struct {
	Component `json:",inline"`
}

type IndexNode struct {
	Component `json:",inline"`
}

type Proxy struct {
	Component `json:",inline"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum={"ClusterIP", "NodePort", "LoadBalancer"}
	// +kubebuilder:default="ClusterIP"
	ServiceType corev1.ServiceType `json:"serviceType,omitempty"`
}

type RootCoordinator struct {
	Component `json:",inline"`
}

type DataCoordinator struct {
	Component `json:",inline"`
}

type QueryCoordinator struct {
	Component `json:",inline"`
}

type IndexCoordinator struct {
	Component `json:",inline"`
}
