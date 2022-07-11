package v1beta1

type DependencyDeletionPolicy string

const (
	DeletionPolicyDelete DependencyDeletionPolicy = "Delete"
	DeletionPolicyRetain DependencyDeletionPolicy = "Retain"
)

const (
	StorageTypeMinIO = "MinIO"
	StorageTypeS3    = "S3"
)

type MilvusDependencies struct {
	// +kubebuilder:validation:Optional
	Etcd MilvusEtcd `json:"etcd"`

	// +kubebuilder:validation:Enum:={"pulsar", "kafka", "rocksmq", ""}
	// +kubebuilder:validation:Optional
	// MsgStreamType default to pulsar for cluster, rocksmq for standalone
	MsgStreamType MsgStreamType `json:"msgStreamType,omitempty"`

	// +kubebuilder:validation:Optional
	Pulsar MilvusPulsar `json:"pulsar,omitempty"`

	// +kubebuilder:validation:Optional
	Kafka MilvusKafka `json:"kafka,omitempty"`

	// +kubebuilder:validation:Optional
	RocksMQ MilvusRocksMQ `json:"rocksmq,omitempty"`

	// +kubebuilder:validation:Optional
	Storage MilvusStorage `json:"storage"`
}

type MsgStreamType string

const (
	MsgStreamTypePulsar  MsgStreamType = "pulsar"
	MsgStreamTypeKafka   MsgStreamType = "kafka"
	MsgStreamTypeRocksMQ MsgStreamType = "rocksmq"
)

type MilvusEtcd struct {
	// +kubebuilder:validation:Optional
	Endpoints []string `json:"endpoints"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=false
	External bool `json:"external,omitempty"`

	// +kubebuilder:validation:Optional
	InCluster *InClusterConfig `json:"inCluster,omitempty"`
}

type InClusterConfig struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Values Values `json:"values,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum:={"Delete", "Retain"}
	// +kubebuilder:default:="Retain"
	DeletionPolicy DependencyDeletionPolicy `json:"deletionPolicy"`

	// +kubebuilder:validation:Optional
	PVCDeletion bool `json:"pvcDeletion,omitempty"`
}

type MilvusStorage struct {
	// +kubebuilder:default:="MinIO"
	// +kubebuilder:validation:Enum:={"MinIO", "S3", ""}
	// +kubebuilder:validation:Optional
	Type string `json:"type"`

	// +kubebuilder:validation:Optional
	SecretRef string `json:"secretRef"`

	// +kubebuilder:validation:Optional
	Endpoint string `json:"endpoint"`

	// +kubebuilder:validation:Optional
	InCluster *InClusterConfig `json:"inCluster,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=false
	External bool `json:"external,omitempty"`
}

// MilvusRocksMQ configuration
type MilvusRocksMQ struct {
	Persistence Persistence `json:"persistence"`
}

type MilvusPulsar struct {
	// +kubebuilder:validation:Optional
	InCluster *InClusterConfig `json:"inCluster,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=false
	External bool `json:"external,omitempty"`

	// +kubebuilder:validation:Optional
	Endpoint string `json:"endpoint"`
}

// MilvusKafka configuration
type MilvusKafka struct {
	// +kubebuilder:validation:Optional
	InCluster *InClusterConfig `json:"inCluster,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=false
	External bool `json:"external,omitempty"`

	// +kubebuilder:validation:Optional
	BrokerList []string `json:"brokerList,omitempty"`
}
