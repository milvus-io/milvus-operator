package v1alpha1

type Node struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=1
	Replicas *int32 `json:"replicas,omitempty"`
}

type Coordinator struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=1
	Replicas *int32 `json:"replicas,omitempty"`
}

type QueryNode struct {
	Node `json:",inline"`

	// Minimum time before the newly inserted data can be searched. Unit: ms
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=1
	GracefulTime *int32 `json:"gracefulTime,omitempty"`
}

type DataNode struct {
	Node `json:",inline"`

	// Maximum row count of a segment buffered in memory
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=1
	InsertBufSize *int64 `json:"insertBufSize,omitempty"`
}

type IndexNode struct {
	Node `json:",inline"`
}

type Proxy struct {
	Node `json:",inline"`
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
