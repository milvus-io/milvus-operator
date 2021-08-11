package milvus

type MilvusConfig struct {
	Etcd       MilvusConfigEtcd       `json:"etcd"`
	Minio      MilvusConfigMinio      `json:"minio"`
	Pulsar     MilvusConfigPulsar     `json:"pulsar"`
	Proxy      MilvusConfigProxy      `json:"proxy"`
	RootCoord  MilvusConfigRootCoord  `json:"rootCoord"`
	QueryCoord MilvusConfigQueryCoord `json:"queryCoord"`
	DataCoord  MilvusConfigDataCoord  `json:"dataCoord"`
	IndexCoord MilvusConfigIndexCoord `json:"indexCoord"`
	QueryNode  MilvusConfigQueryNode  `json:"queryNode"`
	DataNode   MilvusConfigDataNode   `json:"dataNode"`
	IndexNode  MilvusConfigIndexNode  `json:"indexNode"`
	Log        MilvusConfigLog        `json:"log"`
}

type MilvusConfigEtcd struct {
	Endpoints []string `json:"endpoints"`
	RootPath  string   `json:"rootPath"`
}

type MilvusConfigMinio struct {
	Address         string `json:"address"`
	Port            int32  `json:"port"`
	AccessKeyID     string `json:"accessKeyID"`
	SecretAccessKey string `json:"secretAccessKey"`
	UseSSL          bool   `json:"useSSL"`
	BucketName      string `json:"bucketName"`
}

type MilvusConfigPulsar struct {
	Address        string `json:"address"`
	Port           int32  `json:"port"`
	MaxMessageSize int64  `json:"maxMessageSize"`
}

type MilvusConfigGRPC struct {
	ServerMaxRecvSize int64 `json:"serverMaxRecvSize"`
	ServerMaxSendSize int64 `json:"serverMaxSendSize"`
	ClientMaxRecvSize int64 `json:"clientMaxRecvSize"`
	ClientMaxSendSize int64 `json:"clientMaxSendSize"`
}

type MilvusConfigNode struct {
	Port int32            `json:"port"`
	GRPC MilvusConfigGRPC `json:"grpc"`
}

type MilvusConfigCoord struct {
	Address string           `json:"address"`
	Port    int32            `json:"port"`
	GRPC    MilvusConfigGRPC `json:"grpc"`
}

type MilvusConfigRootCoord struct {
	MilvusConfigCoord `json:",inline"`
}

type MilvusConfigQueryCoord struct {
	MilvusConfigCoord `json:",inline"`
}

type MilvusConfigDataCoord struct {
	MilvusConfigCoord `json:",inline"`
}

type MilvusConfigIndexCoord struct {
	MilvusConfigCoord `json:",inline"`
}

type MilvusConfigDataNode struct {
	MilvusConfigNode `json:",inline"`
}

type MilvusConfigQueryNode struct {
	MilvusConfigNode `json:",inline"`
	GracefulTime     int32 `json:"gracefulTime"`
}

type MilvusConfigIndexNode struct {
	MilvusConfigNode `json:",inline"`
}

type MilvusConfigProxy struct {
	MilvusConfigNode `json:",inline"`
}

type MilvusConfigLog struct {
	Level  string `json:"level"`
	Format string `json:"format"`
}
