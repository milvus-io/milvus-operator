package controllers

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestComponentErrorDetail_String(t *testing.T) {
	t.Run("deployment new generation not observed", func(t *testing.T) {
		detail := ComponentErrorDetail{
			ComponentName: "proxy",
			NotObserved:   true,
		}
		assert.Equal(t, "component[proxy]: updating deployment", detail.String())
	})

	t.Run("deployment not created", func(t *testing.T) {
		detail := ComponentErrorDetail{
			ComponentName: "proxy",
			Deployment:    nil,
		}
		assert.Equal(t, "component[proxy]: deployment not created", detail.String())
	})

	t.Run("no pod", func(t *testing.T) {
		detail := ComponentErrorDetail{
			ComponentName: "proxy",
			Deployment: &appsv1.DeploymentCondition{
				Type:    appsv1.DeploymentAvailable,
				Status:  corev1.ConditionFalse,
				Reason:  "...",
				Message: "...",
			},
		}

		assert.Equal(t, "component[proxy]: deployment status[Available:False]: reason[...]: ...", detail.String())
	})

	t.Run("pod schedule failed", func(t *testing.T) {
		detail := ComponentErrorDetail{
			ComponentName: "proxy",
			PodName:       "myrelease-proxy-xxxx-xxxx",
			Pod: &corev1.PodCondition{
				Type:    corev1.PodScheduled,
				Status:  corev1.ConditionFalse,
				Reason:  "Unschedulable",
				Message: "...",
			}}
		assert.Equal(t, "component[proxy]: pod[myrelease-proxy-xxxx-xxxx]: status[PodScheduled:False]: reason[Unschedulable]: ...", detail.String())
	})

	t.Run("pod pull image failed", func(t *testing.T) {
		detail := ComponentErrorDetail{
			ComponentName: "proxy",
			PodName:       "myrelease-proxy-xxxx-xxxx",
			Pod: &corev1.PodCondition{
				Type:   corev1.PodReady,
				Status: corev1.ConditionFalse,
				Reason: "...",
			},
			Container: &corev1.ContainerStatus{
				Name: "main",
				State: corev1.ContainerState{
					Waiting: &corev1.ContainerStateWaiting{
						Reason:  "ErrImagePull",
						Message: "...",
					},
				},
			},
		}
		assert.Equal(t,
			"component[proxy]: pod[myrelease-proxy-xxxx-xxxx]: container[main]: currentState[waiting] reason[ErrImagePull]: ...",
			detail.String())
	})
}

func TestGetComponentErrorDetail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx := context.Background()
	cli := NewMockK8sClient(ctrl)
	component := "proxy"

	t.Run("deployment nil", func(t *testing.T) {
		ret, err := GetComponentErrorDetail(ctx, cli, component, nil)
		assert.NoError(t, err)
		assert.Equal(t, component, ret.ComponentName)
	})

	deploy := &appsv1.Deployment{}
	deploy.Namespace = "ns"
	deploy.Name = "test"
	deploy.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"app": "test",
		},
	}
	deploy.Generation = 1

	t.Run("new generation not observed", func(t *testing.T) {
		deploy.Status.ObservedGeneration = 0
		detail, err := GetComponentErrorDetail(ctx, cli, component, deploy)
		assert.NoError(t, err)
		assert.True(t, detail.NotObserved)
	})

	deploy.Status.ObservedGeneration = 1
	t.Run("creating, no pod", func(t *testing.T) {
		cli.EXPECT().List(ctx, gomock.Any(), gomock.Any()).Return(nil)
		ret, err := GetComponentErrorDetail(ctx, cli, component, deploy)
		assert.NoError(t, err)
		assert.Equal(t, component, ret.ComponentName)
		assert.Equal(t, "creating", ret.Deployment.Message)
	})

	t.Run("creating, pod scheduling", func(t *testing.T) {
		pod := &corev1.Pod{}
		pod.Name = "test"
		pod.Namespace = "ns"
		cli.EXPECT().List(ctx, gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, list interface{}, opts ...interface{}) error {
				podList := list.(*corev1.PodList)
				podList.Items = append(podList.Items, *pod)
				return nil
			})
		ret, err := GetComponentErrorDetail(ctx, cli, component, deploy)
		assert.NoError(t, err)
		assert.Equal(t, "scheduling", ret.Pod.Message)
	})

	t.Run("creating, all pods ready error", func(t *testing.T) {
		pod := &corev1.Pod{}
		pod.Name = "test"
		pod.Namespace = "ns"
		pod.Status.Conditions = []corev1.PodCondition{
			{
				Type:   corev1.PodScheduled,
				Status: corev1.ConditionTrue,
			},
			{
				Type:   corev1.PodInitialized,
				Status: corev1.ConditionTrue,
			},
			{
				Type:   corev1.ContainersReady,
				Status: corev1.ConditionTrue,
			},
			{
				Type:   corev1.PodReady,
				Status: corev1.ConditionTrue,
			},
		}
		cli.EXPECT().List(ctx, gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, list interface{}, opts ...interface{}) error {
				podList := list.(*corev1.PodList)
				podList.Items = append(podList.Items, *pod)
				return nil
			})
		_, err := GetComponentErrorDetail(ctx, cli, component, deploy)
		assert.Error(t, err)
	})
}

func TestGetDeploymentFalseCondition(t *testing.T) {
	t.Run("creating", func(t *testing.T) {
		deployment := appsv1.Deployment{}
		condition, err := GetDeploymentFalseCondition(deployment)
		assert.NoError(t, err)
		assert.Equal(t, appsv1.DeploymentProgressing, condition.Type)
		assert.Equal(t, corev1.ConditionFalse, condition.Status)
		assert.Equal(t, "creating", condition.Message)
	})
	t.Run("DeploymentReplicaFailure", func(t *testing.T) {
		deployment := appsv1.Deployment{}
		condition := appsv1.DeploymentCondition{
			Type:   appsv1.DeploymentReplicaFailure,
			Status: corev1.ConditionTrue,
		}
		deployment.Status.Conditions = append(deployment.Status.Conditions, condition)
		ret, err := GetDeploymentFalseCondition(deployment)
		assert.NoError(t, err)
		assert.Equal(t, condition, *ret)
	})
	t.Run("all condition ok error", func(t *testing.T) {
		deployment := appsv1.Deployment{}
		deployment.Status.Conditions = append(deployment.Status.Conditions, appsv1.DeploymentCondition{
			Type:   appsv1.DeploymentAvailable,
			Status: corev1.ConditionTrue,
		})
		deployment.Status.Conditions = append(deployment.Status.Conditions, appsv1.DeploymentCondition{
			Type:   appsv1.DeploymentProgressing,
			Status: corev1.ConditionTrue,
		})
		_, err := GetDeploymentFalseCondition(deployment)
		assert.Error(t, err)
	})
}

func TestGetPodFalseCondition(t *testing.T) {
	t.Run("scheduling", func(t *testing.T) {
		pod := corev1.Pod{}
		condition, err := GetPodFalseCondition(pod)
		assert.NoError(t, err)
		assert.Equal(t, corev1.PodScheduled, condition.Type)
		assert.Equal(t, corev1.ConditionFalse, condition.Status)
		assert.Equal(t, "scheduling", condition.Message)

	})

	t.Run("schedule failed", func(t *testing.T) {
		pod := corev1.Pod{}
		scheduleCondition := corev1.PodCondition{
			Type:    corev1.PodScheduled,
			Status:  corev1.ConditionFalse,
			Reason:  "Unschedulable",
			Message: "...",
		}
		pod.Status.Conditions = append(pod.Status.Conditions, scheduleCondition)
		condition, err := GetPodFalseCondition(pod)
		assert.NoError(t, err)
		assert.Equal(t, scheduleCondition, *condition)
	})

	t.Run("initializing", func(t *testing.T) {
		pod := corev1.Pod{}
		pod.Status.Conditions = append(pod.Status.Conditions, corev1.PodCondition{
			Type:   corev1.PodScheduled,
			Status: corev1.ConditionTrue,
		})
		condition, err := GetPodFalseCondition(pod)
		assert.NoError(t, err)
		assert.Equal(t, corev1.PodInitialized, condition.Type)
		assert.Equal(t, corev1.ConditionFalse, condition.Status)
		assert.Equal(t, "initializing", condition.Message)

	})

	t.Run("pod ready error", func(t *testing.T) {
		pod := corev1.Pod{}
		pod.Status.Conditions = append(pod.Status.Conditions, corev1.PodCondition{
			Type:   corev1.PodScheduled,
			Status: corev1.ConditionTrue,
		})
		pod.Status.Conditions = append(pod.Status.Conditions, corev1.PodCondition{
			Type:   corev1.PodInitialized,
			Status: corev1.ConditionTrue,
		})
		pod.Status.Conditions = append(pod.Status.Conditions, corev1.PodCondition{
			Type:   corev1.ContainersReady,
			Status: corev1.ConditionTrue,
		})
		pod.Status.Conditions = append(pod.Status.Conditions, corev1.PodCondition{
			Type:   corev1.PodReady,
			Status: corev1.ConditionTrue,
		})
		_, err := GetPodFalseCondition(pod)
		assert.Error(t, err)
	})
}

func TestGetContainerMessage(t *testing.T) {
	t.Run("container running", func(t *testing.T) {
		containerStatus := corev1.ContainerStatus{
			Name: "test",
			State: corev1.ContainerState{
				Running: &corev1.ContainerStateRunning{},
			},
		}
		message := GetContainerMessage(containerStatus)
		assert.Equal(t, "container[test]: currentState[running]", message)
	})

	t.Run("container waiting", func(t *testing.T) {
		containerStatus := corev1.ContainerStatus{
			Name: "test",
			State: corev1.ContainerState{
				Waiting: &corev1.ContainerStateWaiting{
					Reason:  "Reason",
					Message: "message",
				},
			},
		}
		message := GetContainerMessage(containerStatus)
		assert.Equal(t, "container[test]: currentState[waiting] reason[Reason]: message", message)
	})

	t.Run("container restarted", func(t *testing.T) {
		containerStatus := corev1.ContainerStatus{
			Name: "test",
			LastTerminationState: corev1.ContainerState{
				Terminated: &corev1.ContainerStateTerminated{
					Reason:  "Reason",
					Message: "message",
				},
			},
			RestartCount: 1,
		}
		message := GetContainerMessage(containerStatus)
		assert.Equal(t, "container[test]: restartCount[1] lastState[terminated] reason[Reason]: message", message)
	})

	t.Run("container unknown", func(t *testing.T) {
		containerStatus := corev1.ContainerStatus{
			Name: "test",
		}
		message := GetContainerMessage(containerStatus)
		assert.Equal(t, "container[test]: currentState[unknown]", message)
	})
}

func TestGetFirstNotReadyContainerStatus(t *testing.T) {
	ret := getFirstNotReadyContainerStatus(nil)
	assert.Nil(t, ret)

	containerStatuses := []corev1.ContainerStatus{
		{
			Name: "test",
		},
	}
	ret = getFirstNotReadyContainerStatus(containerStatuses)
	assert.Equal(t, &containerStatuses[0], ret)
}
