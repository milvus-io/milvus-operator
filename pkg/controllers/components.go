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

	"github.com/milvus-io/milvus-operator/apis/milvus.io/v1beta1"
	"github.com/milvus-io/milvus-operator/pkg/util"
)

// const name or ports
const (
	MetricPortName = "metrics"
	MetricPath     = "/metrics"

	MixCoordName   = "mixcoord"
	RootCoordName  = "rootcoord"
	DataCoordName  = "datacoord"
	QueryCoordName = "querycoord"
	IndexCoordName = "indexcoord"
	DataNodeName   = "datanode"
	QueryNodeName  = "querynode"
	IndexNodeName  = "indexnode"
	ProxyName      = "proxy"
	StandaloneName = "standalone"
	MilvusName     = "milvus"

	MixCoordFieldName   = "MixCoord"
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
	MultiplePorts  = -1
	RootCoordPort  = 53100
	DataCoordPort  = 13333
	QueryCoordPort = 19531
	IndexCoordPort = 31000
	IndexNodePort  = 21121
	QueryNodePort  = 21123
	DataNodePort   = 21124
	ProxyPort      = 19530
	// TODO: use configurable port?
	MilvusPort     = ProxyPort
	StandalonePort = MilvusPort
)

// MilvusComponent contains basic info of a milvus cluster component
type MilvusComponent struct {
	Name        string
	FieldName   string
	DefaultPort int32
}

// define MilvusComponents
var (
	MixCoord   = MilvusComponent{MixCoordName, MixCoordFieldName, MultiplePorts}
	RootCoord  = MilvusComponent{RootCoordName, RootCoordFieldName, RootCoordPort}
	DataCoord  = MilvusComponent{DataCoordName, DataCoordFieldName, DataCoordPort}
	QueryCoord = MilvusComponent{QueryCoordName, QueryCoordFieldName, QueryCoordPort}
	IndexCoord = MilvusComponent{IndexCoordName, IndexCoordFieldName, IndexCoordPort}
	DataNode   = MilvusComponent{DataNodeName, DataNodeFieldName, DataNodePort}
	QueryNode  = MilvusComponent{QueryNodeName, QueryNodeFieldName, QueryNodePort}
	IndexNode  = MilvusComponent{IndexNodeName, IndexNodeFieldName, IndexNodePort}
	Proxy      = MilvusComponent{ProxyName, ProxyFieldName, ProxyPort}

	// Milvus standalone
	MilvusStandalone = MilvusComponent{StandaloneName, StandaloneFieldName, StandalonePort}

	MixtureComponents = []MilvusComponent{
		MixCoord, DataNode, QueryNode, IndexNode, Proxy,
	}

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

// GetComponentsBySpec returns the components by the spec
func GetComponentsBySpec(spec v1beta1.MilvusSpec) []MilvusComponent {
	if spec.Mode == v1beta1.MilvusModeStandalone {
		return StandaloneComponents
	}
	if spec.Com.MixCoord != nil {
		return MixtureComponents
	}
	return MilvusComponents
}

// IsCoord return if it's a coord by its name
func (c MilvusComponent) IsCoord() bool {
	if c.Name == MixCoordName {
		return true
	}
	return strings.HasSuffix(c.Name, "coord")
}

// IsCoord return if it's a coord by its name
func (c MilvusComponent) IsStandalone() bool {
	return c.Name == StandaloneName
}

// IsCoord return if it's a node by its name
func (c MilvusComponent) IsNode() bool {
	return strings.HasSuffix(c.Name, "node")
}

// GetReplicas returns the replicas for the component
func (c MilvusComponent) GetReplicas(spec v1beta1.MilvusSpec) *int32 {
	replicas, _ := reflect.ValueOf(spec.Com).
		FieldByName(c.FieldName).Elem().
		FieldByName("Component").
		FieldByName("Replicas").Interface().(*int32)
	return replicas
}

var mixtureRunCommands = []string{"mixture", "-rootcoord", "-querycoord", "-datacoord", "-indexcoord"}

// String returns the name of the component
func (c MilvusComponent) GetRunCommands() []string {
	if c.Name == MixCoordName {
		return mixtureRunCommands
	}
	return []string{c.Name}
}

// String returns the name of the component
func (c MilvusComponent) GetName() string {
	return c.Name
}

// GetDeploymentName returns the name of the component deployment
func (c MilvusComponent) GetDeploymentName(instance string) string {
	return fmt.Sprintf("%s-milvus-%s", instance, c.Name)
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
func (c MilvusComponent) SetStatusReplicas(status *v1beta1.MilvusReplicas, replicas int) {
	reflect.ValueOf(status).Elem().FieldByName(c.FieldName).SetInt(int64(replicas))
}

// GetPortName returns the port name of the component container
func (c MilvusComponent) GetPortName() string {
	if c.Name == StandaloneName || c.Name == ProxyName {
		return MilvusName
	}
	return c.Name
}

// GetContainerPorts returns the ports of the component container
func (c MilvusComponent) GetContainerPorts(spec v1beta1.MilvusSpec) []corev1.ContainerPort {
	return []corev1.ContainerPort{
		{
			Name:          c.GetPortName(),
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
func (c MilvusComponent) GetServiceType(spec v1beta1.MilvusSpec) corev1.ServiceType {
	if c == Proxy || c == MilvusStandalone {
		return spec.GetServiceComponent().ServiceType
	}
	return corev1.ServiceTypeClusterIP

}

// GetServicePorts returns the ports of the component service
func (c MilvusComponent) GetServicePorts(spec v1beta1.MilvusSpec) []corev1.ServicePort {
	servicePorts := []corev1.ServicePort{}
	servicePorts = append(servicePorts, corev1.ServicePort{
		Name:       c.GetPortName(),
		Protocol:   corev1.ProtocolTCP,
		Port:       c.GetComponentPort(spec),
		TargetPort: intstr.FromString(c.GetPortName()),
	})
	servicePorts = append(servicePorts, corev1.ServicePort{
		Name:       MetricPortName,
		Protocol:   corev1.ProtocolTCP,
		Port:       MetricPort,
		TargetPort: intstr.FromString(MetricPortName),
	})

	return servicePorts
}

// GetComponentPort returns the port of the component
func (c MilvusComponent) GetComponentPort(spec v1beta1.MilvusSpec) int32 {
	return c.DefaultPort
}

// GetComponentSpec returns the component spec
func (c MilvusComponent) GetComponentSpec(spec v1beta1.MilvusSpec) v1beta1.ComponentSpec {
	value := reflect.ValueOf(spec.Com).FieldByName(c.FieldName).Elem().FieldByName("ComponentSpec")
	comSpec, _ := value.Interface().(v1beta1.ComponentSpec)
	return comSpec
}

// GetConfCheckSum returns the checksum of the component configuration
func GetConfCheckSum(spec v1beta1.MilvusSpec) string {
	conf := map[string]interface{}{}
	conf["conf"] = spec.Conf.Data
	conf["etcd-endpoints"] = spec.Dep.Etcd.Endpoints
	conf["pulsar-endpoint"] = spec.Dep.Pulsar.Endpoint
	conf["kafka-brokerList"] = spec.Dep.Kafka.BrokerList
	conf["storage-endpoint"] = spec.Dep.Storage.Endpoint

	b, err := json.Marshal(conf)
	if err != nil {
		return ""
	}

	return util.CheckSum(b)
}

// GetMilvusConfCheckSum returns the checksum of the component configuration
func GetMilvusConfCheckSum(spec v1beta1.MilvusSpec) string {
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
	if c.IsCoord() || c.IsStandalone() {
		return appsv1.DeploymentStrategy{
			Type: appsv1.RecreateDeploymentStrategyType,
		}
	}

	return appsv1.DeploymentStrategy{
		Type: appsv1.RollingUpdateDeploymentStrategyType,
		RollingUpdate: &appsv1.RollingUpdateDeployment{
			MaxUnavailable: &intstr.IntOrString{Type: intstr.Int, IntVal: 0},
		},
	}
}

type ComponentSpec = v1beta1.ComponentSpec

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
