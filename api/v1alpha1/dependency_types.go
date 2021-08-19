package v1alpha1

type MiluvsEtcd struct {
	Endpoints []string `json:"endpoints"`

	// +kubebuilder:validation:Optional
	RootPath string `json:"rootPath,omitempty"`
}

type MilvusS3 struct {
	Endpoint string `json:"endpoint"`

	// +kubebuilder:validation:Optional
	SecretRef string `json:"secretRef"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	Insecure bool `json:"insecure"`

	// +kubebuilder:validation:Optional
	Bucket string `json:"bucket,omitempty"`
}

type MilvusPulsar struct {
	Endpoint string `json:"endpoint"`
}
