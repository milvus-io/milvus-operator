package controllers

import (
	"fmt"

	"github.com/milvus-io/milvus-operator/pkg/milvus"
	corev1 "k8s.io/api/core/v1"
)

type MilvusComponent string

const (
	RootCoord  MilvusComponent = "rootcoord"
	DataCoord  MilvusComponent = "datacoord"
	QueryCoord MilvusComponent = "querycoord"
	IndexCoord MilvusComponent = "indexcoord"
	DataNode   MilvusComponent = "datanode"
	QueryNode  MilvusComponent = "querynode"
	IndexNode  MilvusComponent = "indexnode"
	Proxy      MilvusComponent = "proxy"
)

var (
	MilvusComponents = []MilvusComponent{
		RootCoord, DataCoord, QueryCoord, IndexCoord, DataNode, QueryNode, IndexNode, Proxy,
	}
)

func (c MilvusComponent) String() string {
	return string(c)
}

func (c MilvusComponent) GetInstanceName(instance string) string {
	return fmt.Sprintf("%s-milvus-%s", instance, c)
}

func (c MilvusComponent) GetDeploymentInstanceName(instance string) string {
	return c.GetInstanceName(instance)
}

func (c MilvusComponent) GetServiceInstanceName(instance string) string {
	if c == Proxy {
		return instance + "-milvus"
	}
	return c.GetInstanceName(instance)
}

func (c MilvusComponent) GetContainerName() string {
	return c.String()
}

func (c MilvusComponent) GetPort() int32 {
	switch c {
	case RootCoord:
		return milvus.RootCoordPort
	case DataCoord:
		return milvus.DataCoordPort
	case QueryCoord:
		return milvus.QueryCoordPort
	case IndexCoord:
		return milvus.IndexCoordPort
	case QueryNode:
		return milvus.QueryNodePort
	case DataNode:
		return milvus.DataNodePort
	case IndexNode:
		return milvus.IndexNodePort
	case Proxy:
		return milvus.ProxyPort
	default:
		return 0
	}
}

func (c MilvusComponent) GetContainerPorts() []corev1.ContainerPort {
	return []corev1.ContainerPort{
		{
			Name:          c.String(),
			ContainerPort: c.GetPort(),
			Protocol:      corev1.ProtocolTCP,
		},
		{
			Name:          "metrics",
			ContainerPort: milvus.MetricPort,
			Protocol:      corev1.ProtocolTCP,
		},
	}
}
