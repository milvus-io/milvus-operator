package controllers

import (
	"context"
	"fmt"

	"github.com/milvus-io/milvus-operator/apis/milvus.io/v1beta1"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate mockgen -package=controllers -source=component_condition.go -destination=component_condition_mock.go ComponentConditionGetter
type ComponentConditionGetter interface {
	GetMilvusInstanceCondition(ctx context.Context, cli client.Client, mc v1beta1.Milvus) (v1beta1.MilvusCondition, error)
}

type ComponentConditionGetterImpl struct{}

func (c ComponentConditionGetterImpl) GetMilvusInstanceCondition(ctx context.Context, cli client.Client, mc v1beta1.Milvus) (v1beta1.MilvusCondition, error) {
	if mc.Spec.IsStopping() {
		reason := v1beta1.ReasonMilvusStopping
		msg := MessageMilvusStopped
		stopped, err := CheckMilvusStopped(ctx, cli, mc)
		if err != nil {
			return v1beta1.MilvusCondition{}, err
		}
		if stopped {
			reason = v1beta1.ReasonMilvusStopped
			msg = MessageMilvusStopping
		}

		return v1beta1.MilvusCondition{
			Type:    v1beta1.MilvusReady,
			Status:  corev1.ConditionFalse,
			Reason:  reason,
			Message: msg,
		}, nil
	}

	if !IsDependencyReady(mc.Status.Conditions) {
		notReadyConditions := GetNotReadyDependencyConditions(mc.Status.Conditions)
		reason := v1beta1.ReasonDependencyNotReady
		var msg string
		for depType, notReadyCondition := range notReadyConditions {
			if notReadyCondition != nil {
				msg += fmt.Sprintf("dep[%s]: %s;", depType, notReadyCondition.Message)
			} else {
				msg = "condition not probed yet"
			}
		}
		ctrl.LoggerFrom(ctx).Info("milvus dependency unhealty", "reason", reason, "msg", msg)
	}

	deployList := &appsv1.DeploymentList{}
	opts := &client.ListOptions{
		Namespace: mc.Namespace,
	}
	opts.LabelSelector = labels.SelectorFromSet(map[string]string{
		AppLabelInstance: mc.GetName(),
		AppLabelName:     "milvus",
	})
	if err := cli.List(ctx, deployList, opts); err != nil {
		return v1beta1.MilvusCondition{}, err
	}

	allComponents := GetComponentsBySpec(mc.Spec)
	var notReadyComponents []string
	var errDetail *ComponentErrorDetail
	var err error
	componentDeploy := makeComponentDeploymentMap(mc, deployList.Items)
	hasReadyReplica := false
	for _, component := range allComponents {
		deployment := componentDeploy[component.Name]
		if deployment != nil && DeploymentReady(deployment.Status) {
			if deployment.Status.ReadyReplicas > 0 {
				hasReadyReplica = true
			}
			continue
		}
		notReadyComponents = append(notReadyComponents, component.Name)
		if errDetail == nil {
			errDetail, err = getComponentErrorDetail(ctx, cli, component.Name, deployment)
			if err != nil {
				return v1beta1.MilvusCondition{}, errors.Wrap(err, "failed to get component err detail")
			}
		}
	}

	cond := v1beta1.MilvusCondition{
		Type: v1beta1.MilvusReady,
	}

	if len(notReadyComponents) == 0 {
		if !hasReadyReplica {
			return v1beta1.MilvusCondition{}, nil
		}
		cond.Status = corev1.ConditionTrue
		cond.Reason = v1beta1.ReasonMilvusHealthy
		cond.Message = MessageMilvusHealthy
	} else {
		cond.Status = corev1.ConditionFalse
		cond.Reason = v1beta1.ReasonMilvusComponentNotHealthy
		cond.Message = fmt.Sprintf("%s not ready, detail: %s", notReadyComponents, errDetail)
		ctrl.LoggerFrom(ctx).Info("milvus unhealty", "reason", cond.Reason, "msg", cond.Message)
	}

	return cond, nil
}

var getComponentErrorDetail = func(ctx context.Context, cli client.Client, component string, deploy *appsv1.Deployment) (*ComponentErrorDetail, error) {
	ret := &ComponentErrorDetail{ComponentName: component}
	if deploy == nil {
		return ret, nil
	}
	if deploy.Status.ObservedGeneration < deploy.Generation {
		ret.NotObserved = true
		return ret, nil
	}
	var err error
	ret.Deployment, err = GetDeploymentFalseCondition(*deploy)
	if err != nil {
		return ret, err
	}

	pods := &corev1.PodList{}
	opts := &client.ListOptions{
		Namespace: deploy.Namespace,
	}
	opts.LabelSelector = labels.SelectorFromSet(deploy.Spec.Selector.MatchLabels)
	if err := cli.List(ctx, pods, opts); err != nil {
		return nil, errors.Wrap(err, "list pods")
	}
	if len(pods.Items) == 0 {
		return ret, nil
	}
	for _, pod := range pods.Items {
		if !PodReady(pod) {
			podCondition, err := GetPodFalseCondition(pod)
			if err != nil {
				return nil, err
			}
			ret.PodName = pod.Name
			ret.Pod = podCondition
			ret.Container = getFirstNotReadyContainerStatus(pod.Status.ContainerStatuses)
			return ret, nil
		}
	}
	return ret, nil
}

func GetComponentConditionGetter() ComponentConditionGetter {
	return singletonComponentConditionGetter
}

var singletonComponentConditionGetter ComponentConditionGetter = ComponentConditionGetterImpl{}

func CheckMilvusStopped(ctx context.Context, cli client.Client, mc v1beta1.Milvus) (bool, error) {
	podList := &corev1.PodList{}
	opts := &client.ListOptions{
		Namespace: mc.Namespace,
	}
	opts.LabelSelector = labels.SelectorFromSet(map[string]string{
		AppLabelInstance: mc.GetName(),
		AppLabelName:     "milvus",
	})
	if err := cli.List(ctx, podList, opts); err != nil {
		return false, err
	}
	if len(podList.Items) > 0 {
		return false, nil
	}
	return true, nil
}
