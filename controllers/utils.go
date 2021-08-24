package controllers

import (
	_ "crypto/sha256"
	"fmt"
	"strings"

	dockerref "github.com/docker/distribution/reference"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
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

func NewAppLabels(instance string, component MilvusComponent) map[string]string {
	return map[string]string{
		AppLabelInstance:  instance,
		AppLabelComponent: component.String(),
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
