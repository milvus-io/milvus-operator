package controllers

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/milvus-io/milvus-operator/apis/milvus.io/v1alpha1"
	"github.com/milvus-io/milvus-operator/pkg/util"
)

// const name or ports
const (
	MetricPortName = "metrics"
	MetricPath     = "/metrics"

	RootCoordName  = "rootcoord"
	DataCoordName  = "datacoord"
	QueryCoordName = "querycoord"
	IndexCoordName = "indexcoord"
	DataNodeName   = "datanode"
	QueryNodeName  = "querynode"
	IndexNodeName  = "indexnode"
	ProxyName      = "proxy"
	MilvusName     = "milvus"

	RootCoordFieldName  = "RootCoord"
	DataCoordFieldName  = "DataCoord"
	QueryCoordFieldName = "QueryCoord"
	IndexCoordFieldName = "IndexCoord"
	DataNodeFieldName   = "DataNode"
	QueryNodeFieldName  = "QueryNode"
	IndexNodeFieldName  = "IndexNode"
	ProxyFieldName      = "Proxy"
	StandaloneFieldName = "Standalone"

	MetricPort     = 9091
	RootCoordPort  = 53100
	DataCoordPort  = 13333
	QueryCoordPort = 19531
	IndexCoordPort = 31000
	IndexNodePort  = 21121
	QueryNodePort  = 21123
	DataNodePort   = 21124
	ProxyPort      = 19530
	// TODO: use configurable port?
	MilvusPort = ProxyPort
)

// MilvusComponent contains basic info of a milvus cluster component
type MilvusComponent struct {
	Name        string
	FieldName   string
	DefaultPort int32
}

// define MilvusComponents
var (
	RootCoord  = MilvusComponent{RootCoordName, RootCoordFieldName, RootCoordPort}
	DataCoord  = MilvusComponent{DataCoordName, DataCoordFieldName, DataCoordPort}
	QueryCoord = MilvusComponent{QueryCoordName, QueryCoordFieldName, QueryCoordPort}
	IndexCoord = MilvusComponent{IndexCoordName, IndexCoordFieldName, IndexCoordPort}
	DataNode   = MilvusComponent{DataNodeName, DataNodeFieldName, DataNodePort}
	QueryNode  = MilvusComponent{QueryNodeName, QueryNodeFieldName, QueryNodePort}
	IndexNode  = MilvusComponent{IndexNodeName, IndexNodeFieldName, IndexNodePort}
	Proxy      = MilvusComponent{ProxyName, ProxyFieldName, ProxyPort}

	// Milvus standalone
	MilvusStandalone = MilvusComponent{MilvusName, StandaloneFieldName, MilvusPort}

	MilvusComponents = []MilvusComponent{
		RootCoord, DataCoord, QueryCoord, IndexCoord, DataNode, QueryNode, IndexNode, Proxy,
	}

	StandaloneComponents = []MilvusComponent{
		MilvusStandalone,
	}

	MilvusCoords = []MilvusComponent{
		RootCoord, DataCoord, QueryCoord, IndexCoord,
	}
)

// IsCoord return if it's a coord by its name
func (c MilvusComponent) IsCoord() bool {
	return strings.HasSuffix(c.Name, "coord")
}

// IsCoord return if it's a node by its name
func (c MilvusComponent) IsNode() bool {
	return strings.HasSuffix(c.Name, "node")
}

// GetReplicas returns the replicas for the component
func (c MilvusComponent) GetReplicas(spec v1alpha1.MilvusClusterSpec) *int32 {
	replicas, _ := reflect.ValueOf(spec.Com).
		FieldByName(c.FieldName).
		FieldByName("Component").
		FieldByName("Replicas").Interface().(*int32)
	return replicas
}

// String returns the name of the component
func (c MilvusComponent) String() string {
	return c.Name
}

// GetInstanceName returns the name of the component instance
func (c MilvusComponent) GetInstanceName(instance string) string {
	if c == MilvusStandalone {
		return instance
	}
	return fmt.Sprintf("%s-milvus-%s", instance, c.Name)
}

// GetDeploymentInstanceName returns the name of the component deployment
func (c MilvusComponent) GetDeploymentInstanceName(instance string) string {
	return c.GetInstanceName(instance)
}

// GetServiceInstanceName returns the name of the component service
func GetServiceInstanceName(instance string) string {
	return instance + "-milvus"
}

// GetContainerName returns the name of the component container
func (c MilvusComponent) GetContainerName() string {
	return c.Name
}

// SetStatusReplica sets the replica status of the component, input status should not be nil
func (c MilvusComponent) SetStatusReplicas(status *v1alpha1.MilvusReplicas, replicas int) {
	reflect.ValueOf(status).Elem().FieldByName(c.FieldName).SetInt(int64(replicas))
}

// GetContainerPorts returns the ports of the component container
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

// GetServiceType returns the type of the component service
func (c MilvusComponent) GetServiceType(spec v1alpha1.MilvusClusterSpec) corev1.ServiceType {
	if c != Proxy {
		return corev1.ServiceTypeClusterIP
	}

	return spec.Com.Proxy.ServiceType
}

// GetServicePorts returns the ports of the component service
func (c MilvusComponent) GetServicePorts(spec v1alpha1.MilvusClusterSpec) []corev1.ServicePort {
	servicePorts := []corev1.ServicePort{}
	if !c.IsNode() {
		servicePorts = append(servicePorts, corev1.ServicePort{
			Name:       c.String(),
			Protocol:   corev1.ProtocolTCP,
			Port:       c.GetComponentPort(spec),
			TargetPort: intstr.FromString(c.String()),
		})
	}
	servicePorts = append(servicePorts, corev1.ServicePort{
		Name:       MetricPortName,
		Protocol:   corev1.ProtocolTCP,
		Port:       MetricPort,
		TargetPort: intstr.FromString(MetricPortName),
	})

	return servicePorts
}

// GetComponentPort returns the port of the component
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

// GetComponentSpec returns the component spec
func (c MilvusComponent) GetComponentSpec(spec v1alpha1.MilvusClusterSpec) v1alpha1.ComponentSpec {
	value := reflect.ValueOf(spec.Com).FieldByName(c.FieldName).FieldByName("ComponentSpec")
	comSpec, _ := value.Interface().(v1alpha1.ComponentSpec)
	return comSpec
}

// GetConfCheckSum returns the checksum of the component configuration
func GetConfCheckSum(spec v1alpha1.MilvusClusterSpec) string {
	conf := map[string]interface{}{}
	conf["conf"] = spec.Conf.Data
	conf["etcd-endpoints"] = spec.Dep.Etcd.Endpoints
	conf["pulsar-endpoint"] = spec.Dep.Pulsar.Endpoint
	conf["storage-endpoint"] = spec.Dep.Storage.Endpoint

	b, err := json.Marshal(conf)
	if err != nil {
		return ""
	}

	return util.CheckSum(b)
}

// GetMilvusConfCheckSum returns the checksum of the component configuration
func GetMilvusConfCheckSum(spec v1alpha1.MilvusSpec) string {
	conf := map[string]interface{}{}
	conf["conf"] = spec.Conf.Data
	conf["etcd-endpoints"] = spec.Dep.Etcd.Endpoints
	conf["storage-endpoint"] = spec.Dep.Storage.Endpoint

	b, err := json.Marshal(conf)
	if err != nil {
		return ""
	}

	return util.CheckSum(b)
}

func GetLivenessProbe() *corev1.Probe {
	return &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   "/healthz",
				Port:   intstr.FromInt(9091),
				Scheme: corev1.URISchemeHTTP,
			},
		},
		InitialDelaySeconds: 120,
		TimeoutSeconds:      3,
		PeriodSeconds:       30,
		FailureThreshold:    2,
		SuccessThreshold:    1,
	}
}

var GetReadinessProbe = GetLivenessProbe

func (c MilvusComponent) GetDeploymentStrategy() appsv1.DeploymentStrategy {
	if c.IsCoord() {
		return appsv1.DeploymentStrategy{
			Type: appsv1.RecreateDeploymentStrategyType,
		}
	}

	return appsv1.DeploymentStrategy{
		Type: appsv1.RollingUpdateDeploymentStrategyType,
		RollingUpdate: &appsv1.RollingUpdateDeployment{
			MaxUnavailable: &intstr.IntOrString{Type: intstr.Int, IntVal: 0},
			MaxSurge:       &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
		},
	}
}

type ComponentSpec = v1alpha1.ComponentSpec

const (
	CacheSizeEnvVarName = "CACHE_SIZE"
)

var (
	CacheSizeEnvVar = corev1.EnvVar{
		Name: CacheSizeEnvVarName,
		ValueFrom: &corev1.EnvVarSource{
			ResourceFieldRef: &corev1.ResourceFieldSelector{
				Divisor:  resource.MustParse("1Gi"),
				Resource: "limits.memory",
			},
		},
	}
)

// MergeComponentSpec merges the src ComponentSpec to dst
func MergeComponentSpec(src, dst ComponentSpec) ComponentSpec {
	if len(src.Image) > 0 {
		dst.Image = src.Image
	}

	if src.ImagePullPolicy != nil {
		dst.ImagePullPolicy = src.ImagePullPolicy
	}
	if dst.ImagePullPolicy == nil {
		policy := corev1.PullIfNotPresent
		dst.ImagePullPolicy = &policy
	}

	if len(src.ImagePullSecrets) > 0 {
		dst.ImagePullSecrets = src.ImagePullSecrets
	}

	src.Env = append(src.Env, CacheSizeEnvVar)
	dst.Env = MergeEnvVar(dst.Env, src.Env)

	if len(src.NodeSelector) > 0 {
		dst.NodeSelector = src.NodeSelector
	}

	if src.Affinity != nil {
		dst.Affinity = src.Affinity
	}

	if len(src.Tolerations) > 0 {
		dst.Tolerations = src.Tolerations
	}

	if src.Resources != nil {
		dst.Resources = src.Resources
	}
	if dst.Resources == nil {
		dst.Resources = &corev1.ResourceRequirements{}
	}
	return dst
}
