package controllers

import (
	"context"
	"fmt"

	"github.com/milvus-io/milvus-operator/api/v1alpha1"
	"github.com/milvus-io/milvus-operator/pkg/milvus"

	corev1 "k8s.io/api/core/v1"
)

//
func (r *MilvusClusterReconciler) makeConfigMap(ctx context.Context, cluster *v1alpha1.MilvusCluster) (corev1.ConfigMap, error) {
	spec := cluster.Spec

	configuration := milvus.MilvusConfig{
		Etcd: milvus.MilvusConfigEtcd{
			Endpoints: ,
		},
	}

	cluster.Spec.

	return corev1.ConfigMap{}, nil
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
