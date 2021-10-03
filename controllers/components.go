package controllers

import (
	"fmt"
	"reflect"

	"github.com/milvus-io/milvus-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	MetricPortName = "metrics"

	RootCoordName  = "rootcoord"
	DataCoordName  = "datacoord"
	QueryCoordName = "querycoord"
	IndexCoordName = "indexcoord"
	DataNodeName   = "datanode"
	QueryNodeName  = "querynode"
	IndexNodeName  = "indexnode"
	ProxyName      = "proxy"

	RootCoordFieldName  = "RootCoord"
	DataCoordFieldName  = "DataCoord"
	QueryCoordFieldName = "QueryCoord"
	IndexCoordFieldName = "IndexCoord"
	DataNodeFieldName   = "DataNode"
	QueryNodeFieldName  = "QueryNode"
	IndexNodeFieldName  = "IndexNode"
	ProxyFieldName      = "Proxy"

	MetricPort     = 9091
	RootCoordPort  = 53100
	DataCoordPort  = 13333
	QueryCoordPort = 19531
	IndexCoordPort = 31000
	IndexNodePort  = 21121
	QueryNodePort  = 21123
	DataNodePort   = 21124
	ProxyPort      = 19530
)

type MilvusComponent struct {
	Name        string
	FieldName   string
	DefaultPort int32
}

var (
	RootCoord  = MilvusComponent{RootCoordName, RootCoordFieldName, RootCoordPort}
	DataCoord  = MilvusComponent{DataCoordName, DataCoordFieldName, DataCoordPort}
	QueryCoord = MilvusComponent{QueryCoordName, QueryCoordFieldName, QueryCoordPort}
	IndexCoord = MilvusComponent{IndexCoordName, IndexCoordFieldName, IndexCoordPort}
	DataNode   = MilvusComponent{DataNodeName, DataNodeFieldName, DataNodePort}
	QueryNode  = MilvusComponent{QueryNodeName, QueryNodeFieldName, QueryNodePort}
	IndexNode  = MilvusComponent{IndexNodeName, IndexNodeFieldName, IndexNodePort}
	Proxy      = MilvusComponent{ProxyName, ProxyFieldName, ProxyPort}

	MilvusComponents = []MilvusComponent{
		RootCoord, DataCoord, QueryCoord, IndexCoord, DataNode, QueryNode, IndexNode, Proxy,
	}

	MilvusCoords = []MilvusComponent{
		RootCoord, DataCoord, QueryCoord, IndexCoord,
	}
)

func (c MilvusComponent) GetEnv(spec v1alpha1.MilvusClusterSpec) []corev1.EnvVar {
	env := c.GetComponentSpec(spec).Env
	return MergeEnvVar(spec.Com.Env, env)
}

func (c MilvusComponent) GetImagePullSecrets(spec v1alpha1.MilvusClusterSpec) []corev1.LocalObjectReference {
	pullSecrets := c.GetComponentSpec(spec).ImagePullSecrets
	if len(pullSecrets) > 0 {
		return pullSecrets
	}
	return spec.Com.ImagePullSecrets
}

func (c MilvusComponent) GetImagePullPolicy(spec v1alpha1.MilvusClusterSpec) corev1.PullPolicy {
	pullPolicy := c.GetComponentSpec(spec).ImagePullPolicy
	if pullPolicy != nil {
		return *pullPolicy
	}

	if spec.Com.ImagePullPolicy != nil {
		return *spec.Com.ImagePullPolicy
	}
	return corev1.PullIfNotPresent
}

func (c MilvusComponent) GetImage(spec v1alpha1.MilvusClusterSpec) string {
	componentImage := c.GetComponentSpec(spec).Image
	if len(componentImage) > 0 {
		return componentImage
	}

	return spec.Com.Image
}

func (c MilvusComponent) GetReplicas(spec v1alpha1.MilvusClusterSpec) *int32 {
	replicas, _ := reflect.ValueOf(spec.Com).
		FieldByName(c.FieldName).
		FieldByName("Component").
		FieldByName("Replicas").Interface().(*int32)
	return replicas
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
			ContainerPort: MetricPort,
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
			Port:       MetricPort,
			TargetPort: intstr.FromString(MetricPortName),
		},
	}
}

func (c MilvusComponent) GetComponentPort(spec v1alpha1.MilvusClusterSpec) int32 {
	port, _ := reflect.ValueOf(spec.Com).
		FieldByName(c.FieldName).
		FieldByName("Component").
		FieldByName("Port").Interface().(int32)

	if port != 0 {
		return port
	}

	return c.DefaultPort
}

func (c MilvusComponent) GetComponentSpec(spec v1alpha1.MilvusClusterSpec) v1alpha1.ComponentSpec {
	value := reflect.ValueOf(spec.Com).FieldByName(c.FieldName).FieldByName("ComponentSpec")
	comSpec, _ := value.Interface().(v1alpha1.ComponentSpec)
	return comSpec
}
