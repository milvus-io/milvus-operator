package controllers

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/milvus-io/milvus-operator/api/v1alpha1"
	milvusv1alpha1 "github.com/milvus-io/milvus-operator/api/v1alpha1"
	"github.com/minio/madmin-go"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MilvusStatusSyncer struct {
	ctx context.Context
	client.Client
	logger logr.Logger

	sync.Once
}

func NewMilvusStatusSyncer(ctx context.Context, client client.Client, logger logr.Logger) *MilvusStatusSyncer {
	return &MilvusStatusSyncer{
		ctx:    ctx,
		Client: client,
		logger: logger,
	}
}

func (r *MilvusStatusSyncer) RunIfNot() {
	r.Once.Do(func() {
		go r.loopWithInterval(r.syncUnhealthy, 30*time.Second)
		go r.loopWithInterval(r.syncHealthy, 1*time.Minute)
	})
}

func (r *MilvusStatusSyncer) loopWithInterval(loopFunc func() error, interval time.Duration) {
	funcName := getFuncName(loopFunc)
	r.logger.Info("setup loop", "func", funcName, "interval", interval.String())
	ticker := time.NewTicker(interval)
	for {
		r.logger.Info("loopfunc run", "func", funcName)
		err := loopFunc()
		if err != nil {
			r.logger.Error(err, "loopFunc err", "func", funcName)
		}
		r.logger.Info("loopfunc end", "func", funcName)
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func WrappedUpdateStatus(
	f func(ctx context.Context, mil *v1alpha1.Milvus) error,
	ctx context.Context, mil *v1alpha1.Milvus) func() error {
	return func() error {
		return f(ctx, mil)
	}
}

func (r *MilvusStatusSyncer) syncUnhealthy() error {
	milvusList := &v1alpha1.MilvusList{}
	err := r.List(r.ctx, milvusList)
	if err != nil {
		return errors.Wrap(err, "list milvuscluster failed")
	}
	g, gtx := NewGroup(r.ctx)
	for i := range milvusList.Items {
		mil := &milvusList.Items[i]
		r.logger.Info("syncUnhealthy mil status", "status", mil.Status.Status)
		if mil.Status.Status == "" ||
			mil.Status.Status == milvusv1alpha1.StatusHealthy {
			continue
		}
		g.Go(WrappedUpdateStatus(r.UpdateStatus, gtx, mil))

	}
	if err := g.Wait(); err != nil {
		return fmt.Errorf("UpdateStatus: %w", err)
	}
	return nil
}

func (r *MilvusStatusSyncer) syncHealthy() error {
	milvusList := &v1alpha1.MilvusList{}
	err := r.List(r.ctx, milvusList)
	if err != nil {
		return errors.Wrap(err, "list milvuscluster failed")
	}
	g, gtx := NewGroup(r.ctx)
	for i := range milvusList.Items {
		mil := &milvusList.Items[i]
		if mil.Status.Status == milvusv1alpha1.StatusHealthy {
			g.Go(WrappedUpdateStatus(r.UpdateStatus, gtx, mil))
		}
	}
	if err := g.Wait(); err != nil {
		return fmt.Errorf("UpdateStatus: %w", err)
	}
	return nil
}

func (r *MilvusStatusSyncer) UpdateStatus(ctx context.Context, mil *v1alpha1.Milvus) error {
	condChan := make(chan condResult, 3)
	var wait sync.WaitGroup

	// ignore if default status not set
	if !IsSetDefaultDone(mil) {
		return nil
	}

	wait.Add(1)
	go func(ch chan<- condResult, w *sync.WaitGroup) {
		defer w.Done()
		ch <- r.GetEtcdCondition(ctx, *mil)
	}(condChan, &wait)

	wait.Add(1)
	go func(ch chan<- condResult, w *sync.WaitGroup) {
		defer w.Done()
		ch <- r.GetMinioCondition(ctx, *mil)
	}(condChan, &wait)

	wait.Wait()
	close(condChan)
	errTexts := []string{}
	for cond := range condChan {
		if cond.err == nil {
			UpdateCondition(&mil.Status, cond.cond)
		} else {
			errTexts = append(errTexts, cond.err.Error())
		}
	}

	if len(errTexts) > 0 {
		return fmt.Errorf("update status error: %s", strings.Join(errTexts, ":"))
	}

	milvusCond := r.GetMilvusCondition(ctx, *mil)
	if milvusCond.err != nil {
		return milvusCond.err
	}
	UpdateCondition(&mil.Status, milvusCond.cond)

	if milvusCond.cond.Status != corev1.ConditionTrue {
		mil.Status.Status = v1alpha1.StatusUnHealthy
	} else {
		mil.Status.Status = v1alpha1.StatusHealthy
	}

	mil.Status.Endpoint = r.GetMilvusEndpoint(ctx, *mil)
	return r.Status().Update(ctx, mil)
}

func (r *MilvusStatusSyncer) GetMilvusEndpoint(ctx context.Context, mil v1alpha1.Milvus) string {
	if mil.Spec.ServiceType == corev1.ServiceTypeLoadBalancer {
		proxy := &corev1.Service{}
		key := NamespacedName(mil.Namespace, Proxy.GetServiceInstanceName(mil.Name))
		if err := r.Get(ctx, key, proxy); err != nil {
			r.logger.Error(err, "Get Milvus endpoint error")
			return ""
		}

		if len(proxy.Status.LoadBalancer.Ingress) > 0 {
			return fmt.Sprintf("%s:%d", proxy.Status.LoadBalancer.Ingress[0].IP, MilvusPort)
		}
	}

	if mil.Spec.ServiceType == corev1.ServiceTypeClusterIP {
		return fmt.Sprintf("%s.%s:%d", mil.Name, mil.Namespace, MilvusPort)
	}

	return ""
}

func (r *MilvusStatusSyncer) GetMilvusCondition(ctx context.Context, mil v1alpha1.Milvus) condResult {
	if !IsDependencyReady(mil.Status) {
		return condResult{
			cond: v1alpha1.MilvusCondition{
				Type:    v1alpha1.MilvusReady,
				Status:  corev1.ConditionFalse,
				Reason:  v1alpha1.ReasonDependencyNotReady,
				Message: "Milvus Dependencies is not ready",
			},
		}
	}

	deployments := &appsv1.DeploymentList{}
	opts := &client.ListOptions{}
	opts.LabelSelector = labels.SelectorFromSet(map[string]string{
		AppLabelInstance: mil.Name,
		AppLabelName:     "milvus",
	})
	if err := r.List(ctx, deployments, opts); err != nil {
		return condResult{err: err}
	}

	ready := 0
	notReadyComponents := []string{}
	for _, deployment := range deployments.Items {
		if metav1.IsControlledBy(&deployment, &mil) {
			if DeploymentReady(deployment) {
				ready++
			} else {
				notReadyComponents = append(notReadyComponents, deployment.Labels[AppLabelComponent])
			}
		}
	}

	cond := v1alpha1.MilvusCondition{
		Type: v1alpha1.MilvusReady,
	}
	if ready > 0 {
		cond.Status = corev1.ConditionTrue
		cond.Reason = v1alpha1.ReasonMilvusHealthy
		cond.Message = MessageMilvusHealthy
	} else {
		cond.Status = corev1.ConditionFalse
		cond.Reason = v1alpha1.ReasonMilvusComponentNotHealthy
		sort.Strings(notReadyComponents)
		cond.Message = fmt.Sprintf("%s not ready", notReadyComponents)
	}

	return condResult{cond: cond}
}

func (r *MilvusStatusSyncer) GetMinioCondition(
	ctx context.Context, mil v1alpha1.Milvus) condResult {
	secret := &corev1.Secret{}
	key := types.NamespacedName{Namespace: mil.Namespace, Name: mil.Spec.Dep.Storage.SecretRef}
	err := r.Get(ctx, key, secret)
	if err != nil && !k8sErrors.IsNotFound(err) {
		return condResult{err: err}
	}

	if k8sErrors.IsNotFound(err) {
		return newErrStorageCondResult(v1alpha1.ReasonSecretNotExist, MessageSecretNotExist)
	}

	accesskey, exist1 := secret.Data[AccessKey]
	secretkey, exist2 := secret.Data[SecretKey]
	if !exist1 || !exist2 {
		return newErrStorageCondResult(v1alpha1.ReasonSecretNotExist, MessageKeyNotExist)
	}

	mdmClnt, err := madmin.New(
		mil.Spec.Dep.Storage.Endpoint,
		string(accesskey), string(secretkey),
		GetMinioSecure(mil.Spec.Conf.Data),
	)

	if err != nil {
		return newErrStorageCondResult(v1alpha1.ReasonClientErr, err.Error())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	st, err := mdmClnt.ServerInfo(ctx)
	if err != nil {
		return newErrStorageCondResult(v1alpha1.ReasonClientErr, err.Error())
	}

	ready := false
	for _, server := range st.Servers {
		if server.State == "online" {
			ready = true
			break
		}
	}

	cond := v1alpha1.MilvusCondition{
		Type:   v1alpha1.StorageReady,
		Status: GetConditionStatus(ready),
		Reason: v1alpha1.ReasonStorageReady,
	}

	if !ready {
		cond.Reason = v1alpha1.ReasonStorageNotReady
		cond.Message = MessageStorageNotReady
	}

	return condResult{cond: cond}
}

func (r *MilvusStatusSyncer) GetEtcdCondition(ctx context.Context, mil v1alpha1.Milvus) condResult {
	endpoints := mil.Spec.Dep.Etcd.Endpoints
	health := GetEndpointsHealth(endpoints)
	etcdReady := false
	for _, ep := range endpoints {
		epHealth := health[ep]
		if epHealth.Health {
			etcdReady = true
		}
	}

	cond := v1alpha1.MilvusCondition{
		Type:    v1alpha1.EtcdReady,
		Status:  GetConditionStatus(etcdReady),
		Reason:  v1alpha1.ReasonEtcdReady,
		Message: MessageEtcdReady,
	}
	if !etcdReady {
		cond.Reason = v1alpha1.ReasonEtcdNotReady
		cond.Message = MessageEtcdNotReady
	}

	return condResult{cond: cond}
}
