package controllers

import (
	"fmt"

	"github.com/milvus-io/milvus-operator/api/v1alpha1"
	"github.com/milvus-io/milvus-operator/pkg/milvus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
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

	MetricPortName = "metrics"
)

var (
	MilvusComponents = []MilvusComponent{
		RootCoord, DataCoord, QueryCoord, IndexCoord, DataNode, QueryNode, IndexNode, Proxy,
	}
)

func (c MilvusComponent) GetReplicas(spec v1alpha1.MilvusClusterSpec) *int32 {
	switch c {
	case Proxy:
		return spec.Proxy.Replicas
	case QueryNode:
		return spec.QueryNode.Replicas
	case DataNode:
		return spec.DataNode.Replicas
	case IndexNode:
		return spec.IndexNode.Replicas
	case RootCoord:
		return spec.RootCoord.Replicas
	case QueryCoord:
		return spec.QueryCoord.Replicas
	case DataCoord:
		return spec.DataCoord.Replicas
	case IndexCoord:
		return spec.IndexCoord.Replicas
	}

	return nil
}

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

func (c MilvusComponent) GetContainerPorts(spec v1alpha1.MilvusClusterSpec) []corev1.ContainerPort {
	return []corev1.ContainerPort{
		{
			Name:          c.String(),
			ContainerPort: c.GetComponentPort(spec),
			Protocol:      corev1.ProtocolTCP,
		},
		{
			Name:          MetricPortName,
			ContainerPort: milvus.MetricPort,
			Protocol:      corev1.ProtocolTCP,
		},
	}
}

func (c MilvusComponent) GetServiceType(spec v1alpha1.MilvusClusterSpec) corev1.ServiceType {
	if c != Proxy {
		return corev1.ServiceTypeClusterIP
	}

	return spec.Proxy.ServiceType
}

func (c MilvusComponent) GetServicePorts(spec v1alpha1.MilvusClusterSpec) []corev1.ServicePort {
	return []corev1.ServicePort{
		{
			Name:       c.String(),
			Protocol:   corev1.ProtocolTCP,
			Port:       c.GetComponentPort(spec),
			TargetPort: intstr.FromString(c.String()),
		},
		{
			Name:       MetricPortName,
			Protocol:   corev1.ProtocolTCP,
			Port:       milvus.MetricPort,
			TargetPort: intstr.FromString(MetricPortName),
		},
	}
}

func (c MilvusComponent) GetComponentPort(spec v1alpha1.MilvusClusterSpec) int32 {
	switch c {
	case RootCoord:
		return GetPortByDefault(spec.RootCoord.Port, milvus.RootCoordPort)

	case DataCoord:
		return GetPortByDefault(spec.DataCoord.Port, milvus.DataCoordPort)

	case QueryCoord:
		return GetPortByDefault(spec.QueryCoord.Port, milvus.QueryCoordPort)

	case IndexCoord:
		return GetPortByDefault(spec.IndexCoord.Port, milvus.IndexCoordPort)

	case QueryNode:
		return GetPortByDefault(spec.QueryNode.Port, milvus.QueryNodePort)

	case DataNode:
		return GetPortByDefault(spec.DataNode.Port, milvus.DataNodePort)

	case IndexNode:
		return GetPortByDefault(spec.IndexNode.Port, milvus.IndexNodePort)

	case Proxy:
		return GetPortByDefault(spec.Proxy.Port, milvus.ProxyPort)

	default:
		return 0
	}
}

func GetPortByDefault(specPort, defaultPort int32) int32 {
	if specPort != 0 {
		return specPort
	}
	return defaultPort
}
