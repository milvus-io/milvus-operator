package v1alpha1

// Config contains config files for milvus
type Config struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Milvus Values `json:"milvus,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Component Values `json:"component,omitempty"`
}
