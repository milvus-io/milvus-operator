package controllers

import (
	"fmt"

	"github.com/milvus-io/milvus-operator/api/v1alpha1"
	"github.com/milvus-io/milvus-operator/pkg/milvus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	MetricPortName     = "metrics"
	RootCoordName      = "rootcoord"
	DataCoordName      = "datacoord"
	QueryCoordName     = "querycoord"
	IndexCoordName     = "indexcoord"
	DataNodeName       = "datanode"
	QueryNodeName      = "querynode"
	IndexNodeName      = "indexnode"
	ProxyName          = "proxy"
	RootCoordConfName  = "rootCoord"
	DataCoordConfName  = "dataCoord"
	QueryCoordConfName = "queryCoord"
	IndexCoordConfName = "indexCoord"
	DataNodeConfName   = "dataNode"
	QueryNodeConfName  = "queryNode"
	IndexNodeConfName  = "indexNode"
	ProxyConfName      = "proxy"
)

type MilvusComponent struct {
	Name     string
	ConfName string
}

var (
	RootCoord  MilvusComponent = MilvusComponent{RootCoordName, RootCoordConfName}
	DataCoord  MilvusComponent = MilvusComponent{DataCoordName, DataCoordConfName}
	QueryCoord MilvusComponent = MilvusComponent{QueryCoordName, QueryCoordConfName}
	IndexCoord MilvusComponent = MilvusComponent{IndexCoordName, IndexCoordConfName}
	DataNode   MilvusComponent = MilvusComponent{DataNodeName, DataNodeConfName}
	QueryNode  MilvusComponent = MilvusComponent{QueryNodeName, QueryNodeConfName}
	IndexNode  MilvusComponent = MilvusComponent{IndexNodeName, IndexNodeConfName}
	Proxy      MilvusComponent = MilvusComponent{ProxyName, ProxyConfName}

	MilvusComponents = []MilvusComponent{
		RootCoord, DataCoord, QueryCoord, IndexCoord, DataNode, QueryNode, IndexNode, Proxy,
	}

	MilvusCoords = []MilvusComponent{
		RootCoord, DataCoord, QueryCoord, IndexCoord,
	}
)

var GetReplicasFuncMap = map[string]func(v1alpha1.MilvusClusterSpec) *int32{
	RootCoordName: func(spec v1alpha1.MilvusClusterSpec) *int32 {
		return spec.Com.RootCoord.Replicas
	},
	DataCoordName: func(spec v1alpha1.MilvusClusterSpec) *int32 {
		return spec.Com.DataCoord.Replicas
	},
	QueryCoordName: func(spec v1alpha1.MilvusClusterSpec) *int32 {
		return spec.Com.QueryCoord.Replicas
	},
	IndexCoordName: func(spec v1alpha1.MilvusClusterSpec) *int32 {
		return spec.Com.IndexCoord.Replicas
	},
	DataNodeName: func(spec v1alpha1.MilvusClusterSpec) *int32 {
		return spec.Com.DataNode.Replicas
	},
	QueryNodeName: func(spec v1alpha1.MilvusClusterSpec) *int32 {
		return spec.Com.QueryNode.Replicas
	},
	IndexNodeName: func(spec v1alpha1.MilvusClusterSpec) *int32 {
		return spec.Com.IndexNode.Replicas
	},
	ProxyName: func(spec v1alpha1.MilvusClusterSpec) *int32 {
		return spec.Com.Proxy.Replicas
	},
}

var GetPortFuncMap = map[string]func(v1alpha1.MilvusClusterSpec) int32{
	RootCoordName: func(spec v1alpha1.MilvusClusterSpec) int32 {
		return GetPortByDefault(spec.Com.RootCoord.Port, milvus.RootCoordPort)
	},
	DataCoordName: func(spec v1alpha1.MilvusClusterSpec) int32 {
		return GetPortByDefault(spec.Com.DataCoord.Port, milvus.DataCoordPort)
	},
	QueryCoordName: func(spec v1alpha1.MilvusClusterSpec) int32 {
		return GetPortByDefault(spec.Com.QueryCoord.Port, milvus.QueryCoordPort)
	},
	IndexCoordName: func(spec v1alpha1.MilvusClusterSpec) int32 {
		return GetPortByDefault(spec.Com.IndexCoord.Port, milvus.IndexCoordPort)
	},
	DataNodeName: func(spec v1alpha1.MilvusClusterSpec) int32 {
		return GetPortByDefault(spec.Com.DataNode.Port, milvus.DataNodePort)
	},
	QueryNodeName: func(spec v1alpha1.MilvusClusterSpec) int32 {
		return GetPortByDefault(spec.Com.QueryNode.Port, milvus.QueryNodePort)
	},
	IndexNodeName: func(spec v1alpha1.MilvusClusterSpec) int32 {
		return GetPortByDefault(spec.Com.IndexNode.Port, milvus.IndexNodePort)
	},
	ProxyName: func(spec v1alpha1.MilvusClusterSpec) int32 {
		return GetPortByDefault(spec.Com.Proxy.Port, milvus.ProxyPort)
	},
}

func (c MilvusComponent) GetReplicas(spec v1alpha1.MilvusClusterSpec) *int32 {
	return GetReplicasFuncMap[c.Name](spec)
}

func (c MilvusComponent) String() string {
	return c.Name
}

func (c MilvusComponent) GetInstanceName(instance string) string {
	return fmt.Sprintf("%s-milvus-%s", instance, c.Name)
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
	return c.Name
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

	return spec.Com.Proxy.ServiceType
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
	return GetPortFuncMap[c.Name](spec)
}

func GetPortByDefault(specPort, defaultPort int32) int32 {
	if specPort != 0 {
		return specPort
	}
	return defaultPort
}
