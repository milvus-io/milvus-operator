package controllers

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/milvus-io/milvus-operator/apis/milvus.io/v1alpha1"
)

const (
	MessageEtcdReady       = "Etcd endpoints is healthy"
	MessageEtcdNotReady    = "All etcd endpoints are unhealthy"
	MessageStorageReady    = "Storage endpoints is healthy"
	MessageStorageNotReady = "All Storage endpoints are unhealthy"
	MessagePulsarReady     = "Pulsar is ready"
	MessagePulsarNotReady  = "Pulsar is not ready"
	MessageSecretNotExist  = "Secret not exist"
	MessageKeyNotExist     = "accesskey or secretkey not exist in secret"
	MessageDecodeErr       = "accesskey or secretkey decode error"
	MessageMilvusHealthy   = "All Milvus components are healthy"
)

var (
	S3ReadyCondition = v1alpha1.MilvusCondition{
		Type:   v1alpha1.StorageReady,
		Status: GetConditionStatus(true),
		Reason: v1alpha1.ReasonS3Ready,
	}
)

type EtcdEndPointHealth struct {
	Ep     string `json:"endpoint"`
	Health bool   `json:"health"`
	Error  string `json:"error,omitempty"`
}

type condResult struct {
	cond v1alpha1.MilvusCondition
	err  error
}

type MilvusClusterStatusSyncer struct {
	ctx context.Context
	client.Client
	logger logr.Logger

	sync.Once
}

func NewMilvusClusterStatusSyncer(ctx context.Context, client client.Client, logger logr.Logger) *MilvusClusterStatusSyncer {
	return &MilvusClusterStatusSyncer{
		ctx:    ctx,
		Client: client,
		logger: logger,
	}
}

func (r *MilvusClusterStatusSyncer) RunIfNot() {
	r.Once.Do(func() {
		go LoopWithInterval(r.ctx, r.syncUnhealthy, 30*time.Second, r.logger)
		go LoopWithInterval(r.ctx, r.syncHealthy, 1*time.Minute, r.logger)
	})
}

func (r *MilvusClusterStatusSyncer) syncUnhealthy() error {
	milvusClusterList := &v1alpha1.MilvusClusterList{}
	err := r.List(r.ctx, milvusClusterList)
	if err != nil {
		return errors.Wrap(err, "list milvuscluster failed")
	}

	argsArray := []Args{}
	for i := range milvusClusterList.Items {
		mc := &milvusClusterList.Items[i]
		r.logger.Info("syncUnhealthy mc status", "status", mc.Status.Status)
		if mc.Status.Status == "" ||
			mc.Status.Status == v1alpha1.StatusHealthy {
			continue
		}
		argsArray = append(argsArray, Args{mc})
	}
	err = defaultGroupRunner.RunDiffArgs(r.UpdateStatus, r.ctx, argsArray)
	return errors.Wrap(err, "UpdateStatus failed")
}

func (r *MilvusClusterStatusSyncer) syncHealthy() error {
	milvusClusterList := &v1alpha1.MilvusClusterList{}
	err := r.List(r.ctx, milvusClusterList)
	if err != nil {
		return errors.Wrap(err, "list milvuscluster failed")
	}
	argsArray := []Args{}
	for i := range milvusClusterList.Items {
		mc := &milvusClusterList.Items[i]
		if mc.Status.Status == v1alpha1.StatusHealthy {
			argsArray = append(argsArray, Args{mc})
		}
	}
	err = defaultGroupRunner.RunDiffArgs(r.UpdateStatus, r.ctx, argsArray)
	return errors.Wrap(err, "UpdateStatus failed")
}

func (r *MilvusClusterStatusSyncer) UpdateStatus(ctx context.Context, mc *v1alpha1.MilvusCluster) error {
	// ignore if default status not set
	if !IsClusterSetDefaultDone(mc) {
		return nil
	}

	funcs := []Func{
		r.GetEtcdCondition,
		r.GetMinioCondition,
		r.GetPulsarCondition,
	}
	ress := defaultGroupRunner.RunWithResult(funcs, ctx, *mc)

	errTexts := []string{}
	for _, res := range ress {
		if res.Err == nil {
			UpdateClusterCondition(&mc.Status, res.Data.(v1alpha1.MilvusCondition))
		} else {
			errTexts = append(errTexts, res.Err.Error())
		}
	}

	if len(errTexts) > 0 {
		return fmt.Errorf("update status error: %s", strings.Join(errTexts, ":"))
	}

	milvusCond, err := r.GetMilvusClusterCondition(ctx, *mc)
	if err != nil {
		return err
	}
	UpdateClusterCondition(&mc.Status, milvusCond)

	if milvusCond.Status != corev1.ConditionTrue {
		mc.Status.Status = v1alpha1.StatusUnHealthy
	} else {
		mc.Status.Status = v1alpha1.StatusHealthy
	}

	mc.Status.Endpoint = r.GetMilvusEndpoint(ctx, *mc)
	return r.Status().Update(ctx, mc)
}

func (r *MilvusClusterStatusSyncer) GetMilvusEndpoint(ctx context.Context, mc v1alpha1.MilvusCluster) string {
	info := MilvusEndpointInfo{
		Namespace:   mc.Namespace,
		Name:        mc.Name,
		ServiceType: mc.Spec.Com.Proxy.ServiceType,
		Port:        Proxy.GetComponentPort(mc.Spec),
	}
	return GetMilvusEndpoint(ctx, r.logger, r.Client, info)
}

func (r *MilvusClusterStatusSyncer) GetMilvusClusterCondition(ctx context.Context, mc v1alpha1.MilvusCluster) (v1alpha1.MilvusCondition, error) {
	info := MilvusConditionInfo{
		Object:     mc.GetObjectMeta(),
		Conditions: mc.Status.Conditions,
		IsCluster:  true,
	}
	return GetMilvusInstanceCondition(ctx, r.Client, info)
}

func (r *MilvusClusterStatusSyncer) GetPulsarCondition(
	ctx context.Context, mc v1alpha1.MilvusCluster) (v1alpha1.MilvusCondition, error) {
	return GetPulsarCondition(ctx, r.logger, mc.Spec.Dep.Pulsar)
}

// TODO: rename as GetStorageCondition
func (r *MilvusClusterStatusSyncer) GetMinioCondition(
	ctx context.Context, mc v1alpha1.MilvusCluster) (v1alpha1.MilvusCondition, error) {
	if mc.Spec.Dep.Storage.Type == v1alpha1.StorageTypeS3 {
		return S3ReadyCondition, nil
	}
	info := StorageConditionInfo{
		Namespace: mc.Namespace,
		Storage:   mc.Spec.Dep.Storage,
		EndPoint:  mc.Spec.Dep.Storage.Endpoint,
		UseSSL:    GetMinioSecure(mc.Spec.Conf.Data),
	}
	return GetMinioCondition(ctx, r.logger, r.Client, info)
}

func (r *MilvusClusterStatusSyncer) GetEtcdCondition(ctx context.Context, mc v1alpha1.MilvusCluster) (v1alpha1.MilvusCondition, error) {
	return GetEtcdCondition(ctx, mc.Spec.Dep.Etcd.Endpoints)
}
