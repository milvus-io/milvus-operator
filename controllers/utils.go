package controllers

import (
	_ "crypto/sha256"
	"fmt"

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

func MergeVolumeMount(src, dst []corev1.VolumeMount) []corev1.VolumeMount {
	if len(src) == 0 {
		return dst
	}
	if len(dst) == 0 {
		return src
	}

	m := map[string]corev1.VolumeMount{}
	for _, volumeMount := range src {
		m[volumeMount.MountPath] = volumeMount
	}
	for _, volumeMount := range dst {
		m[volumeMount.MountPath] = volumeMount
	}

	merged := []corev1.VolumeMount{}
	for _, volumeMount := range m {
		merged = append(merged, volumeMount)
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

	m := map[string]corev1.ContainerPort{}
	for _, port := range src {
		m[port.Name] = port
	}
	for _, port := range dst {
		m[port.Name] = port
	}

	merged := []corev1.ContainerPort{}
	for _, port := range m {
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

	m := map[string]corev1.EnvVar{}
	for _, envVar := range src {
		m[envVar.Name] = envVar
	}

	for _, envVar := range dst {
		m[envVar.Name] = envVar
	}

	merged := []corev1.EnvVar{}
	for _, envVar := range m {
		merged = append(merged, envVar)
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
		AppLabelName:      "MilvusCluster",
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
