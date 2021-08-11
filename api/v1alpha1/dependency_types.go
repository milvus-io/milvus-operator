package v1alpha1

type MiluvsEtcd struct {
	Endpoints []string `json:"endpoints"`
	RootPath  string   `json:"rootPath"`
}

type MilvusS3 struct {
	Endpoint  string `json:"endpoint"`
	SecretRef string `json:"secretRef"`

	// +kubebuilder:default=false
	Insecure bool   `json:"insecure"`
	Bucket   string `json:"bucket"`
}

type MilvusPulsar struct {
	Endpoint string `json:"endpoint"`
}
