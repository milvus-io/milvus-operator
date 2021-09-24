package v1alpha1

type MilvusEtcd struct {
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

	// +kubebuilder:validation:Optional
	Endpoints []string `json:"endpoints"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	External bool `json:"external,omitempty"`

	// +kubebuilder:validation:Optional
	InCluster *InClusterEtcd `json:"inCluster,omitempty"`
}

type InClusterEtcd struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Values Extension `json:"values,omitempty"`
}

type MilvusStorage struct {
	// +kubebuilder:validation:Enum={"Minio", "S3"}
	// +kubebuilder:default="Minio"
	Type string `json:"type"`

	// +kubebuilder:validation:Optional
	SecretRef string `json:"secretRef"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	Insecure bool `json:"insecure,omitempty"`

	// +kubebuilder:validation:Optional
	Bucket string `json:"bucket,omitempty"`

	// +kubebuilder:validation:Optional
	Endpoint string `json:"endpoint"`

	// +kubebuilder:validation:Optional
	InCluster *InClusterStorage `json:"inCluster,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	External bool `json:"external,omitempty"`
}

type InClusterStorage struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Values Extension `json:"values,omitempty"`
}

type MilvusPulsar struct {
	// +kubebuilder:validation:Optional
	MaxMessageSize int64 `json:"maxMessageSize,omitempty"`

	// +kubebuilder:validation:Optional
	InCluster *InClusterPulsar `json:"inCluster,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	External bool `json:"external,omitempty"`

	// +kubebuilder:validation:Optional
	Endpoint string `json:"endpoint"`
}

type InClusterPulsar struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Values Extension `json:"values,omitempty"`
}
