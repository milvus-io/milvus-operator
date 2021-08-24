package v1alpha1

type MiluvsEtcd struct {
	Endpoints []string `json:"endpoints"`

	// +kubebuilder:validation:Optional
	RootPath string `json:"rootPath,omitempty"`

	// +kubebuilder:validation:Optional
	KVSubPath string `json:"kvSubPath,omitempty"`

	// +kubebuilder:validation:Optional
	MetaSubPath string `json:"metaSubPath,omitempty"`

	// +kubebuilder:validation:Optional
	SegmentBinlogSubPath string `json:"segmentBinlogSubPath,omitempty"`

	// +kubebuilder:validation:Optional
	CollectionBinlogSubPath string `json:"collectionBinlogSubPath,omitempty"`

	// +kubebuilder:validation:Optional
	FlushStreamPosSubPath string `json:"flushStreamPosSubPath,omitempty"`

	// +kubebuilder:validation:Optional
	StatsStreamPosSubPath string `json:"statsStreamPosSubPath,omitempty"`
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

	// +kubebuilder:validation:Optional
	MaxMessageSize int64 `json:"maxMessageSize,omitempty"`
}
