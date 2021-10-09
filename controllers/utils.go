package controllers

import (
	"bytes"
	_ "crypto/sha256"
	"fmt"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	dockerref "github.com/docker/distribution/reference"
	"github.com/milvus-io/milvus-operator/api/v1alpha1"
	"github.com/milvus-io/milvus-operator/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kstatus "sigs.k8s.io/kustomize/kstatus/status"
	"sigs.k8s.io/yaml"
)

const (
	AppLabel          = "app.kubernetes.io/"
	AppLabelInstance  = AppLabel + "instance"
	AppLabelVersion   = AppLabel + "version"
	AppLabelComponent = AppLabel + "component"
	AppLabelName      = AppLabel + "name"
)

func IsServicePortDuplicate(srcPorts []corev1.ServicePort, dst corev1.ServicePort) bool {
	for _, src := range srcPorts {
		if src.Name == dst.Name {
			return true
		}

		// Check for duplicate Ports, considering (protocol,port) pairs
		srcKey := corev1.ServicePort{Protocol: src.Protocol, Port: src.Port}
		dstKey := corev1.ServicePort{Protocol: dst.Protocol, Port: dst.Port}
		if srcKey == dstKey {
			return true
		}

		// Check for duplicate NodePorts, considering (protocol,port) pairs
		if dst.NodePort != 0 {
			srcKey = corev1.ServicePort{Protocol: src.Protocol, NodePort: src.NodePort}
			dstKey = corev1.ServicePort{Protocol: dst.Protocol, NodePort: dst.NodePort}
			if srcKey == dstKey {
				return true
			}
		}
	}

	return false
}

func MergeServicePort(src, dst []corev1.ServicePort) []corev1.ServicePort {
	if len(src) == 0 {
		return dst
	}

	if len(dst) == 0 {
		return src
	}

	merged := []corev1.ServicePort{}
	merged = append(merged, dst...)

	for _, srcPort := range src {
		if !IsServicePortDuplicate(dst, srcPort) {
			if srcPort.Name == "" {
				srcPort.Name = strings.ToLower(fmt.Sprintf("%s-%d", srcPort.Protocol, srcPort.Port))
			}
			merged = append(merged, srcPort)
		}
	}

	return merged
}

func MergeVolumeMount(src, dst []corev1.VolumeMount) []corev1.VolumeMount {
	if len(src) == 0 {
		return dst
	}
	if len(dst) == 0 {
		return src
	}

	srcMap := map[string]int{}
	for i, VolumeMount := range src {
		srcMap[VolumeMount.MountPath] = i
	}

	merged := []corev1.VolumeMount{}
	merged = append(merged, src...)
	for _, VolumeMount := range dst {
		if idx, ok := srcMap[VolumeMount.MountPath]; ok {
			merged[idx] = VolumeMount
		} else {
			merged = append(merged, VolumeMount)
		}
	}

	return merged
}

func MergeContainerPort(src, dst []corev1.ContainerPort) []corev1.ContainerPort {
	if len(src) == 0 {
		return dst
	}
	if len(dst) == 0 {
		return src
	}

	srcMap := map[string]int{}
	for i, port := range src {
		if port.Name != "" {
			srcMap[port.Name] = i
		}
	}

	merged := []corev1.ContainerPort{}
	merged = append(merged, src...)
	for _, port := range dst {
		if port.Name != "" {
			if idx, ok := srcMap[port.Name]; ok {
				merged[idx] = port
				continue
			}
		}

		merged = append(merged, port)
	}

	return merged
}

// Merge dst env into src
func MergeEnvVar(src, dst []corev1.EnvVar) []corev1.EnvVar {
	if len(src) == 0 {
		return dst
	}
	if len(dst) == 0 {
		return src
	}

	srcMap := map[string]int{}
	for i, envVar := range src {
		srcMap[envVar.Name] = i
	}

	merged := []corev1.EnvVar{}
	merged = append(merged, src...)
	for _, envVar := range dst {
		if idx, ok := srcMap[envVar.Name]; ok {
			merged[idx] = envVar
		} else {
			merged = append(merged, envVar)
		}
	}

	return merged
}

func GetContainerIndex(containers []corev1.Container, name string) int {
	for i, c := range containers {
		if c.Name == name {
			return i
		}
	}
	return -1
}

func GetVolumeIndex(volumes []corev1.Volume, name string) int {
	for i, v := range volumes {
		if v.Name == name {
			return i
		}
	}
	return -1
}

func GetVolumeMountIndex(volumeMounts []corev1.VolumeMount, mountPath string) int {
	for i, vm := range volumeMounts {
		if vm.MountPath == mountPath {
			return i
		}
	}
	return -1
}

func GetVersionFromImage(image string) string {
	if named, err := dockerref.ParseNormalizedNamed(image); err == nil {
		if tag, isTagged := named.(dockerref.Tagged); isTagged {
			return tag.Tag()
		}
		if digestset, isDigested := named.(dockerref.Digested); isDigested {
			return digestset.Digest().Encoded()
		}
	}

	return image
}

func NewAppLabels(instance, component string) map[string]string {
	return map[string]string{
		AppLabelInstance:  instance,
		AppLabelComponent: component,
		AppLabelName:      "milvus",
	}
}

// MergeLabels merges all labels together and returns a new label.
func MergeLabels(allLabels ...map[string]string) map[string]string {
	lb := make(map[string]string)

	for _, label := range allLabels {
		for k, v := range label {
			lb[k] = v
		}
	}

	return lb
}

// IsEqual check two object is equal.
func IsEqual(obj1, obj2 interface{}) bool {
	return equality.Semantic.DeepEqual(obj1, obj2)
}

func DeploymentReady(deployment appsv1.Deployment) bool {
	ready := true
	errored := false
	inProgress := false
	//ignoredConditions := []string{}

	for _, cond := range deployment.Status.Conditions {
		switch string(cond.Type) {
		case string(appsv1.DeploymentProgressing):
			if cond.Status == corev1.ConditionTrue &&
				cond.Reason != "" &&
				cond.Reason == "NewReplicaSetAvailable" {
				continue
			}
			inProgress = inProgress || cond.Status != corev1.ConditionFalse

		case kstatus.ConditionInProgress.String():
			inProgress = inProgress || cond.Status != corev1.ConditionFalse

		case kstatus.ConditionFailed.String(), string(appsv1.DeploymentReplicaFailure):
			errored = errored || cond.Status == corev1.ConditionTrue

		case "Ready", string(appsv1.DeploymentAvailable):
			ready = ready && cond.Status == corev1.ConditionTrue

			/* default:
			ignoredConditions = append(ignoredConditions, string(cond.Type)) */
		}
	}

	/* if len(ignoredConditions) > 0 {
		logger.Get(ctx).V(1).
			Info("unexpected conditions", "conditions", ignoredConditions)
	} */

	return ready && !inProgress && !errored
}

// PodRunningAndReady returns whether a pod is running and each container has
// passed it's ready state.
func PodRunningAndReady(pod corev1.Pod) (bool, error) {
	switch pod.Status.Phase {
	case corev1.PodFailed, corev1.PodSucceeded:
		return false, fmt.Errorf("pod completed")
	case corev1.PodRunning:
		for _, cond := range pod.Status.Conditions {
			if cond.Type != corev1.PodReady {
				continue
			}
			return cond.Status == corev1.ConditionTrue, nil
		}
		return false, fmt.Errorf("pod ready condition not found")
	}
	return false, nil
}

func GetConditionStatus(b bool) corev1.ConditionStatus {
	if b {
		return corev1.ConditionTrue
	}
	return corev1.ConditionFalse
}

func IsDependencyReady(status v1alpha1.MilvusClusterStatus) bool {
	ready := 0
	for _, c := range status.Conditions {
		if c.Type == v1alpha1.EtcdReady ||
			c.Type == v1alpha1.PulsarReady ||
			c.Type == v1alpha1.StorageReady {
			if c.Status == corev1.ConditionTrue {
				ready++
			}
		}
	}

	return ready == 3
}

func UpdateCondition(status *v1alpha1.MilvusClusterStatus, c v1alpha1.MilvusClusterCondition) {
	for i := range status.Conditions {
		cp := &status.Conditions[i]
		if cp.Type == c.Type {
			if cp.Status != c.Status ||
				cp.Reason != c.Reason ||
				cp.Message != c.Message {
				// Override
				cp.Status = c.Status
				cp.Message = c.Message
				cp.Reason = c.Reason
				// Update timestamp
				now := metav1.Now()
				cp.LastTransitionTime = &now
			}
			return
		}
	}

	// Append if not existing yet
	now := metav1.Now()
	c.LastTransitionTime = &now
	status.Conditions = append(status.Conditions, c)
}

func RenderTemplate(templateText string, data interface{}) ([]byte, error) {
	t, err := template.New("template").
		Funcs(sprig.TxtFuncMap()).Parse(templateText)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	err = t.Execute(&buf, data)
	if err != nil {
		return nil, err
	}

	objJSON, err := yaml.YAMLToJSON(buf.Bytes())
	if err != nil {
		return nil, err
	}

	return objJSON, nil
}

func NamespacedName(namespace, name string) types.NamespacedName {
	return types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
}

func GetMinioSecure(conf map[string]interface{}) bool {
	fields := []string{"minio", "useSSL"}
	usessl, exist := util.GetBoolValue(conf, fields...)
	if exist {
		return usessl
	}

	return false
}
